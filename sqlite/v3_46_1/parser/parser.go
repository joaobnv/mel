// This package deals with the parsing of the SQL.
package parser

import (
	"errors"
	"slices"
	"strings"

	"github.com/joaobnv/mel/sqlite/v3_46_1/lexer"
	"github.com/joaobnv/mel/sqlite/v3_46_1/parsetree"
	"github.com/joaobnv/mel/sqlite/v3_46_1/token"
)

// Parser is a parser for the SQL.
type Parser struct {
	// comments contains the comments for the current SQLStatement being parsed.
	comments map[*token.Token][]*token.Token
	// tok contains the current 2 look ahead tokens.
	tok [2]*token.Token
	// l is the lexer.
	l *lexer.Lexer
}

// New creates a parser.
func New(l *lexer.Lexer) *Parser {
	return &Parser{
		l: l,
	}
}

// SQLStatement parses a SQLStatement and returns your parse tree and a map containing the comments found.
func (p *Parser) SQLStatement() (c parsetree.Construction, comments map[*token.Token][]*token.Token) {
	if p.tok[0] == nil {
		p.advance()
		p.advance()
	}

	comments = make(map[*token.Token][]*token.Token)
	p.comments = comments

	nt := parsetree.NewNonTerminal(parsetree.KindSQLStatement)

	father := nt
	if p.tok[0].Kind == token.KindExplain {
		if p.tok[1].Kind == token.KindQuery {
			c := parsetree.NewNonTerminal(parsetree.KindExplainQueryPlan)
			c.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
			c.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
			if p.tok[0].Kind == token.KindPlan {
				c.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
				p.advance()
			} else {
				c.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "PLAN"`)))
				p.skipTo(c, token.KindAlter)
			}
			father = c
			nt.AddChild(c)
		} else {
			c := parsetree.NewNonTerminal(parsetree.KindExplain)
			c.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
			father = c
			nt.AddChild(c)
		}
	}

	switch p.tok[0].Kind {
	case token.KindAlter:
		father.AddChild(p.alterTable())
	}

	if p.tok[0].Kind == token.KindSemicolon {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else if p.tok[0].Kind == token.KindEOF {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	} else if p.skipTo(nt, token.KindSemicolon) {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	}

	p.comments = nil
	return nt, comments
}

// alterTable parses a alter table statement.
func (p *Parser) alterTable() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindAlterTable)
	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))

	p.advance()
	if p.tok[0].Kind != token.KindTable {
		if !p.skipTo(nt, token.KindTable, token.KindSemicolon) {
			return nt
		}
	}
	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))

	p.advance()
	if p.tok[0].Kind == token.KindIdentifier && p.tok[1].Kind == token.KindDot {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindSchemaName, p.tok[0]))
		p.advance()
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	}

	if p.tok[0].Kind == token.KindIdentifier {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindTableName, p.tok[0]))
		p.advance()
	} else if slices.Contains([]token.Kind{token.KindRename, token.KindAdd, token.KindDrop}, p.tok[0].Kind) {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New("missing table name")))
	}

	p.skipTo(nt, token.KindRename, token.KindAdd, token.KindDrop, token.KindSemicolon)

	if p.tok[0].Kind == token.KindRename && p.tok[1].Kind == token.KindTo {
		rt := parsetree.NewNonTerminal(parsetree.KindRenameTo)
		rt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
		rt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
		if p.tok[0].Kind == token.KindIdentifier {
			rt.AddChild(parsetree.NewTerminal(parsetree.KindTableName, p.tok[0]))
			p.advance()
		} else {
			rt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing table name`)))
			p.skipTo(rt, token.KindSemicolon)
		}
		nt.AddChild(rt)
	} else if p.tok[0].Kind == token.KindRename {
		rc := parsetree.NewNonTerminal(parsetree.KindRenameColumn)
		rc.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		if p.tok[0].Kind == token.KindColumn {
			rc.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		}

		if p.tok[0].Kind == token.KindIdentifier {
			rc.AddChild(parsetree.NewTerminal(parsetree.KindColumnName, p.tok[0]))
			p.advance()
		} else if p.tok[0].Kind == token.KindTo {
			rc.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New("missing column name")))
		}

		p.skipTo(rc, token.KindTo, token.KindIdentifier, token.KindSemicolon)

		if p.tok[0].Kind == token.KindTo {
			rc.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		} else {
			rc.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "TO"`)))
		}

		if p.tok[0].Kind == token.KindIdentifier {
			rc.AddChild(parsetree.NewTerminal(parsetree.KindColumnName, p.tok[0]))
			p.advance()
		} else if p.tok[0].Kind == token.KindSemicolon {
			rc.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing new column name`)))
		}

		nt.AddChild(rc)
	} else if p.tok[0].Kind == token.KindAdd {
		at := parsetree.NewNonTerminal(parsetree.KindAddColumn)
		at.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		if p.tok[0].Kind == token.KindColumn {
			at.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		}

		if p.tok[0].Kind == token.KindIdentifier {
			at.AddChild(p.columnDefinition())
		} else {
			at.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing column definition`)))
		}
		nt.AddChild(at)
	} else if p.tok[0].Kind == token.KindDrop {
		dt := parsetree.NewNonTerminal(parsetree.KindDropColumn)
		dt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		if p.tok[0].Kind == token.KindColumn {
			dt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		}

		if p.tok[0].Kind == token.KindIdentifier {
			dt.AddChild(parsetree.NewTerminal(parsetree.KindColumnName, p.tok[0]))
			p.advance()
		} else {
			dt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing column name`)))
		}
		nt.AddChild(dt)
	}

	return nt
}

