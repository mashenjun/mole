package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/mashenjun/mole/collector/keyviz"
	"github.com/spf13/cobra"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"runtime"
	"strings"
	"time"
)

func keyvizCmd() *cobra.Command {
	var (
		begin  = ""
		end    = ""
		user   = ""
		pwd    = ""
		session = ""
		host   = ""
		output = ""
	)
	cmd := &cobra.Command{
		Use:   `keyviz`,
		Short: `collect heatmap from dashboard api`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if len(user) + len(pwd) + len(session) == 0{
				fmt.Println("login information is not provided")
				_ = cmd.Help()
				return errors.New("login information is not provided")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			cli := &http.Client{
				Transport: &http.Transport{
					Proxy: http.ProxyFromEnvironment,
					DialContext: (&net.Dialer{
						Timeout:   30 * time.Second,
						KeepAlive: 30 * time.Second,
					}).DialContext,
					ForceAttemptHTTP2:     true,
					MaxIdleConns:          10,
					IdleConnTimeout:       90 * time.Second,
					TLSHandshakeTimeout:   5 * time.Second,
					ExpectContinueTimeout: 1 * time.Second,
					MaxIdleConnsPerHost:   runtime.NumCPU() << 1,
				},
				Timeout: 10 * time.Second,
			}
			kc, err := keyviz.NewKeyVizCollect(
				keyviz.WithHttpClient(cli),
				keyviz.WithTimeRange(begin, end),
				keyviz.WithOutput(output),
			)
			if err != nil {
				fmt.Printf("new keyviz error: %+v\n", err)
				return err
			}
			if len(session) > 0 {
				bs, err := ioutil.ReadFile(session)
				if err != nil {
					fmt.Printf("read file %v error: %+v\n", session, err)
					return err
				}
				// in case the file contain newline
				kc.SetSessionCode(strings.TrimRight(string(bs), "\n"))
			}else {
				kc.SetUserPwd(user, pwd)
			}
			endpoint, err := url.Parse(host)
			if err != nil {
				fmt.Printf("host is invaild: %v\n", err)
				return err
			}
			ctx := context.Background()
			token, err := kc.Login(ctx, endpoint)
			if err != nil {
				fmt.Printf("login to dashboard api server %v error: %v\n", host, err)
				return err
			}
			if err := kc.Collect(ctx, token, endpoint); err != nil {
				fmt.Printf("collect heamap error: %v\n", err)
				return err
			}
			return nil
		},
	}

	// fill the mysql config form the flag
	cmd.Flags().StringVarP(&user, "user", "u", "", "the dsn used to connect db")
	cmd.Flags().StringVarP(&pwd, "pwd", "p", "", "the dsn used to connect db")
	cmd.Flags().StringVarP(&host, "host", "H", "", "host for dashboard api with schema://ip:port format")
	cmd.Flags().StringVarP(&output, "output", "o", "", "the output file used to store heatmap data")
	cmd.Flags().StringVarP(&begin, "from", "f", time.Now().Add(time.Hour*-2).Format(time.RFC3339), "start time point when collecting timeseries data")
	cmd.Flags().StringVarP(&end, "to", "t", time.Now().Format(time.RFC3339), "stop time point when collecting timeseries data")
	cmd.Flags().StringVar(&session, "session", "", "file containing session code")

	return cmd
}
