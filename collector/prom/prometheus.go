package prom

import (
	"encoding/json"
	"fmt"
	"github.com/mashenjun/mole/utils"
	"github.com/pingcap/tiup/pkg/cliutil/progress"
	tiuputils "github.com/pingcap/tiup/pkg/utils"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	metricStep = 15 // use 15s step, also 15 seconds is the minimal step
)

// CollectStat is estimated size stats of data to be collected
type CollectStat struct {
	Target string
	Size   int64
}

type MetricsRecord struct {
	Record string `yaml:"record"`
	Expr   string `yaml:"expr"`
}

// MetricsCollect is the options collecting metrics
type MetricsCollect struct {
	timeSteps    []string
	rawMetrics   []string        // raw metric list
	cookedRecord []MetricsRecord // cooked metric list
	targetRecord []MetricsRecord // merge raw metrics and cooked metrics
	concurrency  int
	scrapeBegin  string    // time range to filter metrics.
	scrapeEnd    string    // time range to filter metrics.
	beginTime    time.Time // helper fields just to gen dir name
	endTime      time.Time // helper fields just to gen dir name
	cli          *http.Client
	outputDir    string // dir where the metrics data will be stored.
	merge        bool
	fileFlag     int // file flag used to open file
	continues    bool
	subDirEnable bool // if store the collect metrics in to sub dir, default true
}

type MetricsCollectOpt func(*MetricsCollect) error

func WithTimeRange(begin, end string) MetricsCollectOpt {
	return func(collect *MetricsCollect) error {
		steps, _, err := parseTimeRange(begin, end)
		if err != nil {
			return err
		}
		collect.timeSteps = steps
		collect.scrapeBegin = begin
		collect.scrapeEnd = end
		{
			ts, err := time.Parse(time.RFC3339, begin)
			if err != nil {
				return err
			}
			collect.beginTime = ts
		}
		{
			ts, err := time.Parse(time.RFC3339, end)
			if err != nil {
				return err
			}
			collect.endTime = ts
		}
		return nil
	}
}

func WithHttpCli(cli *http.Client) MetricsCollectOpt {
	return func(collect *MetricsCollect) error {
		collect.cli = cli
		return nil
	}
}

func WithConcurrency(c int) MetricsCollectOpt {
	return func(collect *MetricsCollect) error {
		cpuCnt := runtime.NumCPU()
		if cpuCnt<<1 < c || c == 0 {
			c = cpuCnt << 1
		}
		collect.concurrency = c
		return nil
	}
}

func WithMerge(merge bool) MetricsCollectOpt {
	return func(collect *MetricsCollect) error {
		collect.merge = merge
		if merge {
			collect.fileFlag = os.O_RDWR | os.O_CREATE | os.O_APPEND
		} else {
			collect.fileFlag = os.O_RDWR | os.O_CREATE | os.O_TRUNC
		}
		return nil
	}
}

func WithOutputDir(output string) MetricsCollectOpt {
	return func(collect *MetricsCollect) error {
		if err := utils.EnsureDir(output); err != nil {
			return err
		}
		collect.outputDir = output
		return nil
	}
}

func WithContinues(continues bool) MetricsCollectOpt {
	return func(collect *MetricsCollect) error {
		collect.continues = continues
		return nil
	}
}

func WithSubDirEnable(enable bool) MetricsCollectOpt {
	return func(collect *MetricsCollect) error {
		collect.subDirEnable = enable
		return nil
	}
}

func NewMetricsCollect(opts ...MetricsCollectOpt) (*MetricsCollect, error) {
	mc := &MetricsCollect{
		targetRecord: make([]MetricsRecord, 0),
		subDirEnable: true,
	}
	for _, opt := range opts {
		if err := opt(mc); err != nil {
			return nil, err
		}
	}
	return mc, nil
}

// Desc implements the Collector interface
func (c *MetricsCollect) Desc() string {
	return "metrics from Prometheus node"
}

func (c *MetricsCollect) SetRawMetrics(m []string) {
	c.rawMetrics = m
}

func (c *MetricsCollect) SetCookedRecord(cr []MetricsRecord) {
	c.cookedRecord = cr
}

type Endpoint struct {
	Schema string
	Host   string
	Port   string
}

