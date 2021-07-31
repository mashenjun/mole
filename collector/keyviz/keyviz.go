package keyviz

import (
	"context"
	"errors"
	"fmt"
	"github.com/gorilla/schema"
	"github.com/mashenjun/mole/utils"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
)

const (
	loginAPIPath   = "/dashboard/api/user/login"
	heatmapAPIPath = "/dashboard/api/keyvisual/heatmaps"

	heatMapTypeReadKeys  = "read_keys"
	heatMapTypeReadBytes = "read_bytes"

	loginTypePassword = 0
	loginTypeCode = 1
)

var formEncoder *schema.Encoder

func init() {
	formEncoder = schema.NewEncoder()
}


// KeyVizCollect is the collector collecting heatmap form dashboard
type KeyVizCollect struct {
	//endpoint    string
	cli         *utils.HttpClient
	outputDir   string // dir where the metrics data will be stored.
	loginMode   string
	username    string
	password    string
	beginTS     int64
	endTS       int64
}

type CollectOpt func(c *KeyVizCollect) error

func WithTimeRange(begin, end string) CollectOpt {
	return func(collect *KeyVizCollect) error {
		st, err := utils.ParseTime(begin)
		if err != nil {
			return err
		}
		et, err := utils.ParseTime(end)
		if err != nil {
			return err
		}
		collect.beginTS = st.Unix()
		collect.endTS = et.Unix()
		return nil
	}
}

func WithHttpClient(cli *http.Client) CollectOpt {
	return func(c *KeyVizCollect) error {
		c.cli = utils.New(utils.CliOption(cli))
		return nil
	}
}

func WithOutput(output string) CollectOpt {
	return func(c *KeyVizCollect) error {
		if err := utils.EnsureDir(output); err != nil{
			return err
		}
		c.outputDir = output
		return nil
	}
}

func NewKeyVizCollect(opts...CollectOpt) (*KeyVizCollect, error) {
	c := &KeyVizCollect{}
	for _, opt := range opts {
		if err := opt(c); err != nil {
			return nil, err
		}
	}
	return c, nil
}


func (c *KeyVizCollect) SetUserPwd(user string, pwd string) {
	c.username = user
	c.password = pwd
	c.loginMode = "password"
}

func (c *KeyVizCollect) SetSessionCode(code string) {
	// todo
	c.loginMode = "code"
	c.password = code
}

func (c *KeyVizCollect) Login(ctx context.Context, endpoint string) (string, error) {
	param := LoginParam{}

	if c.loginMode == "password"{
		param.Type = loginTypePassword
		param.Username = c.username
		param.Password = c.password
	}else if c.loginMode == "code" {
		param.Type = loginTypeCode
		param.Password = c.password
	}else {
		return "", errors.New("login mode not support")
	}
	data := LoginData{}
	fmt.Printf("param %+v\n",param)
	u := fmt.Sprintf("http://%s%s",endpoint, loginAPIPath)
	if err := c.cli.CallWithJson(ctx, &data, http.MethodPost, u, param); err != nil {
		return "", err
	}
	return data.Token, nil
}

func (c *KeyVizCollect) Collect(ctx context.Context, token string, endpoint string) error {
	// query read bytes
	if err := c.queryHeapMap(ctx, token, heatMapTypeReadKeys, endpoint); err != nil {
		return err
	}
	if err := c.queryHeapMap(ctx, token, heatMapTypeReadBytes, endpoint); err != nil {
		return err
	}
	return nil
}

func (c *KeyVizCollect) queryHeapMap(ctx context.Context, token string, typ string, endpoint string) error {
	param := QueryHeatMapParam{
		Type:      typ,
		Starttime: c.beginTS,
		Endtime:   c.endTS,
	}
	q, err := param.ToQueryString()
	if err != nil {
		fmt.Printf("parse to query string error: %+v\n", err)
		return err
	}

	u := fmt.Sprintf("http://%s%s?%s", endpoint, heatmapAPIPath, q)
	h := map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", token),
	}
	resp, err := c.cli.DoRequestWithJsonHeader(ctx, http.MethodGet, u, nil, h)
	if err != nil {
		fmt.Printf("query heapmap error: %+v\n", err)
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("query heatmap error: %s\n", resp.Status)
		return errors.New(resp.Status)
	}
	dst, err := os.OpenFile(
		filepath.Join(
			c.outputDir, fmt.Sprintf("%s.json", typ),
		), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		fmt.Printf("open file error: %+v\n",err)
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, resp.Body)
	if err != nil {
		fmt.Printf("write heapmap to file error: %+v\n", err)
		return err
	}
	return nil
}


// proto struct for api request parameter and response data

type LoginParam struct {
	Type     int    `json:"type"`
	Password string `json:"password"`
	Username string `json:"username,omitempty"`
}

type LoginData struct {
	Token  string `json:"token"`
	Expire string `json:"expire"`
}

type QueryHeatMapParam struct {
	Type      string `json:"type" schema:"type"`
	Starttime int64  `json:"starttime" schema:"starttime"`
	Endtime   int64  `json:"endtime" schema:"endtime"`
}

func (p *QueryHeatMapParam) ToQueryString() (string, error) {
	dst := make(url.Values)
	if err:= formEncoder.Encode(p, dst); err != nil {
		return "", err
	}
	return dst.Encode(), nil
}
