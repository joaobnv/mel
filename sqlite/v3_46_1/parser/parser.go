// This package deals with the parsing of the SQL.
package parser

// TODO: implement the quote rule exceptions: lang_keywords.html.

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
	expected []token.Kind
	got      *token.Token
}

func (se *syntaxError) Error() string {
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
	tok [3]*token.Token
	// l is the lexer.
	l         *lexer.Lexer
	treeStack []parsetree.NonTerminal
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

	p.pushTree(parsetree.KindSQLStatement)

	var explain bool
	if p.isSeq(token.KindExplain, token.KindQuery) {
		p.pushTree(parsetree.KindExplainQueryPlan)
		p.term(token.KindExplain)
		p.term(token.KindQuery)
		p.term(token.KindPlan)
		explain = true
	} else if p.is(token.KindExplain) {
		p.pushTree(parsetree.KindExplain)
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
	p.pushTree(parsetree.KindAlterTable)
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
	p.pushTree(parsetree.KindRenameTo)
	p.term(token.KindRename)
	p.term(token.KindTo)
	p.termKind(parsetree.KindTableName, token.KindIdentifier)
	return p.popTree()
}

func (p *Parser) alterTableRenameColumn() parsetree.NonTerminal {
	p.pushTree(parsetree.KindRenameColumn)
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
	p.pushTree(parsetree.KindAddColumn)
	p.term(token.KindAdd)

	if p.is(token.KindColumn) {
		p.term(token.KindColumn)
	}

	p.token(token.KindIdentifier)
	p.addChild(p.columnDefinition())
	return p.popTree()
}

func (p *Parser) alterTableDropColumn() parsetree.NonTerminal {
	p.pushTree(parsetree.KindDropColumn)
	p.term(token.KindDrop)

	if p.is(token.KindColumn) {
		p.term(token.KindColumn)
	}

	p.termKind(parsetree.KindColumnName, token.KindIdentifier)
	return p.popTree()
}

// columnDefinition parses a column definition.
func (p *Parser) columnDefinition() parsetree.NonTerminal {
	p.pushTree(parsetree.KindColumnDefinition)

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
	p.pushTree(parsetree.KindTypeName)

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
	p.pushTree(parsetree.KindColumnConstraint)

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
	p.pushTree(parsetree.KindPrimaryKeyColumnConstraint)
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
	p.pushTree(parsetree.KindNotNullColumnConstraint)
	p.term(token.KindNot)
	p.term(token.KindNull)

	if p.is(token.KindOn) {
		p.addChild(p.conflictClause())
	}

	return p.popTree()
}

// uniqueColumnConstraint parses a unique column constraint clause.
func (p *Parser) uniqueColumnConstraint() parsetree.NonTerminal {
	p.pushTree(parsetree.KindUniqueColumnConstraint)
	p.term(token.KindUnique)

	if p.is(token.KindOn) {
		p.addChild(p.conflictClause())
	}

	return p.popTree()
}

// checkColumnConstraint parses a check column constraint clause.
func (p *Parser) checkColumnConstraint() parsetree.NonTerminal {
	p.pushTree(parsetree.KindCheckColumnConstraint)
	p.term(token.KindCheck)
	p.term(token.KindLeftParen)
	p.addChild(p.expression())
	p.term(token.KindRightParen)
	return p.popTree()
}

// defaultColumnConstraint parses a default column constraint clause.
func (p *Parser) defaultColumnConstraint() parsetree.NonTerminal {
	p.pushTree(parsetree.KindDefaultColumnConstraint)
	p.term(token.KindDefault)

	if p.is(token.KindLeftParen) {
		p.term(token.KindLeftParen)
		p.addChild(p.expression())
		p.term(token.KindRightParen)
	} else if p.isLiteralValue(0) {
		p.term()
	} else { // note that a numeric token is a literal value
		p.token(token.KindPlus, token.KindMinus)
		p.signedNumber()
	}

	return p.popTree()
}

// collateColumnConstraint parses a collate column constraint clause.
func (p *Parser) collateColumnConstraint() parsetree.NonTerminal {
	p.pushTree(parsetree.KindCollateColumnConstraint)
	p.term(token.KindCollate)
	p.termKind(parsetree.KindCollationName, token.KindIdentifier)
	return p.popTree()
}

// generatedColumnConstraint parses a generated column constraint clause.
func (p *Parser) generatedColumnConstraint() parsetree.NonTerminal {
	p.pushTree(parsetree.KindGeneratedColumnConstraint)
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
	p.pushTree(parsetree.KindForeignKeyColumnConstraint)
	p.addChild(p.foreignKeyClause())
	return p.popTree()
}

// foreignKeyClause parses a foreign key clause.
func (p *Parser) foreignKeyClause() parsetree.NonTerminal {
	p.pushTree(parsetree.KindForeignKeyClause)
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
	p.pushTree(parsetree.KindConflictClause)
	p.term(token.KindOn)
	p.term(token.KindConflict)
	p.term(token.KindRollback, token.KindAbort, token.KindFail, token.KindIgnore, token.KindReplace)
	return p.popTree()
}

// analyze parses a analyze statement.
func (p *Parser) analyze() parsetree.NonTerminal {
	p.pushTree(parsetree.KindAnalyze)
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
	p.pushTree(parsetree.KindAttach)
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
	p.pushTree(parsetree.KindBegin)
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
	p.pushTree(parsetree.KindCommit)
	p.term(token.KindCommit, token.KindEnd)

	if p.is(token.KindTransaction) {
		p.term()
	}

	return p.popTree()
}

// rollback parses a rollback statement.
func (p *Parser) rollback() parsetree.NonTerminal {
	p.pushTree(parsetree.KindRollback)
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
	p.pushTree(parsetree.KindCreateIndex)
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
	p.pushTree(parsetree.KindCommaList)
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
	p.pushTree(parsetree.KindIndexedColumn)
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
	p.pushTree(parsetree.KindCreateTable)
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
	p.pushTree(parsetree.KindCommaList)

	p.addChild(p.columnDefinition())
	for p.isSeq(token.KindComma, token.KindIdentifier) {
		p.term(token.KindComma)
		p.addChild(p.columnDefinition())
	}

	return p.popTree()
}

// tableConstraintList parses a list of tale constraints in a create table.
func (p *Parser) tableConstraintList() parsetree.NonTerminal {
	p.pushTree(parsetree.KindCommaList)
	p.addChild(p.tableConstraint())
	for p.is(token.KindComma) {
		p.term()
		p.addChild(p.tableConstraint())
	}
	return p.popTree()
}

// tableConstraint parses a table constraint clause.
func (p *Parser) tableConstraint() parsetree.NonTerminal {
	p.pushTree(parsetree.KindTableConstraint)
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
	p.pushTree(parsetree.KindPrimaryKeyTableConstraint)

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
	p.pushTree(parsetree.KindUniqueTableConstraint)

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
	p.pushTree(parsetree.KindCheckTableConstraint)

	p.term(token.KindCheck)
	p.term(token.KindLeftParen)

	if p.isStartOfExpression(0) {
		p.addChild(p.expression())
	} else {
		panic("TODO: the error must come from the expression parser")
	}
	p.term(token.KindRightParen)

	return p.popTree()
}

// foreignKeyTableConstraint parses a foreign key table constraint clause.
func (p *Parser) foreignKeyTableConstraint() parsetree.NonTerminal {
	p.pushTree(parsetree.KindForeignKeyTableConstraint)

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
	p.pushTree(parsetree.KindCommaList)
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
	p.pushTree(parsetree.KindTableOption)

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
	p.pushTree(parsetree.KindCommaList)
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
	p.pushTree(parsetree.KindCreateTrigger)

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

		if p.isStartOfExpression(0) {
			p.addChild(p.expression())
		} else {
			panic("TODO: the error must come from the expression parser")
		}
	}

	p.addChild(p.triggerBody())

	return p.popTree()
}