// Prepare implements the Collector interface
func (c *MetricsCollect) Prepare(topo []Endpoint) (map[string][]CollectStat, error) {
	if len(topo) < 1 {
		fmt.Println("No Prometheus node found in topology, skip.")
		return nil, nil
	}
	var queryOK bool
	var queryErr error
	var promAddr string
	//var targets []*TargetMetrics
	for _, prom := range topo {
		promAddr = fmt.Sprintf("%s://%s:%s", prom.Schema, prom.Host, prom.Port)
		metrics, err := c.getMetricList(promAddr)
		if err == nil {
			queryOK = true
		}
		queryErr = err

		if queryOK {
			c.rawMetrics = metrics
			break
		}
	}

	// if query success for any one of prometheus, ignore errors for other instances
	if !queryOK {
		return nil, queryErr
	}
	// merge raw metrics and cooked metrics as target metrics
	for _, mtc := range c.rawMetrics {
		c.targetRecord = append(c.targetRecord, MetricsRecord{
			Record: mtc,
			Expr:   mtc,
		})
	}
	c.targetRecord = append(c.targetRecord, c.cookedRecord...)
	return nil, nil
}

// Collect implements the Collector interface
func (c *MetricsCollect) Collect(topo []Endpoint) error {
	if len(topo) < 1 {
		fmt.Println("No Prometheus node found in topology, skip.")
		return nil
	}
	// we may not need the multi bar
	mb := progress.NewMultiBar("+ Dumping metrics")
	bars := make(map[string]*progress.MultiBarItem)
	total := len(c.targetRecord)
	mu := sync.Mutex{}
	for _, prom := range topo {
		key := fmt.Sprintf("%s:%v", prom.Host, prom.Port)
		if _, ok := bars[key]; !ok {
			bars[key] = mb.AddBar(fmt.Sprintf("  - Querying server %s", key))
		}
	}
	mb.StartRenderLoop()
	defer mb.StopRenderLoop()

	tl := utils.NewTokenLimiter(uint(c.concurrency))
	for _, prom := range topo {
		key := fmt.Sprintf("%s:%v", prom.Host, prom.Port)
		done := 1
		// ensure the file path for output is ready
		if err := utils.EnsureDir(c.genDirPath(prom)); err != nil {
			bars[key].UpdateDisplay(&progress.DisplayProps{
				Prefix: fmt.Sprintf("  - Query server %s: %s", key, err),
				Mode:   progress.ModeError,
			})
			return err
		}
		// get existed metrics
		existed, err := c.listExistedMetrics(c.genDirPath(prom))
		if err != nil {
			return err
		}
		for _, r := range c.targetRecord {
			if _, ok := existed[r.Record]; ok {
				bars[key].UpdateDisplay(&progress.DisplayProps{
					Prefix: fmt.Sprintf("  - Querying server %s", key),
					Suffix: fmt.Sprintf("skip %s ...", r.Record),
				})
				done++
				if done >= total {
					bars[key].UpdateDisplay(&progress.DisplayProps{
						Prefix: fmt.Sprintf("  - Query server %s", key),
						Mode:   progress.ModeDone,
					})
				}
				continue
			}
			go func(tok *utils.Token, mtc string, expr string) {
				bars[key].UpdateDisplay(&progress.DisplayProps{
					Prefix: fmt.Sprintf("  - Querying server %s", key),
					Suffix: fmt.Sprintf("%d/%d querying %s ...", done, total, mtc),
				})
				if err := c.collectMetric(prom, c.timeSteps, mtc, expr); err != nil {
					fmt.Printf("collect metrics %+v error: %+v\n", mtc, err)
				}
				mu.Lock()
				done++
				if done >= total {
					bars[key].UpdateDisplay(&progress.DisplayProps{
						Prefix: fmt.Sprintf("  - Query server %s", key),
						Mode:   progress.ModeDone,
					})
				}
				mu.Unlock()
				tl.Put(tok)
			}(tl.Get(), r.Record, r.Expr)
		}
	}
	tl.Wait()
	return nil
}

func (c *MetricsCollect) getMetricList(prom string) ([]string, error) {
	if len(c.rawMetrics) > 0 {
		return c.rawMetrics, nil
	}
	resp, err := c.cli.Get(fmt.Sprintf("%s/api/v1/label/__name__/values", prom))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, err
	}

	r := struct {
		Metrics []string `json:"data"`
	}{}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, err
	}
	return r.Metrics, nil
}

