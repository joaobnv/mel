package parser

import (
	"bytes"
	"errors"
	"fmt"
	"iter"
	"slices"
	"strconv"
	"strings"
	"testing"
	"text/tabwriter"

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
			expected, err := toParsetree(c.tree)
			if err != nil {
				t.Fatal(err)
			}

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
		{code: `FROM`, msg: "unexpected From"},
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
		"S{AlterTable {TT TableName RenameTo {TT TableName}} T}",
		`ALTER TABLE schema_a.table_a RENAME TO table_b`,
		"S{AlterTable {TT SchemaName T TableName RenameTo {TT TableName}} T}",
		`ALTER TABLE table_a RENAME column_a TO column_b`,
		"S{AlterTable {TT TableName RenameColumn {T ColumnName T ColumnName}} T}",
		`ALTER TABLE /* comment */ table_a RENAME COLUMN column_a TO column_b`,
		"S{AlterTable {TT TableName RenameColumn {TT ColumnName T ColumnName}} T}",
		`ALTER TABLE table_a ADD column_b INTEGER INTEGER`,
		"S{AlterTable {TT TableName AddColumn {T ColumnDefinition {ColumnName TypeName{TT}}}} T}",
		`ALTER TABLE table_a ADD COLUMN column_b INTEGER(10)`,
		"S{AlterTable {TT TableName AddColumn {TT ColumnDefinition {ColumnName TypeName {TTTT}}}} T}",
		`ALTER TABLE table_a ADD COLUMN column_b INTEGER(10, 20) PRIMARY KEY`,
		"S{AlterTable {TT TableName AddColumn {TT ColumnDefinition {ColumnName TypeName {TTTTTT} ColConstr {PrimaryKeyColumnConstraint{TT}} }}} T}",
		`ALTER TABLE table_a ADD COLUMN column_b INTEGER CONSTRAINT constr PRIMARY KEY ASC ON CONFLICT ROLLBACK NOT NULL`,
		"S{AlterTable {TT TableName AddColumn {TT ColumnDefinition {ColumnName TypeName{T} ColConstr{T ConstraintName PrimaryKeyColumnConstraint{TTT ConflictClause{TTT}}} ColConstr{NotNullColumnConstraint{TT}} }}} T}",
		`ALTER TABLE table_a ADD COLUMN column_b INTEGER UNIQUE CHECK(10)`,
		"S{AlterTable {TT TableName AddColumn {TT ColumnDefinition {ColumnName TypeName{T} ColConstr{UniqueColumnConstraint{T}} ColConstr{CheckColumnConstraint{TT E{T} T}} }}} T}",
		`ALTER TABLE table_a ADD COLUMN column_b INTEGER DEFAULT NULL COLLATE c PRIMARY KEY DESC AUTOINCREMENT`,
		"S{AlterTable {TT TableName AddColumn {TT ColumnDefinition {ColumnName TypeName{T} ColConstr{DefaultColumnConstraint{TT}} ColConstr{CollateColumnConstraint{T CollationName}} ColConstr{PrimaryKeyColumnConstraint{TTTT}} }}} T}",
		`ALTER TABLE table_a ADD COLUMN column_b INTEGER REFERENCES table_name AS (10)`,
		"S{AlterTable {TT TableName AddColumn {TT ColumnDefinition {ColumnName TypeName{T} ColConstr{ForeignKeyColumnConstraint{ForeignKeyClause{T TableName}}} ColConstr{GeneratedColumnConstraint{TT E{T} T}} }}} T}",
		`ALTER TABLE table_a ADD COLUMN column_b INTEGER NOT NULL ON CONFLICT ROLLBACK UNIQUE ON CONFLICT ROLLBACK`,
		"S{AlterTable {TT TableName AddColumn {TT ColumnDefinition {ColumnName TypeName{T} ColConstr{NotNullColumnConstraint{TT ConflictClause{TTT}}} ColConstr{UniqueColumnConstraint{T ConflictClause{TTT}}} }}} T}",
		`ALTER TABLE table_a ADD COLUMN column_b INTEGER CHECK(a > 10) DEFAULT(a > 10)`,
		"S{AlterTable {TT TableName AddColumn {TT ColumnDefinition {ColumnName TypeName{T} ColConstr{CheckColumnConstraint{TT E{GreaterThan{ColRef{ColName} TT}} T}} ColConstr{DefaultColumnConstraint{TT E{GreaterThan{ColRef{ColName} TT}} T}} }}} T}",
		`ALTER TABLE table_a ADD COLUMN column_b INTEGER DEFAULT 10 COLLATE c AS (10) STORED`,
		"S{AlterTable {TT TableName AddColumn {TT ColumnDefinition {ColumnName TypeName{T} ColConstr{DefaultColumnConstraint{TT}} ColConstr{CollateColumnConstraint{T CollationName}} ColConstr{GeneratedColumnConstraint{TT E{T} T T}} }}} T}",
		`ALTER TABLE table_a ADD COLUMN column_b INTEGER DEFAULT -10 GENERATED ALWAYS AS (10)`,
		"S{AlterTable {TT TableName AddColumn {TT ColumnDefinition {ColumnName TypeName{T} ColConstr{DefaultColumnConstraint{T TT}} ColConstr{GeneratedColumnConstraint{TTTT E{T} T}} }}} T}",
		`ALTER TABLE table_a ADD COLUMN column_b INTEGER DEFAULT +10 AS (10) VIRTUAL`,
		"S{AlterTable {TT TableName AddColumn {TT ColumnDefinition {ColumnName TypeName{T} ColConstr{DefaultColumnConstraint{T TT}} ColConstr{GeneratedColumnConstraint{TT E{T} T T}} }}} T}",
		`ALTER TABLE table_a ADD COLUMN column_b INTEGER REFERENCES table_name`,
		"S{AlterTable {TT TableName AddColumn {TT ColumnDefinition {ColumnName TypeName{T} ColConstr{ForeignKeyColumnConstraint{ForeignKeyClause{T TableName}}} }}} T}",
		`ALTER TABLE table_a DROP column_b`,
		"S{AlterTable {TT TableName DropColumn {T ColumnName}} T}",
		`ALTER TABLE table_a DROP COLUMN column_b`,
		"S{AlterTable {TT TableName DropColumn {TT ColumnName}} T}",
	)

	runTests(t, cases)
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
		{code: `ALTER TABLE table_a ADD COLUMN column_a INTEGER(, -10)`, msg: "expecting Numeric, got Comma"},
		{code: `ALTER TABLE table_a ADD COLUMN column_a INTEGER(-10, )`, msg: "expecting Numeric, got RightParen"},
		{code: `ALTER TABLE table_a ADD COLUMN column_a INTEGER(-10 10)`, msg: "expecting Comma, got Numeric"},
		{code: `ALTER TABLE table_a ADD COLUMN column_a CONSTRAINT PRIMARY KEY`, msg: "expecting Identifier, got Primary"},
		{code: `ALTER TABLE table_a ADD COLUMN column_a PRIMARY`, msg: "expecting Key, got EOF"},
		{code: `ALTER TABLE table_a ADD COLUMN column_a NOT`, msg: "expecting Null, got EOF"},
		{code: `ALTER TABLE table_a ADD COLUMN column_a CHECK a > 10)`, msg: "expecting LeftParen, got Identifier"},
		{code: `ALTER TABLE table_a ADD COLUMN column_a CHECK()`, msg: "unexpected RightParen"},
		{code: `ALTER TABLE table_a ADD COLUMN column_a CHECK(a > 10`, msg: "expecting RightParen, got EOF"},
		{code: `ALTER TABLE table_a ADD COLUMN column_a DEFAULT a`, msg: "expecting [Plus Minus], got Identifier"},
		{code: `ALTER TABLE table_a ADD COLUMN column_a DEFAULT()`, msg: "unexpected RightParen"},
		{code: `ALTER TABLE table_a ADD COLUMN column_a DEFAULT(a > 10`, msg: "expecting RightParen, got EOF"},
		{code: `ALTER TABLE table_a ADD COLUMN column_a DEFAULT +a`, msg: "expecting Numeric, got Identifier"},
		{code: `ALTER TABLE table_a ADD COLUMN column_a COLLATE`, msg: "expecting Identifier, got EOF"},
		{code: `ALTER TABLE table_a ADD COLUMN column_a GENERATED AS (10)`, msg: "expecting Always, got As"},
		{code: `ALTER TABLE table_a ADD COLUMN column_a GENERATED ALWAYS (10)`, msg: "expecting As, got LeftParen"},
		{code: `ALTER TABLE table_a ADD COLUMN column_a AS ()`, msg: "unexpected RightParen"},
		{code: `ALTER TABLE table_a ADD COLUMN column_a AS 10)`, msg: "expecting LeftParen, got Numeric"},
		{code: `ALTER TABLE table_a ADD COLUMN column_a AS (10`, msg: "expecting RightParen, got EOF"},
		{code: `ALTER TABLE table_a DROP COLUMN`, msg: "expecting Identifier, got EOF"},
	}

	runTestsError(t, cases)
}

func TestForeignKeyClause(t *testing.T) {
	cases := testCases(
		`CREATE TABLE table_a (column_b INTEGER INTEGER, column_c, FOREIGN KEY (column_b) REFERENCES table_name);`,
		`S{CreateTable {TT TableName T
			CommaList{ColDef {ColName TypeName{TT}} T ColDef {ColName}} T
			CommaList{TableConstraint{ForeignKeyTableConstraint{TTT CommaList{ColName} T
				ForeignKeyClause{T TableName}}}} T} T}`,
		`CREATE TABLE table_a (column_b INTEGER INTEGER, FOREIGN KEY (column_b) REFERENCES table_name(column_name));`,
		`S{CreateTable {TT TableName T
			CommaList{ColDef {ColName TypeName{TT}}} T
			CommaList{TableConstraint{ForeignKeyTableConstraint{TTT CommaList{ColName} T
				ForeignKeyClause{T TableName T CommaList{ColumnName} T}}}} T} T}`,
		`CREATE TABLE table_a (column_b INTEGER INTEGER, FOREIGN KEY (column_b) REFERENCES table_name ON DELETE SET NULL);`,
		`S{CreateTable {TT TableName T
			CommaList{ColDef {ColName TypeName{TT}}} T
			CommaList{TableConstraint{ForeignKeyTableConstraint{TTT CommaList{ColName} T
				ForeignKeyClause{T TableName TTTT}}}} T} T}`,
		`CREATE TABLE table_a (column_b INTEGER INTEGER, FOREIGN KEY (column_b) REFERENCES table_name ON UPDATE SET DEFAULT);`,
		`S{CreateTable {TT TableName T
			CommaList{ColDef {ColName TypeName{TT}}} T
			CommaList{TableConstraint{ForeignKeyTableConstraint{TTT CommaList{ColName} T
				ForeignKeyClause{T TableName TTTT}}}} T} T}`,
		`CREATE TABLE table_a (column_b INTEGER INTEGER, FOREIGN KEY (column_b) REFERENCES table_name ON DELETE CASCADE);`,
		`S{CreateTable {TT TableName T
			CommaList{ColDef {ColName TypeName{TT}}} T
			CommaList{TableConstraint{ForeignKeyTableConstraint{TTT CommaList{ColName} T
				ForeignKeyClause{T TableName TTT}}}} T} T}`,
		`CREATE TABLE table_a (column_b INTEGER INTEGER, FOREIGN KEY (column_b) REFERENCES table_name ON UPDATE RESTRICT);`,
		`S{CreateTable {TT TableName T
			CommaList{ColDef {ColName TypeName{TT}}} T
			CommaList{TableConstraint{ForeignKeyTableConstraint{TTT CommaList{ColName} T
				ForeignKeyClause{T TableName TTT}}}} T} T}`,
		`CREATE TABLE table_a (column_b INTEGER INTEGER, FOREIGN KEY (column_b) REFERENCES table_name ON DELETE NO ACTION);`,
		`S{CreateTable {TT TableName T
			CommaList{ColDef {ColName TypeName{TT}}} T
			CommaList{TableConstraint{ForeignKeyTableConstraint{TTT CommaList{ColName} T
				ForeignKeyClause{T TableName TTTT}}}} T} T}`,
		`CREATE TABLE table_a (column_b INTEGER INTEGER, FOREIGN KEY (column_b) REFERENCES table_name MATCH name);`,
		`S{CreateTable {TT TableName T
			CommaList{ColDef {ColName TypeName{TT}}} T
			CommaList{TableConstraint{ForeignKeyTableConstraint{TTT CommaList{ColName} T
				ForeignKeyClause{T TableName TT}}}} T} T}`,
		`CREATE TABLE table_a (column_b INTEGER INTEGER, FOREIGN KEY (column_b) REFERENCES table_name DEFERRABLE);`,
		`S{CreateTable {TT TableName T
			CommaList{ColDef {ColName TypeName{TT}}} T
			CommaList{TableConstraint{ForeignKeyTableConstraint{TTT CommaList{ColName} T
				ForeignKeyClause{T TableName T}}}} T} T}`,
		`CREATE TABLE table_a (column_b INTEGER INTEGER, FOREIGN KEY (column_b) REFERENCES table_name NOT DEFERRABLE);`,
		`S{CreateTable {TT TableName T
			CommaList{ColDef {ColName TypeName{TT}}} T
			CommaList{TableConstraint{ForeignKeyTableConstraint{TTT CommaList{ColName} T
				ForeignKeyClause{T TableName TT}}}} T} T}`,
		`CREATE TABLE table_a (column_b INTEGER INTEGER, FOREIGN KEY (column_b) REFERENCES table_name DEFERRABLE INITIALLY DEFERRED);`,
		`S{CreateTable {TT TableName T
			CommaList{ColDef {ColName TypeName{TT}}} T
			CommaList{TableConstraint{ForeignKeyTableConstraint{TTT CommaList{ColName} T
				ForeignKeyClause{T TableName TTT}}}} T} T}`,
		`CREATE TABLE table_a (column_b INTEGER INTEGER, FOREIGN KEY (column_b) REFERENCES table_name DEFERRABLE INITIALLY IMMEDIATE);`,
		`S{CreateTable {TT TableName T
			CommaList{ColDef {ColName TypeName{TT}}} T
			CommaList{TableConstraint{ForeignKeyTableConstraint{TTT CommaList{ColName} T
				ForeignKeyClause{T TableName TTT}}}} T} T}`,
	)

	runTests(t, cases)
}

func TestForeignKeyClauseError(t *testing.T) {
	cases := []testCaseError{
		{code: `CREATE TABLE table_a (column_b INTEGER INTEGER, FOREIGN KEY (column_b) REFERENCES`, msg: "expecting Identifier, got EOF"},
		{code: `CREATE TABLE table_a (column_b INTEGER INTEGER, FOREIGN KEY (column_b) REFERENCES table_name(column_name`, msg: "expecting RightParen, got EOF"},
		{code: `CREATE TABLE table_a (column_b INTEGER INTEGER, FOREIGN KEY (column_b) REFERENCES table_name ON SET NULL`, msg: "expecting [Delete Update], got Set"},
		{code: `CREATE TABLE table_a (column_b INTEGER INTEGER, FOREIGN KEY (column_b) REFERENCES table_name ON UPDATE SET`, msg: "expecting [Null Default], got EOF"},
		{code: `CREATE TABLE table_a (column_b INTEGER INTEGER, FOREIGN KEY (column_b) REFERENCES table_name ON DELETE NO`, msg: "expecting Action, got EOF"},
		{code: `CREATE TABLE table_a (column_b INTEGER INTEGER, FOREIGN KEY (column_b) REFERENCES table_name ON UPDATE`, msg: "expecting [Set Cascade Restrict No], got EOF"},
		{code: `CREATE TABLE table_a (column_b INTEGER INTEGER, FOREIGN KEY (column_b) REFERENCES table_name MATCH`, msg: "expecting Identifier, got EOF"},
		{code: `CREATE TABLE table_a (column_b INTEGER INTEGER, FOREIGN KEY (column_b) REFERENCES table_name DEFERRABLE INITIALLY`, msg: "expecting [Deferred Immediate], got EOF"},
	}
	runTestsError(t, cases)
}

