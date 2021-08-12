package keyviz

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/mashenjun/mole/consts"
	"github.com/mashenjun/mole/proto"
	"github.com/pingcap/tidb-dashboard/pkg/keyvisual/decorator"
	"github.com/pingcap/tidb-dashboard/pkg/keyvisual/matrix"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"
)

type HeatmapConvertor struct {
	sink chan *proto.CSVMsg
	// 左闭右闭
	from int64
	to int64
	// native way to define the filter rule
	filterTable map[string]map[string]struct{} // db name -> (table name -> struct)
	input string
	split bool
}

type HeatmapConvertorConvertorOpt func(*HeatmapConvertor) error

func NewHeatmapConvertor(opts... HeatmapConvertorConvertorOpt) (*HeatmapConvertor, error) {
	mmc := &HeatmapConvertor{
		sink:        make(chan *proto.CSVMsg, 42),
		filterTable: make(map[string]map[string]struct{}),
	}
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

func WithInput(input string) HeatmapConvertorConvertorOpt {
	return func(convertor *HeatmapConvertor) error {
		if len(input) == 0 {
			return errors.New("input is empty")
		}
		convertor.input = input
		return nil
	}
}

func WithSplit() HeatmapConvertorConvertorOpt {
	return func(convertor *HeatmapConvertor) error {
		convertor.split = true
		return nil
	}
}

// SetFilterRules is more easy to use in caller side.
func (c *HeatmapConvertor) SetFilterRules(rules []string) {
	var alwaysMath = map[string]struct{}{
		"*": {},
	}
	for _, rule := range rules {
		sl := strings.Split(strings.TrimSuffix(rule, ":"), ":")
		if len(sl) == 1 {
			c.filterTable[sl[0]] = alwaysMath
		}else {
			if _, ok := c.filterTable[sl[0]]; !ok {
				c.filterTable[sl[0]] = make(map[string]struct{})
			}
			c.filterTable[sl[0]][sl[1]] = struct{}{}
		}
	}
}

func (c *HeatmapConvertor) GetSink() <-chan *proto.CSVMsg {
	return c.sink
}

func (c *HeatmapConvertor) Convert() error {
	defer close(c.sink)
	// 1. read input file and json marshal to heatmap matrix struct
	// 2. convert heatmap data to csv format
	source, err := os.Open(c.input)
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
	if c.split {
		return c.filterAndSplit(&mat)
	}else {
		return c.filterAndSink(&mat)
	}
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
			if !c.isTarget(mat.KeyAxis[j].Labels) {
				continue
			}
			row = append(row, strconv.FormatUint(data[i][j], 10))
		}
		c.sink <- &proto.CSVMsg{
			Data:    row,
		}
	}
	return nil
}

func (c *HeatmapConvertor) filterAndSplit(mat *matrix.Matrix) error {
	// csv header row is not necessary for heatmap data
	data, typ ,err := extractData(mat)
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return nil
	}
	groupIdxs := c.groupIndexByTable(mat.KeyAxis, data, typ)
	for i:=0; i < len(data); i++ {
		ts := mat.TimeAxis[i]
		if !c.inRange(ts) {
			continue
		}

		for _, gi := range groupIdxs {
			row := []string{strconv.FormatInt(ts, 10)}
			for _, j := range gi.index {
				row = append(row, strconv.FormatUint(data[i][j],10))
			}
			c.sink <- &proto.CSVMsg{
				GroupID: gi.groupID,
				Data:    row,
			}
		}
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

// genGroupID generate group id by using the first two labels
func genGroupID(typ string, labels []string) string{
	if len(labels) == 0 {
		return ""
	}
	if len(labels) == 1 {
		return labels[0]
	}
	return fmt.Sprintf("%v:%v:%v", typ, labels[0], labels[1])
}

type groupIndex struct {
	groupID string
	index []int
}

type GroupIndexConstructor struct {
	data []groupIndex
	cursor int
}

func NewGroupIndexConstructor() *GroupIndexConstructor {
	return &GroupIndexConstructor{
		data:   make([]groupIndex, 0),
		cursor: -1,
	}
}

func (gic *GroupIndexConstructor) Append(group string, idx int) {
	if gic.cursor == -1 {
		gic.data = append(gic.data, groupIndex{
			groupID: group,
			index:   []int{idx},
		})
		gic.cursor = 0
		return
	}
	// data is not empty
	if group == gic.data[gic.cursor].groupID {
		gic.data[gic.cursor].index = append(gic.data[gic.cursor].index, idx)
		return
	}
	gic.data = append(gic.data, groupIndex{
		groupID: group,
		index:   []int{idx},
	})
	gic.cursor++
}

func (gic *GroupIndexConstructor) Result() []groupIndex {
	return gic.data
}

// the return lookup is read only and has order
func (c *HeatmapConvertor) groupIndexByTable (keys []decorator.LabelKey, data [][]uint64, tpy string) []groupIndex {
	gic := NewGroupIndexConstructor()

	for idx := range data[0] {
		lk := keys[idx]
		if !c.isTarget(lk.Labels){
			continue
		}
		groupID := genGroupID(tpy, lk.Labels)
		gic.Append(groupID, idx)
	}
	return gic.Result()
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

// TODO move filter function to filter struct
func (c *HeatmapConvertor) isTarget(labels []string) bool {
	if len(labels) <= 1 {
		return false
	}
	// mysql table is for meta data, skip
	if labels[0] == "mysql" || labels[0] == "meta" {
		return false
	}
	if len(c.filterTable) == 0 {
		return true
	}
	if tbl, ok := c.filterTable[labels[0]]; !ok {
		return false
	}else {
		if _, find := tbl["*"]; find {
			fmt.Println(labels)
			return true
		}
		if _, find := tbl[labels[1]]; !find {
			return false
		}
	}
	return true
}