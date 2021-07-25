package desensitize

import (
	"context"
	"github.com/pingcap/parser"
	"github.com/pingcap/parser/ast"
	"io/ioutil"
	"os"
	"testing"
)

func TestSchemaMask_Start(t *testing.T) {
	sql, err := ioutil.ReadFile("test.sql")
	if err != nil {
		t.Fatal(err)
	}
	parser := parser.New()
	nodes1, _,  err := parser.Parse(string(sql), "", "")
	if err != nil {
		t.Fatal(err)
	}
	createTableStmt1, ok := nodes1[0].(*ast.CreateTableStmt)
	if !ok {
		t.Fatal("type not match")
	}
	nodes2, _,  err := parser.Parse(string(sql), "", "")
	if err != nil {
		t.Fatal(err)
	}
	createTableStmt2, ok := nodes2[0].(*ast.CreateTableStmt)
	if !ok {
		t.Fatal("type not match")
	}
	source := make(chan *ast.CreateTableStmt,2)
	source <- createTableStmt1
	source <- createTableStmt2
	close(source)
	enc, err := NewAESEncrypt([]byte("myverystrongpasswordo32bitlength"))
	if err != nil {
		t.Fatal(err)
	}
	sink, err := os.Create("out.sql")
	if err != nil {
		t.Fatal(err)
	}
	sm, err := NewSchemaMask(enc, source, sink)
	if err != nil {
		t.Fatal(err)
	}
	if err := sm.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
}