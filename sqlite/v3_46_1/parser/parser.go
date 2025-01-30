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
	// tok contains the current look ahead tokens.
	tok [3]*token.Token
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
	comments = make(map[*token.Token][]*token.Token)
	p.comments = comments

	if p.tok[0] == nil {
		p.advance()
		p.advance()
		p.advance()
	}

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
	case token.KindSelect:
		father.AddChild(p.selectStatement())
	case token.KindAnalyze:
		father.AddChild(p.analyse())
	case token.KindAttach:
		father.AddChild(p.attach())
	case token.KindBegin:
		father.AddChild(p.begin())
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

// typeName parses a type name.
func (p *Parser) typeName() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindTypeName)

	for p.tok[0].Kind == token.KindIdentifier {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	}

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
		} else if p.isExpressionStart(p.tok[0]) {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "("`)))
		}

		if p.isExpressionStart(p.tok[0]) {
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

			if p.isExpressionStart(p.tok[0]) {
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
		} else if p.isExpressionStart(p.tok[0]) {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "("`)))
		}

		if p.isExpressionStart(p.tok[0]) {
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
				p.skipTo(list, token.KindIdentifier, token.KindComma, token.KindRightParen, token.KindSemicolon)
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

// selectStatement parses a select statement.
func (p *Parser) selectStatement() parsetree.NonTerminal {
	// TODO: implement this method.
	nt := parsetree.NewNonTerminal(parsetree.KindSelect)
	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	p.advance()

	if p.isExpressionStart(p.tok[0]) {
		nt.AddChild(p.expression())
	}

	return nt
}

// analyse parses a analyse statement.
func (p *Parser) analyse() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindAnalyze)
	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	p.advance()

	if p.tok[0].Kind == token.KindIdentifier && p.tok[1].Kind == token.KindDot {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindSchemaName, p.tok[0]))
		p.advance()
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
		if p.tok[0].Kind == token.KindIdentifier {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindTableOrIndexName, p.tok[0]))
			p.advance()
		} else {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing identifier`)))
		}
	} else if p.tok[0].Kind == token.KindIdentifier {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindSchemaIndexOrTableName, p.tok[0]))
		p.advance()
	} else if p.tok[0].Kind == token.KindDot {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing identifier`)))
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
		if p.tok[0].Kind == token.KindIdentifier {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindTableOrIndexName, p.tok[0]))
			p.advance()
		} else {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing identifier`)))
		}
	} else {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing identifier`)))
	}

	return nt
}

// attach parses a attach statement.
func (p *Parser) attach() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindAttach)
	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	p.advance()

	if p.tok[0].Kind == token.KindDatabase {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	}

	if p.isExpressionStart(p.tok[0]) {
		nt.AddChild(p.expression())
	} else if p.tok[0].Kind == token.KindAs {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing expression`)))
	}

	if p.tok[0].Kind == token.KindAs {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else if p.tok[0].Kind == token.KindIdentifier {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "AS"`)))
	}

	if p.tok[0].Kind == token.KindIdentifier {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindSchemaName, p.tok[0]))
		p.advance()
	} else {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing schema name`)))
	}

	return nt
}

// begin parses a begin statement.
func (p *Parser) begin() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindBegin)
	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	p.advance()

	if p.tok[0].Kind == token.KindDeferred || p.tok[0].Kind == token.KindImmediate || p.tok[0].Kind == token.KindExclusive {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	}

	if p.tok[0].Kind == token.KindTransaction {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	}

	return nt
}

// expression parses a expression.
func (p *Parser) expression() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindExpression)
	nt.AddChild(p.expression1())
	return nt
}

// expression1 parses a expression with precedence at least 1 (1 is the lowest precedence).
func (p *Parser) expression1() parsetree.Construction {
	exp := p.expression2()

	for p.tok[0].Kind == token.KindOr {
		nt := parsetree.NewNonTerminal(parsetree.KindOr)
		nt.AddChild(exp)

		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		if p.isExpressionStart(p.tok[0]) {
			nt.AddChild(p.expression2())
		} else {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing expression`)))
			return nt
		}

		exp = nt
	}

	return exp
}

// expression2 parses a expression with precedence at least 2.
func (p *Parser) expression2() parsetree.Construction {
	exp := p.expression3()

	for p.tok[0].Kind == token.KindAnd {
		nt := parsetree.NewNonTerminal(parsetree.KindAnd)
		nt.AddChild(exp)

		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		if p.isExpressionStart(p.tok[0]) {
			nt.AddChild(p.expression3())
		} else {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing expression`)))
			return nt
		}

		exp = nt
	}

	return exp
}

// expression3 parses a expression with precedence at least 3.
func (p *Parser) expression3() parsetree.Construction {
	if p.tok[0].Kind == token.KindNot && p.tok[1].Kind != token.KindExists {
		nt := parsetree.NewNonTerminal(parsetree.KindNot)
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		if p.isExpressionStart(p.tok[0]) {
			nt.AddChild(p.expression3())
		} else {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing expression`)))
		}

		return nt
	}

	return p.expression4()
}

