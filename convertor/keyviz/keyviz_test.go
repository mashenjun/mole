package keyviz

import (
	"os"
	"testing"
)

func TestHeatmapConvertor_Convert(t *testing.T) {
	hc, err := NewHeatmapConvertor()
	if err != nil {
		t.Fatal(err)
	}
	input := os.Getenv("INPUT_FILE")
	t.Log(input)
	sink := hc.GetSink()
	go func() {
		if err := hc.Convert(input); err != nil {
			t.Error(err)
			return
		}
	}()

	for s := range sink {
		t.Log(s)
	}
}
