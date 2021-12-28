package prom

import (
	"context"
	"encoding/json"
	"github.com/prometheus/common/model"
	"os"
	"testing"
)

func TestMetricsMatrixConvertor_Convert(t *testing.T) {
	input := os.Getenv("INPUT_FILE")
	mmc, err := NewMetricsMatrixConvertor(WithInput(input))
	if err != nil {
		t.Fatal(err)
	}
	sink := mmc.GetSink()
	go func() {
		if err := mmc.Convert(context.Background()); err != nil {
			t.Error(err)
			return
		}
	}()

	for s := range sink {
		t.Log(s)
	}
}

func TestCheckAlign(t *testing.T) {
	bs, err := os.ReadFile("../../testdata/gap_metrics.json")
	if err != nil {
		t.Error(err)
	}
	resp := MetricsResp{}
	if err := json.Unmarshal(bs, &resp); err != nil {
		t.Error(err)
	}
	matrix, _ := resp.Data.v.(model.Matrix)
	align, total, gap := checkAlign(matrix)
	if align {
		t.Error("align should be false")
	}
	if total != 480 {
		t.Error("total should be 480")
	}
	if !gap.InAnyGap(41) {
		t.Error("in group return true, should be false")
	}
}
