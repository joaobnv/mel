// This package deals with the parsing of the SQL.
package parser

// TODO: implement the quote rule exceptions: lang_keywords.html.
// TODO: handle tables named TEMP, and also columns named TEMP.

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/joaobnv/mel/sqlite/v3_46_1/lexer"
	"github.com/joaobnv/mel/sqlite/v3_46_1/parsetree"
	"github.com/joaobnv/mel/sqlite/v3_46_1/token"
)

type syntaxError struct {
	msg      string
	expected []token.Kind
	got      *token.Token
}

func (se *syntaxError) Error() string {
	if se.msg != "" {
		return se.msg
	}
	if len(se.expected) == 1 {
		return fmt.Sprintf("expecting %s, got %s", se.expected[0], se.got.Kind)
	}
	return fmt.Sprintf("expecting %s, got %s", se.expected, se.got.Kind)
}

// Parser is a parser for the SQL.
type Parser struct {
	// comments contains the comments for the current SQLStatement being parsed.
	comments map[*token.Token][]*token.Token
	// tok contains the current look ahead tokens.
	tok [4]*token.Token
	// l is the lexer.
	l         *lexer.Lexer
	treeStack []parsetree.Construction
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
		p.advance()
	}

	p.pushTreeKind(parsetree.KindSQLStatement)

	var explain bool
	if p.isSeq(token.KindExplain, token.KindQuery) {
		p.pushTreeKind(parsetree.KindExplainQueryPlan)
		p.term(token.KindExplain)
		p.term(token.KindQuery)
		p.term(token.KindPlan)
		explain = true
	} else if p.is(token.KindExplain) {
		p.pushTreeKind(parsetree.KindExplain)
		p.term(token.KindExplain)
		explain = true
	}

	switch p.tok[0].Kind {
	case token.KindAlter:
		p.addChild(p.alterTable())
	case token.KindAnalyze:
		p.addChild(p.analyze())
	case token.KindAttach:
		p.addChild(p.attach())
	case token.KindBegin:
		p.addChild(p.begin())
	case token.KindCommit, token.KindEnd:
		p.addChild(p.commit())
	case token.KindRollback:
		p.addChild(p.rollback())
	case token.KindCreate:
		p.addChild(p.create())
	case token.KindDelete:
		p.addChild(p.delete(nil))
	case token.KindDetach:
		p.addChild(p.detach())
	case token.KindDrop:
		p.addChild(p.drop())
	case token.KindInsert, token.KindReplace:
		p.addChild(p.insert(nil))
	case token.KindWith:
		p.addChild(p.with())
	case token.KindPragma:
		p.addChild(p.pragma())
	case token.KindReindex:
		p.addChild(p.reindex())
	case token.KindRelease:
		p.addChild(p.release())
	case token.KindSavepoint:
		p.addChild(p.savepoint())
	case token.KindSelect:
		p.addChild(p.selectStatement(nil))
	case token.KindUpdate:
		p.addChild(p.update(nil))
	case token.KindVacuum:
		p.addChild(p.vacuum())
	}

	if explain {
		p.addChild(p.popTree())
	}

	if p.is(token.KindSemicolon) {
		p.term(token.KindSemicolon)
	} else if p.is(token.KindEOF) {
		p.term(token.KindEOF)
	}

	p.comments = nil
	return p.popTree(), comments
}

// alterTable parses a alter table statement.
func (p *Parser) alterTable() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindAlterTable)
	p.term(token.KindAlter)
	p.term(token.KindTable)

	if p.isSeq(token.KindIdentifier, token.KindDot) {
		p.termKind(parsetree.KindSchemaName, token.KindIdentifier)
		p.term(token.KindDot)
	}

	p.termKind(parsetree.KindTableName, token.KindIdentifier)

	switch p.token(token.KindRename, token.KindAdd, token.KindDrop) {
	case token.KindRename:
		switch p.tokenPos(1, token.KindTo, token.KindColumn, token.KindIdentifier) {
		case token.KindTo:
			p.addChild(p.alterTableRenameTo())
		default:
			p.addChild(p.alterTableRenameColumn())
		}
	case token.KindAdd:
		p.addChild(p.alterTableAddColumn())
	case token.KindDrop:
		p.addChild(p.alterTableDropColumn())
	}

	return p.popTree()
}

func (p *Parser) alterTableRenameTo() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindRenameTo)
	p.term(token.KindRename)
	p.term(token.KindTo)
	p.termKind(parsetree.KindTableName, token.KindIdentifier)
	return p.popTree()
}

func (p *Parser) alterTableRenameColumn() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindRenameColumn)
	p.term(token.KindRename)

	if p.is(token.KindColumn) {
		p.term(token.KindColumn)
	}

	p.termKind(parsetree.KindColumnName, token.KindIdentifier)
	p.term(token.KindTo)
	p.termKind(parsetree.KindColumnName, token.KindIdentifier)
	return p.popTree()
}

func (p *Parser) alterTableAddColumn() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindAddColumn)
	p.term(token.KindAdd)

	if p.is(token.KindColumn) {
		p.term(token.KindColumn)
	}

	p.token(token.KindIdentifier)
	p.addChild(p.columnDefinition())
	return p.popTree()
}

func (p *Parser) alterTableDropColumn() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindDropColumn)
	p.term(token.KindDrop)

	if p.is(token.KindColumn) {
		p.term(token.KindColumn)
	}

	p.termKind(parsetree.KindColumnName, token.KindIdentifier)
	return p.popTree()
}

// columnDefinition parses a column definition.
func (p *Parser) columnDefinition() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindColumnDefinition)

	p.termKind(parsetree.KindColumnName, token.KindIdentifier)

	if p.is(token.KindIdentifier) {
		p.addChild(p.typeName())
	}

	for p.isAnyOf(token.KindConstraint, token.KindPrimary, token.KindNot, token.KindUnique, token.KindCheck,
		token.KindDefault, token.KindCollate, token.KindReferences, token.KindGenerated, token.KindAs) {
		p.addChild(p.columnConstraint())
	}

	return p.popTree()
}

// typeName parses a type name.
func (p *Parser) typeName() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindTypeName)

	p.term(token.KindIdentifier)
	for p.is(token.KindIdentifier) {
		p.term(token.KindIdentifier)
	}

	if !p.is(token.KindLeftParen) {
		return p.popTree()
	}

	p.term(token.KindLeftParen)
	p.signedNumber()

	if p.is(token.KindRightParen) {
		p.term(token.KindRightParen)
		return p.popTree()
	}

	p.term(token.KindComma)
	p.signedNumber()
	p.term(token.KindRightParen)

	return p.popTree()
}

func (p *Parser) signedNumber() {
	if p.is(token.KindMinus) {
		p.term(token.KindMinus)
	} else if p.is(token.KindPlus) {
		p.term(token.KindPlus)
	}

	p.term(token.KindNumeric)
}

// columnConstraint parses a column constraint.
func (p *Parser) columnConstraint() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindColumnConstraint)

	if p.is(token.KindConstraint) {
		p.term(token.KindConstraint)
		p.termKind(parsetree.KindConstraintName, token.KindIdentifier)
	}

	switch p.token(token.KindPrimary, token.KindNot, token.KindUnique, token.KindCheck, token.KindDefault,
		token.KindCollate, token.KindReferences, token.KindGenerated, token.KindAs) {
	case token.KindPrimary:
		p.addChild(p.primaryKeyColumnConstraint())
	case token.KindNot:
		p.addChild(p.notNullColumnConstraint())
	case token.KindUnique:
		p.addChild(p.uniqueColumnConstraint())
	case token.KindCheck:
		p.addChild(p.checkColumnConstraint())
	case token.KindDefault:
		p.addChild(p.defaultColumnConstraint())
	case token.KindCollate:
		p.addChild(p.collateColumnConstraint())
	case token.KindReferences:
		p.addChild(p.foreignKeyColumnConstraint())
	case token.KindGenerated, token.KindAs:
		p.addChild(p.generatedColumnConstraint())
	}

	return p.popTree()
}

// primaryKeyColumnConstraint parses a primary key column constraint clause.
func (p *Parser) primaryKeyColumnConstraint() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindPrimaryKeyColumnConstraint)
	p.term(token.KindPrimary)
	p.term(token.KindKey)

	if p.is(token.KindAsc) {
		p.term(token.KindAsc)
	} else if p.is(token.KindDesc) {
		p.term(token.KindDesc)
	}

	if p.is(token.KindOn) {
		p.addChild(p.conflictClause())
	}

	if p.is(token.KindAutoincrement) {
		p.term(token.KindAutoincrement)
	}

	return p.popTree()
}

// notNullColumnConstraint parses a not null column constraint clause.
func (p *Parser) notNullColumnConstraint() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindNotNullColumnConstraint)
	p.term(token.KindNot)
	p.term(token.KindNull)

	if p.is(token.KindOn) {
		p.addChild(p.conflictClause())
	}

	return p.popTree()
}

// uniqueColumnConstraint parses a unique column constraint clause.
func (p *Parser) uniqueColumnConstraint() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindUniqueColumnConstraint)
	p.term(token.KindUnique)

	if p.is(token.KindOn) {
		p.addChild(p.conflictClause())
	}

	return p.popTree()
}

// checkColumnConstraint parses a check column constraint clause.
func (p *Parser) checkColumnConstraint() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindCheckColumnConstraint)
	p.term(token.KindCheck)
	p.term(token.KindLeftParen)
	p.addChild(p.expression())
	p.term(token.KindRightParen)
	return p.popTree()
}

// defaultColumnConstraint parses a default column constraint clause.
func (p *Parser) defaultColumnConstraint() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindDefaultColumnConstraint)
	p.term(token.KindDefault)

	if p.is(token.KindLeftParen) {
		p.term(token.KindLeftParen)
		p.addChild(p.expression())
		p.term(token.KindRightParen)
	} else if p.isLiteralValue() {
		p.term()
	} else { // note that a numeric token is a literal value
		p.token(token.KindPlus, token.KindMinus)
		p.signedNumber()
	}

	return p.popTree()
}

// collateColumnConstraint parses a collate column constraint clause.
func (p *Parser) collateColumnConstraint() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindCollateColumnConstraint)
	p.term(token.KindCollate)
	p.termKind(parsetree.KindCollationName, token.KindIdentifier)
	return p.popTree()
}

// generatedColumnConstraint parses a generated column constraint clause.
func (p *Parser) generatedColumnConstraint() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindGeneratedColumnConstraint)
	if p.is(token.KindGenerated) {
		p.term()
		p.term(token.KindAlways)
	}

	p.term(token.KindAs)
	p.term(token.KindLeftParen)
	p.addChild(p.expression())
	p.term(token.KindRightParen)

	if p.isAnyOf(token.KindStored, token.KindVirtual) {
		p.term()
	}

	return p.popTree()
}

// foreignKeyColumnConstraint parses a foreignKey column constraint clause.
func (p *Parser) foreignKeyColumnConstraint() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindForeignKeyColumnConstraint)
	p.addChild(p.foreignKeyClause())
	return p.popTree()
}

