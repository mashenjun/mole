package prom

import (
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
		if err := mmc.Convert(); err != nil {
			t.Error(err)
			return
		}
	}()

	for s := range sink {
		t.Log(s)
	}
}
