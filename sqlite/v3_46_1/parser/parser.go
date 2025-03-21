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
	case token.KindAnalyze:
		father.AddChild(p.analyze())
	case token.KindAttach:
		father.AddChild(p.attach())
	case token.KindBegin:
		father.AddChild(p.begin())
	case token.KindCommit, token.KindEnd:
		father.AddChild(p.commit())
	case token.KindRollback:
		father.AddChild(p.rollback())
	case token.KindCreate:
		if p.tok[1].Kind == token.KindIndex || p.tok[1].Kind == token.KindUnique {
			father.AddChild(p.createIndex())
		} else if p.tok[1].Kind == token.KindTable {
			father.AddChild(p.createTable())
		} else if p.tok[1].Kind == token.KindTrigger {
			father.AddChild(p.createTrigger())
		} else if p.tok[1].Kind == token.KindView {
			father.AddChild(p.createView())
		} else if p.tok[1].Kind == token.KindTemp || p.tok[1].Kind == token.KindTemporary {
			if p.tok[2].Kind == token.KindTable {
				father.AddChild(p.createTable())
			} else if p.tok[2].Kind == token.KindTrigger {
				father.AddChild(p.createTrigger())
			} else if p.tok[2].Kind == token.KindView {
				father.AddChild(p.createView())
			}
		} else if p.tok[1].Kind == token.KindVirtual {
			father.AddChild(p.createVirtualTable())
		}
	case token.KindDelete:
		father.AddChild(p.delete(nil))
	case token.KindDetach:
		father.AddChild(p.detach())
	case token.KindDrop:
		if p.tok[1].Kind == token.KindIndex {
			father.AddChild(p.dropIndex())
		} else if p.tok[1].Kind == token.KindTable {
			father.AddChild(p.dropTable())
		} else {
			father.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
			// TODO: Update this list when other DROP constructs have parse methods.
			father.AddChild(parsetree.NewError(parsetree.KindErrorExpecting, errors.New(`expecting "INDEX", or "TABLE"`)))
		}
	case token.KindSelect:
		father.AddChild(p.selectStatement())
	case token.KindWith:
		with := p.with()
		switch p.tok[0].Kind {
		case token.KindDelete:
			father.AddChild(p.delete(with))
		default:
			father.AddChild(with)
			father.AddChild(parsetree.NewError(parsetree.KindErrorExpecting, errors.New(`expecting "DELETE", "INSERT", "SELECT", or "UPDATE"`)))
		}
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
		} else {
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

	p.skipTo(nt, token.KindNumeric, token.KindPlus, token.KindMinus, token.KindRightParen, token.KindSemicolon)
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

	switch p.tok[0].Kind {
	case token.KindPrimary:
		nt.AddChild(p.primaryKeyColumnConstraint())
	case token.KindNot:
		nt.AddChild(p.notNullColumnConstraint())
	case token.KindUnique:
		nt.AddChild(p.uniqueColumnConstraint())
	case token.KindCheck:
		nt.AddChild(p.checkColumnConstraint())
	case token.KindDefault:
		nt.AddChild(p.defaultColumnConstraint())
	case token.KindCollate:
		nt.AddChild(p.collateColumnConstraint())
	case token.KindReferences:
		nt.AddChild(p.foreignKeyColumnConstraint())
	case token.KindGenerated, token.KindAs:
		nt.AddChild(p.generatedColumnConstraint())
	default:
		nt.AddChild(parsetree.NewError(
			parsetree.KindErrorExpecting,
			errors.New(`expecting "PRIMARY", "NOT", "UNIQUE", "CHECK", "DEFAULT", "COLLATE", "REFERENCES", "GENERATED", or "AS"`)))
	}

	return nt
}

// primaryKeyColumnConstraint parses a primary key column constraint clause.
func (p *Parser) primaryKeyColumnConstraint() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindPrimaryKeyColumnConstraint)
	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	p.advance()

	if p.tok[0].Kind == token.KindKey {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else {
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

	return nt
}

// notNullColumnConstraint parses a not null column constraint clause.
func (p *Parser) notNullColumnConstraint() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindNotNullColumnConstraint)
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
	return nt
}

// uniqueColumnConstraint parses a unique column constraint clause.
func (p *Parser) uniqueColumnConstraint() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindUniqueColumnConstraint)
	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	p.advance()

	if p.tok[0].Kind == token.KindOn {
		nt.AddChild(p.conflictClause())
	}

	return nt
}

// checkColumnConstraint parses a check column constraint clause.
func (p *Parser) checkColumnConstraint() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindCheckColumnConstraint)
	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	p.advance()

	if p.tok[0].Kind == token.KindLeftParen {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else if p.isStartOfExpression(p.tok[0]) {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "("`)))
	}

	if p.isStartOfExpression(p.tok[0]) {
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

	return nt
}

// defaultColumnConstraint parses a default column constraint clause.
func (p *Parser) defaultColumnConstraint() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindDefaultColumnConstraint)
	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	p.advance()

	if p.tok[0].Kind == token.KindLeftParen {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		if p.isStartOfExpression(p.tok[0]) {
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
	return nt
}

// collateColumnConstraint parses a collate column constraint clause.
func (p *Parser) collateColumnConstraint() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindCollateColumnConstraint)
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

// generatedColumnConstraint parses a generated column constraint clause.
func (p *Parser) generatedColumnConstraint() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindGeneratedColumnConstraint)
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
	} else if p.isStartOfExpression(p.tok[0]) {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "("`)))
	}

	if p.isStartOfExpression(p.tok[0]) {
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

	if p.tok[0].Kind == token.KindStored || p.tok[0].Kind == token.KindVirtual {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	}

	return nt
}

// foreignKeyColumnConstraint parses a foreignKey column constraint clause.
func (p *Parser) foreignKeyColumnConstraint() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindForeignKeyColumnConstraint)
	nt.AddChild(p.foreignKeyClause())
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

	if p.tok[0].Kind == token.KindLeftParen {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		nt.AddChild(p.columnNameList(
			token.KindRightParen, token.KindSemicolon, token.KindEOF, token.KindOn, token.KindMatch, token.KindDeferrable, token.KindNot))

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
				nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing name`)))
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
				nt.AddChild(parsetree.NewError(parsetree.KindErrorExpecting, errors.New(`expecting "DEFERRED", or "IMMEDIATE"`)))
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
		[]token.Kind{token.KindRollback, token.KindAbort, token.KindFail, token.KindIgnore, token.KindReplace}, p.tok[0].Kind,
	) {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "CONFLICT"`)))
	}

	if slices.Contains(
		[]token.Kind{token.KindRollback, token.KindAbort, token.KindFail, token.KindIgnore, token.KindReplace}, p.tok[0].Kind,
	) {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorExpecting, errors.New(`expecting "ROLLBACK", "ABORT", "FAIL", "IGNORE", or "REPLACE"`)))
	}

	return nt
}

