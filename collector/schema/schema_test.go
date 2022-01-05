package schema

import (
	"os"
	"testing"
)

func TestSchemaCollector_Collect(t *testing.T) {
	dsn := os.Getenv("TIDB_DSN")
	if len(dsn) == 0 {
		t.Skip("set TIDB_DSN to run the test")
	}
	config := &MysqlConfig{
		DSN: dsn,
	}
	db, err := Dial(config)
	if err != nil {
		t.Fatal(err)
	}
	c, err := NewSchemaCollector(db, "test")
	if err != nil {
		t.Fatal(err)
	}
	sink := c.GetSink()
	go func() {
		if err := c.Collect(); err != nil {
			panic(err)
		}
	}()
	for n := range sink {
		t.Log(n.Text())
	}
}
