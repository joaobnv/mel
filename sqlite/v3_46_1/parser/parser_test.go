package parser

import (
	"bytes"
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

func TestParserExplain(t *testing.T) {
	t.Parallel()
	cases := []struct {
		code string
		tree string
	}{
		{
			code: `EXPLAIN ALTER TABLE table_a ADD COLUMN column_b NOT NULL AS (10) VIRTUAL;`,
			tree: `SQLStatement {Explain {T AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT} ColumnConstraint{T T Expression{T} T T} }}}} T}`,
		}, {
			code: `EXPLAIN QUERY PLAN ALTER TABLE table_a ADD COLUMN column_b NOT NULL AS (10) VIRTUAL;`,
			tree: `SQLStatement {ExplainQueryPlan {TTT AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT} ColumnConstraint{T T Expression{T} T T} }}}} T}`,
		},
	}

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

func TestParserExplainError(t *testing.T) {
	t.Parallel()
	cases := []struct {
		code string
		tree string
	}{
		{
			code: `EXPLAIN QUERY ALTER TABLE table_a ADD COLUMN column_b NOT NULL AS (10) VIRTUAL;`,
			tree: `SQLStatement {ExplainQueryPlan {TT !ErrorMissing AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT} ColumnConstraint{T T Expression{T} T T} }}}} T}`,
		},
	}

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

func TestParserAlterTable(t *testing.T) {
	t.Parallel()
	cases := []struct {
		code string
		tree string
	}{
		{
			code: `ALTER TABLE table_a RENAME TO table_b;`,
			tree: `SQLStatement {AlterTable {TT TableName RenameTo {TT TableName}} T}`,
		}, {
			code: `ALTER TABLE schema_a.table_a RENAME TO table_b;`,
			tree: `SQLStatement {AlterTable {TT SchemaName T TableName RenameTo {TT TableName}} T}`,
		}, {
			code: `ALTER TABLE table_a RENAME column_a TO column_b;`,
			tree: `SQLStatement {AlterTable {TT TableName RenameColumn {T ColumnName T ColumnName}} T}`,
		}, {
			code: `ALTER TABLE /* comment */ table_a RENAME COLUMN column_a TO column_b;`,
			tree: `SQLStatement {AlterTable {TT TableName RenameColumn {TT ColumnName T ColumnName}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD column_b INTEGER INTEGER;`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {T ColumnDefinition {ColumnName TypeName{TT}}}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b INTEGER(10);`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {ColumnName TypeName {TTTT}}}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b INTEGER(10, 20) PRIMARY KEY;`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName TypeName {TTTTTT} ColumnConstraint {TT} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b INTEGER(+10) CONSTRAINT pk PRIMARY KEY AUTOINCREMENT;`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName TypeName {T TTTT} ColumnConstraint {T ConstraintName TTT} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b PRIMARY KEY ASC ON CONFLICT ROLLBACK;`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TTT ConflictClause {TTT}} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b PRIMARY KEY DESC ON CONFLICT ABORT;`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TTT ConflictClause {TTT}} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b NOT NULL ON CONFLICT FAIL;`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT ConflictClause {TTT}} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b UNIQUE ON CONFLICT IGNORE;`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {T ConflictClause {TTT}} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(10);`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{T} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b DEFAULT(10);`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{T} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b DEFAULT 'a';`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b DEFAULT -10;`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TTT} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b DEFAULT TRUE;`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b COLLATE collate_a;`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {T CollationName} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b REFERENCES table_b ON DELETE SET NULL;`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {ForeignKeyClause {T TableName TTTT}} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b REFERENCES table_b (column_c) ON UPDATE SET DEFAULT;`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {ForeignKeyClause {T TableName T CommaList{ColumnName} T TTTT}} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b REFERENCES table_b (column_c, column_d) ON UPDATE SET DEFAULT;`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {ForeignKeyClause {T TableName T CommaList{ColumnName T ColumnName} T TTTT}} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b REFERENCES table_b MATCH name_a ON DELETE CASCADE;`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {ForeignKeyClause {T TableName TT TTT}} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b REFERENCES table_b ON DELETE RESTRICT DEFERRABLE;`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {ForeignKeyClause {T TableName TTT T}} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b REFERENCES table_b ON DELETE NO ACTION NOT DEFERRABLE INITIALLY DEFERRED;`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {ForeignKeyClause {T TableName TTTT TTTT}} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b REFERENCES table_b ON DELETE NO ACTION NOT DEFERRABLE INITIALLY IMMEDIATE;`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {ForeignKeyClause {T TableName TTTT TTTT}} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b AS (10);`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {T T Expression{T} T}}}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b GENERATED ALWAYS AS (10) STORED;`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TTT T Expression{T} T T}}}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b NOT NULL AS (10) VIRTUAL;`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT} ColumnConstraint{T T Expression{T} T T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a DROP column_b;`,
			tree: `SQLStatement {AlterTable {TT TableName DropColumn {T ColumnName}} T}`,
		}, {
			code: `ALTER TABLE table_a DROP COLUMN column_b;`,
			tree: `SQLStatement {AlterTable {TT TableName DropColumn {TT ColumnName}} T}`,
		},
	}

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