// triggerBody parses the body of a create trigger statement.
func (p *Parser) triggerBody() parsetree.NonTerminal {
	p.pushTree(parsetree.KindTriggerBody)
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
		nt.AddChild(p.selectStatement(nil))
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
func (p *Parser) delete(withClause parsetree.NonTerminal) parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindDelete)
	if withClause != nil {
		nt.AddChild(withClause)
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
		nt.AddChild(p.whereClause())
	}

	if p.tok[0].Kind == token.KindReturning {
		nt.AddChild(p.returningClause())
	}

	return nt
}

// withClause parses a with clause.
func (p *Parser) withClause() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindWithClause)
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
	nt := parsetree.NewNonTerminal(parsetree.KindCommaList)

	nt.AddChild(p.commonTableExpression())

	for p.tok[0].Kind == token.KindComma {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		if p.tok[0].Kind == token.KindIdentifier {
			nt.AddChild(p.commonTableExpression())
		} else {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing CTE`)))
		}
	}

	return nt
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
	} else if p.tok[0].Kind == token.KindWith || p.tok[0].Kind == token.KindSelect {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "("`)))
	}

	var withClause parsetree.NonTerminal
	if p.tok[0].Kind == token.KindWith {
		withClause = p.withClause()
	}

	if p.tok[0].Kind == token.KindSelect {
		nt.AddChild(p.selectStatement(withClause))
	} else if p.tok[0].Kind == token.KindRightParen {
		if withClause != nil {
			nt.AddChild(withClause)
		}
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

	if p.isStartOfExpression(0) || p.tok[0].Kind == token.KindAsterisk {
		nt.AddChild(p.returningItemList())
	} else {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing returning item`)))
	}

	return nt
}

// returningItemList parses a list of itens in a returning clause.
func (p *Parser) returningItemList() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindCommaList)

	nt.AddChild(p.returningItem())

	for p.tok[0].Kind == token.KindComma {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		if p.isStartOfExpression(0) || p.tok[0].Kind == token.KindAsterisk {
			nt.AddChild(p.returningItem())
		} else {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing returning item`)))
		}
	}

	return nt
}