// analyze parses a analyze statement.
func (p *Parser) analyze() parsetree.NonTerminal {
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

	if p.isStartOfExpression(p.tok[0]) {
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

// commit parses a commit statement.
func (p *Parser) commit() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindCommit)
	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	p.advance()

	if p.tok[0].Kind == token.KindTransaction {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	}

	return nt
}

// rollback parses a rollback statement.
func (p *Parser) rollback() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindRollback)
	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	p.advance()

	if p.tok[0].Kind == token.KindTransaction {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	}

	if p.tok[0].Kind == token.KindTo {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		if p.tok[0].Kind == token.KindSavepoint {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		}

		if p.tok[0].Kind == token.KindIdentifier {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindSavepointName, p.tok[0]))
			p.advance()
		} else {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing savepoint name`)))
		}
	}

	return nt
}

// createIndex parses a create index statement.
func (p *Parser) createIndex() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindCreateIndex)
	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	p.advance()

	if p.tok[0].Kind == token.KindUnique {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	}

	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	p.advance()

	if p.tok[0].Kind == token.KindIf {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		if p.tok[0].Kind == token.KindNot {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		} else if p.tok[0].Kind == token.KindExists {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "NOT"`)))
		}

		if p.tok[0].Kind == token.KindExists {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		} else {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "EXISTS"`)))
		}
	}

	if p.tok[0].Kind == token.KindIdentifier && p.tok[1].Kind == token.KindDot {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindSchemaName, p.tok[0]))
		p.advance()
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else if p.tok[0].Kind == token.KindDot {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing schema name`)))
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	}

	if p.tok[0].Kind == token.KindIdentifier {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindIndexName, p.tok[0]))
		p.advance()
	} else if p.tok[0].Kind == token.KindOn {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing index name`)))
	}

	if p.tok[0].Kind == token.KindOn {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else if p.tok[0].Kind == token.KindIdentifier {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "ON"`)))
	}

	if p.tok[0].Kind == token.KindIdentifier {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindTableName, p.tok[0]))
		p.advance()
	} else if p.tok[0].Kind == token.KindLeftParen {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing table name`)))
	}

	if p.tok[0].Kind == token.KindLeftParen {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else if p.isStartOfExpression(p.tok[0]) {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "("`)))
	}

	nt.AddChild(commaListFunc(p, "indexed column",
		func(p *Parser) parsetree.NonTerminal { return p.indexedColumn(true) },
		p.isStartOfExpression, func(t *token.Token) bool {
			return t.Kind == token.KindRightParen || t.Kind == token.KindSemicolon ||
				t.Kind == token.KindEOF
		}))

	if p.tok[0].Kind == token.KindRightParen {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing ")"`)))
	}

	if p.tok[0].Kind == token.KindWhere {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		if p.isStartOfExpression(p.tok[0]) {
			nt.AddChild(p.expression())
		} else {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing expression`)))
		}
	}

	return nt
}

// indexedColumn parses a indexed column.
func (p *Parser) indexedColumn(canBeExpression bool) parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindIndexedColumn)
	if canBeExpression {
		// the collate part is handled by the expression parser
		nt.AddChild(p.expression())
	} else {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindColumnName, p.tok[0]))
		p.advance()

		if p.tok[0].Kind == token.KindCollate {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()

			if p.tok[0].Kind == token.KindIdentifier {
				nt.AddChild(parsetree.NewTerminal(parsetree.KindCollationName, p.tok[0]))
				p.advance()
			} else {
				nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing collation name`)))
			}
		}
	}

	if p.tok[0].Kind == token.KindAsc || p.tok[0].Kind == token.KindDesc {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	}

	return nt
}

// createTable parses a create table statement.
func (p *Parser) createTable() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindCreateTable)
	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	p.advance()

	if p.tok[0].Kind == token.KindTemp || p.tok[0].Kind == token.KindTemporary {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	}

	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	p.advance()

	if p.tok[0].Kind == token.KindIf {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		if p.tok[0].Kind == token.KindNot {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		} else if p.tok[0].Kind == token.KindExists {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "NOT"`)))
		}

		if p.tok[0].Kind == token.KindExists {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		} else {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "EXISTS"`)))
		}
	}

	if (p.tok[0].Kind == token.KindIdentifier || p.tok[0].Kind == token.KindTemp) && p.tok[1].Kind == token.KindDot {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindSchemaName, p.tok[0]))
		p.advance()
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else if p.tok[0].Kind == token.KindDot {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing schema name`)))
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	}

	if p.tok[0].Kind == token.KindIdentifier {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindTableName, p.tok[0]))
		p.advance()
	} else if p.tok[0].Kind == token.KindLeftParen || p.tok[0].Kind == token.KindAs {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing table name`)))
	}

	if p.tok[0].Kind == token.KindLeftParen {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		nt.AddChild(p.columnDefinitionList())

		if p.tok[0].Kind == token.KindComma {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()

			nt.AddChild(p.tableConstraintList())
		}

		if p.tok[0].Kind == token.KindRightParen {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		} else {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing ")"`)))
		}

		if p.tok[0].Kind == token.KindWithout || p.tok[0].Kind == token.KindStrict {
			nt.AddChild(p.tableOptions())
		}
	} else if p.tok[0].Kind == token.KindAs {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		if p.tok[0].Kind == token.KindSelect {
			nt.AddChild(p.selectStatement())
		} else {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing select statement`)))
		}
	} else {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorExpecting, errors.New(`expecting "(", or "AS"`)))
	}

	return nt
}

// columnDefinitionList parses a list of column definitions in a create table. We use this function instead of
// commaList because after a comma can come a column definition or a table constraint.
func (p *Parser) columnDefinitionList() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindCommaList)

	if p.tok[0].Kind == token.KindIdentifier {
		nt.AddChild(p.columnDefinition())
	} else if p.tok[0].Kind == token.KindComma {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing column definition`)))
	}

	for {
		if p.tok[0].Kind == token.KindComma && p.tok[1].Kind == token.KindIdentifier {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
			nt.AddChild(p.columnDefinition())
		} else if p.tok[0].Kind == token.KindComma && p.tok[1].Kind == token.KindComma {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing column definition`)))
		} else {
			break
		}
	}

	return nt
}

// tableConstraintList parses a list of tale constraints in a create table.
func (p *Parser) tableConstraintList() parsetree.NonTerminal {
	return commaList(p, "table constraint", (*Parser).tableConstraint,
		[]token.Kind{token.KindConstraint, token.KindPrimary, token.KindUnique, token.KindCheck, token.KindForeign},
		[]token.Kind{token.KindRightParen, token.KindSemicolon, token.KindEOF},
	)
}

// tableConstraint parses a table constraint clause.
func (p *Parser) tableConstraint() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindTableConstraint)
	if p.tok[0].Kind == token.KindConstraint {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		if p.tok[0].Kind == token.KindIdentifier {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindConstraintName, p.tok[0]))
			p.advance()
		} else {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing constraint name`)))
		}
	}

	if p.tok[0].Kind == token.KindPrimary {
		nt.AddChild(p.primaryKeyTableConstraint())
	} else if p.tok[0].Kind == token.KindUnique {
		nt.AddChild(p.uniqueTableConstraint())
	} else if p.tok[0].Kind == token.KindCheck {
		nt.AddChild(p.checkTableConstraint())
	} else if p.tok[0].Kind == token.KindForeign {
		nt.AddChild(p.foreignKeyTableConstraint())
	} else {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorExpecting, errors.New(`expecting "PRIMARY", "UNIQUE", "CHECK", or "FOREIGN"`)))
	}

	return nt
}

