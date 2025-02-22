package parser

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"iter"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"text/tabwriter"
	"unicode"

	"github.com/joaobnv/mel/sqlite/v3_46_1/lexer"
	"github.com/joaobnv/mel/sqlite/v3_46_1/parsetree"
	"github.com/joaobnv/mel/sqlite/v3_46_1/token"
)

// testCase is a test case.
type testCase struct {
	code string
	tree string
}

// testCases returns a slice of testCase from a list of code, tree pairs.
// testCases panics if given an odd number of arguments.
func testCases(p ...string) (cases []testCase) {
	if len(p)%2 == 1 {
		panic(errors.New("len(p) must be even"))
	}
	for c := range slices.Chunk(p, 2) {
		cases = append(cases, testCase{code: c[0], tree: c[1]})
	}
	return cases
}

func TestSQLStatement(t *testing.T) {
	t.Parallel()
	cases := testCases(
		`EXPLAIN ALTER TABLE table_a ADD COLUMN column_b NOT NULL AS (10) VIRTUAL;`,
		`SQLStatement {Explain {T AlterTable {TT TableName AddColumn {TT ColDef {ColName
			ColConstr {NotNullColumnConstraint{TT}}
			ColConstr {GeneratedColumnConstraint{TT E{T} T T} }}}}} T}`,
		`EXPLAIN QUERY PLAN ALTER TABLE table_a ADD COLUMN column_b NOT NULL AS (10) VIRTUAL;`,
		`SQLStatement {ExplainQueryPlan {TTT AlterTable {TT TableName AddColumn {TT ColDef {ColName
			ColConstr {NotNullColumnConstraint{TT}} ColConstr{GeneratedColumnConstraint{T T E{T} T T}} }}}} T}`,
		`EXPLAIN QUERY ALTER TABLE table_a ADD COLUMN column_b NOT NULL AS (10) VIRTUAL;`,
		`SQLStatement {ExplainQueryPlan {TT !ErrorMissing AlterTable {TT TableName AddColumn {TT ColDef {ColName
			ColConstr {NotNullColumnConstraint{TT}} ColConstr{GeneratedColumnConstraint{T T E{T} T T}} }}}} T}`,
		`ALTER TABLE table_a RENAME TO table_b`,
		"SQLStatement {AlterTable {TT TableName RenameTo {TT TableName}} T}",
		`ANALYZE schema_name`,
		"SQLStatement {Analyze {T SchemaIndexOrTableName} T}",
		`ATTACH DATABASE ':memory' AS schema_name`,
		"SQLStatement {Attach {TT E{T} T SchemaName} T}",
		`BEGIN`,
		"SQLStatement {Begin {T} T}",
		`COMMIT`,
		"SQLStatement {Commit {T} T}",
		`ROLLBACK`,
		"SQLStatement {Rollback {T} T}",
		`CREATE INDEX index_name ON table_name(column_name)`,
		"SQLStatement {CreateIndex {TT IndexName T TableName T CommaList{IndexedColumn{E{ColRef{ColName}}}} T} T}",
		`CREATE TABLE table_name (column_a);`,
		"SQLStatement {CreateTable {TT TableName T CommaList{ColDef{ColName}} T} T}",
		`SELECT 10;`,
		"SQLStatement {Select {T E{T}} T}",
		`SELECT 10 10;`,
		"SQLStatement {Select {T E{T}} Skipped{T} T}",
	)

	for i, c := range cases {
		c := c
		t.Run(strconv.FormatInt(int64(i), 10), func(t *testing.T) {
			t.Parallel()
			tp := newTestParser(newTestLexer(c.tree))
			expected := tp.tree()

			p := New(lexer.New([]byte(c.code)))
			parsed, comments := p.SQLStatement()

			if str, equals := compare(c.code, comments, parsed, expected); !equals {
				fmt.Println(c.code)
				fmt.Println(str)
				t.Fail()
			}
		})
	}
}

func TestAlterTable(t *testing.T) {
	t.Parallel()
	cases := testCases(
		`ALTER TABLE table_a RENAME TO table_b`,
		"AlterTable {TT TableName RenameTo {TT TableName}}",
		`ALTER TABLE schema_a.table_a RENAME TO table_b`,
		"AlterTable {TT SchemaName T TableName RenameTo {TT TableName}}",
		`ALTER TABLE table_a RENAME column_a TO column_b`,
		"AlterTable {TT TableName RenameColumn {T ColumnName T ColumnName}}",
		`ALTER TABLE /* comment */ table_a RENAME COLUMN column_a TO column_b`,
		"AlterTable {TT TableName RenameColumn {TT ColumnName T ColumnName}}",
		`ALTER TABLE table_a ADD column_b INTEGER INTEGER`,
		"AlterTable {TT TableName AddColumn {T ColumnDefinition {ColumnName TypeName{TT}}}}",
		`ALTER TABLE table_a ADD COLUMN column_b INTEGER(10)`,
		"AlterTable {TT TableName AddColumn {TT ColumnDefinition {ColumnName TypeName {TTTT}}}}",
		`ALTER TABLE table_a ADD COLUMN column_b INTEGER(10, 20) PRIMARY KEY`,
		"AlterTable {TT TableName AddColumn {TT ColumnDefinition {ColumnName TypeName {TTTTTT} ColConstr {PrimaryKeyColumnConstraint{TT}} }}}",
		`ALTER TABLE table_a DROP column_b`,
		"AlterTable {TT TableName DropColumn {T ColumnName}}",
		`ALTER TABLE table_a DROP COLUMN column_b`,
		"AlterTable {TT TableName DropColumn {TT ColumnName}}",
		`ALTER TABLE RENAME TO table_b`,
		"AlterTable {TT !ErrorMissing RenameTo {TT TableName}}",
		`ALTER TABLE schema_a. RENAME TO table_b`,
		"AlterTable {TT SchemaName T !ErrorMissing RenameTo {TT TableName}}",
		`ALTER TABLE table_a RENAME TO `,
		"AlterTable {TT TableName RenameTo {TT !ErrorMissing}}",
		`ALTER TABLE table_a RENAME column_a TO `,
		"AlterTable {TT TableName RenameColumn {T ColumnName T !ErrorMissing}}",
		`ALTER TABLE table_a 10 RENAME column_a TO `,
		"AlterTable {TT TableName Skipped {T} RenameColumn {T ColumnName T !ErrorMissing}}",
		`ALTER`,
		"AlterTable {T !ErrorUnexpectedEOF}",
		`ALTER TABLE table_a RENAME column_a column_b `,
		"AlterTable {TT TableName RenameColumn {T ColumnName !ErrorMissing ColumnName}}",
		`ALTER TABLE table_a RENAME COLUMN TO column_b `,
		"AlterTable {TT TableName RenameColumn {TT !ErrorMissing T ColumnName}}",
		`ALTER TABLE table_a ADD COLUMN `,
		"AlterTable {TT TableName AddColumn {TT !ErrorMissing}}",
		`ALTER TABLE table_a ADD COLUMN column_a INTEGER()`,
		"AlterTable {TT TableName AddColumn {TT ColumnDefinition {ColumnName TypeName {T T !ErrorMissing T}}}}",
		`ALTER TABLE table_a ADD COLUMN column_a INTEGER 10)`,
		"AlterTable {TT TableName AddColumn {TT ColumnDefinition {ColumnName TypeName {T !ErrorMissing TT}}}}",
		`ALTER TABLE table_a DROP COLUMN`,
		"AlterTable {TT TableName DropColumn {TT !ErrorMissing}}",
	)

	runTests(t, cases, (*Parser).alterTable)
}

func TestColumnDefinition(t *testing.T) {
	t.Parallel()
	cases := testCases(
		`a INTEGER PRIMARY KEY`,
		"ColDef{ColName TypeName{T} ColConstr{PrimaryKeyColumnConstraint{TT}}}",
	)

	runTests(t, cases, (*Parser).columnDefinition)
}

func TestTypeName(t *testing.T) {
	t.Parallel()
	cases := testCases(
		`INTEGER`, "TypeName{T}",
		`INTEGER(10)`, "TypeName{TTTT}",
		`INTEGER(+10, -10)`, "TypeName{TTTTTTTT}",
		`INTEGER -10)`, "TypeName{T !ErrorMissing TTT}",
		`INTEGER(, -10)`, "TypeName{TT !ErrorMissing TTTT}",
		`INTEGER(-10, )`, "TypeName{TTTTT !ErrorMissing T}",
		`INTEGER(-10 10)`, "TypeName{TTTT !ErrorMissing TT}",
	)

	runTests(t, cases, (*Parser).typeName)
}

func TestColumnConstraint(t *testing.T) {
	t.Parallel()
	cases := testCases(
		`CONSTRAINT constr PRIMARY KEY`,
		"ColConstr{T ConstraintName PrimaryKeyColumnConstraint{TT}}",
		`NOT NULL`,
		"ColConstr{NotNullColumnConstraint{TT}}",
		`UNIQUE`,
		"ColConstr{UniqueColumnConstraint{T}}",
		`CHECK(10)`,
		"ColConstr{CheckColumnConstraint{T T E{T} T}}",
		`DEFAULT NULL`,
		"ColConstr{DefaultColumnConstraint{TT}}",
		`COLLATE c`,
		"ColConstr{CollateColumnConstraint{T CollationName}}",
		`REFERENCES table_name`,
		"ColConstr{ForeignKeyColumnConstraint{ForeignKeyClause{T TableName}}}",
		`AS (10)`,
		"ColConstr{GeneratedColumnConstraint{TT E{T} T}}",
		`CONSTRAINT PRIMARY KEY`,
		"ColConstr{T !ErrorMissing PrimaryKeyColumnConstraint{TT}}",
		`10`,
		"ColConstr{!ErrorExpecting}",
	)

	runTests(t, cases, (*Parser).columnConstraint)
}

func TestPrimaryKeyColumnConstraint(t *testing.T) {
	t.Parallel()
	cases := testCases(
		`PRIMARY KEY`, "PrimaryKeyColumnConstraint{TT}",
		`PRIMARY KEY ASC ON CONFLICT ROLLBACK`,
		"PrimaryKeyColumnConstraint{TTT ConflictClause{TTT}}",
		`PRIMARY KEY DESC AUTOINCREMENT`, "PrimaryKeyColumnConstraint{TTTT}",
		`PRIMARY`, "PrimaryKeyColumnConstraint{T !ErrorMissing}",
	)

	runTests(t, cases, (*Parser).primaryKeyColumnConstraint)
}

func TestNotNullColumnConstraint(t *testing.T) {
	t.Parallel()
	cases := testCases(
		`NOT NULL`, "NotNullColumnConstraint{TT}",
		`NOT NULL ON CONFLICT ROLLBACK`,
		"NotNullColumnConstraint{TT ConflictClause{TTT}}",
		`NOT`, "NotNullColumnConstraint{T !ErrorMissing}",
	)

	runTests(t, cases, (*Parser).notNullColumnConstraint)
}

