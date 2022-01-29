package dispatch

import (
	"context"
	"github.com/prometheus/prometheus/prompb"
)

type VictoriaMetricsDispatcher struct {
	vimEndpoint string
	source      <-chan prompb.TimeSeries
}

func NewVictoriaMetricsDispatcher() (*VictoriaMetricsDispatcher, error) {
	vd := &VictoriaMetricsDispatcher{}

	return vd, nil
}

func (vd *VictoriaMetricsDispatcher) Start(ctx context.Context) error {

	for {
		select {
		case series, ok := <-vd.source:
			if !ok {
				//	process last step and return
				return nil
			}
			// process data
		case <-ctx.Done():
			// process last step and return
			return nil

		}
	}
}
