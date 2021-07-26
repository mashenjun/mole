package collector

import (
	"os"
	"testing"
)

func TestSchemaCollector_Collect(t *testing.T) {
	config := &MysqlConfig{
		DSN:          os.Getenv("TIDB_DSN"),
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
		for n := range sink {
			t.Log(n.Text())
		}
	}()
	if err := c.Collect(); err != nil {
		t.Fatal(err)
	}

}