// expression4 parses a expression with precedence at least 4.
func (p *Parser) expression4() parsetree.Construction {
	exp := p.expression5()

	for {
		switch p.tok[0].Kind {
		case token.KindEqual, token.KindEqualEqual:
			nt := parsetree.NewNonTerminal(parsetree.KindEqual)
			nt.AddChild(exp)

			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()

			if p.isExpressionAtLeast4Start(p.tok[0]) {
				nt.AddChild(p.expression5())
			} else {
				nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing expression (not starting with "NOT")`)))
				return nt
			}

			exp = nt
		case token.KindLessThanGreaterThan, token.KindExclamationEqual:
			nt := parsetree.NewNonTerminal(parsetree.KindNotEqual)
			nt.AddChild(exp)

			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()

			if p.isExpressionAtLeast4Start(p.tok[0]) {
				nt.AddChild(p.expression5())
			} else {
				nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing expression (not starting with "NOT")`)))
				return nt
			}

			exp = nt
		case token.KindIs:
			exp = p.isExpression(exp)
		case token.KindBetween:
			exp = p.between(exp)
		case token.KindIn:
			exp = p.in(exp)
		case token.KindGlob:
			nt := parsetree.NewNonTerminal(parsetree.KindGlob)
			nt.AddChild(exp)
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()

			if p.isExpressionAtLeast4Start(p.tok[0]) {
				nt.AddChild(p.expression5())
			} else {
				nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing expression (not starting with "NOT")`)))
				return nt
			}

			exp = nt
		case token.KindRegexp:
			nt := parsetree.NewNonTerminal(parsetree.KindRegexp)
			nt.AddChild(exp)
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()

			if p.isExpressionAtLeast4Start(p.tok[0]) {
				nt.AddChild(p.expression5())
			} else {
				nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing expression (not starting with "NOT")`)))
				return nt
			}

			exp = nt
		case token.KindMatch:
			nt := parsetree.NewNonTerminal(parsetree.KindMatch)
			nt.AddChild(exp)
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()

			if p.isExpressionAtLeast4Start(p.tok[0]) {
				nt.AddChild(p.expression5())
			} else {
				nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing expression (not starting with "NOT")`)))
				return nt
			}

			exp = nt
		case token.KindLike:
			nt := parsetree.NewNonTerminal(parsetree.KindLike)
			nt.AddChild(exp)
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()

			if p.isExpressionAtLeast4Start(p.tok[0]) {
				nt.AddChild(p.expression5())
			} else {
				nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing expression (not starting with "NOT")`)))
			}

			// apparently there is an error in the ESCAPE precedence documentation
			if p.tok[0].Kind == token.KindEscape {
				nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
				p.advance()

				if p.isExpressionAtLeast4Start(p.tok[0]) {
					nt.AddChild(p.expression4())
				} else {
					nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing expression (not starting with "NOT")`)))
					return nt
				}
			}

			exp = nt
		case token.KindIsnull:
			nt := parsetree.NewNonTerminal(parsetree.KindIsNull)
			nt.AddChild(exp)
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
			exp = nt
		case token.KindNotnull:
			nt := parsetree.NewNonTerminal(parsetree.KindNotnull)
			nt.AddChild(exp)
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
			exp = nt
		case token.KindNot:
			if p.tok[1].Kind == token.KindBetween {
				exp = p.notBetween(exp)
			} else if p.tok[1].Kind == token.KindIn {
				exp = p.notIn(exp)
			} else if p.tok[1].Kind == token.KindGlob {
				nt := parsetree.NewNonTerminal(parsetree.KindNotGlob)
				nt.AddChild(exp)
				nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
				p.advance()
				nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
				p.advance()

				if p.isExpressionAtLeast4Start(p.tok[0]) {
					nt.AddChild(p.expression5())
				} else {
					nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing expression (not starting with "NOT")`)))
					return nt
				}

				exp = nt
			} else if p.tok[1].Kind == token.KindRegexp {
				nt := parsetree.NewNonTerminal(parsetree.KindNotRegexp)
				nt.AddChild(exp)
				nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
				p.advance()
				nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
				p.advance()

				if p.isExpressionAtLeast4Start(p.tok[0]) {
					nt.AddChild(p.expression5())
				} else {
					nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing expression (not starting with "NOT")`)))
					return nt
				}

				exp = nt
			} else if p.tok[1].Kind == token.KindMatch {
				nt := parsetree.NewNonTerminal(parsetree.KindNotMatch)
				nt.AddChild(exp)
				nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
				p.advance()
				nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
				p.advance()

				if p.isExpressionAtLeast4Start(p.tok[0]) {
					nt.AddChild(p.expression5())
				} else {
					nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing expression (not starting with "NOT")`)))
					return nt
				}

				exp = nt
			} else if p.tok[1].Kind == token.KindLike {
				nt := parsetree.NewNonTerminal(parsetree.KindNotLike)
				nt.AddChild(exp)
				nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
				p.advance()
				nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
				p.advance()

				if p.isExpressionAtLeast4Start(p.tok[0]) {
					nt.AddChild(p.expression5())
				} else {
					nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing expression (not starting with "NOT")`)))
				}

				if p.tok[0].Kind == token.KindEscape {
					nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
					p.advance()

					if p.isExpressionAtLeast4Start(p.tok[0]) {
						nt.AddChild(p.expression5())
					} else {
						nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing expression (not starting with "NOT")`)))
						return nt
					}
				}

				exp = nt
			} else if p.tok[1].Kind == token.KindNull {
				nt := parsetree.NewNonTerminal(parsetree.KindNotNull)
				nt.AddChild(exp)
				nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
				p.advance()
				nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
				p.advance()
				exp = nt
			}
		default:
			return exp
		}
	}
}

// isExpression parses a is, is not, is distinct, and is not distinct expression.
func (p *Parser) isExpression(exp parsetree.Construction) parsetree.NonTerminal {
	var nt parsetree.NonTerminal
	if p.tok[1].Kind == token.KindNot {
		if p.isExpressionStart(p.tok[2]) && p.tok[2].Kind != token.KindNot {
			nt = parsetree.NewNonTerminal(parsetree.KindIsNot)
			nt.AddChild(exp)
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
			nt.AddChild(p.expression4())
		} else if p.tok[2].Kind == token.KindDistinct {
			nt = parsetree.NewNonTerminal(parsetree.KindIsNotDistinctFrom)
			nt.AddChild(exp)
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
			if p.tok[0].Kind == token.KindFrom {
				nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
				p.advance()
			} else if p.isExpressionStart(p.tok[0]) {
				nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "FROM"`)))
			}

			if p.isExpressionStart(p.tok[0]) && p.tok[0].Kind != token.KindNot {
				nt.AddChild(p.expression4())
			} else {
				nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing expression (not starting with "NOT")`)))
			}
		} else {
			nt = parsetree.NewNonTerminal(parsetree.KindIsNot)
			nt.AddChild(exp)
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
			nt.AddChild(parsetree.NewError(parsetree.KindErrorExpecting, errors.New(`expecting "DISTINCT", or a expression (not starting with "NOT")`)))
		}
	} else if p.tok[1].Kind == token.KindDistinct {
		nt = parsetree.NewNonTerminal(parsetree.KindIsDistinctFrom)
		nt.AddChild(exp)
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
		if p.tok[0].Kind == token.KindFrom {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		} else if p.isExpressionStart(p.tok[0]) {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "FROM"`)))
		}

		if p.isExpressionStart(p.tok[0]) && p.tok[0].Kind != token.KindNot {
			nt.AddChild(p.expression4())
		} else {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing expression (not starting with "NOT")`)))
		}
	} else if p.isExpressionStart(p.tok[1]) {
		nt = parsetree.NewNonTerminal(parsetree.KindIs)
		nt.AddChild(exp)
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		nt.AddChild(p.expression4())
	} else {
		nt = parsetree.NewNonTerminal(parsetree.KindIs)
		nt.AddChild(exp)
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
		nt.AddChild(parsetree.NewError(parsetree.KindErrorExpecting, errors.New(`expecting "NOT", "DISTINCT", or a expression`)))
	}

	return nt
}

