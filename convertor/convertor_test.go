package convertor

import (
	"os"
	"testing"
)

func TestMetricsMatrixConvertor_Convert(t *testing.T) {
	mmc, err := NewMetricsMatrixConvertor()
	if err != nil {
		t.Fatal(err)
	}
	input := os.Getenv("INPUT_FILE")
	t.Log(input)
	sink := mmc.GetSink()
	go func() {
		if err := mmc.Convert(input); err != nil {
			t.Error(err)
			return
		}
	}()

	for s := range sink {
		t.Log(s)
	}
}