func TestUniqueColumnConstraint(t *testing.T) {
	t.Parallel()
	cases := testCases(
		`UNIQUE`, "UniqueColumnConstraint{T}",
		`UNIQUE ON CONFLICT ROLLBACK`,
		"UniqueColumnConstraint{T ConflictClause{TTT}}",
	)

	runTests(t, cases, (*Parser).uniqueColumnConstraint)
}

func TestCheckColumnConstraint(t *testing.T) {
	t.Parallel()
	cases := testCases(
		`CHECK(a > 10)`,
		"CheckColumnConstraint{TT E{GreaterThan{ColRef{ColName} TT}} T}",
		`CHECK a > 10)`,
		"CheckColumnConstraint{T !ErrorMissing E{GreaterThan{ColRef{ColName} TT}} T}",
		`CHECK()`,
		"CheckColumnConstraint{TT !ErrorMissing T}",
		`CHECK(a > 10`,
		"CheckColumnConstraint{TT E{GreaterThan{ColRef{ColName} TT}} !ErrorMissing}",
	)

	runTests(t, cases, (*Parser).checkColumnConstraint)
}

func TestDefaultColumnConstraint(t *testing.T) {
	t.Parallel()
	cases := testCases(
		`DEFAULT(a > 10)`,
		"DefaultColumnConstraint{TT E{GreaterThan{ColRef{ColName} TT}} T}",
		`DEFAULT 10`,
		"DefaultColumnConstraint{T T}",
		`DEFAULT -10`,
		"DefaultColumnConstraint{T TT}",
		`DEFAULT a`,
		"DefaultColumnConstraint{T !ErrorExpecting}",
		`DEFAULT()`,
		"DefaultColumnConstraint{TT !ErrorMissing T}",
		`DEFAULT(a > 10`,
		"DefaultColumnConstraint{TT E{GreaterThan{ColRef{ColName} TT}} !ErrorMissing}",
		`DEFAULT -a`,
		"DefaultColumnConstraint{TT !ErrorMissing}",
	)

	runTests(t, cases, (*Parser).defaultColumnConstraint)
}

func TestCollateColumnConstraint(t *testing.T) {
	t.Parallel()
	cases := testCases(
		`COLLATE c`,
		"CollateColumnConstraint{T CollationName}",
		`COLLATE`,
		"CollateColumnConstraint{T !ErrorMissing}",
	)

	runTests(t, cases, (*Parser).collateColumnConstraint)
}

func TestGeneratedColumnConstraint(t *testing.T) {
	t.Parallel()
	cases := testCases(
		`GENERATED ALWAYS AS (10)`,
		"GeneratedColumnConstraint{TTTT E{T} T}",
		`AS (10) STORED`,
		"GeneratedColumnConstraint{TT E{T} T T}",
		`AS (10) VIRTUAL`,
		"GeneratedColumnConstraint{TT E{T} T T}",
		`GENERATED AS (10)`,
		"GeneratedColumnConstraint{T !ErrorMissing TT E{T} T}",
		`GENERATED ALWAYS (10)`,
		"GeneratedColumnConstraint{TT !ErrorMissing T E{T} T}",
		`AS ()`,
		"GeneratedColumnConstraint{TT !ErrorMissing T}",
		`AS 10)`,
		"GeneratedColumnConstraint{T !ErrorMissing E{T} T}",
		`AS (10`,
		"GeneratedColumnConstraint{TT E{T} !ErrorMissing}",
	)

	runTests(t, cases, (*Parser).generatedColumnConstraint)
}

func TestForeignKeyColumnConstraint(t *testing.T) {
	t.Parallel()
	cases := testCases(
		`REFERENCES table_name`,
		"ForeignKeyColumnConstraint{ForeignKeyClause{T TableName}}",
	)

	runTests(t, cases, (*Parser).foreignKeyColumnConstraint)
}

func TestForeignKeyClause(t *testing.T) {
	t.Parallel()
	cases := testCases(
		`REFERENCES table_name`,
		"ForeignKeyClause{T TableName}",
		`REFERENCES table_name(column_name)`,
		"ForeignKeyClause{T TableName T CommaList{ColumnName} T}",
		`REFERENCES table_name ON DELETE SET NULL`,
		"ForeignKeyClause{T TableName TTTT}",
		`REFERENCES table_name ON UPDATE SET DEFAULT`,
		"ForeignKeyClause{T TableName TTTT}",
		`REFERENCES table_name ON DELETE CASCADE`,
		"ForeignKeyClause{T TableName TTT}",
		`REFERENCES table_name ON UPDATE RESTRICT`,
		"ForeignKeyClause{T TableName TTT}",
		`REFERENCES table_name ON DELETE NO ACTION`,
		"ForeignKeyClause{T TableName TTTT}",
		`REFERENCES table_name MATCH name`,
		"ForeignKeyClause{T TableName TT}",
		`REFERENCES table_name DEFERRABLE`,
		"ForeignKeyClause{T TableName T}",
		`REFERENCES table_name NOT DEFERRABLE`,
		"ForeignKeyClause{T TableName TT}",
		`REFERENCES table_name DEFERRABLE INITIALLY DEFERRED`,
		"ForeignKeyClause{T TableName TTT}",
		`REFERENCES table_name DEFERRABLE INITIALLY IMMEDIATE`,
		"ForeignKeyClause{T TableName TTT}",
		`REFERENCES`,
		"ForeignKeyClause{T !ErrorMissing}",
		`REFERENCES table_name(column_name`,
		"ForeignKeyClause{T TableName T CommaList{ColumnName} !ErrorMissing}",
		`REFERENCES table_name ON SET NULL`,
		"ForeignKeyClause{T TableName T !ErrorExpecting TT}",
		`REFERENCES table_name ON UPDATE SET`,
		"ForeignKeyClause{T TableName TTT !ErrorExpecting}",
		`REFERENCES table_name ON DELETE NO`,
		"ForeignKeyClause{T TableName TTT !ErrorExpecting}",
		`REFERENCES table_name ON UPDATE`,
		"ForeignKeyClause{T TableName TT !ErrorExpecting}",
		`REFERENCES table_name MATCH`,
		"ForeignKeyClause{T TableName T !ErrorMissing}",
		`REFERENCES table_name DEFERRABLE INITIALLY`,
		"ForeignKeyClause{T TableName TT !ErrorExpecting}",
	)

	runTests(t, cases, (*Parser).foreignKeyClause)
}

func TestConflictClause(t *testing.T) {
	t.Parallel()
	cases := testCases(
		`ON CONFLICT ROLLBACK`, "ConflictClause{TTT}",
		`ON CONFLICT ABORT`, "ConflictClause{TTT}",
		`ON CONFLICT FAIL`, "ConflictClause{TTT}",
		`ON CONFLICT IGNORE`, "ConflictClause{TTT}",
		`ON CONFLICT REPLACE`, "ConflictClause{TTT}",
		`ON REPLACE`, "ConflictClause{T !ErrorMissing T}",
		`ON CONFLICT`, "ConflictClause{TT !ErrorExpecting}",
	)

	runTests(t, cases, (*Parser).conflictClause)
}

func TestAnalyze(t *testing.T) {
	t.Parallel()
	cases := testCases(
		`ANALYZE schema_name`,
		"Analyze {T SchemaIndexOrTableName}",
		`ANALYZE schema_name.table_name`,
		"Analyze {T SchemaName T TableOrIndexName}",
		`ANALYZE `,
		"Analyze {T !ErrorMissing}",
		`ANALYZE schema_name.`,
		"Analyze {T SchemaName T !ErrorMissing}",
		`ANALYZE .table_name`,
		"Analyze {T !ErrorMissing T TableOrIndexName}",
		`ANALYZE .`,
		"Analyze {T !ErrorMissing T !ErrorMissing}",
	)

	runTests(t, cases, (*Parser).analyze)
}

func TestAttach(t *testing.T) {
	t.Parallel()
	cases := testCases(
		`ATTACH DATABASE ':memory' AS schema_name`, "Attach {TT E{T} T SchemaName}",
		`ATTACH '' AS schema_name`, "Attach {T E{T} T SchemaName}",
		`ATTACH ;`, "Attach {T !ErrorMissing}",
		`ATTACH AS ;`, "Attach {T !ErrorMissing T !ErrorMissing}",
		`ATTACH ':memory' schema_name ;`, "Attach {T E{T} !ErrorMissing SchemaName}",
	)

	runTests(t, cases, (*Parser).attach)
}

func TestBegin(t *testing.T) {
	t.Parallel()
	cases := testCases(
		`BEGIN`, "Begin {T}",
		`BEGIN DEFERRED TRANSACTION`, "Begin {TTT}",
		`BEGIN IMMEDIATE`, "Begin {TT}",
		`BEGIN EXCLUSIVE TRANSACTION`, "Begin {TTT}",
	)

	runTests(t, cases, (*Parser).begin)
}

func TestCommit(t *testing.T) {
	t.Parallel()
	cases := testCases(
		`COMMIT`, "Commit {T}",
		`COMMIT TRANSACTION`, "Commit {TT}",
		`END`, "Commit {T}",
		`END TRANSACTION`, "Commit {TT}",
	)

	runTests(t, cases, (*Parser).commit)
}

func TestRollback(t *testing.T) {
	t.Parallel()
	cases := testCases(
		`ROLLBACK`,
		"Rollback {T}",
		`ROLLBACK TRANSACTION`,
		"Rollback {TT}",
		`ROLLBACK TRANSACTION TO save_point_name`,
		"Rollback {TTT SavepointName}",
		`ROLLBACK TO SAVEPOINT save_point_name`,
		"Rollback {TTT SavepointName}",
		`ROLLBACK TO`,
		"Rollback {TT !ErrorMissing}",
		`ROLLBACK TO SAVEPOINT`,
		"Rollback {TTT !ErrorMissing}",
	)

	runTests(t, cases, (*Parser).rollback)
}