// foreignKeyClause parses a foreign key clause.
func (p *Parser) foreignKeyClause() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindForeignKeyClause)
	p.term(token.KindReferences)
	p.termKind(parsetree.KindTableName, token.KindIdentifier)

	if p.is(token.KindLeftParen) {
		p.term(token.KindLeftParen)
		p.addChild(p.columnNameList())
		p.term(token.KindRightParen)
	}

	if !p.isAnyOf(token.KindOn, token.KindMatch, token.KindDeferrable, token.KindNot) {
		return p.popTree()
	}
	for p.isAnyOf(token.KindOn, token.KindMatch) {
		switch p.token(token.KindOn, token.KindMatch) {
		case token.KindOn:
			p.foreignKeyOn()
		case token.KindMatch:
			p.foreignKeyMatch()
		}
	}

	if !p.isAnyOf(token.KindDeferrable, token.KindNot) {
		return p.popTree()
	}

	if p.is(token.KindNot) {
		p.term()
	}
	p.term(token.KindDeferrable)

	if !p.is(token.KindInitially) {
		return p.popTree()
	}
	p.term()
	p.term(token.KindDeferred, token.KindImmediate)

	return p.popTree()
}

func (p *Parser) foreignKeyOn() {
	p.term(token.KindOn)
	p.term(token.KindDelete, token.KindUpdate)

	switch p.token(token.KindSet, token.KindCascade, token.KindRestrict, token.KindNo) {
	case token.KindSet:
		p.term(token.KindSet)
		p.term(token.KindNull, token.KindDefault)
	case token.KindCascade:
		p.term(token.KindCascade)
	case token.KindRestrict:
		p.term(token.KindRestrict)
	case token.KindNo:
		p.term(token.KindNo)
		p.term(token.KindAction)
	}
}

func (p *Parser) foreignKeyMatch() {
	p.term(token.KindMatch)
	p.term(token.KindIdentifier)
}

// conflictClause parses a conflict clause.
func (p *Parser) conflictClause() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindConflictClause)
	p.term(token.KindOn)
	p.term(token.KindConflict)
	p.term(token.KindRollback, token.KindAbort, token.KindFail, token.KindIgnore, token.KindReplace)
	return p.popTree()
}

// analyze parses a analyze statement.
func (p *Parser) analyze() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindAnalyze)
	p.term(token.KindAnalyze)

	if p.isSeq(token.KindIdentifier, token.KindDot) {
		p.termKind(parsetree.KindSchemaName, token.KindIdentifier)
		p.term(token.KindDot)
		p.termKind(parsetree.KindTableOrIndexName, token.KindIdentifier)
	} else {
		p.termKind(parsetree.KindSchemaIndexOrTableName, token.KindIdentifier)
	}

	return p.popTree()
}

// attach parses a attach statement.
func (p *Parser) attach() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindAttach)
	p.term(token.KindAttach)

	if p.is(token.KindDatabase) {
		p.term(token.KindDatabase)
	}

	p.addChild(p.expression())
	p.term(token.KindAs)
	p.termKind(parsetree.KindSchemaName, token.KindIdentifier)

	return p.popTree()
}

// begin parses a begin statement.
func (p *Parser) begin() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindBegin)
	p.term(token.KindBegin)

	if p.is(token.KindDeferred) {
		p.term(token.KindDeferred)
	} else if p.is(token.KindImmediate) {
		p.term(token.KindImmediate)
	} else if p.is(token.KindExclusive) {
		p.term(token.KindExclusive)
	}

	if p.is(token.KindTransaction) {
		p.term(token.KindTransaction)
	}

	return p.popTree()
}

// commit parses a commit statement.
func (p *Parser) commit() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindCommit)
	p.term(token.KindCommit, token.KindEnd)

	if p.is(token.KindTransaction) {
		p.term()
	}

	return p.popTree()
}

// rollback parses a rollback statement.
func (p *Parser) rollback() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindRollback)
	p.term(token.KindRollback)

	if p.is(token.KindTransaction) {
		p.term()
	}

	if !p.is(token.KindTo) {
		return p.popTree()
	}

	p.term(token.KindTo)
	if p.is(token.KindSavepoint) {
		p.term()
	}
	p.termKind(parsetree.KindSavepointName, token.KindIdentifier)

	return p.popTree()
}

func (p *Parser) create() parsetree.NonTerminal {
	if p.isSeq(token.KindCreate, token.KindIndex) || p.isSeq(token.KindCreate, token.KindUnique) {
		return p.createIndex()
	}
	if p.isSeq(token.KindCreate, token.KindTable) {
		return p.createTable()
	}
	if p.isSeq(token.KindCreate, token.KindTrigger) {
		return p.createTrigger()
	}
	if p.isSeq(token.KindCreate, token.KindView) {
		return p.createView()
	}
	if p.isSeq(token.KindCreate, token.KindTemp) || p.isSeq(token.KindCreate, token.KindTemporary) {
		if p.isPos(2, token.KindTable) {
			return p.createTable()
		}
		if p.isPos(2, token.KindTrigger) {
			return p.createTrigger()
		}
		if p.isPos(2, token.KindView) {
			return p.createView()
		}
	}
	// if p.isSeq(token.KindCreate, token.KindVirtual)
	return p.createVirtualTable()
}

// createIndex parses a create index statement.
func (p *Parser) createIndex() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindCreateIndex)
	p.term(token.KindCreate)

	if p.is(token.KindUnique) {
		p.term()
	}

	p.term(token.KindIndex)

	if p.is(token.KindIf) {
		p.term()
		p.term(token.KindNot)
		p.term(token.KindExists)
	}

	if p.isSeq(token.KindIdentifier, token.KindDot) {
		p.termKind(parsetree.KindSchemaName, token.KindIdentifier)
		p.term(token.KindDot)
	}
	p.termKind(parsetree.KindIndexName, token.KindIdentifier)

	p.term(token.KindOn)
	p.termKind(parsetree.KindTableName, token.KindIdentifier)

	p.term(token.KindLeftParen)
	p.addChild(p.indexedColumnList(true))
	p.term(token.KindRightParen)

	if p.is(token.KindWhere) {
		p.term()
		p.addChild(p.expression())
	}

	return p.popTree()
}

func (p *Parser) indexedColumnList(canBeExpression bool) parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindCommaList)
	p.addChild(p.indexedColumn(canBeExpression))
	for p.is(token.KindComma) {
		p.term()
		p.addChild(p.indexedColumn(canBeExpression))
	}
	return p.popTree()
}

// indexedColumn parses a indexed column.
// TODO: remove the canBeExpression param.
func (p *Parser) indexedColumn(canBeExpression bool) parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindIndexedColumn)
	if canBeExpression {
		// the collate part is handled by the expression parser
		p.addChild(p.expression())
	} else {
		p.termKind(parsetree.KindColumnName, token.KindIdentifier)

		if p.is(token.KindCollate) {
			p.term()
			p.termKind(parsetree.KindCollationName, token.KindIdentifier)
		}
	}

	if p.isAnyOf(token.KindAsc, token.KindDesc) {
		p.term()
	}

	return p.popTree()
}

// createTable parses a create table statement.
func (p *Parser) createTable() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindCreateTable)
	p.term(token.KindCreate)

	if p.isAnyOf(token.KindTemp, token.KindTemporary) {
		p.term()
	}

	p.term(token.KindTable)

	if p.is(token.KindIf) {
		p.term()
		p.term(token.KindNot)
		p.term(token.KindExists)
	}

	if p.isSeq(token.KindIdentifier, token.KindDot) {
		p.termKind(parsetree.KindSchemaName, token.KindIdentifier)
		p.term(token.KindDot)
	} else if p.isSeq(token.KindTemp, token.KindDot) {
		p.termKind(parsetree.KindSchemaName, token.KindTemp)
		p.term(token.KindDot)
	}

	p.termKind(parsetree.KindTableName, token.KindIdentifier)

	switch p.token(token.KindLeftParen, token.KindAs) {
	case token.KindLeftParen:
		p.createTableColumns()
	case token.KindAs:
		p.createTableSelect()
	}

	return p.popTree()
}

func (p *Parser) createTableColumns() {
	p.term(token.KindLeftParen)
	p.addChild(p.columnDefinitionList())

	if p.is(token.KindComma) {
		p.term()
		p.addChild(p.tableConstraintList())
	}

	p.term(token.KindRightParen)

	if p.isAnyOf(token.KindWithout, token.KindStrict) {
		p.addChild(p.tableOptions())
	}
}

func (p *Parser) createTableSelect() {
	p.term(token.KindAs)
	p.addChild(p.selectStatement(nil))
}

func (p *Parser) columnDefinitionList() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindCommaList)

	p.addChild(p.columnDefinition())
	for p.isSeq(token.KindComma, token.KindIdentifier) {
		p.term(token.KindComma)
		p.addChild(p.columnDefinition())
	}

	return p.popTree()
}

// tableConstraintList parses a list of tale constraints in a create table.
func (p *Parser) tableConstraintList() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindCommaList)
	p.addChild(p.tableConstraint())
	for p.is(token.KindComma) {
		p.term()
		p.addChild(p.tableConstraint())
	}
	return p.popTree()
}

// tableConstraint parses a table constraint clause.
func (p *Parser) tableConstraint() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindTableConstraint)
	if p.is(token.KindConstraint) {
		p.term()
		p.termKind(parsetree.KindConstraintName, token.KindIdentifier)
	}

	switch p.token(token.KindPrimary, token.KindUnique, token.KindCheck, token.KindForeign) {
	case token.KindPrimary:
		p.addChild(p.primaryKeyTableConstraint())
	case token.KindUnique:
		p.addChild(p.uniqueTableConstraint())
	case token.KindCheck:
		p.addChild(p.checkTableConstraint())
	case token.KindForeign:
		p.addChild(p.foreignKeyTableConstraint())
	}

	return p.popTree()
}

// primaryKeyTableConstraint parses a primary key table constraint clause.
func (p *Parser) primaryKeyTableConstraint() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindPrimaryKeyTableConstraint)

	p.term(token.KindPrimary)
	p.term(token.KindKey)
	p.term(token.KindLeftParen)
	p.addChild(p.indexedColumnList(false))
	p.term(token.KindRightParen)

	if p.is(token.KindOn) {
		p.addChild(p.conflictClause())
	}

	return p.popTree()
}

// uniqueTableConstraint parses a unique table constraint clause.
func (p *Parser) uniqueTableConstraint() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindUniqueTableConstraint)

	p.term(token.KindUnique)
	p.term(token.KindLeftParen)
	p.addChild(p.indexedColumnList(false))
	p.term(token.KindRightParen)

	if p.is(token.KindOn) {
		p.addChild(p.conflictClause())
	}

	return p.popTree()
}

// checkTableConstraint parses a check table constraint clause.
func (p *Parser) checkTableConstraint() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindCheckTableConstraint)

	p.term(token.KindCheck)
	p.term(token.KindLeftParen)
	p.addChild(p.expression())
	p.term(token.KindRightParen)

	return p.popTree()
}

// foreignKeyTableConstraint parses a foreign key table constraint clause.
func (p *Parser) foreignKeyTableConstraint() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindForeignKeyTableConstraint)

	p.term(token.KindForeign)
	p.term(token.KindKey)
	p.term(token.KindLeftParen)
	p.addChild(p.columnNameList())
	p.term(token.KindRightParen)
	p.addChild(p.foreignKeyClause())

	return p.popTree()
}

// tableOptions parses a table options clause.
func (p *Parser) tableOptions() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindCommaList)
	for {
		p.addChild(p.tableOption())
		if !p.is(token.KindComma) {
			break
		}
		p.term(token.KindComma)
	}
	return p.popTree()
}

