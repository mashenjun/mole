package proto

import (
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/labels"
)

type CSVMsg struct {
	GroupID string
	Data    []string
}

// TODO: define in a better way
type MetricsSampleMsg struct {
	Labels    labels.Labels
	Value     model.SampleValue
	Timestamp model.Time
}

type SortableMetricsSampleMsg []MetricsSampleMsg

func (ss SortableMetricsSampleMsg) Len() int      { return len(ss) }
func (ss SortableMetricsSampleMsg) Swap(i, j int) { ss[i], ss[j] = ss[j], ss[i] }
func (ss SortableMetricsSampleMsg) Less(i, j int) bool {
	return ss[i].Timestamp.Before(ss[j].Timestamp)
}