// returningItem parses a item in a returning clause.
func (p *Parser) returningItem() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindReturningItem)

	if p.isStartOfExpression(0) {
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

// whereClause parses a where clause.
func (p *Parser) whereClause() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindWhereClause)
	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	p.advance()

	if p.isStartOfExpression(0) {
		nt.AddChild(p.expression())
	} else {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing expression`)))
	}

	return nt
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

// dropTrigger parses a drop trigger statement.
func (p *Parser) dropTrigger() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindDropTrigger)
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
		nt.AddChild(parsetree.NewTerminal(parsetree.KindTriggerName, p.tok[0]))
		p.advance()
	} else {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing trigger name`)))
	}

	return nt
}

// dropView parses a drop view statement.
func (p *Parser) dropView() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindDropView)
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
		nt.AddChild(parsetree.NewTerminal(parsetree.KindViewName, p.tok[0]))
		p.advance()
	} else {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing view name`)))
	}

	return nt
}

func (p *Parser) expressionList() parsetree.NonTerminal {
	p.pushTree(parsetree.KindCommaList)
	for {
		p.addChild(p.expression())
		if !p.is(token.KindComma) {
			break
		}
		p.term(token.KindComma)
	}
	return p.popTree()
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

		if p.isStartOfExpression(0) {
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

		if p.isStartOfExpression(0) {
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

		if p.isStartOfExpression(0) {
			nt.AddChild(p.expression3())
		} else {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing expression`)))
		}

		return nt
	}

	return p.expression4()
}