// tableOption parses a table option.
func (p *Parser) tableOption() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindTableOption)

	switch p.token(token.KindWithout, token.KindStrict) {
	case token.KindWithout:
		p.term()
		p.term(token.KindRowId)
	case token.KindStrict:
		p.term()
	}

	return p.popTree()
}

// columnNameList parses a list of column names.
// TODO: remove follow.
func (p *Parser) columnNameList(follow ...token.Kind) parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindCommaList)
	for {
		p.termKind(parsetree.KindColumnName, token.KindIdentifier)
		if !p.is(token.KindComma) {
			break
		}
		p.term(token.KindComma)
	}
	return p.popTree()
}

// createTrigger parses a create trigger statement.
func (p *Parser) createTrigger() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindCreateTrigger)

	p.term(token.KindCreate)
	if p.isAnyOf(token.KindTemp, token.KindTemporary) {
		p.term()
	}
	p.term(token.KindTrigger)

	if p.is(token.KindIf) {
		p.term()
		p.term(token.KindNot)
		p.term(token.KindExists)
	}

	if p.isAnyOf(token.KindIdentifier, token.KindTemp) && p.isPos(1, token.KindDot) {
		p.termKind(parsetree.KindSchemaName)
		p.term(token.KindDot)
	}
	p.termKind(parsetree.KindTriggerName, token.KindIdentifier)

	if p.isAnyOf(token.KindBefore, token.KindAfter) {
		p.term()
	} else if p.is(token.KindInstead) {
		p.term()
		p.term(token.KindOf)
	}

	switch p.token(token.KindDelete, token.KindInsert, token.KindUpdate) {
	case token.KindDelete, token.KindInsert:
		p.term()
	case token.KindUpdate:
		p.term()
		if p.is(token.KindOf) {
			p.term()
			p.addChild(p.columnNameList())
		}
	}

	p.term(token.KindOn)
	p.termKind(parsetree.KindTableName, token.KindIdentifier)

	if p.is(token.KindFor) {
		p.term()
		p.term(token.KindEach)
		p.term(token.KindRow)
	}

	if p.is(token.KindWhen) {
		p.term()
		p.addChild(p.expression())
	}

	p.addChild(p.triggerBody())

	return p.popTree()
}

// triggerBody parses the body of a create trigger statement.
func (p *Parser) triggerBody() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindTriggerBody)
	p.term(token.KindBegin)

	for {
		var withClause parsetree.NonTerminal
		if p.is(token.KindWith) {
			withClause = p.withClause()
		}

		switch p.token(token.KindDelete, token.KindInsert, token.KindSelect, token.KindUpdate) {
		case token.KindDelete:
			p.addChild(p.delete(withClause))
		case token.KindInsert:
			p.addChild(p.insert(withClause))
		case token.KindSelect:
			p.addChild(p.selectStatement(withClause))
		case token.KindUpdate:
			p.addChild(p.update(withClause))
		}

		p.term(token.KindSemicolon)

		if p.is(token.KindEnd) {
			break
		}
	}

	p.term(token.KindEnd)

	return p.popTree()
}

// createView parses a create view statement.
func (p *Parser) createView() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindCreateView)

	p.term(token.KindCreate)
	if p.isAnyOf(token.KindTemp, token.KindTemporary) {
		p.term()
	}
	p.term(token.KindView)

	if p.is(token.KindIf) {
		p.term()
		p.term(token.KindNot)
		p.term(token.KindExists)
	}

	if p.isAnyOf(token.KindIdentifier, token.KindTemp) && p.isPos(1, token.KindDot) {
		p.termKind(parsetree.KindSchemaName)
		p.term(token.KindDot)
	}
	p.termKind(parsetree.KindViewName, token.KindIdentifier)

	if p.is(token.KindLeftParen) {
		p.term()
		p.addChild(p.columnNameList())
		p.term(token.KindRightParen)
	}

	p.term(token.KindAs)
	p.addChild(p.selectStatement(nil))

	return p.popTree()
}

// createVirtualTable parses a create virtual table statement.
func (p *Parser) createVirtualTable() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindCreateVirtualTable)

	p.term(token.KindCreate)
	p.term(token.KindVirtual)
	p.term(token.KindTable)

	if p.is(token.KindIf) {
		p.term()
		p.term(token.KindNot)
		p.term(token.KindExists)
	}

	if p.isAnyOf(token.KindIdentifier, token.KindTemp) && p.isPos(1, token.KindDot) {
		p.termKind(parsetree.KindSchemaName)
		p.term(token.KindDot)
	}
	p.termKind(parsetree.KindTableName, token.KindIdentifier)

	p.term(token.KindUsing)
	p.termKind(parsetree.KindModuleName, token.KindIdentifier)

	if p.is(token.KindLeftParen) {
		p.term()
		p.addChild(p.moduleArgumentList())
		p.term(token.KindRightParen)
	}

	return p.popTree()
}

// moduleArgumentList parses a list of module arguments separated by comma.
func (p *Parser) moduleArgumentList() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindCommaList)

	for {
		p.addChild(p.moduleArgument())

		if p.is(token.KindComma) {
			p.term()
		} else {
			break
		}
	}

	return p.popTree()
}

// moduleArgument parses a module argument in a create virtual table statement.
func (p *Parser) moduleArgument() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindModuleArgument)
	var found bool
	for {
		if p.is(token.KindLeftParen) {
			found = true
			p.term()
			p.moduleArgumentInner()
			p.term(token.KindRightParen)
		} else if p.isAnyOf(token.KindComma, token.KindRightParen) {
			if found {
				break
			} else {
				p.errorUnexpected()
			}
		} else if p.is(token.KindEOF) {
			p.errorUnexpected()
		} else {
			found = true
			p.term()
		}
	}
	return p.popTree()
}

// moduleArgumentInner parses a part of a module argument. The part is delimited by parenthesis.
func (p *Parser) moduleArgumentInner() {
	for {
		if p.is(token.KindLeftParen) {
			p.term()
			p.moduleArgumentInner()
			p.term(token.KindRightParen)
		} else if p.is(token.KindRightParen) {
			break
		} else if p.is(token.KindEOF) {
			p.errorUnexpected()
		} else {
			p.term()
		}
	}
}

func (p *Parser) errorUnexpected() {
	panic(&syntaxError{msg: fmt.Sprintf("unexpected %s", p.tok[0].Kind)})
}

// delete parses a delete statement.
func (p *Parser) delete(withClause parsetree.NonTerminal) parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindDelete)
	if withClause != nil {
		p.addChild(withClause)
	}

	p.term(token.KindDelete)
	p.term(token.KindFrom)
	p.addChild(p.qualifiedTableName())

	if p.is(token.KindWhere) {
		p.addChild(p.whereClause())
	}

	if p.is(token.KindReturning) {
		p.addChild(p.returningClause())
	}

	if p.is(token.KindOrder) {
		p.addChild(p.orderByClause())
	}

	if p.is(token.KindLimit) {
		p.addChild(p.limitClause())
	}

	return p.popTree()
}

// withClause parses a with clause.
func (p *Parser) withClause() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindWithClause)

	p.term(token.KindWith)
	if p.is(token.KindRecursive) {
		p.term()
	}
	p.addChild(p.cteList())

	return p.popTree()
}

// cteList parses a list of CTEs in a with clause.
func (p *Parser) cteList() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindCommaList)

	p.addChild(p.cte())
	for p.is(token.KindComma) {
		p.term()
		p.addChild(p.cte())
	}

	return p.popTree()
}

// cte parses a common table expression.
func (p *Parser) cte() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindCTE)

	p.termKind(parsetree.KindTableName, token.KindIdentifier)

	if p.is(token.KindLeftParen) {
		p.term()
		p.addChild(p.columnNameList())
		p.term(token.KindRightParen)
	}

	p.term(token.KindAs)
	if p.is(token.KindNot) {
		p.term()
		p.term(token.KindMaterialized)
	} else if p.is(token.KindMaterialized) {
		p.term()
	}

	p.term(token.KindLeftParen)
	var withClause parsetree.NonTerminal
	if p.is(token.KindWith) {
		withClause = p.withClause()
	}
	p.addChild(p.selectStatement(withClause))
	p.term(token.KindRightParen)

	return p.popTree()
}

// qualifiedTableName parses a qualified table name.
func (p *Parser) qualifiedTableName() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindQualifiedTableName)

	if p.isAnyOf(token.KindIdentifier, token.KindTemp) && p.isPos(1, token.KindDot) {
		p.termKind(parsetree.KindSchemaName)
		p.term(token.KindDot)
	}
	p.termKind(parsetree.KindTableName, token.KindIdentifier)

	if p.is(token.KindAs) {
		p.term()
		p.termKind(parsetree.KindTableAlias, token.KindIdentifier)
	}

	if p.is(token.KindIndexed) {
		p.term()
		p.term(token.KindBy)
		p.termKind(parsetree.KindIndexName, token.KindIdentifier)
	} else if p.is(token.KindNot) {
		p.term()
		p.term(token.KindIndexed)
	}

	return p.popTree()
}

// returningClause parses a returning clause.
func (p *Parser) returningClause() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindReturningClause)

	p.term(token.KindReturning)
	p.addChild(p.returningItemList())

	return p.popTree()
}

// returningItemList parses a list of itens in a returning clause.
func (p *Parser) returningItemList() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindCommaList)

	p.addChild(p.returningItem())
	for p.is(token.KindComma) {
		p.term()
		p.addChild(p.returningItem())
	}

	return p.popTree()
}

// returningItem parses a item in a returning clause.
func (p *Parser) returningItem() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindReturningItem)

	if p.isStartOfExpression() {
		p.addChild(p.expression())
		if p.is(token.KindAs) {
			p.term()
			p.termKind(parsetree.KindColumnAlias, token.KindIdentifier)
		} else if p.is(token.KindIdentifier) {
			p.termKind(parsetree.KindColumnAlias)
		}
	} else if p.is(token.KindAsterisk) {
		p.term()
	} else {
		p.errorUnexpected()
	}

	return p.popTree()
}

// whereClause parses a where clause.
func (p *Parser) whereClause() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindWhereClause)

	p.term(token.KindWhere)
	p.addChild(p.expression())

	return p.popTree()
}

// detach parses a detach statement.
func (p *Parser) detach() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindDetach)

	p.term(token.KindDetach)
	if p.is(token.KindDatabase) {
		p.term()
	}
	p.termKind(parsetree.KindSchemaName, token.KindIdentifier, token.KindTemp)

	return p.popTree()
}

func (p *Parser) drop() parsetree.NonTerminal {
	switch p.tokenPos(1, token.KindIndex, token.KindTable, token.KindTrigger, token.KindView) {
	case token.KindIndex:
		return p.dropIndex()
	case token.KindTable:
		return p.dropTable()
	case token.KindTrigger:
		return p.dropTrigger()
	default: // token.KindView
		return p.dropView()
	}
}

// dropIndex parses a drop index statement.
func (p *Parser) dropIndex() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindDropIndex)

	p.term(token.KindDrop)
	p.term(token.KindIndex)

	if p.is(token.KindIf) {
		p.term()
		p.term(token.KindExists)
	}

	if p.isAnyOf(token.KindIdentifier, token.KindTemp) && p.isPos(1, token.KindDot) {
		p.termKind(parsetree.KindSchemaName)
		p.term(token.KindDot)
	}
	p.termKind(parsetree.KindIndexName, token.KindIdentifier)

	return p.popTree()
}