func TestCreateIndex(t *testing.T) {
	t.Parallel()
	cases := testCases(
		`CREATE INDEX index_name ON table_name(column_name)`,
		"CreateIndex {TT IndexName T TableName T CommaList{IndexedColumn{E{ColRef{ColName}}}} T}",
		`CREATE UNIQUE INDEX IF NOT EXISTS index_name ON table_name(column_name)`,
		"CreateIndex {TTT TTT IndexName T TableName T CommaList{IndexedColumn{E{ColRef{ColName}}}} T}",
		`CREATE INDEX schema_name.index_name ON table_name(column_name)`,
		"CreateIndex {TT SchemaName T IndexName T TableName T CommaList{IndexedColumn{E{ColRef{ColName}}}} T}",
		`CREATE INDEX index_name ON table_name(a + b ASC)`,
		"CreateIndex {TT IndexName T TableName T CommaList{IndexedColumn{E{Add{ColRef{ColName} T ColRef{ColName}}} T}} T}",
		`CREATE INDEX index_name ON table_name(a * b COLLATE collation_name DESC)`,
		"CreateIndex {TT IndexName T TableName T CommaList{IndexedColumn{E{Multiply{ColRef{ColName} T Collate{ColRef{ColName} T CollationName}}} T}} T}",
		`CREATE INDEX index_name ON table_name(column_name1, column_name2)`,
		"CreateIndex {TT IndexName T TableName T CommaList{IndexedColumn{E{ColRef{ColName}}} T IndexedColumn{E{ColRef{ColName}}}} T}",
		`CREATE INDEX index_name ON table_name(column_name) WHERE column_a > 10`,
		"CreateIndex {TT IndexName T TableName T CommaList{IndexedColumn{E{ColRef{ColName}}}} T T E{GreaterThan{ColRef{ColName} T T}}}",
		`CREATE INDEX IF EXISTS index_name ON table_name(column_name)`,
		"CreateIndex {TTT !ErrorMissing T IndexName T TableName T CommaList{IndexedColumn{E{ColRef{ColName}}}} T}",
		`CREATE INDEX IF NOT index_name ON table_name(column_name)`,
		"CreateIndex {TTTT !ErrorMissing IndexName T TableName T CommaList{IndexedColumn{E{ColRef{ColName}}}} T}",
		`CREATE INDEX .index_name ON table_name(column_name)`,
		"CreateIndex {TT !ErrorMissing T IndexName T TableName T CommaList{IndexedColumn{E{ColRef{ColName}}}} T}",
		`CREATE INDEX ON table_name(column_name)`,
		"CreateIndex {TT !ErrorMissing T TableName T CommaList{IndexedColumn{E{ColRef{ColName}}}} T}",
		`CREATE INDEX index_name table_name(column_name)`,
		"CreateIndex {TT IndexName !ErrorMissing TableName T CommaList{IndexedColumn{E{ColRef{ColName}}}} T}",
		`CREATE INDEX index_name ON (column_name)`,
		"CreateIndex {TT IndexName T !ErrorMissing T CommaList{IndexedColumn{E{ColRef{ColName}}}} T}",
		`CREATE INDEX index_name ON table_name column_name)`,
		"CreateIndex {TT IndexName T TableName !ErrorMissing CommaList{IndexedColumn{E{ColRef{ColName}}}} T}",
		`CREATE INDEX index_name ON table_name(column_name `,
		"CreateIndex {TT IndexName T TableName T CommaList{IndexedColumn{E{ColRef{ColName}}}} !ErrorMissing}",
		`CREATE INDEX index_name ON table_name(column_name) WHERE `,
		"CreateIndex {TT IndexName T TableName T CommaList{IndexedColumn{E{ColRef{ColName}}}} TT !ErrorMissing} ",
	)

	runTests(t, cases, (*Parser).createIndex)
}

func TestIndexedColumn(t *testing.T) {
	t.Parallel()
	cases := testCases(
		`column_name`,
		"IndexedColumn{ColName}",
		`column_name COLLATE c ASC`,
		"IndexedColumn{ColName T CollationName T}",
		`column_name COLLATE ASC`,
		"IndexedColumn{ColName T !ErrorMissing T}",
	)

	fn := func(p *Parser) parsetree.NonTerminal {
		return p.indexedColumn(false)
	}

	runTests(t, cases, fn)
}

func TestIndexedColumnExpression(t *testing.T) {
	t.Parallel()
	cases := testCases(
		`column_name`,
		"IndexedColumn{E{ColRef{ColName}}}",
		`column_name ASC`,
		"IndexedColumn{E{ColRef{ColName}} T}",
		`column_name DESC`,
		"IndexedColumn{E{ColRef{ColName}} T}",
	)

	fn := func(p *Parser) parsetree.NonTerminal {
		return p.indexedColumn(true)
	}

	runTests(t, cases, fn)
}

func TestCreateTable(t *testing.T) {
	t.Parallel()
	cases := testCases(
		`CREATE TABLE table_name (column_a);`,
		"CreateTable {TT TableName T CommaList{ColDef{ColName}} T}",
		`CREATE TEMP TABLE IF NOT EXISTS temp.table_name (column_a INTEGER);`,
		"CreateTable {TTT TTT SchemaName T TableName T CommaList{ColDef{ColName TypeName{T}}} T}",
		`CREATE TEMPORARY TABLE table_name AS SELECT 'value';`,
		"CreateTable {TTT TableName T Select{T E{T}}}",
		`CREATE TEMP TABLE IF NOT EXISTS temp.table_name (column_a INTEGER);`,
		"CreateTable {TTT TTT SchemaName T TableName T CommaList{ColDef{ColName TypeName{T}}} T}",
		`CREATE TABLE table_a (column_b INTEGER INTEGER);`,
		"CreateTable {TT TableName T CommaList{ColDef {ColName TypeName{TT}}} T}",
		`CREATE TABLE table_a (column_b INTEGER(10));`,
		"CreateTable {TT TableName T CommaList{ColDef {ColName TypeName {TTTT}}} T}",
		`CREATE TABLE table_a (column_b INTEGER(10, 20) PRIMARY KEY);`,
		`CreateTable {TT TableName T CommaList{ColDef{ColName TypeName {TTTTTT}
			ColConstr {PrimaryKeyColumnConstraint{TT}} }} T}`,
		`CREATE TABLE table_a (column_b) WITHOUT ROWID`,
		`CreateTable {TT TableName T CommaList{ColDef{ColName}} T CommaList{TableOption{TT}}}`,
		`CREATE TABLE table_a`,
		`CreateTable {TT TableName !ErrorExpecting}`,
		`CREATE TABLE IF EXISTS table_name (column_a INTEGER);`,
		"CreateTable {TT T !ErrorMissing T TableName T CommaList{ColDef{ColName TypeName{T}}} T}",
		`CREATE TABLE IF NOT table_name (column_a INTEGER);`,
		"CreateTable {TT TT !ErrorMissing TableName T CommaList{ColDef{ColName TypeName{T}}} T}",
		`CREATE TABLE .table_name (column_a INTEGER);`,
		"CreateTable {TT !ErrorMissing T TableName T CommaList{ColDef{ColName TypeName{T}}} T}",
		`CREATE TABLE (column_a INTEGER);`,
		"CreateTable {TT !ErrorMissing T CommaList{ColDef{ColName TypeName{T}}} T}",
		`CREATE TABLE table_name (column_a INTEGER`,
		"CreateTable {TT TableName T CommaList{ColDef{ColName TypeName{T}}} !ErrorMissing}",
		`CREATE TEMPORARY TABLE table_name AS`,
		"CreateTable {TTT TableName T !ErrorMissing}",
	)

	runTests(t, cases, (*Parser).createTable)
}

func TestColumnDefinitionList(t *testing.T) {
	t.Parallel()
	cases := testCases(
		`column_a`, "CommaList{ColDef{ColName}}",
		`column_a, column_b`, "CommaList{ColDef{ColName} T ColDef{ColName}}",
		`, column_b`, "CommaList{!ErrorMissing T ColDef{ColName}}",
		`column_a, , column_b`, "CommaList{ColDef{ColName} T !ErrorMissing T ColDef{ColName}}",
	)

	runTests(t, cases, (*Parser).columnDefinitionList)
}

func TestTableConstraint(t *testing.T) {
	t.Parallel()
	cases := testCases(
		`CONSTRAINT pk PRIMARY KEY (column_name)`,
		"TableConstraint{T ConstraintName PrimaryKeyTableConstraint{TTT CommaList{IndexedColumn{ColName}} T}}",
		`UNIQUE (column_name)`,
		"TableConstraint{UniqueTableConstraint{TT CommaList{IndexedColumn{ColName}} T}}",
		`CHECK (column_name)`,
		"TableConstraint{CheckTableConstraint{TT E{ColRef{ColName}} T}}",
		`FOREIGN KEY (column_name) REFERENCES table_name(column_name)`,
		`TableConstraint{ForeignKeyTableConstraint{TTT CommaList{ColName} T
			ForeignKeyClause{T TableName T CommaList{ColName} T}}}`,
		`CONSTRAINT PRIMARY KEY (column_name)`,
		"TableConstraint{T !ErrorMissing PrimaryKeyTableConstraint{TTT CommaList{IndexedColumn{ColName}} T}}",
		`CONSTRAINT pk CREATE`,
		"TableConstraint{T ConstraintName !ErrorExpecting}",
	)

	runTests(t, cases, (*Parser).tableConstraint)
}

func TestPrimaryKeyTableConstraint(t *testing.T) {
	t.Parallel()
	cases := testCases(
		`PRIMARY KEY (column_name) ON CONFLICT ROLLBACK`,
		"PrimaryKeyTableConstraint{TT T CommaList{IndexedColumn{ColName}} T ConflictClause{TTT}}",
		`PRIMARY (column_name)`,
		"PrimaryKeyTableConstraint{T !ErrorMissing T CommaList{IndexedColumn{ColName}} T}",
		`PRIMARY KEY column_name)`,
		"PrimaryKeyTableConstraint{TT !ErrorMissing CommaList{IndexedColumn{ColName}} T}",
		`PRIMARY KEY (column_name`,
		"PrimaryKeyTableConstraint{TT T CommaList{IndexedColumn{ColName}} !ErrorMissing}",
	)

	runTests(t, cases, (*Parser).primaryKeyTableConstraint)
}

func TestUniqueTableConstraint(t *testing.T) {
	t.Parallel()
	cases := testCases(
		`UNIQUE (column_name) ON CONFLICT ROLLBACK`,
		"UniqueTableConstraint{T T CommaList{IndexedColumn{ColName}} T ConflictClause{TTT}}",
		`UNIQUE column_name)`,
		"UniqueTableConstraint{T !ErrorMissing CommaList{IndexedColumn{ColName}} T}",
		`UNIQUE (column_name`,
		"UniqueTableConstraint{T T CommaList{IndexedColumn{ColName}} !ErrorMissing}",
	)

	runTests(t, cases, (*Parser).uniqueTableConstraint)
}

func TestCheckTableConstraint(t *testing.T) {
	t.Parallel()
	cases := testCases(
		`CHECK (TRUE)`, "CheckTableConstraint{T T E{T} T}",
		`CHECK TRUE)`, "CheckTableConstraint{T !ErrorMissing E{T} T}",
		`CHECK ()`, "CheckTableConstraint{T T !ErrorMissing T}",
		`CHECK (TRUE`, "CheckTableConstraint{T T E{T} !ErrorMissing }",
	)

	runTests(t, cases, (*Parser).checkTableConstraint)
}

