package rebuild

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mashenjun/mole/proto"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/labels"
)

type CloseableScanner struct {
	*os.File // used to close file
	*bufio.Scanner

	path string
}

func NewCloseableScanner(input string) (*CloseableScanner, error) {
	source, err := os.Open(input)
	if err != nil {
		return nil, err
	}
	fInfo, err := source.Stat()
	if err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(source)
	if fInfo.Size() > bufio.MaxScanTokenSize {
		scanner.Buffer(make([]byte, 4096), int(fInfo.Size()))
	}
	return &CloseableScanner{
		File:    source,
		Scanner: scanner,
		path:    input,
	}, nil
}

type MetricsMatrixDrainer struct {
	pipe     chan proto.MetricsSampleMsg
	inputDir string

	scanners []*CloseableScanner // each file under the inputDir has a scanner
}

type MetricsMatrixDrainerOpt func(*MetricsMatrixDrainer) error

// TODO: the project construe is not good
func NewMetricsMatrixDrainer(opts ...MetricsMatrixDrainerOpt) (*MetricsMatrixDrainer, error) {
	mmd := &MetricsMatrixDrainer{
		pipe:     make(chan proto.MetricsSampleMsg, 42),
		scanners: make([]*CloseableScanner, 0),
	}
	for _, opt := range opts {
		if err := opt(mmd); err != nil {
			return nil, err
		}
	}
	return mmd, nil
}

func WithInput(input string) MetricsMatrixDrainerOpt {
	return func(drainer *MetricsMatrixDrainer) error {
		ds, err := os.ReadDir(input)
		if err != nil {
			return err
		}
		for _, d := range ds {
			if d.IsDir() {
				continue
			}
			sc, err := NewCloseableScanner(filepath.Join(input, d.Name()))
			if err != nil {
				return err
			}
			drainer.scanners = append(drainer.scanners, sc)
		}
		drainer.inputDir = input
		return nil
	}
}

func (c *MetricsMatrixDrainer) GetSink() <-chan proto.MetricsSampleMsg {
	return c.pipe
}

func (c *MetricsMatrixDrainer) Start(ctx context.Context) error {
	defer c.close()
	for {
		eofCnt, err := c.oneRound()
		if err != nil {
			return err
		}
		if eofCnt == len(c.scanners) {
			break
		}
	}
	return nil
}

func (c *MetricsMatrixDrainer) oneRound() (int, error) {
	series := make(proto.SortableMetricsSampleMsg, 0)
	eofCnt := 0
	for _, sc := range c.scanners {
		if !sc.Scan() {
			if err := sc.Err(); err != nil {
				return eofCnt, err
			}
			eofCnt++
			continue
		}
		resp := proto.MetricsResp{}
		if err := json.Unmarshal(sc.Bytes(), &resp); err != nil {
			fmt.Printf("josn unmarshal %v err: %+v\n", sc.path, err)
			return eofCnt, err
		}
		matrix, err := resp.Data.ToMatrix()
		if err != nil {
			return eofCnt, err
		}
		for _, sp := range matrix {
			// get the labels form label set
			labelMap := make(map[string]string)
			for name, value := range sp.Metric {
				labelMap[string(name)] = string(value)
			}
			if _, ok := sp.Metric[model.MetricNameLabel]; !ok {
				labelMap[model.MetricNameLabel] = strings.TrimSuffix(filepath.Base(sc.Name()), ".json")
			}
			for _, pair := range sp.Values {
				series = append(series, proto.MetricsSampleMsg{
					Labels:    labels.FromMap(labelMap),
					Value:     pair.Value,
					Timestamp: pair.Timestamp,
				})
			}
		}
	}
	sort.Sort(series)
	for _, v := range series {
		c.pipe <- v
	}
	return eofCnt, nil
}

func (c *MetricsMatrixDrainer) close() {
	close(c.pipe)
	for _, sc := range c.scanners {
		_ = sc.Close()
	}
}