func (p *Parser) expression4List() parsetree.NonTerminal {
	p.pushTree(parsetree.KindCommaList)
	for {
		p.addChild(p.expression4())
		if !p.is(token.KindComma) {
			break
		}
		p.term(token.KindComma)
	}
	return p.popTree()
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

			if p.isStartOfExpressionAtLeast4(0) {
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

			if p.isStartOfExpressionAtLeast4(0) {
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

			if p.isStartOfExpressionAtLeast4(0) {
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

			if p.isStartOfExpressionAtLeast4(0) {
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

			if p.isStartOfExpressionAtLeast4(0) {
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

			if p.isStartOfExpressionAtLeast4(0) {
				nt.AddChild(p.expression5())
			} else {
				nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing expression (not starting with "NOT")`)))
			}

			// apparently there is an error in the ESCAPE precedence documentation
			if p.tok[0].Kind == token.KindEscape {
				nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
				p.advance()

				if p.isStartOfExpressionAtLeast4(0) {
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

				if p.isStartOfExpressionAtLeast4(0) {
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

				if p.isStartOfExpressionAtLeast4(0) {
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

				if p.isStartOfExpressionAtLeast4(0) {
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

				if p.isStartOfExpressionAtLeast4(0) {
					nt.AddChild(p.expression5())
				} else {
					nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing expression (not starting with "NOT")`)))
				}

				if p.tok[0].Kind == token.KindEscape {
					nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
					p.advance()

					if p.isStartOfExpressionAtLeast4(0) {
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
		if p.isStartOfExpression(2) && p.tok[2].Kind != token.KindNot {
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
			} else if p.isStartOfExpression(0) {
				nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "FROM"`)))
			}

			if p.isStartOfExpression(0) && p.tok[0].Kind != token.KindNot {
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
		} else if p.isStartOfExpression(0) {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "FROM"`)))
		}

		if p.isStartOfExpression(0) && p.tok[0].Kind != token.KindNot {
			nt.AddChild(p.expression4())
		} else {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing expression (not starting with "NOT")`)))
		}
	} else if p.isStartOfExpression(1) {
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

	if p.isStartOfExpressionAtLeast4(0) {
		nt.AddChild(p.expression4())
	} else if p.tok[0].Kind == token.KindAnd {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing expression (not starting with "NOT")`)))
	}

	if p.tok[0].Kind == token.KindAnd {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else if p.isStartOfExpressionAtLeast4(0) {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "AND"`)))
	}

	if p.isStartOfExpressionAtLeast4(0) {
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

	if p.isStartOfExpressionAtLeast4(0) {
		nt.AddChild(p.expression4())
	} else if p.tok[0].Kind == token.KindAnd {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing expression (not starting with "NOT")`)))
	}

	if p.tok[0].Kind == token.KindAnd {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else if p.isStartOfExpressionAtLeast4(0) {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "AND"`)))
	}

	if p.isStartOfExpressionAtLeast4(0) {
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

		if p.tok[0].Kind == token.KindWith || p.tok[0].Kind == token.KindSelect {
			var withClause parsetree.NonTerminal
			if p.tok[0].Kind == token.KindWith {
				withClause = p.withClause()
			}
			nt.AddChild(p.selectStatement(withClause))
		} else if p.isStartOfExpressionAtLeast4(0) {
			nt.AddChild(p.expression4List())
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

		if p.isStartOfExpressionAtLeast4(0) {
			nt.AddChild(p.expression4List())
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

		if p.tok[0].Kind == token.KindWith || p.tok[0].Kind == token.KindSelect {
			var withClause parsetree.NonTerminal
			if p.tok[0].Kind == token.KindWith {
				withClause = p.withClause()
			}
			nt.AddChild(p.selectStatement(withClause))
		} else if p.isStartOfExpressionAtLeast4(0) {
			nt.AddChild(p.expression4List())
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

		if p.isStartOfExpressionAtLeast4(0) {
			nt.AddChild(p.expression4List())
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
func (p *Parser) isStartOfExpressionAtLeast4(pos int) bool {
	return p.isStartOfExpression(pos) && !p.isPos(pos, token.KindNot)
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

		if p.isStartOfExpressionAtLeast4(0) {
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

		if p.isStartOfExpressionAtLeast4(0) {
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

		if p.isStartOfExpressionAtLeast4(0) {
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

		if p.isStartOfExpressionAtLeast4(0) {
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

		if p.isStartOfExpressionAtLeast4(0) {
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

	if p.isStartOfExpressionAtLeast4(0) {
		nt.AddChild(p.expression11())
	} else {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing expression (not starting with "NOT")`)))
	}

	return nt
}