// primaryKeyTableConstraint parses a primary key table constraint clause.
func (p *Parser) primaryKeyTableConstraint() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindPrimaryKeyTableConstraint)

	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	p.advance()

	if p.tok[0].Kind == token.KindKey {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else if p.tok[0].Kind == token.KindLeftParen {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "KEY"`)))
	}

	if p.tok[0].Kind == token.KindLeftParen {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else if p.tok[0].Kind == token.KindIdentifier {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "("`)))
	}

	nt.AddChild(commaListFunc(p, "indexed column",
		func(p *Parser) parsetree.NonTerminal { return p.indexedColumn(false) },
		p.isStartOfExpression, func(t *token.Token) bool {
			return t.Kind == token.KindRightParen || t.Kind == token.KindSemicolon ||
				t.Kind == token.KindEOF
		}))

	if p.tok[0].Kind == token.KindRightParen {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing ")"`)))
	}

	if p.tok[0].Kind == token.KindOn {
		nt.AddChild(p.conflictClause())
	}

	return nt
}

// uniqueTableConstraint parses a unique table constraint clause.
func (p *Parser) uniqueTableConstraint() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindUniqueTableConstraint)

	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	p.advance()

	if p.tok[0].Kind == token.KindLeftParen {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else if p.tok[0].Kind == token.KindIdentifier {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "("`)))
	}

	nt.AddChild(commaListFunc(p, "indexed column",
		func(p *Parser) parsetree.NonTerminal { return p.indexedColumn(false) },
		p.isStartOfExpression, func(t *token.Token) bool {
			return t.Kind == token.KindRightParen || t.Kind == token.KindSemicolon ||
				t.Kind == token.KindEOF
		}))

	if p.tok[0].Kind == token.KindRightParen {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing ")"`)))
	}

	if p.tok[0].Kind == token.KindOn {
		nt.AddChild(p.conflictClause())
	}

	return nt
}

// checkTableConstraint parses a check table constraint clause.
func (p *Parser) checkTableConstraint() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindCheckTableConstraint)
	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	p.advance()

	if p.tok[0].Kind == token.KindLeftParen {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else if p.isStartOfExpression(p.tok[0]) {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "("`)))
	}

	if p.isStartOfExpression(p.tok[0]) {
		nt.AddChild(p.expression())
	} else if p.tok[0].Kind == token.KindRightParen || p.tok[0].Kind == token.KindSemicolon || p.tok[0].Kind == token.KindEOF {
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

// foreignKeyTableConstraint parses a foreign key table constraint clause.
func (p *Parser) foreignKeyTableConstraint() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindForeignKeyTableConstraint)
	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	p.advance()

	if p.tok[0].Kind == token.KindKey {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else if p.tok[0].Kind == token.KindLeftParen {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "KEY"`)))
	}

	if p.tok[0].Kind == token.KindLeftParen {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else if p.isStartOfExpression(p.tok[0]) {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "("`)))
	}

	if p.tok[0].Kind == token.KindIdentifier {
		nt.AddChild(p.columnNameList(token.KindRightParen, token.KindSemicolon, token.KindEOF, token.KindReferences))
	} else if p.tok[0].Kind == token.KindRightParen || p.tok[0].Kind == token.KindSemicolon || p.tok[0].Kind == token.KindEOF {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing column name`)))
	}

	if p.tok[0].Kind == token.KindRightParen {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing ")"`)))
	}

	if p.tok[0].Kind == token.KindReferences {
		nt.AddChild(p.foreignKeyClause())
	} else {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing foreign key clause`)))
	}

	return nt
}

// tableOptions parses a table options clause.
func (p *Parser) tableOptions() parsetree.NonTerminal {
	return commaList(p, "table option", (*Parser).tableOption,
		[]token.Kind{token.KindWithout, token.KindStrict},
		[]token.Kind{token.KindSemicolon, token.KindEOF},
	)
}

// tableOption parses a table option.
func (p *Parser) tableOption() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindTableOption)
	if p.tok[0].Kind == token.KindWithout {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		if p.tok[0].Kind == token.KindRowId {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		} else {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "ROWID"`)))
		}
	} else { // STRICT
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	}

	return nt
}

// columnNameList parses a list of column names. follow is the set of kinds of token that
// follow the list.
func (p *Parser) columnNameList(follow ...token.Kind) parsetree.NonTerminal {
	colNameBuilder := func(p *Parser) parsetree.Terminal {
		t := parsetree.NewTerminal(parsetree.KindColumnName, p.tok[0])
		p.advance()
		return t
	}

	return commaList(p, "column name", colNameBuilder,
		[]token.Kind{token.KindIdentifier},
		follow,
	)
}