// between parses a between expression.
func (p *Parser) between(exp parsetree.Construction) parsetree.NonTerminal {
	var nt parsetree.NonTerminal
	nt = parsetree.NewNonTerminal(parsetree.KindBetween)
	nt.AddChild(exp)
	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	p.advance()

	if p.isExpressionAtLeast4Start(p.tok[0]) {
		nt.AddChild(p.expression4())
	} else if p.tok[0].Kind == token.KindAnd {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing expression (not starting with "NOT")`)))
	}

	if p.tok[0].Kind == token.KindAnd {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else if p.isExpressionAtLeast4Start(p.tok[0]) {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "AND"`)))
	}

	if p.isExpressionAtLeast4Start(p.tok[0]) {
		nt.AddChild(p.expression4())
	} else {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing expression (not starting with "NOT")`)))
	}

	return nt
}

// notBetween parses a not between expression.
func (p *Parser) notBetween(exp parsetree.Construction) parsetree.NonTerminal {
	var nt parsetree.NonTerminal
	nt = parsetree.NewNonTerminal(parsetree.KindNotBetween)
	nt.AddChild(exp)
	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	p.advance()
	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	p.advance()

	if p.isExpressionAtLeast4Start(p.tok[0]) {
		nt.AddChild(p.expression4())
	} else if p.tok[0].Kind == token.KindAnd {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing expression (not starting with "NOT")`)))
	}

	if p.tok[0].Kind == token.KindAnd {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else if p.isExpressionAtLeast4Start(p.tok[0]) {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "AND"`)))
	}

	if p.isExpressionAtLeast4Start(p.tok[0]) {
		nt.AddChild(p.expression4())
	} else {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing expression (not starting with "NOT")`)))
	}

	return nt
}

// in parses a in expression.
func (p *Parser) in(exp parsetree.Construction) parsetree.NonTerminal {
	var nt parsetree.NonTerminal
	nt = parsetree.NewNonTerminal(parsetree.KindIn)
	nt.AddChild(exp)
	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	p.advance()

	if p.tok[0].Kind == token.KindLeftParen {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		if p.tok[0].Kind == token.KindRightParen {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
			return nt
		}

		if p.tok[0].Kind == token.KindSelect {
			nt.AddChild(p.selectStatement())
		} else if p.isExpressionAtLeast4Start(p.tok[0]) {
			nt.AddChild(p.commaListConstruction("expression", p.expression4, p.isExpressionAtLeast4Start, func(t *token.Token) bool {
				return t.Kind == token.KindRightParen || t.Kind == token.KindSemicolon
			}))
		} else {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorExpecting, errors.New(`expecting select statement, or expression (not starting with "NOT")`)))
			p.skipTo(nt, token.KindRightParen, token.KindSemicolon)
		}

		if p.tok[0].Kind == token.KindRightParen {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
			return nt
		} else {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing ")"`)))
		}
		return nt
	}

	if p.tok[0].Kind == token.KindIdentifier && p.tok[1].Kind == token.KindDot {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindSchemaName, p.tok[0]))
		p.advance()
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	}

	if p.tok[0].Kind == token.KindIdentifier && p.tok[1].Kind == token.KindLeftParen {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindTableFunctionName, p.tok[0]))
		p.advance()
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		if p.isExpressionAtLeast4Start(p.tok[0]) {
			nt.AddChild(p.commaListConstruction("expression", p.expression4, p.isExpressionAtLeast4Start, func(t *token.Token) bool {
				return t.Kind == token.KindRightParen || t.Kind == token.KindSemicolon
			}))
		} else if p.tok[0].Kind == token.KindRightParen {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing expression (not starting with "NOT")`)))
		}

		if p.tok[0].Kind == token.KindRightParen {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
			return nt
		} else {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing ")"`)))
		}
	} else if p.tok[0].Kind == token.KindIdentifier {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindTableName, p.tok[0]))
		p.advance()
	} else {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorExpecting, errors.New(`expecting "(", schema name, table name, or table function`)))
	}

	return nt
}

// notIn parses a not in expression.
func (p *Parser) notIn(exp parsetree.Construction) parsetree.NonTerminal {
	var nt parsetree.NonTerminal
	nt = parsetree.NewNonTerminal(parsetree.KindNotIn)
	nt.AddChild(exp)
	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	p.advance()
	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	p.advance()

	if p.tok[0].Kind == token.KindLeftParen {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		if p.tok[0].Kind == token.KindRightParen {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
			return nt
		}

		if p.tok[0].Kind == token.KindSelect {
			nt.AddChild(p.selectStatement())
		} else if p.isExpressionAtLeast4Start(p.tok[0]) {
			nt.AddChild(p.commaListConstruction("expression", p.expression4, p.isExpressionAtLeast4Start, func(t *token.Token) bool {
				return t.Kind == token.KindRightParen || t.Kind == token.KindSemicolon
			}))
		} else {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorExpecting, errors.New(`expecting select statement, or expression (not starting with "NOT")`)))
			p.skipTo(nt, token.KindRightParen, token.KindSemicolon)
		}

		if p.tok[0].Kind == token.KindRightParen {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
			return nt
		} else {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing ")"`)))
		}
		return nt
	}

	if p.tok[0].Kind == token.KindIdentifier && p.tok[1].Kind == token.KindDot {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindSchemaName, p.tok[0]))
		p.advance()
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	}

	if p.tok[0].Kind == token.KindIdentifier && p.tok[1].Kind == token.KindLeftParen {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindTableFunctionName, p.tok[0]))
		p.advance()
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		if p.isExpressionAtLeast4Start(p.tok[0]) {
			nt.AddChild(p.commaListConstruction("expression", p.expression4, p.isExpressionAtLeast4Start, func(t *token.Token) bool {
				return t.Kind == token.KindRightParen || t.Kind == token.KindSemicolon
			}))
		} else if p.tok[0].Kind == token.KindRightParen {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing expression (not starting with "NOT")`)))
		}

		if p.tok[0].Kind == token.KindRightParen {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
			return nt
		} else {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing ")"`)))
		}
	} else if p.tok[0].Kind == token.KindIdentifier {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindTableName, p.tok[0]))
		p.advance()
	} else {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorExpecting, errors.New(`expecting "(", schema name, table name, or table function`)))
	}

	return nt
}

