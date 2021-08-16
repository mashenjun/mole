package dispatch

import (
	"context"
	"github.com/mashenjun/mole/proto"
	"github.com/prometheus/prometheus/tsdb"
	"time"
)

type NoopLogger struct{}

func (nlg *NoopLogger) Log(keyvals ...interface{}) error {
	return nil
}

type TSDBBlockDispatcher struct {
	outputDir string
	source    <-chan proto.MetricsSampleMsg
}

func NewTSDBBlockDispatcher(dir string, source <-chan proto.MetricsSampleMsg) (*TSDBBlockDispatcher, error) {
	d := &TSDBBlockDispatcher{
		outputDir: dir,
		source:    source,
	}
	return d, nil
}

func (bd *TSDBBlockDispatcher) Start(ctx context.Context) error {
	db, err :=  tsdb.Open(bd.outputDir, &NoopLogger{}, nil, &tsdb.Options{
		RetentionDuration: int64(3650 * 24 * time.Hour / time.Millisecond),
		MinBlockDuration:  int64(2 * time.Hour / time.Millisecond),
	}, tsdb.NewDBStats())
	if err != nil {
		return err
	}
	app := db.Appender(ctx)
	for {
		select {
		case ms, ok := <-bd.source:
			if !ok {
				if err := app.Commit(); err != nil {
					return err
				}
				if err := db.Close(); err != nil {
					return err
				}
				return nil
			}
			ts := ms.Timestamp.UnixNano()/1e6
			if _, err := app.Append(0, ms.Labels, ts, float64(ms.Value)); err != nil {
				return err
			}

		case <-ctx.Done():
			if err := app.Commit(); err != nil {
				return err
			}
			if err := db.Close(); err != nil {
				return err
			}
			return nil
		}
	}
}