// createTrigger parses a create trigger statement.
func (p *Parser) createTrigger() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindCreateTrigger)
	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	p.advance()

	if p.tok[0].Kind == token.KindTemp || p.tok[0].Kind == token.KindTemporary {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	}

	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	p.advance()

	if p.tok[0].Kind == token.KindIf {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		if p.tok[0].Kind == token.KindNot {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		} else if p.tok[0].Kind == token.KindExists {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "NOT"`)))
		}

		if p.tok[0].Kind == token.KindExists {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		} else {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "EXISTS"`)))
		}
	}

	if p.tok[0].Kind == token.KindIdentifier || p.tok[0].Kind == token.KindTemp {
		if p.tok[1].Kind == token.KindDot {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindSchemaName, p.tok[0]))
			p.advance()
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		} else if p.tok[1].Kind == token.KindIdentifier {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindSchemaName, p.tok[0]))
			p.advance()
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing dot`)))
		}
	} else if p.tok[0].Kind == token.KindDot {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing schema name`)))
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	}

	if p.tok[0].Kind == token.KindIdentifier {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindTriggerName, p.tok[0]))
		p.advance()
	} else if p.tok[0].Kind == token.KindBefore || p.tok[0].Kind == token.KindAfter || p.tok[0].Kind == token.KindInstead ||
		p.tok[0].Kind == token.KindDelete || p.tok[0].Kind == token.KindInsert || p.tok[0].Kind == token.KindUpdate {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing trigger name`)))
	}

	if p.tok[0].Kind == token.KindBefore || p.tok[0].Kind == token.KindAfter {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else if p.tok[0].Kind == token.KindInstead {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		if p.tok[0].Kind == token.KindOf {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		} else {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "OF"`)))
		}
	}

	if p.tok[0].Kind == token.KindDelete || p.tok[0].Kind == token.KindInsert {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else if p.tok[0].Kind == token.KindUpdate {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		if p.tok[0].Kind == token.KindOf {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()

			nt.AddChild(p.columnNameList(token.KindOn, token.KindSemicolon, token.KindEOF))
		}
	} else {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorExpecting, errors.New(`expecting "DELETE", "INSERT", or "UPDATE"`)))
	}

	if p.tok[0].Kind == token.KindOn {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else if p.tok[0].Kind == token.KindIdentifier {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "ON"`)))
	}

	if p.tok[0].Kind == token.KindIdentifier {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindTableName, p.tok[0]))
		p.advance()
	} else if p.tok[0].Kind == token.KindFor || p.tok[0].Kind == token.KindWhen || p.tok[0].Kind == token.KindBegin {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing table name`)))
	}

	if p.tok[0].Kind == token.KindFor {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		if p.tok[0].Kind == token.KindEach {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		} else if p.tok[0].Kind == token.KindRow {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "EACH"`)))
		}

		if p.tok[0].Kind == token.KindRow {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		} else {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "ROW"`)))
		}
	} else if p.tok[0].Kind == token.KindWhen {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		if p.isStartOfExpression(p.tok[0]) {
			nt.AddChild(p.expression())
		} else {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing expression`)))
		}
	}

	if p.tok[0].Kind == token.KindBegin {
		nt.AddChild(p.triggerBody())
	} else {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing trigger body`)))
	}

	return nt
}

// triggerBody parses the body of a create trigger statement.
func (p *Parser) triggerBody() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindTriggerBody)
	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	p.advance()

	for {
		var with parsetree.NonTerminal
		if p.tok[0].Kind == token.KindWith {
			with = p.with()
		}

		if p.tok[0].Kind == token.KindDelete {
			nt.AddChild(p.delete(with))
		} else if p.tok[0].Kind == token.KindInsert {
			nt.AddChild(p.insert())
		} else if p.tok[0].Kind == token.KindSelect {
			nt.AddChild(p.selectStatement())
		} else if p.tok[0].Kind == token.KindUpdate {
			nt.AddChild(p.update())
		} else if p.tok[0].Kind == token.KindSemicolon {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorExpecting, errors.New(`expecting "WITH", "DELETE", "INSERT", "SELECT", or "UPDATE"`)))
		} else {
			break
		}

		if p.tok[0].Kind == token.KindSemicolon {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		} else if p.tok[0].Kind == token.KindDelete || p.tok[0].Kind == token.KindInsert || p.tok[0].Kind == token.KindSelect ||
			p.tok[0].Kind == token.KindUpdate || p.tok[0].Kind == token.KindEnd {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New("missing semicolon")))
		}
	}

	if p.tok[0].Kind == token.KindEnd {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "END"`)))
	}

	return nt
}

// createView parses a create view statement.
func (p *Parser) createView() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindCreateView)
	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	p.advance()

	if p.tok[0].Kind == token.KindTemp || p.tok[0].Kind == token.KindTemporary {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	}

	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	p.advance()

	if p.tok[0].Kind == token.KindIf {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		if p.tok[0].Kind == token.KindNot {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		} else if p.tok[0].Kind == token.KindExists {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "NOT"`)))
		}

		if p.tok[0].Kind == token.KindExists {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		} else {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "EXISTS"`)))
		}
	}

	if p.tok[0].Kind == token.KindIdentifier || p.tok[0].Kind == token.KindTemp {
		if p.tok[1].Kind == token.KindDot {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindSchemaName, p.tok[0]))
			p.advance()
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		} else if p.tok[1].Kind == token.KindIdentifier {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindSchemaName, p.tok[0]))
			p.advance()
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing dot`)))
		}
	} else if p.tok[0].Kind == token.KindDot {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing schema name`)))
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	}

	if p.tok[0].Kind == token.KindIdentifier {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindViewName, p.tok[0]))
		p.advance()
	} else if p.tok[0].Kind == token.KindAs || p.tok[0].Kind == token.KindLeftParen {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing view name`)))
	}

	if p.tok[0].Kind == token.KindLeftParen {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		nt.AddChild(p.columnNameList(token.KindRightParen, token.KindSemicolon, token.KindAs, token.KindEOF))

		if p.tok[0].Kind == token.KindRightParen {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		} else {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing ")"`)))
		}
	}

	if p.tok[0].Kind == token.KindAs {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else if p.tok[0].Kind == token.KindSelect {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "AS"`)))
	}

	if p.tok[0].Kind == token.KindSelect {
		nt.AddChild(p.selectStatement())
	} else {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing select`)))
	}

	return nt
}

// createVirtualTable parses a create virtual table statement.
func (p *Parser) createVirtualTable() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindCreateVirtualTable)
	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	p.advance()

	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	p.advance()

	if p.tok[0].Kind == token.KindTable {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else if p.tok[0].Kind == token.KindIf || p.tok[0].Kind == token.KindIdentifier {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "TABLE"`)))
	}

	if p.tok[0].Kind == token.KindIf {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		if p.tok[0].Kind == token.KindNot {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		} else if p.tok[0].Kind == token.KindExists {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "NOT"`)))
		}

		if p.tok[0].Kind == token.KindExists {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		} else {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "EXISTS"`)))
		}
	}

	if p.tok[0].Kind == token.KindIdentifier || p.tok[0].Kind == token.KindTemp {
		if p.tok[1].Kind == token.KindDot {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindSchemaName, p.tok[0]))
			p.advance()
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		} else if p.tok[1].Kind == token.KindIdentifier {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindSchemaName, p.tok[0]))
			p.advance()
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing dot`)))
		}
	} else if p.tok[0].Kind == token.KindDot {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing schema name`)))
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	}

	if p.tok[0].Kind == token.KindIdentifier {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindTableName, p.tok[0]))
		p.advance()
	} else if p.tok[0].Kind == token.KindUsing {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing table name`)))
	}

	if p.tok[0].Kind == token.KindUsing {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else if p.tok[0].Kind == token.KindIdentifier {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "USING"`)))
	}

	if p.tok[0].Kind == token.KindIdentifier {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindModuleName, p.tok[0]))
		p.advance()
	} else if p.tok[0].Kind == token.KindLeftParen || p.tok[0].Kind == token.KindSemicolon || p.tok[0].Kind == token.KindEOF {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing module name`)))
	}

	if p.tok[0].Kind == token.KindLeftParen {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		nt.AddChild(p.moduleArgumentList())

		if p.tok[0].Kind == token.KindRightParen {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		} else {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing ")"`)))
		}
	}

	return nt
}