func TestConflictClause(t *testing.T) {
	cases := testCases(
		`CREATE TABLE table_a (column_b, UNIQUE (column_b, column_c) ON CONFLICT ROLLBACK)`,
		`S{CreateTable {TT TableName T
			CommaList{ColDef{ColName}} T
			CommaList{TableConstraint{
				UniqueTableConstraint{TT CommaList{IndexedColumn{ColName} T IndexedColumn{ColName}} T
				ConflictClause{TTT}}}}
			T} T}`,
		`CREATE TABLE table_a (column_b, UNIQUE (column_b, column_c) ON CONFLICT ABORT)`,
		`S{CreateTable {TT TableName T
			CommaList{ColDef{ColName}} T
			CommaList{TableConstraint{
				UniqueTableConstraint{TT CommaList{IndexedColumn{ColName} T IndexedColumn{ColName}} T
				ConflictClause{TTT}}}}
			T} T}`,
		`CREATE TABLE table_a (column_b, UNIQUE (column_b, column_c) ON CONFLICT FAIL)`,
		`S{CreateTable {TT TableName T
			CommaList{ColDef{ColName}} T
			CommaList{TableConstraint{
				UniqueTableConstraint{TT CommaList{IndexedColumn{ColName} T IndexedColumn{ColName}} T
				ConflictClause{TTT}}}}
			T} T}`,
		`CREATE TABLE table_a (column_b, UNIQUE (column_b, column_c) ON CONFLICT IGNORE)`,
		`S{CreateTable {TT TableName T
			CommaList{ColDef{ColName}} T
			CommaList{TableConstraint{
				UniqueTableConstraint{TT CommaList{IndexedColumn{ColName} T IndexedColumn{ColName}} T
				ConflictClause{TTT}}}}
			T} T}`,
		`CREATE TABLE table_a (column_b, UNIQUE (column_b, column_c) ON CONFLICT REPLACE)`,
		`S{CreateTable {TT TableName T
			CommaList{ColDef{ColName}} T
			CommaList{TableConstraint{
				UniqueTableConstraint{TT CommaList{IndexedColumn{ColName} T IndexedColumn{ColName}} T
				ConflictClause{TTT}}}}
			T} T}`,
	)

	runTests(t, cases)
}

func TestConflictClauseError(t *testing.T) {
	cases := []testCaseError{
		{code: `CREATE TABLE table_a (column_b, UNIQUE (column_b, column_c) ON REPLACE`, msg: "expecting Conflict, got Replace"},
		{code: `CREATE TABLE table_a (column_b, UNIQUE (column_b, column_c) ON CONFLICT`, msg: "expecting [Rollback Abort Fail Ignore Replace], got EOF"},
	}

	runTestsError(t, cases)
}

func TestAnalyze(t *testing.T) {
	cases := testCases(
		`ANALYZE schema_name`,
		"S{Analyze {T SchemaIndexOrTableName} T}",
		`ANALYZE schema_name.table_name`,
		"S{Analyze {T SchemaName T TableOrIndexName} T}",
	)

	runTests(t, cases)
}

func TestAnalyzeError(t *testing.T) {
	cases := []testCaseError{
		{code: `ANALYZE `, msg: "expecting Identifier, got EOF"},
		{code: `ANALYZE schema_name.`, msg: "expecting Identifier, got EOF"},
		{code: `ANALYZE .table_name`, msg: "expecting Identifier, got Dot"},
		{code: `ANALYZE .`, msg: "expecting Identifier, got Dot"},
	}

	runTestsError(t, cases)
}

func TestAttach(t *testing.T) {
	cases := testCases(
		`ATTACH DATABASE ':memory' AS schema_name`, "S{Attach {TT E{T} T SchemaName} T}",
		`ATTACH '' AS schema_name`, "S{Attach {T E{T} T SchemaName} T}",
	)

	runTests(t, cases)
}

func TestAttachError(t *testing.T) {
	cases := []testCaseError{
		{code: `ATTACH ;`, msg: "unexpected Semicolon"},
		{code: `ATTACH AS ;`, msg: "unexpected As"},
		{code: `ATTACH ':memory' schema_name ;`, msg: "expecting As, got Identifier"},
	}

	runTestsError(t, cases)
}

func TestBegin(t *testing.T) {
	cases := testCases(
		`BEGIN`, "S{Begin {T} T}",
		`BEGIN DEFERRED TRANSACTION`, "S{Begin {TTT} T}",
		`BEGIN IMMEDIATE`, "S{Begin {TT} T}",
		`BEGIN EXCLUSIVE TRANSACTION`, "S{Begin {TTT} T}",
	)

	runTests(t, cases)
}

func TestCommit(t *testing.T) {
	cases := testCases(
		`COMMIT`, "S{Commit {T} T}",
		`COMMIT TRANSACTION`, "S{Commit {TT} T}",
		`END`, "S{Commit {T} T}",
		`END TRANSACTION`, "S{Commit {TT} T}",
	)

	runTests(t, cases)
}

func TestRollback(t *testing.T) {
	cases := testCases(
		`ROLLBACK`,
		"S{Rollback {T} T}",
		`ROLLBACK TRANSACTION`,
		"S{Rollback {TT} T}",
		`ROLLBACK TRANSACTION TO save_point_name`,
		"S{Rollback {TTT SavepointName} T}",
		`ROLLBACK TO SAVEPOINT save_point_name`,
		"S{Rollback {TTT SavepointName} T}",
	)

	runTests(t, cases)
}

func TestRollbackError(t *testing.T) {
	cases := []testCaseError{
		{code: `ROLLBACK TO`, msg: "expecting Identifier, got EOF"},
		{code: `ROLLBACK TO SAVEPOINT`, msg: "expecting Identifier, got EOF"},
	}

	runTestsError(t, cases)
}

func TestCreateIndex(t *testing.T) {
	cases := testCases(
		`CREATE INDEX index_name ON table_name(column_name)`,
		"S{CreateIndex {TT IndexName T TableName T CommaList{IndexedColumn{E{ColRef{ColName}}}} T} T}",
		`CREATE UNIQUE INDEX IF NOT EXISTS index_name ON table_name(column_name COLLATE c ASC)`,
		"S{CreateIndex {TTT TTT IndexName T TableName T CommaList{IndexedColumn{ E{Collate{ColRef{ColName} T CollationName}} T}} T} T}",
		`CREATE INDEX schema_name.index_name ON table_name(column_name)`,
		"S{CreateIndex {TT SchemaName T IndexName T TableName T CommaList{IndexedColumn{E{ColRef{ColName}}}} T} T}",
		`CREATE INDEX index_name ON table_name(a + b ASC)`,
		"S{CreateIndex {TT IndexName T TableName T CommaList{IndexedColumn{E{Add{ColRef{ColName} T ColRef{ColName}}} T}} T} T}",
		`CREATE INDEX index_name ON table_name(a * b COLLATE collation_name DESC)`,
		"S{CreateIndex {TT IndexName T TableName T CommaList{IndexedColumn{E{Multiply{ColRef{ColName} T Collate{ColRef{ColName} T CollationName}}} T}} T} T}",
		`CREATE INDEX index_name ON table_name(column_name1, column_name2)`,
		"S{CreateIndex {TT IndexName T TableName T CommaList{IndexedColumn{E{ColRef{ColName}}} T IndexedColumn{E{ColRef{ColName}}}} T} T}",
		`CREATE INDEX index_name ON table_name(column_name) WHERE column_a > 10`,
		"S{CreateIndex {TT IndexName T TableName T CommaList{IndexedColumn{E{ColRef{ColName}}}} T T E{GreaterThan{ColRef{ColName} T T}}} T}",
	)

	runTests(t, cases)
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
		{code: `CREATE INDEX index_name ON table_name(column_name COLLATE ASC)`, msg: "expecting Identifier, got Asc"},
	}

	runTestsError(t, cases)
}

func TestCreateTable(t *testing.T) {
	cases := testCases(
		`CREATE TABLE table_name (column_a);`,
		"S{CreateTable {TT TableName T CommaList{ColDef{ColName}} T} T}",
		`CREATE TEMP TABLE IF NOT EXISTS temp.table_name (column_a INTEGER) STRICT, WITHOUT ROWID;`,
		"S{CreateTable {TTT TTT SchemaName T TableName T CommaList{ColDef{ColName TypeName{T}}} T CommaList{TableOption{T} T TableOption{TT}}} T}",
		`CREATE TEMP TABLE IF NOT EXISTS schema_name.table_name (column_a INTEGER);`,
		"S{CreateTable {TTT TTT SchemaName T TableName T CommaList{ColDef{ColName TypeName{T}}} T} T}",
		`CREATE TEMPORARY TABLE table_name AS SELECT 'value';`,
		"S{CreateTable {TTT TableName T SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}}} T}",
		`CREATE TEMP TABLE IF NOT EXISTS temp.table_name (column_a INTEGER);`,
		"S{CreateTable {TTT TTT SchemaName T TableName T CommaList{ColDef{ColName TypeName{T}}} T} T}",
		`CREATE TABLE table_a (column_b INTEGER INTEGER, FOREIGN KEY (column_b) REFERENCES table_b(b));`,
		`S{CreateTable {TT TableName T
			CommaList{ColDef {ColName TypeName{TT}}} T
			CommaList{TableConstraint{ForeignKeyTableConstraint{TTT CommaList{ColName} T
				ForeignKeyClause{T TableName T CommaList{ColName} T}}}} T} T}`,
		`CREATE TABLE table_a (column_b INTEGER(10), CHECK(column_b > 100));`,
		`S{CreateTable {TT TableName T
			CommaList{ColDef {ColName TypeName {TTTT}}} T
			CommaList{TableConstraint{CheckTableConstraint{TT E{GreaterThan{ColRef{ColName} TT}} T}}} T} T}`,
		`CREATE TABLE table_a (column_b INTEGER(10, 20) PRIMARY KEY);`,
		`S{CreateTable {TT TableName T CommaList{ColDef{ColName TypeName {TTTTTT}
			ColConstr {PrimaryKeyColumnConstraint{TT}} }} T} T}`,
		`CREATE TABLE table_a (column_b, UNIQUE (column_b, column_c) ON CONFLICT ROLLBACK) WITHOUT ROWID`,
		`S{CreateTable {TT TableName T
			CommaList{ColDef{ColName}} T
			CommaList{TableConstraint{UniqueTableConstraint{TT CommaList{IndexedColumn{ColName} T IndexedColumn{ColName}} T ConflictClause{TTT}}}}
			T CommaList{TableOption{TT}}} T}`,
		`CREATE TABLE table_a (column_b, CONSTRAINT cons_name CHECK(column_b > 10), PRIMARY KEY (column_b, column_c COLLATE binary ASC) ON CONFLICT ROLLBACK) WITHOUT ROWID`,
		`S{CreateTable {TT TableName T CommaList{ColDef{ColName}} T
			CommaList{
				TableConstraint{T ConstraintName CheckTableConstraint{TT E{GreaterThan{ColRef{ColName} TT}} T}} T
				TableConstraint{PrimaryKeyTableConstraint{TTT CommaList{IndexedColumn{ColName} T IndexedColumn{ColName T CollationName T}} T ConflictClause{TTT}}}
			}
			T CommaList{TableOption{TT}}} T}`,
		`CREATE TABLE a AS WITH CTE AS (SELECT 20) SELECT 10;`,
		`S{CreateTable{TT TableName T SimpleSelect{WithClause{T CommaList{CTE{TableName TT SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}} T}}} 
				SelectCore{T CommaList{ResultColumn{E{T}}}}}} T}`,
	)

	runTests(t, cases)
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
		{code: `CREATE TABLE table_name (column_a, CONSTRAINT PRIMARY KEY (column_name));`, msg: "expecting Identifier, got Primary"},
		{code: `CREATE TABLE table_name (column_a, CONSTRAINT pk CREATE);`, msg: "expecting [Primary Unique Check Foreign], got Create"},
		{code: `CREATE TABLE table_name (column_a, PRIMARY (column_name));`, msg: "expecting Key, got LeftParen"},
		{code: `CREATE TABLE table_name (column_a, PRIMARY KEY column_name));`, msg: "expecting LeftParen, got Identifier"},
		{code: `CREATE TABLE table_name (column_a, PRIMARY KEY (column_name);`, msg: "expecting RightParen, got Semicolon"},
		{code: `CREATE TABLE table_name (column_a, UNIQUE column_name);`, msg: "expecting LeftParen, got Identifier"},
		{code: `CREATE TABLE table_name (column_a, UNIQUE (column_name;`, msg: "expecting RightParen, got Semicolon"},
		{code: `CREATE TABLE table_name (column_a, CHECK TRUE));`, msg: "expecting LeftParen, got Identifier"},
		{code: `CREATE TABLE table_name (column_a, CHECK ());`, msg: "unexpected RightParen"},
		{code: `CREATE TABLE table_name (column_a, CHECK (TRUE);`, msg: "expecting RightParen, got Semicolon"},
		{code: `CREATE TABLE table_name (column_a, FOREIGN (column_name) REFERENCES table_name);`, msg: "expecting Key, got LeftParen"},
		{code: `CREATE TABLE table_name (column_a, FOREIGN KEY column_name) REFERENCES table_name);`, msg: "expecting LeftParen, got Identifier"},
		{code: `CREATE TABLE table_name (column_a, FOREIGN KEY () REFERENCES table_name);`, msg: "expecting Identifier, got RightParen"},
		{code: `CREATE TABLE table_name (column_a, FOREIGN KEY (column_name REFERENCES table_name);`, msg: "expecting RightParen, got References"},
		{code: `CREATE TABLE table_name (column_a, FOREIGN KEY (column_name));`, msg: "expecting References, got RightParen"},
		{code: `CREATE TABLE table_name (column_a, FOREIGN KEY (column_name) REFERENCES);`, msg: "expecting Identifier, got RightParen"},
		{code: `CREATE TABLE table_name (column_a) WITHOUT;`, msg: "expecting RowId, got Semicolon"},
	}

	runTestsError(t, cases)
}

func TestColumnNameList(t *testing.T) {
	cases := testCases(
		`INSERT INTO table_name(a, b) VALUES(10, 'www')`,
		`S{ Insert{TT TableName T CommaList{ColumnName T ColumnName} T T
			InsertValuesList{CommaList{InsertValuesItem{T CommaList{E{T} T E{T}} T}}}} T}`,
	)

	runTests(t, cases)
}

func TestCreateTrigger(t *testing.T) {
	cases := testCases(
		`CREATE TRIGGER trigger_name DELETE ON table_name BEGIN DELETE FROM table_name; END`,
		"S{CreateTrigger{TT TriggerName TT TableName TriggerBody{T Delete {TT QualifiedTableName{TableName}} TT}} T}",
		`CREATE TRIGGER trigger_name DELETE ON table_name BEGIN WITH cte AS (SELECT 10) DELETE FROM table_name; END`,
		`S{CreateTrigger{TT TriggerName TT TableName TriggerBody{T Delete {WithClause{T CommaList{
				CTE{TableName T T SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}}T }}}
			TT QualifiedTableName{TableName}} TT}} T}`,
		`CREATE TEMP TRIGGER IF NOT EXISTS trigger_name BEFORE DELETE ON table_name BEGIN INSERT INTO table_name(name) VALUES('Go'); END;`,
		`S{CreateTrigger{TTT TTT TriggerName TTT TableName TriggerBody{T Insert{TT TableName T CommaList{ColumnName} T T
			InsertValuesList{CommaList{InsertValuesItem{T CommaList{E{T}} T}}} } TT}} T}`,
		`CREATE TEMPORARY TRIGGER schema_name.trigger_name AFTER INSERT ON table_name FOR EACH ROW BEGIN SELECT 10; END`,
		"S{CreateTrigger{TTT SchemaName T TriggerName TTT TableName TTT TriggerBody{T SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}} TT}} T}",
		`CREATE TRIGGER trigger_name INSTEAD OF UPDATE ON table_name WHEN a > 10 BEGIN UPDATE table_name SET a=10; END`,
		`S{CreateTrigger{TT TriggerName TTTT TableName T E{GreaterThan{ColRef{ColName} T T}}
				TriggerBody{T Update {T QualifiedTableName{TableName} T CommaList{UpdateSetItem{ColumnName T E{T}}}} TT}} T}`,
		`CREATE TRIGGER trigger_name UPDATE OF a, b ON table_name BEGIN SELECT 10; END`,
		"S{CreateTrigger{TT TriggerName TT CommaList{ColName T ColName} T TableName TriggerBody{T SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}} TT}} T}",
	)

	runTests(t, cases)
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

	runTestsError(t, cases)
}

