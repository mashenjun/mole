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
	"sync"
	"time"
)

const (
	metricStep = 15 // use 15s step
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
	targetRecord []MetricsRecord
	concurrency  int
	scrapeBegin  string // time range to filter metrics.
	scrapeEnd    string // time range to filter metrics.
	cli          *http.Client
	outputDir    string // dir where the metrics data will be stored.
	merge        bool
	fileFlag     int // file flag used to open file
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

func NewMetricsCollect(opts ...MetricsCollectOpt) (*MetricsCollect, error) {
	mc := &MetricsCollect{
		targetRecord: make([]MetricsRecord, 0),
	}
	for _, opt := range opts {
		if err := opt(mc); err != nil {
			return nil, err
		}
	}
	fmt.Printf("new merge %+v\n", mc.merge)
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
	Host string
	Port string
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
	total := len(c.rawMetrics) + len(c.cookedRecord)
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
		if err := utils.EnsureDir(c.outputDir, c.genDirName(prom)); err != nil {
			bars[key].UpdateDisplay(&progress.DisplayProps{
				Prefix: fmt.Sprintf("  - Query server %s: %s", key, err),
				Mode:   progress.ModeError,
			})
			return err
		}

		for _, r := range c.targetRecord {
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
			}(tl.Get(), r.Record, r.Expr )
		}

		for _, cr := range c.cookedRecord {
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
			}(tl.Get(), cr.Record, cr.Expr)
		}
	}
	tl.Wait()
	return nil
}

func (c *MetricsCollect) getMetricList(prom string) ([]string, error) {
	if len(c.rawMetrics) > 0 {
		// use url encode in case we use prom query in metrics list.
		//for i := range c.metrics {
		//	c.metrics[i] = url.QueryEscape(c.metrics[i])
		//}
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
				resp, err := c.cli.PostForm(
					fmt.Sprintf("%s/api/v1/query_range", promAddr),
					url.Values{
						"query": {expr},
						"start": {ts[i]},
						"end":   {ts[i+1]},
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
				// the following implement is write the response to file
				filename := c.genFileName(mtc, i)
				topoDir := c.genDirName(prom)
				dst, err := os.OpenFile(
					filepath.Join(
						c.outputDir, topoDir, filename,
					), c.fileFlag, 0644)
				if err != nil {
					fmt.Printf("collect metric %s: %s, retry...\n", mtc, err)
				}
				defer dst.Close()

				_, err = io.Copy(dst, resp.Body)
				if err != nil {
					fmt.Printf("write metric %s to file error: %s, retry...\n", mtc, err)
					return err
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

func (c *MetricsCollect) genDirName(ep Endpoint) string {
	return fmt.Sprintf("%s-%v", ep.Host, ep.Port)
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
			ts = append(ts, cursor.Format(time.RFC3339))
		} else {
			ts = append(ts, tsEnd.Format(time.RFC3339))
			break
		}
		cursor = next
	}

	return ts, tsEnd.Unix() - tsStart.Unix(), nil
}