// columnDefinition parses a column definition.
func (p *Parser) columnDefinition() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindColumnDefinition)

	nt.AddChild(parsetree.NewTerminal(parsetree.KindColumnName, p.tok[0]))
	p.advance()

	if p.tok[0].Kind == token.KindIdentifier {
		nt.AddChild(p.typeName())
	}

	for slices.Contains([]token.Kind{token.KindConstraint, token.KindPrimary, token.KindNot, token.KindUnique, token.KindCheck,
		token.KindDefault, token.KindCollate, token.KindReferences, token.KindGenerated, token.KindAs},
		p.tok[0].Kind) {
		nt.AddChild(p.columnConstraint())
	}

	return nt
}

// typeName parses a typ name.
func (p *Parser) typeName() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindTypeName)
	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	p.advance()

	if p.tok[0].Kind == token.KindLeftParen {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else if slices.Contains([]token.Kind{token.KindNumeric, token.KindPlus, token.KindMinus}, p.tok[0].Kind) {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New("missing left paren")))
	} else {
		return nt
	}

	if p.tok[0].Kind == token.KindPlus || p.tok[0].Kind == token.KindMinus {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	}

	if p.tok[0].Kind == token.KindNumeric {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else if p.tok[0].Kind == token.KindRightParen || p.tok[0].Kind == token.KindComma {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing number`)))
	}

	if p.tok[0].Kind == token.KindRightParen {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
		return nt
	}

	p.skipTo(nt, token.KindComma, token.KindNumeric, token.KindSemicolon)

	if p.tok[0].Kind == token.KindComma {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else if p.tok[0].Kind == token.KindNumeric {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing comma`)))
	}

	p.skipTo(nt, token.KindNumeric, token.KindRightParen, token.KindSemicolon)

	if p.tok[0].Kind == token.KindNumeric {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else if p.tok[0].Kind == token.KindRightParen {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing number`)))
	}

	if p.tok[0].Kind == token.KindRightParen {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	}

	return nt
}

// columnConstraint parses a column constraint.
func (p *Parser) columnConstraint() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindColumnConstraint)
	if p.tok[0].Kind == token.KindConstraint {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
		if p.tok[0].Kind == token.KindIdentifier {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindConstraintName, p.tok[0]))
			p.advance()
		} else if slices.Contains(
			[]token.Kind{token.KindPrimary, token.KindNot, token.KindUnique, token.KindCheck, token.KindDefault,
				token.KindCollate, token.KindReferences},
			p.tok[0].Kind,
		) {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing constraint name`)))
		}
	}

	p.skipTo(nt,
		token.KindPrimary, token.KindNot, token.KindUnique, token.KindCheck, token.KindDefault,
		token.KindCollate, token.KindReferences, token.KindGenerated, token.KindAs, token.KindSemicolon)

	switch p.tok[0].Kind {
	case token.KindPrimary:
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		if p.tok[0].Kind == token.KindKey {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		} else if slices.Contains(
			[]token.Kind{token.KindAsc, token.KindDesc, token.KindOn, token.KindAutoincrement, token.KindSemicolon},
			p.tok[0].Kind,
		) {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "KEY"`)))
		}

		if p.tok[0].Kind == token.KindAsc || p.tok[0].Kind == token.KindDesc {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		}

		if p.tok[0].Kind == token.KindOn {
			nt.AddChild(p.conflictClause())
		}

		if p.tok[0].Kind == token.KindAutoincrement {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		}
	case token.KindNot:
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		if p.tok[0].Kind == token.KindNull {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		} else {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "NULL"`)))
		}

		if p.tok[0].Kind == token.KindOn {
			nt.AddChild(p.conflictClause())
		}
	case token.KindUnique:
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		if p.tok[0].Kind == token.KindOn {
			nt.AddChild(p.conflictClause())
		}
	case token.KindCheck:
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		if p.tok[0].Kind == token.KindLeftParen {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		} else if slices.Contains([]token.Kind{token.KindNumeric}, p.tok[0].Kind) {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "("`)))
		}

		if slices.Contains([]token.Kind{token.KindNumeric}, p.tok[0].Kind) {
			nt.AddChild(p.expression())
		} else if p.tok[0].Kind == token.KindRightParen {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing check expression`)))
		}

		if p.tok[0].Kind == token.KindRightParen {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		} else {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing ")"`)))
		}
	case token.KindDefault:
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		if p.tok[0].Kind == token.KindLeftParen {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()

			if slices.Contains([]token.Kind{token.KindNumeric}, p.tok[0].Kind) {
				nt.AddChild(p.expression())
			} else if p.tok[0].Kind == token.KindRightParen {
				nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing expression`)))
			}

			if p.tok[0].Kind == token.KindRightParen {
				nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
				p.advance()
			} else {
				nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing ")"`)))
			}
		} else if p.isLiteralValue(p.tok[0]) {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		} else if p.tok[0].Kind == token.KindPlus || p.tok[0].Kind == token.KindMinus {
			// note that a numeric token is a literal value
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()

			if p.tok[0].Kind == token.KindNumeric {
				nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
				p.advance()
			} else {
				nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing numeric literal`)))
			}
		} else {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorExpecting, errors.New(`expecting "(", a literal value , or a signed number`)))
		}
	case token.KindCollate:
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		if p.tok[0].Kind == token.KindIdentifier {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindCollationName, p.tok[0]))
			p.advance()
		} else {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing collation name`)))
		}
	case token.KindReferences:
		nt.AddChild(p.foreignKeyClause())
	case token.KindGenerated, token.KindAs:
		if p.tok[0].Kind == token.KindGenerated {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()

			if p.tok[0].Kind == token.KindAlways {
				nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
				p.advance()
			} else {
				nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "ALWAYS"`)))
			}
		}

		if p.tok[0].Kind == token.KindAs {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		} else if p.tok[0].Kind == token.KindLeftParen {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "AS"`)))
		}

		if p.tok[0].Kind == token.KindLeftParen {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		} else if slices.Contains([]token.Kind{token.KindNumeric}, p.tok[0].Kind) {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "("`)))
		}

		if slices.Contains([]token.Kind{token.KindNumeric}, p.tok[0].Kind) {
			nt.AddChild(p.expression())
		} else if p.tok[0].Kind == token.KindRightParen {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing expression`)))
		}

		p.skipTo(nt, token.KindRightParen, token.KindSemicolon)

		if p.tok[0].Kind == token.KindRightParen {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		} else {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing ")"`)))
		}

		if p.tok[0].Kind == token.KindStored || p.tok[0].Kind == token.KindVirtual {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		}
	default:
		nt.AddChild(parsetree.NewError(parsetree.KindErrorExpecting, errors.New(`expecting "PRIMARY", "NOT", "UNIQUE", "CHECK", "DEFAULT", "COLLATE", "REFERENCES", "GENERATED", or "AS"`)))
	}

	return nt
}