func TestCreateView(t *testing.T) {
	cases := testCases(
		`CREATE VIEW view_name AS SELECT 10`,
		"S{CreateView{TT ViewName T SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}}} T}",
		`CREATE TEMP VIEW IF NOT EXISTS view_name AS SELECT 10`,
		"S{CreateView{TTT TTT ViewName T SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}}} T}",
		`CREATE TEMPORARY VIEW schema_name.view_name (a, b) AS SELECT 10`,
		"S{CreateView{TTT SchemaName T ViewName  T CommaList{ColName T ColName} T T SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}}} T}",
		`CREATE View a AS WITH CTE AS (SELECT 20) SELECT 10;`,
		`S{CreateView{TT ViewName T SimpleSelect{WithClause{T CommaList{CTE{TableName TT SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}} T}}} 
				SelectCore{T CommaList{ResultColumn{E{T}}}}}} T}`,
	)

	runTests(t, cases)
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

	runTestsError(t, cases)
}

func TestCreateVirtualTable(t *testing.T) {
	cases := testCases(
		`CREATE VIRTUAL TABLE table_name USING module_name`,
		"S{CreateVirtualTable{TTT TableName T ModuleName} T}",
		`CREATE VIRTUAL TABLE IF NOT EXISTS table_name USING module_name`,
		"S{CreateVirtualTable{TTT TTT TableName T ModuleName} T}",
		`CREATE VIRTUAL TABLE temp.table_name USING module_name`,
		"S{CreateVirtualTable{TTT SchemaName T TableName T ModuleName} T}",
		`CREATE VIRTUAL TABLE table_name USING module_name(column1 TEXT, column2 INTEGER)`,
		"S{CreateVirtualTable{TTT TableName T ModuleName T CommaList{ModuleArgument{TT} T ModuleArgument{TT}} T} T}",
		`CREATE VIRTUAL TABLE table_name USING module_name(function(arg))`,
		"S{CreateVirtualTable{TTT TableName T ModuleName T CommaList{ModuleArgument{TTTT}} T} T}",
		`CREATE VIRTUAL TABLE table_name USING module_name(function(function(arg)))`,
		"S{CreateVirtualTable{TTT TableName T ModuleName T CommaList{ModuleArgument{TTTTTTT}} T} T}",
	)

	runTests(t, cases)
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

	runTestsError(t, cases)
}

func TestDelete(t *testing.T) {
	cases := testCases(
		`DELETE FROM table_name`,
		"S{Delete{TT QualifiedTableName{TableName}} T}",
		`DELETE FROM table_name WHERE a > b`,
		"S{Delete{TT QualifiedTableName{TableName} WhereClause{T E{GreaterThan{ColRef{ColName} T ColRef{ColName}}}}} T}",
		`DELETE FROM table_name RETURNING *`,
		"S{Delete{TT QualifiedTableName{TableName} ReturningClause{T CommaList{ReturningItem{T}}}} T}",
		`DELETE FROM table_name ORDER BY 20`,
		"S{Delete{TT QualifiedTableName{TableName} OrderByClause{TT CommaList{OrderingTerm{E{T}}}}} T}",
		`DELETE FROM table_name LIMIT 10`,
		"S{Delete{TT QualifiedTableName{TableName} LimitClause{T E{T}}} T}",
		`DELETE FROM table_name ORDER BY 20, 30`,
		"S{Delete{TT QualifiedTableName{TableName} OrderByClause{TT CommaList{OrderingTerm{E{T}} T OrderingTerm{E{T}}}}} T}",
		`WITH table_a AS (SELECT 10) DELETE FROM table_name`,
		"S{Delete{WithClause{T CommaList{CTE{TableName TT SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}} T}}} TT QualifiedTableName{TableName}} T}",
		"WITH RECURSIVE table_name AS (SELECT 10) DELETE FROM table_name",
		"S{Delete{WithClause{TT CommaList{CTE{TableName TT SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}} T}}} TT QualifiedTableName{TableName}} T}",
		"WITH table_name AS (SELECT 10), table_name2 AS (SELECT 10) DELETE FROM table_name",
		`S{Delete{WithClause{T CommaList{
			CTE{TableName TT SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}} T} T
			CTE{TableName TT SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}} T}}}
			TT QualifiedTableName{TableName}} T}`,
	)

	runTests(t, cases)
}

func TestDeleteError(t *testing.T) {
	cases := []testCaseError{
		{code: `DELETE table_name`, msg: "expecting From, got Identifier"},
		{code: `DELETE FROM`, msg: "expecting Identifier, got EOF"},
		{code: `DELETE FROM table_name WHERE`, msg: "unexpected EOF"},
		{code: `DELETE FROM table_name RETURNING`, msg: "unexpected EOF"},
	}

	runTestsError(t, cases)
}

func TestWithClauseError(t *testing.T) {
	cases := []testCaseError{
		{code: `WITH `, msg: "expecting Identifier, got EOF"},
		{code: `WITH table_name AS (SELECT 10), `, msg: "expecting Identifier, got EOF"},
	}

	runTestsError(t, cases)
}

func TestCTE(t *testing.T) {
	cases := testCases(
		`WITH table_name AS (SELECT 10) SELECT 10`,
		`S{ SimpleSelect{WithClause{T CommaList{
					CTE{TableName T T SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}} T}}}
					SelectCore{T CommaList{ResultColumn{E{T}}}}} T}`,
		`WITH table_name AS MATERIALIZED (SELECT 10) SELECT 10`,
		`S{ SimpleSelect{WithClause{T CommaList{
						CTE{TableName TT T SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}} T}
					}} SelectCore{T CommaList{ResultColumn{E{T}}}} }
				T }`,
		`WITH table_name AS NOT MATERIALIZED (SELECT 10) SELECT 10`,
		`S{ SimpleSelect{WithClause{T CommaList{
						CTE{TableName T TT T SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}} T}
					}} SelectCore{T CommaList{ResultColumn{E{T}}}} }
				T }`,
		`WITH table_name (col_a, col_b) AS (SELECT 10) SELECT 10`,
		`S{ SimpleSelect{WithClause{T CommaList{
						CTE{TableName T CommaList{ColName T ColName} T T T SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}} T}
					}} SelectCore{T CommaList{ResultColumn{E{T}}}} }
				T }`,
	)

	runTests(t, cases)
}

func TestCTEError(t *testing.T) {
	cases := []testCaseError{
		{code: `WITH table_name (col_a, col_b AS (SELECT 10) SELECT 10`, msg: "expecting RightParen, got As"},
		{code: `WITH table_name AS SELECT 10) SELECT 10`, msg: "expecting LeftParen, got Select"},
		{code: `WITH table_name AS (SELECT 10 SELECT 10`, msg: "expecting RightParen, got Select"},
		{code: `WITH table_name (col_a, col_b) (SELECT 10) SELECT 10`, msg: "expecting As, got LeftParen"},
		{code: `WITH table_name AS NOT (SELECT 10) SELECT 10`, msg: "expecting Materialized, got LeftParen"},
		{code: `WITH table_name AS () SELECT 10`, msg: "expecting Select, got RightParen"},
		{code: `WITH table_name AS (WITH cte AS (SELECT 10)) SELECT 10`, msg: "expecting Select, got RightParen"},
	}

	runTestsError(t, cases)
}

func TestQualifiedTableName(t *testing.T) {
	cases := testCases(
		`DELETE FROM table_name`,
		"S{ Delete{TT QualifiedTableName{TableName}} T}",
		`DELETE FROM table_name AS alias_name`,
		"S{ Delete{TT QualifiedTableName{TableName T TableAlias}} T}",
		`DELETE FROM schema_name.table_name AS alias_name`,
		"S{ Delete{TT QualifiedTableName{SchemaName T TableName T TableAlias}} T}",
		`DELETE FROM table_name INDEXED BY index_name`,
		"S{ Delete{TT QualifiedTableName{TableName TT IndexName}} T}",
		`DELETE FROM table_name NOT INDEXED`,
		"S{ Delete{TT QualifiedTableName{TableName TT}} T}",
	)

	runTests(t, cases)
}

func TestQualifiedTableNameError(t *testing.T) {
	cases := []testCaseError{
		{code: `DELETE FROM Schema_name.`, msg: "expecting Identifier, got EOF"},
		{code: `DELETE FROM .table_name`, msg: "expecting Identifier, got Dot"},
		{code: `DELETE FROM table_name AS`, msg: "expecting Identifier, got EOF"},
		{code: `DELETE FROM table_name INDEXED index_name`, msg: "expecting By, got Identifier"},
		{code: `DELETE FROM table_name INDEXED BY`, msg: "expecting Identifier, got EOF"},
		{code: `DELETE FROM table_name NOT`, msg: "expecting Indexed, got EOF"},
	}

	runTestsError(t, cases)
}

func TestReturningClause(t *testing.T) {
	cases := testCases(
		`DELETE FROM table_name RETURNING *`,
		"S{ Delete{TT QualifiedTableName{TableName} ReturningClause{T CommaList{ReturningItem{T}}}} T}",
		`DELETE FROM table_name RETURNING a`,
		"S{ Delete{TT QualifiedTableName{TableName} ReturningClause{T CommaList{ReturningItem{E{ColRef{ColName}}}}}} T}",
		`DELETE FROM table_name RETURNING a AS b`,
		"S{ Delete{TT QualifiedTableName{TableName} ReturningClause{T CommaList{ReturningItem{E{ColRef{ColName}} T ColumnAlias}}}} T}",
		`DELETE FROM table_name RETURNING a b`,
		"S{ Delete{TT QualifiedTableName{TableName} ReturningClause{T CommaList{ReturningItem{E{ColRef{ColName}} ColumnAlias}}}} T}",
		`DELETE FROM table_name RETURNING a AS b, c`,
		`S{ Delete{TT QualifiedTableName{TableName} ReturningClause{T CommaList{ReturningItem{E{ColRef{ColName}} T ColumnAlias}
				T ReturningItem{E{ColRef{ColName}}}}}} T}`,
	)

	runTests(t, cases)
}

func TestReturningClauseError(t *testing.T) {
	cases := []testCaseError{
		{code: `DELETE FROM table_name RETURNING`, msg: "unexpected EOF"},
		{code: `DELETE FROM table_name RETURNING a AS`, msg: "expecting Identifier, got EOF"},
	}

	runTestsError(t, cases)
}

func TestDetach(t *testing.T) {
	cases := testCases(
		`DETACH temp`, "S{Detach{T SchemaName} T}",
		`DETACH DATABASE temp`, "S{Detach{TT SchemaName} T}",
	)

	runTests(t, cases)
}

func TestDetachError(t *testing.T) {
	cases := []testCaseError{
		{code: `DETACH`, msg: "expecting [Identifier Temp], got EOF"},
	}

	runTestsError(t, cases)
}

func TestDropIndex(t *testing.T) {
	cases := testCases(
		`DROP INDEX index_name`, "S{DropIndex{TT IndexName} T}",
		`DROP INDEX IF EXISTS index_name`, "S{DropIndex{TT TT IndexName} T}",
		`DROP INDEX temp.index_name`, "S{DropIndex{TT SchemaName T IndexName} T}",
	)

	runTests(t, cases)
}

func TestDropIndexError(t *testing.T) {
	cases := []testCaseError{
		{code: `DROP INDEX IF index_name`, msg: "expecting Exists, got Identifier"},
		{code: `DROP INDEX .index_name`, msg: "expecting Identifier, got Dot"},
		{code: `DROP INDEX schema_name.`, msg: "expecting Identifier, got EOF"},
	}

	runTestsError(t, cases)
}

func TestDropTable(t *testing.T) {
	cases := testCases(
		`DROP TABLE table_name`, "S{DropTable{TT TableName} T}",
		`DROP TABLE IF EXISTS table_name`, "S{DropTable{TT TT TableName} T}",
		`DROP TABLE temp.table_name`, "S{DropTable{TT SchemaName T TableName} T}",
	)

	runTests(t, cases)
}

func TestDropTableError(t *testing.T) {
	cases := []testCaseError{
		{code: `DROP TABLE IF table_name`, msg: "expecting Exists, got Identifier"},
		{code: `DROP TABLE .table_name`, msg: "expecting Identifier, got Dot"},
		{code: `DROP TABLE schema_name.`, msg: "expecting Identifier, got EOF"},
	}

	runTestsError(t, cases)
}

func TestDropTrigger(t *testing.T) {
	cases := testCases(
		`DROP TRIGGER trigger_name`, "S{DropTrigger{TT TriggerName} T}",
		`DROP TRIGGER IF EXISTS trigger_name`, "S{DropTrigger{TT TT TriggerName} T}",
		`DROP TRIGGER temp.trigger_name`, "S{DropTrigger{TT SchemaName T TriggerName} T}",
	)

	runTests(t, cases)
}

func TestDropTriggerError(t *testing.T) {
	cases := []testCaseError{
		{code: `DROP TRIGGER IF trigger_name`, msg: "expecting Exists, got Identifier"},
		{code: `DROP TRIGGER .trigger_name`, msg: "expecting Identifier, got Dot"},
		{code: `DROP TRIGGER schema_name.`, msg: "expecting Identifier, got EOF"},
	}

	runTestsError(t, cases)
}

func TestDropView(t *testing.T) {
	cases := testCases(
		`DROP VIEW view_name`, "S{DropView{TT ViewName} T}",
		`DROP VIEW IF EXISTS view_name`, "S{DropView{TT TT ViewName} T}",
		`DROP VIEW temp.view_name`, "S{DropView{TT SchemaName T ViewName} T}",
	)

	runTests(t, cases)
}

func TestDropViewError(t *testing.T) {
	cases := []testCaseError{
		{code: `DROP VIEW IF view_name`, msg: "expecting Exists, got Identifier"},
		{code: `DROP VIEW .view_name`, msg: "expecting Identifier, got Dot"},
		{code: `DROP VIEW schema_name.`, msg: "expecting Identifier, got EOF"},
	}

	runTestsError(t, cases)
}

func TestExpression(t *testing.T) {
	cases := testCases(
		`SELECT TRUE OR TRUE`, "S{ SimpleSelect{SelectCore{T CommaList{ ResultColumn{E{Or{TTT}}} }}} T}",
		`SELECT NOT 'a' NOT LIKE 1`, "S{ SimpleSelect{SelectCore{T CommaList{ ResultColumn{E{Not{T NotLike{TTTT}}}} }}} T}",
		`SELECT (NOT 'a') NOT LIKE 1`, "S{ SimpleSelect{SelectCore{T CommaList{ ResultColumn{E{NotLike{ParenE{T CommaList{E{Not{TT}}} T} TTT}}} }}} T}",
		`SELECT 0 AND 1 LIKE 1;`, "S{ SimpleSelect{SelectCore{T CommaList{ ResultColumn{E{And{TT Like{TTT}}}} }}} T}",
		`SELECT (0 AND 1) LIKE 1;`, "S{ SimpleSelect{SelectCore{T CommaList{ ResultColumn{E{Like{ParenE{T CommaList{E{And{TTT}}} T} TT}}} }}} T}",
	)

	runTests(t, cases)
}

func TestExpression1(t *testing.T) {
	cases := testCases(
		`ATTACH TRUE OR TRUE AS schema_name`, "S{ Attach{T E{Or{TTT}} T SchemaName} T}",
	)

	runTests(t, cases)
}

func TestExpression1Error(t *testing.T) {
	cases := []testCaseError{
		{code: `ATTACH TRUE OR AS schema_name`, msg: "unexpected As"},
	}

	runTestsError(t, cases)
}

func TestExpression2(t *testing.T) {
	cases := testCases(
		`SELECT TRUE AND TRUE`, "S{ SimpleSelect{SelectCore{T CommaList{ ResultColumn{E{And{TTT}}} }}} T}",
	)

	runTests(t, cases)
}

func TestExpression2Error(t *testing.T) {
	cases := []testCaseError{
		{code: `SELECT TRUE AND`, msg: "unexpected EOF"},
	}

	runTestsError(t, cases)
}

func TestExpression3(t *testing.T) {
	cases := testCases(
		`ATTACH NOT TRUE AS schema_name`, "S{ Attach{T E{Not{TT}} T SchemaName} T}",
	)

	runTests(t, cases)
}

func TestExpression3Error(t *testing.T) {
	cases := []testCaseError{
		{code: `SELECT NOT`, msg: "unexpected EOF"},
	}

	runTestsError(t, cases)
}

