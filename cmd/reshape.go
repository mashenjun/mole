package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/mashenjun/mole/dispatch"
	"github.com/mashenjun/mole/utils"
	"github.com/prometheus/common/model"
	"golang.org/x/sync/errgroup"
	"gopkg.in/yaml.v3"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/mashenjun/mole/convertor/prom"
	"github.com/spf13/cobra"
)

func reshapeCmd() *cobra.Command {
	var (
		inputDir  = ""
		outputDir = ""
		begin     = ""
		end       = ""
		ruleFile  = ""
	)
	cmd := &cobra.Command{
		Use:   `reshape`,
		Short: `reshape multiple metrics to csv files`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if len(inputDir) == 0 || len(outputDir) == 0 || len(ruleFile) == 0 {
				fmt.Println("input or output or cookbook is not provided")
				_ = cmd.Help()
				return errors.New("input or output or cookbook is not provided")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			var fr FilterRules
			bs, err := ioutil.ReadFile(ruleFile)
			if err != nil {
				fmt.Printf("read file %v error: %+v\n", ruleFile, err)
				return err
			}
			if err := yaml.Unmarshal(bs, &fr); err != nil {
				fmt.Printf("unmarshal yaml error: %+v\n", err)
				return err
			}
			if err := utils.EnsureDir(inputDir); err != nil {
				fmt.Printf("ensure dir error: %+v\n", err)
				return err
			}
			if err := utils.EnsureDir(outputDir); err != nil {
				fmt.Printf("ensure dir error: %+v\n", err)
				return err
			}
			// use error group to fan out task
			//errG, ctx := errgroup.WithContext(context.Background())
			if _, err := os.Stat(filepath.Join(inputDir, "meta.yaml")); err == nil {
				if err := CopyFile(filepath.Join(inputDir, "meta.yaml"), filepath.Join(outputDir, "meta.yaml")); err != nil {
					fmt.Printf("copy meta.yaml error: %+v", err)
					return err
				}
			}
			
			for _, rule := range fr.Rules {
				fmt.Printf("start reshape %v ...\n", rule.Record)
				inputFile := filepath.Join(inputDir, fmt.Sprintf("%s.json", rule.Record))
				mcc, err := prom.NewMetricsMatrixConvertor(
					prom.WithTimeRange(begin, end),
					prom.WithInput(inputFile),
				)
				if err != nil {
					fmt.Printf("new MetricsMatrixConvertor error: %+v\n", err)
					return err
				}
				mcc.SetFilterLabels(rule.Filter)
				if len(rule.ApplyAgg) > 0 {
					mcc.SetAggregation(rule.ApplyAgg)
				}
				source := mcc.GetSink()
				outputFile := filepath.Join(outputDir, fmt.Sprintf("%s.csv", rule.OutputName()))
				d, err := dispatch.NewCSVDispatcher(outputFile, source)
				if err != nil {
					fmt.Printf("new CSVDispatcher error: %v\n", err)
					return err
				}
				errG, ctx := errgroup.WithContext(context.Background())

				errG.Go(func() error {
					return mcc.Convert()
				})
				errG.Go(func() error {
					return d.Start(ctx)
				})
				if err := errG.Wait(); err != nil {
					fmt.Printf("reshape %v error: %v\n", rule.Record, err)
					return err
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&inputDir, "input", "i", "", "the input dir where store the metrics data")
	cmd.Flags().StringVarP(&outputDir, "output", "o", "", "the output dir to store csv data")
	cmd.Flags().StringVarP(&begin, "from", "f", "", "start time point to filter timeseries data, in RFC3339 format")
	cmd.Flags().StringVarP(&end, "to", "t", "", "end time point to filter timeseries data, in RFC3339 format")
	cmd.Flags().StringVarP(&ruleFile, "rule", "", "", "path to the yaml file define rules")
	return cmd
}

type FilterRule struct {
	ReplaceName string           `json:"replace_name" yaml:"replace_name"`
	Record      string           `json:"record" yaml:"record"` // source metrics name
	ApplyAgg    string           `json:"apply_agg" yaml:"apply_agg"`
	Filter      []model.LabelSet `json:"filter" yaml:"filter"` // label filter
}

func (rf *FilterRule) OutputName() string {
	if len(rf.ReplaceName) == 0 {
		return rf.Record
	}
	return rf.ReplaceName
}

type FilterRules struct {
	Rules []FilterRule `json:"rules" yaml:"rules"`
}

func CopyFile(from string, to string) error {
	fin, err := os.Open(from)
	if err != nil {
		return err
	}
	defer fin.Close()

	fout, err := os.Create(to)
	if err != nil {
		return err
	}
	defer fout.Close()

	if _, err = io.Copy(fout, fin); err != nil {
		return err
	}
	return nil
}
