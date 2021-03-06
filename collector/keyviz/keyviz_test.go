package keyviz

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"testing"
	"time"
)

func TestKeyVizCollect_Login(t *testing.T) {
	code := os.Getenv("CODE")
	host := os.Getenv("HOST")
	if len(code) == 0 || len(host) == 0 {
		t.Skip("set CODE and HOST to run the test")
	}
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
	kc, err := NewKeyVizCollect(
		WithHttpClient(cli),
	)
	if err != nil {
		t.Fatal(err)
	}
	endpoint, err := url.Parse(host)
	if err != nil {
		t.Fatal(err)
	}
	kc.SetSessionCode(code)
	token, err := kc.Login(context.Background(), endpoint)
	if err != nil {
		t.Fatal(err)
	}
	if len(token) == 0 {
		t.Fatal("token should not empty")
	}
}
