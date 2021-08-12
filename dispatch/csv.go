package dispatch

import (
	"context"
	"encoding/csv"
	"github.com/mashenjun/mole/proto"
	"os"
)

type CSVDispatcher struct {
	outputFile string
	source    <-chan *proto.CSVMsg
}

func NewCSVDispatcher(file string, source <-chan *proto.CSVMsg) (*CSVDispatcher, error) {
	d := &CSVDispatcher{
		outputFile: file,
		source:    source,
	}
	return d, nil
}

func (md *CSVDispatcher) Start(ctx context.Context) error {
	file, err := os.Create(md.outputFile)

	if err != nil {
		return err
	}
	defer file.Close()
	cw := csv.NewWriter(file)
	lines := 0
	for {
		select {
		case msg, ok := <-md.source:
			if !ok {
				cw.Flush()
				return nil
			}
			if err := cw.Write(msg.Data); err != nil {
				return err
			}
			lines++
			if lines % 100 == 0 {
				cw.Flush()
			}
		case <-ctx.Done():
			return nil
		}
	}
}