func TestForeignKeyTableConstraint(t *testing.T) {
	t.Parallel()
	cases := testCases(
		`FOREIGN KEY (column_name) REFERENCES table_name`,
		"ForeignKeyTableConstraint{TT T CommaList{ColName} T ForeignKeyClause{T TableName}}",
		`FOREIGN (column_name) REFERENCES table_name`,
		"ForeignKeyTableConstraint{T !ErrorMissing T CommaList{ColName} T ForeignKeyClause{T TableName}}",
		`FOREIGN KEY column_name) REFERENCES table_name`,
		"ForeignKeyTableConstraint{TT !ErrorMissing CommaList{ColName} T ForeignKeyClause{T TableName}}",
		`FOREIGN KEY () REFERENCES table_name`,
		"ForeignKeyTableConstraint{TT T !ErrorMissing T ForeignKeyClause{T TableName}}",
		`FOREIGN KEY (column_name REFERENCES table_name`,
		"ForeignKeyTableConstraint{TT T CommaList{ColName} !ErrorMissing ForeignKeyClause{T TableName}}",
		`FOREIGN KEY (column_name)`,
		"ForeignKeyTableConstraint{TT T CommaList{ColName} T !ErrorMissing}",
	)

	runTests(t, cases, (*Parser).foreignKeyTableConstraint)
}

func TestTableOptions(t *testing.T) {
	t.Parallel()
	cases := testCases(
		`WITHOUT ROWID`,
		"CommaList{TableOption{TT}}",
		`STRICT, WITHOUT ROWID`,
		"CommaList{TableOption{T} T TableOption{TT}}}",
	)

	runTests(t, cases, (*Parser).tableOptions)
}

func TestTableOption(t *testing.T) {
	t.Parallel()
	cases := testCases(
		`WITHOUT ROWID`,
		"TableOption{TT}",
		`STRICT`,
		"TableOption{T}",
		`WITHOUT`,
		"TableOption{T !ErrorMissing}",
	)

	runTests(t, cases, (*Parser).tableOption)
}

func TestColumnNameList(t *testing.T) {
	t.Parallel()
	cases := testCases(
		`column_name_1, column_name_2`,
		"CommaList{ColName T ColName}",
		`column_name_1 column_name_2`,
		"CommaList{ColName !ErrorMissing ColName}",
	)

	fn := func(p *Parser) parsetree.NonTerminal {
		return p.columnNameList(token.KindRightParen, token.KindSemicolon, token.KindEOF)
	}

	runTests(t, cases, fn)
}

func TestCreateTrigger(t *testing.T) {
	t.Parallel()
	cases := testCases(
		`CREATE TRIGGER trigger_name DELETE ON table_name BEGIN DELETE 10; END`,
		"CreateTrigger{TT TriggerName TT TableName TriggerBody{T Delete {T E{T}} TT}}",
		`CREATE TEMP TRIGGER IF NOT EXISTS trigger_name BEFORE DELETE ON table_name BEGIN INSERT 10; END`,
		"CreateTrigger{TTT TTT TriggerName TTT TableName TriggerBody{T Insert {T E{T}} TT}}",
		`CREATE TEMPORARY TRIGGER schema_name.trigger_name AFTER INSERT ON table_name FOR EACH ROW BEGIN SELECT 10; END`,
		"CreateTrigger{TTT SchemaName T TriggerName TTT TableName TTT TriggerBody{T Select {T E{T}} TT}}",
		`CREATE TRIGGER trigger_name INSTEAD OF UPDATE ON table_name WHEN a > 10 BEGIN UPDATE 10; END`,
		"CreateTrigger{TT TriggerName TTTT TableName T E{GreaterThan{ColRef{ColName} T T}} TriggerBody{T Update {T E{T}} TT}}",
		`CREATE TRIGGER trigger_name UPDATE OF a, b ON table_name BEGIN SELECT 10; END`,
		"CreateTrigger{TT TriggerName TT CommaList{ColName T ColName} T TableName TriggerBody{T Select {T E{T}} TT}}",
		`CREATE TRIGGER trigger_name DELETE table_name BEGIN SELECT 10; END`,
		"CreateTrigger{TT TriggerName T !ErrorMissing TableName TriggerBody{T Select {T E{T}} TT}}",
		`CREATE TRIGGER trigger_name DELETE ON BEGIN SELECT 10; END`,
		"CreateTrigger{TT TriggerName TT !ErrorMissing TriggerBody{T Select {T E{T}} TT}}",
		`CREATE TRIGGER trigger_name DELETE ON table_name BEGIN SELECT 10 END`,
		"CreateTrigger{TT TriggerName TT TableName TriggerBody{T Select {T E{T}} !ErrorMissing T}}",
		`CREATE TRIGGER trigger_name DELETE ON table_name BEGIN SELECT 10; `,
		"CreateTrigger{TT TriggerName TT TableName TriggerBody{T Select {T E{T}} T !ErrorMissing}}",
		`CREATE TRIGGER trigger_name DELETE ON table_name BEGIN ; END`,
		"CreateTrigger{TT TriggerName TT TableName TriggerBody{T !ErrorExpecting TT}}",
		`CREATE TEMP TRIGGER IF EXISTS trigger_name BEFORE DELETE ON table_name BEGIN SELECT 10; END`,
		"CreateTrigger{TTT T !ErrorMissing T TriggerName TTT TableName TriggerBody{T Select {T E{T}} TT}}",
		`CREATE TEMP TRIGGER IF NOT trigger_name BEFORE DELETE ON table_name BEGIN SELECT 10; END`,
		"CreateTrigger{TTT TT !ErrorMissing TriggerName TTT TableName TriggerBody{T Select {T E{T}} TT}}",
		`CREATE TEMP TRIGGER IF NOT EXISTS trigger_name BEFORE ON table_name BEGIN SELECT 10; END`,
		"CreateTrigger{TTT TTT TriggerName T !ErrorExpecting T TableName TriggerBody{T Select {T E{T}} TT}}",
		`CREATE TEMPORARY TRIGGER .trigger_name AFTER INSERT ON table_name FOR EACH ROW BEGIN SELECT 10; END`,
		"CreateTrigger{TTT !ErrorMissing T TriggerName TTT TableName TTT TriggerBody{T Select {T E{T}} TT}}",
		`CREATE TEMPORARY TRIGGER schema_name trigger_name AFTER INSERT ON table_name FOR EACH ROW BEGIN SELECT 10; END`,
		"CreateTrigger{TTT SchemaName !ErrorMissing TriggerName TTT TableName TTT TriggerBody{T Select {T E{T}} TT}}",
		`CREATE TEMPORARY TRIGGER schema_name.trigger_name AFTER INSERT ON table_name FOR ROW BEGIN SELECT 10; END`,
		"CreateTrigger{TTT SchemaName T TriggerName TTT TableName T !ErrorMissing T TriggerBody{T Select {T E{T}} TT}}",
		`CREATE TEMPORARY TRIGGER schema_name.trigger_name AFTER INSERT ON table_name FOR EACH BEGIN SELECT 10; END`,
		"CreateTrigger{TTT SchemaName T TriggerName TTT TableName TT !ErrorMissing TriggerBody{T Select {T E{T}} TT}}",
		`CREATE TRIGGER trigger_name UPDATE OF a b ON table_name BEGIN SELECT 10; END`,
		"CreateTrigger{TT TriggerName TT CommaList{ColName !ErrorMissing ColName} T TableName TriggerBody{T Select {T E{T}} TT}}",
		`CREATE TRIGGER trigger_name INSTEAD OF UPDATE ON table_name WHEN BEGIN SELECT 10; END`,
		"CreateTrigger{TT TriggerName TTTT TableName T !ErrorMissing TriggerBody{T Select {T E{T}} TT}}",
		`CREATE TRIGGER trigger_name INSTEAD UPDATE ON table_name BEGIN SELECT 10; END`,
		"CreateTrigger{TT TriggerName T !ErrorMissing TT TableName TriggerBody{T Select {T E{T}} TT}}",
		`CREATE TRIGGER INSTEAD OF UPDATE ON table_name BEGIN SELECT 10; END`,
		"CreateTrigger{TT !ErrorMissing TTTT TableName TriggerBody{T Select {T E{T}} TT}}",
		`CREATE TRIGGER trigger_name DELETE ON table_name`,
		"CreateTrigger{TT TriggerName TT TableName !ErrorMissing}",
	)

	runTests(t, cases, (*Parser).createTrigger)
}

func TestExpression(t *testing.T) {
	t.Parallel()
	cases := testCases(
		`TRUE OR TRUE`, "E{Or{TTT}}",
	)

	runTests(t, cases, (*Parser).expression)
}

func TestExpression1(t *testing.T) {
	t.Parallel()
	cases := testCases(
		`TRUE OR TRUE`, "Or{TTT}",
		`TRUE OR`, "Or{TT !ErrorMissing}",
	)

	runTests(t, cases, (*Parser).expression1)
}

func TestExpression2(t *testing.T) {
	t.Parallel()
	cases := testCases(
		`TRUE AND TRUE`, "And{TTT}",
		`TRUE AND`, "And{TT !ErrorMissing}",
	)

	runTests(t, cases, (*Parser).expression2)
}

func TestExpression3(t *testing.T) {
	t.Parallel()
	cases := testCases(
		`NOT TRUE`, "Not{TT}",
		`NOT`, "Not{T !ErrorMissing}",
	)

	runTests(t, cases, (*Parser).expression3)
}

func TestExpression4(t *testing.T) {
	t.Parallel()
	cases := testCases(
		`a = c`, "Equal{ColRef{ColName} T ColRef{ColName}}",
		`a == c`, "Equal{ColRef{ColName} T ColRef{ColName}}",
		`a <> c`, "NotEqual{ColRef{ColName} T ColRef{ColName}}",
		`a != c`, "NotEqual{ColRef{ColName} T ColRef{ColName}}",
		`10 REGEXP '10'`, "Regexp{TTT}",
		`'10' NOT REGEXP '20'`, "NotRegexp{TTTT}",
		`'10' LIKE '20'`, "Like{TTT}",
		`10 LIKE '10' ESCAPE '!'`, "Like{TTTTT}",
		`'10' NOT LIKE '20' ESCAPE '!'`, "NotLike{TTTTTT}",
		`'10' GLOB '20'`, "Glob{TTT}",
		`10 NOT GLOB '10'`, "NotGlob{TTTT}",
		`'10' MATCH '20'`, "Match{TTT}",
		`10 NOT MATCH '10'`, "NotMatch{TTTT}",
		`10 IS '10'`, "Is{TTT}",
		`10 BETWEEN 5 AND 15`, "Between{TTTTT}",
		`10 NOT BETWEEN 5 AND 15`, "NotBetween{TTTTTT}",
		`10 IN (5)`, "In{TTT CommaList{T} T}",
		`10 NOT IN (5)`, "NotIn{TTTT CommaList{T} T}",
		`10 ISNULL`, "Isnull{TT}",
		`10 NOTNULL`, "Notnull{TT}",
		`10 NOT NULL`, "NotNull{TTT}",
		`10 ==`, "Equal{TT !ErrorMissing}",
		`10 <>`, "NotEqual{TT !ErrorMissing}",
		`10 GLOB`, "Glob{TT !ErrorMissing}",
		`10 REGEXP`, "Regexp{TT !ErrorMissing}",
		`10 MATCH`, "Match{TT !ErrorMissing}",
		`10 LIKE ESCAPE`, "Like{TT !ErrorMissing T !ErrorMissing}",
		`10 NOT GLOB`, "NotGlob{TTT !ErrorMissing}",
		`10 NOT REGEXP`, "NotRegexp{TTT !ErrorMissing}",
		`10 NOT MATCH`, "NotMatch{TTT !ErrorMissing}",
		`10 NOT LIKE ESCAPE`, "NotLike{TTT !ErrorMissing T !ErrorMissing}",
		`10 IN table_function_name(10`, "In{TT TableFunctionName T CommaList{T} !ErrorMissing}",
		`10 NOT IN table_function_name(10`, "NotIn{TTT TableFunctionName T CommaList{T} !ErrorMissing}",
	)

	runTests(t, cases, (*Parser).expression4)
}

