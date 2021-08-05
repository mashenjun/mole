package main

import (
	"context"
	"fmt"
	"github.com/mashenjun/mole/convertor"
	"github.com/mashenjun/mole/dispatch"
	"github.com/pingcap/errors"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

func convertCmd() *cobra.Command {
	var (
		input = ""
		output = ""
		begin             = ""
		end               = ""
	)
	cmd := &cobra.Command{
		Use:   `convert`,
		Short: `convert to csv file`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if len(input) == 0 || len(output) == 0 {
				fmt.Println("input or output is not provided")
				_ = cmd.Help()
				return errors.New("input or output is not provided")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := convertor.NewMetricsMatrixConvertor(
				convertor.WithTimeRange(begin, end),
				)
			if err != nil {
				fmt.Printf("new MetricsMatrixConvertor error: %v\n", err)
				return err
			}
			source := c.GetSink()
			d, err := dispatch.NewCSVDispatcher(output, source)
			if err != nil {
				fmt.Printf("new CSVDispatcher error: %v\n", err)
				return err
			}

			errG, ctx := errgroup.WithContext(context.Background())
			errG.Go(func() error {
				return c.Convert(input)
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
	cmd.Flags().StringVarP(&begin, "from", "f", "", "start time point filter timeseries data")
	cmd.Flags().StringVarP(&end, "to", "t", "", "stop time point filter timeseries data")
	return cmd
}

