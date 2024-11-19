package parser

import (
	"bytes"
	"fmt"
	"io"
	"iter"
	"regexp"
	"slices"
	"strings"
	"testing"
	"text/tabwriter"
	"unicode"

	"github.com/joaobnv/mel/sqlite/v3_46_1/lexer"
	"github.com/joaobnv/mel/sqlite/v3_46_1/parsetree"
	"github.com/joaobnv/mel/sqlite/v3_46_1/token"
)

func TestParser(t *testing.T) {
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
			code: `ALTER TABLE table_a ADD column_b INTEGER;`,
			tree: `SQLStatement {AlterTable {TT TableName AddColumn {T ColumnDefinition {ColumnName TypeName{T}}}} T}`,
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
		}, {
			code: `EXPLAIN ALTER TABLE table_a ADD COLUMN column_b NOT NULL AS (10) VIRTUAL;`,
			tree: `SQLStatement {Explain {T AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT} ColumnConstraint{T T Expression{T} T T} }}}} T}`,
		}, {
			code: `EXPLAIN QUERY PLAN ALTER TABLE table_a ADD COLUMN column_b NOT NULL AS (10) VIRTUAL;`,
			tree: `SQLStatement {ExplainQueryPlan {TTT AlterTable {TT TableName AddColumn {TT ColumnDefinition {
				ColumnName ColumnConstraint {TT} ColumnConstraint{T T Expression{T} T T} }}}} T}`,
		},
	}

	for _, c := range cases {
		tp := newTestParser(newTestLexer(c.tree))
		expected := tp.tree()

		p := New(lexer.New([]byte(c.code)))
		parsed, comments := p.SQLStatement()

		if str, equals := compare(c.code, comments, parsed, expected); !equals {
			fmt.Println(c.code)
			fmt.Println(str)
			t.Fail()
		}
	}
}

func TestParserError(t *testing.T) {
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

	for _, c := range cases {
		tp := newTestParser(newTestLexer(c.tree))
		expected := tp.tree()

		p := New(lexer.New([]byte(c.code)))
		parsed, comments := p.SQLStatement()

		if str, equals := compare(c.code, comments, parsed, expected); !equals {
			fmt.Println(c.code)
			fmt.Println(str)
			t.Fail()
		}
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
					strings.Repeat("  ", c.indentLevel), parsed.Kind(),
					strings.Repeat("  ", c.indentLevel), expected.Kind(),
				)
				fmt.Fprintf(c.tw, "%T ≠ %T\n", parsed, expected)
				return false
			}

			return c.compareNonTerminals(p, e)
		case parsetree.Terminal:
			e, ok := expected.(parsetree.Terminal)
			if !ok {
				fmt.Fprintf(c.tw, "%s%s\t%s%s\t",
					strings.Repeat("  ", c.indentLevel), parsed.Kind(),
					strings.Repeat("  ", c.indentLevel), expected.Kind(),
				)
				fmt.Fprintf(c.tw, "%T ≠ %T\n", parsed, expected)
				return false
			}

			return c.compareTerminals(p, e)
		case parsetree.Error:
			e, ok := expected.(parsetree.Error)
			if !ok {
				fmt.Fprintf(c.tw, "%s%s\t%s%s\t",
					strings.Repeat("  ", c.indentLevel), parsed.Kind(),
					strings.Repeat("  ", c.indentLevel), expected.Kind(),
				)
				fmt.Fprintf(c.tw, "%T ≠ %T\n", parsed, expected)
				return false
			}

			return c.compareErrors(p, e)
		default:
			panic(fmt.Errorf("unknown type: %T", p))
		}
	} else if parsed != nil && expected == nil {
		fmt.Fprintf(c.tw, "%s%s\t\t\n", strings.Repeat("  ", c.indentLevel), parsed.Kind())

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
		fmt.Fprintf(c.tw, "\t%s%s\t\n", strings.Repeat("  ", c.indentLevel), expected.Kind())

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
		strings.Repeat("  ", c.indentLevel), parsed.Kind(),
		strings.Repeat("  ", c.indentLevel), expected.Kind(),
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
		strings.Repeat("  ", c.indentLevel), parsed.Kind(),
		strings.Repeat("  ", c.indentLevel), expected.Kind(),
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
		strings.Repeat("  ", c.indentLevel), parsed.Kind(),
		strings.Repeat("  ", c.indentLevel), expected.Kind(),
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
		if !unicode.IsLetter(l.rn) {
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
	"AddColumn":          parsetree.KindAddColumn,
	"AlterTable":         parsetree.KindAlterTable,
	"CollationName":      parsetree.KindCollationName,
	"ColumnConstraint":   parsetree.KindColumnConstraint,
	"ColumnDefinition":   parsetree.KindColumnDefinition,
	"ColumnName":         parsetree.KindColumnName,
	"CommaList":          parsetree.KindCommaList,
	"ConflictClause":     parsetree.KindConflictClause,
	"ConstraintName":     parsetree.KindConstraintName,
	"DropColumn":         parsetree.KindDropColumn,
	"ErrorExpecting":     parsetree.KindErrorExpecting,
	"ErrorMissing":       parsetree.KindErrorMissing,
	"ErrorUnexpectedEOF": parsetree.KindErrorUnexpectedEOF,
	"Explain":            parsetree.KindExplain,
	"ExplainQueryPlan":   parsetree.KindExplainQueryPlan,
	"Expression":         parsetree.KindExpression,
	"ForeignKeyClause":   parsetree.KindForeignKeyClause,
	"RenameColumn":       parsetree.KindRenameColumn,
	"RenameTo":           parsetree.KindRenameTo,
	"SchemaName":         parsetree.KindSchemaName,
	"Skipped":            parsetree.KindSkipped,
	"SQLStatement":       parsetree.KindSQLStatement,
	"TableName":          parsetree.KindTableName,
	"Token":              parsetree.KindToken,
	"TypeName":           parsetree.KindTypeName,
}
