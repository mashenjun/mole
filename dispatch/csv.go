package dispatch

import (
	"context"
	"encoding/csv"
	"os"
)

type CSVDispatcher struct {
	outputFile string
	source    <-chan []string
}

func NewCSVDispatcher(file string, source <-chan []string) (*CSVDispatcher, error) {
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
		case row, ok := <-md.source:
			if !ok {
				cw.Flush()
				return nil
			}
			if err := cw.Write(row); err != nil {
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