// isExpressionAtLeast4Start reports if
func (p *Parser) isExpressionAtLeast4Start(tok *token.Token) bool {
	return p.isExpressionStart(p.tok[0]) && p.tok[0].Kind != token.KindNot
}

// expression5 parses a expression with precedence at least 5.
func (p *Parser) expression5() parsetree.Construction {
	exp := p.expression6()

	for {
		var nt parsetree.NonTerminal

		if p.tok[0].Kind == token.KindLessThan {
			nt = parsetree.NewNonTerminal(parsetree.KindLessThan)
		} else if p.tok[0].Kind == token.KindLessThanOrEqual {
			nt = parsetree.NewNonTerminal(parsetree.KindLessThanOrEqual)
		} else if p.tok[0].Kind == token.KindGreaterThan {
			nt = parsetree.NewNonTerminal(parsetree.KindGreaterThan)
		} else if p.tok[0].Kind == token.KindGreaterThanOrEqual {
			nt = parsetree.NewNonTerminal(parsetree.KindGreaterThanOrEqual)
		} else {
			return exp
		}

		nt.AddChild(exp)

		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		if p.isExpressionAtLeast4Start(p.tok[0]) {
			nt.AddChild(p.expression6())
		} else {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing expression (not starting with "NOT")`)))
			return nt
		}

		exp = nt
	}
}

// expression6 parses a expression with precedence at least 6.
func (p *Parser) expression6() parsetree.Construction {
	// apparently there is an error in the ESCAPE precedence documentation.

	exp := p.expression7()
	for {
		var nt parsetree.NonTerminal

		if p.tok[0].Kind == token.KindAmpersand {
			nt = parsetree.NewNonTerminal(parsetree.KindBitAnd)
		} else if p.tok[0].Kind == token.KindPipe {
			nt = parsetree.NewNonTerminal(parsetree.KindBitOr)
		} else if p.tok[0].Kind == token.KindLessThanLessThan {
			nt = parsetree.NewNonTerminal(parsetree.KindLeftShift)
		} else if p.tok[0].Kind == token.KindGreaterThanGreaterThan {
			nt = parsetree.NewNonTerminal(parsetree.KindRightShift)
		} else {
			return exp
		}
		nt.AddChild(exp)

		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		if p.isExpressionAtLeast4Start(p.tok[0]) {
			nt.AddChild(p.expression7())
		} else {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing expression (not starting with "NOT")`)))
			return nt
		}

		exp = nt
	}
}

// expression7 parses a expression with precedence at least 7.
func (p *Parser) expression7() parsetree.Construction {
	exp := p.expression8()
	for {
		var nt parsetree.NonTerminal

		if p.tok[0].Kind == token.KindPlus {
			nt = parsetree.NewNonTerminal(parsetree.KindAdd)
		} else if p.tok[0].Kind == token.KindMinus {
			nt = parsetree.NewNonTerminal(parsetree.KindSubtract)
		} else {
			return exp
		}
		nt.AddChild(exp)

		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		if p.isExpressionAtLeast4Start(p.tok[0]) {
			nt.AddChild(p.expression8())
		} else {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing expression (not starting with "NOT")`)))
			return nt
		}

		exp = nt
	}
}

// expression8 parses a expression with precedence at least 8.
func (p *Parser) expression8() parsetree.Construction {
	exp := p.expression9()
	for {
		var nt parsetree.NonTerminal

		if p.tok[0].Kind == token.KindAsterisk {
			nt = parsetree.NewNonTerminal(parsetree.KindMultiply)
		} else if p.tok[0].Kind == token.KindSlash {
			nt = parsetree.NewNonTerminal(parsetree.KindDivide)
		} else if p.tok[0].Kind == token.KindPercent {
			nt = parsetree.NewNonTerminal(parsetree.KindMod)
		} else {
			return exp
		}
		nt.AddChild(exp)

		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		if p.isExpressionAtLeast4Start(p.tok[0]) {
			nt.AddChild(p.expression9())
		} else {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing expression (not starting with "NOT")`)))
			return nt
		}

		exp = nt
	}
}