func TestIsExpression(t *testing.T) {
	t.Parallel()
	cases := testCases(
		`10 IS 10`,
		"Is{TTT}",
		`10 IS DISTINCT FROM 10`,
		"IsDistinctFrom{TTTTT}",
		`10 IS NOT 10`,
		"IsNot{TTTT}",
		`10 IS NOT DISTINCT FROM 10`,
		"IsNotDistinctFrom{TTTTTT}",
		`10 IS NOT DISTINCT 10`, "IsNotDistinctFrom{TTTT !ErrorMissing T}",
		`10 IS NOT DISTINCT FROM`, "IsNotDistinctFrom{TTTTT !ErrorMissing}",
		`10 IS NOT`, "IsNot{TTT !ErrorExpecting}",
		`10 IS DISTINCT FROM`, "IsDistinctFrom{TTTT !ErrorMissing}",
		`10 IS DISTINCT 10`, "IsDistinctFrom{TTT !ErrorMissing T}",
		`10 IS`, "Is{TT !ErrorExpecting}",
	)

	fn := func(p *Parser) parsetree.NonTerminal {
		p.advance() // skip the 10
		exp := parsetree.NewTerminal(parsetree.KindToken, token.New([]byte("10"), token.KindNumeric))
		return p.isExpression(exp)
	}

	runTests(t, cases, fn)
}

func TestBetween(t *testing.T) {
	t.Parallel()
	cases := testCases(
		`10 BETWEEN 5 AND 15`,
		`Between{TTTTT}`,
		`10 BETWEEN`, "Between{TT !ErrorMissing}",
		`10 BETWEEN AND 20`, "Between{TT !ErrorMissing TT}",
		`10 BETWEEN 10  20`, "Between{TTT !ErrorMissing T}",
	)

	fn := func(p *Parser) parsetree.NonTerminal {
		p.advance() // skip the 10
		exp := parsetree.NewTerminal(parsetree.KindToken, token.New([]byte("10"), token.KindNumeric))
		return p.between(exp)
	}

	runTests(t, cases, fn)
}

func TestNotBetween(t *testing.T) {
	t.Parallel()
	cases := testCases(
		`10 NOT BETWEEN 5 AND 15`,
		"NotBetween{TTTTTT}",
		`10 NOT BETWEEN`, "NotBetween{TTT !ErrorMissing}",
		`10 NOT BETWEEN AND 20`, "NotBetween{TTT !ErrorMissing TT}",
		`10 NOT BETWEEN 10  20`, "NotBetween{TTTT !ErrorMissing T}",
	)

	fn := func(p *Parser) parsetree.NonTerminal {
		p.advance() // skip the 10
		exp := parsetree.NewTerminal(parsetree.KindToken, token.New([]byte("10"), token.KindNumeric))
		return p.notBetween(exp)
	}

	runTests(t, cases, fn)
}

func TestIn(t *testing.T) {
	t.Parallel()
	cases := testCases(
		`10 IN (10, 20)`, "In{TTT CommaList{TTT} T}",
		`10 IN ()`, "In{TTTT}",
		`10 IN (SELECT)`, "In{TTT Select{T} T}",
		`10 IN schema_name.table_name`, "In{TT SchemaName T TableName}",
		`10 IN schema_a.table_function_a(10, 20)`, "In{TT SchemaName T TableFunctionName T CommaList{TTT} T }",
		`10 IN (ALTER)`, "In{TTT !ErrorExpecting Skipped{T} T}",
		`10 IN (1`, "In{TTT CommaList{T} !ErrorMissing}",
		`10 IN function()`, "In{TT TableFunctionName T !ErrorMissing T}",
		`10 IN function(10`, "In{TT TableFunctionName T CommaList{T} !ErrorMissing}",
		`10 IN ALTER`, `In{TT !ErrorExpecting}`,
	)

	fn := func(p *Parser) parsetree.NonTerminal {
		p.advance() // skip the 10
		exp := parsetree.NewTerminal(parsetree.KindToken, token.New([]byte("10"), token.KindNumeric))
		return p.in(exp)
	}

	runTests(t, cases, fn)
}

func TestNotIn(t *testing.T) {
	t.Parallel()
	cases := testCases(
		`10 NOT IN (10, 20)`, "NotIn{TTTT CommaList{TTT} T}",
		`10 NOT IN ()`, "NotIn{TTTTT}",
		`10 NOT IN schema_name.table_name`, "NotIn{TTT SchemaName T TableName}",
		`10 NOT IN schema_a.table_function_a(10, 20)`, "NotIn{TTT SchemaName T TableFunctionName T CommaList{TTT} T }",
		`10 NOT IN table_a`, "NotIn{TTT TableName}",
		`10 NOT IN (SELECT)`, "NotIn{TTTT Select{T} T}", `10 NOT IN (ALTER)`, "NotIn{TTTT !ErrorExpecting Skipped{T} T}",
		`10 NOT IN (1`, `NotIn{TTTT CommaList{T} !ErrorMissing}`,
		`10 NOT IN function()`, "NotIn{TTT TableFunctionName T !ErrorMissing T}",
		`10 NOT IN function(10`, "NotIn{TTT TableFunctionName T CommaList{T} !ErrorMissing}",
		`10 NOT IN ALTER`, `NotIn{TTT !ErrorExpecting}`,
	)

	fn := func(p *Parser) parsetree.NonTerminal {
		p.advance() // skip the 10
		exp := parsetree.NewTerminal(parsetree.KindToken, token.New([]byte("10"), token.KindNumeric))
		return p.notIn(exp)
	}

	runTests(t, cases, fn)
}

func TestIsStartOfExpressionAtLeast4(t *testing.T) {
	t.Parallel()
	cases := []struct {
		tok    *token.Token
		result bool
	}{
		{tok: token.New([]byte("10"), token.KindNumeric), result: true},
		{tok: token.New([]byte("?"), token.KindQuestionVariable), result: true},
		{tok: token.New([]byte("a"), token.KindIdentifier), result: true},
		{tok: token.New([]byte("NOT"), token.KindNot), result: false},
	}

	p := New(lexer.New(nil))
	for _, c := range cases {
		if p.isStartOfExpressionAtLeast4(c.tok) != c.result {
			t.Logf("isStartOfExpressionAtLeast4(%s) == %v, want %v", c.tok, !c.result, c.result)
			t.Fail()
		}
	}
}

func TestExpression5(t *testing.T) {
	t.Parallel()
	cases := testCases(
		`a < c`, "LessThan{ColRef{ColName} T ColRef{ColName}}",
		`a <= c`, "LessThanOrEqual{ColRef{ColName} T ColRef{ColName}}",
		`a > c`, "GreaterThan{ColRef{ColName} T ColRef{ColName}}",
		`a >= c`, "GreaterThanOrEqual{ColRef{ColName} T ColRef{ColName}}",
		`10 <`, "LessThan{TT !ErrorMissing}",
	)

	runTests(t, cases, (*Parser).expression5)
}

func TestExpression6(t *testing.T) {
	t.Parallel()
	cases := testCases(
		`a & c`, "BitAnd{ColRef{ColName} T ColRef{ColName}}",
		`a | c`, "BitOr{ColRef{ColName} T ColRef{ColName}}",
		`a << c`, "LeftShift{ColRef{ColName} T ColRef{ColName}}",
		`a >> c`, "RightShift{ColRef{ColName} T ColRef{ColName}}",
		`10 &`, "BitAnd{TT !ErrorMissing}",
	)

	runTests(t, cases, (*Parser).expression6)
}

func TestExpression7(t *testing.T) {
	t.Parallel()
	cases := testCases(
		`a + c`, "Add{ColRef{ColName} T ColRef{ColName}}",
		`a - c`, "Subtract{ColRef{ColName} T ColRef{ColName}}",
		`10 +`, "Add{TT !ErrorMissing}",
	)

	runTests(t, cases, (*Parser).expression7)
}

func TestExpression8(t *testing.T) {
	t.Parallel()
	cases := testCases(
		`a * c`, "Multiply{ColRef{ColName} T ColRef{ColName}}",
		`a / c`, "Divide{ColRef{ColName} T ColRef{ColName}}",
		`a % c`, "Mod{ColRef{ColName} T ColRef{ColName}}",
		`10 *`, "Multiply{TT !ErrorMissing}",
	)

	runTests(t, cases, (*Parser).expression8)
}

func TestExpression9(t *testing.T) {
	t.Parallel()
	cases := testCases(
		`a || c`, "Concatenate{ColRef{ColName} T ColRef{ColName}}",
		`a -> c`, "Extract1{ColRef{ColName} T ColRef{ColName}}",
		`a ->> c`, "Extract2{ColRef{ColName} T ColRef{ColName}}",
		`10 ||`, "Concatenate{TT !ErrorMissing}",
	)

	runTests(t, cases, (*Parser).expression9)
}

func TestExpression10(t *testing.T) {
	t.Parallel()
	cases := testCases(
		`a`, "ColRef{ColName}",
		`a COLLATE c`, "Collate{ColRef{ColName} T CollationName}",
		`10 COLLATE`, "Collate{TT !ErrorMissing}",
	)

	runTests(t, cases, (*Parser).expression10)
}

func TestExpression11(t *testing.T) {
	t.Parallel()
	cases := testCases(
		`~a`, "BitNot{T ColRef{ColName}}",
		`+a`, "PrefixPlus{T ColRef{ColName}}",
		`-a`, "Negate{T ColRef{ColName}}",
		`~`, "BitNot{T !ErrorMissing}",
	)

	runTests(t, cases, (*Parser).expression11)
}

func TestSimpleExpression(t *testing.T) {
	t.Parallel()
	cases := testCases(
		`a`, "ColRef{ColName}",
		`?`, "BindParameter",
		`function_name()`, "FunctionCall{FunctionName TT}",
		`(10)`, "ParenE{T CommaList{E{T}} T}",
		`CAST ('10' AS INTEGER)`, "Cast{TT E{T} T TypeName{T} T}",
		`NOT EXISTS (SELECT 10)`, "Not{T Exists{T T Select{T E{T}} T}}",
		`EXISTS (SELECT 10)`, "Exists{T T Select{T E{T}} T}",
		`CASE WHEN TRUE THEN 10 END`, "Case{T When{T E{T} T E{T}} T}",
		`RAISE (IGNORE)`, "Raise{TTTT}",
	)

	runTests(t, cases, (*Parser).simpleExpression)
}