// dropTable parses a drop table statement.
func (p *Parser) dropTable() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindDropTable)

	p.term(token.KindDrop)
	p.term(token.KindTable)

	if p.is(token.KindIf) {
		p.term()
		p.term(token.KindExists)
	}

	if p.isAnyOf(token.KindIdentifier, token.KindTemp) && p.isPos(1, token.KindDot) {
		p.termKind(parsetree.KindSchemaName)
		p.term(token.KindDot)
	}
	p.termKind(parsetree.KindTableName, token.KindIdentifier)

	return p.popTree()
}

// dropTrigger parses a drop trigger statement.
func (p *Parser) dropTrigger() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindDropTrigger)

	p.term(token.KindDrop)
	p.term(token.KindTrigger)

	if p.is(token.KindIf) {
		p.term()
		p.term(token.KindExists)
	}

	if p.isAnyOf(token.KindIdentifier, token.KindTemp) && p.isPos(1, token.KindDot) {
		p.termKind(parsetree.KindSchemaName)
		p.term(token.KindDot)
	}
	p.termKind(parsetree.KindTriggerName, token.KindIdentifier)

	return p.popTree()
}

// dropView parses a drop view statement.
func (p *Parser) dropView() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindDropView)

	p.term(token.KindDrop)
	p.term(token.KindView)

	if p.is(token.KindIf) {
		p.term()
		p.term(token.KindExists)
	}

	if p.isAnyOf(token.KindIdentifier, token.KindTemp) && p.isPos(1, token.KindDot) {
		p.termKind(parsetree.KindSchemaName)
		p.term(token.KindDot)
	}
	p.termKind(parsetree.KindViewName, token.KindIdentifier)

	return p.popTree()
}

func (p *Parser) expressionList() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindCommaList)

	p.addChild(p.expression())
	for p.is(token.KindComma) {
		p.term()
		p.addChild(p.expression())
	}

	return p.popTree()
}

// expression parses a expression.
func (p *Parser) expression() parsetree.NonTerminal {
	if !p.isStartOfExpression() {
		p.errorUnexpected()
	}
	p.pushTreeKind(parsetree.KindExpression)
	p.addChild(p.expression1())
	return p.popTree()
}

// expression1 parses a expression with precedence at least 1 (1 is the lowest precedence).
func (p *Parser) expression1() parsetree.Construction {
	p.pushTree(p.expression2())

	for p.is(token.KindOr) {
		exp := p.popConstr()
		p.pushTreeKind(parsetree.KindOr)
		p.addChild(exp)

		p.term(token.KindOr)

		p.addChild(p.expression2())
	}

	return p.popConstr()
}

// expression2 parses a expression with precedence at least 2.
func (p *Parser) expression2() parsetree.Construction {
	p.pushTree(p.expression3())

	for p.is(token.KindAnd) {
		exp := p.popConstr()
		p.pushTreeKind(parsetree.KindAnd)
		p.addChild(exp)

		p.term(token.KindAnd)

		p.addChild(p.expression3())
	}

	return p.popConstr()
}

// expression3 parses a expression with precedence at least 3.
func (p *Parser) expression3() parsetree.Construction {
	if p.is(token.KindNot) && !p.isPos(1, token.KindExists) {
		return p.expression3WithNot()
	}

	p.expression4()
	return p.popConstr()
}

func (p *Parser) isStartOfExpression3WithNot() bool {
	return p.is(token.KindNot) && !p.isPos(1, token.KindExists)
}

func (p *Parser) expression3WithNot() parsetree.Construction {
	p.pushTreeKind(parsetree.KindNot)
	p.term(token.KindNot)
	p.addChild(p.expression3())
	return p.popConstr()
}

func (p *Parser) expression4List() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindCommaList)

	p.expression4()
	p.addChild(p.popConstr())
	for p.is(token.KindComma) {
		p.term()
		p.expression4()
		p.addChild(p.popConstr())
	}

	return p.popTree()
}

// expression4 parses a expression with precedence at least 4.
func (p *Parser) expression4() {
	p.pushTree(p.expression5())

	for {
		if p.isAnyOf(token.KindEqual, token.KindEqualEqual) {
			p.equal()
		} else if p.isAnyOf(token.KindLessThanGreaterThan, token.KindExclamationEqual) {
			p.notEqual()
		} else if p.is(token.KindIs) {
			p.parseIsExpressions()
		} else if p.is(token.KindBetween) {
			p.between()
		} else if p.is(token.KindIn) {
			p.in()
		} else if p.is(token.KindGlob) {
			p.glob()
		} else if p.is(token.KindRegexp) {
			p.regexp()
		} else if p.is(token.KindMatch) {
			p.match()
		} else if p.is(token.KindLike) {
			p.like()
		} else if p.is(token.KindIsnull) {
			p.parseIsnull()
		} else if p.is(token.KindNotnull) {
			p.notnull()
		} else if p.is(token.KindNot) {
			p.expression4Not()
		} else {
			return
		}
	}
}

func (p *Parser) expression4Not() {
	if p.isPos(1, token.KindBetween) {
		p.notBetween()
	} else if p.isPos(1, token.KindIn) {
		p.notIn()
	} else if p.isPos(1, token.KindGlob) {
		p.notGlob()
	} else if p.isPos(1, token.KindRegexp) {
		p.notRegexp()
	} else if p.isPos(1, token.KindMatch) {
		p.notMatch()
	} else if p.isPos(1, token.KindLike) {
		p.notLike()
	} else if p.isPos(1, token.KindNull) {
		p.notNull()
	}
}

func (p *Parser) equal() {
	t := p.popConstr()
	p.pushTreeKind(parsetree.KindEqual)
	p.addChild(t)
	p.term(token.KindEqual, token.KindEqualEqual)

	if p.isStartOfExpression3WithNot() {
		p.addChild(p.expression3WithNot())
	} else {
		p.addChild(p.expression5())
	}
}

func (p *Parser) notEqual() {
	t := p.popConstr()
	p.pushTreeKind(parsetree.KindNotEqual)
	p.addChild(t)
	p.term(token.KindLessThanGreaterThan, token.KindExclamationEqual)

	if p.isStartOfExpression3WithNot() {
		p.addChild(p.expression3WithNot())
	} else {
		p.addChild(p.expression5())
	}
}

// parseIsExpressions parses a is, is not, is distinct, and is not distinct expression.
func (p *Parser) parseIsExpressions() {
	if p.isSeq(token.KindIs, token.KindNot, token.KindDistinct) {
		p.parseIsNotDistinct()
	} else if p.isSeq(token.KindIs, token.KindNot) {
		p.parseIsNot()
	} else if p.isSeq(token.KindIs, token.KindDistinct) {
		p.parseIsDistinct()
	} else {
		p.parseIs()
	}
}

func (p *Parser) parseIsNot() {
	t := p.popConstr()
	p.pushTreeKind(parsetree.KindIsNot)
	p.addChild(t)

	p.term(token.KindIs)
	p.term(token.KindNot)

	if p.isStartOfExpression3WithNot() {
		p.addChild(p.expression3WithNot())
	} else {
		p.addChild(p.expression5())
	}
}

func (p *Parser) parseIsNotDistinct() {
	t := p.popConstr()
	p.pushTreeKind(parsetree.KindIsNotDistinctFrom)
	p.addChild(t)

	p.term(token.KindIs)
	p.term(token.KindNot)
	p.term(token.KindDistinct)
	p.term(token.KindFrom)

	if p.isStartOfExpression3WithNot() {
		p.addChild(p.expression3WithNot())
	} else {
		p.addChild(p.expression5())
	}
}

func (p *Parser) parseIsDistinct() {
	t := p.popConstr()
	p.pushTreeKind(parsetree.KindIsDistinctFrom)
	p.addChild(t)

	p.term(token.KindIs)
	p.term(token.KindDistinct)
	p.term(token.KindFrom)

	if p.isStartOfExpression3WithNot() {
		p.addChild(p.expression3WithNot())
	} else {
		p.addChild(p.expression5())
	}
}

func (p *Parser) parseIs() {
	t := p.popConstr()
	p.pushTreeKind(parsetree.KindIs)
	p.addChild(t)

	p.term(token.KindIs)
	p.addChild(p.expression5())
}

// between parses a between expression.
func (p *Parser) between() {
	t := p.popConstr()
	p.pushTreeKind(parsetree.KindBetween)
	p.addChild(t)

	p.addBetween()
}

func (p *Parser) addBetween() {
	p.term(token.KindBetween)
	p.addChild(p.expression3())
	p.term(token.KindAnd)
	p.addChild(p.expression3())
}

// notBetween parses a not between expression.
func (p *Parser) notBetween() {
	t := p.popConstr()
	p.pushTreeKind(parsetree.KindNotBetween)
	p.addChild(t)

	p.term(token.KindNot)
	p.addBetween()
}

// in parses a in expression.
func (p *Parser) in() {
	t := p.popConstr()
	p.pushTreeKind(parsetree.KindIn)
	p.addChild(t)

	p.addIn()
}

func (p *Parser) addIn() {
	p.term(token.KindIn)

	if p.is(token.KindLeftParen) {
		p.term()

		if p.isAnyOf(token.KindWith, token.KindSelect) {
			var withClause parsetree.NonTerminal
			if p.is(token.KindWith) {
				withClause = p.withClause()
			}
			p.addChild(p.selectStatement(withClause))
		} else if p.isStartOfExpression() {
			p.addChild(p.expressionList())
		}

		p.term(token.KindRightParen)
		return
	}

	if p.isAnyOf(token.KindIdentifier, token.KindTemp) && p.isPos(1, token.KindDot) {
		p.termKind(parsetree.KindSchemaName)
		p.term(token.KindDot)
	}

	if p.isSeq(token.KindIdentifier, token.KindLeftParen) {
		p.termKind(parsetree.KindTableFunctionName, token.KindIdentifier)
		p.term(token.KindLeftParen)
		if p.isStartOfExpressionAtLeast4() {
			p.addChild(p.expression4List())
		}
		p.term(token.KindRightParen)
	} else {
		p.termKind(parsetree.KindTableName, token.KindIdentifier)
	}
}

// notIn parses a not in expression.
func (p *Parser) notIn() {
	t := p.popConstr()
	p.pushTreeKind(parsetree.KindNotIn)
	p.addChild(t)

	p.term(token.KindNot)
	p.addIn()
}

func (p *Parser) glob() {
	t := p.popConstr()
	p.pushTreeKind(parsetree.KindGlob)
	p.addChild(t)

	p.term(token.KindGlob)

	if p.isStartOfExpression3WithNot() {
		p.addChild(p.expression3WithNot())
	} else {
		p.addChild(p.expression5())
	}
}

func (p *Parser) notGlob() {
	t := p.popConstr()
	p.pushTreeKind(parsetree.KindNotGlob)
	p.addChild(t)

	p.term(token.KindNot)
	p.term(token.KindGlob)

	if p.isStartOfExpression3WithNot() {
		p.addChild(p.expression3WithNot())
	} else {
		p.addChild(p.expression5())
	}
}

func (p *Parser) regexp() {
	t := p.popConstr()
	p.pushTreeKind(parsetree.KindRegexp)
	p.addChild(t)

	p.term(token.KindRegexp)

	if p.isStartOfExpression3WithNot() {
		p.addChild(p.expression3WithNot())
	} else {
		p.addChild(p.expression5())
	}
}

