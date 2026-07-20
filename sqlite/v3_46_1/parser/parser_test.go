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

type testCase struct {
	code string
	tree string
}

type testCaseError struct {
	code string
	msg  string
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
	cases := testCases(
		`EXPLAIN ALTER TABLE table_a ADD COLUMN column_b NOT NULL AS (10) VIRTUAL;`,
		`SQLStatement {Explain {T AlterTable {TT TableName AddColumn {TT ColDef {ColName
			ColConstr {NotNullColumnConstraint{TT}}
			ColConstr {GeneratedColumnConstraint{TT E{T} T T} }}}}} T}`,
		`EXPLAIN QUERY PLAN ALTER TABLE table_a ADD COLUMN column_b NOT NULL AS (10) VIRTUAL;`,
		`SQLStatement {ExplainQueryPlan {TTT AlterTable {TT TableName AddColumn {TT ColDef {ColName
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
		"SQLStatement {SimpleSelect {SelectCore{T CommaList{ResultColumn{E{T}}}}} T}",
		`CREATE TABLE table_name (column_a);`,
		"SQLStatement{CreateTable {TT TableName T CommaList{ColDef{ColName}} T} T}",
		`CREATE TEMP TABLE IF NOT EXISTS temp.table_name (column_a INTEGER);`,
		"SQLStatement{CreateTable {TTT TTT SchemaName T TableName T CommaList{ColDef{ColName TypeName{T}}} T} T}",
		`CREATE TRIGGER trigger_name DELETE ON table_name BEGIN DELETE FROM table_name; END;`,
		"SQLStatement{CreateTrigger{TT TriggerName TT TableName TriggerBody{T Delete {TT QualifiedTableName{TableName}} TT}} T}",
		`CREATE TEMP TRIGGER IF NOT EXISTS trigger_name BEFORE DELETE ON table_name BEGIN INSERT INTO table_name(name) VALUES('Go'); END;`,
		`SQLStatement{CreateTrigger{TTT TTT TriggerName TTT TableName TriggerBody{T Insert{TT TableName T CommaList{ColumnName} T T
			InsertValuesList{CommaList{InsertValuesItem{T CommaList{E{T}} T}}} } TT}} T}`,
		`CREATE VIEW view_name AS SELECT 10;`,
		"SQLStatement{CreateView{TT ViewName T SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}}} T}",
		`CREATE TEMP VIEW IF NOT EXISTS view_name AS SELECT 10;`,
		"SQLStatement{CreateView{TTT TTT ViewName T SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}}} T}",
		`CREATE VIRTUAL TABLE table_name USING module_name`,
		"SQLStatement{CreateVirtualTable{TTT TableName T ModuleName} T}",
		`DELETE FROM tableName`,
		"SQLStatement{Delete{TT QualifiedTableName{TableName}} T}",
		`WITH cte AS (SELECT 10) DELETE FROM tableName`,
		`SQLStatement{Delete{WithClause{T CommaList{CTE{TableName T T SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}} T}}}
			TT QualifiedTableName{TableName}} T}`,
		`DETACH temp`,
		"SQLStatement{Detach{T SchemaName} T}",
		`DROP INDEX index_name`,
		"SQLStatement{DropIndex{TT IndexName} T}",
		`DROP TABLE table_name`,
		"SQLStatement{DropTable{TT TableName} T}",
		`DROP TRIGGER trigger_name`,
		"SQLStatement{DropTrigger{TT TriggerName} T}",
		`DROP VIEW view_name`,
		"SQLStatement{DropView{TT ViewName} T}",
		`WITH cte AS (SELECT 10) INSERT INTO table_name(name) VALUES('Go')`,
		`SQLStatement{Insert{WithClause{T CommaList{CTE{TableName T T SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}} T}}}
			TT TableName T CommaList{ColumnName} T T
			InsertValuesList{CommaList{InsertValuesItem{T CommaList{E{T}} T}}}} T}`,
		`INSERT INTO table_name(name) VALUES('Go')`,
		`SQLStatement{Insert{TT TableName T CommaList{ColumnName} T T
			InsertValuesList{CommaList{InsertValuesItem{T CommaList{E{T}} T}}}} T}`,
		`REPLACE INTO table_name(name) VALUES('Go')`,
		`SQLStatement{Insert{TT TableName T CommaList{ColumnName} T T
			InsertValuesList{CommaList{InsertValuesItem{T CommaList{E{T}} T}}}} T}`,
		`PRAGMA pragma_name`, "SQLStatement{Pragma{T PragmaName} T}",
		`REINDEX`, "SQLStatement{Reindex{T} T}",
		`RELEASE savepoint_name`, "SQLStatement{Release{T SavepointName} T}",
		`SAVEPOINT savepoint_name`, "SQLStatement{Savepoint{T SavepointName} T}",
		`WITH cte AS (SELECT 10) SELECT 10;`,
		`SQLStatement{SimpleSelect{
				WithClause{T CommaList{CTE{TableName T T SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}} T}} }
				SelectCore{T CommaList{ResultColumn{E{T}}}} } T}`,
		`UPDATE table_name SET a=10;`,
		`SQLStatement{Update{
				T QualifiedTableName{TableName} T CommaList{UpdateSetItem{ColName T E{T}}} } T}`,
		`WITH cte AS (SELECT 10) UPDATE table_name SET a=10;`,
		`SQLStatement{Update{
				WithClause{T CommaList{CTE{TableName T T SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}} T}} }
				T QualifiedTableName{TableName} T CommaList{UpdateSetItem{ColName T E{T}}} } T}`,
		`VACUUM;`, "SQLStatement{Vacuum{T} T}",
	)

	for i, c := range cases {
		c := c
		t.Run(strconv.FormatInt(int64(i), 10), func(t *testing.T) {
			defer func() {
				switch v := recover().(type) {
				case *syntaxError:
					t.Fatalf("%d=>%q\n\t%s", i, c.code, v)
				case nil:
				default:
					panic(v)
				}
			}()
			tp := newTestParser(newTestLexer(c.tree))
			expected := tp.tree()

			p := New(lexer.New([]byte(c.code)))
			parsed, comments := p.SQLStatement()

			if str, equals := compare(c.code, comments, parsed, expected); !equals {
				t.Log(c.code)
				t.Log("\n" + str)
				t.Fail()
			}
		})
	}
}

func TestSQLStatementError(t *testing.T) {
	cases := []testCaseError{
		{code: `DROP`, msg: "expecting [Index Table Trigger View], got EOF"},
		{code: `WITH cte AS (SELECT 10) `, msg: "expecting [Delete Insert Replace Select Update], got EOF"},
		{code: `EXPLAIN QUERY ALTER TABLE table_a ADD COLUMN column_b NOT NULL AS (10) VIRTUAL;`, msg: "expecting Plan, got Alter"},
	}

	for i, c := range cases {
		t.Run(strconv.FormatInt(int64(i), 10), func(t *testing.T) {
			defer func() {
				switch v := recover().(type) {
				case error:
					if c.msg != v.Error() {
						t.Errorf("got error message %q, want %q", v.Error(), c.msg)
					}
				case nil:
					t.Error("not panicked")
				default:
					t.Errorf("unexpected %v", v)
				}
			}()
			p := New(lexer.New([]byte(c.code)))
			p.SQLStatement()
		})
	}
}

func TestAlterTable(t *testing.T) {
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
	)

	runTests(t, cases, (*Parser).alterTable)
}

func TestAlterTableError(t *testing.T) {
	cases := []testCaseError{
		{code: `ALTER TABLE RENAME TO table_b`, msg: "expecting Identifier, got Rename"},
		{code: `ALTER TABLE schema_a. RENAME TO table_b`, msg: "expecting Identifier, got Rename"},
		{code: `ALTER TABLE table_a RENAME TO `, msg: "expecting Identifier, got EOF"},
		{code: `ALTER TABLE table_a RENAME column_a TO `, msg: "expecting Identifier, got EOF"},
		{code: `ALTER TABLE table_a 10 RENAME column_a TO `, msg: "expecting [Rename Add Drop], got Numeric"},
		{code: `ALTER`, msg: "expecting Table, got EOF"},
		{code: `ALTER TABLE table_a RENAME column_a column_b `, msg: "expecting To, got Identifier"},
		{code: `ALTER TABLE table_a RENAME COLUMN TO column_b `, msg: "expecting Identifier, got To"},
		{code: `ALTER TABLE table_a ADD COLUMN `, msg: "expecting Identifier, got EOF"},
		{code: `ALTER TABLE table_a ADD COLUMN column_a INTEGER()`, msg: "expecting Numeric, got RightParen"},
		{code: `ALTER TABLE table_a DROP COLUMN`, msg: "expecting Identifier, got EOF"},
	}

	runTestsError(t, cases, (*Parser).alterTable)
}

func TestColumnDefinitionList(t *testing.T) {
	cases := testCases(
		`column_a`, "CommaList{ColDef{ColName}}",
		`column_a, column_b`, "CommaList{ColDef{ColName} T ColDef{ColName}}",
	)

	runTests(t, cases, (*Parser).columnDefinitionList)
}

func TestColumnDefinitionListError(t *testing.T) {
	cases := []testCaseError{
		{code: `, column_b`, msg: "expecting Identifier, got Comma"},
	}

	runTestsError(t, cases, (*Parser).columnDefinitionList)
}

func TestColumnDefinition(t *testing.T) {
	cases := testCases(
		`a INTEGER PRIMARY KEY`,
		"ColDef{ColName TypeName{T} ColConstr{PrimaryKeyColumnConstraint{TT}}}",
	)

	runTests(t, cases, (*Parser).columnDefinition)
}

func TestTypeName(t *testing.T) {
	cases := testCases(
		`INTEGER`, "TypeName{T}",
		`INTEGER(10)`, "TypeName{TTTT}",
		`INTEGER(+10, -10)`, "TypeName{TTTTTTTT}",
	)

	runTests(t, cases, (*Parser).typeName)
}

func TestTypeNameError(t *testing.T) {
	cases := []testCaseError{
		{code: `INTEGER(, -10)`, msg: "expecting Numeric, got Comma"},
		{code: `INTEGER(-10, )`, msg: "expecting Numeric, got RightParen"},
		{code: `INTEGER(-10 10)`, msg: "expecting Comma, got Numeric"},
	}

	runTestsError(t, cases, (*Parser).typeName)
}

func TestColumnConstraint(t *testing.T) {
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
	)

	runTests(t, cases, (*Parser).columnConstraint)
}

func TestColumnConstraintError(t *testing.T) {
	cases := []testCaseError{
		{code: `CONSTRAINT PRIMARY KEY`, msg: "expecting Identifier, got Primary"},
		{code: `10`, msg: "expecting [Primary Not Unique Check Default Collate References Generated As], got Numeric"},
	}

	runTestsError(t, cases, (*Parser).columnConstraint)
}

func TestPrimaryKeyColumnConstraint(t *testing.T) {
	cases := testCases(
		`PRIMARY KEY`,
		"PrimaryKeyColumnConstraint{TT}",
		`PRIMARY KEY ASC ON CONFLICT ROLLBACK`,
		"PrimaryKeyColumnConstraint{TTT ConflictClause{TTT}}",
		`PRIMARY KEY DESC AUTOINCREMENT`,
		"PrimaryKeyColumnConstraint{TTTT}",
	)

	runTests(t, cases, (*Parser).primaryKeyColumnConstraint)
}

func TestPrimaryKeyColumnConstraintError(t *testing.T) {
	cases := []testCaseError{
		{code: `PRIMARY`, msg: "expecting Key, got EOF"},
	}

	runTestsError(t, cases, (*Parser).primaryKeyColumnConstraint)
}

func TestNotNullColumnConstraint(t *testing.T) {
	cases := testCases(
		`NOT NULL`,
		"NotNullColumnConstraint{TT}",
		`NOT NULL ON CONFLICT ROLLBACK`,
		"NotNullColumnConstraint{TT ConflictClause{TTT}}",
	)

	runTests(t, cases, (*Parser).notNullColumnConstraint)
}

func TestNotNullColumnConstraintError(t *testing.T) {
	cases := []testCaseError{
		{code: `NOT`, msg: "expecting Null, got EOF"},
	}

	runTestsError(t, cases, (*Parser).notNullColumnConstraint)
}

func TestUniqueColumnConstraint(t *testing.T) {
	cases := testCases(
		`UNIQUE`, "UniqueColumnConstraint{T}",
		`UNIQUE ON CONFLICT ROLLBACK`,
		"UniqueColumnConstraint{T ConflictClause{TTT}}",
	)

	runTests(t, cases, (*Parser).uniqueColumnConstraint)
}

func TestCheckColumnConstraint(t *testing.T) {
	cases := testCases(
		`CHECK(a > 10)`,
		"CheckColumnConstraint{TT E{GreaterThan{ColRef{ColName} TT}} T}",
	)

	runTests(t, cases, (*Parser).checkColumnConstraint)
}

func TestCheckColumnConstraintError(t *testing.T) {
	cases := []testCaseError{
		{code: `CHECK a > 10)`, msg: "expecting LeftParen, got Identifier"},
		{code: `CHECK()`, msg: "unexpected RightParen"},
		{code: `CHECK(a > 10`, msg: "expecting RightParen, got EOF"},
	}

	runTestsError(t, cases, (*Parser).checkColumnConstraint)
}

func TestDefaultColumnConstraint(t *testing.T) {
	cases := testCases(
		`DEFAULT(a > 10)`,
		"DefaultColumnConstraint{TT E{GreaterThan{ColRef{ColName} TT}} T}",
		`DEFAULT 10`,
		"DefaultColumnConstraint{T T}",
		`DEFAULT -10`,
		"DefaultColumnConstraint{T TT}",
	)

	runTests(t, cases, (*Parser).defaultColumnConstraint)
}

func TestDefaultColumnConstraintError(t *testing.T) {
	cases := []testCaseError{
		{code: `DEFAULT a`, msg: "expecting [Plus Minus], got Identifier"},
		{code: `DEFAULT()`, msg: "unexpected RightParen"},
		{code: `DEFAULT(a > 10`, msg: "expecting RightParen, got EOF"},
		{code: `DEFAULT -a`, msg: "expecting Numeric, got Identifier"},
	}

	runTestsError(t, cases, (*Parser).defaultColumnConstraint)
}

func TestCollateColumnConstraint(t *testing.T) {
	cases := testCases(
		`COLLATE c`,
		"CollateColumnConstraint{T CollationName}",
	)

	runTests(t, cases, (*Parser).collateColumnConstraint)
}

func TestCollateColumnConstraintError(t *testing.T) {
	cases := []testCaseError{
		{code: `COLLATE`, msg: "expecting Identifier, got EOF"},
	}

	runTestsError(t, cases, (*Parser).collateColumnConstraint)
}

func TestGeneratedColumnConstraint(t *testing.T) {
	cases := testCases(
		`GENERATED ALWAYS AS (10)`,
		"GeneratedColumnConstraint{TTTT E{T} T}",
		`AS (10) STORED`,
		"GeneratedColumnConstraint{TT E{T} T T}",
		`AS (10) VIRTUAL`,
		"GeneratedColumnConstraint{TT E{T} T T}",
	)

	runTests(t, cases, (*Parser).generatedColumnConstraint)
}

func TestGeneratedColumnConstraintError(t *testing.T) {
	cases := []testCaseError{
		{code: `GENERATED AS (10)`, msg: "expecting Always, got As"},
		{code: `GENERATED ALWAYS (10)`, msg: "expecting As, got LeftParen"},
		{code: `AS ()`, msg: "unexpected RightParen"},
		{code: `AS 10)`, msg: "expecting LeftParen, got Numeric"},
		{code: `AS (10`, msg: "expecting RightParen, got EOF"},
	}

	runTestsError(t, cases, (*Parser).generatedColumnConstraint)
}

func TestForeignKeyColumnConstraint(t *testing.T) {
	cases := testCases(
		`REFERENCES table_name`,
		"ForeignKeyColumnConstraint{ForeignKeyClause{T TableName}}",
	)

	runTests(t, cases, (*Parser).foreignKeyColumnConstraint)
}

func TestForeignKeyClause(t *testing.T) {
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
	)

	runTests(t, cases, (*Parser).foreignKeyClause)
}

func TestForeignKeyClauseError(t *testing.T) {
	cases := []testCaseError{
		{code: `REFERENCES`, msg: "expecting Identifier, got EOF"},
		{code: `REFERENCES table_name(column_name`, msg: "expecting RightParen, got EOF"},
		{code: `REFERENCES table_name ON SET NULL`, msg: "expecting [Delete Update], got Set"},
		{code: `REFERENCES table_name ON UPDATE SET`, msg: "expecting [Null Default], got EOF"},
		{code: `REFERENCES table_name ON DELETE NO`, msg: "expecting Action, got EOF"},
		{code: `REFERENCES table_name ON UPDATE`, msg: "expecting [Set Cascade Restrict No], got EOF"},
		{code: `REFERENCES table_name MATCH`, msg: "expecting Identifier, got EOF"},
		{code: `REFERENCES table_name DEFERRABLE INITIALLY`, msg: "expecting [Deferred Immediate], got EOF"},
	}
	runTestsError(t, cases, (*Parser).foreignKeyClause)
}

func TestConflictClause(t *testing.T) {
	cases := testCases(
		`ON CONFLICT ROLLBACK`, "ConflictClause{TTT}",
		`ON CONFLICT ABORT`, "ConflictClause{TTT}",
		`ON CONFLICT FAIL`, "ConflictClause{TTT}",
		`ON CONFLICT IGNORE`, "ConflictClause{TTT}",
		`ON CONFLICT REPLACE`, "ConflictClause{TTT}",
	)

	runTests(t, cases, (*Parser).conflictClause)
}

func TestConflictClauseError(t *testing.T) {
	cases := []testCaseError{
		{code: `ON REPLACE`, msg: "expecting Conflict, got Replace"},
		{code: `ON CONFLICT`, msg: "expecting [Rollback Abort Fail Ignore Replace], got EOF"},
	}

	runTestsError(t, cases, (*Parser).conflictClause)
}

func TestAnalyze(t *testing.T) {
	cases := testCases(
		`ANALYZE schema_name`,
		"Analyze {T SchemaIndexOrTableName}",
		`ANALYZE schema_name.table_name`,
		"Analyze {T SchemaName T TableOrIndexName}",
	)

	runTests(t, cases, (*Parser).analyze)
}

func TestAnalyzeError(t *testing.T) {
	cases := []testCaseError{
		{code: `ANALYZE `, msg: "expecting Identifier, got EOF"},
		{code: `ANALYZE schema_name.`, msg: "expecting Identifier, got EOF"},
		{code: `ANALYZE .table_name`, msg: "expecting Identifier, got Dot"},
		{code: `ANALYZE .`, msg: "expecting Identifier, got Dot"},
	}

	runTestsError(t, cases, (*Parser).analyze)
}

func TestAttach(t *testing.T) {
	cases := testCases(
		`ATTACH DATABASE ':memory' AS schema_name`, "Attach {TT E{T} T SchemaName}",
		`ATTACH '' AS schema_name`, "Attach {T E{T} T SchemaName}",
	)

	runTests(t, cases, (*Parser).attach)
}

func TestAttachError(t *testing.T) {
	cases := []testCaseError{
		{code: `ATTACH ;`, msg: "unexpected Semicolon"},
		{code: `ATTACH AS ;`, msg: "unexpected As"},
		{code: `ATTACH ':memory' schema_name ;`, msg: "expecting As, got Identifier"},
	}

	runTestsError(t, cases, (*Parser).attach)
}

func TestBegin(t *testing.T) {
	cases := testCases(
		`BEGIN`, "Begin {T}",
		`BEGIN DEFERRED TRANSACTION`, "Begin {TTT}",
		`BEGIN IMMEDIATE`, "Begin {TT}",
		`BEGIN EXCLUSIVE TRANSACTION`, "Begin {TTT}",
	)

	runTests(t, cases, (*Parser).begin)
}

func TestCommit(t *testing.T) {
	cases := testCases(
		`COMMIT`, "Commit {T}",
		`COMMIT TRANSACTION`, "Commit {TT}",
		`END`, "Commit {T}",
		`END TRANSACTION`, "Commit {TT}",
	)

	runTests(t, cases, (*Parser).commit)
}

func TestRollback(t *testing.T) {
	cases := testCases(
		`ROLLBACK`,
		"Rollback {T}",
		`ROLLBACK TRANSACTION`,
		"Rollback {TT}",
		`ROLLBACK TRANSACTION TO save_point_name`,
		"Rollback {TTT SavepointName}",
		`ROLLBACK TO SAVEPOINT save_point_name`,
		"Rollback {TTT SavepointName}",
	)

	runTests(t, cases, (*Parser).rollback)
}

func TestRollbackError(t *testing.T) {
	cases := []testCaseError{
		{code: `ROLLBACK TO`, msg: "expecting Identifier, got EOF"},
		{code: `ROLLBACK TO SAVEPOINT`, msg: "expecting Identifier, got EOF"},
	}

	runTestsError(t, cases, (*Parser).rollback)
}

func TestCreateIndex(t *testing.T) {
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
	)

	runTests(t, cases, (*Parser).createIndex)
}

func TestCreateIndexError(t *testing.T) {
	cases := []testCaseError{
		{code: `CREATE INDEX IF EXISTS index_name ON table_name(column_name)`, msg: "expecting Not, got Exists"},
		{code: `CREATE INDEX IF NOT index_name ON table_name(column_name)`, msg: "expecting Exists, got Identifier"},
		{code: `CREATE INDEX .index_name ON table_name(column_name)`, msg: "expecting Identifier, got Dot"},
		{code: `CREATE INDEX ON table_name(column_name)`, msg: "expecting Identifier, got On"},
		{code: `CREATE INDEX index_name table_name(column_name)`, msg: "expecting On, got Identifier"},
		{code: `CREATE INDEX index_name ON (column_name)`, msg: "expecting Identifier, got LeftParen"},
		{code: `CREATE INDEX index_name ON table_name column_name)`, msg: "expecting LeftParen, got Identifier"},
		{code: `CREATE INDEX index_name ON table_name(column_name `, msg: "expecting RightParen, got EOF"},
		{code: `CREATE INDEX index_name ON table_name(column_name) WHERE `, msg: "unexpected EOF"},
	}

	runTestsError(t, cases, (*Parser).createIndex)
}

func TestIndexedColumn(t *testing.T) {
	cases := testCases(
		`column_name`,
		"IndexedColumn{ColName}",
		`column_name COLLATE c ASC`,
		"IndexedColumn{ColName T CollationName T}",
	)

	runTests(t, cases, (*Parser).indexedColumn)
}

func TestIndexedColumnError(t *testing.T) {
	cases := []testCaseError{
		{code: `column_name COLLATE ASC`, msg: "expecting Identifier, got Asc"},
	}

	runTestsError(t, cases, (*Parser).indexedColumn)
}

func TestIndexedColumnExpr(t *testing.T) {
	cases := testCases(
		`column_name`,
		"IndexedColumn{E{ColRef{ColName}}}",
		`column_name ASC`,
		"IndexedColumn{E{ColRef{ColName}} T}",
		`column_name DESC`,
		"IndexedColumn{E{ColRef{ColName}} T}",
	)

	runTests(t, cases, (*Parser).indexedColumnExpr)
}

func TestCreateTable(t *testing.T) {
	cases := testCases(
		`CREATE TABLE table_name (column_a);`,
		"CreateTable {TT TableName T CommaList{ColDef{ColName}} T}",
		`CREATE TEMP TABLE IF NOT EXISTS temp.table_name (column_a INTEGER);`,
		"CreateTable {TTT TTT SchemaName T TableName T CommaList{ColDef{ColName TypeName{T}}} T}",
		`CREATE TEMP TABLE IF NOT EXISTS schema_name.table_name (column_a INTEGER);`,
		"CreateTable {TTT TTT SchemaName T TableName T CommaList{ColDef{ColName TypeName{T}}} T}",
		`CREATE TEMPORARY TABLE table_name AS SELECT 'value';`,
		"CreateTable {TTT TableName T SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}}}",
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
		`CREATE TABLE table_a (column_b, CONSTRAINT cons_name CHECK(column_b > 10), PRIMARY KEY (column_b)) WITHOUT ROWID`,
		`CreateTable {TT TableName T CommaList{ColDef{ColName}} T
			CommaList{
				TableConstraint{T ConstraintName CheckTableConstraint{TT E{GreaterThan{ColRef{ColName} TT}} T}} T
				TableConstraint{PrimaryKeyTableConstraint{TTT CommaList{IndexedColumn{ColName}} T}}
			}
			T CommaList{TableOption{TT}}}`,
	)

	runTests(t, cases, (*Parser).createTable)
}

func TestCreateTableError(t *testing.T) {
	cases := []testCaseError{
		{code: `CREATE TABLE table_a`, msg: "expecting [LeftParen As], got EOF"},
		{code: `CREATE TABLE IF EXISTS table_name (column_a INTEGER);`, msg: "expecting Not, got Exists"},
		{code: `CREATE TABLE IF NOT table_name (column_a INTEGER);`, msg: "expecting Exists, got Identifier"},
		{code: `CREATE TABLE .table_name (column_a INTEGER);`, msg: "expecting Identifier, got Dot"},
		{code: `CREATE TABLE (column_a INTEGER);`, msg: "expecting Identifier, got LeftParen"},
		{code: `CREATE TABLE table_name (column_a INTEGER`, msg: "expecting RightParen, got EOF"},
		{code: `CREATE TEMPORARY TABLE table_name AS`, msg: "expecting Select, got EOF"},
	}

	runTestsError(t, cases, (*Parser).createTable)
}

func TestTableConstraint(t *testing.T) {
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
	)

	runTests(t, cases, (*Parser).tableConstraint)
}

func TestTableConstraintError(t *testing.T) {
	cases := []testCaseError{
		{code: `CONSTRAINT PRIMARY KEY (column_name)`, msg: "expecting Identifier, got Primary"},
		{code: `CONSTRAINT pk CREATE`, msg: "expecting [Primary Unique Check Foreign], got Create"},
	}

	runTestsError(t, cases, (*Parser).tableConstraint)
}

func TestPrimaryKeyTableConstraint(t *testing.T) {
	cases := testCases(
		`PRIMARY KEY (column_name) ON CONFLICT ROLLBACK`,
		"PrimaryKeyTableConstraint{TT T CommaList{IndexedColumn{ColName}} T ConflictClause{TTT}}",
	)

	runTests(t, cases, (*Parser).primaryKeyTableConstraint)
}

func TestPrimaryKeyTableConstraintError(t *testing.T) {
	cases := []testCaseError{
		{code: `PRIMARY (column_name)`, msg: "expecting Key, got LeftParen"},
		{code: `PRIMARY KEY column_name)`, msg: "expecting LeftParen, got Identifier"},
		{code: `PRIMARY KEY (column_name`, msg: "expecting RightParen, got EOF"},
	}

	runTestsError(t, cases, (*Parser).primaryKeyTableConstraint)
}

func TestUniqueTableConstraint(t *testing.T) {
	cases := testCases(
		`UNIQUE (column_name_a, column_name_b) ON CONFLICT ROLLBACK`,
		"UniqueTableConstraint{T T CommaList{IndexedColumn{ColName} T IndexedColumn{ColName}} T ConflictClause{TTT}}",
	)

	runTests(t, cases, (*Parser).uniqueTableConstraint)
}

func TestUniqueTableConstraintError(t *testing.T) {
	cases := []testCaseError{
		{code: `UNIQUE column_name)`, msg: "expecting LeftParen, got Identifier"},
		{code: `UNIQUE (column_name`, msg: "expecting RightParen, got EOF"},
	}

	runTestsError(t, cases, (*Parser).uniqueTableConstraint)
}

func TestCheckTableConstraint(t *testing.T) {
	cases := testCases(
		`CHECK (TRUE)`, "CheckTableConstraint{T T E{T} T}",
	)

	runTests(t, cases, (*Parser).checkTableConstraint)
}

func TestCheckTableConstraintError(t *testing.T) {
	cases := []testCaseError{
		{code: `CHECK TRUE)`, msg: "expecting LeftParen, got Identifier"},
		{code: `CHECK ()`, msg: "unexpected RightParen"},
		{code: `CHECK (TRUE`, msg: "expecting RightParen, got EOF"},
	}

	runTestsError(t, cases, (*Parser).checkTableConstraint)
}

func TestForeignKeyTableConstraint(t *testing.T) {
	cases := testCases(
		`FOREIGN KEY (column_name) REFERENCES table_name`,
		"ForeignKeyTableConstraint{TT T CommaList{ColName} T ForeignKeyClause{T TableName}}",
	)

	runTests(t, cases, (*Parser).foreignKeyTableConstraint)
}

func TestForeignKeyTableConstraintError(t *testing.T) {
	cases := []testCaseError{
		{code: `FOREIGN (column_name) REFERENCES table_name`, msg: "expecting Key, got LeftParen"},
		{code: `FOREIGN KEY column_name) REFERENCES table_name`, msg: "expecting LeftParen, got Identifier"},
		{code: `FOREIGN KEY () REFERENCES table_name`, msg: "expecting Identifier, got RightParen"},
		{code: `FOREIGN KEY (column_name REFERENCES table_name`, msg: "expecting RightParen, got References"},
		{code: `FOREIGN KEY (column_name)`, msg: "expecting References, got EOF"},
	}

	runTestsError(t, cases, (*Parser).foreignKeyTableConstraint)
}

func TestTableOptions(t *testing.T) {
	cases := testCases(
		`WITHOUT ROWID`,
		"CommaList{TableOption{TT}}",
		`STRICT, WITHOUT ROWID`,
		"CommaList{TableOption{T} T TableOption{TT}}}",
	)

	runTests(t, cases, (*Parser).tableOptions)
}

func TestTableOption(t *testing.T) {
	cases := testCases(
		`WITHOUT ROWID`,
		"TableOption{TT}",
		`STRICT`,
		"TableOption{T}",
	)

	runTests(t, cases, (*Parser).tableOption)
}

func TestTableOptionError(t *testing.T) {
	cases := []testCaseError{
		{code: `WITHOUT`, msg: "expecting RowId, got EOF"},
	}

	runTestsError(t, cases, (*Parser).tableOption)
}

func TestColumnNameList(t *testing.T) {
	cases := testCases(
		`column_name_1, column_name_2`,
		"CommaList{ColName T ColName}",
	)

	runTests(t, cases, (*Parser).columnNameList)
}

func TestCreateTrigger(t *testing.T) {
	cases := testCases(
		`CREATE TRIGGER trigger_name DELETE ON table_name BEGIN DELETE FROM table_name; END`,
		"CreateTrigger{TT TriggerName TT TableName TriggerBody{T Delete {TT QualifiedTableName{TableName}} TT}}",
		`CREATE TRIGGER trigger_name DELETE ON table_name BEGIN WITH cte AS (SELECT 10) DELETE FROM table_name; END`,
		`CreateTrigger{TT TriggerName TT TableName TriggerBody{T Delete {WithClause{T CommaList{
				CTE{TableName T T SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}}T }}}
			TT QualifiedTableName{TableName}} TT}}`,
		`CREATE TEMP TRIGGER IF NOT EXISTS trigger_name BEFORE DELETE ON table_name BEGIN INSERT INTO table_name(name) VALUES('Go'); END;`,
		`CreateTrigger{TTT TTT TriggerName TTT TableName TriggerBody{T Insert{TT TableName T CommaList{ColumnName} T T
			InsertValuesList{CommaList{InsertValuesItem{T CommaList{E{T}} T}}} } TT}}`,
		`CREATE TEMPORARY TRIGGER schema_name.trigger_name AFTER INSERT ON table_name FOR EACH ROW BEGIN SELECT 10; END`,
		"CreateTrigger{TTT SchemaName T TriggerName TTT TableName TTT TriggerBody{T SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}} TT}}",
		`CREATE TRIGGER trigger_name INSTEAD OF UPDATE ON table_name WHEN a > 10 BEGIN UPDATE table_name SET a=10; END`,
		`CreateTrigger{TT TriggerName TTTT TableName T E{GreaterThan{ColRef{ColName} T T}}
				TriggerBody{T Update {T QualifiedTableName{TableName} T CommaList{UpdateSetItem{ColumnName T E{T}}}} TT}}`,
		`CREATE TRIGGER trigger_name UPDATE OF a, b ON table_name BEGIN SELECT 10; END`,
		"CreateTrigger{TT TriggerName TT CommaList{ColName T ColName} T TableName TriggerBody{T SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}} TT}}",
	)

	runTests(t, cases, (*Parser).createTrigger)
}

func TestCreateTriggerError(t *testing.T) {
	cases := []testCaseError{
		{code: `CREATE TRIGGER trigger_name DELETE table_name BEGIN SELECT 10; END`, msg: "expecting On, got Identifier"},
		{code: `CREATE TRIGGER trigger_name DELETE ON BEGIN SELECT 10; END`, msg: "expecting Identifier, got Begin"},
		{code: `CREATE TRIGGER trigger_name DELETE ON table_name BEGIN SELECT 10 END`, msg: "expecting Semicolon, got End"},
		{code: `CREATE TRIGGER trigger_name DELETE ON table_name BEGIN SELECT 10; `, msg: "expecting [Delete Insert Select Update], got EOF"},
		{code: `CREATE TRIGGER trigger_name DELETE ON table_name BEGIN ; END`, msg: "expecting [Delete Insert Select Update], got Semicolon"},
		{code: `CREATE TEMP TRIGGER IF EXISTS trigger_name BEFORE DELETE ON table_name BEGIN SELECT 10; END`, msg: "expecting Not, got Exists"},
		{code: `CREATE TEMP TRIGGER IF NOT trigger_name BEFORE DELETE ON table_name BEGIN SELECT 10; END`, msg: "expecting Exists, got Identifier"},
		{code: `CREATE TEMP TRIGGER IF NOT EXISTS trigger_name BEFORE ON table_name BEGIN SELECT 10; END`, msg: "expecting [Delete Insert Update], got On"},
		{code: `CREATE TEMPORARY TRIGGER .trigger_name AFTER INSERT ON table_name FOR EACH ROW BEGIN SELECT 10; END`, msg: "expecting Identifier, got Dot"},
		{code: `CREATE TEMPORARY TRIGGER schema_name trigger_name AFTER INSERT ON table_name FOR EACH ROW BEGIN SELECT 10; END`, msg: "expecting [Delete Insert Update], got Identifier"},
		{code: `CREATE TEMPORARY TRIGGER schema_name.trigger_name AFTER INSERT ON table_name FOR ROW BEGIN SELECT 10; END`, msg: "expecting Each, got Row"},
		{code: `CREATE TEMPORARY TRIGGER schema_name.trigger_name AFTER INSERT ON table_name FOR EACH BEGIN SELECT 10; END`, msg: "expecting Row, got Begin"},
		{code: `CREATE TRIGGER trigger_name UPDATE OF a b ON table_name BEGIN SELECT 10; END`, msg: "expecting On, got Identifier"},
		{code: `CREATE TRIGGER trigger_name INSTEAD OF UPDATE ON table_name WHEN BEGIN SELECT 10; END`, msg: "unexpected Begin"},
		{code: `CREATE TRIGGER trigger_name INSTEAD UPDATE ON table_name BEGIN SELECT 10; END`, msg: "expecting Of, got Update"},
		{code: `CREATE TRIGGER INSTEAD OF UPDATE ON table_name BEGIN SELECT 10; END`, msg: "expecting Identifier, got Instead"},
		{code: `CREATE TRIGGER trigger_name DELETE ON table_name`, msg: "expecting Begin, got EOF"},
		{code: `CREATE TRIGGER trigger_name DELETE ON table_name BEGIN WITH cte AS (SELECT 10) ; END`, msg: "expecting [Delete Insert Select Update], got Semicolon"},
		{code: `CREATE TRIGGER trigger_name DELETE ON table_name BEGIN WITH cte AS (SELECT 10) END`, msg: "expecting [Delete Insert Select Update], got End"},
	}

	runTestsError(t, cases, (*Parser).createTrigger)
}

func TestCreateView(t *testing.T) {
	cases := testCases(
		`CREATE VIEW view_name AS SELECT 10`,
		"CreateView{TT ViewName T SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}}}",
		`CREATE TEMP VIEW IF NOT EXISTS view_name AS SELECT 10`,
		"CreateView{TTT TTT ViewName T SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}}}",
		`CREATE TEMPORARY VIEW schema_name.view_name (a, b) AS SELECT 10`,
		"CreateView{TTT SchemaName T ViewName  T CommaList{ColName T ColName} T T SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}}}",
	)

	runTests(t, cases, (*Parser).createView)
}

func TestCreateViewError(t *testing.T) {
	cases := []testCaseError{
		{code: `CREATE TEMP VIEW IF EXISTS view_name AS SELECT 10`, msg: "expecting Not, got Exists"},
		{code: `CREATE TEMP VIEW IF NOT view_name AS SELECT 10`, msg: "expecting Exists, got Identifier"},
		{code: `CREATE TEMP VIEW AS SELECT 10`, msg: "expecting Identifier, got As"},
		{code: `CREATE TEMP VIEW schema_name view_name AS SELECT 10`, msg: "expecting As, got Identifier"},
		{code: `CREATE TEMP VIEW view_name AS `, msg: "expecting Select, got EOF"},
		{code: `CREATE TEMPORARY VIEW view_name (a b) AS SELECT 10`, msg: "expecting RightParen, got Identifier"},
		{code: `CREATE TEMPORARY VIEW view_name (a, b AS SELECT 10`, msg: "expecting RightParen, got As"},
		{code: `CREATE TEMP VIEW view_name SELECT 10`, msg: "expecting As, got Select"},
		{code: `CREATE VIEW .view_name AS SELECT 10`, msg: "expecting Identifier, got Dot"},
	}

	runTestsError(t, cases, (*Parser).createView)
}

func TestCreateVirtualTable(t *testing.T) {
	cases := testCases(
		`CREATE VIRTUAL TABLE table_name USING module_name`,
		"CreateVirtualTable{TTT TableName T ModuleName}",
		`CREATE VIRTUAL TABLE IF NOT EXISTS table_name USING module_name`,
		"CreateVirtualTable{TTT TTT TableName T ModuleName}",
		`CREATE VIRTUAL TABLE temp.table_name USING module_name`,
		"CreateVirtualTable{TTT SchemaName T TableName T ModuleName}",
		`CREATE VIRTUAL TABLE table_name USING module_name(column1 TEXT, column2 INTEGER)`,
		"CreateVirtualTable{TTT TableName T ModuleName T CommaList{ModuleArgument{TT} T ModuleArgument{TT}} T}",
		`CREATE VIRTUAL TABLE table_name USING module_name(function(arg))`,
		"CreateVirtualTable{TTT TableName T ModuleName T CommaList{ModuleArgument{TTTT}} T}",
		`CREATE VIRTUAL TABLE table_name USING module_name(function(function(arg)))`,
		"CreateVirtualTable{TTT TableName T ModuleName T CommaList{ModuleArgument{TTTTTTT}} T}",
	)

	runTests(t, cases, (*Parser).createVirtualTable)
}

func TestCreateVirtualTableError(t *testing.T) {
	cases := []testCaseError{
		{code: `CREATE VIRTUAL table_name USING module_name`, msg: "expecting Table, got Identifier"},
		{code: `CREATE VIRTUAL TABLE IF EXISTS table_name USING module_name`, msg: "expecting Not, got Exists"},
		{code: `CREATE VIRTUAL TABLE IF NOT table_name USING module_name`, msg: "expecting Exists, got Identifier"},
		{code: `CREATE VIRTUAL TABLE .table_name USING module_name`, msg: "expecting Identifier, got Dot"},
		{code: `CREATE VIRTUAL TABLE schema_name. USING module_name`, msg: "expecting Identifier, got Using"},
		{code: `CREATE VIRTUAL TABLE schema_name table_name USING module_name`, msg: "expecting Using, got Identifier"},
		{code: `CREATE VIRTUAL TABLE schema_name.table_name module_name`, msg: "expecting Using, got Identifier"},
		{code: `CREATE VIRTUAL TABLE table_name USING`, msg: "expecting Identifier, got EOF"},
		{code: `CREATE VIRTUAL TABLE table_name USING module_name()`, msg: "unexpected RightParen"},
		{code: `CREATE VIRTUAL TABLE table_name USING module_name(function(function(`, msg: "unexpected EOF"},
		{code: `CREATE VIRTUAL TABLE table_name USING module_name(function`, msg: "unexpected EOF"},
	}

	runTestsError(t, cases, (*Parser).createVirtualTable)
}

func TestDelete(t *testing.T) {
	cases := testCases(
		`DELETE FROM table_name`,
		"Delete{TT QualifiedTableName{TableName}}",
		`DELETE FROM table_name WHERE a > b`,
		"Delete{TT QualifiedTableName{TableName} WhereClause{T E{GreaterThan{ColRef{ColName} T ColRef{ColName}}}}}",
		`DELETE FROM table_name RETURNING *`,
		"Delete{TT QualifiedTableName{TableName} ReturningClause{T CommaList{ReturningItem{T}}}}",
		`DELETE FROM table_name ORDER BY 20`,
		"Delete{TT QualifiedTableName{TableName} OrderByClause{TT CommaList{OrderingTerm{E{T}}}}}",
		`DELETE FROM table_name LIMIT 10`,
		"Delete{TT QualifiedTableName{TableName} LimitClause{T E{T}}}",
		`DELETE FROM table_name ORDER BY 20, 30`,
		"Delete{TT QualifiedTableName{TableName} OrderByClause{TT CommaList{OrderingTerm{E{T}} T OrderingTerm{E{T}}}}}",
	)

	runTests(t, cases, func(p *Parser) parsetree.NonTerminal { return p.delete(nil) })
}

func TestDeleteError(t *testing.T) {
	cases := []testCaseError{
		{code: `DELETE table_name`, msg: "expecting From, got Identifier"},
		{code: `DELETE FROM`, msg: "expecting Identifier, got EOF"},
		{code: `DELETE FROM table_name WHERE`, msg: "unexpected EOF"},
		{code: `DELETE FROM table_name RETURNING`, msg: "unexpected EOF"},
	}

	runTestsError(t, cases, func(p *Parser) parsetree.NonTerminal { return p.delete(nil) })
}

func TestWithClause(t *testing.T) {
	cases := testCases(
		`WITH table_name AS (SELECT 10)`,
		"WithClause{T CommaList{CTE{TableName TT SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}} T}}}",
		`WITH RECURSIVE table_name AS (SELECT 10)`,
		"WithClause{TT CommaList{CTE{TableName TT SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}} T}}}",
		`WITH table_name AS (SELECT 10), table_name2 AS (SELECT 10)`,
		`WithClause{T CommaList{CTE{TableName TT SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}} T} T
					CTE{TableName TT SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}} T}}}`,
		`WITH table_name AS (WITH table_name_2 AS (SELECT 10) SELECT 10)`,
		`WithClause{T CommaList{CTE{TableName TT SimpleSelect{
				WithClause{T CommaList{CTE{TableName TT SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}} T}}}
					SelectCore{T CommaList{ResultColumn{E{T}}}}} T}}}`,
	)

	runTests(t, cases, (*Parser).withClause)
}

func TestWithClauseError(t *testing.T) {
	cases := []testCaseError{
		{code: `WITH `, msg: "expecting Identifier, got EOF"},
		{code: `WITH table_name AS (SELECT 10), `, msg: "expecting Identifier, got EOF"},
	}

	runTestsError(t, cases, (*Parser).withClause)
}

func TestCTE(t *testing.T) {
	cases := testCases(
		`table_name AS (SELECT 10)`,
		"CTE{TableName T T SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}} T}",
		`table_name AS MATERIALIZED (SELECT 10)`,
		"CTE{TableName TT T SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}} T}",
		`table_name AS NOT MATERIALIZED (SELECT 10)`,
		"CTE{TableName T TT T SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}} T}",
		`table_name (col_a, col_b) AS (SELECT 10)`,
		"CTE{TableName T CommaList{ColName T ColName} T T T SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}} T}",
	)

	runTests(t, cases, (*Parser).cte)
}

func TestCTEError(t *testing.T) {
	cases := []testCaseError{
		{code: `table_name (col_a, col_b AS (SELECT 10)`, msg: "expecting RightParen, got As"},
		{code: `table_name AS SELECT 10)`, msg: "expecting LeftParen, got Select"},
		{code: `table_name AS (SELECT 10`, msg: "expecting RightParen, got EOF"},
		{code: `table_name (col_a, col_b) (SELECT 10)`, msg: "expecting As, got LeftParen"},
		{code: `table_name AS NOT (SELECT 10)`, msg: "expecting Materialized, got LeftParen"},
		{code: `table_name AS ()`, msg: "expecting Select, got RightParen"},
		{code: `table_name AS (WITH cte AS (SELECT 10))`, msg: "expecting Select, got RightParen"},
	}

	runTestsError(t, cases, (*Parser).cte)
}

func TestQualifiedTableName(t *testing.T) {
	cases := testCases(
		`table_name`,
		"QualifiedTableName{TableName}",
		`table_name AS alias_name`,
		"QualifiedTableName{TableName T TableAlias}",
		`schema_name.table_name AS alias_name`,
		"QualifiedTableName{SchemaName T TableName T TableAlias}",
		`table_name INDEXED BY index_name`,
		"QualifiedTableName{TableName TT IndexName}",
		`table_name NOT INDEXED`,
		"QualifiedTableName{TableName TT}",
	)

	runTests(t, cases, (*Parser).qualifiedTableName)
}

func TestQualifiedTableNameError(t *testing.T) {
	cases := []testCaseError{
		{code: `Schema_name.`, msg: "expecting Identifier, got EOF"},
		{code: `.table_name`, msg: "expecting Identifier, got Dot"},
		{code: `table_name AS`, msg: "expecting Identifier, got EOF"},
		{code: `table_name INDEXED index_name`, msg: "expecting By, got Identifier"},
		{code: `table_name INDEXED BY`, msg: "expecting Identifier, got EOF"},
		{code: `table_name NOT`, msg: "expecting Indexed, got EOF"},
	}

	runTestsError(t, cases, (*Parser).qualifiedTableName)
}

func TestReturningClause(t *testing.T) {
	cases := testCases(
		`RETURNING *`,
		"ReturningClause{T CommaList{ReturningItem{T}}}",
		`RETURNING a`,
		"ReturningClause{T CommaList{ReturningItem{E{ColRef{ColName}}}}}",
		`RETURNING a AS b`,
		"ReturningClause{T CommaList{ReturningItem{E{ColRef{ColName}} T ColumnAlias}}}",
		`RETURNING a b`,
		"ReturningClause{T CommaList{ReturningItem{E{ColRef{ColName}} ColumnAlias}}}",
		`RETURNING a AS b, c`,
		`ReturningClause{T CommaList{ReturningItem{E{ColRef{ColName}} T ColumnAlias}
			T ReturningItem{E{ColRef{ColName}}}}}`,
	)

	runTests(t, cases, (*Parser).returningClause)
}

func TestReturningClauseError(t *testing.T) {
	cases := []testCaseError{
		{code: `RETURNING`, msg: "unexpected EOF"},
		{code: `RETURNING a AS`, msg: "expecting Identifier, got EOF"},
	}

	runTestsError(t, cases, (*Parser).returningClause)
}

func TestDetach(t *testing.T) {
	cases := testCases(
		`DETACH temp`, "Detach{T SchemaName}",
		`DETACH DATABASE temp`, "Detach{TT SchemaName}",
	)

	runTests(t, cases, (*Parser).detach)
}

func TestDetachError(t *testing.T) {
	cases := []testCaseError{
		{code: `DETACH`, msg: "expecting [Identifier Temp], got EOF"},
	}

	runTestsError(t, cases, (*Parser).detach)
}

func TestDropIndex(t *testing.T) {
	cases := testCases(
		`DROP INDEX index_name`, "DropIndex{TT IndexName}",
		`DROP INDEX IF EXISTS index_name`, "DropIndex{TT TT IndexName}",
		`DROP INDEX temp.index_name`, "DropIndex{TT SchemaName T IndexName}",
	)

	runTests(t, cases, (*Parser).dropIndex)
}

func TestDropIndexError(t *testing.T) {
	cases := []testCaseError{
		{code: `DROP INDEX IF index_name`, msg: "expecting Exists, got Identifier"},
		{code: `DROP INDEX .index_name`, msg: "expecting Identifier, got Dot"},
		{code: `DROP INDEX schema_name.`, msg: "expecting Identifier, got EOF"},
	}

	runTestsError(t, cases, (*Parser).dropIndex)
}

func TestDropTable(t *testing.T) {
	cases := testCases(
		`DROP TABLE table_name`, "DropTable{TT TableName}",
		`DROP TABLE IF EXISTS table_name`, "DropTable{TT TT TableName}",
		`DROP TABLE temp.table_name`, "DropTable{TT SchemaName T TableName}",
	)

	runTests(t, cases, (*Parser).dropTable)
}

func TestDropTableError(t *testing.T) {
	cases := []testCaseError{
		{code: `DROP TABLE IF table_name`, msg: "expecting Exists, got Identifier"},
		{code: `DROP TABLE .table_name`, msg: "expecting Identifier, got Dot"},
		{code: `DROP TABLE schema_name.`, msg: "expecting Identifier, got EOF"},
	}

	runTestsError(t, cases, (*Parser).dropTable)
}

func TestDropTrigger(t *testing.T) {
	cases := testCases(
		`DROP TRIGGER trigger_name`, "DropTrigger{TT TriggerName}",
		`DROP TRIGGER IF EXISTS trigger_name`, "DropTrigger{TT TT TriggerName}",
		`DROP TRIGGER temp.trigger_name`, "DropTrigger{TT SchemaName T TriggerName}",
	)

	runTests(t, cases, (*Parser).dropTrigger)
}

func TestDropTriggerError(t *testing.T) {
	cases := []testCaseError{
		{code: `DROP TRIGGER IF trigger_name`, msg: "expecting Exists, got Identifier"},
		{code: `DROP TRIGGER .trigger_name`, msg: "expecting Identifier, got Dot"},
		{code: `DROP TRIGGER schema_name.`, msg: "expecting Identifier, got EOF"},
	}

	runTestsError(t, cases, (*Parser).dropTrigger)
}

func TestDropView(t *testing.T) {
	cases := testCases(
		`DROP VIEW view_name`, "DropView{TT ViewName}",
		`DROP VIEW IF EXISTS view_name`, "DropView{TT TT ViewName}",
		`DROP VIEW temp.view_name`, "DropView{TT SchemaName T ViewName}",
	)

	runTests(t, cases, (*Parser).dropView)
}

func TestDropViewError(t *testing.T) {
	cases := []testCaseError{
		{code: `DROP VIEW IF view_name`, msg: "expecting Exists, got Identifier"},
		{code: `DROP VIEW .view_name`, msg: "expecting Identifier, got Dot"},
		{code: `DROP VIEW schema_name.`, msg: "expecting Identifier, got EOF"},
	}

	runTestsError(t, cases, (*Parser).dropView)
}

func TestExpression(t *testing.T) {
	cases := testCases(
		`TRUE OR TRUE`, "E{Or{TTT}}",
		`NOT 'a' NOT LIKE 1`, "E{Not{T NotLike{TTTT}}}",
		`(NOT 'a') NOT LIKE 1`, "E{NotLike{ParenE{T CommaList{E{Not{TT}}} T} TTT}}",
		`0 AND 1 LIKE 1;`, "E{And{TT Like{TTT}}}",
		`(0 AND 1) LIKE 1;`, "E{Like{ParenE{T CommaList{E{And{TTT}}} T} TT}}",
	)

	runTests(t, cases, (*Parser).expression)
}

func TestExpression1(t *testing.T) {
	cases := testCases(
		`TRUE OR TRUE`, "E{Or{TTT}}",
	)

	runTests(t, cases, (*Parser).expression)
}

func TestExpression1Error(t *testing.T) {
	cases := []testCaseError{
		{code: `TRUE OR`, msg: "unexpected EOF"},
	}

	runTestsError(t, cases, (*Parser).expression)
}

func TestExpression2(t *testing.T) {
	cases := testCases(
		`TRUE AND TRUE`, "E{And{TTT}}",
	)

	runTests(t, cases, (*Parser).expression)
}

func TestExpression2Error(t *testing.T) {
	cases := []testCaseError{
		{code: `TRUE AND`, msg: "unexpected EOF"},
	}

	runTestsError(t, cases, (*Parser).expression)
}

func TestExpression3(t *testing.T) {
	cases := testCases(
		`NOT TRUE`, "E{Not{TT}}",
	)

	runTests(t, cases, (*Parser).expression)
}

func TestExpression3Error(t *testing.T) {
	cases := []testCaseError{
		{code: `NOT`, msg: "unexpected EOF"},
	}

	runTestsError(t, cases, (*Parser).expression)
}

func TestExpression4(t *testing.T) {
	cases := testCases(
		`a = c`, "E{Equal{ColRef{ColName} T ColRef{ColName}}}",
		`a == c`, "E{Equal{ColRef{ColName} T ColRef{ColName}}}",
		`a <> c`, "E{NotEqual{ColRef{ColName} T ColRef{ColName}}}",
		`a != c`, "E{NotEqual{ColRef{ColName} T ColRef{ColName}}}",
		`10 REGEXP '10'`, "E{Regexp{TTT}}",
		`'10' NOT REGEXP '20'`, "E{NotRegexp{TTTT}}",
		`'10' LIKE '20'`, "E{Like{TTT}}",
		`10 LIKE '10' ESCAPE '!'`, "E{Like{TTTTT}}",
		`'10' NOT LIKE '20' ESCAPE '!'`, "E{NotLike{TTTTTT}}",
		`'10' GLOB '20'`, "E{Glob{TTT}}",
		`10 NOT GLOB '10'`, "E{NotGlob{TTTT}}",
		`'10' MATCH '20'`, "E{Match{TTT}}",
		`10 NOT MATCH '10'`, "E{NotMatch{TTTT}}",
		`10 IS '10'`, "E{Is{TTT}}",
		`10 BETWEEN 5 AND 15`, "E{Between{TTTTT}}",
		`10 NOT BETWEEN 5 AND 15`, "E{NotBetween{TTTTTT}}",
		`10 IN (5)`, "E{In{TTT CommaList{E{T}} T}}",
		`10 NOT IN (5)`, "E{NotIn{TTTT CommaList{E{T}} T}}",
		`10 ISNULL`, "E{Isnull{TT}}",
		`10 NOTNULL`, "E{Notnull{TT}}",
		`10 NOT NULL`, "E{NotNull{TTT}}",
	)

	runTests(t, cases, (*Parser).expression)
}

func TestExpression4Error(t *testing.T) {
	cases := []testCaseError{
		{code: `10 ==`, msg: "unexpected EOF"},
		{code: `10 <>`, msg: "unexpected EOF"},
		{code: `10 GLOB`, msg: "unexpected EOF"},
		{code: `10 NOT GLOB`, msg: "unexpected EOF"},
		{code: `10 REGEXP`, msg: "unexpected EOF"},
		{code: `10 NOT REGEXP`, msg: "unexpected EOF"},
		{code: `10 MATCH`, msg: "unexpected EOF"},
		{code: `10 NOT MATCH`, msg: "unexpected EOF"},
		{code: `10 LIKE ESCAPE`, msg: "unexpected Escape"},
		{code: `10 NOT LIKE ESCAPE`, msg: "unexpected Escape"},
		{code: `10 IN table_function_name(10`, msg: "expecting RightParen, got EOF"},
		{code: `10 NOT IN table_function_name(10`, msg: "expecting RightParen, got EOF"},
	}

	runTestsError(t, cases, (*Parser).expression)
}

func TestEqual(t *testing.T) {
	cases := testCases(
		`a = c`, "E{Equal{ColRef{ColName} T ColRef{ColName}}}",
		`a = c = d`, "E{Equal{Equal{ColRef{ColName} T ColRef{ColName}} T ColRef{ColName}}}",
		`a == c`, "E{Equal{ColRef{ColName} T ColRef{ColName}}}",
		`a = c == d`, "E{Equal{Equal{ColRef{ColName} T ColRef{ColName}} T ColRef{ColName}}}",
		`a = NOT d`, "E{Equal{ColRef{ColName} T Not{T ColRef{ColName}}}}",
	)

	runTests(t, cases, (*Parser).expression)
}

func TestEqualError(t *testing.T) {
	cases := []testCaseError{
		{code: `10 =`, msg: "unexpected EOF"},
		{code: `10 ==`, msg: "unexpected EOF"},
	}

	runTestsError(t, cases, (*Parser).expression)
}

func TestNotEqual(t *testing.T) {
	cases := testCases(
		`a <> c`, "E{NotEqual{ColRef{ColName} T ColRef{ColName}}}",
		`a <> NOT c`, "E{NotEqual{ColRef{ColName} T Not{T ColRef{ColName}}}}",
		`a <> c = d`, "E{Equal{NotEqual{ColRef{ColName} T ColRef{ColName}} T ColRef{ColName}}}",
		`a != c`, "E{NotEqual{ColRef{ColName} T ColRef{ColName}}}",
		`a != NOT c`, "E{NotEqual{ColRef{ColName} T Not{T ColRef{ColName}}}}",
		`a != c = d`, "E{Equal{NotEqual{ColRef{ColName} T ColRef{ColName}} T ColRef{ColName}}}",
	)

	runTests(t, cases, (*Parser).expression)
}

func TestNotEqualError(t *testing.T) {
	cases := []testCaseError{
		{code: `10 <>`, msg: "unexpected EOF"},
		{code: `10 !=`, msg: "unexpected EOF"},
	}

	runTestsError(t, cases, (*Parser).expression)
}

func TestRegexp(t *testing.T) {
	cases := testCases(
		`10 REGEXP '10'`, "E{Regexp{TTT}}",
		`1 REGEXP NOT 1`, "E{Regexp{TT Not{TT}}}",
		`'10' NOT REGEXP '20'`, "E{NotRegexp{TTTT}}",
		`1 NOT REGEXP NOT 1`, "E{NotRegexp{TTT Not{TT}}}",
	)

	runTests(t, cases, (*Parser).expression)
}

func TestRegexpError(t *testing.T) {
	cases := []testCaseError{
		{code: `10 REGEXP`, msg: "unexpected EOF"},
		{code: `10 NOT REGEXP`, msg: "unexpected EOF"},
	}

	runTestsError(t, cases, (*Parser).expression)
}

func TestLike(t *testing.T) {
	cases := testCases(
		`10 LIKE '10' ESCAPE '!'`, "E{Like{TTTTT}}",
		`1 LIKE NOT 1`, "E{Like{TT Not{TT}}}",
		`1 LIKE 1 ESCAPE NOT 1`, "E{Like{TTTT Not{TT}}}",
		`'10' NOT LIKE '20' ESCAPE '!'`, "E{NotLike{TTTTTT}}",
		`'10' NOT LIKE '20' ESCAPE NOT 1`, "E{NotLike{TTTTT Not{TT}}}",
		`1 NOT LIKE NOT 1`, "E{NotLike{TTT Not{TT}}}",
	)

	runTests(t, cases, (*Parser).expression)
}

func TestLikeError(t *testing.T) {
	cases := []testCaseError{
		{code: `10 LIKE ESCAPE`, msg: "unexpected Escape"},
		{code: `10 LIKE`, msg: "unexpected EOF"},
		{code: `10 NOT LIKE ESCAPE`, msg: "unexpected Escape"},
		{code: `10 NOT LIKE`, msg: "unexpected EOF"},
	}

	runTestsError(t, cases, (*Parser).expression)
}

func TestGlob(t *testing.T) {
	cases := testCases(
		`'10' GLOB '20'`, "E{Glob{TTT}}",
		`1 GLOB NOT 1`, "E{Glob{TT Not{TT}}}",
		`10 NOT GLOB '10'`, "E{NotGlob{TTTT}}",
		`1 NOT GLOB NOT 1`, "E{NotGlob{TTT Not{TT}}}",
	)

	runTests(t, cases, (*Parser).expression)
}

func TestGlobError(t *testing.T) {
	cases := []testCaseError{
		{code: `10 GLOB`, msg: "unexpected EOF"},
		{code: `10 NOT GLOB`, msg: "unexpected EOF"},
	}

	runTestsError(t, cases, (*Parser).expression)
}

func TestIsExpression(t *testing.T) {
	cases := testCases(
		`10 IS 10`,
		"E{Is{TTT}}",
		`10 IS DISTINCT FROM 10`,
		"E{IsDistinctFrom{TTTTT}}",
		`10 IS DISTINCT FROM NOT 10`,
		"E{IsDistinctFrom{TTTT Not{TT}}}",
		`10 IS NOT 10`,
		"E{IsNot{TTTT}}",
		`10 IS NOT NOT 10`,
		"E{IsNot{TTT Not{TT}}}",
		`10 IS NOT DISTINCT FROM 10`,
		"E{IsNotDistinctFrom{TTTTTT}}",
		`10 IS NOT DISTINCT FROM NOT 10`,
		"E{IsNotDistinctFrom{TTTTT Not{TT}}}",
	)

	runTests(t, cases, (*Parser).expression)
}

func TestIsExpressionError(t *testing.T) {
	cases := []testCaseError{
		{code: `10 IS NOT DISTINCT 10`, msg: "expecting From, got Numeric"},
		{code: `10 IS NOT DISTINCT FROM`, msg: "unexpected EOF"},
		{code: `10 IS NOT`, msg: "unexpected EOF"},
		{code: `10 IS DISTINCT FROM`, msg: "unexpected EOF"},
		{code: `10 IS DISTINCT FROM NOT`, msg: "unexpected EOF"},
		{code: `10 IS DISTINCT 10`, msg: "expecting From, got Numeric"},
		{code: `10 IS`, msg: "unexpected EOF"},
	}

	runTestsError(t, cases, (*Parser).expression)
}

func TestBetween(t *testing.T) {
	cases := testCases(
		`10 BETWEEN 5 AND 15`,
		`E{Between{TTTTT}}`,
		`10 BETWEEN NOT 5 AND NOT 15`,
		`E{Between{TT Not{TT} T Not{TT}}}`,
	)

	runTests(t, cases, (*Parser).expression)
}

func TestBetweenError(t *testing.T) {
	cases := []testCaseError{
		{code: `10 BETWEEN`, msg: "unexpected EOF"},
		{code: `10 BETWEEN AND 20`, msg: "unexpected And"},
		{code: `10 BETWEEN 10  20`, msg: "expecting And, got Numeric"},
	}

	runTestsError(t, cases, (*Parser).expression)
}

func TestNotBetween(t *testing.T) {
	cases := testCases(
		`10 NOT BETWEEN 5 AND 15`,
		"E{NotBetween{TTTTTT}}",
	)

	runTests(t, cases, (*Parser).expression)
}

func TestNotBetweenError(t *testing.T) {
	cases := []testCaseError{
		{code: `10 NOT BETWEEN`, msg: "unexpected EOF"},
		{code: `10 NOT BETWEEN AND 20`, msg: "unexpected And"},
		{code: `10 NOT BETWEEN 10  20`, msg: "expecting And, got Numeric"},
	}

	runTestsError(t, cases, (*Parser).expression)
}

func TestIn(t *testing.T) {
	cases := testCases(
		`10 IN (10, NOT 20)`, "E{In{TTT CommaList{E{T} T E{Not{TT}}} T}}",
		`10 IN ()`, "E{In{TTTT}}",
		`10 IN (SELECT 10)`, "E{In{TTT SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}} T}}",
		`10 IN (WITH table_name_a AS (SELECT 10) SELECT 10)`,
		`E{In{TTT SimpleSelect{
				WithClause{T CommaList{CTE{TableName TT SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}} T}}}
				SelectCore{T CommaList{ResultColumn{E{T}}}}} T}}`,
		`10 IN schema_name.table_name`, "E{In{TT SchemaName T TableName}}",
		`10 IN schema_a.table_function_a(10, 20)`, "E{In{TT SchemaName T TableFunctionName T CommaList{TTT} T}}",
		`10 IN MAX()`, "E{In{TT TableFunctionName TT}}",
	)

	runTests(t, cases, (*Parser).expression)
}

func TestInError(t *testing.T) {
	cases := []testCaseError{
		{code: `10 IN (ALTER)`, msg: "expecting RightParen, got Alter"},
		{code: `10 IN (1`, msg: "expecting RightParen, got EOF"},
		{code: `10 IN function(10`, msg: "expecting RightParen, got EOF"},
		{code: `10 IN ALTER`, msg: "expecting Identifier, got Alter"},
		{code: `10 IN (SELECT)`, msg: "unexpected RightParen"},
	}

	runTestsError(t, cases, (*Parser).expression)
}

func TestNotIn(t *testing.T) {
	cases := testCases(
		`10 NOT IN (10, NOT 20)`, "E{NotIn{TTTT CommaList{E{T} T E{Not{TT}}} T}}",
		`10 NOT IN ()`, "E{NotIn{TTTTT}}",
		`10 NOT IN (SELECT 10)`, "E{NotIn{TTTT SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}} T}}",
		`10 NOT IN (WITH table_name_a AS (SELECT 10) SELECT 10)`,
		`E{NotIn{TTTT SimpleSelect{
				WithClause{T CommaList{CTE{TableName TT SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}} T}}}
				SelectCore{T CommaList{ResultColumn{E{T}}}}} T}}`,
		`10 NOT IN schema_name.table_name`, "E{NotIn{TTT SchemaName T TableName}}",
		`10 NOT IN schema_a.table_function_a(10, 20)`, "E{NotIn{TTT SchemaName T TableFunctionName T CommaList{TTT} T}}",
		`10 NOT IN MAX()`, "E{NotIn{TTT TableFunctionName TT}}",
		`10 NOT IN table_a`, "E{NotIn{TTT TableName}}",
	)

	runTests(t, cases, (*Parser).expression)
}

func TestMatch(t *testing.T) {
	cases := testCases(
		`10 MATCH 10`, "E{Match{TTT}}",
		`10 MATCH NOT 10`, "E{Match{TT Not{TT}}}",
	)

	runTests(t, cases, (*Parser).expression)
}

func TestMatchError(t *testing.T) {
	cases := []testCaseError{
		{code: `10 MATCH`, msg: "unexpected EOF"},
	}

	runTestsError(t, cases, (*Parser).expression)
}

func TestNotMatch(t *testing.T) {
	cases := testCases(
		`10 NOT MATCH 10`, "E{NotMatch{TTTT}}",
		`10 NOT MATCH NOT 10`, "E{NotMatch{TTT Not{TT}}}",
	)

	runTests(t, cases, (*Parser).expression)
}

func TestNotMatchError(t *testing.T) {
	cases := []testCaseError{
		{code: `10 NOT MATCH`, msg: "unexpected EOF"},
	}

	runTestsError(t, cases, (*Parser).expression)
}

func TestNotInError(t *testing.T) {
	cases := []testCaseError{
		{code: `10 NOT IN (SELECT)`, msg: "unexpected RightParen"},
		{code: `10 NOT IN (ALTER)`, msg: "expecting RightParen, got Alter"},
		{code: `10 NOT IN (1`, msg: "expecting RightParen, got EOF"},
		{code: `10 NOT IN MAX(10`, msg: "expecting RightParen, got EOF"},
		{code: `10 NOT IN ALTER`, msg: "expecting Identifier, got Alter"},
	}

	runTestsError(t, cases, (*Parser).expression)
}

func TestIsStartOfExpressionAtLeast4(t *testing.T) {
	cases := []struct {
		tok    string
		result bool
	}{
		{tok: "10", result: true},
		{tok: "?", result: true},
		{tok: "a", result: true},
		{tok: "NOT", result: false},
	}

	for _, c := range cases {
		p := New(lexer.New([]byte(c.tok)))
		p.advance()
		p.advance()
		p.advance()
		p.advance()
		if p.isStartOfExpressionAtLeast4() != c.result {
			t.Logf("isStartOfExpressionAtLeast4(%s) == %v, want %v", c.tok, !c.result, c.result)
			t.Fail()
		}
	}
}

func TestExpression5(t *testing.T) {
	cases := testCases(
		`a < c`, "E{LessThan{ColRef{ColName} T ColRef{ColName}}}",
		`a <= c`, "E{LessThanOrEqual{ColRef{ColName} T ColRef{ColName}}}",
		`a > c`, "E{GreaterThan{ColRef{ColName} T ColRef{ColName}}}",
		`a >= c`, "E{GreaterThanOrEqual{ColRef{ColName} T ColRef{ColName}}}",
	)

	runTests(t, cases, (*Parser).expression)
}

func TestExpression5Error(t *testing.T) {
	cases := []testCaseError{
		{code: `10 <`, msg: "unexpected EOF"},
	}

	runTestsError(t, cases, (*Parser).expression)
}

func TestExpression6(t *testing.T) {
	cases := testCases(
		`a & c`, "E{BitAnd{ColRef{ColName} T ColRef{ColName}}}",
		`a | c`, "E{BitOr{ColRef{ColName} T ColRef{ColName}}}",
		`a << c`, "E{LeftShift{ColRef{ColName} T ColRef{ColName}}}",
		`a >> c`, "E{RightShift{ColRef{ColName} T ColRef{ColName}}}",
	)

	runTests(t, cases, (*Parser).expression)
}

func TestExpression6Error(t *testing.T) {
	cases := []testCaseError{
		{code: `10 &`, msg: "unexpected EOF"},
	}

	runTestsError(t, cases, (*Parser).expression)
}

func TestExpression7(t *testing.T) {
	cases := testCases(
		`a + c`, "E{Add{ColRef{ColName} T ColRef{ColName}}}",
		`a - c`, "E{Subtract{ColRef{ColName} T ColRef{ColName}}}",
	)

	runTests(t, cases, (*Parser).expression)
}

func TestExpression7Error(t *testing.T) {
	cases := []testCaseError{
		{code: `10 +`, msg: "unexpected EOF"},
	}

	runTestsError(t, cases, (*Parser).expression)
}

func TestExpression8(t *testing.T) {
	cases := testCases(
		`a * c`, "E{Multiply{ColRef{ColName} T ColRef{ColName}}}",
		`a / c`, "E{Divide{ColRef{ColName} T ColRef{ColName}}}",
		`a % c`, "E{Mod{ColRef{ColName} T ColRef{ColName}}}",
	)

	runTests(t, cases, (*Parser).expression)
}

func TestExpression8Error(t *testing.T) {
	cases := []testCaseError{
		{code: `10 *`, msg: "unexpected EOF"},
	}

	runTestsError(t, cases, (*Parser).expression)
}

func TestExpression9(t *testing.T) {
	cases := testCases(
		`a || c`, "E{Concatenate{ColRef{ColName} T ColRef{ColName}}}",
		`a -> c`, "E{Extract1{ColRef{ColName} T ColRef{ColName}}}",
		`a ->> c`, "E{Extract2{ColRef{ColName} T ColRef{ColName}}}",
	)

	runTests(t, cases, (*Parser).expression)
}

func TestExpression9Error(t *testing.T) {
	cases := []testCaseError{
		{code: `10 ||`, msg: "unexpected EOF"},
	}

	runTestsError(t, cases, (*Parser).expression)
}

func TestExpression10(t *testing.T) {
	cases := testCases(
		`a`, "E{ColRef{ColName}}",
		`a COLLATE c`, "E{Collate{ColRef{ColName} T CollationName}}",
	)

	runTests(t, cases, (*Parser).expression)
}

func TestExpression10Error(t *testing.T) {
	cases := []testCaseError{
		{code: `10 COLLATE`, msg: "expecting Identifier, got EOF"},
	}

	runTestsError(t, cases, (*Parser).expression)
}

func TestExpression11(t *testing.T) {
	cases := testCases(
		`~a`, "E{BitNot{T ColRef{ColName}}}",
		`+a`, "E{PrefixPlus{T ColRef{ColName}}}",
		`-a`, "E{Negate{T ColRef{ColName}}}",
	)

	runTests(t, cases, (*Parser).expression)
}

func TestExpression11Error(t *testing.T) {
	cases := []testCaseError{
		{code: `~`, msg: "unexpected EOF"},
	}

	runTestsError(t, cases, (*Parser).expression)
}

func TestSimpleExpression(t *testing.T) {
	cases := testCases(
		`a`, "E{ColRef{ColName}}",
		`?`, "E{BindParameter}",
		`function_name()`, "E{FunctionCall{FunctionName T FnArgs{} T}}",
		`(10)`, "E{ParenE{T CommaList{E{T}} T}}",
		`(SELECT 10)`, "E{ParenSelect{T SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}} T}}",
		`(WITH go AS (SELECT rust) SELECT 10)`,
		`E{ParenSelect{T SimpleSelect{
			WithClause{T CommaList{CTE{TableName T T SimpleSelect{SelectCore{T CommaList{ResultColumn{E{ColRef{ColName}}}}}} T }}}
			SelectCore{T CommaList{ResultColumn{E{T}}}}} T}}`,
		`CAST ('10' AS INTEGER)`, "E{Cast{TT E{T} T TypeName{T} T}}",
		`NOT EXISTS (SELECT 10)`, "E{Not{T Exists{T T SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}} T}}}",
		`EXISTS (SELECT 10)`, "E{Exists{T T SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}} T}}",
		`CASE WHEN TRUE THEN 10 END`, "E{Case{T When{T E{T} T E{T}} T}}",
		`RAISE (IGNORE)`, "E{Raise{TTTT}}",
	)

	runTests(t, cases, (*Parser).expression)
}

func TestIsStartOfExpression(t *testing.T) {
	cases := []struct {
		tok    string
		result bool
	}{
		{tok: "10", result: true},
		{tok: "?", result: true},
		{tok: "a", result: true},
		{tok: "TEMP", result: true},
		{tok: "+", result: true},
		{tok: "-", result: true},
		{tok: "~", result: true},
		{tok: "*", result: false},
		{tok: "(", result: true},
		{tok: "CAST", result: true},
		{tok: "NOT", result: true},
		{tok: "EXISTS", result: true},
		{tok: "RAISE", result: true},
		{tok: "SELECT", result: false},
		{tok: ",", result: false},
		{tok: "LIKE", result: false},
	}

	for _, c := range cases {
		p := New(lexer.New([]byte(c.tok)))
		p.advance()
		p.advance()
		p.advance()
		p.advance()
		if p.isStartOfExpression() != c.result {
			t.Logf("isStartOfExpression(%s) == %v, want %v", c.tok, !c.result, c.result)
			t.Fail()
		}
	}
}

func TestIsLiteralValue(t *testing.T) {
	cases := []struct {
		tok    string
		result bool
	}{
		{tok: "NULL", result: true},
		{tok: "CURRENT_TIME", result: true},
		{tok: "TRUE", result: true},
		{tok: "SELECT", result: false},
	}

	for _, c := range cases {
		p := New(lexer.New([]byte(c.tok)))
		p.advance()
		p.advance()
		p.advance()
		p.advance()
		if p.isLiteralValue() != c.result {
			t.Logf("isLiteralValue(%s) == %v, want %v", c.tok, !c.result, c.result)
			t.Fail()
		}
	}
}

func TestIsBindParameter(t *testing.T) {
	cases := []struct {
		tok    string
		result bool
	}{
		{tok: "@a", result: true},
		{tok: ":a", result: true},
		{tok: "$1", result: true},
		{tok: "SELECT", result: false},
	}

	for _, c := range cases {
		p := New(lexer.New([]byte(c.tok)))
		p.advance()
		p.advance()
		p.advance()
		p.advance()
		if is := p.isBindParameter(); is != c.result {
			t.Logf("isBindParameter(%s) == %v, want %v", c.tok, is, c.result)
			t.Fail()
		}
	}
}

func TestColumnReference(t *testing.T) {
	cases := testCases(
		`a`, "E{ColRef{ColName}}",
		`TEMP`, "E{ColRef{ColName}}",
		`a.a`, "E{ColRef{TableName T ColName}}",
		`TEMP.TEMP`, "E{ColRef{TableName T ColName}}",
		`a.a.a`, "E{ColRef{SchemaName T TableName T ColName}}",
		`TEMP.TEMP.TEMP`, "E{ColRef{SchemaName T TableName T ColName}}",
	)

	runTests(t, cases, (*Parser).expression)
}

func TestColumnReferenceError(t *testing.T) {
	cases := []testCaseError{
		{code: `a.`, msg: "expecting [Identifier Temp], got EOF"},
		{code: `TEMP.`, msg: "expecting [Identifier Temp], got EOF"},
		{code: `a..a`, msg: "expecting [Identifier Temp], got Dot"},
		{code: `TEMP..TEMP`, msg: "expecting [Identifier Temp], got Dot"},
	}

	runTestsError(t, cases, (*Parser).expression)
}

func TestFunctionCall(t *testing.T) {
	cases := testCases(
		`MAX()`,
		"E{FnCall{FnName T FnArgs{} T}}",
		`MAX('a')`,
		"E{FnCall{FnName T FnArgs{CommaList{E{T}}} T}}",
		`MAX('a') OVER window_name`,
		"E{FnCall{FnName T FnArgs{CommaList{E{T}}} T OverClause{T WindowName}}}",
	)

	runTests(t, cases, (*Parser).expression)
}

func TestFunctionCallError(t *testing.T) {
	cases := []testCaseError{
		{code: `MAX(10 ORDER a);`, msg: "expecting By, got Identifier"},
		{code: `MAX(10 ORDER BY);`, msg: "unexpected RightParen"},
		{code: `MAX(10 ORDER BY a NULLS);`, msg: "expecting [First Last], got RightParen"},
		{code: `MAX(;`, msg: "expecting RightParen, got Semicolon"},
		{code: `MAX(DISTINCT );`, msg: "unexpected RightParen"},
		{code: `MAX() FILTER WHERE a);`, msg: "expecting LeftParen, got Where"},
		{code: `MAX() FILTER (a);`, msg: "expecting Where, got Identifier"},
		{code: `MAX(10 ORDER BY a COLLATE);`, msg: "expecting Identifier, got RightParen"},
	}

	runTestsError(t, cases, (*Parser).expression)
}

func TestFunctionArguments(t *testing.T) {
	cases := testCases(
		`MAX('a')`, "E{FunctionCall{FunctionName T FnArgs{CommaList{E{T}}} T}}",
		`MAX(DISTINCT 'a')`, "E{FunctionCall{FunctionName T FnArgs{T CommaList{E{T}}} T}}",
		`MAX(DISTINCT 'a' ORDER BY a)`,
		`E{FunctionCall{FunctionName T
			FnArgs{T CommaList{E{T}} OrderByClause{TT CommaList{OrderingTerm{E{ColRef{ColName}}}}}} T}}`,
		`MAX('a' ORDER BY a)`,
		`E{FunctionCall{FunctionName T
			FnArgs{CommaList{E{T}} OrderByClause{TT CommaList{OrderingTerm{E{ColRef{ColName}}}}}} T}}`,
		`MAX(*)`, "E{FunctionCall{FunctionName T FnArgs{T} T}}",
		`MAX()`, "E{FunctionCall{FunctionName T FnArgs{} T}}",
	)

	runTests(t, cases, (*Parser).expression)
}

func TestFunctionArgumentsError(t *testing.T) {
	cases := []testCaseError{
		{code: `MAX(DISTINCT)`, msg: "unexpected RightParen"},
		{code: `MAX(ORDER BY a)`, msg: "expecting RightParen, got Order"},
		{code: `MAX(DISTINCT *)`, msg: "unexpected Asterisk"},
		{code: `MAX(* ORDER BY 10)`, msg: "expecting RightParen, got Order"},
	}

	runTestsError(t, cases, (*Parser).expression)
}

func TestOrderByClause(t *testing.T) {
	cases := testCases(
		`MAX() OVER (ORDER BY a COLLATE c)`,
		`E{FunctionCall{FunctionName T FunctionArguments{} T OverClause{T WindowDefinition{T
			OrderByClause{TT CommaList{OrderingTerm{E{Collate{ColRef{ColName} T CollationName}}}}} T}}}}`,
	)

	runTests(t, cases, (*Parser).expression)
}

func TestOrderByClauseError(t *testing.T) {
	cases := []testCaseError{
		{code: `MAX() OVER (ORDER a)`, msg: "expecting By, got Identifier"},
		{code: `MAX() OVER (ORDER BY)`, msg: "unexpected RightParen"},
	}

	runTestsError(t, cases, (*Parser).expression)
}

func TestOrderingTerm(t *testing.T) {
	cases := testCases(
		`MAX() OVER (ORDER BY a ASC NULLS LAST)`,
		`E{FunctionCall{FunctionName T FunctionArguments{} T 
				OverClause{T WindowDefinition{T OrderByClause{TT CommaList{OrderingTerm{E{ColRef{ColName}} TTT}}} T}} }}`,
		`MAX() OVER (ORDER BY a DESC NULLS FIRST)`,
		`E{FunctionCall{FunctionName T FunctionArguments{} T 
				OverClause{T WindowDefinition{T OrderByClause{TT CommaList{OrderingTerm{E{ColRef{ColName}} TTT}}} T}} }}`,
	)

	runTests(t, cases, (*Parser).expression)
}

func TestOrderingTermError(t *testing.T) {
	cases := []testCaseError{
		{code: `MAX() OVER (ORDER BY a DESC NULLS)`, msg: "expecting [First Last], got RightParen"},
	}

	runTestsError(t, cases, (*Parser).expression)
}

func TestFilterClause(t *testing.T) {
	cases := testCases(
		`MAX() FILTER (WHERE a + b)`,
		`E{FunctionCall{FunctionName T FunctionArguments{} T 
				FilterClause{TT T E{Add{ColRef{ColName} T ColRef{ColName}}} T} }}`,
	)

	runTests(t, cases, (*Parser).expression)
}

func TestFilterClauseError(t *testing.T) {
	cases := []testCaseError{
		{code: `MAX() FILTER WHERE a)`, msg: "expecting LeftParen, got Where"},
		{code: `MAX() FILTER (a)`, msg: "expecting Where, got Identifier"},
		{code: `MAX() FILTER (WHERE)`, msg: "unexpected RightParen"},
		{code: `MAX() FILTER (WHERE a`, msg: "expecting RightParen, got EOF"},
	}

	runTestsError(t, cases, (*Parser).expression)
}

func TestOverClause(t *testing.T) {
	cases := testCases(
		`MAX() OVER window_a`,
		`E{FunctionCall{FunctionName T FunctionArguments{} T 
				OverClause{T WindowName} }}`,
		`MAX() OVER ()`,
		`E{FunctionCall{FunctionName T FunctionArguments{} T 
				OverClause{T WindowDefinition{TT}} }}`,
		`MAX() OVER (window_a PARTITION BY a, b));`,
		`E{FunctionCall{FunctionName T FunctionArguments{} T 
				OverClause{T WindowDefinition{T WindowName PartitionBy{TT CommaList{E{ColRef{ColName}} T E{ColRef{ColName}}}} T}} }}`,
		`MAX() OVER (ORDER BY a)`,
		`E{FunctionCall{FunctionName T FunctionArguments{} T 
				OverClause{T WindowDefinition{T OrderByClause{TT CommaList{OrderingTerm{E{ColRef{ColName}}}}} T}} }}`,
		`MAX() OVER (RANGE CURRENT ROW)`,
		`E{FunctionCall{FunctionName T FunctionArguments{} T 
				OverClause{T WindowDefinition{T FrameSpec{TTT} T}} }}`,
	)

	runTests(t, cases, (*Parser).expression)
}

func TestOverClauseError(t *testing.T) {
	cases := []testCaseError{
		{code: `MAX() OVER ORDER BY a)`, msg: "expecting LeftParen, got Order"},
		{code: `MAX() OVER (window_a PARTITION a));`, msg: "expecting By, got Identifier"},
		{code: `MAX() OVER (window_a PARTITION BY));`, msg: "unexpected RightParen"},
		{code: `MAX() OVER (RANGE CURRENT ROW`, msg: "expecting RightParen, got EOF"},
		{code: `MAX() OVER`, msg: "expecting LeftParen, got EOF"},
	}

	runTestsError(t, cases, (*Parser).expression)
}

func TestFrameSpec(t *testing.T) {
	cases := testCases(
		`MAX() OVER (ROWS UNBOUNDED PRECEDING EXCLUDE NO OTHERS)`,
		`E{FunctionCall{FunctionName T FunctionArguments{} T OverClause{T WindowDefinition{T
			FrameSpec{TTTTTT} T}}}}`,
		`MAX() OVER (GROUPS 10 PRECEDING EXCLUDE CURRENT ROW)`,
		`E{FunctionCall{FunctionName T FunctionArguments{} T OverClause{T WindowDefinition{T
			FrameSpec{T E{T} TTTT} T}}}}`,
		`MAX() OVER (RANGE CURRENT ROW EXCLUDE GROUP)`,
		`E{FunctionCall{FunctionName T FunctionArguments{} T OverClause{T WindowDefinition{T
			FrameSpec{TTTTT} T}}}}`,
		`MAX() OVER (ROWS BETWEEN 10 PRECEDING AND 10 FOLLOWING EXCLUDE TIES)`,
		`E{FunctionCall{FunctionName T FunctionArguments{} T OverClause{T WindowDefinition{T
			FrameSpec{T FrameSpecBetween{T E{T} TT E{T} T} TT} T}}}}`,
	)

	runTests(t, cases, (*Parser).expression)
}

func TestFrameSpecError(t *testing.T) {
	cases := []testCaseError{
		{code: `MAX() OVER (ROWS UNBOUNDED EXCLUDE NO OTHERS)`, msg: "expecting Preceding, got Exclude"},
		{code: `MAX() OVER (RANGE CURRENT EXCLUDE GROUP)`, msg: "expecting Row, got Exclude"},
		{code: `MAX() OVER (GROUPS 10 EXCLUDE CURRENT ROW)`, msg: "expecting Preceding, got Exclude"},
		{code: `MAX() OVER (ROWS)`, msg: "unexpected RightParen"},
		{code: `MAX() OVER (ROWS UNBOUNDED PRECEDING EXCLUDE NO)`, msg: "expecting Others, got RightParen"},
		{code: `MAX() OVER (GROUPS 10 PRECEDING EXCLUDE CURRENT)`, msg: "expecting Row, got RightParen"},
		{code: `MAX() OVER (ROWS UNBOUNDED PRECEDING EXCLUDE)`, msg: "expecting [No Current Group Ties], got RightParen"},
	}

	runTestsError(t, cases, (*Parser).expression)
}

func TestFrameSpecBetween(t *testing.T) {
	cases := testCases(
		`MAX() OVER (RANGE BETWEEN UNBOUNDED PRECEDING AND 10 PRECEDING)`,
		`E{FunctionCall{FunctionName T FunctionArguments{} T OverClause{T WindowDefinition{T
			FrameSpec{T FrameSpecBetween{TTTT E{T} T}} T}}}}`,
		`MAX() OVER (RANGE BETWEEN 10 PRECEDING AND CURRENT ROW)`,
		`E{FunctionCall{FunctionName T FunctionArguments{} T OverClause{T WindowDefinition{T
			FrameSpec{T FrameSpecBetween{T E{T} TTTT}} T}}}}`,
		`MAX() OVER (RANGE BETWEEN CURRENT ROW AND 10 FOLLOWING)`,
		`E{FunctionCall{FunctionName T FunctionArguments{} T OverClause{T WindowDefinition{T
			FrameSpec{T FrameSpecBetween{TTTT E{T} T}} T}}}}`,
		`MAX() OVER (RANGE BETWEEN 10 FOLLOWING AND UNBOUNDED FOLLOWING)`,
		`E{FunctionCall{FunctionName T FunctionArguments{} T OverClause{T WindowDefinition{T
			FrameSpec{T FrameSpecBetween{T E{T} TTTT}} T}}}}`,
	)

	runTests(t, cases, (*Parser).expression)
}

func TestFrameSpecBetweenError(t *testing.T) {
	cases := []testCaseError{
		{code: `MAX() OVER (RANGE BETWEEN UNBOUNDED AND 10 PRECEDING)`, msg: "expecting Preceding, got And"},
		{code: `MAX() OVER (RANGE BETWEEN CURRENT AND 10 FOLLOWING)`, msg: "expecting Row, got And"},
		{code: `MAX() OVER (RANGE BETWEEN 10 AND CURRENT ROW)`, msg: "expecting [Preceding Following], got And"},
		{code: `MAX() OVER (RANGE BETWEEN PRECEDING AND 10 PRECEDING)`, msg: "unexpected Preceding"},
		{code: `MAX() OVER (RANGE BETWEEN ROW AND 10 FOLLOWING)`, msg: "unexpected Row"},
		{code: `MAX() OVER (RANGE BETWEEN FOLLOWING AND UNBOUNDED FOLLOWING)`, msg: "unexpected Following"},
		{code: `MAX() OVER (RANGE BETWEEN AND 10 PRECEDING)`, msg: "unexpected And"},
		{code: `MAX() OVER (RANGE BETWEEN UNBOUNDED PRECEDING 10 PRECEDING)`, msg: "expecting And, got Numeric"},
		{code: `MAX() OVER (RANGE BETWEEN CURRENT ROW AND UNBOUNDED)`, msg: "expecting Following, got RightParen"},
		{code: `MAX() OVER (RANGE BETWEEN 10 PRECEDING AND CURRENT)`, msg: "expecting Row, got RightParen"},
		{code: `MAX() OVER (RANGE BETWEEN UNBOUNDED PRECEDING AND 10)`, msg: "expecting [Preceding Following], got RightParen"},
		{code: `MAX() OVER (RANGE BETWEEN UNBOUNDED PRECEDING AND)`, msg: "unexpected RightParen"},
	}

	runTestsError(t, cases, (*Parser).expression)
}

func TestParenExpression(t *testing.T) {
	cases := testCases(
		`(TRUE = FALSE OR 10 == 20)`,
		"E{ParenE{T CommaList{E{Or{Equal{TTT} T Equal{TTT}}}} T}}",
		`(TRUE > FALSE < 10)`,
		"E{ParenE{T CommaList{E{LessThan{GreaterThan{TTT} TT}}} T}}",
		`(10 + column_a)`,
		"E{ParenE{T CommaList{Expression{Add{T T ColumnReference{ColumnName}}}} T}}",
	)

	runTests(t, cases, (*Parser).expression)
}

func TestParenExpressionError(t *testing.T) {
	cases := []testCaseError{
		{code: `();`, msg: "unexpected RightParen"},
		{code: `(, 10);`, msg: "unexpected Comma"},
		{code: `(10 10);`, msg: "expecting RightParen, got Numeric"},
		{code: `(10 AS 10);`, msg: "expecting RightParen, got As"},
		{code: `(10,,10);`, msg: "unexpected Comma"},
		{code: `(10, AS);`, msg: "unexpected As"},
		{code: `(10;`, msg: "expecting RightParen, got Semicolon"},
	}

	runTestsError(t, cases, (*Parser).expression)
}

func TestCastExpression(t *testing.T) {
	cases := testCases(
		`CAST ('10' AS INTEGER)`, "E{Cast{TT E{T} T TypeName{T} T}}",
	)

	runTests(t, cases, (*Parser).expression)
}

func TestCastExpressionError(t *testing.T) {
	cases := []testCaseError{
		{code: `CAST 10 AS NUMBER);`, msg: "expecting LeftParen, got Numeric"},
		{code: `CAST AS NUMBER);`, msg: "expecting LeftParen, got As"},
		{code: `CAST(10 AS);`, msg: "expecting Identifier, got RightParen"},
		{code: `CAST(10 NUMBER);`, msg: "expecting As, got Identifier"},
		{code: `CAST(10 AS NUMBER;`, msg: "expecting RightParen, got Semicolon"},
	}

	runTestsError(t, cases, (*Parser).castExpression)
}

func TestExists(t *testing.T) {
	cases := testCases(
		`EXISTS(SELECT 10)`, "E{Exists{TT SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}} T}}",
		`EXISTS(WITH table_name_a AS (SELECT 10) SELECT 10)`,
		`E{Exists{TT SimpleSelect{
				WithClause{T CommaList{CTE{TableName TT SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}} T}}}
				SelectCore{T CommaList{ResultColumn{E{T}}}}} T}}`,
	)

	runTests(t, cases, (*Parser).expression)
}

func TestExistsError(t *testing.T) {
	cases := []testCaseError{
		{code: `EXISTS(SELECT)`, msg: "unexpected RightParen"},
		{code: `EXISTS SELECT 10);`, msg: "expecting LeftParen, got Select"},
		{code: `EXISTS(10);`, msg: "expecting Select, got Numeric"},
		{code: `EXISTS(10;`, msg: "expecting Select, got Numeric"},
		{code: `EXISTS (SELECT 10;`, msg: "expecting RightParen, got Semicolon"},
		{code: `EXISTS ();`, msg: "expecting Select, got RightParen"},
	}

	runTestsError(t, cases, (*Parser).expression)
}

func TestCaseExpression(t *testing.T) {
	cases := testCases(
		`CASE WHEN 10 THEN TRUE ELSE FALSE END`,
		"E{Case{T When {T E{T} T E{T}} Else {T E{T}} T}}",
		`CASE a WHEN 10 THEN TRUE END`,
		"E{Case{T E{ColRef{ColName}} When {T E{T} T E{T}} T}}",
		`CASE WHEN 10 THEN TRUE WHEN 20 THEN 'a' ELSE FALSE END`,
		"E{Case{T When {T E{T} T E{T}} When {T E{T} T E{T}} Else {T E{T}} T}}",
	)

	runTests(t, cases, (*Parser).expression)
}

func TestCaseExpressionError(t *testing.T) {
	cases := []testCaseError{
		{code: `CASE ELSE 10`, msg: "expecting When, got Else"},
		{code: `CASE WHEN THEN 10 END`, msg: "unexpected Then"},
		{code: `CASE WHEN 10 10 END`, msg: "expecting Then, got Numeric"},
		{code: `CASE WHEN 10 THEN END`, msg: "unexpected End"},
		{code: `CASE WHEN 10 THEN 10 ELSE END`, msg: "unexpected End"},
	}

	runTestsError(t, cases, (*Parser).expression)
}

func TestRaise(t *testing.T) {
	cases := testCases(
		`RAISE (IGNORE)`, "E{Raise{TTTT}}",
		`RAISE (ROLLBACK, 'error message')`, "E{Raise{TTTT ErrorMessage{E{T}} T}}",
	)

	runTests(t, cases, (*Parser).expression)
}

func TestRaiseError(t *testing.T) {
	cases := []testCaseError{
		{code: `RAISE;`, msg: "expecting LeftParen, got Semicolon"},
		{code: `RAISE IGNORE);`, msg: "expecting LeftParen, got Ignore"},
		{code: `RAISE(IGNORE;`, msg: "expecting RightParen, got Semicolon"},
		{code: `RAISE();`, msg: "expecting [Ignore Rollback Abort Fail], got RightParen"},
		{code: `RAISE(, 'error');`, msg: "expecting [Ignore Rollback Abort Fail], got Comma"},
		{code: `RAISE(ROLLBACK 'error');`, msg: "expecting Comma, got String"},
		{code: `RAISE(ROLLBACK, );`, msg: "unexpected RightParen"},
		{code: `RAISE(ROLLBACK, 'error';`, msg: "expecting RightParen, got Semicolon"},
	}

	runTestsError(t, cases, (*Parser).expression)
}

func TestInsert(t *testing.T) {
	cases := testCases(
		`INSERT INTO table_name(name) VALUES('Go')`,
		`Insert{TT TableName T CommaList{ColumnName} T T
			InsertValuesList{CommaList{InsertValuesItem{T CommaList{E{T}} T}}}}`,
		`REPLACE INTO table_name(name) VALUES('Go')`,
		`Insert{TT TableName T CommaList{ColumnName} T T
			InsertValuesList{CommaList{InsertValuesItem{T CommaList{E{T}} T}}}}`,
		`INSERT OR ABORT INTO temp.table_name(name) VALUES('Go')`,
		`Insert{T TT T SchemaName T TableName T CommaList{ColumnName} T T
		 	InsertValuesList{CommaList{InsertValuesItem{T CommaList{E{T}} T}}}}`,
		`INSERT OR FAIL INTO table_name AS table_alias (name) VALUES('Go')`,
		`Insert{T TT T TableName T TableAlias T CommaList{ColumnName} T T
		 	InsertValuesList{CommaList{InsertValuesItem{T CommaList{E{T}} T}}}}`,
		`INSERT INTO table_name(name) VALUES('Go') ON CONFLICT DO NOTHING`,
		`Insert{TT TableName T CommaList{ColumnName} T T
			InsertValuesList{CommaList{InsertValuesItem{T CommaList{E{T}} T}}}
			UpsertClause{UpsertClauseItem{TT TT}}}`,
		`INSERT INTO table_name(name) SELECT 'Go'`,
		"Insert{TT TableName T CommaList{ColumnName} T SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}}}",
		`INSERT INTO table_name(name) SELECT 'Go' ON CONFLICT DO NOTHING`,
		"Insert{TT TableName T CommaList{ColumnName} T SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}} UpsertClause{UpsertClauseItem{TT TT}}}",
		`INSERT INTO table_name(name) DEFAULT VALUES`,
		"Insert{TT TableName T CommaList{ColumnName} T TT}",
		`INSERT INTO table_name(name) DEFAULT VALUES RETURNING *`,
		"Insert{TT TableName T CommaList{ColumnName} T TT ReturningClause{T CommaList{ReturningItem{T}}}}",
		`INSERT INTO table_name(name) WITH cte AS (SELECT 10) SELECT 'Go'`,
		`Insert{TT TableName T CommaList{ColumnName} T SimpleSelect{
			WithClause{T CommaList{CTE{TableName T T SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}} T}} }
			SelectCore{T CommaList{ResultColumn{E{T}}}}}}`,
	)

	runTests(t, cases, func(p *Parser) parsetree.NonTerminal { return p.insert(nil) })
}

func TestInsertError(t *testing.T) {
	cases := []testCaseError{
		{code: `INSERT INTO table_name(name) WITH cte AS (SELECT 10)`, msg: "expecting Select, got EOF"},
		{code: `INSERT OR INTO table_name(name) VALUES('Go')`, msg: "expecting [Abort Fail Ignore Replace Rollback], got Into"},
		{code: `INSERT table_name(name) VALUES('Go')`, msg: "expecting Into, got Identifier"},
		{code: `INSERT INTO AS table_alias (name) VALUES('Go')`, msg: "expecting [Identifier Temp], got As"},
		{code: `INSERT INTO .table_name (name) VALUES('Go')`, msg: "expecting [Identifier Temp], got Dot"},
		{code: `INSERT INTO schema_name table_name(name) VALUES('Go')`, msg: "expecting [Values With Select Default], got Identifier"},
		{code: `INSERT OR FAIL INTO table_name AS (name) VALUES('Go')`, msg: "expecting Identifier, got LeftParen"},
		{code: `INSERT INTO table_name(name VALUES('Go')`, msg: "expecting RightParen, got Values"},
		{code: `INSERT INTO temp.table_name name) VALUES('Go')`, msg: "expecting [Values With Select Default], got Identifier"},
		{code: `INSERT INTO table_name(name VALUES('Go')`, msg: "expecting RightParen, got Values"},
		{code: `INSERT INTO table_name() VALUES('Go')`, msg: "expecting Identifier, got RightParen"},
		{code: `INSERT INTO table_name(name) VALUES('Go'), 'Rust')`, msg: "expecting LeftParen, got String"},
		{code: `INSERT INTO table_name(name) VALUES`, msg: "expecting LeftParen, got EOF"},
		{code: `INSERT INTO table_name(name) DEFAULT`, msg: "expecting Values, got EOF"},
	}

	runTestsError(t, cases, func(p *Parser) parsetree.NonTerminal { return p.insert(nil) })
}

func TestUpsertClause(t *testing.T) {
	cases := testCases(
		`ON CONFLICT DO NOTHING ON CONFLICT DO NOTHING`,
		"UpsertClause{UpsertClauseItem{TT TT} UpsertClauseItem{TT TT}}",
	)

	runTests(t, cases, (*Parser).upsertClause)
}

func TestUpsertClauseItem(t *testing.T) {
	cases := testCases(
		`ON CONFLICT DO NOTHING`,
		"UpsertClauseItem{TT TT}",
		`ON CONFLICT DO UPDATE SET col='value'`,
		"UpsertClauseItem{TT TT T CommaList{UpdateSetItem{ColName T E{T}}}}",
		`ON CONFLICT DO UPDATE SET col_a='value_a', col_b='value_b'`,
		"UpsertClauseItem{TT TT T CommaList{UpdateSetItem{ColName T E{T}} T UpdateSetItem{ColName T E{T}}}}",
		`ON CONFLICT DO UPDATE SET (col_a, col_b)='value_ab', col_c='value_c'`,
		"UpsertClauseItem{TT TT T CommaList{UpdateSetItem{T CommaList{ColName T ColName} T T E{T}} T UpdateSetItem{ColName T E{T}}} }",
		`ON CONFLICT DO UPDATE SET col='value' WHERE condition=true`,
		"UpsertClauseItem{TT TT T CommaList{UpdateSetItem{ColName T E{T}}} WhereClause{T E{Equal{ColRef{ColName} TT}}}}} }",
		`ON CONFLICT (col) DO NOTHING`,
		"UpsertClauseItem{TT T CommaList{IndexedColumn{E{ColRef{ColName}}}} T TT}",
		`ON CONFLICT (col_a, col_b) WHERE condition=true DO NOTHING`,
		`UpsertClauseItem{TT T CommaList{IndexedColumn{E{ColRef{ColName}}} T IndexedColumn{E{ColRef{ColName}}}} T
			WhereClause{T E{Equal{ColRef{ColName} T T}}} TT}`,
	)

	runTests(t, cases, (*Parser).upsertClauseItem)
}

func TestUpsertClauseItemError(t *testing.T) {
	cases := []testCaseError{
		{code: `ON CONFLICT DO UPDATE SET (col_a, col_b ='value_ab', col_c='value_c'`, msg: "expecting RightParen, got Equal"},
		{code: `ON CONFLICT DO UPDATE SET ()='value_ab', col_c='value_c'`, msg: "expecting Identifier, got RightParen"},
		{code: `ON CONFLICT DO UPDATE col='value'`, msg: "expecting Set, got Identifier"},
		{code: `ON CONFLICT DO UPDATE SET col='value' WHERE`, msg: "unexpected EOF"},
		{code: `ON CONFLICT DO UPDATE SET WHERE condition=true`, msg: "expecting [Identifier LeftParen], got Where"},
		{code: `ON CONFLICT DO UPDATE SET col 'value'`, msg: "expecting Equal, got String"},
		{code: `ON CONFLICT DO UPDATE SET col=`, msg: "unexpected EOF"},
		{code: `ON DO NOTHING`, msg: "expecting Conflict, got Do"},
		{code: `ON CONFLICT NOTHING`, msg: "expecting Do, got Nothing"},
		{code: `ON CONFLICT DO`, msg: "expecting [Nothing Update], got EOF"},
		{code: `ON CONFLICT (col DO NOTHING`, msg: "expecting RightParen, got Do"},
		{code: `ON CONFLICT (col_a) WHERE DO NOTHING`, msg: "unexpected Do"},
	}

	runTestsError(t, cases, (*Parser).upsertClauseItem)
}

func TestPragma(t *testing.T) {
	cases := testCases(
		`PRAGMA pragma_name`, "Pragma{T PragmaName}",
		`PRAGMA temp.pragma_name`, "Pragma{T SchemaName T PragmaName}",
		`PRAGMA pragma_name=10`, "Pragma{T PragmaName T PragmaValue{T}}",
		`PRAGMA pragma_name=-10`, "Pragma{T PragmaName T PragmaValue{TT}}",
		`PRAGMA pragma_name=+SELECT`, "Pragma{T PragmaName T PragmaValue{TT}}",
		`PRAGMA pragma_name='yes'`, "Pragma{T PragmaName T PragmaValue{T}}",
		`PRAGMA pragma_name('yes')`, "Pragma{T PragmaName T PragmaValue{T} T}",
	)

	runTests(t, cases, (*Parser).pragma)
}

func TestPragmaError(t *testing.T) {
	cases := []testCaseError{
		{code: `PRAGMA .pragma_name`, msg: "expecting [Identifier Temp], got Dot"},
		{code: `PRAGMA schema_name.`, msg: "expecting [Identifier Temp], got EOF"},
		{code: `PRAGMA pragma_name=`, msg: "unexpected EOF"},
		{code: `PRAGMA pragma_name=+`, msg: "unexpected EOF"},
		{code: `PRAGMA pragma_name()`, msg: "unexpected RightParen"},
		{code: `PRAGMA pragma_name(10`, msg: "expecting RightParen, got EOF"},
	}

	runTestsError(t, cases, (*Parser).pragma)
}

func TestReindex(t *testing.T) {
	cases := testCases(
		`REINDEX`, "Reindex{T}",
		`REINDEX table_name`, "Reindex{T CollationTableOrIndexName}",
		`REINDEX temp.table_name`, "Reindex{T SchemaName T TableOrIndexName}",
	)

	runTests(t, cases, (*Parser).reindex)
}

func TestReindexError(t *testing.T) {
	cases := []testCaseError{
		{code: `REINDEX schema_name.`, msg: "expecting Identifier, got EOF"},
	}

	runTestsError(t, cases, (*Parser).reindex)
}

func TestRelease(t *testing.T) {
	cases := testCases(
		`RELEASE savepoint_name`, "Release{T SavepointName}",
		`RELEASE SAVEPOINT savepoint_name`, "Release{TT SavepointName}",
	)

	runTests(t, cases, (*Parser).release)
}

func TestReleaseError(t *testing.T) {
	cases := []testCaseError{
		{code: `RELEASE`, msg: "expecting Identifier, got EOF"},
		{code: `RELEASE SAVEPOINT`, msg: "expecting Identifier, got EOF"},
	}

	runTestsError(t, cases, (*Parser).release)
}

func TestSavepoint(t *testing.T) {
	cases := testCases(
		`SAVEPOINT savepoint_name`, "Savepoint{T SavepointName}",
	)

	runTests(t, cases, (*Parser).savepoint)
}

func TestSavepointError(t *testing.T) {
	cases := []testCaseError{
		{code: `SAVEPOINT`, msg: "expecting Identifier, got EOF"},
	}

	runTestsError(t, cases, (*Parser).savepoint)
}

func TestSelectStatement(t *testing.T) {
	cases := testCases(
		`SELECT 10`, "SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}}",
		`SELECT 10 ORDER BY 20`, `SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}
			OrderByClause{TT CommaList{OrderingTerm{E{T}}}}}`,
		`SELECT 10 LIMIT 10`, "SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}} LimitClause{T E{T}}}",
		`SELECT 10 ORDER BY 20 LIMIT 10`, `SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}
			OrderByClause{TT CommaList{OrderingTerm{E{T}}}} LimitClause{T E{T}} }`,
		`SELECT 10 UNION ALL SELECT 20`,
		`CompoundSelect{SelectCore{T CommaList{ResultColumn{E{T}}}} CompoundOperator{TT} 
			SelectCore{T CommaList{ResultColumn{E{T}}}} }`,
	)

	runTests(t, cases, func(p *Parser) parsetree.NonTerminal { return p.selectStatement(nil) })
}

func TestSelectStatementError(t *testing.T) {
	cases := []testCaseError{
		{code: `SELECT 10 UNION`, msg: "expecting Select, got EOF"},
		{code: `SELECT 10, LIMIT 10`, msg: "unexpected Limit"},
		{code: `SELECT 10 LIMIT`, msg: "unexpected EOF"},
	}

	runTestsError(t, cases, func(p *Parser) parsetree.NonTerminal { return p.selectStatement(nil) })
}

func TestSelectStatementCTE(t *testing.T) {
	cases := testCases(
		`WITH table_name AS (SELECT 10) SELECT 10`,
		`SimpleSelect{WithClause{T CommaList{CTE{TableName TT
			SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}} T}}}
			SelectCore{T CommaList{ResultColumn{E{T}}}}}`,
		`WITH table_name AS (SELECT 10) SELECT 10 INTERSECT SELECT 20`,
		`CompoundSelect{WithClause{T CommaList{CTE{TableName TT SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}} T}}}
			SelectCore{T CommaList{ResultColumn{E{T}}}} CompoundOperator{T} SelectCore{T CommaList{ResultColumn{E{T}}}}}`,
	)

	for i, c := range cases {
		c := c
		t.Run(strconv.FormatInt(int64(i), 10), func(t *testing.T) {
			tp := newTestParser(newTestLexer(c.tree))
			expected := tp.tree()

			p := New(lexer.New([]byte(c.code)))
			p.comments = make(map[*token.Token][]*token.Token)
			p.advance()
			p.advance()
			p.advance()
			p.advance()

			w := p.withClause()
			parsed := p.selectStatement(w)

			if str, equals := compare(c.code, p.comments, parsed, expected); !equals {
				fmt.Println(c.code)
				fmt.Println(str)
				t.Fail()
			}
		})
	}
}

func TestSelectCore(t *testing.T) {
	cases := testCases(
		`SELECT 10`, "SelectCore{T CommaList{ResultColumn{E{T}}}}",
		`SELECT ALL 10, 20`, "SelectCore{T T CommaList{ResultColumn{E{T}} T ResultColumn{E{T}}}}",
		`SELECT DISTINCT 10`, "SelectCore{T T CommaList{ResultColumn{E{T}}}}",
		`SELECT 10 FROM table_name`, "SelectCore{T CommaList{ResultColumn{E{T}}} FromClause{T JoinClause{TableOrSubquery{TableName}}}}",
		`SELECT 10 WHERE FALSE`, "SelectCore{T CommaList{ResultColumn{E{T}}} WhereClause{T E{T}}}",
		`SELECT 10 GROUP BY 20`, "SelectCore{T CommaList{ResultColumn{E{T}}} GroupByClause{TT CommaList{E{T}}}}",
		`SELECT 10 HAVING 20`, "SelectCore{T CommaList{ResultColumn{E{T}}} HavingClause{T E{T}}}",
		`SELECT 10 WINDOW window_a AS (PARTITION BY 10)`,
		`SelectCore{T CommaList{ResultColumn{E{T}}}
			WindowClause{T CommaList{WindowClauseItem{WindowName T WindowDefinition{T PartitionBy{TT CommaList{E{T}}} T}}}} }`,
		`SELECT 10 WINDOW window_a AS (PARTITION BY 10), window_b AS (PARTITION BY 10)`,
		`SelectCore{T CommaList{ResultColumn{E{T}}}
			WindowClause{T CommaList{
				WindowClauseItem{WindowName T WindowDefinition{T PartitionBy{TT CommaList{E{T}}} T}} T
				WindowClauseItem{WindowName T WindowDefinition{T PartitionBy{TT CommaList{E{T}}} T}} }} }`,
		`VALUES (10), (20)`, `SelectCore{ValuesClause{T CommaList{ValuesItem{T CommaList{E{T}} T} T ValuesItem{T CommaList{E{T}} T}}}}`,
	)

	runTests(t, cases, (*Parser).selectCore)
}

func TestSelectCoreError(t *testing.T) {
	cases := []testCaseError{
		{code: `SELECT`, msg: "unexpected EOF"},
	}

	runTestsError(t, cases, (*Parser).selectCore)
}

func TestResultColumn(t *testing.T) {
	cases := testCases(
		`col`, "ResultColumn{E{ColRef{ColName}}}",
		`col col_alias`, "ResultColumn{E{ColRef{ColName}} ColumnAlias}",
		`col AS alias`, "ResultColumn{E{ColRef{ColName}} T ColumnAlias}",
		`table_name.*`, "ResultColumn{TableName T T}",
		`*`, "ResultColumn{T}",
	)

	runTests(t, cases, (*Parser).resultColumn)
}

func TestResultColumnError(t *testing.T) {
	cases := []testCaseError{
		{code: `col AS`, msg: "expecting Identifier, got EOF"},
		{code: `insert`, msg: "unexpected Insert"},
	}

	runTestsError(t, cases, (*Parser).resultColumn)
}

func TestFromClause(t *testing.T) {
	cases := testCases(
		`FROM table_name`, "FromClause{T JoinClause{TableOrSubquery{TableName}}}",
	)

	runTests(t, cases, (*Parser).fromClause)
}

func TestFromClauseError(t *testing.T) {
	cases := []testCaseError{
		{code: `FROM 10`, msg: "expecting LeftParen, got Numeric"},
	}

	runTestsError(t, cases, (*Parser).fromClause)
}

func TestJoinClause(t *testing.T) {
	cases := testCases(
		`table_name`, "JoinClause{TableOrSubquery{TableName}}",
		`table_name_1, table_name_2`, "JoinClause{TableOrSubquery{TableName} JoinOp{T} TableOrSubquery{TableName}}",
		`table_name_1 JOIN table_name_2`, "JoinClause{TableOrSubquery{TableName} JoinOp{T} TableOrSubquery{TableName}}",
		`table_name_1 FULL JOIN table_name_2 ON TRUE`,
		"JoinClause{TableOrSubquery{TableName} JoinOp{TT} TableOrSubquery{TableName} JoinConstr{T E{T}}}",
		`table_name_1 JOIN table_name_2, table_name_2 ON a > 2`,
		`JoinClause{TableOrSubquery{TableName} JoinOp{T} TableOrSubquery{TableName}
			JoinOp{T} TableOrSubquery{TableName} JoinConstr{T E{GreaterThan{ColRef{ColName} TT}}} }`,
	)

	runTests(t, cases, (*Parser).joinClause)
}

func TestJoinClauseError(t *testing.T) {
	cases := []testCaseError{
		{code: `table_name JOIN`, msg: "expecting LeftParen, got EOF"},
	}

	runTestsError(t, cases, (*Parser).joinClause)
}

func TestJoinOperator(t *testing.T) {
	cases := testCases(
		`,`, "JoinOp{T}",
		`JOIN`, "JoinOp{T}",
		`CROSS JOIN`, "JoinOp{TT}",
		`FULL JOIN`, "JoinOp{TT}",
		`INNER JOIN`, "JoinOp{TT}",
		`LEFT NATURAL JOIN`, "JoinOp{TTT}",
		`OUTER RIGHT JOIN`, "JoinOp{TTT}",
		`INNER OUTER JOIN`, "JoinOp{TTT}",
		`LEFT NATURAL OUTER JOIN`, "JoinOp{TTTT}",
		`FULL CROSS OUTER JOIN`, "JoinOp{TTTT}",
	)

	runTests(t, cases, (*Parser).joinOperator)
}

func TestJoinOperatorError(t *testing.T) {
	cases := []testCaseError{
		{code: `CROSS`, msg: "expecting Join, got EOF"},
		{code: `FULL`, msg: "expecting Join, got EOF"},
		{code: `INNER`, msg: "expecting Join, got EOF"},
		{code: `LEFT`, msg: "expecting Join, got EOF"},
		{code: `OUTER`, msg: "expecting Join, got EOF"},
		{code: `RIGHT`, msg: "expecting Join, got EOF"},
		{code: `NATURAL`, msg: "expecting Join, got EOF"},
	}

	runTestsError(t, cases, (*Parser).joinOperator)
}

func TestJoinConstraint(t *testing.T) {
	cases := testCases(
		`ON true`, "JoinConstr{T E{T}}",
		`USING (col_a, col_b)`, "JoinConstr{T T CommaList{ColName T ColName} T}",
	)

	runTests(t, cases, (*Parser).joinConstraint)
}

func TestJoinConstraintError(t *testing.T) {
	cases := []testCaseError{
		{code: `ON`, msg: "unexpected EOF"},
		{code: `USING col_a, col_b)`, msg: "expecting LeftParen, got Identifier"},
		{code: `USING (col_a`, msg: "expecting RightParen, got EOF"},
		{code: `USING ()`, msg: "expecting Identifier, got RightParen"},
	}

	runTestsError(t, cases, (*Parser).joinConstraint)
}

func TestTableOrSubquery(t *testing.T) {
	cases := testCases(
		`table_name`, "TableOrSubquery{TableName}",
		`table_name table_alias`, "TableOrSubquery{TableName TableAlias}",
		`table_name AS table_alias`, "TableOrSubquery{TableName T TableAlias}",
		`table_name INDEXED BY index_name`, "TableOrSubquery{TableName TT IndexName}",
		`table_name NOT INDEXED`, "TableOrSubquery{TableName TT}",
		`temp.table_name AS table_alias`, "TableOrSubquery{SchemaName T TableName T TableAlias}",
		`table_function_name(10)`, "TableOrSubquery{TableFunctionName T CommaList{E{T}} T}",
		`table_function_name(10, 20) AS table_alias`, "TableOrSubquery{TableFunctionName T CommaList{E{T} T E{T}} T T TableAlias}",
		`table_function_name(10) table_alias`, "TableOrSubquery{TableFunctionName T CommaList{E{T}} T TableAlias}",
		`(SELECT 10)`, "TableOrSubquery{T SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}} T}",
		`(SELECT 10) AS table_alias`, "TableOrSubquery{T SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}} T T TableAlias}",
		`(SELECT 10) table_alias`, "TableOrSubquery{T SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}} T TableAlias}",
		`(table_name)`, "TableOrSubquery{T JoinClause{TableOrSubquery{TableName}} T}",
		`(table_name_1 JOIN table_name_2, table_name_2 ON a > 2)`,
		`TableOrSubquery{T JoinClause{TableOrSubquery{TableName} JoinOp{T} TableOrSubquery{TableName}
			JoinOp{T} TableOrSubquery{TableName} JoinConstr{T E{GreaterThan{ColRef{ColName} TT}}} } T}`,
		`(WITH go AS (SELECT rust) SELECT 10)`, `TableOrSubquery{T SimpleSelect{
			WithClause{T CommaList{CTE{TableName T T SimpleSelect{SelectCore{T CommaList{ResultColumn{E{ColRef{ColName}}}}}} T }}}
			SelectCore{T CommaList{ResultColumn{E{T}}}}} T}`,
	)

	runTests(t, cases, (*Parser).tableOrSubquery)
}

func TestTableOrSubqueryError(t *testing.T) {
	cases := []testCaseError{
		{code: `.table_name`, msg: "expecting LeftParen, got Dot"},
		{code: `schema_name.`, msg: "expecting [Identifier Temp], got EOF"},
		{code: `table_name AS`, msg: "expecting Identifier, got EOF"},
		{code: `table_name INDEXED index_name`, msg: "expecting By, got Identifier"},
		{code: `table_name NOT`, msg: "expecting Indexed, got EOF"},
		{code: `table_function_name()`, msg: "unexpected RightParen"},
		{code: `table_function_name(10`, msg: "expecting RightParen, got EOF"},
		{code: `table_function_name(10, 20) AS`, msg: "expecting Identifier, got EOF"},
		{code: `(SELECT 10`, msg: "expecting RightParen, got EOF"},
		{code: `(SELECT 10) AS`, msg: "expecting Identifier, got EOF"},
		{code: `(table_name`, msg: "expecting RightParen, got EOF"},
		{code: `()`, msg: "expecting [With Select Identifier Temp LeftParen], got RightParen"},
	}

	runTestsError(t, cases, (*Parser).tableOrSubquery)
}

func TestGroupByClause(t *testing.T) {
	cases := testCases(
		`GROUP BY 10`, "GroupByClause{TT CommaList{E{T}}}",
	)

	runTests(t, cases, (*Parser).groupByClause)
}

func TestGroupByClauseError(t *testing.T) {
	cases := []testCaseError{
		{code: `GROUP 10`, msg: "expecting By, got Numeric"},
		{code: `GROUP BY `, msg: "unexpected EOF"},
	}

	runTestsError(t, cases, (*Parser).groupByClause)
}

func TestHavingClause(t *testing.T) {
	cases := testCases(
		`HAVING 10`, "HavingClause{T E{T}}",
	)

	runTests(t, cases, (*Parser).havingClause)
}

func TestHavingClauseError(t *testing.T) {
	cases := []testCaseError{
		{code: `HAVING `, msg: "unexpected EOF"},
	}

	runTestsError(t, cases, (*Parser).havingClause)
}

func TestWindowClause(t *testing.T) {
	cases := testCases(
		`WINDOW window_a AS (PARTITION BY 10)`,
		"WindowClause{T CommaList{WindowClauseItem{WindowName T WindowDefinition{T PartitionBy{TT CommaList{E{T}}} T}}}}",
	)

	runTests(t, cases, (*Parser).windowClause)
}

func TestWindowClauseError(t *testing.T) {
	cases := []testCaseError{
		{code: `WINDOW `, msg: "expecting Identifier, got EOF"},
	}

	runTestsError(t, cases, (*Parser).windowClause)
}

func TestWindowClauseItem(t *testing.T) {
	cases := testCases(
		`window_name AS (PARTITION BY 10)`,
		"WindowClauseItem{WindowName T WindowDefinition{T PartitionBy{TT CommaList{E{T}}} T}}",
	)

	runTests(t, cases, (*Parser).windowClauseItem)
}

func TestWindowClauseItemError(t *testing.T) {
	cases := []testCaseError{
		{code: `window_name (PARTITION BY 10)`, msg: "expecting As, got LeftParen"},
		{code: `window_name`, msg: "expecting As, got EOF"},
	}

	runTestsError(t, cases, (*Parser).windowClauseItem)
}

func TestValuesClause(t *testing.T) {
	cases := testCases(
		`VALUES (10), (20)`, "ValuesClause{T CommaList{ValuesItem{T CommaList{E{T}} T} T ValuesItem{T CommaList{E{T}} T}}}",
	)

	runTests(t, cases, (*Parser).valuesClause)
}

func TestValuesClauseError(t *testing.T) {
	cases := []testCaseError{
		{code: `VALUES `, msg: "expecting LeftParen, got EOF"},
	}

	runTestsError(t, cases, (*Parser).valuesClause)
}

func TestValuesItem(t *testing.T) {
	cases := testCases(
		`(10, 20)`, "ValuesItem{T CommaList{E{T} T E{T}} T}",
	)

	runTests(t, cases, (*Parser).valuesItem)
}

func TestValuesItemError(t *testing.T) {
	cases := []testCaseError{
		{code: `() `, msg: "unexpected RightParen"},
	}

	runTestsError(t, cases, (*Parser).valuesItem)
}

func TestLimitClause(t *testing.T) {
	cases := testCases(
		`LIMIT 10`, "LimitClause{T E{T}}",
		`LIMIT 10, 20`, "LimitClause{T E{T} T E{T}}",
		`LIMIT 10 OFFSET 20`, "LimitClause{T E{T} T E{T}}",
	)

	runTests(t, cases, (*Parser).limitClause)
}

func TestLimitClauseError(t *testing.T) {
	cases := []testCaseError{
		{code: `LIMIT 10,`, msg: "unexpected EOF"},
		{code: `LIMIT 10 OFFSET`, msg: "unexpected EOF"},
	}

	runTestsError(t, cases, (*Parser).limitClause)
}

func TestUpdate(t *testing.T) {
	cases := testCases(
		`UPDATE table_name SET col_name=10`, "Update{T QualifiedTableName{TableName} T CommaList{UpdateSetItem{ColName T E{T}}}}",
		`UPDATE table_name SET a=10, b=20`,
		"Update{T QualifiedTableName{TableName} T CommaList{UpdateSetItem{ColName T E{T}} T UpdateSetItem{ColName T E{T}}}}",
		`UPDATE OR ABORT table_name SET col_name=10`, "Update{T TT QualifiedTableName{TableName} T CommaList{UpdateSetItem{ColName T E{T}}}}",
		`UPDATE table_name SET (a, b)=(10, 20)`,
		`Update{T QualifiedTableName{TableName} T CommaList{UpdateSetItem{T CommaList{ColName T ColName} TT
				E{ParenExpression{T CommaList{E{T} T E{T}} T}}}}}`,
		`UPDATE table_name SET col_name=10 FROM table_name_2`,
		"Update{T QualifiedTableName{TableName} T CommaList{UpdateSetItem{ColName T E{T}}} FromClause{T JoinClause{TableOrSubquery{TableName}}}}",
		`UPDATE table_name SET col_name=10 WHERE TRUE`,
		"Update{T QualifiedTableName{TableName} T CommaList{UpdateSetItem{ColName T E{T}}} WhereClause{T E{T}}}",
		`UPDATE table_name SET col_name=10 RETURNING *`,
		"Update{T QualifiedTableName{TableName} T CommaList{UpdateSetItem{ColName T E{T}}} ReturningClause{T CommaList{ReturningItem{T}}}}",
		`UPDATE table_name SET col_name=10 ORDER BY 20`,
		"Update{T QualifiedTableName{TableName} T CommaList{UpdateSetItem{ColName T E{T}}} OrderByClause{TT CommaList{OrderingTerm{E{T}}}}}",
		`UPDATE table_name SET col_name=10 LIMIT 10`,
		"Update{T QualifiedTableName{TableName} T CommaList{UpdateSetItem{ColName T E{T}}} LimitClause{T E{T}}}",
	)

	runTests(t, cases, func(p *Parser) parsetree.NonTerminal { return p.update(nil) })
}

func TestUpdateError(t *testing.T) {
	cases := []testCaseError{
		{code: `UPDATE OR table_name SET col_name=10`, msg: "expecting [Abort Fail Ignore Replace Rollback], got Identifier"},
		{code: `UPDATE schema_name.table_name col_name=10`, msg: "expecting Set, got Identifier"},
		{code: `UPDATE SET col_name=10`, msg: "expecting Identifier, got Set"},
		{code: `UPDATE table_name SET `, msg: "expecting [Identifier LeftParen], got EOF"},
		{code: `UPDATE table_name SET a=10, `, msg: "expecting [Identifier LeftParen], got EOF"},
		{code: `UPDATE table_name SET col_name=10 RETURNING *, `, msg: "unexpected EOF"},
	}

	runTestsError(t, cases, func(p *Parser) parsetree.NonTerminal { return p.update(nil) })
}

func TestVacuum(t *testing.T) {
	cases := testCases(
		`VACUUM`, "Vacuum{T}",
		`VACUUM temp`, "Vacuum{T SchemaName}",
		`VACUUM INTO 'db.db'`, "Vacuum{T T FileName{E{T}}}",
	)

	runTests(t, cases, (*Parser).vacuum)
}

func TestVacuumError(t *testing.T) {
	cases := []testCaseError{
		{code: `VACUUM INTO`, msg: "unexpected EOF"},
		{code: `VACUUM schame_name INTO`, msg: "unexpected EOF"},
	}

	runTestsError(t, cases, (*Parser).vacuum)
}

// runTests executes tests of the function parseFunc.
func runTests[T parsetree.Construction](t *testing.T, cases []testCase, parseFunc func(*Parser) T) {
	for i, c := range cases {
		c := c
		t.Run(strconv.FormatInt(int64(i), 10), func(t *testing.T) {
			defer func() {
				switch v := recover().(type) {
				case *syntaxError:
					t.Fatalf("%d=>%q\n\t%s", i, c.code, v)
				case nil:
				default:
					panic(v)
				}
			}()
			tp := newTestParser(newTestLexer(c.tree))
			expected := tp.tree()

			p := New(lexer.New([]byte(c.code)))
			p.comments = make(map[*token.Token][]*token.Token)
			p.advance()
			p.advance()
			p.advance()
			p.advance()

			parsed := parseFunc(p)

			if str, equals := compare(c.code, p.comments, parsed, expected); !equals {
				t.Log(c.code)
				t.Log("\n" + str)
				t.Fail()
			}
		})
	}
}

func runTestsError[T parsetree.Construction](t *testing.T, cases []testCaseError, parseFunc func(*Parser) T) {
	for i, c := range cases {
		c := c
		t.Run(strconv.FormatInt(int64(i), 10), func(t *testing.T) {
			defer func() {
				t.Logf("%q", c.code)
				switch v := recover().(type) {
				case error:
					if c.msg != v.Error() {
						t.Errorf("got error message %q, want %q", v.Error(), c.msg)
					}
				case nil:
					t.Error("not panicked")
				default:
					t.Errorf("unexpected: %v", v)
				}
			}()
			p := New(lexer.New([]byte(c.code)))
			p.comments = make(map[*token.Token][]*token.Token)
			p.advance()
			p.advance()
			p.advance()
			p.advance()

			parseFunc(p)
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
		default:
			panic(fmt.Errorf("unknown type: %T", p))
		}
	} else if parsed != nil && expected == nil {
		fmt.Fprintf(c.tw, "%s%s\t%s<nil>\t\n", strings.Repeat(c.indent, c.indentLevel), parsed.Kind(), strings.Repeat(c.indent, c.indentLevel))

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
		fmt.Fprintf(c.tw, "%s<nil>\t%s%s\t\n", strings.Repeat(c.indent, c.indentLevel), strings.Repeat(c.indent, c.indentLevel), expected.Kind())

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

// log returns the data generated by the comparator.
func (c *comparator) log() string {
	c.tw.Flush()
	return c.b.String()
}

// testTokenKind is the kind of a token in the language for especifying a parse tree.
type testTokenKind int

const (
	tokenKindIdentifier testTokenKind = iota
	tokenKindLeftBrace
	tokenKindRightBrace
	tokenKindTokens
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
	for i := parsetree.KindAdd; i <= parsetree.KindWithClause; i++ {
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
	treeKinds["JoinOp"] = parsetree.KindJoinOperator
	treeKinds["JoinConstr"] = parsetree.KindJoinConstraint
}