func TestExpression4(t *testing.T) {
	cases := testCases(
		`SELECT a = c`, "S{ SimpleSelect{SelectCore{T CommaList{ ResultColumn{ E{Equal{ColRef{ColName} T ColRef{ColName}}} } }}} T}",
		`SELECT a == c`, "S{ SimpleSelect{SelectCore{T CommaList{ ResultColumn{ E{Equal{ColRef{ColName} T ColRef{ColName}}} } }}} T}",
		`SELECT a <> c`, "S{ SimpleSelect{SelectCore{T CommaList{ ResultColumn{ E{NotEqual{ColRef{ColName} T ColRef{ColName}}} } }}} T}",
		`SELECT a != c`, "S{ SimpleSelect{SelectCore{T CommaList{ ResultColumn{ E{NotEqual{ColRef{ColName} T ColRef{ColName}}} } }}} T}",
		`SELECT 10 REGEXP '10'`, "S{ SimpleSelect{SelectCore{T CommaList{ ResultColumn{ E{Regexp{TTT}} } }}} T}",
		`SELECT '10' NOT REGEXP '20'`, "S{ SimpleSelect{SelectCore{T CommaList{ ResultColumn{ E{NotRegexp{TTTT}} } }}} T}",
		`SELECT '10' LIKE '20'`, "S{ SimpleSelect{SelectCore{T CommaList{ ResultColumn{ E{Like{TTT}} } }}} T}",
		`SELECT 10 LIKE '10' ESCAPE '!'`, "S{ SimpleSelect{SelectCore{T CommaList{ ResultColumn{ E{Like{TTTTT}} } }}} T}",
		`SELECT '10' NOT LIKE '20' ESCAPE '!'`, "S{ SimpleSelect{SelectCore{T CommaList{ ResultColumn{ E{NotLike{TTTTTT}} } }}} T}",
		`SELECT '10' GLOB '20'`, "S{ SimpleSelect{SelectCore{T CommaList{ ResultColumn{ E{Glob{TTT}} } }}} T}",
		`SELECT 10 NOT GLOB '10'`, "S{ SimpleSelect{SelectCore{T CommaList{ ResultColumn{ E{NotGlob{TTTT}} } }}} T}",
		`SELECT '10' MATCH '20'`, "S{ SimpleSelect{SelectCore{T CommaList{ ResultColumn{ E{Match{TTT}} } }}} T}",
		`SELECT 10 NOT MATCH '10'`, "S{ SimpleSelect{SelectCore{T CommaList{ ResultColumn{ E{NotMatch{TTTT}} } }}} T}",
		`SELECT 10 IS '10'`, "S{ SimpleSelect{SelectCore{T CommaList{ ResultColumn{ E{Is{TTT}} } }}} T}",
		`SELECT 10 BETWEEN 5 AND 15`, "S{ SimpleSelect{SelectCore{T CommaList{ ResultColumn{ E{Between{TTTTT}} } }}} T}",
		`SELECT 10 NOT BETWEEN 5 AND 15`, "S{ SimpleSelect{SelectCore{T CommaList{ ResultColumn{ E{NotBetween{TTTTTT}} } }}} T}",
		`SELECT 10 IN (5)`, "S{ SimpleSelect{SelectCore{T CommaList{ ResultColumn{ E{In{TTT CommaList{E{T}} T}} } }}} T}",
		`SELECT 10 NOT IN (5)`, "S{ SimpleSelect{SelectCore{T CommaList{ ResultColumn{ E{NotIn{TTTT CommaList{E{T}} T}} } }}} T}",
		`SELECT 10 ISNULL`, "S{ SimpleSelect{SelectCore{T CommaList{ ResultColumn{ E{Isnull{TT}} } }}} T}",
		`SELECT 10 NOTNULL`, "S{ SimpleSelect{SelectCore{T CommaList{ ResultColumn{ E{Notnull{TT}} } }}} T}",
		`SELECT 10 NOT NULL`, "S{ SimpleSelect{SelectCore{T CommaList{ ResultColumn{ E{NotNull{TTT}} } }}} T}",
	)

	runTests(t, cases)
}

func TestExpression4Error(t *testing.T) {
	cases := []testCaseError{
		{code: `SELECT 10 ==`, msg: "unexpected EOF"},
		{code: `SELECT 10 <>`, msg: "unexpected EOF"},
		{code: `SELECT 10 GLOB`, msg: "unexpected EOF"},
		{code: `SELECT 10 NOT GLOB`, msg: "unexpected EOF"},
		{code: `SELECT 10 REGEXP`, msg: "unexpected EOF"},
		{code: `SELECT 10 NOT REGEXP`, msg: "unexpected EOF"},
		{code: `SELECT 10 MATCH`, msg: "unexpected EOF"},
		{code: `SELECT 10 NOT MATCH`, msg: "unexpected EOF"},
		{code: `SELECT 10 LIKE ESCAPE`, msg: "unexpected Escape"},
		{code: `SELECT 10 NOT LIKE ESCAPE`, msg: "unexpected Escape"},
		{code: `SELECT 10 IN table_function_name(10`, msg: "expecting RightParen, got EOF"},
		{code: `SELECT 10 NOT IN table_function_name(10`, msg: "expecting RightParen, got EOF"},
	}

	runTestsError(t, cases)
}

func TestEqual(t *testing.T) {
	cases := testCases(
		`ATTACH a = c AS schema_name`, "S{ Attach{T E{Equal{ColRef{ColName} T ColRef{ColName}}} T SchemaName} T}",
		`ATTACH a = c = d AS schema_name`, "S{ Attach{T E{Equal{Equal{ColRef{ColName} T ColRef{ColName}} T ColRef{ColName}}} T SchemaName} T}",
		`ATTACH a == c AS schema_name`, "S{ Attach{T E{Equal{ColRef{ColName} T ColRef{ColName}}} T SchemaName} T}",
		`ATTACH a = c == d AS schema_name`, "S{ Attach{T E{Equal{Equal{ColRef{ColName} T ColRef{ColName}} T ColRef{ColName}}} T SchemaName} T}",
		`ATTACH a = NOT d AS schema_name`, "S{ Attach{T E{Equal{ColRef{ColName} T Not{T ColRef{ColName}}}} T SchemaName} T}",
	)

	runTests(t, cases)
}

func TestEqualError(t *testing.T) {
	cases := []testCaseError{
		{code: `ATTACH 10 =`, msg: "unexpected EOF"},
		{code: `ATTACH 10 ==`, msg: "unexpected EOF"},
	}

	runTestsError(t, cases)
}

func TestNotEqual(t *testing.T) {
	cases := testCases(
		`SELECT a <> c`, "S{ SimpleSelect{SelectCore{T CommaList{ ResultColumn{ E{NotEqual{ColRef{ColName} T ColRef{ColName}}} } }}} T}",
		`SELECT a <> NOT c`, "S{ SimpleSelect{SelectCore{T CommaList{ ResultColumn{ E{NotEqual{ColRef{ColName} T Not{T ColRef{ColName}}}} } }}} T}",
		`SELECT a <> c = d`, "S{ SimpleSelect{SelectCore{T CommaList{ ResultColumn{ E{Equal{NotEqual{ColRef{ColName} T ColRef{ColName}} T ColRef{ColName}}} } }}} T}",
		`SELECT a != c`, "S{ SimpleSelect{SelectCore{T CommaList{ ResultColumn{ E{NotEqual{ColRef{ColName} T ColRef{ColName}}} } }}} T}",
		`SELECT a != NOT c`, "S{ SimpleSelect{SelectCore{T CommaList{ ResultColumn{ E{NotEqual{ColRef{ColName} T Not{T ColRef{ColName}}}} } }}} T}",
		`SELECT a != c = d`, "S{ SimpleSelect{SelectCore{T CommaList{ ResultColumn{ E{Equal{NotEqual{ColRef{ColName} T ColRef{ColName}} T ColRef{ColName}}} } }}} T}",
	)

	runTests(t, cases)
}

func TestNotEqualError(t *testing.T) {
	cases := []testCaseError{
		{code: `SELECT 10 <>`, msg: "unexpected EOF"},
		{code: `SELECT 10 !=`, msg: "unexpected EOF"},
	}

	runTestsError(t, cases)
}

func TestRegexp(t *testing.T) {
	cases := testCases(
		`SELECT 10 REGEXP '10'`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{Regexp{TTT}} }}}} T}",
		`SELECT 1 REGEXP NOT 1`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{Regexp{TT Not{TT}}} }}}} T}",
		`SELECT '10' NOT REGEXP '20'`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{NotRegexp{TTTT}} }}}} T}",
		`SELECT 1 NOT REGEXP NOT 1`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{NotRegexp{TTT Not{TT}}} }}}} T}",
	)

	runTests(t, cases)
}

func TestRegexpError(t *testing.T) {
	cases := []testCaseError{
		{code: `SELECT 10 REGEXP`, msg: "unexpected EOF"},
		{code: `SELECT 10 NOT REGEXP`, msg: "unexpected EOF"},
	}

	runTestsError(t, cases)
}

func TestLike(t *testing.T) {
	cases := testCases(
		`SELECT 10 LIKE '10' ESCAPE '!'`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{Like{TTTTT}} }}}} T}",
		`SELECT 1 LIKE NOT 1`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{Like{TT Not{TT}}} }}}} T}",
		`SELECT 1 LIKE 1 ESCAPE NOT 1`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{Like{TTTT Not{TT}}} }}}} T}",
		`SELECT '10' NOT LIKE '20' ESCAPE '!'`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{NotLike{TTTTTT}} }}}} T}",
		`SELECT '10' NOT LIKE '20' ESCAPE NOT 1`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{NotLike{TTTTT Not{TT}}} }}}} T}",
		`SELECT 1 NOT LIKE NOT 1`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{NotLike{TTT Not{TT}}} }}}} T}",
	)

	runTests(t, cases)
}

func TestLikeError(t *testing.T) {
	cases := []testCaseError{
		{code: `SELECT 10 LIKE ESCAPE`, msg: "unexpected Escape"},
		{code: `SELECT 10 LIKE`, msg: "unexpected EOF"},
		{code: `SELECT 10 NOT LIKE ESCAPE`, msg: "unexpected Escape"},
		{code: `SELECT 10 NOT LIKE`, msg: "unexpected EOF"},
	}

	runTestsError(t, cases)
}

func TestGlob(t *testing.T) {
	cases := testCases(
		`SELECT '10' GLOB '20'`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{Glob{TTT}} }}}} T}",
		`SELECT 1 GLOB NOT 1`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{Glob{TT Not{TT}}} }}}} T}",
		`SELECT 10 NOT GLOB '10'`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{NotGlob{TTTT}} }}}} T}",
		`SELECT 1 NOT GLOB NOT 1`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{NotGlob{TTT Not{TT}}} }}}} T}",
	)

	runTests(t, cases)
}

func TestGlobError(t *testing.T) {
	cases := []testCaseError{
		{code: `SELECT 10 GLOB`, msg: "unexpected EOF"},
		{code: `SELECT 10 NOT GLOB`, msg: "unexpected EOF"},
	}

	runTestsError(t, cases)
}

func TestIsExpression(t *testing.T) {
	cases := testCases(
		`SELECT 10 IS 10`,
		"S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{Is{TTT}} }}}} T}",
		`SELECT 10 IS DISTINCT FROM 10`,
		"S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{IsDistinctFrom{TTTTT}} }}}} T}",
		`SELECT 10 IS DISTINCT FROM NOT 10`,
		"S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{IsDistinctFrom{TTTT Not{TT}}} }}}} T}",
		`SELECT 10 IS NOT 10`,
		"S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{IsNot{TTTT}} }}}} T}",
		`SELECT 10 IS NOT NOT 10`,
		"S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{IsNot{TTT Not{TT}}} }}}} T}",
		`SELECT 10 IS NOT DISTINCT FROM 10`,
		"S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{IsNotDistinctFrom{TTTTTT}} }}}} T}",
		`SELECT 10 IS NOT DISTINCT FROM NOT 10`,
		"S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{IsNotDistinctFrom{TTTTT Not{TT}}} }}}} T}",
	)

	runTests(t, cases)
}

func TestIsExpressionError(t *testing.T) {
	cases := []testCaseError{
		{code: `SELECT 10 IS NOT DISTINCT 10`, msg: "expecting From, got Numeric"},
		{code: `SELECT 10 IS NOT DISTINCT FROM`, msg: "unexpected EOF"},
		{code: `SELECT 10 IS NOT`, msg: "unexpected EOF"},
		{code: `SELECT 10 IS DISTINCT FROM`, msg: "unexpected EOF"},
		{code: `SELECT 10 IS DISTINCT FROM NOT`, msg: "unexpected EOF"},
		{code: `SELECT 10 IS DISTINCT 10`, msg: "expecting From, got Numeric"},
		{code: `SELECT 10 IS`, msg: "unexpected EOF"},
	}

	runTestsError(t, cases)
}

func TestBetween(t *testing.T) {
	cases := testCases(
		`SELECT 10 BETWEEN 5 AND 15`,
		`S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{Between{TTTTT}} }}}} T}`,
		`SELECT 10 BETWEEN NOT 5 AND NOT 15`,
		`S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{Between{TT Not{TT} T Not{TT}}} }}}} T}`,
	)

	runTests(t, cases)
}

func TestBetweenError(t *testing.T) {
	cases := []testCaseError{
		{code: `SELECT 10 BETWEEN`, msg: "unexpected EOF"},
		{code: `SELECT 10 BETWEEN AND 20`, msg: "unexpected And"},
		{code: `SELECT 10 BETWEEN 10  20`, msg: "expecting And, got Numeric"},
	}

	runTestsError(t, cases)
}

func TestNotBetween(t *testing.T) {
	cases := testCases(
		`SELECT 10 NOT BETWEEN 5 AND 15`,
		"S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{NotBetween{TTTTTT}} }}}} T}",
	)

	runTests(t, cases)
}

func TestNotBetweenError(t *testing.T) {
	cases := []testCaseError{
		{code: `SELECT 10 NOT BETWEEN`, msg: "unexpected EOF"},
		{code: `SELECT 10 NOT BETWEEN AND 20`, msg: "unexpected And"},
		{code: `SELECT 10 NOT BETWEEN 10  20`, msg: "expecting And, got Numeric"},
	}

	runTestsError(t, cases)
}

func TestIn(t *testing.T) {
	cases := testCases(
		`SELECT 10 IN (10, NOT 20)`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{In{TTT CommaList{E{T} T E{Not{TT}}} T}} }}}} T}",
		`SELECT 10 IN ()`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{In{TTTT}} }}}} T}",
		`SELECT 10 IN (SELECT 10)`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{In{TTT SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}} T}} }}}} T}",
		`SELECT 10 IN (WITH table_name_a AS (SELECT 10) SELECT 10)`,
		`S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{In{TTT SimpleSelect{
				WithClause{T CommaList{CTE{TableName TT SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}} T}}}
				SelectCore{T CommaList{ResultColumn{E{T}}}}} T}} }}}} T}`,
		`SELECT 10 IN schema_name.table_name`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{In{TT SchemaName T TableName}} }}}} T}",
		`SELECT 10 IN schema_a.table_function_a(10, 20)`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{In{TT SchemaName T TableFunctionName T CommaList{TTT} T}} }}}} T}",
		`SELECT 10 IN MAX()`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{In{TT TableFunctionName TT}} }}}} T}",
	)

	runTests(t, cases)
}

func TestInError(t *testing.T) {
	cases := []testCaseError{
		{code: `SELECT 10 IN (ALTER)`, msg: "expecting RightParen, got Alter"},
		{code: `SELECT 10 IN (1`, msg: "expecting RightParen, got EOF"},
		{code: `SELECT 10 IN function(10`, msg: "expecting RightParen, got EOF"},
		{code: `SELECT 10 IN ALTER`, msg: "expecting Identifier, got Alter"},
		{code: `SELECT 10 IN (SELECT)`, msg: "unexpected RightParen"},
	}

	runTestsError(t, cases)
}

func TestNotIn(t *testing.T) {
	cases := testCases(
		`SELECT 10 NOT IN (10, NOT 20)`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{NotIn{TTTT CommaList{E{T} T E{Not{TT}}} T}} }}}} T}",
		`SELECT 10 NOT IN ()`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{NotIn{TTTTT}} }}}} T}",
		`SELECT 10 NOT IN (SELECT 10)`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{NotIn{TTTT SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}} T}} }}}} T}",
		`SELECT 10 NOT IN (WITH table_name_a AS (SELECT 10) SELECT 10)`,
		`S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{NotIn{TTTT SimpleSelect{
				WithClause{T CommaList{CTE{TableName TT SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}} T}}}
				SelectCore{T CommaList{ResultColumn{E{T}}}}} T}} }}}} T}`,
		`SELECT 10 NOT IN schema_name.table_name`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{NotIn{TTT SchemaName T TableName}} }}}} T}",
		`SELECT 10 NOT IN schema_a.table_function_a(10, 20)`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{NotIn{TTT SchemaName T TableFunctionName T CommaList{TTT} T}} }}}} T}",
		`SELECT 10 NOT IN MAX()`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{NotIn{TTT TableFunctionName TT}} }}}} T}",
		`SELECT 10 NOT IN table_a`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{NotIn{TTT TableName}} }}}} T}",
	)

	runTests(t, cases)
}

