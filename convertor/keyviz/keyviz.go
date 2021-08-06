package keyviz

import (
	"encoding/json"
	"errors"
	"github.com/mashenjun/mole/consts"
	"github.com/pingcap/tidb-dashboard/pkg/keyvisual/matrix"
	"io/ioutil"
	"os"
	"strconv"
	"time"
)

type HeatmapConvertor struct {
	sink chan []string
	// 左闭右闭
	from int64
	to int64

}

type HeatmapConvertorConvertorOpt func(*HeatmapConvertor) error

func NewHeatmapConvertor(opts... HeatmapConvertorConvertorOpt) (*HeatmapConvertor, error) {
	mmc := &HeatmapConvertor{sink: make(chan []string, 42)}
	for _, opt := range opts {
		if err := opt(mmc); err != nil {
			return nil, err
		}
	}
	return mmc, nil
}

func WithTimeRange(begin string, end string) HeatmapConvertorConvertorOpt {
	return func(convertor *HeatmapConvertor) error {
		if len(begin) > 0 {
			ts, err := time.Parse(time.RFC3339, begin)
			if err != nil {
				return err
			}
			convertor.from = ts.Unix()
		}
		if len(end) > 0 {
			ts, err := time.Parse(time.RFC3339, end)
			if err != nil {
				return err
			}
			convertor.to = ts.Unix()
		}
		return nil
	}
}

func (c *HeatmapConvertor) GetSink() <-chan []string {
	return c.sink
}

func (c *HeatmapConvertor) Convert(input string) error {
	defer close(c.sink)
	// 1. read input file and json marshal to heatmap matrix struct
	// 2. convert heatmap data to csv format
	source, err := os.Open(input)
	if err != nil {
		return err
	}
	defer source.Close()
	bs, err := ioutil.ReadAll(source)
	if err != nil {
		return err
	}
	mat := matrix.Matrix{}
	if err := json.Unmarshal(bs, &mat); err != nil {
		return err
	}
	// convert to csv row format
	return c.filterAndSink(&mat)
}

func (c *HeatmapConvertor) filterAndSink(mat *matrix.Matrix) error {
	// csv header row is not necessary for heatmap data
	data, _ ,err := extractData(mat)
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return nil
	}
	for i:=0 ; i < len(data); i++ {
		ts := mat.TimeAxis[i]
		if !c.inRange(ts) {
			continue
		}
		row := []string{strconv.FormatInt(ts, 10)}
		for j := 0; j <= len(data[i]); j++ {
			if !isTarget(mat.KeyAxis[j].Labels) {
				continue
			}
			row = append(row, strconv.FormatUint(data[i][j], 10))
		}
		c.sink <- row
	}
	return nil
}

func extractData(mat *matrix.Matrix) ([][]uint64, string, error) {
	if data, ok := mat.DataMap[consts.HeatMapTypeReadKeys]; ok {
		return data, consts.HeatMapTypeReadKeys, nil
	}
	if data, ok := mat.DataMap[consts.HeatMapTypeReadBytes]; ok {
		return data, consts.HeatMapTypeReadBytes,nil
	}
	if data, ok := mat.DataMap[consts.HeatMapTypeWriteKeys]; ok {
		return data, consts.HeatMapTypeWriteKeys,nil
	}
	if data, ok := mat.DataMap[consts.HeadMapTypeWriteBytes]; ok {
		return data, consts.HeadMapTypeWriteBytes, nil
	}
	return nil, "", errors.New("heatmap data is empty")
}

func (c *HeatmapConvertor) inRange(ts int64) bool {
	if c.from == 0 && c.to == 0 {
		return true
	}
	if c.to > 0 && ts > c.to {
		return false
	}
	if c.from > 0 && ts < c.from {
		return false
	}
	return true
}

func isTarget(labels []string) bool {
	if len(labels) == 0 {
		return false
	}
	// mysql table is for meta data, skip
	if labels[0] == "mysql" || labels[0] == "meta" {
		return false
	}
	return true
}
