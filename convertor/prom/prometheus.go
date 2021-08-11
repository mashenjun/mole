package prom

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/cznic/mathutil"
	"github.com/prometheus/common/model"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

type MetricsMatrixConvertor struct {
	sink chan []string
	// 左闭右闭
	from time.Time
	to time.Time
	filterLabels []model.LabelSet
	input string
	headerSend bool
}

type MetricsMatrixConvertorOpt func(*MetricsMatrixConvertor) error

func NewMetricsMatrixConvertor(opts...MetricsMatrixConvertorOpt) (*MetricsMatrixConvertor, error) {
	mmc := &MetricsMatrixConvertor{sink: make(chan []string, 42)}
	for _, opt := range opts {
		if err := opt(mmc); err != nil {
			return nil, err
		}
	}
	return mmc, nil
}

func WithTimeRange(begin string, end string) MetricsMatrixConvertorOpt {
	return func(convertor *MetricsMatrixConvertor) error {
		if len(begin) > 0 {
			ts, err := time.Parse(time.RFC3339, begin)
			if err != nil {
				return err
			}
			convertor.from = ts
		}
		if len(end) > 0 {
			ts, err := time.Parse(time.RFC3339, end)
			if err != nil {
				return err
			}
			convertor.to = ts
		}
		return nil
	}
}

func WithInput(input string) MetricsMatrixConvertorOpt {
	return func(convertor *MetricsMatrixConvertor) error {
		convertor.input = input
		return nil
	}
}

// SetFilterLabels is more easy to use in caller side.
func (c *MetricsMatrixConvertor) SetFilterLabels(labels []model.LabelSet) {
	c.filterLabels = labels
}

func (c *MetricsMatrixConvertor) GetSink() <-chan []string {
	return c.sink
}

// Convert converts metrics json to csv to help `numpy` to do data processing, a very native implementation
func (c *MetricsMatrixConvertor) Convert() error {
	defer close(c.sink)
	// 1. read input file and json marshal to metrics struct
	// 2. select interested metrics by labels and write to csv
	source, err := os.Open(c.input)
	if err != nil {
		return err
	}
	defer source.Close()
	fInfo, err := source.Stat()
	if err != nil {
		return err
	}

	// read the source file line by line
	// set the buffer since the MaxScanTokenSize = 64kb which is not large enough.

	scanner := bufio.NewScanner(source)
	if fInfo.Size() > bufio.MaxScanTokenSize {
		scanner.Buffer(make([]byte, 4096), int(fInfo.Size()))
	}
	for scanner.Scan() {
		if err := c.filterAndSink(scanner.Bytes(), nil); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func (c *MetricsMatrixConvertor) filterAndSink(b []byte, filter model.LabelSet) error {
	resp := MetricsResp{}
	if err := json.Unmarshal(b, &resp); err != nil {
		return err
	}
	matrix, ok := resp.Data.v.(model.Matrix)
	if !ok {
		return fmt.Errorf("type %t is not supported", resp.Data.v)
	}
	// TODO: each sample series may have different time point.
	if !c.headerSend {
		c.sink <- extractHeader(matrix)
		c.headerSend = true
	}
	align, total := checkAlign(matrix)
	if !align {
		fmt.Println("not aligned")

	}
	idx := 0
	for idx < total {
		row := make([]string, 0)
		for _, sampleStream := range matrix {
			// filter on label set
			if !c.matchLabels(model.LabelSet(sampleStream.Metric)) {
				continue
			}
			// filter on timestamp
			pair := sampleStream.Values[idx]
			if !c.inRange(pair.Timestamp.Time()){
				continue
			}
			// append timestamp first
			if len(row) == 0 {
				row = append(row, strconv.FormatInt(pair.Timestamp.Unix(), 10))
			}
			// append data
			row = append(row, pair.Value.String())
		}
		c.sink <- row
		idx++
	}
	return nil
}

func (c *MetricsMatrixConvertor) inRange(t time.Time) bool {
	if c.from.IsZero() && c.to.IsZero() {
		return true
	}
	if !c.to.IsZero() && t.After(c.to) {
		return false
	}
	if !c.from.IsZero() && t.Before(c.from) {
		return false
	}
	return true
}

func (c *MetricsMatrixConvertor) matchLabels(target model.LabelSet) bool {
	if len(c.filterLabels) == 0 {
		return true
	}
	for _, query := range c.filterLabels {
		if match(query, target) {
			return true
		}
	}
	return false
}

func match(query model.LabelSet, target model.LabelSet) bool {
	for name, v := range query{
		value, ok := target[name]
		if ok && value != v {
			return false
		}
	}
	return true
}

func checkAlign(matrix model.Matrix) (bool, int) {
	if len(matrix) == 0 {
		return true, 0
	}
	if len(matrix) == 1 {
		return true, len(matrix[0].Values)
	}
	align, size:= true, len(matrix[0].Values)
	for _, sp := range matrix {
		if align && len(sp.Values) != size {
			align = false
		}
		size = mathutil.Min(size, len(sp.Values))
	}
	return align, size
}


// the header is the same order in the json file.
func extractHeader(matrix model.Matrix) []string {
	labelNames := make(model.LabelNames, 0)
	header := []string{"timestamp"}
	for i, sp := range matrix {
		if i == 0 {
			// collect and sort label name first
			for lname := range sp.Metric {
				if string(lname) == "__name__" || string(lname) == "job" {
					continue
				}
				labelNames = append(labelNames, lname)
			}
			sort.Sort(labelNames)
		}
		lvales := make([]string, len(labelNames))
		for i, lname := range labelNames {
			lvales[i] = strings.Split(string(sp.Metric[lname]), ":")[0]
		}
		header = append(header, strings.Join(lvales, ":"))
	}
	return header
}

