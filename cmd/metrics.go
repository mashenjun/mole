package main

import (
	"errors"
	"fmt"
	"github.com/mashenjun/mole/collector"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"net"
	"net/http"
	"runtime"
	"strings"
	"time"
)

func metricsCmd() *cobra.Command {
	var (
		begin             = ""
		end               = ""
		concurrency int64 = 0
		output            = ""
		merge             = true
		hosts             = make([]string, 0)
		target            = ""
	)

	cmd := &cobra.Command{
		Use:   `metrics`,
		Short: `collect metrics from target prometheus`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if len(output) == 0 {
				_ = cmd.Help()
				return errors.New("miss output flag")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("run merge %+v\n", merge)
			// TODO: init the related component and start the process
			topo := make([]collector.Endpoint, 0, len(hosts))
			for _, h := range hosts {
				s := strings.Split(h, ":")
				if len(s) != 2 {
					fmt.Println("hosts invalid")
					return errors.New("hosts invalid")
				}
				topo = append(topo, collector.Endpoint{
					Host: s[0],
					Port: s[1],
				})
			}

			cli := &http.Client{
				Transport: &http.Transport{
					Proxy: http.ProxyFromEnvironment,
					DialContext: (&net.Dialer{
						Timeout:   30 * time.Second,
						KeepAlive: 30 * time.Second,
					}).DialContext,
					ForceAttemptHTTP2:     true,
					MaxIdleConns:          100,
					IdleConnTimeout:       90 * time.Second,
					TLSHandshakeTimeout:   10 * time.Second,
					ExpectContinueTimeout: 1 * time.Second,
					MaxIdleConnsPerHost:   runtime.NumCPU() << 1,
				},
				Timeout: 60 * time.Second,
			}
			ml := MetricsList{Raw: make([]string, 0), Cooked: make([]collector.MetricsRecord, 0)}
			if len(target) != 0 {
				b, err := ioutil.ReadFile(target)
				if err != nil {
					fmt.Printf("read file error: %+v", err)
					return err
				}
				if err := yaml.Unmarshal(b, &ml); err != nil {
					fmt.Printf("unmarshal yaml file error: %+v", err)
					return err
				}
			}

			mc, err := collector.NewMetricsCollect(
				collector.WithHttpCli(cli),
				collector.WithTimeRange(begin, end),
				collector.WithConcurrency(int(concurrency)),
				collector.WithMerge(merge),
				collector.WithOutputDir(output),
			)
			if err != nil {
				fmt.Printf("new metrics collect error: %+v\n", err)
				return err
			}
			// set target metrics if user give target metrics file
			mc.SetRawMetrics(ml.Raw)
			mc.SetCookedRecord(ml.Cooked)
			// use Collect method
			if _, err := mc.Prepare(topo); err != nil {
				fmt.Printf("prepare metrics error: %+v\n", err)
				return err
			}
			if err := mc.Collect(topo); err != nil {
				fmt.Printf("collect metrics error: %+v\n", err)
				return err
			}
			// use Drain method
			//errG, ctx := errgroup.WithContext(context.Background())
			//errG.Go(func() error {
			//	if _, err := mc.Prepare(topo); err != nil {
			//		return err
			//	}
			//	return nil
			//})
			//errG.Go(func() error {
			//	return mc.Drain(ctx)
			//})
			//if err := errG.Wait(); err != nil {
			//	fmt.Printf("collect metrics error: %+v\n", err)
			//	return err
			//}

			// use Dispatch method

			//d, err := dispatch.NewMetricDispatcher(output, mc.GetSink(), merge)
			//if err != nil {
			//	fmt.Printf("new metrics dispatcher error: %+v\n",err)
			//	return err
			//}
			//
			//if _, err := mc.Prepare(topo); err != nil {
			//	fmt.Printf("prepare metrics error: %+v\n",err)
			//	return err
			//}
			//errG, ctx := errgroup.WithContext(context.Background())
			//errG.Go(func() error {
			//	if err := mc.Sink(topo); err != nil {
			//		return err
			//	}
			//	return nil
			//})
			//errG.Go(func() error {
			//	return d.Start(ctx)
			//})
			//if err := errG.Wait(); err != nil {
			//	fmt.Printf("collect metrics error: %+v\n", err)
			//	return err
			//}
			return nil
		},
	}
	cmd.Flags().Int64VarP(&concurrency, "concurrency", "c", int64(runtime.NumCPU()), "concurrency setting")
	cmd.Flags().StringVarP(&begin, "from", "f", time.Now().Add(time.Hour*-2).Format(time.RFC3339), "start time point when collecting timeseries data")
	cmd.Flags().StringVarP(&end, "to", "t", time.Now().Format(time.RFC3339), "stop time point when collecting timeseries data")
	cmd.Flags().StringVarP(&output, "output", "o", "", "output directory of collected data")
	cmd.Flags().BoolVarP(&merge, "merge", "m", true, "merge content of different range for one metrics into one file")
	cmd.Flags().StringSliceVarP(&hosts, "hosts", "H", nil, "hosts list")
	cmd.Flags().StringVarP(&target, "target", "T", "", "path to yaml file containing target metrics")
	return cmd
}

type MetricsList struct {
	Raw    []string       `yaml:"raw"`
	Cooked []collector.MetricsRecord `yaml:"cooked"`
}