func (p *Parser) notRegexp() {
	t := p.popConstr()
	p.pushTreeKind(parsetree.KindNotRegexp)
	p.addChild(t)

	p.term(token.KindNot)
	p.term(token.KindRegexp)

	if p.isStartOfExpression3WithNot() {
		p.addChild(p.expression3WithNot())
	} else {
		p.addChild(p.expression5())
	}
}

func (p *Parser) match() {
	t := p.popConstr()
	p.pushTreeKind(parsetree.KindMatch)
	p.addChild(t)

	p.term(token.KindMatch)

	if p.isStartOfExpression3WithNot() {
		p.addChild(p.expression3WithNot())
	} else {
		p.addChild(p.expression5())
	}
}

func (p *Parser) notMatch() {
	t := p.popConstr()
	p.pushTreeKind(parsetree.KindNotMatch)
	p.addChild(t)

	p.term(token.KindNot)
	p.term(token.KindMatch)

	if p.isStartOfExpression3WithNot() {
		p.addChild(p.expression3WithNot())
	} else {
		p.addChild(p.expression5())
	}
}

func (p *Parser) like() {
	t := p.popConstr()
	p.pushTreeKind(parsetree.KindLike)
	p.addChild(t)

	p.term(token.KindLike)

	if p.isStartOfExpression3WithNot() {
		p.addChild(p.expression3WithNot())
	} else {
		p.addChild(p.expression5())
	}

	// apparently there is an error in the ESCAPE precedence documentation
	if p.is(token.KindEscape) {
		p.term()

		if p.isStartOfExpression3WithNot() {
			p.addChild(p.expression3WithNot())
		} else {
			p.addChild(p.expression5())
		}
	}
}

func (p *Parser) notLike() {
	t := p.popConstr()
	p.pushTreeKind(parsetree.KindNotLike)
	p.addChild(t)

	p.term(token.KindNot)
	p.term(token.KindLike)

	if p.isStartOfExpression3WithNot() {
		p.addChild(p.expression3WithNot())
	} else {
		p.addChild(p.expression5())
	}

	// apparently there is an error in the ESCAPE precedence documentation
	if p.is(token.KindEscape) {
		p.term()

		if p.isStartOfExpression3WithNot() {
			p.addChild(p.expression3WithNot())
		} else {
			p.addChild(p.expression5())
		}
	}
}

func (p *Parser) parseIsnull() {
	t := p.popConstr()
	p.pushTreeKind(parsetree.KindIsnull)
	p.addChild(t)

	p.term(token.KindIsnull)
}

func (p *Parser) notnull() {
	t := p.popConstr()
	p.pushTreeKind(parsetree.KindNotnull)
	p.addChild(t)

	p.term(token.KindNotnull)
}

func (p *Parser) notNull() {
	t := p.popConstr()
	p.pushTreeKind(parsetree.KindNotNull)
	p.addChild(t)

	p.term(token.KindNot)
	p.term(token.KindNull)
}

// isStartOfExpressionAtLeast4 reports whether p.tok[0] is a start of expression of precedence at least 4.
func (p *Parser) isStartOfExpressionAtLeast4() bool {
	return p.isStartOfExpression() && !p.is(token.KindNot)
}

// expression5 parses a expression with precedence at least 5.
func (p *Parser) expression5() parsetree.Construction {
	p.pushTree(p.expression6())

	for {
		t := p.popConstr()
		if p.is(token.KindLessThan) {
			p.pushTreeKind(parsetree.KindLessThan)
		} else if p.is(token.KindLessThanOrEqual) {
			p.pushTreeKind(parsetree.KindLessThanOrEqual)
		} else if p.is(token.KindGreaterThan) {
			p.pushTreeKind(parsetree.KindGreaterThan)
		} else if p.is(token.KindGreaterThanOrEqual) {
			p.pushTreeKind(parsetree.KindGreaterThanOrEqual)
		} else {
			return t
		}

		p.addChild(t)

		p.term()
		p.addChild(p.expression6())
	}
}

// expression6 parses a expression with precedence at least 6.
func (p *Parser) expression6() parsetree.Construction {
	// apparently there is an error in the ESCAPE precedence documentation.

	p.pushTree(p.expression7())
	for {
		t := p.popConstr()
		if p.is(token.KindAmpersand) {
			p.pushTreeKind(parsetree.KindBitAnd)
		} else if p.is(token.KindPipe) {
			p.pushTreeKind(parsetree.KindBitOr)
		} else if p.is(token.KindLessThanLessThan) {
			p.pushTreeKind(parsetree.KindLeftShift)
		} else if p.is(token.KindGreaterThanGreaterThan) {
			p.pushTreeKind(parsetree.KindRightShift)
		} else {
			return t
		}
		p.addChild(t)

		p.term()
		p.addChild(p.expression7())
	}
}

// expression7 parses a expression with precedence at least 7.
func (p *Parser) expression7() parsetree.Construction {
	p.pushTree(p.expression8())
	for {
		t := p.popConstr()
		if p.is(token.KindPlus) {
			p.pushTreeKind(parsetree.KindAdd)
		} else if p.is(token.KindMinus) {
			p.pushTreeKind(parsetree.KindSubtract)
		} else {
			return t
		}
		p.addChild(t)

		p.term()
		p.addChild(p.expression8())
	}
}

// expression8 parses a expression with precedence at least 8.
func (p *Parser) expression8() parsetree.Construction {
	p.pushTree(p.expression9())
	for {
		t := p.popConstr()
		if p.is(token.KindAsterisk) {
			p.pushTreeKind(parsetree.KindMultiply)
		} else if p.is(token.KindSlash) {
			p.pushTreeKind(parsetree.KindDivide)
		} else if p.is(token.KindPercent) {
			p.pushTreeKind(parsetree.KindMod)
		} else {
			return t
		}
		p.addChild(t)

		p.term()
		p.addChild(p.expression9())
	}
}

// expression9 parses a expression with precedence at least 9.
func (p *Parser) expression9() parsetree.Construction {
	p.pushTree(p.expression10())
	for {
		t := p.popConstr()
		if p.is(token.KindPipePipe) {
			p.pushTreeKind(parsetree.KindConcatenate)
		} else if p.is(token.KindMinusGreaterThan) {
			p.pushTreeKind(parsetree.KindExtract1)
		} else if p.is(token.KindMinusGreaterThanGreaterThan) {
			p.pushTreeKind(parsetree.KindExtract2)
		} else {
			return t
		}
		p.addChild(t)

		p.term()
		p.addChild(p.expression10())
	}
}

// expression10 parses a expression with precedence at least 10.
func (p *Parser) expression10() parsetree.Construction {
	p.pushTree(p.expression11())

	if !p.is(token.KindCollate) {
		return p.popConstr()
	}

	t := p.popConstr()
	p.pushTreeKind(parsetree.KindCollate)
	p.addChild(t)

	p.term(token.KindCollate)
	p.termKind(parsetree.KindCollationName, token.KindIdentifier)
	return p.popConstr()
}

// expression11 parses a expression with precedence at least 11.
func (p *Parser) expression11() parsetree.Construction {

	var k parsetree.Kind
	if p.is(token.KindTilde) {
		k = parsetree.KindBitNot
	} else if p.is(token.KindPlus) {
		k = parsetree.KindPrefixPlus
	} else if p.is(token.KindMinus) {
		k = parsetree.KindNegate
	} else {
		return p.simpleExpression()
	}

	p.pushTreeKind(k)
	p.term()
	p.addChild(p.expression11())

	return p.popConstr()
}

// simpleExpression parses a simple expression, that is, a expression with the highest precedence.
func (p *Parser) simpleExpression() parsetree.Construction {
	if p.isLiteralValue() {
		t := parsetree.NewTerminal(parsetree.KindToken, p.tok[0])
		p.advance()
		return t
	} else if p.isBindParameter() {
		t := parsetree.NewTerminal(parsetree.KindBindParameter, p.tok[0])
		p.advance()
		return t
	} else if p.isSeq(token.KindIdentifier, token.KindLeftParen) {
		return p.functionCall()
	} else if p.isAnyOf(token.KindIdentifier, token.KindTemp) {
		return p.columnReference()
	} else if p.is(token.KindLeftParen) {
		return p.parenExpression()
	} else if p.is(token.KindCast) {
		return p.castExpression()
	} else if p.is(token.KindNot) {
		// can only be NOT EXISTS, see expression3
		return p.notExists()
	} else if p.is(token.KindExists) {
		return p.exists()
	} else if p.is(token.KindCase) {
		return p.caseExpression()
	} else if p.is(token.KindRaise) {
		return p.raise()
	} else {
		p.errorUnexpected()
		return nil
	}
}

// isStartOfExpression reports whether p.tok[0] is a start of expression.
func (p *Parser) isStartOfExpression() bool {
	if p.isLiteralValue() {
		return true
	}
	if p.isBindParameter() {
		return true
	}

	return p.isAnyOf(token.KindIdentifier, token.KindTemp, token.KindTilde, token.KindPlus,
		token.KindMinus, token.KindNot, token.KindLeftParen, token.KindCast, token.KindExists,
		token.KindCase, token.KindRaise)
}

// isLiteralValue reports whether p.tok[0] is a literal value.
func (p *Parser) isLiteralValue() bool {
	if p.isAnyOf(token.KindNumeric, token.KindString, token.KindBlob,
		token.KindNull, token.KindCurrentTime, token.KindCurrentDate,
		token.KindCurrentTimestamp, token.KindRowId,
	) {
		return true
	}

	if !p.is(token.KindIdentifier) {
		return false
	}

	lex := strings.ToLower(string(p.tok[0].Lexeme))
	return lex == "true" || lex == "false"
}

// isBindParameter reports whether p.tok[0] is a bind parameter.
func (p *Parser) isBindParameter() bool {
	return p.isAnyOf(token.KindAtVariable, token.KindColonVariable, token.KindDollarVariable, token.KindQuestionVariable)
}

// columnReference parses a column reference.
func (p *Parser) columnReference() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindColumnReference)

	if p.isAnyOfPos(0, token.KindIdentifier, token.KindTemp) && p.isPos(1, token.KindDot) &&
		p.isAnyOfPos(2, token.KindIdentifier, token.KindTemp) && p.isPos(3, token.KindDot) {
		p.termKind(parsetree.KindSchemaName, token.KindIdentifier, token.KindTemp)
		p.term(token.KindDot)
		p.termKind(parsetree.KindTableName, token.KindIdentifier, token.KindTemp)
		p.term(token.KindDot)
		p.termKind(parsetree.KindColumnName, token.KindIdentifier, token.KindTemp)
	} else if p.isAnyOfPos(0, token.KindIdentifier, token.KindTemp) && p.isPos(1, token.KindDot) {
		p.termKind(parsetree.KindTableName, token.KindIdentifier, token.KindTemp)
		p.term(token.KindDot)
		p.termKind(parsetree.KindColumnName, token.KindIdentifier, token.KindTemp)
	} else {
		p.termKind(parsetree.KindColumnName, token.KindIdentifier, token.KindTemp)
	}

	return p.popTree()
}

// functionCall parses a function call.
func (p *Parser) functionCall() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindFunctionCall)

	p.termKind(parsetree.KindFunctionName, token.KindIdentifier)
	p.term(token.KindLeftParen)
	p.addChild(p.functionArguments())
	p.term(token.KindRightParen)

	if p.is(token.KindFilter) {
		p.addChild(p.filterClause())
	}

	if p.is(token.KindOver) {
		p.addChild(p.overClause())
	}

	return p.popTree()
}