// expression9 parses a expression with precedence at least 9.
func (p *Parser) expression9() parsetree.Construction {
	exp := p.expression10()
	for {
		var nt parsetree.NonTerminal

		if p.tok[0].Kind == token.KindPipePipe {
			nt = parsetree.NewNonTerminal(parsetree.KindConcatenate)
		} else if p.tok[0].Kind == token.KindMinusGreaterThan {
			nt = parsetree.NewNonTerminal(parsetree.KindExtract1)
		} else if p.tok[0].Kind == token.KindMinusGreaterThanGreaterThan {
			nt = parsetree.NewNonTerminal(parsetree.KindExtract2)
		} else {
			return exp
		}
		nt.AddChild(exp)

		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		if p.isExpressionAtLeast4Start(p.tok[0]) {
			nt.AddChild(p.expression10())
		} else {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing expression (not starting with "NOT")`)))
		}

		exp = nt
	}
}

// expression10 parses a expression with precedence at least 10.
func (p *Parser) expression10() parsetree.Construction {
	exp := p.expression11()

	if p.tok[0].Kind != token.KindCollate {
		return exp
	}

	nt := parsetree.NewNonTerminal(parsetree.KindCollate)
	nt.AddChild(exp)
	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	p.advance()

	if p.tok[0].Kind == token.KindIdentifier {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindCollationName, p.tok[0]))
		p.advance()
	} else {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing collation name`)))
	}

	return nt
}

// expression11 parses a expression with precedence at least 11.
func (p *Parser) expression11() parsetree.Construction {
	var nt parsetree.NonTerminal

	if p.tok[0].Kind == token.KindTilde {
		nt = parsetree.NewNonTerminal(parsetree.KindBitNot)
	} else if p.tok[0].Kind == token.KindPlus {
		nt = parsetree.NewNonTerminal(parsetree.KindPrefixPlus)
	} else if p.tok[0].Kind == token.KindMinus {
		nt = parsetree.NewNonTerminal(parsetree.KindNegate)
	} else {
		return p.simpleExpression()
	}

	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	p.advance()

	if p.isExpressionAtLeast4Start(p.tok[0]) {
		nt.AddChild(p.expression11())
	} else {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing expression (not starting with "NOT")`)))
	}

	return nt
}

// simpleExpression parses a simple expression, that is, a expression with the highest precedence.
func (p *Parser) simpleExpression() parsetree.Construction {
	if p.isLiteralValue(p.tok[0]) {
		t := parsetree.NewTerminal(parsetree.KindToken, p.tok[0])
		p.advance()
		return t
	} else if p.isBindParameter(p.tok[0]) {
		t := parsetree.NewTerminal(parsetree.KindBindParameter, p.tok[0])
		p.advance()
		return t
	} else if p.tok[0].Kind == token.KindIdentifier && p.tok[1].Kind == token.KindLeftParen {
		return p.functionCall()
	} else if p.tok[0].Kind == token.KindIdentifier {
		return p.columnReference()
	} else if p.tok[0].Kind == token.KindLeftParen {
		return p.parenExpression()
	} else if p.tok[0].Kind == token.KindCast {
		return p.castExpression()
	} else if p.tok[0].Kind == token.KindNot {
		nt := parsetree.NewNonTerminal(parsetree.KindNot)
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		// can only be token.KindExists, see expression3
		nt.AddChild(p.exists())

		return nt
	} else if p.tok[0].Kind == token.KindExists {
		return p.exists()
	} else if p.tok[0].Kind == token.KindCase {
		return p.caseExpression()
	} else { // token.KindRaise
		return p.raise()
	}
}

// isExpressionStart reports if tok is a start of expression.
func (p *Parser) isExpressionStart(tok *token.Token) bool {
	if p.isLiteralValue(tok) {
		return true
	}
	if p.isBindParameter(tok) {
		return true
	}

	switch tok.Kind {
	case token.KindIdentifier, token.KindTilde, token.KindPlus, token.KindMinus, token.KindNot, token.KindLeftParen,
		token.KindCast, token.KindExists, token.KindCase, token.KindRaise:
		return true
	}

	return false
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

// isBindParameter reports if tok is a bind parameter.
func (p *Parser) isBindParameter(tok *token.Token) bool {
	switch tok.Kind {
	case token.KindAtVariable, token.KindColonVariable, token.KindDollarVariable, token.KindQuestionVariable:
		return true
	}

	return false
}

// columnReference parses a column reference.
func (p *Parser) columnReference() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindColumnReference)

	var tokens []*token.Token
	tokens = append(tokens, p.tok[0])
	p.advance()

	if p.tok[0].Kind == token.KindDot && p.tok[1].Kind == token.KindIdentifier {
		tokens = append(tokens, p.tok[0], p.tok[1])
		p.advance()
		p.advance()
	} else {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindColumnName, tokens[0]))
		return nt
	}

	if p.tok[0].Kind == token.KindDot && p.tok[1].Kind == token.KindIdentifier {
		tokens = append(tokens, p.tok[0], p.tok[1])
		p.advance()
		p.advance()
	} else {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindTableName, tokens[0]))
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, tokens[1]))
		nt.AddChild(parsetree.NewTerminal(parsetree.KindColumnName, tokens[2]))
		return nt
	}

	nt.AddChild(parsetree.NewTerminal(parsetree.KindSchemaName, tokens[0]))
	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, tokens[1]))
	nt.AddChild(parsetree.NewTerminal(parsetree.KindTableName, tokens[2]))
	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, tokens[3]))
	nt.AddChild(parsetree.NewTerminal(parsetree.KindColumnName, tokens[4]))

	return nt
}

// functionCall parses a function call.
func (p *Parser) functionCall() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindFunctionCall)

	nt.AddChild(parsetree.NewTerminal(parsetree.KindFunctionName, p.tok[0]))
	p.advance()

	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	p.advance()

	if p.tok[0].Kind == token.KindDistinct || p.isExpressionStart(p.tok[0]) || p.tok[0].Kind == token.KindAsterisk {
		nt.AddChild(p.functionArguments())
	}

	if p.tok[0].Kind == token.KindRightParen {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing ")"`)))
	}

	if p.tok[0].Kind == token.KindFilter {
		nt.AddChild(p.filterClause())
	}

	if p.tok[0].Kind == token.KindOver {
		nt.AddChild(p.overClause())
	}

	return nt
}

