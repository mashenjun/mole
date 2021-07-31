package utils

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// default timeout 1000ms
var DefaultHttpClient = New(TimeoutOption(1000 * time.Millisecond))

func TimeoutOption(timeout time.Duration) func(client *HttpClient) {
	return func(c *HttpClient) {
		c.Client.Timeout = timeout
	}
}

func MaxIdleConnOption(connCount int) func(client *HttpClient) {
	return func(c *HttpClient) {
		// todo: is there better way to update transport
		// get the old transport and replace by new one
		oldTransport := c.Transport
		oldTransportPointer, ok := oldTransport.(*http.Transport)
		if !ok {
			panic("transport not an *http.Transport")
		}
		// create new transport
		newTransport := &http.Transport{
			Proxy:                 oldTransportPointer.Proxy,
			DialContext:           oldTransportPointer.DialContext,
			MaxIdleConns:          connCount,
			MaxIdleConnsPerHost:   connCount,
			IdleConnTimeout:       oldTransportPointer.IdleConnTimeout,
			TLSHandshakeTimeout:   oldTransportPointer.TLSHandshakeTimeout,
			ExpectContinueTimeout: oldTransportPointer.ExpectContinueTimeout,
			// todo: set tls config
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		c.Transport = newTransport
	}
}

func CliOption(cli *http.Client) func(client *HttpClient) {
	return func(c *HttpClient) {
		c.Client = cli
	}
}


type HttpClient struct {
	*http.Client
}

func New(opts ...func(client *HttpClient)) *HttpClient {
	cli := HttpClient{Client: &http.Client{Transport: http.DefaultTransport}}
	for _, opt := range opts {
		opt(&cli)
	}
	return &cli
}

func (r *HttpClient) DoRequestWithForm(
	ctx context.Context, method, uri string, data map[string][]string) (resp *http.Response, err error) {

	msg := url.Values(data).Encode()
	if method == "GET" || method == "HEAD" || method == "DELETE" {
		if strings.ContainsRune(uri, '?') {
			uri += "&"
		} else {
			uri += "?"
		}
		return r.DoRequest(ctx, method, uri+msg)
	}
	return r.DoRequestWith(
		ctx, method, uri, "application/x-www-form-urlencoded", strings.NewReader(msg), int64(len(msg)))
}

func (r *HttpClient) DoRequestWithFormHeader(
	ctx context.Context, method, uri string, data map[string][]string, h map[string]string) (resp *http.Response, err error) {

	msg := url.Values(data).Encode()
	if method == "GET" || method == "HEAD" || method == "DELETE" {
		if strings.ContainsRune(uri, '?') {
			uri += "&"
		} else {
			uri += "?"
		}
		return r.DoRequestWithHeader(ctx, method, uri+msg, "application/x-www-form-urlencoded", nil, 0, h)
	}
	return r.DoRequestWithHeader(
		ctx, method, uri, "application/x-www-form-urlencoded", strings.NewReader(msg), int64(len(msg)), h)
}

func (r *HttpClient) DoRequestWithJson(
	ctx context.Context, method, uri string, data interface{}) (resp *http.Response, err error) {

	msg, err := json.Marshal(data)
	if err != nil {
		return
	}
	return r.DoRequestWith(
		ctx, method, uri, "application/json", bytes.NewReader(msg), int64(len(msg)))
}

func (r *HttpClient) DoRequestWithJsonHeader(
	ctx context.Context, method, uri string, data interface{}, h map[string]string) (resp *http.Response, err error) {

	msg, err := json.Marshal(data)
	if err != nil {
		return
	}
	return r.DoRequestWithHeader(
		ctx, method, uri, "application/json", bytes.NewReader(msg), int64(len(msg)), h)
}

func (r *HttpClient) DoRequest(ctx context.Context, method, uri string) (resp *http.Response, err error) {

	req, err := http.NewRequest(method, uri, nil)
	if err != nil {
		return
	}
	return r.Do(ctx, req)
}

func (r *HttpClient) DoRequestWithHeader(
	ctx context.Context, method, uri string,
	bodyType string, body io.Reader, bodyLength int64, m map[string]string) (resp *http.Response, err error) {

	req, err := http.NewRequest(method, uri, body)
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", bodyType)
	for key, value := range m {
		req.Header.Set(key, value)
	}
	req.ContentLength = bodyLength
	return r.Do(ctx, req)
}

func (r *HttpClient) DoRequestWith(
	ctx context.Context, method, uri string,
	bodyType string, body io.Reader, bodyLength int64) (resp *http.Response, err error) {

	req, err := http.NewRequest(method, uri, body)
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", bodyType)
	req.ContentLength = bodyLength
	return r.Do(ctx, req)
}

func (r *HttpClient) Do(ctx context.Context, req *http.Request) (resp *http.Response, err error) {
	if ctx == nil {
		ctx = context.Background()
	}
	return r.Client.Do(req.WithContext(ctx))
}

func (r *HttpClient) CallWithForm(
	ctx context.Context, ret interface{}, method, url1 string, param map[string][]string) (err error) {

	resp, err := r.DoRequestWithForm(ctx, method, url1, param)
	if err != nil {
		return err
	}
	return r.callRet(ctx, ret, resp)
}

func (r *HttpClient) CallWithFormHeader(
	ctx context.Context, ret interface{}, method, url1 string, param map[string][]string, h map[string]string) (err error) {

	resp, err := r.DoRequestWithFormHeader(ctx, method, url1, param, h)
	if err != nil {
		return err
	}
	return r.callRet(ctx, ret, resp)
}

func (r *HttpClient) CallWithJson(
	ctx context.Context, ret interface{}, method, url1 string, param interface{}) (err error) {

	resp, err := r.DoRequestWithJson(ctx, method, url1, param)
	if err != nil {
		return err
	}
	return r.callRet(ctx, ret, resp)
}

func (r *HttpClient) CallWithJsonHeader(
	ctx context.Context, ret interface{}, method, url1 string, param interface{}, h map[string]string) (err error) {

	resp, err := r.DoRequestWithJsonHeader(ctx, method, url1, param, h)
	if err != nil {
		return err
	}
	return r.callRet(ctx, ret, resp)
}

type RespError interface {
	Error() string
	HttpCode() int
	DecodeError(errInfo interface{}) error
}

type errorInfo struct {
	Err     string   `json:"error"`
	Code    int      `json:"code"`
}


func (r *errorInfo) Error() string {
	if r.Err != "" {
		return r.Err
	}
	return http.StatusText(r.Code)
}

func (r *errorInfo) HttpCode() int {
	return r.Code
}

func (r *errorInfo) DecodeError(errInfo interface{}) error {
	return json.Unmarshal([]byte(r.Err), errInfo)
}

func parseError(r io.Reader) (err string) {
	body, err1 := ioutil.ReadAll(r)
	if err1 != nil {
		return err1.Error()
	}
	return string(body)
}

func responseError(resp *http.Response) (err error) {
	e := &errorInfo{
		Code:    resp.StatusCode,
	}
	if resp.StatusCode > 299 {
		if resp.ContentLength != 0 {
			if ct := resp.Header.Get("Content-Type"); strings.TrimSpace(strings.SplitN(ct, ";", 2)[0]) == "application/json" {
				e.Err = parseError(resp.Body)
			}
		}
	}
	return e
}

func (r *HttpClient) callRet(ctx context.Context, ret interface{}, resp *http.Response) (err error) {
	defer func() {
		_, _ = io.Copy(ioutil.Discard, resp.Body)
		_ = resp.Body.Close() // must close http body
	}()

	if resp.StatusCode/100 == 2 {
		if ret != nil && resp.ContentLength != 0 {
			err = json.NewDecoder(resp.Body).Decode(ret)
			if err != nil {
				return
			}
		}
		if resp.StatusCode == 200 {
			return nil
		}
	}
	return responseError(resp)
}