// functionCall parses the arguments of a function call.
func (p *Parser) functionArguments() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindFunctionArguments)

	if !p.is(token.KindDistinct) && !p.isStartOfExpression() && !p.is(token.KindAsterisk) {
		return p.popTree()
	}

	if p.is(token.KindAsterisk) {
		p.term()
		return p.popTree()
	}

	if p.is(token.KindDistinct) {
		p.term()
	}
	p.addChild(p.expressionList())
	if p.is(token.KindOrder) {
		p.addChild(p.orderByClause())
	}

	return p.popTree()
}

// orderByClause parses an order by clause.
func (p *Parser) orderByClause() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindOrderByClause)

	p.term(token.KindOrder)
	p.term(token.KindBy)
	p.addChild(p.orderingTermList())

	return p.popTree()
}

func (p *Parser) orderingTermList() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindCommaList)

	p.addChild(p.orderingTerm())
	for p.is(token.KindComma) {
		p.term()
		p.addChild(p.orderingTerm())
	}

	return p.popTree()
}

// orderingTerm parses an ordering term.
func (p *Parser) orderingTerm() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindOrderingTerm)
	p.addChild(p.expression())

	if p.isAnyOf(token.KindAsc, token.KindDesc) {
		p.term()
	}

	if p.is(token.KindNulls) {
		p.term()
		p.term(token.KindFirst, token.KindLast)
	}

	return p.popTree()
}

// filterClause parses a filter clause.
func (p *Parser) filterClause() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindFilterClause)

	p.term(token.KindFilter)
	p.term(token.KindLeftParen)
	p.term(token.KindWhere)
	p.addChild(p.expression())
	p.term(token.KindRightParen)

	return p.popTree()
}

// overClause parses a over clause.
func (p *Parser) overClause() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindOverClause)

	p.term(token.KindOver)

	if p.is(token.KindIdentifier) {
		p.termKind(parsetree.KindWindowName, token.KindIdentifier)
	} else {
		p.addChild(p.windowDefinition())
	}

	return p.popTree()
}

// windowDefinition parses a window definition.
func (p *Parser) windowDefinition() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindWindowDefinition)

	p.term(token.KindLeftParen)

	if p.is(token.KindIdentifier) {
		p.termKind(parsetree.KindWindowName)
	}

	if p.is(token.KindPartition) {
		p.addChild(p.partitionBy())
	}

	if p.is(token.KindOrder) {
		p.addChild(p.orderByClause())
	}

	if p.isAnyOf(token.KindRange, token.KindRows, token.KindGroups) {
		p.addChild(p.frameSpec())
	}

	p.term(token.KindRightParen)

	return p.popTree()
}

func (p *Parser) partitionBy() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindPartitionBy)
	p.term(token.KindPartition)
	p.term(token.KindBy)
	p.addChild(p.expressionList())
	return p.popTree()
}

// frameSpec parses a frame spec clause.
func (p *Parser) frameSpec() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindFrameSpec)

	p.term(token.KindRange, token.KindRows, token.KindGroups)

	if p.is(token.KindBetween) {
		p.addChild(p.frameSpecBetween())
	} else if p.is(token.KindUnbounded) {
		p.term()
		p.term(token.KindPreceding)
	} else if p.is(token.KindCurrent) {
		p.term()
		p.term(token.KindRow)
	} else if p.isStartOfExpression() {
		p.addChild(p.expression())
		p.term(token.KindPreceding)
	} else {
		p.errorUnexpected()
	}

	if p.is(token.KindExclude) {
		p.term()

		switch p.token(token.KindNo, token.KindCurrent, token.KindGroup, token.KindTies) {
		case token.KindNo:
			p.term()
			p.term(token.KindOthers)
		case token.KindCurrent:
			p.term()
			p.term(token.KindRow)
		case token.KindGroup:
			p.term()
		default: //token.KindTies
			p.term()
		}
	}

	return p.popTree()
}

// frameSpecBetween parses the between part of a frame spec.
func (p *Parser) frameSpecBetween() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindFrameSpecBetween)
	p.term(token.KindBetween)

	if p.is(token.KindUnbounded) {
		p.term()
		p.term(token.KindPreceding)
	} else if p.is(token.KindCurrent) {
		p.term()
		p.term(token.KindRow)
	} else if p.isStartOfExpressionAtLeast4() {
		p.pushTreeKind(parsetree.KindExpression)
		p.expression4()
		p.addChild(p.popConstr())
		p.addChild(p.popTree())

		p.term(token.KindPreceding, token.KindFollowing)
	} else {
		p.errorUnexpected()
	}

	p.term(token.KindAnd)

	if p.is(token.KindUnbounded) {
		p.term()
		p.term(token.KindFollowing)
	} else if p.is(token.KindCurrent) {
		p.term()
		p.term(token.KindRow)
	} else if p.isStartOfExpressionAtLeast4() {
		p.pushTreeKind(parsetree.KindExpression)
		p.expression4()
		p.addChild(p.popConstr())
		p.addChild(p.popTree())

		p.term(token.KindPreceding, token.KindFollowing)
	} else {
		p.errorUnexpected()
	}

	return p.popTree()
}

// parenExpression parses a expression enclosed in parenthesis.
// BUG: missing subquery
func (p *Parser) parenExpression() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindParenExpression)

	p.term(token.KindLeftParen)
	p.addChild(p.expressionList())
	p.term(token.KindRightParen)

	return p.popTree()
}

// castExpression parses a cast expression.
func (p *Parser) castExpression() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindCast)

	p.term(token.KindCast)
	p.term(token.KindLeftParen)
	p.addChild(p.expression())
	p.term(token.KindAs)
	p.addChild(p.typeName())
	p.term(token.KindRightParen)

	return p.popTree()
}

// exists parses a exists expression.
func (p *Parser) exists() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindExists)

	p.term(token.KindExists)
	p.term(token.KindLeftParen)

	var withClause parsetree.NonTerminal
	if p.is(token.KindWith) {
		withClause = p.withClause()
	}
	p.addChild(p.selectStatement(withClause))

	p.term(token.KindRightParen)

	return p.popTree()
}

func (p *Parser) notExists() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindNot)

	p.term(token.KindNot)
	p.addChild(p.exists())

	return p.popTree()
}

// caseExpression parses a case expression.
func (p *Parser) caseExpression() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindCase)

	p.term(token.KindCase)
	if p.isStartOfExpression() {
		p.addChild(p.expression())
	}

	p.addChild(p.when())
	for p.is(token.KindWhen) {
		p.addChild(p.when())
	}

	if p.is(token.KindElse) {
		p.addChild(p.caseElse())
	}

	p.term(token.KindEnd)

	return p.popTree()
}

// when parses a when part of a case expression.
func (p *Parser) when() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindWhen)

	p.term(token.KindWhen)
	p.addChild(p.expression())
	p.term(token.KindThen)
	p.addChild(p.expression())

	return p.popTree()
}

// caseElse parses an else part of a case expression.
func (p *Parser) caseElse() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindElse)

	p.term(token.KindElse)
	p.addChild(p.expression())

	return p.popTree()
}

// raise parses a raise function call.
func (p *Parser) raise() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindRaise)

	p.term(token.KindRaise)
	p.term(token.KindLeftParen)

	switch p.token(token.KindIgnore, token.KindRollback, token.KindAbort, token.KindFail) {
	case token.KindIgnore:
		p.term()
	default:
		p.term()
		p.term(token.KindComma)
		p.addChild(p.errorMessage())
	}

	p.term(token.KindRightParen)

	return p.popTree()
}

func (p *Parser) errorMessage() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindErrorMessage)
	p.addChild(p.expression())
	return p.popTree()
}

// insert parses a insert statement.
func (p *Parser) insert(withClause parsetree.NonTerminal) parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindInsert)

	if withClause != nil {
		p.addChild(withClause)
	}

	switch p.token(token.KindInsert, token.KindReplace) {
	case token.KindInsert:
		p.term()

		if p.is(token.KindOr) {
			p.term()
			p.term(token.KindAbort, token.KindFail, token.KindIgnore, token.KindReplace, token.KindRollback)
		}
	default:
		p.term(token.KindReplace)
	}

	p.term(token.KindInto)

	if p.isAnyOfPos(0, token.KindIdentifier, token.KindTemp) && p.isPos(1, token.KindDot) {
		p.termKind(parsetree.KindSchemaName, token.KindIdentifier, token.KindTemp)
		p.term(token.KindDot)
	}

	p.termKind(parsetree.KindTableName, token.KindIdentifier, token.KindTemp)

	if p.is(token.KindAs) {
		p.term()
		p.termKind(parsetree.KindTableAlias, token.KindIdentifier)
	}

	if p.is(token.KindLeftParen) {
		p.term()
		p.addChild(p.columnNameList())
		p.term(token.KindRightParen)
	}

	switch p.token(token.KindValues, token.KindWith, token.KindSelect, token.KindDefault) {
	case token.KindValues:
		p.term()
		p.addChild(p.insertValuesList())
		if p.is(token.KindOn) {
			p.addChild(p.upsertClause())
		}
	case token.KindWith, token.KindSelect:
		var withClause parsetree.NonTerminal
		if p.tok[0].Kind == token.KindWith {
			withClause = p.withClause()
		}
		p.addChild(p.selectStatement(withClause))

		if p.is(token.KindOn) {
			p.addChild(p.upsertClause())
		}
	default:
		p.term(token.KindDefault)
		p.term(token.KindValues)
	}

	if p.is(token.KindReturning) {
		p.addChild(p.returningClause())
	}

	return p.popTree()
}

// insertValues parses the values list of a insert.
func (p *Parser) insertValuesList() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindInsertValuesList)
	p.addChild(p.insertValuesItemList())
	return p.popTree()
}

func (p *Parser) insertValuesItemList() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindCommaList)

	p.addChild(p.insertValuesItem())
	for p.is(token.KindComma) {
		p.term(token.KindComma)
		p.addChild(p.insertValuesItem())
	}

	return p.popTree()
}

// insertValuesItem parses a item in a insert values list.
func (p *Parser) insertValuesItem() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindInsertValuesItem)

	p.term(token.KindLeftParen)
	p.addChild(p.expressionList())
	p.term(token.KindRightParen)

	return p.popTree()
}

// upsertClause parses a upsert clause.
func (p *Parser) upsertClause() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindUpsertClause)

	p.addChild(p.upsertClauseItem())
	for p.is(token.KindOn) {
		p.addChild(p.upsertClauseItem())
	}

	return p.popTree()
}

// upsertClauseItem parses a item of an upsert clause.
func (p *Parser) upsertClauseItem() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindUpsertClauseItem)

	p.term(token.KindOn)
	p.term(token.KindConflict)

	if p.is(token.KindLeftParen) {
		p.term()
		p.addChild(p.indexedColumnList(true))
		p.term(token.KindRightParen)

		if p.is(token.KindWhere) {
			p.addChild(p.whereClause())
		}
	}

	p.term(token.KindDo)

	switch p.token(token.KindNothing, token.KindUpdate) {
	case token.KindNothing:
		p.term()
	default:
		p.term(token.KindUpdate)
		p.term(token.KindSet)
		p.addChild(p.updateSetItemList())

		if p.is(token.KindWhere) {
			p.addChild(p.whereClause())
		}
	}

	return p.popTree()
}