// simpleExpression parses a simple expression, that is, a expression with the highest precedence.
func (p *Parser) simpleExpression() parsetree.Construction {
	if p.isLiteralValue(0) {
		t := parsetree.NewTerminal(parsetree.KindToken, p.tok[0])
		p.advance()
		return t
	} else if p.isBindParameter(0) {
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
func (p *Parser) isStartOfExpression(pos int) bool {
	if p.isLiteralValue(pos) {
		return true
	}
	if p.isBindParameter(pos) {
		return true
	}

	return p.isAnyOfPos(pos, token.KindIdentifier, token.KindTilde, token.KindPlus, token.KindMinus,
		token.KindNot, token.KindLeftParen,
		token.KindCast, token.KindExists, token.KindCase, token.KindRaise)
}

// isLiteralValue reports whether tok is a literal value.
func (p *Parser) isLiteralValue(pos int) bool {
	if p.isAnyOfPos(pos, token.KindNumeric, token.KindString, token.KindBlob,
		token.KindNull, token.KindCurrentTime, token.KindCurrentDate,
		token.KindCurrentTimestamp, token.KindRowId,
	) {
		return true
	}

	if !p.isPos(pos, token.KindIdentifier) {
		return false
	}

	lex := strings.ToLower(string(p.tok[pos].Lexeme))
	return lex == "true" || lex == "false"
}

// isBindParameter reports whether tok is a bind parameter.
func (p *Parser) isBindParameter(pos int) bool {
	return p.isAnyOfPos(pos, token.KindAtVariable, token.KindColonVariable, token.KindDollarVariable, token.KindQuestionVariable)
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

	if p.tok[0].Kind == token.KindDistinct || p.isStartOfExpression(0) || p.tok[0].Kind == token.KindAsterisk {
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

	if p.isStartOfExpression(0) {
		nt.AddChild(p.expressionList())
	} else {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing argument`)))
	}

	if p.tok[0].Kind == token.KindOrder {
		nt.AddChild(p.orderByClause(func(t *token.Token) bool {
			return t.Kind == token.KindRightParen || t.Kind == token.KindSemicolon || t.Kind == token.KindEOF
		}))
	}

	return nt
}

// orderByClause parses an order by clause.
func (p *Parser) orderByClause(isInFollowSet func(*token.Token) bool) parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindOrderByClause)
	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	p.advance()

	if p.tok[0].Kind == token.KindBy {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else if p.isStartOfExpression(0) || isInFollowSet(p.tok[0]) {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "BY"`)))
	}

	if p.isStartOfExpression(0) {
		nt.AddChild(p.orderingTermList())
	} else {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing ordering term`)))
	}

	return nt
}

func (p *Parser) orderingTermList() parsetree.NonTerminal {
	p.pushTree(parsetree.KindCommaList)
	for {
		p.addChild(p.orderingTerm())
		if !p.is(token.KindComma) {
			break
		}
		p.term(token.KindComma)
	}
	return p.popTree()
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
	} else if p.isStartOfExpression(0) {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "WHERE"`)))
	}

	if p.isStartOfExpression(0) {
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
	} else if p.tok[0].Kind == token.KindLeftParen {
		nt.AddChild(p.windowDefinition())
	} else {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorExpecting, errors.New(`expecting identifier, or "("`)))
	}

	return nt
}