// moduleArgumentList parses a list of module arguments separated by comma.
func (p *Parser) moduleArgumentList() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindCommaList)

	if p.tok[0].Kind == token.KindRightParen {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing module argument`)))
		return nt
	}

	for {
		nt.AddChild(p.moduleArgument())

		switch p.tok[0].Kind {
		case token.KindRightParen, token.KindEOF:
			return nt
		case token.KindComma:
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		}
	}
}

// moduleArgument parses a module argument in a create virtual table statement.
func (p *Parser) moduleArgument() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindModuleArgument)
	for {
		switch p.tok[0].Kind {
		case token.KindComma, token.KindRightParen, token.KindEOF:
			return nt
		case token.KindLeftParen:
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
			p.moduleArgumentInner(nt)
			if p.tok[0].Kind == token.KindRightParen {
				nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
				p.advance()
			} else {
				return nt
			}
		default:
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		}
	}
}

// moduleArgumentInner parses a part of a module argument. The part is delimited by parenthesis.
func (p *Parser) moduleArgumentInner(nt parsetree.NonTerminal) {
	for {
		switch p.tok[0].Kind {
		case token.KindRightParen, token.KindEOF:
			return
		case token.KindLeftParen:
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
			p.moduleArgumentInner(nt)
			if p.tok[0].Kind == token.KindRightParen {
				nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
				p.advance()
			} else {
				return
			}
		default:
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		}
	}
}

// delete parses a delete statement.
func (p *Parser) delete(with parsetree.NonTerminal) parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindDelete)
	if with != nil {
		nt.AddChild(with)
	}

	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	p.advance()

	if p.tok[0].Kind == token.KindFrom {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else if p.tok[0].Kind == token.KindIdentifier {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "FROM"`)))
	}

	if p.tok[0].Kind == token.KindIdentifier {
		nt.AddChild(p.qualifiedTableName())
	} else {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing identifier`)))
	}

	if p.tok[0].Kind == token.KindWhere {
		w := parsetree.NewNonTerminal(parsetree.KindWhere)
		w.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		if p.isStartOfExpression(p.tok[0]) {
			w.AddChild(p.expression())
		} else {
			w.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing expression`)))
		}

		nt.AddChild(w)
	}

	if p.tok[0].Kind == token.KindReturning {
		nt.AddChild(p.returningClause())
	}

	return nt
}

// with parses a with clause.
func (p *Parser) with() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindWith)
	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	p.advance()

	if p.tok[0].Kind == token.KindRecursive {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else if p.tok[0].Kind != token.KindIdentifier {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorExpecting, errors.New(`expecting "RECURSIVE", or a CTE`)))
		return nt
	}

	nt.AddChild(p.commonTableExpressionList())

	return nt
}

// commonTableExpressionList parses a list of CTEs in a with clause.
func (p *Parser) commonTableExpressionList() parsetree.NonTerminal {
	return commaList(p, "CTE", (*Parser).commonTableExpression,
		[]token.Kind{token.KindIdentifier},
		// TODO: Update this list when other constructs, containing WITH, have parse methods.
		[]token.Kind{token.KindDelete, token.KindSemicolon, token.KindEOF},
	)
}

// commonTableExpression parses a common table expression.
func (p *Parser) commonTableExpression() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindCommonTableExpression)
	nt.AddChild(parsetree.NewTerminal(parsetree.KindTableName, p.tok[0]))
	p.advance()

	if p.tok[0].Kind == token.KindLeftParen {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		nt.AddChild(p.columnNameList(token.KindRightParen, token.KindAs, token.KindSemicolon, token.KindEOF))

		if p.tok[0].Kind == token.KindRightParen {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		} else {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing ")"`)))
		}
	}

	if p.tok[0].Kind == token.KindAs {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else if p.tok[0].Kind == token.KindNot || p.tok[0].Kind == token.KindMaterialized ||
		p.tok[0].Kind == token.KindLeftParen {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "AS"`)))
	}

	if p.tok[0].Kind == token.KindNot {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		if p.tok[0].Kind == token.KindMaterialized {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		} else {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "MATERIALIZED"`)))
		}
	} else if p.tok[0].Kind == token.KindMaterialized {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	}

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
	}

	if p.tok[0].Kind == token.KindRightParen {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing ")"`)))
	}

	return nt
}

