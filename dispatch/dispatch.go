package dispatch

import (
	"context"
	"encoding/csv"
	"fmt"
	"github.com/mashenjun/mole/proto"
	"github.com/mashenjun/mole/utils"
	"os"
	"path/filepath"
)

type CSVFile struct {
	*os.File // used to close file
	csvWriter *csv.Writer
}

type DirDispatcher struct {
	outputDir string
	source    <-chan *proto.CSVMsg
	lookup map[string]*CSVFile
}

func NewDirDispatcher(dir string, source <-chan *proto.CSVMsg) (*DirDispatcher, error) {
	md := &DirDispatcher{
		outputDir: dir,
		source:    source,
		lookup:  make(map[string]*CSVFile),
	}
	if err := utils.EnsureDir(md.outputDir); err != nil {
		return nil, err
	}
	return md, nil
}

func (dd *DirDispatcher) Start(ctx context.Context) error {
	for {
		select {
		case msg, ok := <-dd.source:
			if !ok {
				dd.close()
				return nil
			}
			if err := dd.process(msg); err != nil {
				// log error
				fmt.Println(err)
			}
		case <-ctx.Done():
			dd.close()
			return nil
		}
	}
}

// get csv writer from lookup, if not existed, create a new one
func (dd *DirDispatcher) getCSVWriter(name string) (*csv.Writer, error) {
	w, ok := dd.lookup[name]
	if !ok {
		dst, err := os.Create(filepath.Join(dd.outputDir, dd.genFilename(name)))
		if err != nil {
			return nil, err
		}
		csvW := csv.NewWriter(dst)
		dd.lookup[name] = &CSVFile{
			File:      dst,
			csvWriter: csvW,
		}
		return csvW, nil
	}
	return w.csvWriter, nil
}

func (dd *DirDispatcher) close() {
	for _, f := range dd.lookup {
		f.csvWriter.Flush()
		f.File.Close()
	}
}

func (dd *DirDispatcher) genFilename(name string) string {
	return fmt.Sprintf("%s.csv", name)
}

func (dd *DirDispatcher) process(msg *proto.CSVMsg) error {
	w, err := dd.getCSVWriter(msg.GroupID)
	if err != nil {
		return err
	}
	return w.Write(msg.Data)
}
