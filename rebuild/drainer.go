package rebuild

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"github.com/pingcap/log"
	"github.com/prometheus/prometheus/prompb"
	"go.uber.org/zap"
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
	//source, err := os.Open(input)
	//if err != nil {
	//	return nil, err
	//}
	//fInfo, err := source.Stat()
	//if err != nil {
	//	return nil, err
	//}
	//scanner := bufio.NewScanner(source)
	//if fInfo.Size() > bufio.MaxScanTokenSize {
	//	scanner.Buffer(make([]byte, 4096), int(fInfo.Size()))
	//}
	return &CloseableScanner{
		path: input,
	}, nil
}

func (cs *CloseableScanner) Open() error {
	if len(cs.path) == 0 {
		return fmt.Errorf("can not open colseable scanner with empty path")
	}
	source, err := os.Open(cs.path)
	if err != nil {
		return err
	}
	cs.File = source
	fInfo, err := source.Stat()
	if err != nil {
		return err
	}
	scanner := bufio.NewScanner(source)
	if fInfo.Size() > bufio.MaxScanTokenSize {
		scanner.Buffer(make([]byte, 4096), int(fInfo.Size()))
	}
	cs.Scanner = scanner
	return nil
}

type MetricsMatrixDrainer struct {
	pipe     chan proto.MetricsSampleMsg
	pipe2    chan *prompb.TimeSeries
	inputDir string

	scanners []*CloseableScanner // each file under the inputDir has a scanner
}

type MetricsMatrixDrainerOpt func(*MetricsMatrixDrainer) error

// TODO: the project construe is not good
func NewMetricsMatrixDrainer(opts ...MetricsMatrixDrainerOpt) (*MetricsMatrixDrainer, error) {
	mmd := &MetricsMatrixDrainer{
		pipe:     make(chan proto.MetricsSampleMsg, 42),
		pipe2:    make(chan *prompb.TimeSeries, 42),
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

func (c *MetricsMatrixDrainer) GetSink2() <-chan *prompb.TimeSeries {
	return c.pipe2
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

func (c *MetricsMatrixDrainer) Start2(ctx context.Context) error {
	defer c.close()
	for _, scanner := range c.scanners {
		fmt.Printf("processing %s\n", scanner.path)
		if err := c.drainScanner(scanner); err != nil {
			return err
		}
	}
	return nil
}

func (c *MetricsMatrixDrainer) drainScanner(scanner *CloseableScanner) error {
	if err := scanner.Open(); err != nil {
		return err
	}
	defer scanner.Close()
	for scanner.Scan() {
		if err := scanner.Err(); err != nil {
			log.Error("scanner failed", zap.String("path", scanner.path), zap.Error(err))
			return err
		}
		resp := proto.MetricsResp{}

		if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
			log.Error("json marshall failed, skip this metrics", zap.String("path", scanner.path))
			continue
		}
		matrix, err := resp.Data.ToMatrix()
		if err != nil {
			log.Error("to matrix failed", zap.Error(err))
			return err
		}
		log.Debug("debug matrix", zap.Int("len", len(matrix)))
		for _, sp := range matrix {
			timeseries := &prompb.TimeSeries{
				Labels:  nil,
				Samples: nil,
			}
			for name, value := range sp.Metric {
				timeseries.Labels = append(timeseries.Labels, prompb.Label{
					Name:  string(name),
					Value: string(value),
				})
			}
			for _, pair := range sp.Values {
				timeseries.Samples = append(timeseries.Samples, prompb.Sample{
					Value:     float64(pair.Value),
					Timestamp: pair.Timestamp.Unix() * 1e3,
				})
			}
			c.pipe2 <- timeseries
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
	close(c.pipe2)

	// TODO(shenjun): skip close
	//for _, sc := range c.scanners {
	//	_ = sc.Close()
	//}
}