func TestMatch(t *testing.T) {
	cases := testCases(
		`SELECT 10 MATCH 10`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{Match{TTT}} }}}} T}",
		`SELECT 10 MATCH NOT 10`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{Match{TT Not{TT}}} }}}} T}",
	)

	runTests(t, cases)
}

func TestMatchError(t *testing.T) {
	cases := []testCaseError{
		{code: `SELECT 10 MATCH`, msg: "unexpected EOF"},
	}

	runTestsError(t, cases)
}

func TestNotMatch(t *testing.T) {
	cases := testCases(
		`SELECT 10 NOT MATCH 10`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{NotMatch{TTTT}} }}}} T}",
		`SELECT 10 NOT MATCH NOT 10`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{NotMatch{TTT Not{TT}}} }}}} T}",
	)

	runTests(t, cases)
}

func TestNotMatchError(t *testing.T) {
	cases := []testCaseError{
		{code: `SELECT 10 NOT MATCH`, msg: "unexpected EOF"},
	}

	runTestsError(t, cases)
}

func TestNotInError(t *testing.T) {
	cases := []testCaseError{
		{code: `SELECT 10 NOT IN (SELECT)`, msg: "unexpected RightParen"},
		{code: `SELECT 10 NOT IN (ALTER)`, msg: "expecting RightParen, got Alter"},
		{code: `SELECT 10 NOT IN (1`, msg: "expecting RightParen, got EOF"},
		{code: `SELECT 10 NOT IN MAX(10`, msg: "expecting RightParen, got EOF"},
		{code: `SELECT 10 NOT IN ALTER`, msg: "expecting Identifier, got Alter"},
	}

	runTestsError(t, cases)
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
		`SELECT a < c`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{LessThan{ColRef{ColName} T ColRef{ColName}}} }}}} T}",
		`SELECT a <= c`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{LessThanOrEqual{ColRef{ColName} T ColRef{ColName}}} }}}} T}",
		`SELECT a > c`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{GreaterThan{ColRef{ColName} T ColRef{ColName}}} }}}} T}",
		`SELECT a >= c`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{GreaterThanOrEqual{ColRef{ColName} T ColRef{ColName}}} }}}} T}",
	)

	runTests(t, cases)
}

func TestExpression5Error(t *testing.T) {
	cases := []testCaseError{
		{code: `SELECT 10 <`, msg: "unexpected EOF"},
	}

	runTestsError(t, cases)
}

func TestExpression6(t *testing.T) {
	cases := testCases(
		`SELECT a & c`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{BitAnd{ColRef{ColName} T ColRef{ColName}}} }}}} T}",
		`SELECT a | c`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{BitOr{ColRef{ColName} T ColRef{ColName}}} }}}} T}",
		`SELECT a << c`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{LeftShift{ColRef{ColName} T ColRef{ColName}}} }}}} T}",
		`SELECT a >> c`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{RightShift{ColRef{ColName} T ColRef{ColName}}} }}}} T}",
	)

	runTests(t, cases)
}

func TestExpression6Error(t *testing.T) {
	cases := []testCaseError{
		{code: `SELECT 10 &`, msg: "unexpected EOF"},
	}

	runTestsError(t, cases)
}

func TestExpression7(t *testing.T) {
	cases := testCases(
		`SELECT a + c`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{Add{ColRef{ColName} T ColRef{ColName}}} }}}} T}",
		`SELECT a - c`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{Subtract{ColRef{ColName} T ColRef{ColName}}} }}}} T}",
	)

	runTests(t, cases)
}

func TestExpression7Error(t *testing.T) {
	cases := []testCaseError{
		{code: `SELECT 10 +`, msg: "unexpected EOF"},
	}

	runTestsError(t, cases)
}

func TestExpression8(t *testing.T) {
	cases := testCases(
		`SELECT a * c`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{Multiply{ColRef{ColName} T ColRef{ColName}}} }}}} T}",
		`SELECT a / c`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{Divide{ColRef{ColName} T ColRef{ColName}}} }}}} T}",
		`SELECT a % c`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{Mod{ColRef{ColName} T ColRef{ColName}}} }}}} T}",
	)

	runTests(t, cases)
}

func TestExpression8Error(t *testing.T) {
	cases := []testCaseError{
		{code: `SELECT 10 *`, msg: "unexpected EOF"},
	}

	runTestsError(t, cases)
}

func TestExpression9(t *testing.T) {
	cases := testCases(
		`SELECT a || c`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{Concatenate{ColRef{ColName} T ColRef{ColName}}} }}}} T}",
		`SELECT a -> c`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{Extract1{ColRef{ColName} T ColRef{ColName}}} }}}} T}",
		`SELECT a ->> c`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{Extract2{ColRef{ColName} T ColRef{ColName}}} }}}} T}",
	)

	runTests(t, cases)
}

func TestExpression9Error(t *testing.T) {
	cases := []testCaseError{
		{code: `SELECT 10 ||`, msg: "unexpected EOF"},
	}

	runTestsError(t, cases)
}

func TestExpression10(t *testing.T) {
	cases := testCases(
		`SELECT a`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{ColRef{ColName}} }}}} T}",
		`SELECT a COLLATE c`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{Collate{ColRef{ColName} T CollationName}} }}}} T}",
	)

	runTests(t, cases)
}

func TestExpression10Error(t *testing.T) {
	cases := []testCaseError{
		{code: `SELECT 10 COLLATE`, msg: "expecting Identifier, got EOF"},
	}

	runTestsError(t, cases)
}

func TestExpression11(t *testing.T) {
	cases := testCases(
		`SELECT ~a`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{BitNot{T ColRef{ColName}}} }}}} T}",
		`SELECT +a`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{PrefixPlus{T ColRef{ColName}}} }}}} T}",
		`SELECT -a`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{Negate{T ColRef{ColName}}} }}}} T}",
	)

	runTests(t, cases)
}

func TestExpression11Error(t *testing.T) {
	cases := []testCaseError{
		{code: `SELECT ~`, msg: "unexpected EOF"},
	}

	runTestsError(t, cases)
}

func TestSimpleExpression(t *testing.T) {
	cases := testCases(
		`SELECT a`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{ColRef{ColName}} }}}} T}",
		`SELECT ?`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{BindParameter} }}}} T}",
		`SELECT function_name()`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{FunctionCall{FunctionName T FnArgs{} T}} }}}} T}",
		`SELECT (10)`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{ParenE{T CommaList{E{T}} T}} }}}} T}",
		`SELECT (SELECT 10)`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{ParenSelect{T SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}} T}} }}}} T}",
		`SELECT (WITH go AS (SELECT rust) SELECT 10)`,
		`S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{ParenSelect{T SimpleSelect{
			WithClause{T CommaList{CTE{TableName T T SimpleSelect{SelectCore{T CommaList{ResultColumn{E{ColRef{ColName}}}}}} T }}}
			SelectCore{T CommaList{ResultColumn{E{T}}}}} T}} }}}} T}`,
		`SELECT CAST ('10' AS INTEGER)`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{Cast{TT E{T} T TypeName{T} T}} }}}} T}",
		`SELECT NOT EXISTS (SELECT 10)`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{Not{T Exists{T T SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}} T}}} }}}} T}",
		`SELECT EXISTS (SELECT 10)`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{Exists{T T SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}} T}} }}}} T}",
		`SELECT CASE WHEN TRUE THEN 10 END`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{Case{T When{T E{T} T E{T}} T}} }}}} T}",
		`SELECT RAISE (IGNORE)`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{Raise{TTTT}} }}}} T}",
	)

	runTests(t, cases)
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
		`SELECT a`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{ColRef{ColName}} }}}} T}",
		`SELECT TEMP`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{ColRef{ColName}} }}}} T}",
		`SELECT a.a`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{ColRef{TableName T ColName}} }}}} T}",
		`SELECT TEMP.TEMP`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{ColRef{TableName T ColName}} }}}} T}",
		`SELECT a.a.a`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{ColRef{SchemaName T TableName T ColName}} }}}} T}",
		`SELECT TEMP.TEMP.TEMP`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{ColRef{SchemaName T TableName T ColName}} }}}} T}",
	)

	runTests(t, cases)
}

func TestColumnReferenceError(t *testing.T) {
	cases := []testCaseError{
		{code: `SELECT a.`, msg: "expecting [Identifier Temp], got EOF"},
		{code: `SELECT TEMP.`, msg: "expecting [Identifier Temp], got EOF"},
		{code: `SELECT a..a`, msg: "expecting [Identifier Temp], got Dot"},
		{code: `SELECT TEMP..TEMP`, msg: "expecting [Identifier Temp], got Dot"},
	}

	runTestsError(t, cases)
}

func TestFunctionCall(t *testing.T) {
	cases := testCases(
		`SELECT MAX()`,
		"S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{FnCall{FnName T FnArgs{} T}} }}}} T}",
		`SELECT MAX('a')`,
		"S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{FnCall{FnName T FnArgs{CommaList{E{T}}} T}} }}}} T}",
		`SELECT MAX('a') OVER window_name`,
		"S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{FnCall{FnName T FnArgs{CommaList{E{T}}} T OverClause{T WindowName}}} }}}} T}",
		`SELECT MAX(10, 20);`,
		`S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{
				FunctionCall{FunctionName T FunctionArguments{CommaList{E{T} T E{T}} } T}
			}}} }} T}`,
		`SELECT MAX(10, 20) FILTER (WHERE true) OVER lisp;`,
		`S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{
				FunctionCall{FunctionName T FunctionArguments{CommaList{E{T} T E{T}} } T
					FilterClause{TTT E{T} T}
					OverClause{T WindowName}}
			}}} }} T}`,
		`SELECT MAX(10, 20) OVER (lisp PARTITION BY scheme ORDER BY ml);`,
		`S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{
				FunctionCall{FunctionName T FunctionArguments{CommaList{E{T} T E{T}} } T
					OverClause{T WindowDefinition{T WindowName PartitionBy{TT CommaList{E{ColRef{ColName}}}}
						OrderByClause{TT CommaList{OrderingTerm{E{ColRef{ColName}}}}} T}} }
			}}} }} T}`,
		`SELECT MAX(10, 20) OVER (lisp RANGE BETWEEN UNBOUNDED PRECEDING AND UNBOUNDED FOLLOWING);`,
		`S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{
				FunctionCall{FunctionName T FunctionArguments{CommaList{E{T} T E{T}} } T
					OverClause{T WindowDefinition{T WindowName FrameSpec{T FrameSpecBetween{T TT T TT}} T}} }
			}}} }} T}`,
		`SELECT MAX(10, 20) OVER (lisp ROWS BETWEEN CURRENT ROW AND CURRENT ROW);`,
		`S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{
				FunctionCall{FunctionName T FunctionArguments{CommaList{E{T} T E{T}} } T
					OverClause{T WindowDefinition{T WindowName FrameSpec{T FrameSpecBetween{T TT T TT}} T}} }
			}}} }} T}`,
	)

	runTests(t, cases)
}

func TestFunctionCallError(t *testing.T) {
	cases := []testCaseError{
		{code: `SELECT MAX(10 ORDER a);`, msg: "expecting By, got Identifier"},
		{code: `SELECT MAX(10 ORDER BY);`, msg: "unexpected RightParen"},
		{code: `SELECT MAX(10 ORDER BY a NULLS);`, msg: "expecting [First Last], got RightParen"},
		{code: `SELECT MAX(;`, msg: "expecting RightParen, got Semicolon"},
		{code: `SELECT MAX(DISTINCT );`, msg: "unexpected RightParen"},
		{code: `SELECT MAX() FILTER WHERE a);`, msg: "expecting LeftParen, got Where"},
		{code: `SELECT MAX() FILTER (a);`, msg: "expecting Where, got Identifier"},
		{code: `SELECT MAX(10 ORDER BY a COLLATE);`, msg: "expecting Identifier, got RightParen"},
		{code: `SELECT MAX(10 20);`, msg: "expecting RightParen, got Numeric"},
		{code: `SELECT MAX(10, 20;`, msg: "expecting RightParen, got Semicolon"},
		{code: `SELECT MAX(10, 20) FILTER(true)`, msg: "expecting Where, got Identifier"},
	}

	runTestsError(t, cases)
}

func TestFunctionArguments(t *testing.T) {
	cases := testCases(
		`SELECT MAX('a')`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{FunctionCall{FunctionName T FnArgs{CommaList{E{T}}} T}} }}}} T}",
		`SELECT MAX(DISTINCT 'a')`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{FunctionCall{FunctionName T FnArgs{T CommaList{E{T}}} T}} }}}} T}",
		`SELECT MAX(DISTINCT 'a' ORDER BY a)`,
		`S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{FunctionCall{FunctionName T
			FnArgs{T CommaList{E{T}} OrderByClause{TT CommaList{OrderingTerm{E{ColRef{ColName}}}}}} T}} }}}} T}`,
		`SELECT MAX('a' ORDER BY a)`,
		`S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{FunctionCall{FunctionName T
			FnArgs{CommaList{E{T}} OrderByClause{TT CommaList{OrderingTerm{E{ColRef{ColName}}}}}} T}} }}}} T}`,
		`SELECT MAX(*)`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{FunctionCall{FunctionName T FnArgs{T} T}} }}}} T}",
		`SELECT MAX()`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{FunctionCall{FunctionName T FnArgs{} T}} }}}} T}",
	)

	runTests(t, cases)
}

func TestFunctionArgumentsError(t *testing.T) {
	cases := []testCaseError{
		{code: `SELECT MAX(DISTINCT)`, msg: "unexpected RightParen"},
		{code: `SELECT MAX(ORDER BY a)`, msg: "expecting RightParen, got Order"},
		{code: `SELECT MAX(DISTINCT *)`, msg: "unexpected Asterisk"},
		{code: `SELECT MAX(* ORDER BY 10)`, msg: "expecting RightParen, got Order"},
	}

	runTestsError(t, cases)
}

func TestOrderByClause(t *testing.T) {
	cases := testCases(
		`SELECT MAX() OVER (ORDER BY a COLLATE c)`,
		`S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{FunctionCall{FunctionName T FunctionArguments{} T OverClause{T WindowDefinition{T
			OrderByClause{TT CommaList{OrderingTerm{E{Collate{ColRef{ColName} T CollationName}}}}} T}}}} }}}} T}`,
	)

	runTests(t, cases)
}

func TestOrderByClauseError(t *testing.T) {
	cases := []testCaseError{
		{code: `SELECT MAX() OVER (ORDER a)`, msg: "expecting By, got Identifier"},
		{code: `SELECT MAX() OVER (ORDER BY)`, msg: "unexpected RightParen"},
	}

	runTestsError(t, cases)
}

func TestOrderingTerm(t *testing.T) {
	cases := testCases(
		`SELECT MAX() OVER (ORDER BY a ASC NULLS LAST);`,
		`S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{FunctionCall{FunctionName T FunctionArguments{} T
				OverClause{T WindowDefinition{T OrderByClause{TT CommaList{OrderingTerm{E{ColRef{ColName}} TTT}}} T}}}} }}}} T}`,
		`SELECT MAX() OVER (ORDER BY a DESC NULLS FIRST)`,
		`S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{FunctionCall{FunctionName T FunctionArguments{} T
				OverClause{T WindowDefinition{T OrderByClause{TT CommaList{OrderingTerm{E{ColRef{ColName}} TTT}}} T}}}} }}}} T}`,
		`SELECT a ORDER BY a, b;`,
		`S{SimpleSelect{SelectCore{T CommaList{ResultColumn{E{ColRef{ColName}}}}} OrderByClause{TT CommaList{OrderingTerm{E{ColRef{ColName}}} T OrderingTerm{E{ColRef{ColName}}}} }} T}`,
	)

	runTests(t, cases)
}

func TestOrderingTermError(t *testing.T) {
	cases := []testCaseError{
		{code: `SELECT MAX() OVER (ORDER BY a DESC NULLS)`, msg: "expecting [First Last], got RightParen"},
	}

	runTestsError(t, cases)
}

func TestFilterClause(t *testing.T) {
	cases := testCases(
		`SELECT MAX() FILTER (WHERE a + b)`,
		`S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{FunctionCall{FunctionName T FunctionArguments{} T
					FilterClause{TT T E{Add{ColRef{ColName} T ColRef{ColName}}} T} }} }}}} T}`,
	)

	runTests(t, cases)
}