func TestIsStartOfExpression(t *testing.T) {
	t.Parallel()
	cases := []struct {
		tok    *token.Token
		result bool
	}{
		{tok: token.New([]byte("10"), token.KindNumeric), result: true},
		{tok: token.New([]byte("?"), token.KindQuestionVariable), result: true},
		{tok: token.New([]byte("a"), token.KindIdentifier), result: true},
		{tok: token.New([]byte("SELECT"), token.KindSelect), result: false},
	}

	p := New(lexer.New(nil))
	for _, c := range cases {
		if p.isStartOfExpressionAtLeast4(c.tok) != c.result {
			t.Logf("isStartOfExpression(%s) == %v, want %v", c.tok, !c.result, c.result)
			t.Fail()
		}
	}
}

func TestIsLiteralValue(t *testing.T) {
	t.Parallel()
	cases := []struct {
		tok    *token.Token
		result bool
	}{
		{tok: token.New([]byte("NULL"), token.KindNull), result: true},
		{tok: token.New([]byte("CURRENT_TIME"), token.KindCurrentTime), result: true},
		{tok: token.New([]byte("TRUE"), token.KindIdentifier), result: true},
		{tok: token.New([]byte("SELECT"), token.KindSelect), result: false},
	}

	p := New(lexer.New(nil))
	for _, c := range cases {
		if p.isLiteralValue(c.tok) != c.result {
			t.Logf("isLiteralValue(%s) == %v, want %v", c.tok, !c.result, c.result)
			t.Fail()
		}
	}
}

func TestIsBindParameter(t *testing.T) {
	t.Parallel()
	cases := []struct {
		tok    *token.Token
		result bool
	}{
		{tok: token.New([]byte("@a"), token.KindAtVariable), result: true},
		{tok: token.New([]byte(":a"), token.KindColonVariable), result: true},
		{tok: token.New([]byte("$1"), token.KindDollarVariable), result: true},
		{tok: token.New([]byte("SELECT"), token.KindSelect), result: false},
	}

	p := New(lexer.New(nil))
	for _, c := range cases {
		if p.isBindParameter(c.tok) != c.result {
			t.Logf("isBindParameter(%s) == %v, want %v", c.tok, !c.result, c.result)
			t.Fail()
		}
	}
}

func TestColumnReference(t *testing.T) {
	t.Parallel()
	cases := testCases(
		`a`, "ColRef{ColName}",
		`a.a`, "ColRef{TableName T ColName}",
		`a.a.a`, "ColRef{SchemaName T TableName T ColName}",
	)

	runTests(t, cases, (*Parser).columnReference)
}

func TestFunctionCall(t *testing.T) {
	t.Parallel()
	cases := testCases(
		`func()`,
		"FnCall{FnName TT}",
		`func('a')`,
		"FnCall{FnName T FnArgs{CommaList{E{T}}} T}",
		`func('a') OVER window_name`,
		"FnCall{FnName T FnArgs{CommaList{E{T}}} T OverClause{T WindowName}}",
		`function(;`,
		"FnCall{FnName T !ErrorMissing}",
		`function(;`,
		"FnCall{FnName T !ErrorMissing}",
		`function(DISTINCT );`,
		"FnCall{FnName T FnArgs{T !ErrorMissing} T}",
		`function() FILTER WHERE a);`,
		"FnCall{FnName TT FilterClause{T !ErrorMissing T E{ColRef{ColName}} T}}",
		`function() FILTER (a);`,
		"FnCall{FnName TT FilterClause{TT !ErrorMissing E{ColRef{ColName}} T}}",
		`function(10 ORDER a);`,
		"FnCall{FnName T FnArgs{CommaList{E{T}} OrderBy{T !ErrorMissing CommaList{OrderingTerm{E{ColRef{ColName}}}}}} T}",
		`function(10 ORDER BY);`,
		"FnCall{FnName T FnArgs{CommaList{E{T}} OrderBy{TT !ErrorMissing}} T}",
		`function(10 ORDER BY a COLLATE);`,
		"FnCall{FnName T FnArgs{CommaList{E{T}} OrderBy{TT CommaList{ OrderingTerm{E{Collate{ColRef{ColName} T !ErrorMissing}}}} }} T}",
		`function(10 ORDER BY a NULLS);`,
		"FnCall{FnName T FnArgs{CommaList{E{T}} OrderBy{TT CommaList{OrderingTerm{E{ColRef{ColName}} T !ErrorExpecting}} }} T}",
	)

	runTests(t, cases, (*Parser).functionCall)
}

func TestFunctionArguments(t *testing.T) {
	t.Parallel()
	cases := testCases(
		`'a'`, "FnArgs{CommaList{E{T}}}",
		`DISTINCT 'a'`, "FnArgs{T CommaList{E{T}}}",
		`'a' ORDER BY a`, "FnArgs{CommaList{E{T}} OrderBy{TT CommaList{OrderingTerm{E{ColRef{ColName}}}}}}",
		`*`, "FnArgs{T}",
		``, "FnArgs{!ErrorMissing}",
	)

	runTests(t, cases, (*Parser).functionArguments)
}

func TestOrderBy(t *testing.T) {
	t.Parallel()
	cases := testCases(
		`ORDER BY a COLLATE c`,
		"OrderBy{TT CommaList{OrderingTerm{E{Collate{ColRef{ColName} T CollationName}}}}}",
		`ORDER a`, "OrderBy{T !ErrorMissing CommaList{OrderingTerm{E{ColRef{ColName}}}}}",
		`ORDER BY`, "OrderBy{TT !ErrorMissing}",
	)

	fn := func(p *Parser) parsetree.NonTerminal {
		return p.orderBy(func(t *token.Token) bool {
			return t.Kind == token.KindEOF
		})
	}

	runTests(t, cases, fn)
}

func TestOrderingTerm(t *testing.T) {
	t.Parallel()
	cases := testCases(
		`a ASC NULLS LAST`,
		"OrderingTerm{E{ColRef{ColName}} TTT}",
		`a DESC NULLS FIRST`,
		"OrderingTerm{E{ColRef{ColName}} TTT}",
		`a DESC NULLS`,
		"OrderingTerm{E{ColRef{ColName}} TT !ErrorExpecting}",
	)

	runTests(t, cases, (*Parser).orderingTerm)
}

func TestFilterClause(t *testing.T) {
	t.Parallel()
	cases := testCases(
		`FILTER (WHERE a + b)`,
		"FilterClause{TT T E{Add{ColRef{ColName} T ColRef{ColName}}} T}",
		`FILTER WHERE a)`,
		"FilterClause{T !ErrorMissing T E{ColRef{ColName}} T}",
		`FILTER (a)`,
		"FilterClause{TT !ErrorMissing E{ColRef{ColName}} T}",
		`FILTER (WHERE)`,
		"FilterClause{TTT !ErrorMissing T}",
		`FILTER (WHERE a`,
		"FilterClause{TT T E{ColRef{ColName}} !ErrorMissing}",
	)

	runTests(t, cases, (*Parser).filterClause)
}

func TestOverClause(t *testing.T) {
	t.Parallel()
	cases := testCases(
		`OVER window_a`,
		"OverClause{T WindowName}",
		`OVER ()`,
		"OverClause{TTT}",
		`OVER (window_a PARTITION BY a, b));`,
		"OverClause{TT WindowName PartitionBy{TT CommaList{E{ColRef{ColName}} T E{ColRef{ColName}}}} T}",
		`OVER (ORDER BY a)`,
		"OverClause{TT OrderBy{TT CommaList{OrderingTerm{E{ColRef{ColName}}}}} T}",
		`OVER (RANGE CURRENT ROW)`,
		"OverClause{TT FrameSpec{TTT} T}",
		`OVER ORDER BY a)`,
		"OverClause{T !ErrorExpecting OrderBy{TT CommaList{OrderingTerm{E{ColRef{ColName}}}}} T}",
		`OVER (window_a PARTITION a));`,
		"OverClause{TT WindowName PartitionBy{T !ErrorMissing CommaList{E{ColRef{ColName}}}} T}",
		`OVER (window_a PARTITION BY));`,
		"OverClause{TT WindowName PartitionBy{TT !ErrorMissing} T}",
		`OVER (RANGE CURRENT ROW`,
		"OverClause{TT FrameSpec{TTT} !ErrorMissing}",
		`OVER`,
		"OverClause{T !ErrorExpecting}",
	)

	runTests(t, cases, (*Parser).overClause)
}

func TestFrameSpec(t *testing.T) {
	t.Parallel()
	cases := testCases(
		`ROWS UNBOUNDED PRECEDING EXCLUDE NO OTHERS`,
		"FrameSpec{TTTTTT}",
		`GROUPS 10 PRECEDING EXCLUDE CURRENT ROW`,
		"FrameSpec{T E{T} TTTT}",
		`RANGE CURRENT ROW EXCLUDE GROUP`,
		"FrameSpec{TTTTT}",
		`ROWS BETWEEN 10 PRECEDING AND 10 FOLLOWING EXCLUDE TIES`,
		"FrameSpec{T FrameSpecBetween{T E{T} TT E{T} T} TT}",
		`ROWS UNBOUNDED EXCLUDE NO OTHERS`,
		"FrameSpec{TT !ErrorMissing TTT}",
		`RANGE CURRENT EXCLUDE GROUP`,
		"FrameSpec{TT !ErrorMissing TT}",
		`GROUPS 10 EXCLUDE CURRENT ROW`,
		"FrameSpec{T E{T} !ErrorMissing TTT}",
		`ROWS`,
		"FrameSpec{T !ErrorExpecting}",
		`ROWS UNBOUNDED PRECEDING EXCLUDE NO`,
		"FrameSpec{TTTTT !ErrorMissing}",
		`GROUPS 10 PRECEDING EXCLUDE CURRENT`,
		"FrameSpec{T E{T} TTT !ErrorMissing}",
		`ROWS UNBOUNDED PRECEDING EXCLUDE`,
		"FrameSpec{TTTT !ErrorExpecting}",
	)

	runTests(t, cases, (*Parser).frameSpec)
}

