package prom

import "github.com/prometheus/common/model"

type IGap interface {
	InAnyGap(idx int) bool
	InAllGap(idx int) bool
	GetAlignedIdx(metricsName string, idx int) int
	GetGapInfo() ([]string, int)
	GetFirstTsUnix() int64
}

type NoGap struct {
	firstTsUnix int64
}

func (ng *NoGap) InAnyGap(int) bool {
	return false
}

func (ng *NoGap) InAllGap(int) bool {
	return false
}

func (ng *NoGap) GetAlignedIdx(_ string, idx int) int {
	return idx
}

func (ng *NoGap) GetGapInfo() ([]string, int) {
	return nil, 0
}

func (ng *NoGap) GetFirstTsUnix() int64 {
	return ng.firstTsUnix
}

type MergedGap struct {
	// todo
	firstTsUnix    int64
	gapStreamCnt   int
	totalStreamCnt int
	overview       []int            // idx is slot, element means how many stream has value on this slot
	slots          map[string][]int // gap information for each metric,
}

func (mg *MergedGap) InAnyGap(idx int) bool {
	if idx < 0 || idx >= len(mg.overview) {
		return true
	}
	return mg.overview[idx] < mg.gapStreamCnt
}

func (mg *MergedGap) InAllGap(idx int) bool {
	if idx < 0 || idx >= len(mg.overview) {
		return true
	}
	return mg.overview[idx] == 0 && mg.gapStreamCnt == mg.totalStreamCnt
}

// GetAlignedIdx return -1 if the given metrics has gap on given idx
func (mg *MergedGap) GetAlignedIdx(name string, idx int) int {
	if slot, ok := mg.slots[name]; ok {
		return slot[idx]
	}
	return idx
}

func (mg *MergedGap) GetGapInfo() ([]string, int) {
	gapSlot := make([]int, 0)
	for idx, cnt := range mg.overview {
		if cnt > 0 && cnt < mg.gapStreamCnt {
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

func (ng *MergedGap) GetFirstTsUnix() int64 {
	return ng.firstTsUnix
}

type MergedGapBuilder struct {
	firstTsUnix    int64
	step           int64
	gapStreamCnt   int
	totalStreamCnt int
	size           int
	slots          map[string][]int
	slotsCnt       []int
}

func NewMergedGapBuilder(gapStreamCnt int, firstTsUnix int64, step int64, size int, totalStreamCnt int) *MergedGapBuilder {
	return &MergedGapBuilder{
		firstTsUnix:    firstTsUnix,
		step:           step,
		gapStreamCnt:   gapStreamCnt,
		totalStreamCnt: totalStreamCnt,
		size:           size,
		slots:          make(map[string][]int),
		slotsCnt:       make([]int, size),
	}
}

func (gb *MergedGapBuilder) Push(name string, pairs []model.SamplePair) {
	slot := make([]int, gb.size)
	for i := range slot {
		slot[i] = -1
	}
	for idx, value := range pairs {
		i := tsToSlot(gb.firstTsUnix, value.Timestamp.Unix(), gb.step)
		slot[i] = idx
		gb.slotsCnt[i]++
	}
	gb.slots[name] = slot
}

func (gb *MergedGapBuilder) Build() *MergedGap {
	mg := &MergedGap{
		firstTsUnix:    gb.firstTsUnix,
		gapStreamCnt:   gb.gapStreamCnt,
		totalStreamCnt: gb.totalStreamCnt,
		overview:       gb.slotsCnt,
		slots:          gb.slots,
	}
	return mg
}

func tsToSlot(startTs, ts, step int64) int {
	if ts <= startTs {
		return 0
	}
	return int((ts - startTs) / step)
}