func TestParserAlterTableError(t *testing.T) {
	t.Parallel()
	cases := []struct {
		code string
		tree string
	}{
		{
			code: `ALTER TABLE RENAME TO table_b;`,
			tree: `SQLStatement {AlterTable {TT !ErrorMissing RenameTo {TT TableName}} T}`,
		}, {
			code: `ALTER TABLE schema_a. RENAME TO table_b;`,
			tree: `SQLStatement {AlterTable {TT SchemaName T !ErrorMissing RenameTo {TT TableName}} T}`,
		}, {
			code: `ALTER TABLE table_a RENAME TO ;`,
			tree: `SQLStatement {AlterTable {TT TableName RenameTo {TT !ErrorMissing}} T}`,
		}, {
			code: `ALTER TABLE table_a RENAME column_a TO ;`,
			tree: `SQLStatement {AlterTable {TT TableName RenameColumn {T ColumnName T !ErrorMissing}} T}`,
		}, {
			code: `ALTER TABLE table_a 10 RENAME column_a TO ;`,
			tree: `SQLStatement {AlterTable {TT TableName Skipped {T} RenameColumn {T ColumnName T !ErrorMissing}} T}`,
		}, {
			code: `ALTER `,
			tree: `SQLStatement {AlterTable {T !ErrorUnexpectedEOF} T}`,
		}, {
			code: `ALTER TABLE table_a RENAME column_a column_b ;`,
			tree: `SQLStatement {AlterTable {TT TableName RenameColumn {T ColumnName !ErrorMissing ColumnName}} T}`,
		}, {
			code: `ALTER TABLE table_a RENAME COLUMN TO column_b ;`,
			tree: `SQLStatement {AlterTable {TT TableName RenameColumn {TT !ErrorMissing T ColumnName}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN ;`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT !ErrorMissing}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_a INTEGER();`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {ColumnName TypeName {T T !ErrorMissing T}}}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_a INTEGER 10);`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {ColumnName TypeName {T !ErrorMissing TT}}}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_a INTEGER (10 20);`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName TypeName {T TT !ErrorMissing TT}}}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_a INTEGER (10, );`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName TypeName {T TTT !ErrorMissing T}}}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN CONSTRAINT;`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT !ErrorMissing}} Skipped{T} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_a CONSTRAINT PRIMARY KEY;`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {T !ErrorMissing TT}}}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_a PRIMARY;`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {
				TT ColumnDefinition{ColumnName ColumnConstraint {T !ErrorMissing}}}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_a NOT;`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {
				TT ColumnDefinition{ColumnName ColumnConstraint {T !ErrorMissing}}}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_a NOT NULL ON CONFLICT;`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {
				TT ColumnDefinition{ColumnName ColumnConstraint {TT ConflictClause{TT !ErrorExpecting}}}}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b NOT NULL ON  FAIL;`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT ConflictClause {T !ErrorMissing T}} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_a UNIQUE ON;`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {
				TT ColumnDefinition{ColumnName ColumnConstraint {T ConflictClause{T !ErrorExpecting}}}}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_a CHECK 10);`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {
				TT ColumnDefinition{ColumnName ColumnConstraint {T !ErrorMissing Expression{T} T}}}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_a CHECK (10;`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {
				TT ColumnDefinition{ColumnName ColumnConstraint {TT Expression{T} !ErrorMissing }}}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_a CHECK();`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {
				TT ColumnDefinition{ColumnName ColumnConstraint {TT !ErrorMissing T}}}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_a DEFAULT;`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {
				TT ColumnDefinition{ColumnName ColumnConstraint {T !ErrorExpecting}}}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_a DEFAULT ();`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {
				TT ColumnDefinition{ColumnName ColumnConstraint {T T !ErrorMissing T}}}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_a DEFAULT (10;`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {
				TT ColumnDefinition{ColumnName ColumnConstraint {T T Expression{T} !ErrorMissing }}}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_a DEFAULT +;`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {
				TT ColumnDefinition{ColumnName ColumnConstraint {T T !ErrorMissing}}}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_a COLLATE;`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {
				TT ColumnDefinition{ColumnName ColumnConstraint {T !ErrorMissing}}}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_a REFERENCES;`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {
				TT ColumnDefinition{ColumnName ColumnConstraint {ForeignKeyClause{T !ErrorMissing}}}}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_a REFERENCES table_b ();`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {
				TT ColumnDefinition{ColumnName ColumnConstraint {ForeignKeyClause{T TableName T CommaList{!ErrorMissing} T}}}}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_a REFERENCES table_b (column_a) ON;`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {
				TT ColumnDefinition{ColumnName ColumnConstraint {
					ForeignKeyClause{T TableName T CommaList{ColumnName} T T !ErrorExpecting}}}}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b REFERENCES table_b (column_c column_d) ON UPDATE SET DEFAULT;`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {ForeignKeyClause {T TableName T CommaList{ColumnName !ErrorMissing ColumnName} T TTTT}} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b REFERENCES table_b (column_c,) ON UPDATE SET DEFAULT;`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {ForeignKeyClause {T TableName T CommaList{ColumnName T !ErrorMissing} T TTTT}} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b REFERENCES table_b (column_c 10) ON UPDATE SET DEFAULT;`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {ForeignKeyClause {T TableName T CommaList{ColumnName Skipped{T}} T TTTT}} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b REFERENCES table_b (column_c 10, 10) ON UPDATE SET DEFAULT;`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {ForeignKeyClause {T TableName T CommaList{ColumnName Skipped{T} T Skipped{T}} T TTTT}} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b REFERENCES table_b (column_c, column_d, column_e;`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {ForeignKeyClause {T TableName T CommaList{
					ColumnName T ColumnName T ColumnName} !ErrorMissing}} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b REFERENCES table_b (10 column_c) ON UPDATE SET DEFAULT;`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {ForeignKeyClause {T TableName T CommaList{Skipped{T} ColumnName} T TTTT}} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_a REFERENCES table_b (column_a) ON DELETE SET;`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {
				TT ColumnDefinition{ColumnName ColumnConstraint {
					ForeignKeyClause{T TableName T CommaList{ColumnName} T TTT !ErrorExpecting }}}}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_a REFERENCES table_b (column_a) ON DELETE NO;`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {
				TT ColumnDefinition{ColumnName ColumnConstraint {
					ForeignKeyClause{T TableName T CommaList{ColumnName} T TTT !ErrorExpecting }}}}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_a REFERENCES table_b (column_a) ON NO ACTION;`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {
				TT ColumnDefinition{ColumnName ColumnConstraint {
					ForeignKeyClause{T TableName T CommaList{ColumnName} T T !ErrorExpecting TT }}}}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_a REFERENCES table_b MATCH;`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {
				TT ColumnDefinition{ColumnName ColumnConstraint {
					ForeignKeyClause{T TableName T !ErrorExpecting}}}}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_a REFERENCES table_b DEFERRABLE INITIALLY;`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {
				TT ColumnDefinition{ColumnName ColumnConstraint {
					ForeignKeyClause{T TableName TT !ErrorExpecting}}}}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_a AS ();`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {
				TT ColumnDefinition{ColumnName ColumnConstraint {T T !ErrorMissing T}}}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_a GENERATED ALWAYS AS 10);`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {
				TT ColumnDefinition{ColumnName ColumnConstraint {TTT !ErrorMissing Expression{T} T}}}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_a GENERATED AS (10);`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {
				TT ColumnDefinition{ColumnName ColumnConstraint {T !ErrorMissing TT Expression{T} T}}}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_a GENERATED ALWAYS (10);`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {
				TT ColumnDefinition{ColumnName ColumnConstraint {TT !ErrorMissing T Expression{T} T}}}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_a GENERATED ALWAYS AS (10;`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {
				TT ColumnDefinition{ColumnName ColumnConstraint {TTT T Expression{T} !ErrorMissing}}}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_a CONSTRAINT constraint_a;`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {
				TT ColumnDefinition{ColumnName ColumnConstraint {T ConstraintName !ErrorExpecting}}}} T}`,
		}, {
			code: `ALTER TABLE table_a DROP COLUMN;`,
			tree: `SQLStatement {AlterTable {TT TableName DropColumn {TT !ErrorMissing}} T}`,
		}, {
			code: `EXPLAIN QUERY ALTER TABLE table_a ADD COLUMN column_b NOT NULL AS (10) VIRTUAL;`,
			tree: `SQLStatement {ExplainQueryPlan {TT !ErrorMissing AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT} ColumnConstraint{T T Expression{T} T T} }}}} T}`,
		},
	}

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

