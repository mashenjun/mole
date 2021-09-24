package prom

import (
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

func TestMetricCollect_Collect(t *testing.T) {
	endpoint := os.Getenv("ENDPOINT")
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
			MaxIdleConnsPerHost:   16,
		},
		Timeout: 60 * time.Second,
	}
	s := strings.Split(endpoint, ":")
	topo := []Endpoint{
		{
			Schema: "http",
			Host: s[0],
			Port: s[1],
		},
	}

	mc, err := NewMetricsCollect(
		WithHttpCli(cli),
		WithTimeRange("",""),
		)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := mc.Prepare(topo); err != nil {
		t.Fatal(err)
	}
}

func Test_ParseTimeRange(t *testing.T) {
	begin := "2021-08-02T17:11:50+08:00"
	end := "2021-08-03T17:11:50+08:00"
	ts,_, err := parseTimeRange(begin, end)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range ts {
		t.Log(s)
	}
}

func TestMetricsCollect_GetInstanceCnt(t *testing.T) {
	endpoint := os.Getenv("ENDPOINT")
	s := strings.Split(endpoint, ":")
	topo := []Endpoint{
		{
			Schema: "http",
			Host: s[0],
			Port: s[1],
		},
	}

	mc, err := NewMetricsCollect(
		WithHttpCli(http.DefaultClient),
	)
	if err != nil {
		t.Fatal(err)
	}
	cnt, err := mc.getInstanceCnt(topo[0], "tikv");
	if err != nil {
		t.Fatal(err)
	}
	if cnt == 0 {
		t.Fatal("wrong result")
	}
}