func (c *MetricsCollect) collectMetric(prom Endpoint, ts []string, mtc string, expr string) error {
	promAddr := fmt.Sprintf("%s://%s:%s", prom.Schema, prom.Host, prom.Port)
	for i := 0; i < len(ts)-1; i++ {
		if err := tiuputils.Retry(
			func() error {
				start, end := ts[i], ts[i+1]
				if start != end {
					// offset end by 1 second
					et, err := time.Parse(time.RFC3339, end)
					if err != nil {
						return err
					}
					end = et.Add(-1 * time.Second).Format(time.RFC3339)
				}
				resp, err := c.cli.PostForm(
					fmt.Sprintf("%s/api/v1/query_range", promAddr),
					url.Values{
						"query": {expr},
						"start": {start},
						"end":   {end},
						"step":  {strconv.Itoa(metricStep)},
					},
				)
				if err != nil {
					fmt.Printf("failed query metric %s: %s, retry...\n", mtc, err)
					return err
				}
				defer resp.Body.Close()
				if resp.StatusCode != http.StatusOK {
					fmt.Printf("failed query metric %s: %s, retry...\n", mtc, resp.Status)
					return err
				}
				// implement 2
				// the following implementation writes response body to file directly
				filename := c.genFileName(mtc, i)
				topoDir := c.genDirPath(prom)
				dst, err := os.OpenFile(filepath.Join(
					topoDir, filename,
				), c.fileFlag, 0644)
				if err != nil {
					fmt.Printf("collect metric %s: %s, retry...\n", mtc, err)
				}
				defer dst.Close()

				cnt, err := io.Copy(dst, resp.Body)
				if err != nil {
					fmt.Printf("write metric %s to file error: %s, retry...\n", mtc, err)
					return err
				}
				if cnt == 0 {
					fmt.Println("warning, zero bytes in response body")
				}
				if c.merge {
					if _, err := dst.Write([]byte("\n")); err != nil {
						fmt.Printf("write file error: %+v\n", err)
						return err
					}
				}
				return nil
			},
			tiuputils.RetryOption{
				Attempts: 3,
				Delay:    time.Microsecond * 300,
				Timeout:  time.Second * 120,
			},
		); err != nil {
			fmt.Printf("fetch metrics %v from %v to %v error: %v", mtc, ts[i], ts[i+1], err)
		}
	}
	return nil
}

func (c *MetricsCollect) genFileName(mtc string, idx int) string {
	if c.merge {
		return fmt.Sprintf("%s.json", mtc)
	}
	// if not merge, the file name should include idx
	return fmt.Sprintf("%s-%v.json", mtc, idx)
}

// the dir name should also include the timestamp range.
func (c *MetricsCollect) genDirPath(ep Endpoint) string {
	if c.subDirEnable {
		return filepath.Join(c.outputDir, fmt.Sprintf("%s-%v-%s-%s", ep.Host, ep.Port,
			c.beginTime.Format("060102T150405Z0700"),
			c.endTime.Format("060102T150405Z0700")))
	}
	return c.outputDir
}

func (c *MetricsCollect) listExistedMetrics(dir string) (map[string]struct{}, error) {
	lookup := make(map[string]struct{})
	if !c.continues {
		return lookup, nil
	}
	ds, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, d := range ds {
		if d.IsDir() {
			continue
		}
		lookup[strings.TrimSuffix(d.Name(), filepath.Ext(d.Name()))] = struct{}{}
	}
	return lookup, nil
}

func parseTimeRange(scrapeStart, scrapeEnd string) ([]string, int64, error) {
	currTime := time.Now()

	end := scrapeEnd
	if end == "" {
		end = currTime.Format(time.RFC3339)
	}
	tsEnd, err := utils.ParseTime(end)
	if err != nil {
		return nil, 0, err
	}

	begin := scrapeStart
	if begin == "" {
		begin = tsEnd.Add(time.Duration(-1) * time.Hour).Format(time.RFC3339)
	}
	tsStart, err := utils.ParseTime(begin)
	if err != nil {
		return nil, 0, err
	}

	// split time into smaller ranges to avoid querying too many data
	// in one request
	ts := []string{tsStart.Format(time.RFC3339)}
	block := time.Second * 3600 * 2
	cursor := tsStart
	for {
		if cursor.After(tsEnd) {
			ts = append(ts, tsEnd.Format(time.RFC3339))
			break
		}
		next := cursor.Add(block)
		if next.Before(tsEnd) {
			ts = append(ts, next.Format(time.RFC3339))
		} else {
			ts = append(ts, tsEnd.Format(time.RFC3339))
			break
		}
		cursor = next
	}

	return ts, tsEnd.Unix() - tsStart.Unix(), nil
}