// windowDefinition parses a window definition.
func (p *Parser) windowDefinition() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindWindowDefinition)

	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	p.advance()

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
		} else if p.isStartOfExpression(0) {
			pb.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "BY"`)))
		}

		followSet := []token.Kind{
			token.KindOrder, token.KindRange, token.KindRows, token.KindGroups, token.KindRightParen, token.KindSemicolon,
		}
		if p.isStartOfExpression(0) {
			pb.AddChild(p.expressionList())
		} else if slices.Contains(followSet, p.tok[0].Kind) {
			pb.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing expression`)))
		}

		nt.AddChild(pb)
	}

	if p.tok[0].Kind == token.KindOrder {
		followSet := []token.Kind{token.KindRange, token.KindRows, token.KindGroups, token.KindRightParen, token.KindSemicolon}
		nt.AddChild(p.orderByClause(func(t *token.Token) bool { return slices.Contains(followSet, t.Kind) }))
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
	} else if p.isStartOfExpression(0) {
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
	} else if p.isStartOfExpressionAtLeast4(0) {
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
	} else if p.isStartOfExpressionAtLeast4(0) {
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

	if p.isStartOfExpression(0) || p.tok[0].Kind == token.KindComma {
		nt.AddChild(p.expressionList())
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

	if p.isStartOfExpression(0) {
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

	if p.tok[0].Kind == token.KindWith || p.tok[0].Kind == token.KindSelect {
		var withClause parsetree.NonTerminal
		if p.tok[0].Kind == token.KindWith {
			withClause = p.withClause()
		}
		nt.AddChild(p.selectStatement(withClause))
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

	if p.isStartOfExpression(0) {
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

	if p.isStartOfExpression(0) {
		nt.AddChild(p.expression())
	} else if p.tok[0].Kind == token.KindThen {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing expression`)))
	}

	if p.tok[0].Kind == token.KindThen {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else if p.isStartOfExpression(0) {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "THEN"`)))
	}

	if p.isStartOfExpression(0) {
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

	if p.isStartOfExpression(0) {
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
	} else if p.isStartOfExpression(0) {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing ","`)))
	}

	if p.isStartOfExpression(0) {
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

// insert parses a insert statement.
func (p *Parser) insert(withClause parsetree.NonTerminal) parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindInsert)

	if withClause != nil {
		nt.AddChild(withClause)
	}

	if p.tok[0].Kind == token.KindInsert {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		if p.tok[0].Kind == token.KindOr {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()

			switch p.tok[0].Kind {
			case token.KindAbort, token.KindFail, token.KindIgnore, token.KindReplace, token.KindRollback:
				nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
				p.advance()
			case token.KindInto:
				nt.AddChild(parsetree.NewError(parsetree.KindErrorExpecting, errors.New(`expecting "ABORT", "FAIL", "IGNORE", "REPLACE", or "ROLLBACK"`)))
			}
		}
	} else {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	}

	if p.tok[0].Kind == token.KindInto {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else if p.tok[0].Kind == token.KindIdentifier {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "INTO"`)))
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

	switch p.tok[0].Kind {
	case token.KindIdentifier:
		nt.AddChild(parsetree.NewTerminal(parsetree.KindTableName, p.tok[0]))
		p.advance()
	case token.KindAs, token.KindLeftParen, token.KindValues, token.KindWith, token.KindSelect, token.KindDefault:
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing table name`)))
	}

	if p.tok[0].Kind == token.KindAs {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		switch p.tok[0].Kind {
		case token.KindIdentifier:
			nt.AddChild(parsetree.NewTerminal(parsetree.KindTableAlias, p.tok[0]))
			p.advance()
		case token.KindLeftParen, token.KindValues, token.KindWith, token.KindSelect, token.KindDefault:
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing table alias`)))
		}
	}

	if p.tok[0].Kind == token.KindLeftParen {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		if p.tok[0].Kind == token.KindIdentifier || p.tok[0].Kind == token.KindComma {
			nt.AddChild(p.columnNameList(token.KindRightParen, token.KindValues, token.KindWith, token.KindSelect,
				token.KindDefault, token.KindSemicolon, token.KindEOF))
		} else if p.tok[0].Kind == token.KindRightParen {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing column name`)))
		}

		if p.tok[0].Kind == token.KindRightParen {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		} else if p.tok[0].Kind == token.KindValues || p.tok[0].Kind == token.KindWith || p.tok[0].Kind == token.KindSelect ||
			p.tok[0].Kind == token.KindDefault {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing ")"`)))
		}
	}

	if p.tok[0].Kind == token.KindValues {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		if p.tok[0].Kind == token.KindLeftParen {
			nt.AddChild(p.insertValuesList())
		} else {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "("`)))
		}

		if p.tok[0].Kind == token.KindOn {
			nt.AddChild(p.upsertClause())
		}
	} else if p.tok[0].Kind == token.KindWith || p.tok[0].Kind == token.KindSelect {
		var withClause parsetree.NonTerminal
		if p.tok[0].Kind == token.KindWith {
			withClause = p.withClause()
		}

		if p.tok[0].Kind == token.KindSelect {
			nt.AddChild(p.selectStatement(withClause))
		} else if withClause != nil {
			nt.AddChild(withClause)
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing select`)))
		}

		if p.tok[0].Kind == token.KindOn {
			nt.AddChild(p.upsertClause())
		}
	} else if p.tok[0].Kind == token.KindDefault {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		if p.tok[0].Kind == token.KindValues {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		} else {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "VALUES"`)))
		}
	} else {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorExpecting, errors.New(`expecting "VALUES", "WITH", "SELECT", or "DEFAULT"`)))
	}

	if p.tok[0].Kind == token.KindReturning {
		nt.AddChild(p.returningClause())
	}

	return nt
}

// insertValues parses the values list of a insert.
func (p *Parser) insertValuesList() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindInsertValuesList)
	nt.AddChild(p.insertValuesItemList())
	return nt
}

func (p *Parser) insertValuesItemList() parsetree.NonTerminal {
	p.pushTree(parsetree.KindCommaList)

	for {
		p.addChild(p.insertValuesItem())
		if !p.is(token.KindComma) {
			break
		}
		p.term(token.KindComma)
	}

	return p.popTree()
}

// insertValuesItem parses a item in a insert values list.
func (p *Parser) insertValuesItem() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindInsertValuesItem)

	if p.tok[0].Kind == token.KindLeftParen {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else { // expression
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "("`)))
	}

	nt.AddChild(p.expressionList())

	if p.tok[0].Kind == token.KindRightParen {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	}

	return nt
}