// functionCall parses the arguments of a function call.
func (p *Parser) functionArguments() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindFunctionArguments)

	if p.tok[0].Kind == token.KindAsterisk {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
		return nt
	}

	if p.tok[0].Kind == token.KindDistinct {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	}

	if p.isExpressionStart(p.tok[0]) {
		nt.AddChild(p.commaList("argument", p.expression, p.isExpressionStart, func(t *token.Token) bool {
			return t.Kind == token.KindOrder || t.Kind == token.KindRightParen || t.Kind == token.KindSemicolon
		}))
	} else {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing argument`)))
	}

	if p.tok[0].Kind == token.KindOrder {
		nt.AddChild(p.orderBy(func(t *token.Token) bool {
			return t.Kind == token.KindRightParen || t.Kind == token.KindSemicolon
		}))
	}

	return nt
}

// orderBy parses an order by clause.
func (p *Parser) orderBy(isInFollowSet func(*token.Token) bool) parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindOrderBy)
	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	p.advance()

	if p.tok[0].Kind == token.KindBy {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else if p.isExpressionStart(p.tok[0]) || isInFollowSet(p.tok[0]) {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "BY"`)))
	}

	if p.isExpressionStart(p.tok[0]) {
		nt.AddChild(p.commaList("ordering term", p.orderingTerm, p.isExpressionStart, isInFollowSet))
	} else {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing ordering term`)))
	}

	return nt
}

// orderingTerm parses an ordering term.
func (p *Parser) orderingTerm() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindOrderingTerm)
	nt.AddChild(p.expression())

	if p.tok[0].Kind == token.KindAsc || p.tok[0].Kind == token.KindDesc {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	}

	if p.tok[0].Kind == token.KindNulls {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		if p.tok[0].Kind == token.KindFirst || p.tok[0].Kind == token.KindLast {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		} else {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorExpecting, errors.New(`expecting "FIRST", or "LAST"`)))
		}
	}

	return nt
}

// filterClause parses a filter clause.
func (p *Parser) filterClause() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindFilterClause)
	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	p.advance()

	if p.tok[0].Kind == token.KindLeftParen {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else if p.tok[0].Kind == token.KindWhere {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "("`)))
	}

	if p.tok[0].Kind == token.KindWhere {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else if p.isExpressionStart(p.tok[0]) {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "WHERE"`)))
	}

	if p.isExpressionStart(p.tok[0]) {
		nt.AddChild(p.expression())
	} else {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing expression`)))
	}

	if p.tok[0].Kind == token.KindRightParen {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing ")"`)))
	}

	return nt
}

// overClause parses a over clause.
func (p *Parser) overClause() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindOverClause)
	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	p.advance()

	if p.tok[0].Kind == token.KindIdentifier {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindWindowName, p.tok[0]))
		p.advance()
		return nt
	}

	if p.tok[0].Kind == token.KindLeftParen {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorExpecting, errors.New(`expecting window name, or "("`)))
		return nt
	}

	if p.tok[0].Kind == token.KindIdentifier {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindWindowName, p.tok[0]))
		p.advance()
	}

	if p.tok[0].Kind == token.KindPartition {
		pb := parsetree.NewNonTerminal(parsetree.KindPartitionBy)
		pb.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		if p.tok[0].Kind == token.KindBy {
			pb.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		} else if p.isExpressionStart(p.tok[0]) {
			pb.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "BY"`)))
		}

		followSet := []token.Kind{
			token.KindOrder, token.KindRange, token.KindRows, token.KindGroups, token.KindRightParen, token.KindSemicolon,
		}
		if p.isExpressionStart(p.tok[0]) {
			pb.AddChild(p.commaList("expression", p.expression, p.isExpressionStart, func(t *token.Token) bool {
				return slices.Contains(followSet, t.Kind)
			}))
		} else if slices.Contains(followSet, p.tok[0].Kind) {
			pb.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing expression`)))
		}

		nt.AddChild(pb)
	}

	if p.tok[0].Kind == token.KindOrder {
		followSet := []token.Kind{token.KindRange, token.KindRows, token.KindGroups, token.KindRightParen, token.KindSemicolon}
		nt.AddChild(p.orderBy(func(t *token.Token) bool { return slices.Contains(followSet, t.Kind) }))
	}

	if p.tok[0].Kind == token.KindRange || p.tok[0].Kind == token.KindRows || p.tok[0].Kind == token.KindGroups {
		nt.AddChild(p.frameSpec())
	}

	if p.tok[0].Kind == token.KindRightParen {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing ")"`)))
	}

	return nt
}