func TestFilterClauseError(t *testing.T) {
	cases := []testCaseError{
		{code: `SELECT MAX() FILTER WHERE a)`, msg: "expecting LeftParen, got Where"},
		{code: `SELECT MAX() FILTER (a)`, msg: "expecting Where, got Identifier"},
		{code: `SELECT MAX() FILTER (WHERE)`, msg: "unexpected RightParen"},
		{code: `SELECT MAX() FILTER (WHERE a`, msg: "expecting RightParen, got EOF"},
	}

	runTestsError(t, cases)
}

func TestOverClause(t *testing.T) {
	cases := testCases(
		`SELECT MAX() OVER window_a`,
		`S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{FunctionCall{FunctionName T FunctionArguments{} T
					OverClause{T WindowName} }} }}}} T}`,
		`SELECT MAX() OVER ()`,
		`S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{FunctionCall{FunctionName T FunctionArguments{} T
					OverClause{T WindowDefinition{TT}} }} }}}} T}`,
		`SELECT MAX() OVER (window_a PARTITION BY a, b);`,
		`S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{FunctionCall{FunctionName T FunctionArguments{} T
					OverClause{T WindowDefinition{T WindowName PartitionBy{TT CommaList{E{ColRef{ColName}} T E{ColRef{ColName}}}} T}} }} }}}} T}`,
		`SELECT MAX() OVER (ORDER BY a)`,
		`S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{FunctionCall{FunctionName T FunctionArguments{} T
					OverClause{T WindowDefinition{T OrderByClause{TT CommaList{OrderingTerm{E{ColRef{ColName}}}}} T}} }} }}}} T}`,
		`SELECT MAX() OVER (RANGE CURRENT ROW)`,
		`S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{FunctionCall{FunctionName T FunctionArguments{} T
					OverClause{T WindowDefinition{T FrameSpec{TTT} T}} }} }}}} T}`,
	)

	runTests(t, cases)
}

func TestOverClauseError(t *testing.T) {
	cases := []testCaseError{
		{code: `SELECT MAX() OVER ORDER BY a)`, msg: "expecting LeftParen, got Order"},
		{code: `SELECT MAX() OVER (window_a PARTITION a));`, msg: "expecting By, got Identifier"},
		{code: `SELECT MAX() OVER (window_a PARTITION BY));`, msg: "unexpected RightParen"},
		{code: `SELECT MAX() OVER (RANGE CURRENT ROW`, msg: "expecting RightParen, got EOF"},
		{code: `SELECT MAX() OVER`, msg: "expecting LeftParen, got EOF"},
	}

	runTestsError(t, cases)
}

func TestFrameSpec(t *testing.T) {
	cases := testCases(
		`SELECT MAX() OVER (ROWS UNBOUNDED PRECEDING EXCLUDE NO OTHERS)`,
		`S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{FunctionCall{FunctionName T FunctionArguments{} T OverClause{T WindowDefinition{T
				FrameSpec{TTTTTT} T}}}} }}}} T}`,
		`SELECT MAX() OVER (GROUPS 10 PRECEDING EXCLUDE CURRENT ROW)`,
		`S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{FunctionCall{FunctionName T FunctionArguments{} T OverClause{T WindowDefinition{T
				FrameSpec{T E{T} TTTT} T}}}} }}}} T}`,
		`SELECT MAX() OVER (RANGE CURRENT ROW EXCLUDE GROUP)`,
		`S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{FunctionCall{FunctionName T FunctionArguments{} T OverClause{T WindowDefinition{T
				FrameSpec{TTTTT} T}}}} }}}} T}`,
		`SELECT MAX() OVER (ROWS BETWEEN 10 PRECEDING AND 10 FOLLOWING EXCLUDE TIES)`,
		`S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{FunctionCall{FunctionName T FunctionArguments{} T OverClause{T WindowDefinition{T
				FrameSpec{T FrameSpecBetween{T E{T} TT E{T} T} TT} T}}}} }}}} T}`,
	)

	runTests(t, cases)
}

func TestFrameSpecError(t *testing.T) {
	cases := []testCaseError{
		{code: `SELECT MAX() OVER (ROWS UNBOUNDED EXCLUDE NO OTHERS)`, msg: "expecting Preceding, got Exclude"},
		{code: `SELECT MAX() OVER (RANGE CURRENT EXCLUDE GROUP)`, msg: "expecting Row, got Exclude"},
		{code: `SELECT MAX() OVER (GROUPS 10 EXCLUDE CURRENT ROW)`, msg: "expecting Preceding, got Exclude"},
		{code: `SELECT MAX() OVER (ROWS)`, msg: "unexpected RightParen"},
		{code: `SELECT MAX() OVER (ROWS UNBOUNDED PRECEDING EXCLUDE NO)`, msg: "expecting Others, got RightParen"},
		{code: `SELECT MAX() OVER (GROUPS 10 PRECEDING EXCLUDE CURRENT)`, msg: "expecting Row, got RightParen"},
		{code: `SELECT MAX() OVER (ROWS UNBOUNDED PRECEDING EXCLUDE)`, msg: "expecting [No Current Group Ties], got RightParen"},
	}

	runTestsError(t, cases)
}

func TestFrameSpecBetween(t *testing.T) {
	cases := testCases(
		`SELECT MAX() OVER (RANGE BETWEEN UNBOUNDED PRECEDING AND 10 PRECEDING)`,
		`S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{FunctionCall{FunctionName T FunctionArguments{} T OverClause{T WindowDefinition{T
				FrameSpec{T FrameSpecBetween{TTTT E{T} T}} T}}}} }}}} T}`,
		`SELECT MAX() OVER (RANGE BETWEEN 10 PRECEDING AND CURRENT ROW)`,
		`S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{FunctionCall{FunctionName T FunctionArguments{} T OverClause{T WindowDefinition{T
				FrameSpec{T FrameSpecBetween{T E{T} TTTT}} T}}}} }}}} T}`,
		`SELECT MAX() OVER (RANGE BETWEEN CURRENT ROW AND 10 FOLLOWING)`,
		`S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{FunctionCall{FunctionName T FunctionArguments{} T OverClause{T WindowDefinition{T
				FrameSpec{T FrameSpecBetween{TTTT E{T} T}} T}}}} }}}} T}`,
		`SELECT MAX() OVER (RANGE BETWEEN 10 FOLLOWING AND UNBOUNDED FOLLOWING)`,
		`S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{FunctionCall{FunctionName T FunctionArguments{} T OverClause{T WindowDefinition{T
				FrameSpec{T FrameSpecBetween{T E{T} TTTT}} T}}}} }}}} T}`,
	)

	runTests(t, cases)
}

func TestFrameSpecBetweenError(t *testing.T) {
	cases := []testCaseError{
		{code: `SELECT MAX() OVER (RANGE BETWEEN UNBOUNDED AND 10 PRECEDING)`, msg: "expecting Preceding, got And"},
		{code: `SELECT MAX() OVER (RANGE BETWEEN CURRENT AND 10 FOLLOWING)`, msg: "expecting Row, got And"},
		{code: `SELECT MAX() OVER (RANGE BETWEEN 10 AND CURRENT ROW)`, msg: "expecting [Preceding Following], got And"},
		{code: `SELECT MAX() OVER (RANGE BETWEEN PRECEDING AND 10 PRECEDING)`, msg: "unexpected Preceding"},
		{code: `SELECT MAX() OVER (RANGE BETWEEN ROW AND 10 FOLLOWING)`, msg: "unexpected Row"},
		{code: `SELECT MAX() OVER (RANGE BETWEEN FOLLOWING AND UNBOUNDED FOLLOWING)`, msg: "unexpected Following"},
		{code: `SELECT MAX() OVER (RANGE BETWEEN AND 10 PRECEDING)`, msg: "unexpected And"},
		{code: `SELECT MAX() OVER (RANGE BETWEEN UNBOUNDED PRECEDING 10 PRECEDING)`, msg: "expecting And, got Numeric"},
		{code: `SELECT MAX() OVER (RANGE BETWEEN CURRENT ROW AND UNBOUNDED)`, msg: "expecting Following, got RightParen"},
		{code: `SELECT MAX() OVER (RANGE BETWEEN 10 PRECEDING AND CURRENT)`, msg: "expecting Row, got RightParen"},
		{code: `SELECT MAX() OVER (RANGE BETWEEN UNBOUNDED PRECEDING AND 10)`, msg: "expecting [Preceding Following], got RightParen"},
		{code: `SELECT MAX() OVER (RANGE BETWEEN UNBOUNDED PRECEDING AND)`, msg: "unexpected RightParen"},
	}

	runTestsError(t, cases)
}

func TestParenExpression(t *testing.T) {
	cases := testCases(
		`SELECT (TRUE = FALSE OR 10 == 20)`,
		"S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{ParenE{T CommaList{E{Or{Equal{TTT} T Equal{TTT}}}} T}} }}}} T}",
		`SELECT (TRUE > FALSE < 10)`,
		"S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{ParenE{T CommaList{E{LessThan{GreaterThan{TTT} TT}}} T}} }}}} T}",
		`SELECT (10 + column_a)`,
		"S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{ParenE{T CommaList{Expression{Add{T T ColumnReference{ColumnName}}}} T}} }}}} T}",
	)

	runTests(t, cases)
}

func TestParenExpressionError(t *testing.T) {
	cases := []testCaseError{
		{code: `SELECT ();`, msg: "unexpected RightParen"},
		{code: `SELECT (, 10);`, msg: "unexpected Comma"},
		{code: `SELECT (10 10);`, msg: "expecting RightParen, got Numeric"},
		{code: `SELECT (10 AS 10);`, msg: "expecting RightParen, got As"},
		{code: `SELECT (10,,10);`, msg: "unexpected Comma"},
		{code: `SELECT (10, AS);`, msg: "unexpected As"},
		{code: `SELECT (10;`, msg: "expecting RightParen, got Semicolon"},
	}

	runTestsError(t, cases)
}

func TestCastExpression(t *testing.T) {
	cases := testCases(
		`SELECT CAST ('10' AS INTEGER)`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{Cast{TT E{T} T TypeName{T} T}} }}}} T}",
	)

	runTests(t, cases)
}

func TestCastExpressionError(t *testing.T) {
	cases := []testCaseError{
		{code: `SELECT CAST 10 AS NUMBER);`, msg: "expecting LeftParen, got Numeric"},
		{code: `SELECT CAST AS NUMBER);`, msg: "expecting LeftParen, got As"},
		{code: `SELECT CAST(10 AS);`, msg: "expecting Identifier, got RightParen"},
		{code: `SELECT CAST(10 NUMBER);`, msg: "expecting As, got Identifier"},
		{code: `SELECT CAST(10 AS NUMBER;`, msg: "expecting RightParen, got Semicolon"},
	}

	runTestsError(t, cases)
}

func TestExists(t *testing.T) {
	cases := testCases(
		`SELECT EXISTS(SELECT 10)`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{Exists{TT SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}} T}} }}}} T}",
		`SELECT EXISTS(WITH table_name_a AS (SELECT 10) SELECT 10)`,
		`S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{Exists{TT SimpleSelect{
					WithClause{T CommaList{CTE{TableName TT SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}} T}}}
					SelectCore{T CommaList{ResultColumn{E{T}}}}} T}} }}}} T}`,
	)

	runTests(t, cases)
}

func TestExistsError(t *testing.T) {
	cases := []testCaseError{
		{code: `SELECT EXISTS(SELECT)`, msg: "unexpected RightParen"},
		{code: `SELECT EXISTS SELECT 10);`, msg: "expecting LeftParen, got Select"},
		{code: `SELECT EXISTS(10);`, msg: "expecting Select, got Numeric"},
		{code: `SELECT EXISTS(10;`, msg: "expecting Select, got Numeric"},
		{code: `SELECT EXISTS (SELECT 10;`, msg: "expecting RightParen, got Semicolon"},
		{code: `SELECT EXISTS ();`, msg: "expecting Select, got RightParen"},
	}

	runTestsError(t, cases)
}

func TestCaseExpression(t *testing.T) {
	cases := testCases(
		`SELECT CASE WHEN 10 THEN TRUE ELSE FALSE END`,
		"S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{Case{T When {T E{T} T E{T}} Else {T E{T}} T}}}}}} T}",
		`SELECT CASE a WHEN 10 THEN TRUE END`,
		"S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{Case{T E{ColRef{ColName}} When {T E{T} T E{T}} T}}}}}} T}",
		`SELECT CASE WHEN 10 THEN TRUE WHEN 20 THEN 'a' ELSE FALSE END`,
		"S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{Case{T When {T E{T} T E{T}} When {T E{T} T E{T}} Else {T E{T}} T}}}}}} T}",
	)

	runTests(t, cases)
}

func TestCaseExpressionError(t *testing.T) {
	cases := []testCaseError{
		{code: `SELECT CASE ELSE 10`, msg: "expecting When, got Else"},
		{code: `SELECT CASE WHEN THEN 10 END`, msg: "unexpected Then"},
		{code: `SELECT CASE WHEN 10 10 END`, msg: "expecting Then, got Numeric"},
		{code: `SELECT CASE WHEN 10 THEN END`, msg: "unexpected End"},
		{code: `SELECT CASE WHEN 10 THEN 10 ELSE END`, msg: "unexpected End"},
	}

	runTestsError(t, cases)
}

func TestRaise(t *testing.T) {
	cases := testCases(
		`SELECT RAISE (IGNORE)`, "S{ SimpleSelect{SelectCore{ T CommaList{ResultColumn{E{Raise{TTTT}}}}}} T}",
		`SELECT RAISE (ROLLBACK, 'error message')`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{Raise{TTTT ErrorMessage{E{T}} T}}}}}} T}",
	)

	runTests(t, cases)
}

func TestRaiseError(t *testing.T) {
	cases := []testCaseError{
		{code: `SELECT RAISE;`, msg: "expecting LeftParen, got Semicolon"},
		{code: `SELECT RAISE IGNORE);`, msg: "expecting LeftParen, got Ignore"},
		{code: `SELECT RAISE(IGNORE;`, msg: "expecting RightParen, got Semicolon"},
		{code: `SELECT RAISE();`, msg: "expecting [Ignore Rollback Abort Fail], got RightParen"},
		{code: `SELECT RAISE(, 'error');`, msg: "expecting [Ignore Rollback Abort Fail], got Comma"},
		{code: `SELECT RAISE(ROLLBACK 'error');`, msg: "expecting Comma, got String"},
		{code: `SELECT RAISE(ROLLBACK, );`, msg: "unexpected RightParen"},
		{code: `SELECT RAISE(ROLLBACK, 'error';`, msg: "expecting RightParen, got Semicolon"},
	}

	runTestsError(t, cases)
}

func TestInsert(t *testing.T) {
	cases := testCases(
		`INSERT INTO table_name(name) VALUES('Go')`,
		`S{ Insert{TT TableName T CommaList{ColumnName} T T
			InsertValuesList{CommaList{InsertValuesItem{T CommaList{E{T}} T}}}} T}`,
		`REPLACE INTO table_name(name) VALUES('Go')`,
		`S{ Insert{TT TableName T CommaList{ColumnName} T T
			InsertValuesList{CommaList{InsertValuesItem{T CommaList{E{T}} T}}}} T}`,
		`INSERT OR ABORT INTO temp.table_name(name) VALUES('Go')`,
		`S{ Insert{T TT T SchemaName T TableName T CommaList{ColumnName} T T
		 	InsertValuesList{CommaList{InsertValuesItem{T CommaList{E{T}} T}}}} T}`,
		`INSERT OR FAIL INTO table_name AS table_alias (name) VALUES('Go')`,
		`S{ Insert{T TT T TableName T TableAlias T CommaList{ColumnName} T T
		 	InsertValuesList{CommaList{InsertValuesItem{T CommaList{E{T}} T}}}} T}`,
		`INSERT INTO table_name(name) VALUES('Go') ON CONFLICT DO NOTHING`,
		`S{ Insert{TT TableName T CommaList{ColumnName} T T
			InsertValuesList{CommaList{InsertValuesItem{T CommaList{E{T}} T}}}
			UpsertClause{UpsertClauseItem{TT TT}}} T}`,
		`INSERT INTO table_name(name) SELECT 'Go'`,
		"S{ Insert{TT TableName T CommaList{ColumnName} T SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}}} T}",
		`INSERT INTO table_name(name) SELECT 'Go' ON CONFLICT DO NOTHING`,
		"S{ Insert{TT TableName T CommaList{ColumnName} T SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}} UpsertClause{UpsertClauseItem{TT TT}}} T}",
		`INSERT INTO table_name(name) DEFAULT VALUES`,
		"S{ Insert{TT TableName T CommaList{ColumnName} T TT} T}",
		`INSERT INTO table_name(name) DEFAULT VALUES RETURNING *`,
		"S{ Insert{TT TableName T CommaList{ColumnName} T TT ReturningClause{T CommaList{ReturningItem{T}}}} T}",
		`INSERT INTO table_name(name) WITH cte AS (SELECT 10) SELECT 'Go'`,
		`S{ Insert{TT TableName T CommaList{ColumnName} T SimpleSelect{
			WithClause{T CommaList{CTE{TableName T T SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}} T}} }
			SelectCore{T CommaList{ResultColumn{E{T}}}}}} T}`,
	)

	runTests(t, cases)
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

	runTestsError(t, cases)
}