// qualifiedTableName parses a qualified table name.
func (p *Parser) qualifiedTableName() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindQualifiedTableName)

	if p.tok[0].Kind == token.KindIdentifier || p.tok[0].Kind == token.KindTemp {
		if p.tok[1].Kind == token.KindDot {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindSchemaName, p.tok[0]))
			p.advance()
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		} else if p.tok[1].Kind == token.KindIdentifier {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindSchemaName, p.tok[0]))
			p.advance()
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing dot`)))
		}
	} else if p.tok[0].Kind == token.KindDot {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing schema name`)))
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	}

	if p.tok[0].Kind == token.KindIdentifier {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindTableName, p.tok[0]))
		p.advance()
	} else {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing table name`)))
	}

	if p.tok[0].Kind == token.KindAs {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		if p.tok[0].Kind == token.KindIdentifier {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindTableAlias, p.tok[0]))
			p.advance()
		} else {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing table alias`)))
		}
	}

	if p.tok[0].Kind == token.KindIndexed {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		if p.tok[0].Kind == token.KindBy {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		} else if p.tok[0].Kind == token.KindIdentifier {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "BY"`)))
		}

		if p.tok[0].Kind == token.KindIdentifier {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindIndexName, p.tok[0]))
			p.advance()
		} else {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing index name`)))
		}
	} else if p.tok[0].Kind == token.KindNot {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		if p.tok[0].Kind == token.KindIndexed {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		} else {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "INDEXED"`)))
		}
	}

	return nt
}

// returningClause parses a returning clause.
func (p *Parser) returningClause() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindReturningClause)
	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	p.advance()

	if p.isStartOfExpression(p.tok[0]) || p.tok[0].Kind == token.KindAsterisk {
		nt.AddChild(p.returningItemList())
	} else {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing returning item`)))
	}

	return nt
}

// returningItemList parses a list of itens in a returning clause.
func (p *Parser) returningItemList() parsetree.NonTerminal {
	return commaListFunc(p, "returning item", (*Parser).returningItem,
		func(t *token.Token) bool { return p.isStartOfExpression(t) || t.Kind == token.KindAsterisk },
		func(t *token.Token) bool {
			return t.Kind == token.KindRightParen || t.Kind == token.KindSemicolon || t.Kind == token.KindEOF
		},
	)
}

// returningItem parses a item in a returning clause.
func (p *Parser) returningItem() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindReturningItem)

	if p.isStartOfExpression(p.tok[0]) {
		nt.AddChild(p.expression())
		if p.tok[0].Kind != token.KindAs && p.tok[0].Kind != token.KindIdentifier {
			return nt
		}

		if p.tok[0].Kind == token.KindAs {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		}

		if p.tok[0].Kind == token.KindIdentifier {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindColumnAlias, p.tok[0]))
			p.advance()
		} else {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing column alias`)))
		}
	} else if p.tok[0].Kind == token.KindAsterisk {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	}

	return nt
}

// detach parses a detach statement.
func (p *Parser) detach() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindDetach)
	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	p.advance()

	if p.tok[0].Kind == token.KindDatabase {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	}

	if p.tok[0].Kind == token.KindIdentifier || p.tok[0].Kind == token.KindTemp {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindSchemaName, p.tok[0]))
		p.advance()
	} else {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing schema name`)))
	}

	return nt
}

// dropIndex parses a drop index statement.
func (p *Parser) dropIndex() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindDropIndex)
	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	p.advance()

	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	p.advance()

	if p.tok[0].Kind == token.KindIf {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		if p.tok[0].Kind == token.KindExists {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		} else {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "EXISTS"`)))
		}
	}

	if p.tok[0].Kind == token.KindIdentifier || p.tok[0].Kind == token.KindTemp {
		if p.tok[1].Kind == token.KindDot {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindSchemaName, p.tok[0]))
			p.advance()
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		} else if p.tok[1].Kind == token.KindIdentifier {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindSchemaName, p.tok[0]))
			p.advance()
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing dot`)))
		}
	} else if p.tok[0].Kind == token.KindDot {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing schema name`)))
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	}

	if p.tok[0].Kind == token.KindIdentifier {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindIndexName, p.tok[0]))
		p.advance()
	} else {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing index name`)))
	}

	return nt
}

// dropTable parses a drop table statement.
func (p *Parser) dropTable() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindDropTable)
	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	p.advance()

	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	p.advance()

	if p.tok[0].Kind == token.KindIf {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		if p.tok[0].Kind == token.KindExists {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		} else {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "EXISTS"`)))
		}
	}

	if p.tok[0].Kind == token.KindIdentifier || p.tok[0].Kind == token.KindTemp {
		if p.tok[1].Kind == token.KindDot {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindSchemaName, p.tok[0]))
			p.advance()
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		} else if p.tok[1].Kind == token.KindIdentifier {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindSchemaName, p.tok[0]))
			p.advance()
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing dot`)))
		}
	} else if p.tok[0].Kind == token.KindDot {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing schema name`)))
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	}

	if p.tok[0].Kind == token.KindIdentifier {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindTableName, p.tok[0]))
		p.advance()
	} else {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing table name`)))
	}

	return nt
}

// insert parses a insert statement.
func (p *Parser) insert() parsetree.NonTerminal {
	// TODO: implement this method.
	nt := parsetree.NewNonTerminal(parsetree.KindInsert)
	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	p.advance()

	if p.isStartOfExpression(p.tok[0]) {
		nt.AddChild(p.expression())
	}

	return nt
}

// selectStatement parses a select statement.
func (p *Parser) selectStatement() parsetree.NonTerminal {
	// TODO: implement this method.
	nt := parsetree.NewNonTerminal(parsetree.KindSelect)
	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	p.advance()

	if p.isStartOfExpression(p.tok[0]) {
		nt.AddChild(p.expression())
	}

	return nt
}