// frameSpec parses a frame spec clause.
func (p *Parser) frameSpec() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindFrameSpec)
	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	p.advance()

	if p.tok[0].Kind == token.KindBetween {
		nt.AddChild(p.frameSpecBetween())
	} else if p.tok[0].Kind == token.KindUnbounded {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		if p.tok[0].Kind == token.KindPreceding {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		} else {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "PRECEDING"`)))
		}
	} else if p.tok[0].Kind == token.KindCurrent {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		if p.tok[0].Kind == token.KindRow {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		} else {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "ROW"`)))
		}
	} else if p.isExpressionStart(p.tok[0]) {
		nt.AddChild(p.expression())

		if p.tok[0].Kind == token.KindPreceding {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		} else {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "PRECEDING"`)))
		}
	} else {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorExpecting, errors.New(`expecting "BETWEEN", "UNBOUNDED", "CURRENT", or an expression`)))
	}

	if p.tok[0].Kind == token.KindExclude {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		if p.tok[0].Kind == token.KindNo {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()

			if p.tok[0].Kind == token.KindOthers {
				nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
				p.advance()
			} else {
				nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "OTHERS"`)))
			}
		} else if p.tok[0].Kind == token.KindCurrent {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()

			if p.tok[0].Kind == token.KindRow {
				nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
				p.advance()
			} else {
				nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "ROW"`)))
			}
		} else if p.tok[0].Kind == token.KindGroup {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		} else if p.tok[0].Kind == token.KindTies {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		} else {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorExpecting, errors.New(`expecting "NO", "CURRENT", "GROUP", or "TIES"`)))
		}
	}

	return nt
}

// frameSpecBetween parses the between part of a frame spec .
func (p *Parser) frameSpecBetween() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindFrameSpecBetween)
	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	p.advance()

	if p.tok[0].Kind == token.KindUnbounded {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		if p.tok[0].Kind == token.KindPreceding {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		} else if p.tok[0].Kind == token.KindAnd {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "PRECEDING"`)))
		} else {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "PRECEDING"`)))
			return nt
		}
	} else if p.tok[0].Kind == token.KindCurrent {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		if p.tok[0].Kind == token.KindRow {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		} else if p.tok[0].Kind == token.KindAnd {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "ROW"`)))
		} else {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "ROW"`)))
			return nt
		}
	} else if p.isExpressionAtLeast4Start(p.tok[0]) {
		exp := parsetree.NewNonTerminal(parsetree.KindExpression)
		exp.AddChild(p.expression4())

		nt.AddChild(exp)

		if p.tok[0].Kind == token.KindPreceding || p.tok[0].Kind == token.KindFollowing {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		} else if p.tok[0].Kind == token.KindAnd {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorExpecting, errors.New(`expecting "PRECEDING", or "FOLLOWING"`)))
		} else {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorExpecting, errors.New(`expecting "PRECEDING", or "FOLLOWING"`)))
			return nt
		}
	} else if p.tok[0].Kind == token.KindPreceding {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorExpecting, errors.New(`expecting "UNBOUNDED", or an expression`)))
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else if p.tok[0].Kind == token.KindRow {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "CURRENT"`)))
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else if p.tok[0].Kind == token.KindFollowing {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing expression`)))
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else if p.tok[0].Kind == token.KindAnd {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorExpecting, errors.New(`expecting "UNBOUNDED", "CURRENT", or a expression`)))
	} else {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorExpecting, errors.New(`expecting "UNBOUNDED", "CURRENT", or a expression`)))
		return nt
	}

	if p.tok[0].Kind == token.KindAnd {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else if p.tok[0].Kind == token.KindUnbounded || p.tok[0].Kind == token.KindCurrent || p.isExpressionStart(p.tok[0]) {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "AND"`)))
	}

	if p.tok[0].Kind == token.KindUnbounded {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		if p.tok[0].Kind == token.KindFollowing {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		} else {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "FOLLOWING"`)))
		}
	} else if p.tok[0].Kind == token.KindCurrent {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		if p.tok[0].Kind == token.KindRow {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		} else {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "ROW"`)))
		}
	} else if p.isExpressionAtLeast4Start(p.tok[0]) {
		exp := parsetree.NewNonTerminal(parsetree.KindExpression)
		exp.AddChild(p.expression4())

		nt.AddChild(exp)

		if p.tok[0].Kind == token.KindPreceding || p.tok[0].Kind == token.KindFollowing {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		} else {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorExpecting, errors.New(`expecting "PRECEDING", or "FOLLOWING"`)))
		}
	} else {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorExpecting, errors.New(`expecting "UNBOUNDED", "CURRENT", or an expression`)))
	}

	return nt
}

// parenExpression parses a expression enclosed in parenthesis.
func (p *Parser) parenExpression() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindParenExpression)
	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	p.advance()

	if p.isExpressionStart(p.tok[0]) || p.tok[0].Kind == token.KindComma {
		nt.AddChild(p.commaList("expression", p.expression, p.isExpressionStart, func(t *token.Token) bool {
			return t.Kind == token.KindRightParen || t.Kind == token.KindSemicolon
		}))
	} else if p.tok[0].Kind == token.KindRightParen {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing expression`)))
	}

	if p.tok[0].Kind == token.KindRightParen {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing ")"`)))
	}
	return nt
}

// castExpression parses a cast expression.
func (p *Parser) castExpression() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindCast)
	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	p.advance()

	if p.tok[0].Kind == token.KindLeftParen {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "("`)))
	}

	if p.isExpressionStart(p.tok[0]) {
		nt.AddChild(p.expression())
	} else {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing expression`)))
	}

	if p.tok[0].Kind == token.KindAs {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "AS"`)))
	}

	if p.tok[0].Kind == token.KindIdentifier {
		nt.AddChild(p.typeName())
	} else {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing type name`)))
	}

	if p.tok[0].Kind == token.KindRightParen {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing ")"`)))
	}

	return nt
}

// exists parses a exists expression.
func (p *Parser) exists() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindExists)
	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	p.advance()

	if p.tok[0].Kind == token.KindLeftParen {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else if p.tok[0].Kind == token.KindSelect {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "("`)))
	}

	if p.tok[0].Kind == token.KindSelect {
		nt.AddChild(p.selectStatement())
	} else if p.tok[0].Kind == token.KindRightParen {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing select statement`)))
	} else {
		p.skipTo(nt, token.KindRightParen, token.KindSemicolon)
		if p.tok[0].Kind == token.KindSemicolon {
			return nt
		}
	}

	if p.tok[0].Kind == token.KindRightParen {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing ")"`)))
	}

	return nt
}

// caseExpression parses a case expression.
func (p *Parser) caseExpression() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindCase)
	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	p.advance()

	if p.isExpressionStart(p.tok[0]) {
		nt.AddChild(p.expression())
	}

	p.skipTo(nt, token.KindWhen, token.KindElse, token.KindEnd, token.KindSemicolon)

	if p.tok[0].Kind == token.KindWhen {
		for p.tok[0].Kind == token.KindWhen {
			nt.AddChild(p.when())
		}
		p.skipTo(nt, token.KindWhen, token.KindElse, token.KindEnd, token.KindSemicolon)
	} else if p.tok[0].Kind == token.KindElse {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing when clause`)))
	}

	p.skipTo(nt, token.KindElse, token.KindEnd, token.KindSemicolon)

	if p.tok[0].Kind == token.KindElse {
		nt.AddChild(p.caseElse())
	}

	if p.tok[0].Kind == token.KindEnd {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "END"`)))
	}

	return nt
}

