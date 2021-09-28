package prom

import "github.com/prometheus/common/model"

type IGap interface {
	InGap(idx int) bool
	GetAlignedIdx(metricsName string, idx int) int
	GetGapInfo() ([]string, int)
}

type NoGap struct{}

func (ng *NoGap) InGap(int) bool {
	return false
}

func (ng *NoGap) GetAlignedIdx(_ string, idx int) int {
	return idx
}

func (ng *NoGap) GetGapInfo() ([]string, int) {
	return nil, 0
}

type MergedGap struct {
	// todo
	width int
	overview []int           // gap overview information
	slots    map[string][]int // gap information for each metric
}

func (mg *MergedGap) InGap(idx int) bool {
	if idx < 0 || idx >= len(mg.overview) {
		return true
	}
	return mg.overview[idx] < mg.width
}

func (mg *MergedGap) GetAlignedIdx(name string, idx int) int {
	if slot, ok := mg.slots[name]; ok {
		return slot[idx]
	}
	return idx
}

func (mg *MergedGap) GetGapInfo() ([]string, int) {
	gapSlot := make([]int, 0)
	for idx, cnt := range mg.overview {
		if cnt > 0 && cnt < mg.width {
			gapSlot = append(gapSlot, idx)
		}
	}
	names := make([]string, 0, len(mg.slots))
	for name, slot := range mg.slots {
		for _, v := range gapSlot {
			if slot[v] == -1 {
				names = append(names, name)
				break
			}
		}
		//names = append(names, name)
	}
	return names, len(gapSlot)
}

type MergedGapBuilder struct {
	startTs  int64
	step     int64
	width    int
	size     int
	slots    map[string][]int
	slotsCnt []int
}

func NewMergedGapBuilder(width int, startTs int64, step int64, size int) *MergedGapBuilder {
	return &MergedGapBuilder{
		startTs:  startTs,
		step:     step,
		width:    width,
		size:     size,
		slots:    make(map[string][]int),
		slotsCnt: make([]int, size),
	}
}

func (gb *MergedGapBuilder) Push(name string, pairs []model.SamplePair) {
	slot := make([]int, gb.size)
	for i := range slot {
		slot[i] = -1
	}
	for idx, value := range pairs {
		i := tsToSlot(gb.startTs, value.Timestamp.Unix(), gb.step)
		slot[i] = idx
		gb.slotsCnt[i]++
	}
	gb.slots[name] = slot
}

func (gb *MergedGapBuilder) Build() *MergedGap {
	mg := &MergedGap{
		width: gb.width,
		overview: gb.slotsCnt,
		slots:    gb.slots,
	}
	//for i, cnt := range gb.slotsCnt {
	//	mg.overview[i] = cnt< gb.width
	//}
	return mg
}

func tsToSlot(startTs, ts, step int64) int {
	if ts <= startTs {
		return 0
	}
	return int((ts - startTs) / step)
}
