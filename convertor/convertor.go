package convertor

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
}

type MetricsMatrixConvertorOpt func(*MetricsMatrixConvertor) error

func NewMetricsMatrixConvertor(opts... MetricsMatrixConvertorOpt) (*MetricsMatrixConvertor, error) {
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

func (c *MetricsMatrixConvertor) GetSink() <-chan []string {
	return c.sink
}

// Convert converts metrics json to csv to help `numpy` to do data processing, a very native implementation
func (c *MetricsMatrixConvertor) Convert(input string) error {
	defer close(c.sink)
	// 1. read input file and json marshal to metrics struct
	// 2. select interested metrics by labels and write to csv
	source, err := os.Open(input)
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
		if err := c.filterOnLabel(scanner.Bytes(), nil); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func (c *MetricsMatrixConvertor) filterOnLabel(b []byte, filter model.LabelSet) error {
	resp := MetricsResp{}
	if err := json.Unmarshal(b, &resp); err != nil {
		return err
	}
	matrix, ok := resp.Data.v.(model.Matrix)
	if !ok {
		return fmt.Errorf("type %t is not supported", resp.Data.v)
	}
	// TODO: each sample series may have different time point.
	csvHeader := extractHeader(matrix)
	c.sink <- csvHeader
	align, total := checkAlign(matrix)
	if align {
		idx := 0
		for idx < total {
			row := make([]string, 0)
			for _, sampleStream := range matrix {
				pair := sampleStream.Values[idx]
				if !c.inRange(pair.Timestamp.Time()){
					continue
				}
				if len(row) == 0 {
					// append timestamp first
					row = append(row, strconv.FormatInt(pair.Timestamp.Unix(), 10))
				}
				row = append(row, pair.Value.String())
			}
			c.sink <- row
			idx++
		}
	}else {
		fmt.Println("not aligned")
		// TODO: case that timestamp is not aligned
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

