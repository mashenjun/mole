package collector

import (
	"errors"
	"fmt"
	"github.com/pingcap/parser"
	"github.com/pingcap/parser/ast"
	_ "github.com/pingcap/tidb/types/parser_driver"
	"gorm.io/gorm"
)

// SchemaCollector access target tidb and retrieve schema information
type SchemaCollector struct {
	db *gorm.DB
	sink chan *ast.CreateTableStmt
	parser *parser.Parser
}

func NewSchemaCollector(db *gorm.DB) (*SchemaCollector, error) {
	collector := &SchemaCollector{
		db: db,
		sink: make(chan *ast.CreateTableStmt),
		parser: parser.New(),
	}
	return collector, nil
}

func (sc *SchemaCollector) GetSink() <-chan *ast.CreateTableStmt {
	return sc.sink
}

func (sc *SchemaCollector) Collect() error {
	defer close(sc.sink)
	// get all table in the given db
	tbls := make([]string, 0)
	if err:= sc.db.Raw("SHOW TABLES").Scan(&tbls).Error; err != nil {
		return err
	}
	// for each table, retrieve schema
	var tableName string
	var schema string
	for _, tbl := range tbls {
		row := sc.db.Raw(fmt.Sprintf("SHOW CREATE TABLE %s", tbl)).Row()
		if err := row.Scan(&tableName, &schema); err != nil {
			return err
		}
		node, err := sc.parse(schema)
		if err != nil {
			return err
		}
		createTableStmt, ok := node.(*ast.CreateTableStmt)
		if !ok {
			return errors.New("statement not match")
		}
		sc.sink <- createTableStmt
	}
	return nil
}

func (sc *SchemaCollector) parse(sql string) (ast.Node, error) {
	stmts, _, err := sc.parser.Parse(sql, "","")
	if err != nil {
		return nil, err
	}
	return stmts[0], nil
}

