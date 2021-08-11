package main

import (
	"errors"
	"fmt"
	"github.com/mashenjun/mole/collector/prom"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"runtime"
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
		target    = ""
		continues = false
	)

	cmd := &cobra.Command{
		Use:   `metrics`,
		Short: `collect metrics from target prometheus`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if len(output) == 0 {
				_ = cmd.Help()
				return errors.New("missing output flag")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: init the related component and start the process
			topo := make([]prom.Endpoint, 0, len(hosts))
			for _, h := range hosts {
				u, err := url.Parse(h)
				if err != nil {
					fmt.Printf("hosts is invalid: %v\n", err)
					return err
				}
				topo = append(topo, prom.Endpoint{
					Schema: u.Scheme,
					Host: u.Hostname(),
					Port: u.Port(),
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
			ml := MetricsList{Raw: make([]string, 0), Cooked: make([]prom.MetricsRecord, 0)}
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
			mc, err := prom.NewMetricsCollect(
				prom.WithHttpCli(cli),
				prom.WithTimeRange(begin, end),
				prom.WithConcurrency(int(concurrency)),
				prom.WithMerge(merge),
				prom.WithOutputDir(output),
				prom.WithContinues(continues),
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
			return nil
		},
	}
	cmd.Flags().Int64VarP(&concurrency, "concurrency", "c", int64(runtime.NumCPU()), "concurrency setting")
	cmd.Flags().StringVarP(&begin, "from", "f", time.Now().Add(time.Hour*-2).Format(time.RFC3339), "start time point when collecting timeseries data")
	cmd.Flags().StringVarP(&end, "to", "t", time.Now().Format(time.RFC3339), "stop time point when collecting timeseries data")
	cmd.Flags().StringVarP(&output, "output", "o", "", "output directory of collected data")
	cmd.Flags().BoolVarP(&merge, "merge", "m", true, "merge content of different range for one metrics into one file")
	cmd.Flags().StringSliceVarP(&hosts, "hosts", "H", nil, "hosts list with schema://ip:port format")
	cmd.Flags().StringVarP(&target, "target", "T", "", "path to yaml file containing target metrics")
	cmd.Flags().BoolVarP(&continues, "continues", "C", false, "set the collect to skip the existed metrics")
	return cmd
}

type MetricsList struct {
	Raw    []string             `yaml:"raw"`
	Cooked []prom.MetricsRecord `yaml:"cooked"`
}
