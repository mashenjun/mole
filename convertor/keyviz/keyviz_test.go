package keyviz

import (
	"os"
	"testing"
)

func TestHeatmapConvertor_Convert(t *testing.T) {
	input := os.Getenv("INPUT_FILE")
	if len(input) == 0 {
		t.Skip("set INPUT_FILE to run the test")
	}
	t.Log(input)
	hc, err := NewHeatmapConvertor(WithInput(input))
	if err != nil {
		t.Fatal(err)
	}
	sink := hc.GetSink()
	//go func() {
	//	if err := hc.Convert(); err != nil {
	//		t.Error(err)
	//		return
	//	}
	//}()
	for s := range sink {
		t.Log(s)
	}
}

func TestNewGroupIndexConstructor(t *testing.T) {
	gic := NewGroupIndexConstructor()

	gic.Append("a", 1)
	gic.Append("a", 2)
	gic.Append("b", 3)
	gic.Append("c", 4)
	gi := gic.Result()
	if len(gi) != 3 {
		t.Fatal("not match")
	}
	t.Log(gi)
}