// upsertClause parses a upsert clause.
func (p *Parser) upsertClause() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindUpsertClause)

	for p.tok[0].Kind == token.KindOn {
		nt.AddChild(p.upsertClauseItem())
	}

	return nt
}

// upsertClauseItem parses a item of an upsert clause.
func (p *Parser) upsertClauseItem() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindUpsertClauseItem)
	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	p.advance()

	if p.tok[0].Kind == token.KindConflict {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else if p.tok[0].Kind == token.KindLeftParen || p.tok[0].Kind == token.KindDo {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "CONFLICT"`)))
	}

	if p.tok[0].Kind == token.KindLeftParen {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		nt.AddChild(p.indexedColumnList(true))

		if p.tok[0].Kind == token.KindRightParen {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		} else if p.tok[0].Kind == token.KindWhere || p.tok[0].Kind == token.KindDo {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing ")"`)))
		}

		if p.tok[0].Kind == token.KindWhere {
			nt.AddChild(p.whereClause())
		}
	}

	if p.tok[0].Kind == token.KindDo {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else if p.tok[0].Kind == token.KindNothing || p.tok[0].Kind == token.KindUpdate {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "DO"`)))
	}

	if p.tok[0].Kind == token.KindNothing {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else if p.tok[0].Kind == token.KindUpdate {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		if p.tok[0].Kind == token.KindSet {
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		} else if p.tok[0].Kind == token.KindIdentifier || p.tok[0].Kind == token.KindLeftParen {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "SET"`)))
		}

		if p.tok[0].Kind == token.KindIdentifier || p.tok[0].Kind == token.KindLeftParen {
			nt.AddChild(p.updateSetItemList())
		} else if p.tok[0].Kind == token.KindWhere {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorExpecting, errors.New(`expecting column name, or "("`)))
		}

		if p.tok[0].Kind == token.KindWhere {
			nt.AddChild(p.whereClause())
		}
	} else {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorExpecting, errors.New(`expecting "NOTHING", or "UPDATE"`)))
	}

	return nt
}