func (p *Parser) updateSetItemList() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindCommaList)

	for {
		p.addChild(p.updateSetItem())
		if !p.is(token.KindComma) {
			break
		}
		p.term(token.KindComma)
	}

	return p.popTree()
}

// updateSetItem parses a set item in a upsert or update clause.
func (p *Parser) updateSetItem() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindUpdateSetItem)

	switch p.token(token.KindIdentifier, token.KindLeftParen) {
	case token.KindIdentifier:
		p.termKind(parsetree.KindColumnName)
	default: //token.KindLeftParen
		p.term()
		p.addChild(p.columnNameList(token.KindRightParen, token.KindEqual, token.KindSemicolon, token.KindEOF))
		p.term(token.KindRightParen)
	}

	p.term(token.KindEqual)
	p.addChild(p.expression())

	return p.popTree()
}

func (p *Parser) with() parsetree.NonTerminal {
	withClause := p.withClause()
	switch p.token(token.KindDelete, token.KindInsert, token.KindReplace, token.KindSelect, token.KindUpdate) {
	case token.KindDelete:
		return p.delete(withClause)
	case token.KindInsert, token.KindReplace:
		return p.insert(withClause)
	case token.KindSelect:
		return p.selectStatement(withClause)
	default: // token.KindUpdate:
		return p.update(withClause)
	}
}

// pragma parses a pragma statement.
func (p *Parser) pragma() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindPragma)

	p.term(token.KindPragma)

	if p.isAnyOfPos(0, token.KindIdentifier, token.KindTemp) && p.isPos(1, token.KindDot) {
		p.termKind(parsetree.KindSchemaName, token.KindIdentifier, token.KindTemp)
		p.term(token.KindDot)
	}
	p.termKind(parsetree.KindPragmaName, token.KindIdentifier, token.KindTemp)

	if p.is(token.KindEqual) {
		p.term()
		p.addChild(p.pragmaValue())
	} else if p.is(token.KindLeftParen) {
		p.term()
		p.addChild(p.pragmaValue())
		p.term(token.KindRightParen)
	}

	return p.popTree()
}

// pragmaValue parses a pragma value.
func (p *Parser) pragmaValue() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindPragmaValue)

	if p.isAnyOf(token.KindPlus, token.KindMinus) {
		p.term()
	}

	if p.isAnyOf(token.KindNumeric, token.KindIdentifier, token.KindString) || p.isKeyword() {
		p.term()
	} else {
		p.errorUnexpected()
	}

	return p.popTree()
}

// reindex parses a reindex statement.
func (p *Parser) reindex() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindReindex)

	p.term(token.KindReindex)

	if p.isAnyOf(token.KindIdentifier, token.KindTemp) && p.isPos(1, token.KindDot) {
		p.termKind(parsetree.KindSchemaName)
		p.term(token.KindDot)
		p.termKind(parsetree.KindTableOrIndexName, token.KindIdentifier)
	} else if p.is(token.KindIdentifier) {
		p.termKind(parsetree.KindCollationTableOrIndexName, token.KindIdentifier)
	}

	return p.popTree()
}

// release parses a release statement.
func (p *Parser) release() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindRelease)
	p.term(token.KindRelease)

	if p.is(token.KindSavepoint) {
		p.term()
	}

	p.termKind(parsetree.KindSavepointName, token.KindIdentifier)

	return p.popTree()
}

// savepoint parses a savepoint statement.
func (p *Parser) savepoint() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindSavepoint)
	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	p.advance()

	if p.tok[0].Kind == token.KindIdentifier {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindSavepointName, p.tok[0]))
		p.advance()
	} else {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing savepoint name`)))
	}

	return nt
}

// selectStatement parses a select statement.
func (p *Parser) selectStatement(withClause parsetree.NonTerminal) parsetree.NonTerminal {
	var nt parsetree.NonTerminal

	sc := p.selectCore()
	if p.tok[0].Kind == token.KindUnion || p.tok[0].Kind == token.KindIntersect || p.tok[0].Kind == token.KindExcept {
		nt = parsetree.NewNonTerminal(parsetree.KindCompoundSelect)
		if withClause != nil {
			nt.AddChild(withClause)
		}

		nt.AddChild(sc)
		for p.tok[0].Kind == token.KindUnion || p.tok[0].Kind == token.KindIntersect || p.tok[0].Kind == token.KindExcept {
			nt.AddChild(p.compoundOperator())
			if p.tok[0].Kind == token.KindSelect || p.tok[0].Kind == token.KindValues {
				nt.AddChild(p.selectCore())
			} else {
				nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing select statement`)))
				break
			}
		}
	} else {
		nt = parsetree.NewNonTerminal(parsetree.KindSimpleSelect)
		if withClause != nil {
			nt.AddChild(withClause)
		}

		nt.AddChild(sc)
	}

	if p.tok[0].Kind == token.KindOrder {
		nt.AddChild(p.orderByClause())
	}

	if p.tok[0].Kind == token.KindLimit {
		nt.AddChild(p.limitClause())
	}

	return nt
}

// selectCore parses a select core.
func (p *Parser) selectCore() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindSelectCore)
	if p.tok[0].Kind == token.KindSelect {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		if p.tok[0].Kind == token.KindAll || p.tok[0].Kind == token.KindDistinct {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		}

		if p.isStartOfExpression() || p.tok[0].Kind == token.KindAsterisk {
			nt.AddChild(p.resultColumnList())
		} else {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing result column`)))
		}

		if p.tok[0].Kind == token.KindFrom {
			nt.AddChild(p.fromClause())
		}

		if p.tok[0].Kind == token.KindWhere {
			nt.AddChild(p.whereClause())
		}

		if p.tok[0].Kind == token.KindGroup {
			nt.AddChild(p.groupByClause())
		}

		if p.tok[0].Kind == token.KindHaving {
			nt.AddChild(p.havingClause())
		}

		if p.tok[0].Kind == token.KindWindow {
			nt.AddChild(p.windowClause())
		}
	} else { // VALUES
		nt.AddChild(p.valuesClause())
	}

	return nt
}

// resultColumnList parses a list of result columns.
func (p *Parser) resultColumnList() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindCommaList)

	nt.AddChild(p.resultColumn())

	for p.tok[0].Kind == token.KindComma {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		if p.isStartOfExpression() || p.tok[0].Kind == token.KindAsterisk {
			nt.AddChild(p.resultColumn())
		} else {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing result column`)))
		}
	}

	return nt
}

// resultColumn parses a result column of a select statement.
func (p *Parser) resultColumn() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindResultColumn)
	if p.tok[0].Kind == token.KindAsterisk {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else if p.tok[0].Kind == token.KindIdentifier && p.tok[1].Kind == token.KindDot && p.tok[2].Kind == token.KindAsterisk {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindTableName, p.tok[0]))
		p.advance()
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else if p.isStartOfExpression() {
		nt.AddChild(p.expression())
		if p.tok[0].Kind == token.KindAs {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()

			if p.tok[0].Kind == token.KindIdentifier {
				nt.AddChild(parsetree.NewTerminal(parsetree.KindColumnAlias, p.tok[0]))
				p.advance()
			} else {
				nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing column alias`)))
			}
		} else if p.tok[0].Kind == token.KindIdentifier {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindColumnAlias, p.tok[0]))
			p.advance()
		}
	} else {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorExpecting, errors.New(`expecting *, identifier, or expression`)))
	}
	return nt
}

// fromClause parses a from clause.
func (p *Parser) fromClause() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindFromClause)
	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	p.advance()

	if p.tok[0].Kind == token.KindIdentifier || p.tok[0].Kind == token.KindLeftParen {
		nt.AddChild(p.joinClause())
	} else {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorExpecting, errors.New(`expecting identifier, or "("`)))
	}

	return nt
}

// tableOrSubquery parses a table-or-subquery clause.
func (p *Parser) tableOrSubquery() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindTableOrSubquery)

	if p.tok[0].Kind == token.KindIdentifier || p.tok[0].Kind == token.KindTemp || p.tok[0].Kind == token.KindDot {
		p.tableOrSubquery_table(nt)
	} else if p.tok[0].Kind == token.KindLeftParen {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		if p.tok[0].Kind == token.KindWith || p.tok[0].Kind == token.KindSelect {
			nt.AddChild(p.selectStatement(nil))

			if p.tok[0].Kind == token.KindRightParen {
				nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
				p.advance()
			} else {
				nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing ")"`)))
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
			} else if p.tok[0].Kind == token.KindIdentifier {
				nt.AddChild(parsetree.NewTerminal(parsetree.KindTableAlias, p.tok[0]))
				p.advance()
			}
		} else if p.tok[0].Kind == token.KindIdentifier || p.tok[0].Kind == token.KindTemp || p.tok[0].Kind == token.KindLeftParen {
			nt.AddChild(p.joinClause())

			if p.tok[0].Kind == token.KindRightParen {
				nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
				p.advance()
			} else {
				nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing ")"`)))
			}
		} else {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorExpecting, errors.New(`expecting select statement, schema name, table name, or "("`)))

			if p.tok[0].Kind == token.KindRightParen {
				nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
				p.advance()
			}
		}
	}

	return nt
}

// tableOrSubquery_table parses a table-or-subquery that is a table.
func (p *Parser) tableOrSubquery_table(tableOrSubquery parsetree.NonTerminal) {
	if p.tok[0].Kind == token.KindIdentifier || p.tok[0].Kind == token.KindTemp {
		if p.tok[1].Kind == token.KindDot {
			tableOrSubquery.AddChild(parsetree.NewTerminal(parsetree.KindSchemaName, p.tok[0]))
			p.advance()
			tableOrSubquery.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		}
	} else if p.tok[0].Kind == token.KindDot {
		tableOrSubquery.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing schema name`)))
		tableOrSubquery.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	}

	if p.tok[0].Kind == token.KindIdentifier {
		if p.tok[1].Kind == token.KindLeftParen {
			tableOrSubquery.AddChild(parsetree.NewTerminal(parsetree.KindTableFunctionName, p.tok[0]))
			p.advance()
		} else {
			tableOrSubquery.AddChild(parsetree.NewTerminal(parsetree.KindTableName, p.tok[0]))
			p.advance()
		}
	} else {
		tableOrSubquery.AddChild(parsetree.NewError(parsetree.KindErrorExpecting, errors.New(`expecting table name, or table-function name`)))
	}

	if p.tok[0].Kind == token.KindAs || p.tok[0].Kind == token.KindIdentifier ||
		p.tok[0].Kind == token.KindIndexed || p.tok[0].Kind == token.KindNot {
		if p.tok[0].Kind == token.KindAs {
			tableOrSubquery.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()

			if p.tok[0].Kind == token.KindIdentifier {
				tableOrSubquery.AddChild(parsetree.NewTerminal(parsetree.KindTableAlias, p.tok[0]))
				p.advance()
			} else {
				tableOrSubquery.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing table alias`)))
			}
		} else if p.tok[0].Kind == token.KindIdentifier {
			tableOrSubquery.AddChild(parsetree.NewTerminal(parsetree.KindTableAlias, p.tok[0]))
			p.advance()
		}

		if p.tok[0].Kind == token.KindIndexed {
			tableOrSubquery.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()

			if p.tok[0].Kind == token.KindBy {
				tableOrSubquery.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
				p.advance()
			} else if p.tok[0].Kind == token.KindIdentifier {
				tableOrSubquery.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "BY"`)))
			}

			if p.tok[0].Kind == token.KindIdentifier {
				tableOrSubquery.AddChild(parsetree.NewTerminal(parsetree.KindIndexName, p.tok[0]))
				p.advance()
			}
		} else if p.tok[0].Kind == token.KindNot {
			tableOrSubquery.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()

			if p.tok[0].Kind == token.KindIndexed {
				tableOrSubquery.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
				p.advance()
			} else {
				tableOrSubquery.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "INDEXED"`)))
			}
		}
	} else if p.tok[0].Kind == token.KindLeftParen {
		tableOrSubquery.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		if p.isStartOfExpression() {
			tableOrSubquery.AddChild(p.expressionList())
		} else if p.tok[0].Kind == token.KindRightParen {
			tableOrSubquery.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing expression`)))
		}

		if p.tok[0].Kind == token.KindRightParen {
			tableOrSubquery.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		} else {
			tableOrSubquery.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing ")"`)))
		}

		if p.tok[0].Kind == token.KindAs {
			tableOrSubquery.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()

			if p.tok[0].Kind == token.KindIdentifier {
				tableOrSubquery.AddChild(parsetree.NewTerminal(parsetree.KindTableAlias, p.tok[0]))
				p.advance()
			} else {
				tableOrSubquery.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing table alias`)))
			}
		} else if p.tok[0].Kind == token.KindIdentifier {
			tableOrSubquery.AddChild(parsetree.NewTerminal(parsetree.KindTableAlias, p.tok[0]))
			p.advance()
		}
	}
}