// foreignKeyClause parses a foreign key clause.
func (p *Parser) foreignKeyClause() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindForeignKeyClause)

	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	p.advance()

	if p.tok[0].Kind == token.KindIdentifier {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindTableName, p.tok[0]))
		p.advance()
	} else {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing table name`)))
	}

	p.skipTo(nt, token.KindLeftParen, token.KindOn, token.KindMatch, token.KindDeferrable, token.KindNot, token.KindSemicolon)

	if p.tok[0].Kind == token.KindLeftParen {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		list := parsetree.NewNonTerminal(parsetree.KindCommaList)

		var skipped bool

		if p.tok[0].Kind == token.KindIdentifier {
			list.AddChild(parsetree.NewTerminal(parsetree.KindColumnName, p.tok[0]))
			p.advance()
		} else if slices.Contains([]token.Kind{token.KindComma, token.KindRightParen, token.KindSemicolon}, p.tok[0].Kind) {
			list.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing column name`)))
		} else {
			p.skipTo(list, token.KindIdentifier, token.KindComma, token.KindRightParen, token.KindSemicolon)
			skipped = true
		}

		for p.tok[0].Kind != token.KindRightParen && p.tok[0].Kind != token.KindSemicolon {
			if p.tok[0].Kind == token.KindComma {
				list.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
				p.advance()
				skipped = false
			} else if p.tok[0].Kind == token.KindIdentifier {
				if !skipped {
					list.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing comma`)))
				}
				skipped = false
			} else {
				p.skipTo(list, token.KindIdentifier, token.KindComma, token.KindRightParen, token.KindSemicolon)
				skipped = true
			}

			if p.tok[0].Kind == token.KindIdentifier {
				list.AddChild(parsetree.NewTerminal(parsetree.KindColumnName, p.tok[0]))
				p.advance()
				skipped = false
			} else if p.tok[0].Kind == token.KindComma || p.tok[0].Kind == token.KindRightParen || p.tok[0].Kind == token.KindSemicolon {
				if !skipped {
					list.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing column name`)))
				}
				skipped = false
			} else {
				p.skipTo(list, token.KindComma, token.KindRightParen, token.KindSemicolon)
				skipped = true
			}
		}

		nt.AddChild(list)

		if p.tok[0].Kind == token.KindRightParen {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		} else {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing ")"`)))
		}
	}

	for slices.Contains([]token.Kind{token.KindOn, token.KindMatch}, p.tok[0].Kind) {
		if p.tok[0].Kind == token.KindOn {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()

			if p.tok[0].Kind == token.KindDelete || p.tok[0].Kind == token.KindUpdate {
				nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
				p.advance()
			} else if slices.Contains(
				[]token.Kind{token.KindSet, token.KindCascade, token.KindRestrict, token.KindNo},
				p.tok[0].Kind,
			) {
				nt.AddChild(parsetree.NewError(parsetree.KindErrorExpecting, errors.New(`expecting "DELETE", or "UPDATE"`)))
			}

			if p.tok[0].Kind == token.KindSet {
				nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
				p.advance()

				if p.tok[0].Kind == token.KindNull || p.tok[0].Kind == token.KindDefault {
					nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
					p.advance()
				} else {
					nt.AddChild(parsetree.NewError(parsetree.KindErrorExpecting, errors.New(`expecting "NULL", or "DEFAULT"`)))
				}
			} else if p.tok[0].Kind == token.KindCascade {
				nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
				p.advance()
			} else if p.tok[0].Kind == token.KindRestrict {
				nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
				p.advance()
			} else if p.tok[0].Kind == token.KindNo {
				nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
				p.advance()

				if p.tok[0].Kind == token.KindAction {
					nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
					p.advance()
				} else {
					nt.AddChild(parsetree.NewError(parsetree.KindErrorExpecting, errors.New(`expecting "ACTION"`)))
				}
			} else {
				nt.AddChild(parsetree.NewError(parsetree.KindErrorExpecting, errors.New(`expecting "SET", "CASCADE", "RESTRICT", or "NO"`)))
			}
		} else if p.tok[0].Kind == token.KindMatch {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()

			if p.tok[0].Kind == token.KindIdentifier {
				nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
				p.advance()
			} else {
				nt.AddChild(parsetree.NewError(parsetree.KindErrorExpecting, errors.New(`expecting a name`)))
			}
		}
	}

	if p.tok[0].Kind == token.KindDeferrable || p.tok[0].Kind == token.KindNot {
		if p.tok[0].Kind == token.KindNot {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		}
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		if p.tok[0].Kind == token.KindInitially {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()

			if p.tok[0].Kind == token.KindDeferred || p.tok[0].Kind == token.KindImmediate {
				nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
				p.advance()
			} else {
				nt.AddChild(parsetree.NewError(parsetree.KindErrorExpecting, errors.New(`expecting a "DEFERRED", or "IMMEDIATE"`)))
			}
		}
	}

	return nt
}

// isLiteralValue reports if tok is a literal value.
func (p *Parser) isLiteralValue(tok *token.Token) bool {
	if slices.Contains(
		[]token.Kind{token.KindNumeric, token.KindString, token.KindBlob,
			token.KindNull, token.KindCurrentTime, token.KindCurrentDate,
			token.KindCurrentTimestamp},
		tok.Kind,
	) {
		return true
	}

	if tok.Kind != token.KindIdentifier {
		return false
	}

	lex := strings.ToLower(string(tok.Lexeme))
	return lex == "true" || lex == "false"
}

// conflictClause parses a conflict clause.
func (p *Parser) conflictClause() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindConflictClause)
	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	p.advance()

	if p.tok[0].Kind == token.KindConflict {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else if slices.Contains(
		[]token.Kind{token.KindRollback, token.KindAbort, token.KindFail, token.KindIgnore, token.KindIgnore}, p.tok[0].Kind,
	) {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "CONFLICT"`)))
	}

	p.skipTo(nt, token.KindRollback, token.KindAbort, token.KindFail, token.KindIgnore, token.KindReplace, token.KindSemicolon)

	if slices.Contains(
		[]token.Kind{token.KindRollback, token.KindAbort, token.KindFail, token.KindIgnore, token.KindReplace}, p.tok[0].Kind,
	) {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorExpecting, errors.New(`expecting "ROLLBACK", "ABORT", "FAIL", "IGNORE", or "IGNORE"`)))
	}

	return nt
}