// update parses a update statement.
func (p *Parser) update() parsetree.NonTerminal {
	// TODO: implement this method.
	nt := parsetree.NewNonTerminal(parsetree.KindUpdate)
	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	p.advance()

	if p.isStartOfExpression(p.tok[0]) {
		nt.AddChild(p.expression())
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

		if p.isStartOfExpression(p.tok[0]) {
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

		if p.isStartOfExpression(p.tok[0]) {
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

		if p.isStartOfExpression(p.tok[0]) {
			nt.AddChild(p.expression3())
		} else {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing expression`)))
		}

		return nt
	}

	return p.expression4()
}

// expression4 parses a expression with precedence at least 4.
// TODO: refactor the if bodies to be in separated functions. Maybe a function
// for GLOB, a for NOT GLOB, etc.
func (p *Parser) expression4() parsetree.Construction {
	exp := p.expression5()

	for {
		switch p.tok[0].Kind {
		case token.KindEqual, token.KindEqualEqual:
			nt := parsetree.NewNonTerminal(parsetree.KindEqual)
			nt.AddChild(exp)

			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()

			if p.isStartOfExpressionAtLeast4(p.tok[0]) {
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

			if p.isStartOfExpressionAtLeast4(p.tok[0]) {
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

			if p.isStartOfExpressionAtLeast4(p.tok[0]) {
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

			if p.isStartOfExpressionAtLeast4(p.tok[0]) {
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

			if p.isStartOfExpressionAtLeast4(p.tok[0]) {
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

			if p.isStartOfExpressionAtLeast4(p.tok[0]) {
				nt.AddChild(p.expression5())
			} else {
				nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing expression (not starting with "NOT")`)))
			}

			// apparently there is an error in the ESCAPE precedence documentation
			if p.tok[0].Kind == token.KindEscape {
				nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
				p.advance()

				if p.isStartOfExpressionAtLeast4(p.tok[0]) {
					nt.AddChild(p.expression4())
				} else {
					nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing expression (not starting with "NOT")`)))
					return nt
				}
			}

			exp = nt
		case token.KindIsnull:
			nt := parsetree.NewNonTerminal(parsetree.KindIsnull)
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

				if p.isStartOfExpressionAtLeast4(p.tok[0]) {
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

				if p.isStartOfExpressionAtLeast4(p.tok[0]) {
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

				if p.isStartOfExpressionAtLeast4(p.tok[0]) {
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

				if p.isStartOfExpressionAtLeast4(p.tok[0]) {
					nt.AddChild(p.expression5())
				} else {
					nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing expression (not starting with "NOT")`)))
				}

				if p.tok[0].Kind == token.KindEscape {
					nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
					p.advance()

					if p.isStartOfExpressionAtLeast4(p.tok[0]) {
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
		if p.isStartOfExpression(p.tok[2]) && p.tok[2].Kind != token.KindNot {
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
			} else if p.isStartOfExpression(p.tok[0]) {
				nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "FROM"`)))
			}

			if p.isStartOfExpression(p.tok[0]) && p.tok[0].Kind != token.KindNot {
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
		} else if p.isStartOfExpression(p.tok[0]) {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "FROM"`)))
		}

		if p.isStartOfExpression(p.tok[0]) && p.tok[0].Kind != token.KindNot {
			nt.AddChild(p.expression4())
		} else {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing expression (not starting with "NOT")`)))
		}
	} else if p.isStartOfExpression(p.tok[1]) {
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
	nt := parsetree.NewNonTerminal(parsetree.KindBetween)
	nt.AddChild(exp)
	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	p.advance()

	if p.isStartOfExpressionAtLeast4(p.tok[0]) {
		nt.AddChild(p.expression4())
	} else if p.tok[0].Kind == token.KindAnd {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing expression (not starting with "NOT")`)))
	}

	if p.tok[0].Kind == token.KindAnd {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else if p.isStartOfExpressionAtLeast4(p.tok[0]) {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "AND"`)))
	}

	if p.isStartOfExpressionAtLeast4(p.tok[0]) {
		nt.AddChild(p.expression4())
	} else {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing expression (not starting with "NOT")`)))
	}

	return nt
}

