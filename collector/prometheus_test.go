package collector

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
	//sink := mc.GetSink()
	//go func() {
	//	if err := mc.Collect(topo); err != nil {
	//		panic(err)
	//	}
	//}()
	//for msg := range sink {
	//	if b, err := ioutil.ReadAll(msg.Handler); err != nil {
	//		t.Log(err)
	//	} else {
	//		t.Logf("%s, %v", msg.Name, len(b))
	//	}
	//}
}
