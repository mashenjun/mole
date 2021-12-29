package prom

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"io/ioutil"
)

type Endpoint struct {
	Schema string
	Host   string
	Port   string
}

func (e *Endpoint) Address() string {
	return fmt.Sprintf("%s:%s", e.Host, e.Port)
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
