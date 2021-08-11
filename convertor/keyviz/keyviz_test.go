package keyviz

import (
	"os"
	"testing"
)

func TestHeatmapConvertor_Convert(t *testing.T) {
	input := os.Getenv("INPUT_FILE")
	t.Log(input)
	hc, err := NewHeatmapConvertor(WithInput(input))
	if err != nil {
		t.Fatal(err)
	}

	sink := hc.GetSink()
	go func() {
		if err := hc.Convert(); err != nil {
			t.Error(err)
			return
		}
	}()

	for s := range sink {
		t.Log(s)
	}
}
