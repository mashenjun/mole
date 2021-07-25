package desensitize

import (
	"context"
	"fmt"
	"github.com/pingcap/parser/ast"
	"github.com/pingcap/parser/format"
	"io"
	"strings"
)

var (
	endingBytes = []byte(";\n")
)

// SchemaVisitor traverse the statement tree and mask some information
type SchemaVisitor struct {
	columnNameVisitor ast.Visitor
}

func NewSchemaVisitor(cv ast.Visitor) (*SchemaVisitor, error) {
	return &SchemaVisitor{columnNameVisitor: cv}, nil
}

func (sv *SchemaVisitor) Enter(in ast.Node) (node ast.Node, skipChildren bool) {
	stmt, ok := in.(*ast.CreateTableStmt)
	if !ok {
		return in, true
	}
	// if the in node is create table statement
	// we need encrypt column name, index, partition
	// todo
	// replace column name in column def
	for _, col := range stmt.Cols {
		col.Name.Accept(sv.columnNameVisitor)
	}
	for i, ct := range stmt.Constraints {
		ct.Name = fmt.Sprintf("idx_%v", i)
		for _, col := range ct.Keys {
			col.Column.Accept(sv.columnNameVisitor)
		}
	}
	if stmt.Partition != nil {
		columnNameExpr, ok := stmt.Partition.PartitionMethod.Expr.(*ast.ColumnNameExpr)
		if ok {
			columnNameExpr.Name.Accept(sv.columnNameVisitor)
		}
	}
	return in, true
}

func (sv *SchemaVisitor) Leave(in ast.Node) (node ast.Node, ok bool) {
	//fmt.Printf("Leave: %#v\n", in)
	return in, true
}

type ColumnNameVisitor struct {
	enc *AESEncrypt
}

func NewColumnNameVisitor(enc *AESEncrypt) (*ColumnNameVisitor,error){
	return &ColumnNameVisitor{
		enc: enc,
	}, nil
}

func (cv *ColumnNameVisitor) Enter(in ast.Node) (node ast.Node, skipChildren bool) {
	columnName, ok := in.(*ast.ColumnName);
	if !ok {
		return in, true
	}
	columnName.Name.O = cv.enc.Encrypt(columnName.Name.O)
	columnName.Name.L = strings.ToLower(columnName.Name.O)
	return in, true
}

func (cv *ColumnNameVisitor) Leave(in ast.Node) (node ast.Node, ok bool) {
	return in, ok
}

type SchemaMask struct {
	visitor ast.Visitor
	source  <-chan *ast.CreateTableStmt
	sink    io.WriteCloser
}

// TODO should use opt function style
func NewSchemaMask(enc *AESEncrypt, source <-chan *ast.CreateTableStmt, sink io.WriteCloser) (*SchemaMask, error) {
	cv, err := NewColumnNameVisitor(enc)
	if err != nil {
		return nil, err
	}
	sv, err := NewSchemaVisitor(cv)
	if err != nil {
		return nil, err
	}
	sm := &SchemaMask{
		visitor: sv,
		source:  source,
		sink:    sink,
	}
	return sm, nil
}

func (sm *SchemaMask) Start(ctx context.Context) error {
	for {
		select {
		case stmt, ok := <-sm.source:
			if !ok {
				// no more stmt, close the writer
				return sm.sink.Close()
			}
			encryptedStmt, ok := stmt.Accept(sm.visitor)
			if !ok {
				fmt.Println("encrypt schema wrong")
			}
			rctx := format.NewRestoreCtx(format.DefaultRestoreFlags, sm.sink)
			if err := encryptedStmt.Restore(rctx); err != nil {
				fmt.Println(err)
			}
			n, err := sm.sink.Write(endingBytes)
			if err != nil {
				return err
			}
			if n != 2 {
				fmt.Println("warning write incorrect")
			}
		case <-ctx.Done():
			fmt.Println("ctx done")
			return sm.sink.Close()
		}
	}
}