// notBetween parses a not between expression.
func (p *Parser) notBetween(exp parsetree.Construction) parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindNotBetween)
	nt.AddChild(exp)
	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	p.advance()
	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	p.advance()

	if p.isStartOfExpressionAtLeast4(p.tok[0]) {
		nt.AddChild(p.expression4())
	} else if p.tok[0].Kind == token.KindAnd {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing expression (not starting with "NOT")`)))
	}

	if p.tok[0].Kind == token.KindAnd {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else if p.isStartOfExpressionAtLeast4(p.tok[0]) {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "AND"`)))
	}

	if p.isStartOfExpressionAtLeast4(p.tok[0]) {
		nt.AddChild(p.expression4())
	} else {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing expression (not starting with "NOT")`)))
	}

	return nt
}

// in parses a in expression.
func (p *Parser) in(exp parsetree.Construction) parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindIn)
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
		} else if p.isStartOfExpressionAtLeast4(p.tok[0]) {
			nt.AddChild(commaListFunc(p, "expression", (*Parser).expression4, p.isStartOfExpressionAtLeast4, func(t *token.Token) bool {
				return t.Kind == token.KindRightParen || t.Kind == token.KindSemicolon || t.Kind == token.KindEOF
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

		if p.isStartOfExpressionAtLeast4(p.tok[0]) {
			nt.AddChild(commaListFunc(p, "expression", (*Parser).expression4, p.isStartOfExpressionAtLeast4, func(t *token.Token) bool {
				return t.Kind == token.KindRightParen || t.Kind == token.KindSemicolon || t.Kind == token.KindEOF
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
	nt := parsetree.NewNonTerminal(parsetree.KindNotIn)
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
		} else if p.isStartOfExpressionAtLeast4(p.tok[0]) {
			nt.AddChild(commaListFunc(p, "expression", (*Parser).expression4, p.isStartOfExpressionAtLeast4, func(t *token.Token) bool {
				return t.Kind == token.KindRightParen || t.Kind == token.KindSemicolon || t.Kind == token.KindEOF
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

		if p.isStartOfExpressionAtLeast4(p.tok[0]) {
			nt.AddChild(commaListFunc(p, "expression", (*Parser).expression4, p.isStartOfExpressionAtLeast4, func(t *token.Token) bool {
				return t.Kind == token.KindRightParen || t.Kind == token.KindSemicolon || t.Kind == token.KindEOF
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

// isStartOfExpressionAtLeast4 reports whether tok is a start of expression of precedence at least 4.
func (p *Parser) isStartOfExpressionAtLeast4(tok *token.Token) bool {
	return p.isStartOfExpression(tok) && tok.Kind != token.KindNot
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

		if p.isStartOfExpressionAtLeast4(p.tok[0]) {
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

		if p.isStartOfExpressionAtLeast4(p.tok[0]) {
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

		if p.isStartOfExpressionAtLeast4(p.tok[0]) {
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

		if p.isStartOfExpressionAtLeast4(p.tok[0]) {
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

		if p.isStartOfExpressionAtLeast4(p.tok[0]) {
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

	if p.isStartOfExpressionAtLeast4(p.tok[0]) {
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

// isStartOfExpression reports whether tok is a start of expression.
func (p *Parser) isStartOfExpression(tok *token.Token) bool {
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

// isLiteralValue reports whether tok is a literal value.
func (p *Parser) isLiteralValue(tok *token.Token) bool {
	if slices.Contains(
		[]token.Kind{token.KindNumeric, token.KindString, token.KindBlob,
			token.KindNull, token.KindCurrentTime, token.KindCurrentDate,
			token.KindCurrentTimestamp, token.KindRowId},
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

// isBindParameter reports whether tok is a bind parameter.
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

	if p.tok[0].Kind == token.KindDistinct || p.isStartOfExpression(p.tok[0]) || p.tok[0].Kind == token.KindAsterisk {
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

	if p.isStartOfExpression(p.tok[0]) {
		nt.AddChild(commaListFunc(p, "argument", (*Parser).expression, p.isStartOfExpression, func(t *token.Token) bool {
			return t.Kind == token.KindOrder || t.Kind == token.KindRightParen || t.Kind == token.KindSemicolon ||
				t.Kind == token.KindEOF
		}))
	} else {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing argument`)))
	}

	if p.tok[0].Kind == token.KindOrder {
		nt.AddChild(p.orderBy(func(t *token.Token) bool {
			return t.Kind == token.KindRightParen || t.Kind == token.KindSemicolon || t.Kind == token.KindEOF
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
	} else if p.isStartOfExpression(p.tok[0]) || isInFollowSet(p.tok[0]) {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "BY"`)))
	}

	if p.isStartOfExpression(p.tok[0]) {
		nt.AddChild(commaListFunc(p, "ordering term", (*Parser).orderingTerm, p.isStartOfExpression, isInFollowSet))
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
	} else if p.isStartOfExpression(p.tok[0]) {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "WHERE"`)))
	}

	if p.isStartOfExpression(p.tok[0]) {
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
	} else if slices.Contains(
		[]token.Kind{token.KindPartition, token.KindOrder, token.KindRange, token.KindRows, token.KindGroups},
		p.tok[0].Kind) {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorExpecting, errors.New(`expecting window name, or "("`)))
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
		} else if p.isStartOfExpression(p.tok[0]) {
			pb.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "BY"`)))
		}

		followSet := []token.Kind{
			token.KindOrder, token.KindRange, token.KindRows, token.KindGroups, token.KindRightParen, token.KindSemicolon,
		}
		if p.isStartOfExpression(p.tok[0]) {
			pb.AddChild(commaListFunc(p, "expression", (*Parser).expression, p.isStartOfExpression, func(t *token.Token) bool {
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
	} else if p.isStartOfExpression(p.tok[0]) {
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
		} else {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "PRECEDING"`)))
		}
	} else if p.tok[0].Kind == token.KindCurrent {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		if p.tok[0].Kind == token.KindRow {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		} else if p.tok[0].Kind == token.KindAnd {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "ROW"`)))
		}
	} else if p.isStartOfExpressionAtLeast4(p.tok[0]) {
		exp := parsetree.NewNonTerminal(parsetree.KindExpression)
		exp.AddChild(p.expression4())

		nt.AddChild(exp)

		if p.tok[0].Kind == token.KindPreceding || p.tok[0].Kind == token.KindFollowing {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		} else if p.tok[0].Kind == token.KindAnd {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorExpecting, errors.New(`expecting "PRECEDING", or "FOLLOWING"`)))
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
	}

	if p.tok[0].Kind == token.KindAnd {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else {
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
	} else if p.isStartOfExpressionAtLeast4(p.tok[0]) {
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

	if p.isStartOfExpression(p.tok[0]) || p.tok[0].Kind == token.KindComma {
		nt.AddChild(commaListFunc(p, "expression", (*Parser).expression, p.isStartOfExpression, func(t *token.Token) bool {
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

	if p.isStartOfExpression(p.tok[0]) {
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

	if p.isStartOfExpression(p.tok[0]) {
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

	if p.isStartOfExpression(p.tok[0]) {
		nt.AddChild(p.expression())
	} else if p.tok[0].Kind == token.KindThen {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing expression`)))
	}

	if p.tok[0].Kind == token.KindThen {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else if p.isStartOfExpression(p.tok[0]) {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "THEN"`)))
	}

	if p.isStartOfExpression(p.tok[0]) {
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

	if p.isStartOfExpression(p.tok[0]) {
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
	} else if p.isStartOfExpression(p.tok[0]) {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing ","`)))
	}

	if p.isStartOfExpression(p.tok[0]) {
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

// commaList parses a list of items separated by commas. itemMeaning is the meaning of the item, item is the function
// that parses a item, itemStart is for the tokens that starts a item, followSet is for the tokens that follow the list.
func commaList[I parsetree.Construction](p *Parser, itemMeaning string, item func(p *Parser) I, itemStart, followSet []token.Kind) parsetree.NonTerminal {
	return commaListFunc(p, itemMeaning, item,
		func(t *token.Token) bool { return slices.Contains(itemStart, t.Kind) },
		func(t *token.Token) bool { return slices.Contains(followSet, t.Kind) },
	)
}

// commaList parses a list of items separated by commas. itemMeaning is the meaning of the item, item is the function
// that parses a item, isItemStart is for the tokens that starts a item, isInFollowSet is for the tokens that follow the list.
func commaListFunc[I parsetree.Construction](p *Parser, itemMeaning string, item func(p *Parser) I, isItemStart, isInFollowSet func(*token.Token) bool) parsetree.NonTerminal {
	return listFunc(p, token.KindComma, itemMeaning, item, isItemStart, isInFollowSet)
}

// list parses a list of items separated by sep. itemMeaning is the meaning of the item, item is the function
// that parses a item, isItemStart is for the tokens that starts a item, isInFollowSet is for the tokens that follow the list.
func listFunc[I parsetree.Construction](p *Parser, sep token.Kind, itemMeaning string, item func(p *Parser) I, isItemStart, isInFollowSet func(*token.Token) bool) parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindCommaList)

	var skipped bool

	skipPred := func(t *token.Token) bool {
		return isItemStart(t) || isInFollowSet(t) || t.Kind == sep
	}

	if isItemStart(p.tok[0]) {
		nt.AddChild(item(p))
	} else if p.tok[0].Kind == sep {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing `+itemMeaning)))
	}

	for !isInFollowSet(p.tok[0]) {
		if p.tok[0].Kind == sep {
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
			nt.AddChild(item(p))
			skipped = false
		} else if p.tok[0].Kind == sep || isInFollowSet(p.tok[0]) {
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
