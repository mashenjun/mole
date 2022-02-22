package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/mashenjun/mole/convertor"
	"github.com/mashenjun/mole/convertor/keyviz"
	"github.com/mashenjun/mole/convertor/prom"
	"github.com/mashenjun/mole/dispatch"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

func convertCmd() *cobra.Command {
	var (
		input   = ""
		output  = ""
		begin   = ""
		end     = ""
		format  = ""
		filters = make([]string, 0)
	)
	cmd := &cobra.Command{
		Use:   `convert`,
		Short: `convert to csv file`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if len(input) == 0 || len(output) == 0 || len(format) == 0 {
				fmt.Println("input or output or format is not provided")
				_ = cmd.Help()
				return errors.New("input or output or format is not provided")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			var c convertor.IConvert
			switch format {
			case "prom":
				mcc, err := prom.NewMetricsMatrixConvertor(
					prom.WithTimeRange(begin, end),
					prom.WithInput(input),
				)
				if err != nil {
					fmt.Printf("new MetricsMatrixConvertor error: %v\n", err)
					return err
				}
				c = mcc
			case "heatmap":
				hc, err := keyviz.NewHeatmapConvertor(
					keyviz.WithTimeRange(begin, end),
					keyviz.WithInput(input),
				)
				if err != nil {
					fmt.Printf("new NewHeatmapConvertor error: %v\n", err)
					return err
				}
				if len(filters) > 0 {
					hc.SetFilterRules(filters)
				}
				c = hc
			default:
				fmt.Printf("format %v is not supported\n", format)
				return fmt.Errorf("format %v is not supported", format)
			}
			source := c.GetSink()
			d, err := dispatch.NewCSVDispatcher(output, source)
			if err != nil {
				fmt.Printf("new CSVDispatcher error: %v\n", err)
				return err
			}

			errG, ctx := errgroup.WithContext(context.Background())
			errG.Go(func() error {
				return c.Convert(ctx)
			})
			errG.Go(func() error {
				return d.Start(ctx)
			})
			if err := errG.Wait(); err != nil {
				fmt.Printf("convert metrics to csv error: %+v\n", err)
				return err
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&input, "input", "i", "", "the input file used to store metrics data")
	cmd.Flags().StringVarP(&output, "output", "o", "", "the output file used to store csv data")
	cmd.Flags().StringVarP(&begin, "from", "f", "", "start time point to filter timeseries data")
	cmd.Flags().StringVarP(&end, "to", "t", "", "end time point to filter timeseries data")
	cmd.Flags().StringVarP(&format, "format", "", "", "source data format, possible value is [prom|heatmap]")
	cmd.Flags().StringSliceVarP(&filters, "filters", "", nil, "list of filter rule with db:table format, only used for heatmap data")
	return cmd
}