// when parses a when part of a case expression.
func (p *Parser) when() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindWhen)
	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	p.advance()

	if p.isExpressionStart(p.tok[0]) {
		nt.AddChild(p.expression())
	} else if p.tok[0].Kind == token.KindThen {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing expression`)))
	}

	if p.tok[0].Kind == token.KindThen {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else if p.isExpressionStart(p.tok[0]) {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "THEN"`)))
	}

	if p.isExpressionStart(p.tok[0]) {
		nt.AddChild(p.expression())
	} else {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing expression`)))
	}

	return nt
}

// caseElse parses an else part of a case expression.
func (p *Parser) caseElse() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindElse)
	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	p.advance()

	if p.isExpressionStart(p.tok[0]) {
		nt.AddChild(p.expression())
	} else {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing expression`)))
	}

	return nt
}

// raise parses a raise function call.
func (p *Parser) raise() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindRaise)
	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	p.advance()

	if p.tok[0].Kind == token.KindLeftParen {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else if p.tok[0].Kind == token.KindIgnore || p.tok[0].Kind == token.KindRollback || p.tok[0].Kind == token.KindAbort ||
		p.tok[0].Kind == token.KindFail {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "("`)))
	} else {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "("`)))
		p.skipTo(nt, token.KindIgnore, token.KindRollback, token.KindAbort, token.KindFail,
			token.KindRightParen, token.KindSemicolon)
		if p.tok[0].Kind == token.KindSemicolon {
			return nt
		}
	}

	if p.tok[0].Kind == token.KindIgnore {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		if p.tok[0].Kind == token.KindRightParen {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		} else {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing ")"`)))
		}
		return nt
	}

	if p.tok[0].Kind == token.KindRollback || p.tok[0].Kind == token.KindAbort || p.tok[0].Kind == token.KindFail {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else if p.tok[0].Kind == token.KindComma {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorExpecting, errors.New(`expecting "IGNORE", "ROLLBACK", "ABORT", or "FAIL"`)))
	} else if p.tok[0].Kind == token.KindRightParen {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorExpecting, errors.New(`expecting "IGNORE", "ROLLBACK", "ABORT", or "FAIL"`)))
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
		return nt
	}

	if p.tok[0].Kind == token.KindComma {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else if p.isExpressionStart(p.tok[0]) {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing ","`)))
	}

	if p.isExpressionStart(p.tok[0]) {
		em := parsetree.NewNonTerminal(parsetree.KindErrorMessage)
		em.AddChild(p.expression())
		nt.AddChild(em)
	} else if p.tok[0].Kind == token.KindRightParen {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing error message`)))
	}

	if p.tok[0].Kind == token.KindRightParen {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing ")"`)))
	}

	return nt
}

// commaListConstruction parses a list of items separated by commas. itemMeaning is the meaning of the item, item is the function
// that parses a item, isItemStart is for the tokens that starts a item, isInFollowSet is for the tokens that follow the list.
func (p *Parser) commaListConstruction(itemMeaning string, item func() parsetree.Construction, isItemStart, isInFollowSet func(*token.Token) bool) parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindCommaList)

	var skipped bool

	skipPred := func(t *token.Token) bool {
		return isItemStart(t) || isInFollowSet(t) || t.Kind == token.KindComma
	}

	if isItemStart(p.tok[0]) {
		nt.AddChild(item())
	} else if p.tok[0].Kind == token.KindComma {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing `+itemMeaning)))
	}

	for !isInFollowSet(p.tok[0]) {
		if p.tok[0].Kind == token.KindComma {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
			skipped = false
		} else if isItemStart(p.tok[0]) {
			if !skipped {
				nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing comma`)))
			}
			skipped = false
		} else {
			p.skipToFunc(nt, skipPred)
			skipped = true
		}

		if isItemStart(p.tok[0]) {
			nt.AddChild(item())
			skipped = false
		} else if p.tok[0].Kind == token.KindComma || isInFollowSet(p.tok[0]) {
			if !skipped {
				nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing `+itemMeaning)))
			}
			skipped = false
		} else {
			p.skipToFunc(nt, skipPred)
			skipped = true
		}
	}

	return nt
}

// commaList parses a list of items separated by commas. itemMeaning is the meaning of the item, item is the function
// that parses a item, isItemStart is for the tokens that starts a item, isInFollowSet is for the tokens that follow the list.
func (p *Parser) commaList(itemMeaning string, item func() parsetree.NonTerminal, isItemStart, isInFollowSet func(*token.Token) bool) parsetree.NonTerminal {
	return p.commaListConstruction(itemMeaning, func() parsetree.Construction { return item() }, isItemStart, isInFollowSet)
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
			p.tok[1] = p.tok[2]
			p.tok[2] = tok
			break
		}
	}
}

// skipTo do nothing if set contains the kind of p.tok[0], or put a parse tree with kind
// unexpected EOF in father if the kind of p.tok[0] is EOF, or advances until set contains
// the kind of p.tok[0] or the EOF is found. In the later case the skipped tokens, if any,
// are put in father. The return is true if, possibly after some advances, set contains the
// kind of token in p.tok[0]. The set must not contain token.KindEof.
func (p *Parser) skipTo(father parsetree.NonTerminal, set ...token.Kind) bool {
	return p.skipToFunc(father, func(t *token.Token) bool { return slices.Contains(set, t.Kind) })
}

// skipTo do nothing if predicate(p.tok[0]) return true, or put a parse tree with kind
// unexpected EOF in father if the kind of p.tok[0] is EOF, or advances until predicate(p.tok[0])
// return true or the EOF is found. In the later case the skipped tokens, if any,
// are put in father. The return is true if, possibly after some advances, predicate(p.tok[0]) returns true.
// The set must not contain token.KindEof.
func (p *Parser) skipToFunc(father parsetree.NonTerminal, predicate func(*token.Token) bool) bool {
	if predicate(p.tok[0]) {
		return true
	}
	if p.tok[0].Kind == token.KindEOF {
		father.AddChild(parsetree.NewError(parsetree.KindErrorUnexpectedEOF, errors.New(`unexpected EOF`)))
		return false
	}
	skipped := parsetree.NewNonTerminal(parsetree.KindSkipped)
	for !predicate(p.tok[0]) && p.tok[0].Kind != token.KindEOF {
		skipped.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	}
	father.AddChild(skipped)
	return p.tok[0].Kind != token.KindEOF
}
