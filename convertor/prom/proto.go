package prom

import (
	"encoding/json"
	"fmt"
	"github.com/prometheus/common/model"
)

// copy from prometheus client golang.

type MetricsResp struct {
	Status string              `json:"status"`
	Data   *MetricsQueryResult `json:"data"`
}

type MetricsQueryResult struct {
	// The decoded value.
	v model.Value
}

func (qr *MetricsQueryResult) UnmarshalJSON(b []byte) error {
	v := struct {
		Type   model.ValueType `json:"resultType"`
		Result json.RawMessage `json:"result"`
	}{}

	err := json.Unmarshal(b, &v)
	if err != nil {
		return err
	}

	switch v.Type {
	case model.ValScalar:
		var sv model.Scalar
		err = json.Unmarshal(v.Result, &sv)
		qr.v = &sv

	case model.ValVector:
		var vv model.Vector
		err = json.Unmarshal(v.Result, &vv)
		qr.v = vv

	case model.ValMatrix:
		var mv model.Matrix
		err = json.Unmarshal(v.Result, &mv)
		qr.v = mv

	default:
		err = fmt.Errorf("unexpected value type %q", v.Type)
	}
	return err
}
