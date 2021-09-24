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
	 TiKVInstanceCnt int `yaml:"tikv_instance_cnt"`
}

func (m *Meta) SaveFile(fileName string) error {
	data, err := yaml.Marshal(&m)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(fileName, data, 0644)
}
