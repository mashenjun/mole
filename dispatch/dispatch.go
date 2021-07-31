package dispatch

import (
	"context"
	"fmt"
	"github.com/mashenjun/mole/utils"
	"io"
	"os"
	"path/filepath"
)
// Useless package

type MetricMsg struct {
	Name    string
	Handler io.Reader
}

func NewMetricMsg(name string, handler io.Reader) *MetricMsg {
	return &MetricMsg{
		Name:    name,
		Handler: handler,
	}
}

type MetricDispatcher struct {
	outputDir string
	source    <-chan *MetricMsg
	fileFlag  int
	merge     bool
	metricCnt map[string]int
}

func NewMetricDispatcher(dir string, source <-chan *MetricMsg, merge bool) (*MetricDispatcher, error) {
	md := &MetricDispatcher{
		outputDir: dir,
		source:    source,
		merge:     merge,
		metricCnt: make(map[string]int),
	}
	if merge {
		md.fileFlag = os.O_RDWR | os.O_CREATE | os.O_APPEND
	} else {
		md.fileFlag = os.O_RDWR | os.O_CREATE | os.O_TRUNC
	}
	if err := utils.EnsureDir(md.outputDir); err != nil {
		return nil, err
	}
	return md, nil
}

func (md *MetricDispatcher) Start(ctx context.Context) error {
	for {
		select {
		case msg, ok := <-md.source:
			if !ok {
				return nil
			}
			if err := md.process(msg); err != nil {
				// log error
				fmt.Println(err)
			}
		case <-ctx.Done():
			return nil
		}
	}
}

func (md *MetricDispatcher) genFilename(name string) string {
	if md.merge {
		return name
	}
	md.metricCnt[name]++
	return fmt.Sprintf("%s_%v", name, md.metricCnt[name])
}

func (md *MetricDispatcher) process(msg *MetricMsg) error {
	dst, err := os.OpenFile(filepath.Join(md.outputDir, md.genFilename(msg.Name)), md.fileFlag, 0644)
	if err != nil {
		// log err and continue
		fmt.Printf("open file error: %+v\n", err)
		return err
	}
	defer dst.Close()
	if _, err := io.Copy(dst, msg.Handler); err != nil {
		// log err
		fmt.Printf("write metric error: %+v\n", err)
		return err
	}
	return nil
}