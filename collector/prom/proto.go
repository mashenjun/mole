package prom

import (
	"fmt"
	"github.com/mashenjun/mole/consts"
	"gopkg.in/yaml.v3"
	"io/ioutil"
)

type EndpointType int

const (
	EndpointPrometheus EndpointType = iota
	EndpointVMSelect
)

type Endpoint struct {
	Schema    string
	Host      string
	Port      string
	Type      EndpointType
	AccountID int
}

func (e *Endpoint) HostPort() string {
	return fmt.Sprintf("%s:%s", e.Host, e.Port)
}

func (e *Endpoint) Address() string {
	return fmt.Sprintf("%s://%s:%s", e.Schema, e.Host, e.Port)
}

func (e *Endpoint) WithPrefixPath(p string) string {
	switch e.Type {
	case EndpointPrometheus:
		return p
	case EndpointVMSelect:
		prefix := fmt.Sprintf(consts.VMSelectPromPrefix, e.AccountID)
		return fmt.Sprintf("%s%s", prefix, p)
	default:
		// do nothing
	}
	return p
}

type Meta struct {
	TiKVInstanceCnt int    `yaml:"tikv_instance_cnt"`
	BeginTimestamp  string `yaml:"begin_timestamp"`
	EndTimestamp    string `yaml:"end_timestamp"`
}

func (m *Meta) SaveTo(fileName string) error {
	data, err := yaml.Marshal(&m)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(fileName, data, 0644)
}
