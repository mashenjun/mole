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

type metricsCmdOption struct {
	begin       string
	end         string
	concurrency int64
	output      string
	merge       bool
	hosts       []string
	target      string
	continues   bool
	subDir      bool
	clusterID   string
	style       string
	accountID   int
}

func (opt *metricsCmdOption) validate() error {
	if len(opt.output) == 0 {
		return errors.New("output is not set")
	}
	if opt.style == "vm-select" && opt.accountID == 0 {
		return errors.New("account-id is not set with vm-select style")
	}
	return nil
}

func metricsCmd() *cobra.Command {
	// TODO@shenjun: move all the field to an option struct and check validation.
	//var (
	//	begin             = ""
	//	end               = ""
	//	concurrency int64 = 0
	//	output            = ""
	//	merge             = true
	//	hosts             = make([]string, 0)
	//	target            = ""
	//	continues         = false
	//	subDir            = true
	//	clusterID         = ""
	//	style             = "prometheus"
	//	accountID         = 0
	//)

	var opt = metricsCmdOption{
		merge:  true,
		hosts:  make([]string, 0),
		subDir: true,
		style:  "prometheus",
	}

	cmd := &cobra.Command{
		Use:   `metrics`,
		Short: `collect metrics from target prometheus`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if err := opt.validate(); err != nil {
				_ = cmd.Help()
				return err
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// init endpoint list
			topo := make([]prom.Endpoint, 0, len(opt.hosts))
			for _, h := range opt.hosts {
				u, err := url.Parse(h)
				if err != nil {
					fmt.Printf("hosts is invalid: %v\n", err)
					return err
				}
				ep := prom.Endpoint{
					Schema: u.Scheme,
					Host:   u.Hostname(),
					Port:   u.Port(),
				}
				switch opt.style {
				case "vm-select":
					ep.Type = prom.EndpointVMSelect
					ep.AccountID = opt.accountID
				default:
					// do nothing
				}
				topo = append(topo, ep)
			}

			// init http client
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

			// init metrics list
			ml := MetricsList{Raw: make([]string, 0), Cooked: make([]prom.MetricsRecord, 0)}
			if len(opt.target) != 0 {
				b, err := ioutil.ReadFile(opt.target)
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
				prom.WithTimeRange(opt.begin, opt.end),
				prom.WithConcurrency(int(opt.concurrency)),
				prom.WithMerge(opt.merge),
				prom.WithOutputDir(opt.output),
				prom.WithContinues(opt.continues),
				prom.WithSubDirEnable(opt.subDir),
			)
			if err != nil {
				fmt.Printf("new metrics collect error: %+v\n", err)
				return err
			}
			// set target metrics if user give target metrics file
			mc.SetRawMetrics(ml.Raw)
			mc.SetCookedRecord(ml.Cooked)
			if len(opt.clusterID) > 0 {
				mc.SetClusterID(opt.clusterID)
			}
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
	cmd.Flags().Int64VarP(&opt.concurrency, "concurrency", "c", int64(runtime.NumCPU()), "concurrency setting")
	cmd.Flags().StringVarP(&opt.begin, "from", "f", time.Now().Add(time.Hour*-2).Format(time.RFC3339), "start time point when collecting timeseries data")
	cmd.Flags().StringVarP(&opt.end, "to", "t", time.Now().Format(time.RFC3339), "stop time point when collecting time-series data")
	cmd.Flags().StringVarP(&opt.output, "output", "o", "", "output directory of collected data")
	cmd.Flags().BoolVarP(&opt.merge, "merge", "m", true, "merge content of different range for one metrics into one file")
	cmd.Flags().StringSliceVarP(&opt.hosts, "hosts", "H", nil, "hosts list with schema://ip:port format")
	cmd.Flags().StringVarP(&opt.target, "target", "T", "", "path to yaml file containing target metrics")
	cmd.Flags().BoolVarP(&opt.continues, "continues", "C", false, "set the collector to skip the existed metrics")
	cmd.Flags().BoolVarP(&opt.subDir, "subdir", "", true, "set the collector to store data in sub directories")
	cmd.Flags().StringVarP(&opt.clusterID, "cluster-id", "", "", "set cluster id")
	cmd.Flags().StringVarP(&opt.style, "style", "s", "prometheus", "set endpoint type, support value is prometheus, vm-select")
	cmd.Flags().IntVarP(&opt.accountID, "account-id", "", 1, "set account if for vm select endpoint")
	return cmd
}

type MetricsList struct {
	Raw    []string             `yaml:"raw"`
	Cooked []prom.MetricsRecord `yaml:"cooked"`
}
