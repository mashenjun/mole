package dispatch

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/golang/snappy"
	"github.com/pingcap/log"
	"github.com/prometheus/prometheus/prompb"
	"go.uber.org/zap"
)

const (
	writePath = "/api/v1/write"
)

type VictoriaMetricsDispatcher struct {
	endpoint string
	source   <-chan *prompb.TimeSeries
}

func NewVictoriaMetricsDispatcher(endpoint string, source <-chan *prompb.TimeSeries) (*VictoriaMetricsDispatcher, error) {
	vd := &VictoriaMetricsDispatcher{
		endpoint: endpoint,
		source:   source,
	}

	return vd, nil
}

func (vd *VictoriaMetricsDispatcher) Start(ctx context.Context) error {

	for {
		select {
		case series, ok := <-vd.source:
			if !ok {
				return nil
			}
			// process series
			log.Debug("dispatch receiving ts", zap.Int("label len", len(series.Labels)))
			req := prompb.WriteRequest{
				Timeseries: []prompb.TimeSeries{*series},
			}
			data, err := req.Marshal()
			if err != nil {
				log.Error("write request marshal failed", zap.Error(err))
				continue
			}
			if err := vd.sink(data); err != nil {
				log.Error("sink data failed", zap.Error(err))
				continue
			}

		case <-ctx.Done():
			// process last step and return
			return nil
		}
	}
}

func (vd *VictoriaMetricsDispatcher) sink(data []byte) error {
	body := snappy.Encode(nil, data)

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s%s", vd.endpoint, writePath), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-protobuf")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Error("request vm failed", zap.Error(err))
		return err
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("send data to vm failed, err: %v", resp.Status)
	}

	return nil
}