func TestFrameSpecBetween(t *testing.T) {
	t.Parallel()
	cases := testCases(
		`BETWEEN UNBOUNDED PRECEDING AND 10 PRECEDING`,
		"FrameSpecBetween{TTTT E{T} T}",
		`BETWEEN 10 PRECEDING AND CURRENT ROW`,
		"FrameSpecBetween{T E{T} TTTT}",
		`BETWEEN CURRENT ROW AND 10 FOLLOWING`,
		"FrameSpecBetween{TTTT E{T} T}",
		`BETWEEN 10 FOLLOWING AND UNBOUNDED FOLLOWING`,
		"FrameSpecBetween{T E{T} TTTT}",
		`BETWEEN UNBOUNDED AND 10 PRECEDING`,
		"FrameSpecBetween{TT !ErrorMissing T E{T} T}",
		`BETWEEN CURRENT AND 10 FOLLOWING`,
		"FrameSpecBetween{TT !ErrorMissing T E{T} T}",
		`BETWEEN 10 AND CURRENT ROW`,
		"FrameSpecBetween{T E{T} !ErrorExpecting TTT}",
		`BETWEEN PRECEDING AND 10 PRECEDING`,
		"FrameSpecBetween{T !ErrorExpecting TT E{T} T}",
		`BETWEEN ROW AND 10 FOLLOWING`,
		"FrameSpecBetween{T !ErrorMissing TT E{T} T}",
		`BETWEEN FOLLOWING AND UNBOUNDED FOLLOWING`,
		"FrameSpecBetween{T !ErrorMissing TTTT}",
		`BETWEEN AND 10 PRECEDING`,
		"FrameSpecBetween{T !ErrorExpecting T E{T} T}",
		`BETWEEN UNBOUNDED PRECEDING 10 PRECEDING`,
		"FrameSpecBetween{TTT !ErrorMissing E{T} T}",
		`BETWEEN CURRENT ROW AND UNBOUNDED`,
		"FrameSpecBetween{TTTTT !ErrorMissing}",
		`BETWEEN 10 PRECEDING AND CURRENT`,
		"FrameSpecBetween{T E{T} TTT !ErrorMissing}",
		`BETWEEN UNBOUNDED PRECEDING AND 10`,
		"FrameSpecBetween{TTTT E{T} !ErrorExpecting}",
		`BETWEEN UNBOUNDED PRECEDING AND`,
		"FrameSpecBetween{TTT T !ErrorExpecting}",
	)

	runTests(t, cases, (*Parser).frameSpecBetween)
}

func TestParenExpression(t *testing.T) {
	t.Parallel()
	cases := testCases(
		`(TRUE = FALSE OR 10 == 20)`,
		"ParenE{T CommaList{E{Or{Equal{TTT} T Equal{TTT}}}} T}",
		`(TRUE > FALSE < 10)`,
		"ParenE{T CommaList{E{LessThan{GreaterThan{TTT} TT}}} T}",
		`(10 + column_a)`,
		"ParenExpression{T CommaList{Expression{Add{T T ColumnReference{ColumnName}}}} T}",
		`();`,
		"ParenE{T !ErrorMissing T}",
		`(, 10);`,
		"ParenE{T CommaList{!ErrorMissing T E{T}} T}",
		`(10 10);`,
		"ParenE{T CommaList{E{T} !ErrorMissing E{T}} T}",
		`(10 AS 10);`,
		"ParenE{T CommaList{E{T} Skipped{T} E{T}} T}",
		`(10,,10);`,
		"ParenE{T CommaList{E{T} T !ErrorMissing T E{T}} T}",
		`(10, AS);`,
		"ParenE{T CommaList{E{T} T Skipped{T}} T}",
		`(10;`,
		"ParenE{T CommaList{E{T}} !ErrorMissing}",
	)

	runTests(t, cases, (*Parser).parenExpression)
}

func TestCastExpression(t *testing.T) {
	t.Parallel()
	cases := testCases(
		`CAST ('10' AS INTEGER)`, "Cast{TT E{T} T TypeName{T} T}",
		`CAST 10 AS NUMBER);`,
		"Cast{T !ErrorMissing Expression{T} T TypeName{T} T}",
		`CAST AS NUMBER);`,
		"Cast{T !ErrorMissing !ErrorMissing T TypeName{T} T}",
		`CAST(10 AS);`,
		"Cast{TT Expression{T} T !ErrorMissing T}",
		`CAST(10 NUMBER);`,
		"Cast{TT Expression{T} !ErrorMissing TypeName{T} T}",
		`CAST(10 AS NUMBER;`,
		"Cast{TT Expression{T} T TypeName{T} !ErrorMissing}",
	)

	runTests(t, cases, (*Parser).castExpression)
}

func TestExists(t *testing.T) {
	t.Parallel()
	cases := testCases(
		`EXISTS(SELECT)`, "Exists{TT Select{T} T}",
		`EXISTS SELECT 10);`,
		"Exists{T !ErrorMissing Select{T Expression{T}} T}",
		`EXISTS(10);`,
		"Exists{TT Skipped{T} T}",
		`EXISTS(10;`,
		"Exists{TT Skipped{T}}",
		`EXISTS (SELECT 10;`,
		"Exists{TT Select{T Expression{T}} !ErrorMissing}",
		`EXISTS ();`,
		"Exists{TT !ErrorMissing T}",
	)

	runTests(t, cases, (*Parser).exists)
}

func TestCaseExpression(t *testing.T) {
	t.Parallel()
	cases := testCases(
		`CASE WHEN 10 THEN TRUE ELSE FALSE END`, "Case{T When {T E{T} T E{T}} Else {T E{T}} T}",
		`CASE a WHEN 10 THEN TRUE END`, "Case{T E{ColRef{ColName}} When {T E{T} T E{T}} T}",
		`CASE ELSE 10`,
		"Case{T !ErrorMissing Else{T Expression{T}} !ErrorMissing}",
		`CASE WHEN THEN 10 END`,
		"Case{T When{T !ErrorMissing T Expression{T}} T}",
		`CASE WHEN 10 10 END`,
		"Case{T When{T Expression{T} !ErrorMissing Expression{T}} T}",
		`CASE WHEN 10 THEN END`,
		"Case{T When{T Expression{T} T !ErrorMissing} T}",
		`CASE WHEN 10 THEN 10 ELSE END`,
		"Case{T When{T Expression{T} T Expression{T}} Else{T !ErrorMissing} T}",
	)

	runTests(t, cases, (*Parser).caseExpression)
}

func TestWhen(t *testing.T) {
	t.Parallel()
	cases := testCases(
		`WHEN 10 THEN TRUE`, "When {T E{T} T E{T}}",
		`WHEN 10 THEN 'a' END);`, "When{T E{T} T E{T}}",
		`WHEN 20 THEN 'b'`, "When{T E{T} T E{T}}",
		`WHEN THEN TRUE`, "When {T !ErrorMissing T E{T}}",
		`WHEN 10 TRUE`, "When {T E{T} !ErrorMissing E{T}}",
		`WHEN 10 THEN`, "When {T E{T} T !ErrorMissing}",
	)

	runTests(t, cases, (*Parser).when)
}

func TestCaseElse(t *testing.T) {
	t.Parallel()
	cases := testCases(
		`ELSE FALSE END`, "Else{T E{T}}",
		`ELSE END`, "Else{T !ErrorMissing}",
	)

	runTests(t, cases, (*Parser).caseElse)
}

func TestRaise(t *testing.T) {
	t.Parallel()
	cases := testCases(
		`RAISE (IGNORE)`, "Raise{TTTT}",
		`RAISE (ROLLBACK, 'error message')`, "Raise{TTTT ErrorMessage{E{T}} T}",
		`RAISE;`,
		"Raise{T !ErrorMissing}",
		`RAISE IGNORE);`,
		"Raise{T !ErrorMissing TT}",
		`RAISE(IGNORE;`,
		"Raise{TTT !ErrorMissing}",
		`RAISE();`,
		"Raise{TT !ErrorExpecting T}",
		`RAISE(, 'error');`,
		"Raise{TT !ErrorExpecting T ErrorMessage{Expression{T}} T}",
		`RAISE(ROLLBACK 'error');`,
		"Raise{TTT !ErrorMissing ErrorMessage{Expression{T}} T}",
		`RAISE(ROLLBACK, );`,
		"Raise{TTTT !ErrorMissing T}",
		`RAISE(ROLLBACK, 'error';`,
		"Raise{TTTT ErrorMessage{Expression{T}} !ErrorMissing}",
	)

	runTests(t, cases, (*Parser).raise)
}

// runTests executes tests of the function parseFunc.
func runTests[T parsetree.Construction](t *testing.T, cases []testCase, parseFunc func(*Parser) T) {
	for i, c := range cases {
		c := c
		t.Run(strconv.FormatInt(int64(i), 10), func(t *testing.T) {
			t.Parallel()
			tp := newTestParser(newTestLexer(c.tree))
			expected := tp.tree()

			p := New(lexer.New([]byte(c.code)))
			p.comments = make(map[*token.Token][]*token.Token)
			p.advance()
			p.advance()
			p.advance()

			parsed := parseFunc(p)

			if str, equals := compare(c.code, p.comments, parsed, expected); !equals {
				fmt.Println(c.code)
				fmt.Println(str)
				t.Fail()
			}
		})
	}
}

// compare compares parsed with expected. If they are not equal it returns false.
func compare(code string, comments map[*token.Token][]*token.Token, parsed, expected parsetree.Construction) (string, bool) {
	c := newComparator(code, comments)
	equals := c.compare(parsed, expected)
	return c.log(), equals
}

// comparator deals with the comparation of parse trees.
type comparator struct {
	l           *lexer.Lexer
	tw          *tabwriter.Writer
	b           *bytes.Buffer
	indentLevel int
	indent      string
	comments    map[*token.Token][]*token.Token
}

// newComparator creates a comparator.
func newComparator(code string, comments map[*token.Token][]*token.Token) *comparator {
	b := new(bytes.Buffer)
	tw := tabwriter.NewWriter(b, 8, 1, 1, ' ', 0)
	fmt.Fprintf(tw, "%s\t%s\t%s\n", "PARSED", "EXPECTED", "ERROR")
	return &comparator{
		tw:       tw,
		b:        b,
		l:        lexer.New([]byte(code)),
		indent:   ". ",
		comments: comments,
	}
}

// compare compares parsed with expected. If they are not equal it returns false.
func (c *comparator) compare(parsed, expected parsetree.Construction) bool {
	if parsed != nil && expected != nil {
		switch p := parsed.(type) {
		case parsetree.NonTerminal:
			e, ok := expected.(parsetree.NonTerminal)
			if !ok {
				fmt.Fprintf(c.tw, "%s%s\t%s%s\t",
					strings.Repeat(c.indent, c.indentLevel), parsed.Kind(),
					strings.Repeat(c.indent, c.indentLevel), expected.Kind(),
				)
				fmt.Fprintf(c.tw, "%T ≠ %T\n", parsed, expected)
				return false
			}

			return c.compareNonTerminals(p, e)
		case parsetree.Terminal:
			e, ok := expected.(parsetree.Terminal)
			if !ok {
				fmt.Fprintf(c.tw, "%s%s\t%s%s\t",
					strings.Repeat(c.indent, c.indentLevel), parsed.Kind(),
					strings.Repeat(c.indent, c.indentLevel), expected.Kind(),
				)
				fmt.Fprintf(c.tw, "%T ≠ %T\n", parsed, expected)
				return false
			}

			return c.compareTerminals(p, e)
		case parsetree.Error:
			e, ok := expected.(parsetree.Error)
			if !ok {
				fmt.Fprintf(c.tw, "%s%s\t%s%s\t",
					strings.Repeat(c.indent, c.indentLevel), parsed.Kind(),
					strings.Repeat(c.indent, c.indentLevel), expected.Kind(),
				)
				fmt.Fprintf(c.tw, "%T ≠ %T <%s> \n", parsed, expected, p.Error())
				return false
			}

			return c.compareErrors(p, e)
		default:
			panic(fmt.Errorf("unknown type: %T", p))
		}
	} else if parsed != nil && expected == nil {
		fmt.Fprintf(c.tw, "%s%s\t\t\n", strings.Repeat(c.indent, c.indentLevel), parsed.Kind())

		nt, ok := parsed.(parsetree.NonTerminal)
		if !ok {
			return false
		}

		for child := range nt.Children {
			c.indentLevel++
			c.compare(child, nil)
			c.indentLevel--
		}
		return false
	} else if parsed == nil && expected != nil {
		fmt.Fprintf(c.tw, "\t%s%s\t\n", strings.Repeat(c.indent, c.indentLevel), expected.Kind())

		nt, ok := expected.(parsetree.NonTerminal)
		if !ok {
			return false
		}

		for child := range nt.Children {
			c.indentLevel++
			c.compare(nil, child)
			c.indentLevel--
		}
		return false
	}

	return true
}

