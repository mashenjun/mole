package desensitize

import (
	"bytes"
	"github.com/pingcap/parser"
	"github.com/pingcap/parser/format"
	_ "github.com/pingcap/tidb/types/parser_driver"
	"io/ioutil"
	"testing"
)

func TestSchemaCollector_Parse(t *testing.T) {
	bs, err := ioutil.ReadFile("test.sql")
	if err != nil {
		t.Fatal(err)
	}
	p := parser.New()
	stmt, _, err := p.Parse(string(bs), "", "")
	if err != nil {
		t.Fatal(err)
	}
	enc, err := NewAESEncrypt([]byte("myverystrongpasswordo32bitlength"))
	if err != nil {
		t.Fatal(err)
	}
	cv, err := NewColumnNameVisitor(enc)
	if err != nil {
		t.Fatal(err)
	}
	v, err := NewSchemaVisitor(cv)
	if err != nil {
		t.Fatal(err)
	}
	nstmt, ok := stmt[0].Accept(v)
	buf := bytes.NewBuffer(make([]byte, 0))
	rctx := format.NewRestoreCtx(format.DefaultRestoreFlags, buf)
	if err := nstmt.Restore(rctx); err != nil {
		t.Fatal(err)
	}
	t.Log(buf.String(), ok)
}