func TestParserExpression(t *testing.T) {
	t.Parallel()
	cases := []struct {
		code string
		tree string
	}{
		{
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(TRUE OR FALSE OR 10);`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{Or{Or{TTT} TT}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(TRUE AND FALSE AND 10);`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{And{And{TTT} TT}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(NOT NOT (TRUE = FALSE OR 10 == 20));`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT
					Expression{Not {T Not{T ParenExpression{T CommaList{Expression{Or{Equal{TTT} T Equal{TTT}}}} T}}}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(TRUE <> FALSE != 10);`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT
					Expression{NotEqual{NotEqual{TTT} TT}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK((TRUE > FALSE < 10) AND (10 >= 20 <= 10));`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT
					Expression{And{
						ParenExpression{T CommaList{Expression{LessThan{GreaterThan{TTT} TT}}} T} T
						ParenExpression{T CommaList{Expression{LessThanOrEqual{GreaterThanOrEqual{TTT} TT}}} T}}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(10 IS 10);`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{Is{TTT}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(10 IS DISTINCT FROM 10);`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{IsDistinctFrom{TTTTT}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(10 IS NOT 10);`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{IsNot{TTTT}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(10 IS NOT DISTINCT FROM 10);`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{IsNotDistinctFrom{TTTTTT}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(10 & 20 | 30 << 40 >> 50);`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{RightShift{LeftShift{BitOr{BitAnd{TTT} TT} TT} TT}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(10 + 2 - 5);`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{Subtract{Add{TTT} TT}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(10 * 2 / 5 % 2);`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{Mod{Divide{Multiply{TTT} TT} TT}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK('a' || 'b' -> 'c' ->> 'd');`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{Extract2{Extract1{Concatenate{TTT} TT} TT}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(~10 + +20 * -30);`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{Add{BitNot{TT} T Multiply{PrefixPlus{TT} T Negate{TT}}}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(10 BETWEEN 5 AND 15);`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{Between{TTTTT}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(10 NOT BETWEEN 5 AND 15);`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{NotBetween{TTTTTT}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(10 REGEXP '10');`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{Regexp{TTT}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(10 LIKE '10' ESCAPE '!');`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{Like{TTTTT}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(10 NOT GLOB '10');`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{NotGlob{TTTT}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(10 NOT MATCH '10');`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{NotMatch{TTTT}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(10);`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{T} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK('string');`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{T} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(?);`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{BindParameter} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK($a);`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{BindParameter} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(column_a);`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{ColumnReference{ColumnName}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(table_a.column_a);`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{ColumnReference{TableName T ColumnName}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(schema_a.table_a.column_a);`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{ColumnReference{SchemaName T TableName T ColumnName}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(func());`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{FunctionCall{
					FunctionName TT}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(func('a'));`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{FunctionCall{
					FunctionName T FunctionArguments{CommaList{Expression{T}}} T}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(func(DISTINCT 'a', 10 ORDER BY a, b) FILTER (WHERE a + b));`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{FunctionCall{
					FunctionName T FunctionArguments{T CommaList{Expression{T} T Expression{T}}
						OrderBy {TT CommaList {
							OrderingTerm{Expression{ColumnReference{ColumnName}}} T
							OrderingTerm{Expression{ColumnReference{ColumnName}}}
						}}} T
						FilterClause{TT T Expression{Add{ColumnReference{ColumnName} T ColumnReference{ColumnName}}}
							T}}} T} }}} T}`,
		}, {
			code: `SELECT function(10 ORDER BY a ASC NULLS LAST);`,
			tree: `SQLStatement {Select{T Expression{FunctionCall{
				FunctionName T FunctionArguments{CommaList{Expression{T}}
					OrderBy{TT CommaList{
						OrderingTerm{Expression{ColumnReference{ColumnName}} TTT}} }} T}}} T}`,
		}, {
			code: `SELECT function(10 ORDER BY a DESC NULLS FIRST);`,
			tree: `SQLStatement {Select{T Expression{FunctionCall{
				FunctionName T FunctionArguments{CommaList{Expression{T}}
					OrderBy{TT CommaList{
						OrderingTerm{Expression{ColumnReference{ColumnName}} TTT}} }} T}}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(func(*) OVER window_a);`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{FunctionCall{FunctionName T FunctionArguments{T} T
					OverClause{T WindowName}}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(func(*) OVER ());`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{FunctionCall{FunctionName T FunctionArguments{T} T
					OverClause{TTT}}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(func(*) OVER (window_a PARTITION BY a, b));`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{FunctionCall{FunctionName T FunctionArguments{T} T
					OverClause{TT WindowName
						PartitionBy{TT CommaList{Expression{ColumnReference{ColumnName}} T
							Expression{ColumnReference{ColumnName}}}} T}}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(func(*) OVER (ORDER BY a));`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{FunctionCall{FunctionName T FunctionArguments{T} T
					OverClause{TT OrderBy{TT CommaList{OrderingTerm{Expression{ColumnReference{ColumnName}}}}} T}}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(func(*) OVER (RANGE BETWEEN UNBOUNDED PRECEDING AND 10 PRECEDING));`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{FunctionCall{FunctionName T FunctionArguments{T} T
					OverClause{TT FrameSpec{T FrameSpecBetween{TTTT Expression{T} T}} T} }} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(func(*) OVER (ROWS BETWEEN 10 PRECEDING AND CURRENT ROW));`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{FunctionCall{FunctionName T FunctionArguments{T} T
					OverClause{TT FrameSpec{T FrameSpecBetween{T Expression{T} TTTT}} T} }} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(func(*) OVER (GROUPS BETWEEN CURRENT ROW AND 10 FOLLOWING));`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{FunctionCall{FunctionName T FunctionArguments{T} T
					OverClause{TT FrameSpec{T FrameSpecBetween{TTTT Expression{T} T}} T} }} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(func(*) OVER (RANGE BETWEEN 10 FOLLOWING AND UNBOUNDED FOLLOWING));`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{FunctionCall{FunctionName T FunctionArguments{T} T
					OverClause{TT FrameSpec{T FrameSpecBetween{T Expression{T} TTTT}} T} }} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(func(*) OVER (ROWS UNBOUNDED PRECEDING EXCLUDE NO OTHERS));`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{FunctionCall{FunctionName T FunctionArguments{T} T
					OverClause{TT FrameSpec{TTTTTT} T} }} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(func(*) OVER (GROUPS 10 PRECEDING EXCLUDE CURRENT ROW));`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{FunctionCall{FunctionName T FunctionArguments{T} T
					OverClause{TT FrameSpec{T Expression{T} TTTT} T} }} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(func(*) OVER (RANGE CURRENT ROW EXCLUDE GROUP));`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{FunctionCall{FunctionName T FunctionArguments{T} T
					OverClause{TT FrameSpec{TTTTT} T} }} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(func(*) OVER (ROWS BETWEEN 10 PRECEDING AND 10 FOLLOWING EXCLUDE TIES));`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{FunctionCall{FunctionName T FunctionArguments{T} T
					OverClause{TT FrameSpec{T FrameSpecBetween{T Expression{T} TT Expression{T} T} TT} T} }} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK((10 + column_a));`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{
					ParenExpression{T CommaList{Expression{Add{T T ColumnReference{ColumnName}}}} T}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(CAST ('10' AS INTEGER));`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{Cast{TT Expression{T} T TypeName{T} T}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK('10' COLLATE collate_a);`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{Collate{TT CollationName}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK('10' LIKE '20');`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{Like{TTT}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK('10' NOT LIKE '20' ESCAPE '!');`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{NotLike{TTTTTT}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK('10' GLOB '20');`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{Glob{TTT}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK('10' NOT REGEXP '20');`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{NotRegexp{TTTT}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK('10' MATCH '20');`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{Match{TTT}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(10 ISNULL);`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{IsNull{TT}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(10 NOTNULL);`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{Notnull{TT}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(10 NOT NULL);`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{NotNull{TTT}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(10 IS DISTINCT FROM 20);`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{IsDistinctFrom{TTTTT}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(10 IS NOT 20);`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{IsNot{TTTT}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(10 IN (10, 20));`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{In{TTT CommaList{TTT} T}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(10 IN ());`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{In{TTTT}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(10 IN (SELECT));`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{In{TTT Select{T} T}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(10 IN schema_name.table_name);`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{In{TT SchemaName T TableName}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(10 IN schema_a.table_function_a(10, 20));`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{In{TT SchemaName T TableFunctionName T CommaList{TTT} T }} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(10 NOT IN (10, 20));`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{NotIn{TTTT CommaList{TTT} T}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(10 NOT IN ());`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{NotIn{TTTTT}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(10 NOT IN schema_name.table_name);`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{NotIn{TTT SchemaName T TableName}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(10 NOT IN schema_a.table_function_a(10, 20));`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{NotIn{TTT SchemaName T TableFunctionName T CommaList{TTT} T }} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(10 NOT IN table_a);`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{NotIn{TTT TableName}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(10 NOT IN (SELECT));`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{NotIn{TTTT Select{T} T}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(EXISTS(SELECT));`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{Exists{TT Select{T} T}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(NOT EXISTS(SELECT));`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{Not {T Exists{TT Select{T} T}}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(CASE 10 WHEN 10 THEN TRUE ELSE FALSE END);`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{
					Case{T Expression{T} When {T Expression{T} T Expression{T}} Else{T Expression{T}} T}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(CASE WHEN 10 THEN 'a' WHEN 20 THEN 'b' END);`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{
					Case{T When{T Expression{T} T Expression{T}} When{T Expression{T} T Expression{T}} T}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(RAISE (IGNORE));`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{Raise{TTTT}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(RAISE (ROLLBACK, 'error message'));`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{Raise{TTTT ErrorMessage{Expression{T}} T}} T} }}} T}`,
		},
	}

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
				b := new(strings.Builder)
				writeErrors(b, parsed)
				fmt.Print(b.String())
				t.Fail()
			}
		})
	}
}

func TestParserExpressionError(t *testing.T) {
	t.Parallel()
	cases := []struct {
		code string
		tree string
	}{
		{
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(TRUE OR);`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{Or{TT !ErrorMissing}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(TRUE AND);`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{And{TT !ErrorMissing}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(NOT);`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{Not{T !ErrorMissing}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(10 ==);`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{Equal{TT !ErrorMissing}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(10 <>);`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{NotEqual{TT !ErrorMissing}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(10 GLOB);`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{Glob{TT !ErrorMissing}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(10 REGEXP);`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{Regexp{TT !ErrorMissing}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(10 MATCH);`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{Match{TT !ErrorMissing}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(10 LIKE ESCAPE);`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{Like{TT !ErrorMissing T !ErrorMissing}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(10 NOT GLOB);`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{NotGlob{TTT !ErrorMissing}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(10 NOT REGEXP);`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{NotRegexp{TTT !ErrorMissing}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(10 NOT MATCH);`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{NotMatch{TTT !ErrorMissing}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(10 NOT LIKE ESCAPE);`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{NotLike{TTT !ErrorMissing T !ErrorMissing}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(10 IS NOT DISTINCT 10);`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{IsNotDistinctFrom{TTTT !ErrorMissing T}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(10 IS NOT DISTINCT FROM);`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{IsNotDistinctFrom{TTTTT !ErrorMissing}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(10 IS NOT);`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{IsNot{TTT !ErrorExpecting}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(10 IS DISTINCT FROM);`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{IsDistinctFrom{TTTT !ErrorMissing}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(10 IS DISTINCT 10);`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{IsDistinctFrom{TTT !ErrorMissing T}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(10 IS);`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{Is{TT !ErrorExpecting}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(10 BETWEEN);`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{Between{TT !ErrorMissing}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(10 BETWEEN AND 20);`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{Between{TT !ErrorMissing TT}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(10 BETWEEN 10  20);`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{Between{TTT !ErrorMissing T}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(10 NOT BETWEEN);`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{NotBetween{TTT !ErrorMissing}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(10 NOT BETWEEN AND 20);`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{NotBetween{TTT !ErrorMissing TT}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(10 NOT BETWEEN 10  20);`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{NotBetween{TTTT !ErrorMissing T}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(10 IN (ALTER));`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{In{TTT !ErrorExpecting Skipped{T} T}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(10 IN (10;`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{In{TTT CommaList{T} !ErrorMissing}} !ErrorMissing} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(10 IN function());`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{In{TT TableFunctionName T !ErrorMissing T}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(10 IN function(;`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{In{TT TableFunctionName T !ErrorMissing}}
					!ErrorMissing} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(10 IN ALTER);`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{In{TT !ErrorExpecting}} !ErrorMissing} }}} Skipped{TT} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(10 NOT IN (ALTER));`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{NotIn{TTTT !ErrorExpecting Skipped{T} T}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(10 NOT IN (10;`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{NotIn{TTTT CommaList{T} !ErrorMissing}} !ErrorMissing} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(10 NOT IN function());`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{NotIn{TTT TableFunctionName T !ErrorMissing T}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(10 NOT IN function(;`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{NotIn{TTT TableFunctionName T !ErrorMissing}}
					!ErrorMissing} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(10 NOT IN ALTER);`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{NotIn{TTT !ErrorExpecting}} !ErrorMissing} }}} Skipped{TT} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(10 <);`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{LessThan{TT !ErrorMissing}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(10 &);`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{BitAnd{TT !ErrorMissing}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(10 +);`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{Add{TT !ErrorMissing}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(10 *);`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{Multiply{TT !ErrorMissing}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(10 ||);`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{Concatenate{TT !ErrorMissing}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(10 COLLATE);`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{Collate{TT !ErrorMissing}} T} }}} T}`,
		}, {
			code: `ALTER TABLE table_a ADD COLUMN column_b CHECK(~);`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT Expression{BitNot{T !ErrorMissing}} T} }}} T}`,
		}, {
			code: `SELECT function(;`,
			tree: `SQLStatement {Select{T Expression{FunctionCall{FunctionName T !ErrorMissing}}} T}`,
		}, {
			code: `SELECT function(DISTINCT );`,
			tree: `SQLStatement {Select{T Expression{FunctionCall{FunctionName T FunctionArguments{T !ErrorMissing} T}}} T}`,
		}, {
			code: `SELECT function(10 ORDER a);`,
			tree: `SQLStatement {Select{T Expression{FunctionCall{
				FunctionName T FunctionArguments{CommaList{Expression{T}}
					OrderBy{T !ErrorMissing CommaList{OrderingTerm{Expression{ColumnReference{ColumnName}}}}}} T}}} T}`,
		}, {
			code: `SELECT function(10 ORDER BY);`,
			tree: `SQLStatement {Select{T Expression{FunctionCall{
				FunctionName T FunctionArguments{CommaList{Expression{T}}
					OrderBy{TT !ErrorMissing}} T}}} T}`,
		}, {
			code: `SELECT function(10 ORDER BY a COLLATE);`,
			tree: `SQLStatement {Select{T Expression{FunctionCall{
				FunctionName T FunctionArguments{CommaList{Expression{T}}
					OrderBy{TT CommaList{
						OrderingTerm{Expression{Collate{ColumnReference{ColumnName} T !ErrorMissing}}}} }} T}}} T}`,
		}, {
			code: `SELECT function(10 ORDER BY a NULLS);`,
			tree: `SQLStatement {Select{T Expression{FunctionCall{
				FunctionName T FunctionArguments{CommaList{Expression{T}}
					OrderBy{TT CommaList{
						OrderingTerm{Expression{ColumnReference{ColumnName}} T !ErrorExpecting}} }} T}}} T}`,
		}, {
			code: `SELECT function() FILTER WHERE a);`,
			tree: `SQLStatement {Select{T Expression{FunctionCall{
				FunctionName TT FilterClause{T !ErrorMissing T Expression{ColumnReference{ColumnName}} T}}}} T}`,
		}, {
			code: `SELECT function() FILTER (a);`,
			tree: `SQLStatement {Select{T Expression{FunctionCall{
				FunctionName TT FilterClause{TT !ErrorMissing Expression{ColumnReference{ColumnName}} T}}}} T}`,
		}, {
			code: `SELECT function() FILTER (WHERE );`,
			tree: `SQLStatement {Select{T Expression{FunctionCall{
				FunctionName TT FilterClause{TTT !ErrorMissing T}}}} T}`,
		}, {
			code: `SELECT function() FILTER (WHERE 10;`,
			tree: `SQLStatement {Select{T Expression{FunctionCall{
				FunctionName TT FilterClause{TTT Expression{T} !ErrorMissing}}}} T}`,
		}, {
			code: `SELECT function() OVER;`,
			tree: `SQLStatement {Select{T Expression{
				FunctionCall{FunctionName TT OverClause{T !ErrorExpecting}}}} T}`,
		}, {
			code: `SELECT function() OVER (PARTITION a);`,
			tree: `SQLStatement {Select{T Expression{
				FunctionCall{FunctionName TT OverClause{TT
					PartitionBy{T !ErrorMissing CommaList{Expression{ColumnReference{ColumnName}}}} T}}}} T}`,
		}, {
			code: `SELECT function() OVER (PARTITION BY);`,
			tree: `SQLStatement {Select{T Expression{
				FunctionCall{FunctionName TT OverClause{TT PartitionBy{TT !ErrorMissing} T}}}} T}`,
		}, {
			code: `SELECT function() OVER (;`,
			tree: `SQLStatement {Select{T Expression{
				FunctionCall{FunctionName TT OverClause{TT !ErrorMissing}}}} T}`,
		}, {
			code: `SELECT function() OVER (RANGE);`,
			tree: `SQLStatement {Select{T Expression{
				FunctionCall{FunctionName TT OverClause{TT FrameSpec{T !ErrorExpecting} T}}}} T}`,
		}, {
			code: `SELECT function() OVER (RANGE BETWEEN);`,
			tree: `SQLStatement {Select{T Expression{
				FunctionCall{FunctionName TT OverClause{TT FrameSpec{T FrameSpecBetween{T !ErrorExpecting}} T}}}} T}`,
		}, {
			code: `SELECT function() OVER (RANGE BETWEEN PRECEDING);`,
			tree: `SQLStatement {Select{T Expression{
				FunctionCall{FunctionName TT OverClause{TT FrameSpec{T
					FrameSpecBetween{T !ErrorExpecting T !ErrorExpecting}} T}}}} T}`,
		}, {
			code: `SELECT function() OVER (RANGE BETWEEN ROW);`,
			tree: `SQLStatement {Select{T Expression{
				FunctionCall{FunctionName TT OverClause{TT FrameSpec{T
					FrameSpecBetween{T !ErrorMissing T !ErrorExpecting}} T}}}} T}`,
		}, {
			code: `SELECT function() OVER (RANGE BETWEEN FOLLOWING);`,
			tree: `SQLStatement {Select{T Expression{
				FunctionCall{FunctionName TT OverClause{TT FrameSpec{T
					FrameSpecBetween{T !ErrorMissing T !ErrorExpecting}} T}}}} T}`,
		}, {
			code: `SELECT function() OVER (RANGE BETWEEN AND);`,
			tree: `SQLStatement {Select{T Expression{
				FunctionCall{FunctionName TT OverClause{TT FrameSpec{T
					FrameSpecBetween{T !ErrorExpecting T !ErrorExpecting}} T}}}} T}`,
		}, {
			code: `SELECT function() OVER (RANGE BETWEEN UNBOUNDED);`,
			tree: `SQLStatement {Select{T Expression{
				FunctionCall{FunctionName TT OverClause{TT FrameSpec{T
					FrameSpecBetween{TT !ErrorMissing}} T}}}} T}`,
		}, {
			code: `SELECT function() OVER (RANGE BETWEEN 10);`,
			tree: `SQLStatement {Select{T Expression{
				FunctionCall{FunctionName TT OverClause{TT FrameSpec{T
					FrameSpecBetween{T Expression{T} !ErrorExpecting}} T}}}} T}`,
		}, {
			code: `SELECT function() OVER (RANGE BETWEEN CURRENT);`,
			tree: `SQLStatement {Select{T Expression{
				FunctionCall{FunctionName TT OverClause{TT FrameSpec{T
					FrameSpecBetween{TT !ErrorMissing}} T}}}} T}`,
		}, {
			code: `SELECT function() OVER (RANGE BETWEEN CURRENT ROW CURRENT);`,
			tree: `SQLStatement {Select{T Expression{
				FunctionCall{FunctionName TT OverClause{TT FrameSpec{T
					FrameSpecBetween{TTT !ErrorMissing T !ErrorMissing}} T}}}} T}`,
		}, {
			code: `SELECT function() OVER (RANGE BETWEEN CURRENT ROW UNBOUNDED);`,
			tree: `SQLStatement {Select{T Expression{
				FunctionCall{FunctionName TT OverClause{TT FrameSpec{T
					FrameSpecBetween{TTT !ErrorMissing T !ErrorMissing}} T}}}} T}`,
		}, {
			code: `SELECT function() OVER (RANGE BETWEEN CURRENT ROW 10);`,
			tree: `SQLStatement {Select{T Expression{
				FunctionCall{FunctionName TT OverClause{TT FrameSpec{T
					FrameSpecBetween{TTT !ErrorMissing Expression{T} !ErrorExpecting}} T}}}} T}`,
		}, {
			code: `SELECT function() OVER (RANGE BETWEEN UNBOUNDED AND);`,
			tree: `SQLStatement {Select{T Expression{
				FunctionCall{FunctionName TT OverClause{TT FrameSpec{T
					FrameSpecBetween{TT !ErrorMissing T !ErrorExpecting}} T}}}} T}`,
		}, {
			code: `SELECT function() OVER (RANGE BETWEEN CURRENT AND);`,
			tree: `SQLStatement {Select{T Expression{
				FunctionCall{FunctionName TT OverClause{TT FrameSpec{T
					FrameSpecBetween{TT !ErrorMissing T !ErrorExpecting}} T}}}} T}`,
		}, {
			code: `SELECT function() OVER (RANGE BETWEEN 10 AND);`,
			tree: `SQLStatement {Select{T Expression{
				FunctionCall{FunctionName TT OverClause{TT FrameSpec{T
					FrameSpecBetween{T Expression{T} !ErrorExpecting T !ErrorExpecting}} T}}}} T}`,
		}, {
			code: `SELECT function() OVER (RANGE UNBOUNDED);`,
			tree: `SQLStatement {Select{T Expression{
				FunctionCall{FunctionName TT OverClause{TT FrameSpec{TT !ErrorMissing} T}}}} T}`,
		}, {
			code: `SELECT function() OVER (RANGE CURRENT);`,
			tree: `SQLStatement {Select{T Expression{
				FunctionCall{FunctionName TT OverClause{TT FrameSpec{TT !ErrorMissing} T}}}} T}`,
		}, {
			code: `SELECT function() OVER (RANGE 10);`,
			tree: `SQLStatement {Select{T Expression{
				FunctionCall{FunctionName TT OverClause{TT FrameSpec{T Expression{T} !ErrorMissing} T}}}} T}`,
		}, {
			code: `SELECT function() OVER (RANGE CURRENT ROW EXCLUDE);`,
			tree: `SQLStatement {Select{T Expression{
				FunctionCall{FunctionName TT OverClause{TT FrameSpec{TTTT !ErrorExpecting} T}}}} T}`,
		}, {
			code: `SELECT function() OVER (RANGE CURRENT ROW EXCLUDE NO);`,
			tree: `SQLStatement {Select{T Expression{
				FunctionCall{FunctionName TT OverClause{TT FrameSpec{TTTTT !ErrorMissing} T}}}} T}`,
		}, {
			code: `SELECT function() OVER (RANGE CURRENT ROW EXCLUDE CURRENT);`,
			tree: `SQLStatement {Select{T Expression{
				FunctionCall{FunctionName TT OverClause{TT FrameSpec{TTTTT !ErrorMissing} T}}}} T}`,
		}, {
			code: `SELECT ();`,
			tree: `SQLStatement {Select{T Expression{ParenExpression{T !ErrorMissing T}}} T}`,
		}, {
			code: `SELECT (, 10);`,
			tree: `SQLStatement {Select{T Expression{ParenExpression{T CommaList{!ErrorMissing T Expression{T}} T}}} T}`,
		}, {
			code: `SELECT (10 10);`,
			tree: `SQLStatement {Select{T Expression{ParenExpression{T
				CommaList{Expression{T} !ErrorMissing Expression{T}} T}}} T}`,
		}, {
			code: `SELECT (10 AS 10);`,
			tree: `SQLStatement {Select{T Expression{ParenExpression{T
				CommaList{Expression{T} Skipped{T} Expression{T}} T}}} T}`,
		}, {
			code: `SELECT (10,,10);`,
			tree: `SQLStatement {Select{T Expression{ParenExpression{T
				CommaList{Expression{T} T !ErrorMissing T Expression{T}} T}}} T}`,
		}, {
			code: `SELECT (10, AS);`,
			tree: `SQLStatement {Select{T Expression{ParenExpression{T
				CommaList{Expression{T} T Skipped{T}} T}}} T}`,
		}, {
			code: `SELECT (10;`,
			tree: `SQLStatement {Select{T Expression{ParenExpression{T CommaList{Expression{T}} !ErrorMissing}}} T}`,
		}, {
			code: `SELECT CAST 10 AS NUMBER);`,
			tree: `SQLStatement {Select{T Expression{Cast{T !ErrorMissing Expression{T} T TypeName{T} T}}} T}`,
		}, {
			code: `SELECT CAST AS NUMBER);`,
			tree: `SQLStatement {Select{T Expression{Cast{T !ErrorMissing !ErrorMissing T TypeName{T} T}}} T}`,
		}, {
			code: `SELECT CAST(10 AS);`,
			tree: `SQLStatement {Select{T Expression{Cast{TT Expression{T} T !ErrorMissing T}}} T}`,
		}, {
			code: `SELECT CAST(10 NUMBER);`,
			tree: `SQLStatement {Select{T Expression{Cast{TT Expression{T} !ErrorMissing TypeName{T} T}}} T}`,
		}, {
			code: `SELECT CAST(10 AS NUMBER;`,
			tree: `SQLStatement {Select{T Expression{Cast{TT Expression{T} T TypeName{T} !ErrorMissing}}} T}`,
		}, {
			code: `SELECT EXISTS SELECT 10);`,
			tree: `SQLStatement {Select{T Expression{Exists{T !ErrorMissing Select{T Expression{T}} T}}} T}`,
		}, {
			code: `SELECT EXISTS(10);`,
			tree: `SQLStatement {Select{T Expression{Exists{TT Skipped{T} T}}} T}`,
		}, {
			code: `SELECT EXISTS(10;`,
			tree: `SQLStatement {Select{T Expression{Exists{TT Skipped{T}}}} T}`,
		}, {
			code: `SELECT EXISTS (SELECT 10;`,
			tree: `SQLStatement {Select{T Expression{Exists{TT Select{T Expression{T}} !ErrorMissing}}} T}`,
		}, {
			code: `SELECT EXISTS ();`,
			tree: `SQLStatement {Select{T Expression{Exists{TT !ErrorMissing T}}} T}`,
		}, {
			code: `SELECT CASE ELSE 10`,
			tree: `SQLStatement {Select{T Expression{Case{T !ErrorMissing Else{T Expression{T}} !ErrorMissing}}} T}`,
		}, {
			code: `SELECT CASE WHEN THEN 10 END`,
			tree: `SQLStatement {Select{T Expression{Case{T When{T !ErrorMissing T Expression{T}} T}}} T}`,
		}, {
			code: `SELECT CASE WHEN 10 10 END`,
			tree: `SQLStatement {Select{T Expression{Case{T When{T Expression{T} !ErrorMissing Expression{T}} T}}} T}`,
		}, {
			code: `SELECT CASE WHEN 10 THEN END`,
			tree: `SQLStatement {Select{T Expression{Case{T When{T Expression{T} T !ErrorMissing} T}}} T}`,
		}, {
			code: `SELECT CASE WHEN 10 THEN 10 ELSE END`,
			tree: `SQLStatement {Select{T Expression{Case{T When{T Expression{T} T Expression{T}} Else{T !ErrorMissing} T}}} T}`,
		}, {
			code: `SELECT RAISE;`,
			tree: `SQLStatement {Select{T Expression{Raise{T !ErrorMissing}}} T}`,
		}, {
			code: `SELECT RAISE IGNORE);`,
			tree: `SQLStatement {Select{T Expression{Raise{T !ErrorMissing TT}}} T}`,
		}, {
			code: `SELECT RAISE(IGNORE;`,
			tree: `SQLStatement {Select{T Expression{Raise{TTT !ErrorMissing}}} T}`,
		}, {
			code: `SELECT RAISE();`,
			tree: `SQLStatement {Select{T Expression{Raise{TT !ErrorExpecting T}}} T}`,
		}, {
			code: `SELECT RAISE(, 'error');`,
			tree: `SQLStatement {Select{T Expression{Raise{TT !ErrorExpecting T ErrorMessage{Expression{T}} T}}} T}`,
		}, {
			code: `SELECT RAISE(ROLLBACK 'error');`,
			tree: `SQLStatement {Select{T Expression{Raise{TTT !ErrorMissing ErrorMessage{Expression{T}} T}}} T}`,
		}, {
			code: `SELECT RAISE(ROLLBACK, );`,
			tree: `SQLStatement {Select{T Expression{Raise{TTTT !ErrorMissing T}}} T}`,
		}, {
			code: `SELECT RAISE(ROLLBACK, 'error';`,
			tree: `SQLStatement {Select{T Expression{Raise{TTTT ErrorMessage{Expression{T}} !ErrorMissing}}} T}`,
		},
	}

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
				fmt.Fprintf(c.tw, "%T ≠ %T\n", parsed, expected)
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
var treeKinds = map[string]parsetree.Kind{
	"Add":                parsetree.KindAdd,
	"AddColumn":          parsetree.KindAddColumn,
	"AlterTable":         parsetree.KindAlterTable,
	"And":                parsetree.KindAnd,
	"Between":            parsetree.KindBetween,
	"BindParameter":      parsetree.KindBindParameter,
	"BitAnd":             parsetree.KindBitAnd,
	"BitNot":             parsetree.KindBitNot,
	"BitOr":              parsetree.KindBitOr,
	"Case":               parsetree.KindCase,
	"Cast":               parsetree.KindCast,
	"Collate":            parsetree.KindCollate,
	"CollationName":      parsetree.KindCollationName,
	"ColumnConstraint":   parsetree.KindColumnConstraint,
	"ColumnDefinition":   parsetree.KindColumnDefinition,
	"ColumnName":         parsetree.KindColumnName,
	"ColumnReference":    parsetree.KindColumnReference,
	"CommaList":          parsetree.KindCommaList,
	"Concatenate":        parsetree.KindConcatenate,
	"ConflictClause":     parsetree.KindConflictClause,
	"ConstraintName":     parsetree.KindConstraintName,
	"Divide":             parsetree.KindDivide,
	"DropColumn":         parsetree.KindDropColumn,
	"Else":               parsetree.KindElse,
	"Equal":              parsetree.KindEqual,
	"ErrorExpecting":     parsetree.KindErrorExpecting,
	"ErrorMessage":       parsetree.KindErrorMessage,
	"ErrorMissing":       parsetree.KindErrorMissing,
	"ErrorUnexpectedEOF": parsetree.KindErrorUnexpectedEOF,
	"Exists":             parsetree.KindExists,
	"Explain":            parsetree.KindExplain,
	"ExplainQueryPlan":   parsetree.KindExplainQueryPlan,
	"Expression":         parsetree.KindExpression,
	"Extract1":           parsetree.KindExtract1,
	"Extract2":           parsetree.KindExtract2,
	"FilterClause":       parsetree.KindFilterClause,
	"ForeignKeyClause":   parsetree.KindForeignKeyClause,
	"FrameSpec":          parsetree.KindFrameSpec,
	"FrameSpecBetween":   parsetree.KindFrameSpecBetween,
	"FunctionArguments":  parsetree.KindFunctionArguments,
	"FunctionCall":       parsetree.KindFunctionCall,
	"FunctionName":       parsetree.KindFunctionName,
	"Glob":               parsetree.KindGlob,
	"GreaterThan":        parsetree.KindGreaterThan,
	"GreaterThanOrEqual": parsetree.KindGreaterThanOrEqual,
	"In":                 parsetree.KindIn,
	"Is":                 parsetree.KindIs,
	"IsDistinctFrom":     parsetree.KindIsDistinctFrom,
	"IsNot":              parsetree.KindIsNot,
	"IsNotDistinctFrom":  parsetree.KindIsNotDistinctFrom,
	"IsNull":             parsetree.KindIsNull,
	"LeftShift":          parsetree.KindLeftShift,
	"LessThan":           parsetree.KindLessThan,
	"LessThanOrEqual":    parsetree.KindLessThanOrEqual,
	"Like":               parsetree.KindLike,
	"Match":              parsetree.KindMatch,
	"Mod":                parsetree.KindMod,
	"Multiply":           parsetree.KindMultiply,
	"Negate":             parsetree.KindNegate,
	"Not":                parsetree.KindNot,
	"NotBetween":         parsetree.KindNotBetween,
	"NotEqual":           parsetree.KindNotEqual,
	"NotGlob":            parsetree.KindNotGlob,
	"NotIn":              parsetree.KindNotIn,
	"NotLike":            parsetree.KindNotLike,
	"NotMatch":           parsetree.KindNotMatch,
	"Notnull":            parsetree.KindNotnull,
	"NotNull":            parsetree.KindNotNull,
	"NotRegexp":          parsetree.KindNotRegexp,
	"Or":                 parsetree.KindOr,
	"OrderBy":            parsetree.KindOrderBy,
	"OrderingTerm":       parsetree.KindOrderingTerm,
	"OverClause":         parsetree.KindOverClause,
	"ParenExpression":    parsetree.KindParenExpression,
	"PartitionBy":        parsetree.KindPartitionBy,
	"PrefixPlus":         parsetree.KindPrefixPlus,
	"Raise":              parsetree.KindRaise,
	"Regexp":             parsetree.KindRegexp,
	"RenameColumn":       parsetree.KindRenameColumn,
	"RenameTo":           parsetree.KindRenameTo,
	"RightShift":         parsetree.KindRightShift,
	"SchemaName":         parsetree.KindSchemaName,
	"Select":             parsetree.KindSelect,
	"Skipped":            parsetree.KindSkipped,
	"SQLStatement":       parsetree.KindSQLStatement,
	"Subtract":           parsetree.KindSubtract,
	"TableFunctionName":  parsetree.KindTableFunctionName,
	"TableName":          parsetree.KindTableName,
	"Token":              parsetree.KindToken,
	"TypeName":           parsetree.KindTypeName,
	"When":               parsetree.KindWhen,
	"WindowName":         parsetree.KindWindowName,
}