// compareNonTerminals compares non terminals.
func (c *comparator) compareNonTerminals(parsed, expected parsetree.NonTerminal) bool {
	fmt.Fprintf(c.tw, "%s%s\t%s%s\t",
		strings.Repeat(c.indent, c.indentLevel), parsed.Kind(),
		strings.Repeat(c.indent, c.indentLevel), expected.Kind(),
	)

	if parsed.Kind() != expected.Kind() {
		fmt.Fprintf(c.tw, "%s ≠ %s\n", parsed.Kind(), expected.Kind())
		return false
	}

	if parsed.NumberOfChildren() != expected.NumberOfChildren() {
		fmt.Fprintf(c.tw, "number of children: %d ≠ %d",
			parsed.NumberOfChildren(), expected.NumberOfChildren(),
		)
	}

	fmt.Fprintln(c.tw)

	pn, ps := iter.Pull(parsed.Children)
	defer ps()
	en, es := iter.Pull(expected.Children)
	defer es()

	equals := true
	for range max(parsed.NumberOfChildren(), expected.NumberOfChildren()) {
		pc, _ := pn()
		ec, _ := en()

		c.indentLevel++
		if !c.compare(pc, ec) {
			equals = false
		}
		c.indentLevel--
	}

	return equals
}

// compareTerminals compares terminals.
func (c *comparator) compareTerminals(parsed, expected parsetree.Terminal) bool {
	fmt.Fprintf(c.tw, "%s%s\t%s%s\t",
		strings.Repeat(c.indent, c.indentLevel), parsed.Kind(),
		strings.Repeat(c.indent, c.indentLevel), expected.Kind(),
	)

	if parsed.Kind() != expected.Kind() {
		fmt.Fprintf(c.tw, "%s ≠ %s\n", parsed.Kind(), expected.Kind())
		return false
	}

	var tok *token.Token
	var comments []*token.Token
	for {
		tok = c.l.Next()
		if tok.Kind == token.KindSQLComment || tok.Kind == token.KindCComment {
			comments = append(comments, tok)
		} else if tok.Kind != token.KindWhiteSpace {
			break
		}
	}

	if parsed.Token().Kind != tok.Kind || !bytes.Equal(parsed.Token().Lexeme, tok.Lexeme) {
		fmt.Fprintf(c.tw, "%s ≠ %s\n", parsed.Token(), tok)
		return false
	} else if slices.CompareFunc(comments, c.comments[parsed.Token()], func(a, b *token.Token) int {
		return strings.Compare(string(a.Lexeme), string(b.Lexeme))
	}) != 0 {
		fmt.Fprintf(c.tw, "comments differ\n")
		pc := c.comments[parsed.Token()]
		for i := range max(len(pc), len(comments)) {
			fmt.Fprint(c.tw, "\t\t")
			if i < len(pc) {
				fmt.Fprintf(c.tw, "%q", string(pc[i].Lexeme))
			} else {
				fmt.Fprintf(c.tw, "%q", "")
			}

			fmt.Fprint(c.tw, " ≠ ")

			if i < len(comments) {
				fmt.Fprintf(c.tw, "%q\n", string(comments[i].Lexeme))
			} else {
				fmt.Fprintf(c.tw, "%q\n", "")
			}
		}

		return false
	}

	fmt.Fprintln(c.tw)

	return true
}

// compareErrors compares errors.
func (c *comparator) compareErrors(parsed, expected parsetree.Error) bool {
	fmt.Fprintf(c.tw, "%s%s\t%s%s\t",
		strings.Repeat(c.indent, c.indentLevel), parsed.Kind(),
		strings.Repeat(c.indent, c.indentLevel), expected.Kind(),
	)

	if parsed.Kind() != expected.Kind() {
		fmt.Fprintf(c.tw, "%s ≠ %s\n", parsed.Kind(), expected.Kind())
		return false
	}

	fmt.Fprintln(c.tw)

	return true
}

// log returns the data generated by the comparator.
func (c *comparator) log() string {
	c.tw.Flush()
	return c.b.String()
}

// writeErrors writes to b errors messages in c.
func writeErrors(b *strings.Builder, c parsetree.Construction) {
	switch c := c.(type) {
	case parsetree.NonTerminal:
		for child := range c.Children {
			writeErrors(b, child)
		}
	case parsetree.Error:
		b.WriteString(c.Error() + "\n")
	}
}

// testTokenKind is the kind of a token in the language for especifying a parse tree.
type testTokenKind int

const (
	tokenKindIdentifier testTokenKind = iota
	tokenKindLeftBrace
	tokenKindRightBrace
	tokenKindTokens
	tokenKindError
	tokenKindEOF
)

// testToken is a token in the language of especifying a parse tree.
type testToken struct {
	kind   testTokenKind
	lexeme string
}

// testLexer is a lexer for a language for especifying a parse tree.
type testLexer struct {
	r *strings.Reader
	// rn is the current lookahead rune.
	rn  rune
	eof bool
}

// newTestLexer creates a testLexer.
func newTestLexer(code string) *testLexer {
	l := &testLexer{
		r: strings.NewReader(code),
	}
	l.advance()
	return l
}

// next returns the next token.
func (l *testLexer) next() *testToken {
	if l.eof {
		return &testToken{kind: tokenKindEOF}
	}
	for unicode.IsSpace(l.rn) {
		if l.advance() {
			return &testToken{kind: tokenKindEOF}
		}
	}

	if unicode.IsLetter(l.rn) {
		return l.word()
	} else if l.rn == '!' {
		l.advance()
		if !unicode.IsLetter(l.rn) {
			panic(fmt.Errorf("invalid rune: %q", l.rn))
		}
		tok := l.word()
		tok.kind = tokenKindError
		return tok
	} else if l.rn == '{' {
		l.advance()
		return &testToken{kind: tokenKindLeftBrace}
	} else if l.rn == '}' {
		l.advance()
		return &testToken{kind: tokenKindRightBrace}
	} else {
		panic(fmt.Errorf("invalid rune: %q", l.rn))
	}
}

// word scans a word.
func (l *testLexer) word() *testToken {
	var b strings.Builder
	b.WriteRune(l.rn)

	for eof := l.advance(); !eof; eof = l.advance() {
		if !unicode.IsLetter(l.rn) && !unicode.IsDigit(l.rn) {
			break
		}
		b.WriteRune(l.rn)
	}
	return l.createWordToken(b.String())
}

// createWordToken creates a token of type identifier or tokens, according with the lexeme.
func (l *testLexer) createWordToken(lexeme string) *testToken {
	reTokens := regexp.MustCompile(`^(T|(TT+))$`)
	tok := &testToken{lexeme: lexeme}
	if reTokens.MatchString(lexeme) {
		tok.kind = tokenKindTokens
	} else {
		tok.kind = tokenKindIdentifier
	}
	return tok
}

// advance advances the reader.
func (l *testLexer) advance() (eof bool) {
	rn, _, err := l.r.ReadRune()
	if err == io.EOF {
		l.eof = true
		return true
	} else if err != nil {
		panic(err)
	}
	l.rn = rn

	return false
}

// testParser is a parser for the language of specifying the parse tree.
type testParser struct {
	l *testLexer
	// tok is the current lookahead.
	tok *testToken
}

// newTestParser creates a testParser.
func newTestParser(l *testLexer) *testParser {
	return &testParser{l: l}
}

// tree parses a parse tree.
func (p *testParser) tree() (t parsetree.Construction) {
	if p.tok == nil {
		p.advance()
	}

	var kind parsetree.Kind
	if p.tok.kind == tokenKindIdentifier {
		kind = p.treeKind(p.tok.lexeme)
		p.advance()
	} else {
		panic("expecting tree kind")
	}

	if p.tok.kind == tokenKindEOF {
		t = parsetree.NewTerminal(kind, nil)
		return t
	}

	if p.tok.kind == tokenKindLeftBrace {
		nt := parsetree.NewNonTerminal(kind)
		p.advance()
		for _, c := range p.children() {
			nt.AddChild(c)
		}
		if p.tok.kind == tokenKindRightBrace {
			p.advance()
		} else {
			panic("expecting right brace")
		}
		t = nt
	} else {
		return parsetree.NewTerminal(kind, nil)
	}
	return
}

// children parses children trees.
func (p *testParser) children() (cs []parsetree.Construction) {
	for {
		if p.tok.kind == tokenKindIdentifier {
			cs = append(cs, p.tree())
		} else if p.tok.kind == tokenKindError {
			cs = append(cs, parsetree.NewError(p.treeKind(p.tok.lexeme), nil))
			p.advance()
		} else if p.tok.kind == tokenKindTokens {
			for range p.tok.lexeme {
				cs = append(cs, parsetree.NewTerminal(parsetree.KindToken, nil))
			}
			p.advance()
		} else {
			return
		}
	}
}

// treeKind return the parse tree kind correponding to lexeme.
func (p *testParser) treeKind(lexeme string) parsetree.Kind {
	k, ok := treeKinds[lexeme]
	if !ok {
		panic(fmt.Errorf("unknown kind: %s", lexeme))
	}
	return k
}

// advance advances the lexer.
func (p *testParser) advance() {
	p.tok = p.l.next()
}

// treeKinds maps a identifier, in the language of specifying the parse tree, to your corresponding
// kind.
var treeKinds = map[string]parsetree.Kind{}

// init initializes treeKinds.
func init() {
	for i := parsetree.KindAdd; i <= parsetree.KindWindowName; i++ {
		treeKinds[i.String()] = i
	}
	treeKinds["E"] = parsetree.KindExpression
	treeKinds["ParenE"] = parsetree.KindParenExpression
	treeKinds["ColConstr"] = parsetree.KindColumnConstraint
	treeKinds["ColDef"] = parsetree.KindColumnDefinition
	treeKinds["ColRef"] = parsetree.KindColumnReference
	treeKinds["ColName"] = parsetree.KindColumnName
	treeKinds["FnName"] = parsetree.KindFunctionName
	treeKinds["FnCall"] = parsetree.KindFunctionCall
	treeKinds["FnArgs"] = parsetree.KindFunctionArguments
}