func TestUpsertClause(t *testing.T) {
	cases := testCases(
		`INSERT INTO table_name(name) VALUES('Go') ON CONFLICT DO NOTHING ON CONFLICT DO NOTHING`,
		`S{ Insert{TT TableName T CommaList{ColumnName} T T
			InsertValuesList{CommaList{InsertValuesItem{T CommaList{E{T}} T}}}
			UpsertClause{UpsertClauseItem{TT TT} UpsertClauseItem{TT TT}}} T}`,
	)

	runTests(t, cases)
}

func TestUpsertClauseItem(t *testing.T) {
	cases := testCases(
		`INSERT INTO table_name(name) VALUES('Go') ON CONFLICT DO NOTHING`,
		`S{ Insert{TT TableName T CommaList{ColumnName} T T
			InsertValuesList{CommaList{InsertValuesItem{T CommaList{E{T}} T}}}
			UpsertClause{UpsertClauseItem{TT TT}}} T}`,
		`INSERT INTO table_name(name) VALUES('Go') ON CONFLICT DO UPDATE SET col='value'`,
		`S{ Insert{TT TableName T CommaList{ColumnName} T T
			InsertValuesList{CommaList{InsertValuesItem{T CommaList{E{T}} T}}}
			UpsertClause{UpsertClauseItem{TT TT T CommaList{UpdateSetItem{ColName T E{T}}}}}} T}`,
		`INSERT INTO table_name(name) VALUES('Go') ON CONFLICT DO UPDATE SET col_a='value_a', col_b='value_b'`,
		`S{ Insert{TT TableName T CommaList{ColumnName} T T
			InsertValuesList{CommaList{InsertValuesItem{T CommaList{E{T}} T}}}
			UpsertClause{UpsertClauseItem{TT TT T CommaList{UpdateSetItem{ColName T E{T}} T UpdateSetItem{ColName T E{T}}}}}} T}`,
		`INSERT INTO table_name(name) VALUES('Go') ON CONFLICT DO UPDATE SET (col_a, col_b)='value_ab', col_c='value_c'`,
		`S{ Insert{TT TableName T CommaList{ColumnName} T T
			InsertValuesList{CommaList{InsertValuesItem{T CommaList{E{T}} T}}}
			UpsertClause{UpsertClauseItem{TT TT T CommaList{UpdateSetItem{T CommaList{ColName T ColName} T T E{T}} T UpdateSetItem{ColName T E{T}}} }}} T}`,
		`INSERT INTO table_name(name) VALUES('Go') ON CONFLICT DO UPDATE SET col='value' WHERE condition=true`,
		`S{ Insert{TT TableName T CommaList{ColumnName} T T
			InsertValuesList{CommaList{InsertValuesItem{T CommaList{E{T}} T}}}
			UpsertClause{UpsertClauseItem{TT TT T CommaList{UpdateSetItem{ColName T E{T}}} WhereClause{T E{Equal{ColRef{ColName} TT}}} }}} T}`,
		`INSERT INTO table_name(name) VALUES('Go') ON CONFLICT (col) DO NOTHING`,
		`S{ Insert{TT TableName T CommaList{ColumnName} T T
			InsertValuesList{CommaList{InsertValuesItem{T CommaList{E{T}} T}}}
			UpsertClause{UpsertClauseItem{TT T CommaList{IndexedColumn{E{ColRef{ColName}}}} T TT}}} T}`,
		`INSERT INTO table_name(name) VALUES('Go') ON CONFLICT (col_a, col_b) WHERE condition=true DO NOTHING`,
		`S{ Insert{TT TableName T CommaList{ColumnName} T T
			InsertValuesList{CommaList{InsertValuesItem{T CommaList{E{T}} T}}}
			UpsertClause{UpsertClauseItem{TT T CommaList{IndexedColumn{E{ColRef{ColName}}} T IndexedColumn{E{ColRef{ColName}}}} T
			WhereClause{T E{Equal{ColRef{ColName} T T}}} TT}}} T}`,
	)

	runTests(t, cases)
}

func TestUpsertClauseItemError(t *testing.T) {
	cases := []testCaseError{
		{code: `INSERT INTO table_name(name) VALUES('Go') ON CONFLICT DO UPDATE SET (col_a, col_b ='value_ab', col_c='value_c'`, msg: "expecting RightParen, got Equal"},
		{code: `INSERT INTO table_name(name) VALUES('Go') ON CONFLICT DO UPDATE SET ()='value_ab', col_c='value_c'`, msg: "expecting Identifier, got RightParen"},
		{code: `INSERT INTO table_name(name) VALUES('Go') ON CONFLICT DO UPDATE col='value'`, msg: "expecting Set, got Identifier"},
		{code: `INSERT INTO table_name(name) VALUES('Go') ON CONFLICT DO UPDATE SET col='value' WHERE`, msg: "unexpected EOF"},
		{code: `INSERT INTO table_name(name) VALUES('Go') ON CONFLICT DO UPDATE SET WHERE condition=true`, msg: "expecting [Identifier LeftParen], got Where"},
		{code: `INSERT INTO table_name(name) VALUES('Go') ON CONFLICT DO UPDATE SET col 'value'`, msg: "expecting Equal, got String"},
		{code: `INSERT INTO table_name(name) VALUES('Go') ON CONFLICT DO UPDATE SET col=`, msg: "unexpected EOF"},
		{code: `INSERT INTO table_name(name) VALUES('Go') ON DO NOTHING`, msg: "expecting Conflict, got Do"},
		{code: `INSERT INTO table_name(name) VALUES('Go') ON CONFLICT NOTHING`, msg: "expecting Do, got Nothing"},
		{code: `INSERT INTO table_name(name) VALUES('Go') ON CONFLICT DO`, msg: "expecting [Nothing Update], got EOF"},
		{code: `INSERT INTO table_name(name) VALUES('Go') ON CONFLICT (col DO NOTHING`, msg: "expecting RightParen, got Do"},
		{code: `INSERT INTO table_name(name) VALUES('Go') ON CONFLICT (col_a) WHERE DO NOTHING`, msg: "unexpected Do"},
	}

	runTestsError(t, cases)
}

func TestPragma(t *testing.T) {
	cases := testCases(
		`PRAGMA pragma_name`, "S{ Pragma{T PragmaName} T}",
		`PRAGMA temp.pragma_name`, "S{ Pragma{T SchemaName T PragmaName} T}",
		`PRAGMA pragma_name=10`, "S{ Pragma{T PragmaName T PragmaValue{T}} T}",
		`PRAGMA pragma_name=-10`, "S{ Pragma{T PragmaName T PragmaValue{TT}} T}",
		`PRAGMA pragma_name=+SELECT`, "S{ Pragma{T PragmaName T PragmaValue{TT}} T}",
		`PRAGMA pragma_name='yes'`, "S{ Pragma{T PragmaName T PragmaValue{T}} T}",
		`PRAGMA pragma_name('yes')`, "S{ Pragma{T PragmaName T PragmaValue{T} T} T}",
	)

	runTests(t, cases)
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

	runTestsError(t, cases)
}

func TestReindex(t *testing.T) {
	cases := testCases(
		`REINDEX`, "S{ Reindex{T} T}",
		`REINDEX table_name`, "S{ Reindex{T CollationTableOrIndexName} T}",
		`REINDEX temp.table_name`, "S{ Reindex{T SchemaName T TableOrIndexName} T}",
	)

	runTests(t, cases)
}

func TestReindexError(t *testing.T) {
	cases := []testCaseError{
		{code: `REINDEX schema_name.`, msg: "expecting Identifier, got EOF"},
	}

	runTestsError(t, cases)
}

func TestRelease(t *testing.T) {
	cases := testCases(
		`RELEASE savepoint_name`, "S{ Release{T SavepointName} T}",
		`RELEASE SAVEPOINT savepoint_name`, "S{ Release{TT SavepointName} T}",
	)

	runTests(t, cases)
}

func TestReleaseError(t *testing.T) {
	cases := []testCaseError{
		{code: `RELEASE`, msg: "expecting Identifier, got EOF"},
		{code: `RELEASE SAVEPOINT`, msg: "expecting Identifier, got EOF"},
	}

	runTestsError(t, cases)
}

func TestSavepoint(t *testing.T) {
	cases := testCases(
		`SAVEPOINT savepoint_name`, "S{ Savepoint{T SavepointName} T}",
	)

	runTests(t, cases)
}

func TestSavepointError(t *testing.T) {
	cases := []testCaseError{
		{code: `SAVEPOINT`, msg: "expecting Identifier, got EOF"},
	}

	runTestsError(t, cases)
}

func TestSelectStatement(t *testing.T) {
	cases := testCases(
		`SELECT 10`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}} T}",
		`SELECT 10 ORDER BY 20`, `S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}
			OrderByClause{TT CommaList{OrderingTerm{E{T}}}}}T}`,
		`SELECT 10 LIMIT 10`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}} LimitClause{T E{T}}} T}",
		`SELECT 10 ORDER BY 20 LIMIT 10`, `S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}
			OrderByClause{TT CommaList{OrderingTerm{E{T}}}} LimitClause{T E{T}} } T}`,
		`SELECT 10 UNION ALL SELECT 20`,
		`S{ CompoundSelect{SelectCore{T CommaList{ResultColumn{E{T}}}} CompoundOperator{TT} 
			SelectCore{T CommaList{ResultColumn{E{T}}}} } T}`,
		`WITH table_name AS (SELECT 10) SELECT 10`,
		`S{ SimpleSelect{WithClause{T CommaList{CTE{TableName TT
			SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}} T}}}
			SelectCore{T CommaList{ResultColumn{E{T}}}}} T}`,
		`WITH table_name AS (SELECT 10) SELECT 10 INTERSECT SELECT 20`,
		`S{ CompoundSelect{WithClause{T CommaList{CTE{TableName TT SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}} T}}}
			SelectCore{T CommaList{ResultColumn{E{T}}}} CompoundOperator{T} SelectCore{T CommaList{ResultColumn{E{T}}}}} T}`,
		`WITH table_name AS (SELECT 10) SELECT 10 EXCEPT SELECT 20`,
		`S{ CompoundSelect{WithClause{T CommaList{CTE{TableName TT SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}} T}}}
			SelectCore{T CommaList{ResultColumn{E{T}}}} CompoundOperator{T} SelectCore{T CommaList{ResultColumn{E{T}}}}} T}`,
	)

	runTests(t, cases)
}

func TestSelectStatementError(t *testing.T) {
	cases := []testCaseError{
		{code: `SELECT 10 UNION`, msg: "expecting Select, got EOF"},
		{code: `SELECT 10, LIMIT 10`, msg: "unexpected Limit"},
		{code: `SELECT 10 LIMIT`, msg: "unexpected EOF"},
	}

	runTestsError(t, cases)
}

func TestSelectCore(t *testing.T) {
	cases := testCases(
		`SELECT 10`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}} T}",
		`SELECT ALL 10, 20`, "S{ SimpleSelect{SelectCore{T T CommaList{ResultColumn{E{T}} T ResultColumn{E{T}}}}} T}",
		`SELECT DISTINCT 10`, "S{ SimpleSelect{SelectCore{T T CommaList{ResultColumn{E{T}}}}} T}",
		`SELECT 10 FROM table_name`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}} FromClause{T JoinClause{TableOrSubquery{TableName}}}}} T}",
		`SELECT 10 WHERE FALSE`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}} WhereClause{T E{T}}}} T}",
		`SELECT 10 GROUP BY 20`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}} GroupByClause{TT CommaList{E{T}}}}} T}",
		`SELECT 10 HAVING 20`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}} HavingClause{T E{T}}}} T}",
		`SELECT 10 WINDOW window_a AS (PARTITION BY 10)`,
		`S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}
			WindowClause{T CommaList{WindowClauseItem{WindowName T WindowDefinition{T PartitionBy{TT CommaList{E{T}}} T}}}}} } T}`,
		`SELECT 10 WINDOW window_a AS (PARTITION BY 10), window_b AS (PARTITION BY 10)`,
		`S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}
			WindowClause{T CommaList{
				WindowClauseItem{WindowName T WindowDefinition{T PartitionBy{TT CommaList{E{T}}} T}} T
				WindowClauseItem{WindowName T WindowDefinition{T PartitionBy{TT CommaList{E{T}}} T}} }} }} T}`,
		`VALUES (10), (20)`, `S{ SimpleSelect{SelectCore{ValuesClause{T CommaList{ValuesItem{T CommaList{E{T}} T} T ValuesItem{T CommaList{E{T}} T}}}}} T}`,
	)

	runTests(t, cases)
}

func TestSelectCoreError(t *testing.T) {
	cases := []testCaseError{
		{code: `SELECT`, msg: "unexpected EOF"},
	}

	runTestsError(t, cases)
}

func TestResultColumn(t *testing.T) {
	cases := testCases(
		`SELECT col`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{ColRef{ColName}}}} }} T}",
		`SELECT col col_alias`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{ColRef{ColName}} ColumnAlias}} }} T}",
		`SELECT col AS alias`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{E{ColRef{ColName}} T ColumnAlias}} }} T}",
		`SELECT table_name.*`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{TableName T T}} }} T}",
		`SELECT *`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{T}} }} T}",
	)

	runTests(t, cases)
}

func TestResultColumnError(t *testing.T) {
	cases := []testCaseError{
		{code: `SELECT col AS`, msg: "expecting Identifier, got EOF"},
		{code: `SELECT insert`, msg: "unexpected Insert"},
	}

	runTestsError(t, cases)
}

func TestFromClause(t *testing.T) {
	cases := testCases(
		`SELECT * FROM table_name`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{T}} FromClause{T JoinClause{TableOrSubquery{TableName}}} }} T}",
	)

	runTests(t, cases)
}

func TestFromClauseError(t *testing.T) {
	cases := []testCaseError{
		{code: `SELECT * FROM 10`, msg: "expecting LeftParen, got Numeric"},
	}

	runTestsError(t, cases)
}

func TestJoinClause(t *testing.T) {
	cases := testCases(
		`SELECT * FROM table_name`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{T}} FromClause{T JoinClause{TableOrSubquery{TableName}}} }} T}",
		`SELECT * FROM table_name_1, table_name_2`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{T}} FromClause{T JoinClause{TableOrSubquery{TableName} JoinOp{T} TableOrSubquery{TableName}}} }} T}",
		`SELECT * FROM table_name_1 JOIN table_name_2`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{T}} FromClause{T JoinClause{TableOrSubquery{TableName} JoinOp{T} TableOrSubquery{TableName}}} }} T}",
		`SELECT * FROM table_name_1 FULL JOIN table_name_2 ON TRUE`,
		"S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{T}} FromClause{T JoinClause{TableOrSubquery{TableName} JoinOp{TT} TableOrSubquery{TableName} JoinConstr{T E{T}}} }}} T}",
		`SELECT * FROM table_name_1 JOIN table_name_2, table_name_2 ON a > 2`,
		`S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{T}} FromClause{T JoinClause{TableOrSubquery{TableName} JoinOp{T} TableOrSubquery{TableName}
			JoinOp{T} TableOrSubquery{TableName} JoinConstr{T E{GreaterThan{ColRef{ColName} TT}}}} } }} T}`,
	)

	runTests(t, cases)
}

func TestJoinClauseError(t *testing.T) {
	cases := []testCaseError{
		{code: `SELECT * FROM table_name JOIN`, msg: "expecting LeftParen, got EOF"},
	}

	runTestsError(t, cases)
}