// expression parses a expression.
func (p *Parser) expression() parsetree.NonTerminal {
	// TODO: implement the parsing of expressions
	nt := parsetree.NewNonTerminal(parsetree.KindExpression)
	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	p.advance()
	return nt
}

// advance advances the lexer and put the next comments in p.comments
// and the token after the comments in p.tok.
func (p *Parser) advance() {
	var tok *token.Token
	var comments []*token.Token
	for {
		tok = p.l.Next()
		if tok.Kind == token.KindSQLComment || tok.Kind == token.KindCComment {
			comments = append(comments, tok)
		} else if tok.Kind != token.KindWhiteSpace {
			if len(comments) > 0 {
				p.comments[tok] = comments
			}
			p.tok[0] = p.tok[1]
			p.tok[1] = tok
			break
		}
	}
}

// skipTo do nothing if set contains the kind of p.tok, or put a parse tree with kind
// unexpected EOF in father if the kind of p.tok is EOF, or advances until set contains
// the kind of p.tok[0] or the EOF is found. In the later case the skipped tokens, if any,
// are put in father. The return is true if set contains the kind of token in p.tok[0] after
// the advances, if any. The set must not contain token.KindEof.
func (p *Parser) skipTo(father parsetree.NonTerminal, set ...token.Kind) bool {
	if slices.Contains(set, p.tok[0].Kind) {
		return true
	}
	if p.tok[0].Kind == token.KindEOF {
		father.AddChild(parsetree.NewError(parsetree.KindErrorUnexpectedEOF, errors.New("unexpected EOF")))
		return false
	}
	skipped := parsetree.NewNonTerminal(parsetree.KindSkipped)
	for !slices.Contains(set, p.tok[0].Kind) && p.tok[0].Kind != token.KindEOF {
		skipped.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	}
	father.AddChild(skipped)
	return p.tok[0].Kind != token.KindEOF
}
