package main

import (
	"context"
	"fmt"
	"github.com/mashenjun/mole/convertor/keyviz"
	"github.com/mashenjun/mole/dispatch"
	"github.com/pingcap/errors"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	"os"
	"path/filepath"
)

func splitCmd() *cobra.Command {
	var (
		inputDir  = ""
		outputDir = ""
		begin  = ""
		end    = ""
		filters = make([]string, 0)
	)
	cmd := &cobra.Command{
		Use:   `split`,
		Short: `split heatmap data to multiple csv file`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if len(inputDir) == 0 || len(outputDir) == 0 {
				fmt.Println("input or output is not provided")
				_ = cmd.Help()
				return errors.New("input or output is not provided")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {

			ds, err := os.ReadDir(inputDir)
			if err != nil {
				return err
			}
			errG, ctx := errgroup.WithContext(context.Background())

			for  _, d := range ds {
				if d.IsDir() {
					continue
				}
				input := filepath.Join(inputDir, d.Name())
				hc, err := keyviz.NewHeatmapConvertor(
					keyviz.WithTimeRange(begin, end),
					keyviz.WithInput(input),
					keyviz.WithSplit(),
				)
				if err != nil {
					fmt.Printf("new NewHeatmapConvertor error: %v\n", err)
					return err
				}
				if len(filters) > 0 {
					hc.SetFilterRules(filters)
				}
				source := hc.GetSink()
				d, err := dispatch.NewDirDispatcher(outputDir, source)
				if err != nil {
					fmt.Printf("new DirDispatcher error: %v\n", err)
					return err
				}
				//errG, ctx := errgroup.WithContext(context.Background())
				errG.Go(func() error {
					return hc.Convert()
				})
				errG.Go(func() error {
					return d.Start(ctx)
				})
			}
			if err := errG.Wait(); err != nil {
				fmt.Printf("split heatmap to csv error: %+v\n", err)
				return err
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&inputDir, "input", "i", "", "the input dir where store the heatmap data")
	cmd.Flags().StringVarP(&outputDir, "output", "o", "", "the output dir where store the csv data")
	cmd.Flags().StringVarP(&begin, "from", "f", "", "start time point to filter timeseries data")
	cmd.Flags().StringVarP(&end, "to", "t", "", "end time point to filter timeseries data")
	cmd.Flags().StringSliceVarP(&filters, "filters", "", nil, "list of filter rule with db:table format, only used for heatmap data")
	return cmd
}
