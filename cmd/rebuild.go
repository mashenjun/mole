package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/mashenjun/mole/dispatch"
	"github.com/mashenjun/mole/rebuild"
	"github.com/mashenjun/mole/utils"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	"os"
)

func rebuildCmd() *cobra.Command {
	var (
		inputDir  = ""
		outputDir = ""
		endpoint  = ""
	)
	cmd := &cobra.Command{
		Use:   `rebuild`,
		Short: `rebuild tsdb block form metrics data`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if len(inputDir) == 0 {
				fmt.Println("input is not provided")
				_ = cmd.Help()
				return errors.New("input is not provided")
			}
			if len(outputDir) == 0 && len(endpoint) == 0 {
				fmt.Println("output or endpoint is not provided")
				_ = cmd.Help()
				return errors.New("output or endpoint is not provided")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(outputDir) != 0 {
				if err := utils.EnsureDir(outputDir); err != nil {
					fmt.Printf("ensure dir error: %+v\n", err)
					return err
				}
				df, err := os.ReadDir(outputDir)
				if err != nil {
					fmt.Printf("read dir err: %+v\n", err)
					return err
				}
				if len(df) > 0 {
					fmt.Printf("%+v is not empty", outputDir)
					return fmt.Errorf("%+v is not empty", outputDir)
				}
			}
			// use error group to fan out task
			errG, ctx := errgroup.WithContext(context.Background())
			hc, err := rebuild.NewMetricsMatrixDrainer(
				rebuild.WithInput(inputDir),
			)
			if err != nil {
				fmt.Printf("new MetricsMatrixDrainer error: %v\n", err)
				return err
			}
			source := hc.GetSink2()
			d, err := dispatch.NewVictoriaMetricsDispatcher(endpoint, source)
			if err != nil {
				fmt.Printf("new NewVictoriaMetricsDispatcher error: %v\n", err)
				return err
			}
			//d, err := dispatch.NewTSDBBlockDispatcher(outputDir, source)
			//if err != nil {
			//	fmt.Printf("new TSDBBlockDispatcher error: %v\n", err)
			//	return err
			//}
			errG.Go(func() error {
				if err := hc.Start2(ctx); err != nil {
					fmt.Printf("metrics matrix drainer start err: %+v\n", err)
					return err
				}
				fmt.Println("metrics matrix drainer done")
				return nil
			})
			errG.Go(func() error {
				if err := d.Start(ctx); err != nil {
					fmt.Printf("dispatch start err: %+v\n", err)
					return err
				}
				fmt.Println("dispatch done")
				return nil
			})
			if err := errG.Wait(); err != nil {
				fmt.Printf("rebuild tsdb storage error: %+v\n", err)
				return err
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&inputDir, "input", "i", "", "the input dir where store the metrics data")
	cmd.Flags().StringVarP(&outputDir, "output", "o", "", "the output dir to store tsdb block")
	cmd.Flags().StringVarP(&endpoint, "endpoint", "e", "", "the endpoint to store victoria metrics")
	return cmd
}