func TestJoinOperator(t *testing.T) {
	cases := testCases(
		`SELECT * FROM a, b`,
		"S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{T}} FromClause{T JoinClause{TableOrSubquery{TableName} JoinOp{T} TableOrSubquery{TableName}}} }} T}",
		`SELECT * FROM a JOIN b`,
		"S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{T}} FromClause{T JoinClause{TableOrSubquery{TableName} JoinOp{T} TableOrSubquery{TableName}}} }} T}",
		`SELECT * FROM a CROSS JOIN b`,
		"S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{T}} FromClause{T JoinClause{TableOrSubquery{TableName} JoinOp{TT} TableOrSubquery{TableName}}} }} T}",
		`SELECT * FROM a FULL JOIN b`,
		"S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{T}} FromClause{T JoinClause{TableOrSubquery{TableName} JoinOp{TT} TableOrSubquery{TableName}}} }} T}",
		`SELECT * FROM a INNER JOIN b`,
		"S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{T}} FromClause{T JoinClause{TableOrSubquery{TableName} JoinOp{TT} TableOrSubquery{TableName}}} }} T}",
		`SELECT * FROM a LEFT NATURAL JOIN b`,
		"S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{T}} FromClause{T JoinClause{TableOrSubquery{TableName} JoinOp{TTT} TableOrSubquery{TableName}}} }} T}",
		`SELECT * FROM a OUTER RIGHT JOIN b`,
		"S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{T}} FromClause{T JoinClause{TableOrSubquery{TableName} JoinOp{TTT} TableOrSubquery{TableName}}} }} T}",
		`SELECT * FROM a INNER OUTER JOIN b`,
		"S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{T}} FromClause{T JoinClause{TableOrSubquery{TableName} JoinOp{TTT} TableOrSubquery{TableName}}} }} T}",
		`SELECT * FROM a LEFT NATURAL OUTER JOIN b`,
		"S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{T}} FromClause{T JoinClause{TableOrSubquery{TableName} JoinOp{TTTT} TableOrSubquery{TableName}}} }} T}",
		`SELECT * FROM a FULL CROSS OUTER JOIN b`,
		"S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{T}} FromClause{T JoinClause{TableOrSubquery{TableName} JoinOp{TTTT} TableOrSubquery{TableName}}} }} T}",
	)

	runTests(t, cases)
}

func TestJoinOperatorError(t *testing.T) {
	cases := []testCaseError{
		{code: `SELECT * FROM a CROSS b`, msg: "expecting Join, got Identifier"},
		{code: `SELECT * FROM a FULL b`, msg: "expecting Join, got Identifier"},
		{code: `SELECT * FROM a INNER b`, msg: "expecting Join, got Identifier"},
		{code: `SELECT * FROM a LEFT b`, msg: "expecting Join, got Identifier"},
		{code: `SELECT * FROM a OUTER b`, msg: "expecting Join, got Identifier"},
		{code: `SELECT * FROM a RIGHT b`, msg: "expecting Join, got Identifier"},
		{code: `SELECT * FROM a NATURAL b`, msg: "expecting Join, got Identifier"},
	}

	runTestsError(t, cases)
}

func TestJoinConstraint(t *testing.T) {
	cases := testCases(
		`SELECT * FROM a JOIN b ON true`,
		"S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{T}} FromClause{T JoinClause{TableOrSubquery{TableName} JoinOp{T} TableOrSubquery{TableName} JoinConstr{T E{T}}}} }} T}",
		`SELECT * FROM a JOIN b USING (col_a, col_b)`,
		"S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{T}} FromClause{T JoinClause{TableOrSubquery{TableName} JoinOp{T} TableOrSubquery{TableName} JoinConstr{T T CommaList{ColName T ColName} T}}} }} T}",
	)

	runTests(t, cases)
}

func TestJoinConstraintError(t *testing.T) {
	cases := []testCaseError{
		{code: `SELECT * FROM a JOIN b ON`, msg: "unexpected EOF"},
		{code: `SELECT * FROM a JOIN b USING col_a, col_b)`, msg: "expecting LeftParen, got Identifier"},
		{code: `SELECT * FROM a JOIN b USING (col_a`, msg: "expecting RightParen, got EOF"},
		{code: `SELECT * FROM a JOIN b USING ()`, msg: "expecting Identifier, got RightParen"},
	}

	runTestsError(t, cases)
}

func TestTableOrSubquery(t *testing.T) {
	cases := testCases(
		`SELECT * FROM table_name`,
		"S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{T}} FromClause{T JoinClause{TableOrSubquery{TableName}} }}} T}",
		`SELECT * FROM table_name table_alias`,
		"S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{T}} FromClause{T JoinClause{TableOrSubquery{TableName TableAlias}} }}} T}",
		`SELECT * FROM table_name AS table_alias`,
		"S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{T}} FromClause{T JoinClause{TableOrSubquery{TableName T TableAlias}} }}} T}",
		`SELECT * FROM table_name INDEXED BY index_name`,
		"S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{T}} FromClause{T JoinClause{TableOrSubquery{TableName TT IndexName}} }}} T}",
		`SELECT * FROM table_name NOT INDEXED`,
		"S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{T}} FromClause{T JoinClause{TableOrSubquery{TableName TT}} }}} T}",
		`SELECT * FROM temp.table_name AS table_alias`,
		"S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{T}} FromClause{T JoinClause{TableOrSubquery{SchemaName T TableName T TableAlias}} }}} T}",
		`SELECT * FROM table_function_name(10)`,
		"S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{T}} FromClause{T JoinClause{TableOrSubquery{TableFunctionName T CommaList{E{T}} T}} }}} T}",
		`SELECT * FROM table_function_name(10, 20) AS table_alias`,
		"S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{T}} FromClause{T JoinClause{TableOrSubquery{TableFunctionName T CommaList{E{T} T E{T}} T T TableAlias}} }}} T}",
		`SELECT * FROM table_function_name(10) table_alias`,
		"S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{T}} FromClause{T JoinClause{TableOrSubquery{TableFunctionName T CommaList{E{T}} T TableAlias}} }}} T}",
		`SELECT * FROM (SELECT 10)`,
		"S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{T}} FromClause{T JoinClause{TableOrSubquery{T SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}} T}} }}} T}",
		`SELECT * FROM (SELECT 10) AS table_alias`,
		"S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{T}} FromClause{T JoinClause{TableOrSubquery{T SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}} T T TableAlias}} }}} T}",
		`SELECT * FROM (SELECT 10) table_alias`,
		"S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{T}} FromClause{T JoinClause{TableOrSubquery{T SimpleSelect{SelectCore{T CommaList{ResultColumn{E{T}}}}} T TableAlias}} }}} T}",
		`SELECT * FROM (table_name)`,
		"S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{T}} FromClause{T JoinClause{TableOrSubquery{T JoinClause{TableOrSubquery{TableName}} T}} }}} T}",
		`SELECT * FROM (table_name_1 JOIN table_name_2, table_name_2 ON a > 2)`,
		`S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{T}} FromClause{T JoinClause{TableOrSubquery{T JoinClause{TableOrSubquery{TableName} JoinOp{T} TableOrSubquery{TableName}
			JoinOp{T} TableOrSubquery{TableName} JoinConstr{T E{GreaterThan{ColRef{ColName} TT}}} } T}} }}} T}`,
		`SELECT * FROM (WITH go AS (SELECT rust) SELECT 10)`,
		`S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{T}} FromClause{T JoinClause{TableOrSubquery{T SimpleSelect{
			WithClause{T CommaList{CTE{TableName T T SimpleSelect{SelectCore{T CommaList{ResultColumn{E{ColRef{ColName}}}}}} T }}}
			SelectCore{T CommaList{ResultColumn{E{T}}}}} T}} }}} T}`,
	)

	runTests(t, cases)
}

func TestTableOrSubqueryError(t *testing.T) {
	cases := []testCaseError{
		{code: `SELECT * FROM  .table_name`, msg: "expecting LeftParen, got Dot"},
		{code: `SELECT * FROM  schema_name.`, msg: "expecting [Identifier Temp], got EOF"},
		{code: `SELECT * FROM  table_name AS`, msg: "expecting Identifier, got EOF"},
		{code: `SELECT * FROM  table_name INDEXED index_name`, msg: "expecting By, got Identifier"},
		{code: `SELECT * FROM  table_name NOT`, msg: "expecting Indexed, got EOF"},
		{code: `SELECT * FROM  table_function_name()`, msg: "unexpected RightParen"},
		{code: `SELECT * FROM  table_function_name(10`, msg: "expecting RightParen, got EOF"},
		{code: `SELECT * FROM  table_function_name(10, 20) AS`, msg: "expecting Identifier, got EOF"},
		{code: `SELECT * FROM  (SELECT 10`, msg: "expecting RightParen, got EOF"},
		{code: `SELECT * FROM  (SELECT 10) AS`, msg: "expecting Identifier, got EOF"},
		{code: `SELECT * FROM  (table_name`, msg: "expecting RightParen, got EOF"},
		{code: `SELECT * FROM  ()`, msg: "expecting [With Select Identifier Temp LeftParen], got RightParen"},
	}

	runTestsError(t, cases)
}

func TestGroupByClause(t *testing.T) {
	cases := testCases(
		`SELECT * GROUP BY 10`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{T}} GroupByClause{TT CommaList{E{T}}} }} T}",
	)

	runTests(t, cases)
}

func TestGroupByClauseError(t *testing.T) {
	cases := []testCaseError{
		{code: `SELECT * GROUP 10`, msg: "expecting By, got Numeric"},
		{code: `SELECT * GROUP BY `, msg: "unexpected EOF"},
	}

	runTestsError(t, cases)
}

func TestHavingClause(t *testing.T) {
	cases := testCases(
		`SELECT * HAVING 10`, "S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{T}} HavingClause{T E{T}} }} T}",
	)

	runTests(t, cases)
}

func TestHavingClauseError(t *testing.T) {
	cases := []testCaseError{
		{code: `SELECT * HAVING `, msg: "unexpected EOF"},
	}

	runTestsError(t, cases)
}

func TestWindowClause(t *testing.T) {
	cases := testCases(
		`SELECT * WINDOW window_a AS (PARTITION BY 10)`,
		"S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{T}} WindowClause{T CommaList{WindowClauseItem{WindowName T WindowDefinition{T PartitionBy{TT CommaList{E{T}}} T}}}} }} T}",
	)

	runTests(t, cases)
}

func TestWindowClauseError(t *testing.T) {
	cases := []testCaseError{
		{code: `SELECT * WINDOW `, msg: "expecting Identifier, got EOF"},
	}

	runTestsError(t, cases)
}

func TestWindowClauseItem(t *testing.T) {
	cases := testCases(
		`SELECT * WINDOW window_name AS (PARTITION BY 10)`,
		"S{ SimpleSelect{SelectCore{T CommaList{ResultColumn{T}} WindowClause{T CommaList{WindowClauseItem{WindowName T WindowDefinition{T PartitionBy{TT CommaList{E{T}}} T}} }}}} T}",
	)

	runTests(t, cases)
}

func TestWindowClauseItemError(t *testing.T) {
	cases := []testCaseError{
		{code: `SELECT * WINDOW window_name (PARTITION BY 10)`, msg: "expecting As, got LeftParen"},
		{code: `SELECT * WINDOW window_name`, msg: "expecting As, got EOF"},
	}

	runTestsError(t, cases)
}

func TestValuesClause(t *testing.T) {
	cases := testCases(
		`VALUES (10), (20)`, "S{ SimpleSelect{SelectCore{ValuesClause{T CommaList{ValuesItem{T CommaList{E{T}} T} T ValuesItem{T CommaList{E{T}} T}}} }} T}",
	)

	runTests(t, cases)
}

func TestValuesClauseError(t *testing.T) {
	cases := []testCaseError{
		{code: `VALUES `, msg: "expecting LeftParen, got EOF"},
	}

	runTestsError(t, cases)
}

func TestValuesItem(t *testing.T) {
	cases := testCases(
		`VALUES (10, 20)`, "S{ SimpleSelect{SelectCore{ValuesClause{T CommaList{ValuesItem{T CommaList{E{T} T E{T}} T}}} }} T}",
	)

	runTests(t, cases)
}

func TestValuesItemError(t *testing.T) {
	cases := []testCaseError{
		{code: `VALUES () `, msg: "unexpected RightParen"},
	}

	runTestsError(t, cases)
}

func TestLimitClause(t *testing.T) {
	cases := testCases(
		`UPDATE table_name SET col_name=10 LIMIT 10`,
		"S{ Update{T QualifiedTableName{TableName} T CommaList{UpdateSetItem{ColName T E{T}}} LimitClause{T E{T}}} T}",
		`UPDATE table_name SET col_name=10 LIMIT 10, 20`,
		"S{ Update{T QualifiedTableName{TableName} T CommaList{UpdateSetItem{ColName T E{T}}} LimitClause{T E{T} T E{T}}} T}",
		`UPDATE table_name SET col_name=10 LIMIT 10 OFFSET 20`,
		"S{ Update{T QualifiedTableName{TableName} T CommaList{UpdateSetItem{ColName T E{T}}} LimitClause{T E{T} T E{T}}} T}",
	)

	runTests(t, cases)
}

func TestLimitClauseError(t *testing.T) {
	cases := []testCaseError{
		{code: `UPDATE table_name SET col_name=10 LIMIT 10,`, msg: "unexpected EOF"},
		{code: `UPDATE table_name SET col_name=10 LIMIT 10 OFFSET`, msg: "unexpected EOF"},
	}

	runTestsError(t, cases)
}

func TestUpdate(t *testing.T) {
	cases := testCases(
		`UPDATE table_name SET col_name=10`, "S{Update{T QualifiedTableName{TableName} T CommaList{UpdateSetItem{ColName T E{T}}}} T}",
		`UPDATE table_name SET a=10, b=20`,
		"S{Update{T QualifiedTableName{TableName} T CommaList{UpdateSetItem{ColName T E{T}} T UpdateSetItem{ColName T E{T}}}} T}",
		`UPDATE OR ABORT table_name SET col_name=10`, "S{ Update{T TT QualifiedTableName{TableName} T CommaList{UpdateSetItem{ColName T E{T}}}} T}",
		`UPDATE table_name SET (a, b)=(10, 20)`,
		`S{ Update{T QualifiedTableName{TableName} T CommaList{UpdateSetItem{T CommaList{ColName T ColName} TT
				E{ParenExpression{T CommaList{E{T} T E{T}} T}}}}} T}`,
		`UPDATE table_name SET col_name=10 FROM table_name_2`,
		"S{ Update{T QualifiedTableName{TableName} T CommaList{UpdateSetItem{ColName T E{T}}} FromClause{T JoinClause{TableOrSubquery{TableName}}}} T}",
		`UPDATE table_name SET col_name=10 WHERE TRUE`,
		"S{ Update{T QualifiedTableName{TableName} T CommaList{UpdateSetItem{ColName T E{T}}} WhereClause{T E{T}}} T}",
		`UPDATE table_name SET col_name=10 RETURNING *`,
		"S{ Update{T QualifiedTableName{TableName} T CommaList{UpdateSetItem{ColName T E{T}}} ReturningClause{T CommaList{ReturningItem{T}}}} T}",
		`UPDATE table_name SET col_name=10 ORDER BY 20`,
		"S{ Update{T QualifiedTableName{TableName} T CommaList{UpdateSetItem{ColName T E{T}}} OrderByClause{TT CommaList{OrderingTerm{E{T}}}}} T}",
		`UPDATE table_name SET col_name=10 LIMIT 10`,
		"S{ Update{T QualifiedTableName{TableName} T CommaList{UpdateSetItem{ColName T E{T}}} LimitClause{T E{T}}} T}",
	)

	runTests(t, cases)
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

	runTestsError(t, cases)
}

func TestVacuum(t *testing.T) {
	cases := testCases(
		`VACUUM`, "S{ Vacuum{T} T}",
		`VACUUM temp`, "S{ Vacuum{T SchemaName} T}",
		`VACUUM INTO 'db.db'`, "S{ Vacuum{T T FileName{E{T}}} T}",
	)

	runTests(t, cases)
}

func TestVacuumError(t *testing.T) {
	cases := []testCaseError{
		{code: `VACUUM INTO`, msg: "unexpected EOF"},
		{code: `VACUUM schame_name INTO`, msg: "unexpected EOF"},
	}

	runTestsError(t, cases)
}

func runTests(t *testing.T, cases []testCase) {
	for i, c := range cases {
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
			expected, err := toParsetree(c.tree)
			if err != nil {
				t.Fatal(err)
			}

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

func runTestsError(t *testing.T, cases []testCaseError) {
	for i, c := range cases {
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
			p.SQLStatement()
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