// joinClause parses a join clause.
func (p *Parser) joinClause() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindJoinClause)
	nt.AddChild(p.tableOrSubquery())

	joinOpStart := []token.Kind{token.KindComma, token.KindCross, token.KindFull, token.KindInner, token.KindLeft,
		token.KindNatural, token.KindOuter, token.KindRight, token.KindJoin}

	for {
		if !slices.Contains(joinOpStart, p.tok[0].Kind) {
			break
		}
		nt.AddChild(p.joinOperator())
		if p.tok[0].Kind == token.KindIdentifier || p.tok[0].Kind == token.KindLeftParen {
			nt.AddChild(p.tableOrSubquery())
		} else {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorExpecting, errors.New(`expecting identifier, or "("`)))
		}

		if p.tok[0].Kind == token.KindOn || p.tok[0].Kind == token.KindUsing {
			nt.AddChild(p.joinConstraint())
		}
	}

	return nt
}

// joinOperator parses a join operator.
func (p *Parser) joinOperator() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindJoinOperator)
	if p.tok[0].Kind == token.KindComma {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
		return nt
	}

FOR:
	for range 3 {
		switch p.tok[0].Kind {
		case token.KindCross, token.KindFull, token.KindInner, token.KindLeft, token.KindNatural,
			token.KindOuter, token.KindRight:
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		default:
			break FOR
		}
	}

	if p.tok[0].Kind == token.KindJoin {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "JOIN"`)))
	}

	return nt
}

// joinConstraint parses a join constraint.
func (p *Parser) joinConstraint() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindJoinConstraint)
	if p.tok[0].Kind == token.KindOn {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		if p.isStartOfExpression() {
			nt.AddChild(p.expression())
		} else {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing expression`)))
		}
	} else { // USING
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		if p.tok[0].Kind == token.KindLeftParen {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		} else if p.tok[0].Kind == token.KindIdentifier {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "("`)))
		}

		if p.tok[0].Kind == token.KindIdentifier {
			nt.AddChild(p.columnNameList(token.KindRightParen, token.KindSemicolon, token.KindEOF))
		} else if p.tok[0].Kind == token.KindRightParen {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing column name`)))
		}

		if p.tok[0].Kind == token.KindRightParen {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		} else {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing ")"`)))
		}
	}
	return nt
}

// groupByClause parses a group by clause.
func (p *Parser) groupByClause() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindGroupByClause)

	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	p.advance()

	if p.tok[0].Kind == token.KindBy {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else if p.isStartOfExpression() {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "BY"`)))
	}

	if p.isStartOfExpression() {
		nt.AddChild(p.expressionList())
	} else {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing expression`)))
	}

	return nt
}

// havingClause parses a having clause.
func (p *Parser) havingClause() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindHavingClause)
	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	p.advance()

	if p.isStartOfExpression() {
		nt.AddChild(p.expression())
	} else {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing expression`)))
	}

	return nt
}

// windowClause parses a window clause.
func (p *Parser) windowClause() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindWindowClause)

	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	p.advance()

	if p.tok[0].Kind == token.KindIdentifier {
		nt.AddChild(p.windowClauseItemList())
	} else {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing window declaration`)))
	}

	return nt
}

func (p *Parser) windowClauseItemList() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindCommaList)

	for {
		p.addChild(p.windowClauseItem())
		if !p.is(token.KindComma) {
			break
		}
		p.term(token.KindComma)
	}

	return p.popTree()
}

// windowClauseItem parses a window declaration.
func (p *Parser) windowClauseItem() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindWindowClauseItem)
	nt.AddChild(parsetree.NewTerminal(parsetree.KindWindowName, p.tok[0]))
	p.advance()

	if p.tok[0].Kind == token.KindAs {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else if p.tok[0].Kind == token.KindLeftParen {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "AS"`)))
	}

	if p.tok[0].Kind == token.KindLeftParen {
		nt.AddChild(p.windowDefinition())
	} else {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing window definition`)))
	}

	return nt
}

// valuesClause parses a values clause.
func (p *Parser) valuesClause() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindValuesClause)
	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	p.advance()

	if p.tok[0].Kind == token.KindLeftParen {
		nt.AddChild(p.valuesItemList())
	} else {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing values item`)))
	}

	return nt
}

func (p *Parser) valuesItemList() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindCommaList)

	for {
		p.addChild(p.valuesItem())
		if !p.is(token.KindComma) {
			break
		}
		p.term(token.KindComma)
	}

	return p.popTree()
}

// valuesItem parses a item of a values clause.
func (p *Parser) valuesItem() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindValuesItem)
	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	p.advance()

	if p.isStartOfExpression() {
		nt.AddChild(p.expressionList())
	} else if p.tok[0].Kind == token.KindRightParen {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing expression`)))
	}

	if p.tok[0].Kind == token.KindRightParen {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	}

	return nt
}

// compoundOperator parses a item of a compound operator.
func (p *Parser) compoundOperator() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindCompoundOperator)
	if p.tok[0].Kind == token.KindUnion {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		if p.tok[0].Kind == token.KindAll {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		}
	} else { // INSTERSECT, EXCEPT
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	}

	return nt
}

// limitClause parses a limit clause.
func (p *Parser) limitClause() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindLimitClause)

	p.term(token.KindLimit)
	p.addChild(p.expression())

	if p.isAnyOf(token.KindOffset, token.KindComma) {
		p.term()
		p.addChild(p.expression())
	}

	return p.popTree()
}

// update parses a update statement.
func (p *Parser) update(withClause parsetree.NonTerminal) parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindUpdate)
	if withClause != nil {
		p.addChild(withClause)
	}

	p.term(token.KindUpdate)

	if p.is(token.KindOr) {
		p.term()
		p.term(token.KindAbort, token.KindFail, token.KindIgnore, token.KindReplace, token.KindRollback)
	}

	p.addChild(p.qualifiedTableName())

	p.term(token.KindSet)
	p.addChild(p.updateSetItemList())

	if p.is(token.KindFrom) {
		p.addChild(p.fromClause())
	}

	if p.is(token.KindWhere) {
		p.addChild(p.whereClause())
	}

	if p.is(token.KindReturning) {
		p.addChild(p.returningClause())
	}

	if p.is(token.KindOrder) {
		p.addChild(p.orderByClause())
	}

	if p.is(token.KindLimit) {
		p.addChild(p.limitClause())
	}

	return p.popTree()
}

// vacuum parses a vacuum statement.
func (p *Parser) vacuum() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindVacuum)
	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	p.advance()

	if p.tok[0].Kind == token.KindIdentifier || p.tok[0].Kind == token.KindTemp {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindSchemaName, p.tok[0]))
		p.advance()
	}

	if p.tok[0].Kind == token.KindInto {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		if p.isStartOfExpression() {
			fn := parsetree.NewNonTerminal(parsetree.KindFileName)
			fn.AddChild(p.expression())
			nt.AddChild(fn)
		} else {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing file name`)))
		}
	}

	return nt
}

func (p *Parser) pushTree(t parsetree.Construction) {
	p.treeStack = append(p.treeStack, t)
}

func (p *Parser) pushTreeKind(k parsetree.Kind) {
	p.pushTree(parsetree.NewNonTerminal(k))
}

func (p *Parser) addChild(t parsetree.Construction) {
	p.treeStack[len(p.treeStack)-1].(parsetree.NonTerminal).AddChild(t)
}

func (p *Parser) popTree() parsetree.NonTerminal {
	return p.popConstr().(parsetree.NonTerminal)
}

func (p *Parser) popConstr() parsetree.Construction {
	t := p.treeStack[len(p.treeStack)-1]
	p.treeStack = p.treeStack[:len(p.treeStack)-1]
	return t
}

func (p *Parser) is(k token.Kind) bool {
	return p.isPos(0, k)
}

func (p *Parser) isPos(pos int, k token.Kind) bool {
	return p.tok[pos].Kind == k
}

func (p *Parser) isSeq(k0, k1 token.Kind, ks ...token.Kind) bool {
	if !p.isPos(0, k0) {
		return false
	}
	if !p.isPos(1, k1) {
		return false
	}
	for i := range ks {
		if !p.isPos(i+2, ks[i]) {
			return false
		}
	}
	return true
}

func (p *Parser) isAnyOf(k0, k1 token.Kind, ks ...token.Kind) bool {
	return p.isAnyOfPos(0, k0, k1, ks...)
}

func (p *Parser) isAnyOfPos(pos int, k0, k1 token.Kind, ks ...token.Kind) bool {
	if p.tok[pos].Kind == k0 || p.tok[pos].Kind == k1 {
		return true
	}
	return slices.Contains(ks, p.tok[pos].Kind)
}

func (p *Parser) isKeyword() bool {
	return p.tok[0].Kind.IsKeyword()
}

func (p *Parser) term(tokKinds ...token.Kind) {
	p.termKind(parsetree.KindToken, tokKinds...)
}

func (p *Parser) termKind(treeKind parsetree.Kind, tokKinds ...token.Kind) {
	if len(tokKinds) > 0 {
		p.token(tokKinds[0], tokKinds[1:]...)
	}
	t := parsetree.NewTerminal(treeKind, p.tok[0])
	p.addChild(t)
	p.advance()
}

func (p *Parser) token(k token.Kind, ks ...token.Kind) token.Kind {
	return p.tokenPos(0, k, ks...)
}

func (p *Parser) tokenPos(pos int, k token.Kind, ks ...token.Kind) token.Kind {
	if p.tok[pos].Kind != k && !slices.Contains(ks, p.tok[pos].Kind) {
		panic(&syntaxError{expected: append([]token.Kind{k}, ks...), got: p.tok[pos]})
	}
	return p.tok[pos].Kind
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
			copy(p.tok[:], p.tok[1:])
			p.tok[len(p.tok)-1] = tok
			break
		}
	}
}