func (p *Parser) updateSetItemList() parsetree.NonTerminal {
	p.pushTree(parsetree.KindCommaList)

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
	p.pushTree(parsetree.KindUpdateSetItem)

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
	nt := parsetree.NewNonTerminal(parsetree.KindPragma)

	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	p.advance()

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
		nt.AddChild(parsetree.NewTerminal(parsetree.KindPragmaName, p.tok[0]))
		p.advance()
	} else {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing pragma name`)))
	}

	if p.tok[0].Kind == token.KindEqual {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		if p.tok[0].Kind == token.KindPlus || p.tok[0].Kind == token.KindMinus || p.tok[0].Kind == token.KindNumeric || p.tok[0].Kind.IsKeyword() || p.tok[0].Kind == token.KindIdentifier || p.tok[0].Kind == token.KindString {
			nt.AddChild(p.pragmaValue())
		} else {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing pragma value`)))
		}
	} else if p.tok[0].Kind == token.KindLeftParen {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		if p.tok[0].Kind == token.KindPlus || p.tok[0].Kind == token.KindMinus || p.tok[0].Kind == token.KindNumeric || p.tok[0].Kind.IsKeyword() || p.tok[0].Kind == token.KindIdentifier || p.tok[0].Kind == token.KindString {
			nt.AddChild(p.pragmaValue())
		} else {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing pragma value`)))
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

// pragmaValue parses a pragma value.
func (p *Parser) pragmaValue() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindPragmaValue)

	if p.tok[0].Kind == token.KindPlus || p.tok[0].Kind == token.KindMinus {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	}

	if p.tok[0].Kind == token.KindNumeric || p.tok[0].Kind.IsKeyword() || p.tok[0].Kind == token.KindIdentifier || p.tok[0].Kind == token.KindString {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()
	} else {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorExpecting, errors.New(`expecting number, keyword, identifier, or string`)))
	}

	return nt
}

// reindex parses a reindex statement.
func (p *Parser) reindex() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindReindex)

	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	p.advance()

	var hasSchema bool
	if p.tok[0].Kind == token.KindIdentifier || p.tok[0].Kind == token.KindTemp {
		if p.tok[1].Kind == token.KindDot {
			hasSchema = true

			nt.AddChild(parsetree.NewTerminal(parsetree.KindSchemaName, p.tok[0]))
			p.advance()
			nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
			p.advance()
		} else if p.tok[1].Kind == token.KindIdentifier {
			hasSchema = true

			nt.AddChild(parsetree.NewTerminal(parsetree.KindSchemaName, p.tok[0]))
			p.advance()
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing dot`)))
		}
	}

	if p.tok[0].Kind == token.KindIdentifier {
		k := parsetree.KindCollationTableOrIndexName
		if hasSchema {
			k = parsetree.KindTableOrIndexName
		}
		nt.AddChild(parsetree.NewTerminal(k, p.tok[0]))
		p.advance()
	} else if hasSchema {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing table name, or index name`)))
	}

	return nt
}

// release parses a release statement.
func (p *Parser) release() parsetree.NonTerminal {
	nt := parsetree.NewNonTerminal(parsetree.KindRelease)
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

	return nt
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
		nt.AddChild(p.orderByClause(func(t *token.Token) bool {
			return t.Kind == token.KindLimit || t.Kind == token.KindSemicolon || t.Kind == token.KindEOF
		}))
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

		if p.isStartOfExpression(0) || p.tok[0].Kind == token.KindAsterisk {
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

		if p.isStartOfExpression(0) || p.tok[0].Kind == token.KindAsterisk {
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
	} else if p.isStartOfExpression(0) {
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

		if p.isStartOfExpression(0) {
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

		if p.isStartOfExpression(0) {
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
	} else if p.isStartOfExpression(0) {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing "BY"`)))
	}

	if p.isStartOfExpression(0) {
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

	if p.isStartOfExpression(0) {
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
	p.pushTree(parsetree.KindCommaList)

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
	p.pushTree(parsetree.KindCommaList)

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

	if p.isStartOfExpression(0) {
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
	nt := parsetree.NewNonTerminal(parsetree.KindLimitClause)
	nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
	p.advance()

	if p.isStartOfExpression(0) {
		nt.AddChild(p.expression())
	} else {
		nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing expression`)))
	}

	if p.tok[0].Kind == token.KindOffset || p.tok[0].Kind == token.KindComma {
		nt.AddChild(parsetree.NewTerminal(parsetree.KindToken, p.tok[0]))
		p.advance()

		if p.isStartOfExpression(0) {
			nt.AddChild(p.expression())
		} else {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing expression`)))
		}
	}

	return nt
}

// update parses a update statement.
func (p *Parser) update(withClause parsetree.NonTerminal) parsetree.NonTerminal {
	p.pushTree(parsetree.KindUpdate)
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
		p.addChild(p.orderByClause(func(t *token.Token) bool {
			return t.Kind == token.KindLimit || t.Kind == token.KindSemicolon || t.Kind == token.KindEOF
		}))
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

		if p.isStartOfExpression(0) {
			fn := parsetree.NewNonTerminal(parsetree.KindFileName)
			fn.AddChild(p.expression())
			nt.AddChild(fn)
		} else {
			nt.AddChild(parsetree.NewError(parsetree.KindErrorMissing, errors.New(`missing file name`)))
		}
	}

	return nt
}

func (p *Parser) pushTree(k parsetree.Kind) {
	p.treeStack = append(p.treeStack, parsetree.NewNonTerminal(k))
}

func (p *Parser) addChild(t parsetree.Construction) {
	p.treeStack[len(p.treeStack)-1].AddChild(t)
}

func (p *Parser) popTree() parsetree.NonTerminal {
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
		if !p.isPos(i+1, ks[i]) {
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
