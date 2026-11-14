// This package deals with the parsing of the SQL.
package parser

// TODO: implement the quote rule exceptions: lang_keywords.html.
// TODO: handle tables named TEMP, and also columns named TEMP.

import (
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
		p.alterTable()
	case token.KindAnalyze:
		p.analyze()
	case token.KindAttach:
		p.attach()
	case token.KindBegin:
		p.begin()
	case token.KindCommit, token.KindEnd:
		p.commit()
	case token.KindRollback:
		p.rollback()
	case token.KindCreate:
		p.create()
	case token.KindDelete:
		p.delete(nil)
	case token.KindDetach:
		p.detach()
	case token.KindDrop:
		p.drop()
	case token.KindInsert, token.KindReplace:
		p.insert(nil)
	case token.KindWith:
		p.with()
	case token.KindPragma:
		p.pragma()
	case token.KindReindex:
		p.reindex()
	case token.KindRelease:
		p.release()
	case token.KindSavepoint:
		p.savepoint()
	case token.KindSelect, token.KindValues:
		p.selectStatement(nil)
	case token.KindUpdate:
		p.update(nil)
	case token.KindVacuum:
		p.vacuum()
	default:
		p.errorUnexpected()
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
func (p *Parser) alterTable() {
	p.pushChild(parsetree.KindAlterTable)
	defer p.popTree()

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
			p.alterTableRenameTo()
		default:
			p.alterTableRenameColumn()
		}
	case token.KindAdd:
		p.alterTableAddColumn()
	case token.KindDrop:
		p.alterTableDropColumn()
	}
}

func (p *Parser) alterTableRenameTo() {
	p.pushChild(parsetree.KindRenameTo)
	defer p.popTree()

	p.term(token.KindRename)
	p.term(token.KindTo)
	p.termKind(parsetree.KindTableName, token.KindIdentifier)
}

func (p *Parser) alterTableRenameColumn() {
	p.pushChild(parsetree.KindRenameColumn)
	defer p.popTree()

	p.term(token.KindRename)

	if p.is(token.KindColumn) {
		p.term(token.KindColumn)
	}

	p.termKind(parsetree.KindColumnName, token.KindIdentifier)
	p.term(token.KindTo)
	p.termKind(parsetree.KindColumnName, token.KindIdentifier)
}

func (p *Parser) alterTableAddColumn() {
	p.pushChild(parsetree.KindAddColumn)
	defer p.popTree()

	p.term(token.KindAdd)

	if p.is(token.KindColumn) {
		p.term(token.KindColumn)
	}

	p.token(token.KindIdentifier)
	p.columnDefinition()
}

func (p *Parser) alterTableDropColumn() {
	p.pushChild(parsetree.KindDropColumn)
	defer p.popTree()

	p.term(token.KindDrop)

	if p.is(token.KindColumn) {
		p.term(token.KindColumn)
	}

	p.termKind(parsetree.KindColumnName, token.KindIdentifier)
}

func (p *Parser) columnDefinitionList() {
	p.pushChild(parsetree.KindCommaList)
	defer p.popTree()

	p.columnDefinition()
	for p.isSeq(token.KindComma, token.KindIdentifier) {
		p.term(token.KindComma)
		p.columnDefinition()
	}
}

// columnDefinition parses a column definition.
func (p *Parser) columnDefinition() {
	p.pushChild(parsetree.KindColumnDefinition)
	defer p.popTree()

	p.termKind(parsetree.KindColumnName, token.KindIdentifier)

	if p.is(token.KindIdentifier) {
		p.typeName()
	}

	for p.isAnyOf(token.KindConstraint, token.KindPrimary, token.KindNot, token.KindUnique, token.KindCheck,
		token.KindDefault, token.KindCollate, token.KindReferences, token.KindGenerated, token.KindAs) {
		p.columnConstraint()
	}
}

// typeName parses a type name.
func (p *Parser) typeName() {
	p.pushChild(parsetree.KindTypeName)
	defer p.popTree()

	p.term(token.KindIdentifier)
	for p.is(token.KindIdentifier) {
		p.term(token.KindIdentifier)
	}

	if !p.is(token.KindLeftParen) {
		return
	}

	p.term(token.KindLeftParen)
	p.signedNumber()

	if p.is(token.KindRightParen) {
		p.term(token.KindRightParen)
		return
	}

	p.term(token.KindComma)
	p.signedNumber()
	p.term(token.KindRightParen)
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
func (p *Parser) columnConstraint() {
	p.pushChild(parsetree.KindColumnConstraint)
	defer p.popTree()

	if p.is(token.KindConstraint) {
		p.term(token.KindConstraint)
		p.termKind(parsetree.KindConstraintName, token.KindIdentifier)
	}

	switch p.token(token.KindPrimary, token.KindNot, token.KindUnique, token.KindCheck, token.KindDefault,
		token.KindCollate, token.KindReferences, token.KindGenerated, token.KindAs) {
	case token.KindPrimary:
		p.primaryKeyColumnConstraint()
	case token.KindNot:
		p.notNullColumnConstraint()
	case token.KindUnique:
		p.uniqueColumnConstraint()
	case token.KindCheck:
		p.checkColumnConstraint()
	case token.KindDefault:
		p.defaultColumnConstraint()
	case token.KindCollate:
		p.collateColumnConstraint()
	case token.KindReferences:
		p.foreignKeyColumnConstraint()
	case token.KindGenerated, token.KindAs:
		p.generatedColumnConstraint()
	}
}

// primaryKeyColumnConstraint parses a primary key column constraint clause.
func (p *Parser) primaryKeyColumnConstraint() {
	p.pushChild(parsetree.KindPrimaryKeyColumnConstraint)
	defer p.popTree()

	p.term(token.KindPrimary)
	p.term(token.KindKey)

	if p.is(token.KindAsc) {
		p.term(token.KindAsc)
	} else if p.is(token.KindDesc) {
		p.term(token.KindDesc)
	}

	if p.is(token.KindOn) {
		p.conflictClause()
	}

	if p.is(token.KindAutoincrement) {
		p.term(token.KindAutoincrement)
	}
}

// notNullColumnConstraint parses a not null column constraint clause.
func (p *Parser) notNullColumnConstraint() {
	p.pushChild(parsetree.KindNotNullColumnConstraint)
	defer p.popTree()

	p.term(token.KindNot)
	p.term(token.KindNull)

	if p.is(token.KindOn) {
		p.conflictClause()
	}
}

// uniqueColumnConstraint parses a unique column constraint clause.
func (p *Parser) uniqueColumnConstraint() {
	p.pushChild(parsetree.KindUniqueColumnConstraint)
	defer p.popTree()

	p.term(token.KindUnique)

	if p.is(token.KindOn) {
		p.conflictClause()
	}
}

// checkColumnConstraint parses a check column constraint clause.
func (p *Parser) checkColumnConstraint() {
	p.pushChild(parsetree.KindCheckColumnConstraint)
	defer p.popTree()

	p.term(token.KindCheck)
	p.term(token.KindLeftParen)
	p.expression()
	p.term(token.KindRightParen)
}

// defaultColumnConstraint parses a default column constraint clause.
func (p *Parser) defaultColumnConstraint() {
	p.pushChild(parsetree.KindDefaultColumnConstraint)
	defer p.popTree()

	p.term(token.KindDefault)

	if p.is(token.KindLeftParen) {
		p.term(token.KindLeftParen)
		p.expression()
		p.term(token.KindRightParen)
	} else if p.isLiteralValue() {
		p.term()
	} else { // note that a numeric token is a literal value
		p.token(token.KindPlus, token.KindMinus)
		p.signedNumber()
	}
}

// collateColumnConstraint parses a collate column constraint clause.
func (p *Parser) collateColumnConstraint() {
	p.pushChild(parsetree.KindCollateColumnConstraint)
	defer p.popTree()

	p.term(token.KindCollate)
	p.termKind(parsetree.KindCollationName, token.KindIdentifier)
}

// generatedColumnConstraint parses a generated column constraint clause.
func (p *Parser) generatedColumnConstraint() {
	p.pushChild(parsetree.KindGeneratedColumnConstraint)
	defer p.popTree()

	if p.is(token.KindGenerated) {
		p.term()
		p.term(token.KindAlways)
	}

	p.term(token.KindAs)
	p.term(token.KindLeftParen)
	p.expression()
	p.term(token.KindRightParen)

	if p.isAnyOf(token.KindStored, token.KindVirtual) {
		p.term()
	}
}

// foreignKeyColumnConstraint parses a foreignKey column constraint clause.
func (p *Parser) foreignKeyColumnConstraint() {
	p.pushChild(parsetree.KindForeignKeyColumnConstraint)
	defer p.popTree()

	p.foreignKeyClause()
}

// foreignKeyClause parses a foreign key clause.
func (p *Parser) foreignKeyClause() {
	p.pushChild(parsetree.KindForeignKeyClause)
	defer p.popTree()

	p.term(token.KindReferences)
	p.termKind(parsetree.KindTableName, token.KindIdentifier)

	if p.is(token.KindLeftParen) {
		p.term(token.KindLeftParen)
		p.columnNameList()
		p.term(token.KindRightParen)
	}

	if !p.isAnyOf(token.KindOn, token.KindMatch, token.KindDeferrable, token.KindNot) {
		return
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
		return
	}

	if p.is(token.KindNot) {
		p.term()
	}
	p.term(token.KindDeferrable)

	if !p.is(token.KindInitially) {
		return
	}
	p.term()
	p.term(token.KindDeferred, token.KindImmediate)
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
func (p *Parser) conflictClause() {
	p.pushChild(parsetree.KindConflictClause)
	defer p.popTree()

	p.term(token.KindOn)
	p.term(token.KindConflict)
	p.term(token.KindRollback, token.KindAbort, token.KindFail, token.KindIgnore, token.KindReplace)
}

// analyze parses a analyze statement.
func (p *Parser) analyze() {
	p.pushChild(parsetree.KindAnalyze)
	defer p.popTree()

	p.term(token.KindAnalyze)

	if p.isSeq(token.KindIdentifier, token.KindDot) {
		p.termKind(parsetree.KindSchemaName, token.KindIdentifier)
		p.term(token.KindDot)
		p.termKind(parsetree.KindTableOrIndexName, token.KindIdentifier)
	} else {
		p.termKind(parsetree.KindSchemaIndexOrTableName, token.KindIdentifier)
	}
}

// attach parses a attach statement.
func (p *Parser) attach() {
	p.pushChild(parsetree.KindAttach)
	defer p.popTree()

	p.term(token.KindAttach)

	if p.is(token.KindDatabase) {
		p.term(token.KindDatabase)
	}

	p.expression()
	p.term(token.KindAs)
	p.termKind(parsetree.KindSchemaName, token.KindIdentifier)
}

// begin parses a begin statement.
func (p *Parser) begin() {
	p.pushChild(parsetree.KindBegin)
	defer p.popTree()

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
}

// commit parses a commit statement.
func (p *Parser) commit() {
	p.pushChild(parsetree.KindCommit)
	defer p.popTree()

	p.term(token.KindCommit, token.KindEnd)

	if p.is(token.KindTransaction) {
		p.term()
	}
}

// rollback parses a rollback statement.
func (p *Parser) rollback() {
	p.pushChild(parsetree.KindRollback)
	defer p.popTree()

	p.term(token.KindRollback)

	if p.is(token.KindTransaction) {
		p.term()
	}

	if !p.is(token.KindTo) {
		return
	}

	p.term(token.KindTo)
	if p.is(token.KindSavepoint) {
		p.term()
	}
	p.termKind(parsetree.KindSavepointName, token.KindIdentifier)
}

func (p *Parser) create() {
	if p.isSeq(token.KindCreate, token.KindIndex) || p.isSeq(token.KindCreate, token.KindUnique) {
		p.createIndex()
	} else if p.isSeq(token.KindCreate, token.KindTable) {
		p.createTable()
	} else if p.isSeq(token.KindCreate, token.KindTrigger) {
		p.createTrigger()
	} else if p.isSeq(token.KindCreate, token.KindView) {
		p.createView()
	} else if p.isSeq(token.KindCreate, token.KindTemp) || p.isSeq(token.KindCreate, token.KindTemporary) {
		if p.isPos(2, token.KindTable) {
			p.createTable()
		} else if p.isPos(2, token.KindTrigger) {
			p.createTrigger()
		} else if p.isPos(2, token.KindView) {
			p.createView()
		}
	} else { // if p.isSeq(token.KindCreate, token.KindVirtual)
		p.createVirtualTable()
	}
}

// createIndex parses a create index statement.
func (p *Parser) createIndex() {
	p.pushChild(parsetree.KindCreateIndex)
	defer p.popTree()

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
	p.indexedColumnExprList()
	p.term(token.KindRightParen)

	if p.is(token.KindWhere) {
		p.term()
		p.expression()
	}
}

func (p *Parser) indexedColumnList() {
	p.pushChild(parsetree.KindCommaList)
	defer p.popTree()

	p.indexedColumn()
	for p.is(token.KindComma) {
		p.term()
		p.indexedColumn()
	}
}

// indexedColumn parses a indexed column.
func (p *Parser) indexedColumn() {
	p.pushChild(parsetree.KindIndexedColumn)
	defer p.popTree()

	p.termKind(parsetree.KindColumnName, token.KindIdentifier)

	if p.is(token.KindCollate) {
		p.term()
		p.termKind(parsetree.KindCollationName, token.KindIdentifier)
	}

	if p.isAnyOf(token.KindAsc, token.KindDesc) {
		p.term()
	}
}

func (p *Parser) indexedColumnExprList() {
	p.pushChild(parsetree.KindCommaList)
	defer p.popTree()

	p.indexedColumnExpr()
	for p.is(token.KindComma) {
		p.term()
		p.indexedColumnExpr()
	}
}

func (p *Parser) indexedColumnExpr() {
	p.pushChild(parsetree.KindIndexedColumn)
	defer p.popTree()

	// the collate part is handled by the expression parser
	p.expression()

	if p.isAnyOf(token.KindAsc, token.KindDesc) {
		p.term()
	}
}

// createTable parses a create table statement.
func (p *Parser) createTable() {
	p.pushChild(parsetree.KindCreateTable)
	defer p.popTree()

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
}

func (p *Parser) createTableColumns() {
	p.term(token.KindLeftParen)
	p.columnDefinitionList()

	if p.is(token.KindComma) {
		p.term()
		p.tableConstraintList()
	}

	p.term(token.KindRightParen)

	if p.isAnyOf(token.KindWithout, token.KindStrict) {
		p.tableOptions()
	}
}

func (p *Parser) createTableSelect() {
	p.term(token.KindAs)

	// TODO: see if we can put p.withClause inside p.selectStatement
	var withClause parsetree.NonTerminal
	if p.is(token.KindWith) {
		withClause = p.withClause()
	}

	p.selectStatement(withClause)
}

// tableConstraintList parses a list of tale constraints in a create table.
func (p *Parser) tableConstraintList() {
	p.pushChild(parsetree.KindCommaList)
	defer p.popTree()

	p.tableConstraint()
	for p.is(token.KindComma) {
		p.term()
		p.tableConstraint()
	}
}

// tableConstraint parses a table constraint clause.
func (p *Parser) tableConstraint() {
	p.pushChild(parsetree.KindTableConstraint)
	defer p.popTree()

	if p.is(token.KindConstraint) {
		p.term()
		p.termKind(parsetree.KindConstraintName, token.KindIdentifier)
	}

	switch p.token(token.KindPrimary, token.KindUnique, token.KindCheck, token.KindForeign) {
	case token.KindPrimary:
		p.primaryKeyTableConstraint()
	case token.KindUnique:
		p.uniqueTableConstraint()
	case token.KindCheck:
		p.checkTableConstraint()
	case token.KindForeign:
		p.foreignKeyTableConstraint()
	}
}

// primaryKeyTableConstraint parses a primary key table constraint clause.
func (p *Parser) primaryKeyTableConstraint() {
	p.pushChild(parsetree.KindPrimaryKeyTableConstraint)
	defer p.popTree()

	p.term(token.KindPrimary)
	p.term(token.KindKey)
	p.term(token.KindLeftParen)
	p.indexedColumnList()
	p.term(token.KindRightParen)

	if p.is(token.KindOn) {
		p.conflictClause()
	}
}

// uniqueTableConstraint parses a unique table constraint clause.
func (p *Parser) uniqueTableConstraint() {
	p.pushChild(parsetree.KindUniqueTableConstraint)
	defer p.popTree()

	p.term(token.KindUnique)
	p.term(token.KindLeftParen)
	p.indexedColumnList()
	p.term(token.KindRightParen)

	if p.is(token.KindOn) {
		p.conflictClause()
	}
}

// checkTableConstraint parses a check table constraint clause.
func (p *Parser) checkTableConstraint() {
	p.pushChild(parsetree.KindCheckTableConstraint)
	defer p.popTree()

	p.term(token.KindCheck)
	p.term(token.KindLeftParen)
	p.expression()
	p.term(token.KindRightParen)
}

// foreignKeyTableConstraint parses a foreign key table constraint clause.
func (p *Parser) foreignKeyTableConstraint() {
	p.pushChild(parsetree.KindForeignKeyTableConstraint)
	defer p.popTree()

	p.term(token.KindForeign)
	p.term(token.KindKey)
	p.term(token.KindLeftParen)
	p.columnNameList()
	p.term(token.KindRightParen)
	p.foreignKeyClause()
}

// tableOptions parses a table options clause.
func (p *Parser) tableOptions() {
	p.pushChild(parsetree.KindCommaList)
	defer p.popTree()

	p.tableOption()
	for p.is(token.KindComma) {
		p.term()
		p.tableOption()
	}
}

// tableOption parses a table option.
func (p *Parser) tableOption() {
	p.pushChild(parsetree.KindTableOption)
	defer p.popTree()

	switch p.token(token.KindWithout, token.KindStrict) {
	case token.KindWithout:
		p.term()
		p.term(token.KindRowId)
	case token.KindStrict:
		p.term()
	}
}

// columnNameList parses a list of column names.
func (p *Parser) columnNameList() {
	p.pushChild(parsetree.KindCommaList)
	defer p.popTree()

	p.termKind(parsetree.KindColumnName, token.KindIdentifier)
	for p.is(token.KindComma) {
		p.term()
		p.termKind(parsetree.KindColumnName, token.KindIdentifier)
	}
}

// createTrigger parses a create trigger statement.
func (p *Parser) createTrigger() {
	p.pushChild(parsetree.KindCreateTrigger)
	defer p.popTree()

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
			p.columnNameList()
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
		p.expression()
	}

	p.triggerBody()
}

// triggerBody parses the body of a create trigger statement.
func (p *Parser) triggerBody() {
	p.pushChild(parsetree.KindTriggerBody)
	defer p.popTree()

	p.term(token.KindBegin)

	for {
		var withClause parsetree.NonTerminal
		if p.is(token.KindWith) {
			withClause = p.withClause()
		}

		switch p.token(token.KindDelete, token.KindInsert, token.KindSelect, token.KindUpdate) {
		case token.KindDelete:
			p.delete(withClause)
		case token.KindInsert:
			p.insert(withClause)
		case token.KindSelect:
			p.selectStatement(withClause)
		case token.KindUpdate:
			p.update(withClause)
		}

		p.term(token.KindSemicolon)

		if p.is(token.KindEnd) {
			break
		}
	}

	p.term(token.KindEnd)
}

// createView parses a create view statement.
func (p *Parser) createView() {
	p.pushChild(parsetree.KindCreateView)
	defer p.popTree()

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
		p.columnNameList()
		p.term(token.KindRightParen)
	}

	p.term(token.KindAs)

	var withClause parsetree.NonTerminal
	if p.is(token.KindWith) {
		withClause = p.withClause()
	}

	p.selectStatement(withClause)
}

// createVirtualTable parses a create virtual table statement.
func (p *Parser) createVirtualTable() {
	p.pushChild(parsetree.KindCreateVirtualTable)
	defer p.popTree()

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
		p.moduleArgumentList()
		p.term(token.KindRightParen)
	}
}

// moduleArgumentList parses a list of module arguments separated by comma.
func (p *Parser) moduleArgumentList() {
	p.pushChild(parsetree.KindCommaList)
	defer p.popTree()

	p.moduleArgument()
	for p.is(token.KindComma) {
		p.term()
		p.moduleArgument()
	}
}

// moduleArgument parses a module argument in a create virtual table statement.
func (p *Parser) moduleArgument() {
	p.pushChild(parsetree.KindModuleArgument)
	defer p.popTree()

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
func (p *Parser) delete(withClause parsetree.NonTerminal) {
	p.pushChild(parsetree.KindDelete)
	defer p.popTree()

	if withClause != nil {
		p.addChild(withClause)
	}

	p.term(token.KindDelete)
	p.term(token.KindFrom)
	p.qualifiedTableName()

	if p.is(token.KindWhere) {
		p.whereClause()
	}

	if p.is(token.KindReturning) {
		p.returningClause()
	}

	if p.is(token.KindOrder) {
		p.orderByClause()
	}

	if p.is(token.KindLimit) {
		p.limitClause()
	}
}

// withClause parses a with clause.
func (p *Parser) withClause() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindWithClause)

	p.term(token.KindWith)
	if p.is(token.KindRecursive) {
		p.term()
	}
	p.cteList()

	return p.popTree()
}

// cteList parses a list of CTEs in a with clause.
func (p *Parser) cteList() {
	p.pushChild(parsetree.KindCommaList)
	defer p.popTree()

	p.cte()
	for p.is(token.KindComma) {
		p.term()
		p.cte()
	}
}

// cte parses a common table expression.
func (p *Parser) cte() {
	p.pushChild(parsetree.KindCTE)
	defer p.popTree()

	p.termKind(parsetree.KindTableName, token.KindIdentifier)

	if p.is(token.KindLeftParen) {
		p.term()
		p.columnNameList()
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
	p.selectStatement(withClause)
	p.term(token.KindRightParen)
}

// qualifiedTableName parses a qualified table name.
func (p *Parser) qualifiedTableName() {
	p.pushChild(parsetree.KindQualifiedTableName)
	defer p.popTree()

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
}

// returningClause parses a returning clause.
func (p *Parser) returningClause() {
	p.pushChild(parsetree.KindReturningClause)
	defer p.popTree()

	p.term(token.KindReturning)
	p.returningItemList()
}

// returningItemList parses a list of itens in a returning clause.
func (p *Parser) returningItemList() {
	p.pushChild(parsetree.KindCommaList)
	defer p.popTree()

	p.returningItem()
	for p.is(token.KindComma) {
		p.term()
		p.returningItem()
	}
}

// returningItem parses a item in a returning clause.
func (p *Parser) returningItem() {
	p.pushChild(parsetree.KindReturningItem)
	defer p.popTree()

	if p.isStartOfExpression() {
		p.expression()
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
}

// whereClause parses a where clause.
func (p *Parser) whereClause() {
	p.pushChild(parsetree.KindWhereClause)
	defer p.popTree()

	p.term(token.KindWhere)
	p.expression()
}

// detach parses a detach statement.
func (p *Parser) detach() {
	p.pushChild(parsetree.KindDetach)
	defer p.popTree()

	p.term(token.KindDetach)
	if p.is(token.KindDatabase) {
		p.term()
	}
	p.termKind(parsetree.KindSchemaName, token.KindIdentifier, token.KindTemp)
}

func (p *Parser) drop() {
	switch p.tokenPos(1, token.KindIndex, token.KindTable, token.KindTrigger, token.KindView) {
	case token.KindIndex:
		p.dropIndex()
	case token.KindTable:
		p.dropTable()
	case token.KindTrigger:
		p.dropTrigger()
	default: // token.KindView
		p.dropView()
	}
}

// dropIndex parses a drop index statement.
func (p *Parser) dropIndex() {
	p.pushChild(parsetree.KindDropIndex)
	defer p.popTree()

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
}

// dropTable parses a drop table statement.
func (p *Parser) dropTable() {
	p.pushChild(parsetree.KindDropTable)
	defer p.popTree()

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
}

// dropTrigger parses a drop trigger statement.
func (p *Parser) dropTrigger() {
	p.pushChild(parsetree.KindDropTrigger)
	defer p.popTree()

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
}

// dropView parses a drop view statement.
func (p *Parser) dropView() {
	p.pushChild(parsetree.KindDropView)
	defer p.popTree()

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
}

func (p *Parser) expressionList() {
	p.pushChild(parsetree.KindCommaList)
	defer p.popTree()

	p.expression()
	for p.is(token.KindComma) {
		p.term()
		p.expression()
	}
}

// expression parses a expression.
func (p *Parser) expression() {
	if !p.isStartOfExpression() {
		p.errorUnexpected()
	}

	p.pushChild(parsetree.KindExpression)
	defer p.popTree()

	p.addChild(p.expression1())
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

	return p.expression4()
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

	p.addChild(p.expression4())
	for p.is(token.KindComma) {
		p.term()
		p.addChild(p.expression4())
	}

	return p.popTree()
}

// expression4 parses a expression with precedence at least 4.
func (p *Parser) expression4() parsetree.Construction {
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
			break
		}
	}

	return p.popConstr()
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
			p.selectStatement(withClause)
		} else if p.isStartOfExpression() {
			p.expressionList()
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
		if p.isAnyOfPos(1, token.KindSelect, token.KindWith) {
			return p.parenSelect()
		}
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
	p.functionArguments()
	p.term(token.KindRightParen)

	if p.is(token.KindFilter) {
		p.filterClause()
	}

	if p.is(token.KindOver) {
		p.overClause()
	}

	return p.popTree()
}

// functionCall parses the arguments of a function call.
func (p *Parser) functionArguments() {
	p.pushChild(parsetree.KindFunctionArguments)
	defer p.popTree()

	if !p.is(token.KindDistinct) && !p.isStartOfExpression() && !p.is(token.KindAsterisk) {
		return
	}

	if p.is(token.KindAsterisk) {
		p.term()
		return
	}

	if p.is(token.KindDistinct) {
		p.term()
	}
	p.expressionList()
	if p.is(token.KindOrder) {
		p.orderByClause()
	}
}

// orderByClause parses an order by clause.
func (p *Parser) orderByClause() {
	p.pushChild(parsetree.KindOrderByClause)
	defer p.popTree()

	p.term(token.KindOrder)
	p.term(token.KindBy)
	p.orderingTermList()
}

func (p *Parser) orderingTermList() {
	p.pushChild(parsetree.KindCommaList)
	defer p.popTree()

	p.orderingTerm()
	for p.is(token.KindComma) {
		p.term()
		p.orderingTerm()
	}
}

// orderingTerm parses an ordering term.
func (p *Parser) orderingTerm() {
	p.pushChild(parsetree.KindOrderingTerm)
	defer p.popTree()

	p.expression()

	if p.isAnyOf(token.KindAsc, token.KindDesc) {
		p.term()
	}

	if p.is(token.KindNulls) {
		p.term()
		p.term(token.KindFirst, token.KindLast)
	}
}

// filterClause parses a filter clause.
func (p *Parser) filterClause() {
	p.pushChild(parsetree.KindFilterClause)
	defer p.popTree()

	p.term(token.KindFilter)
	p.term(token.KindLeftParen)
	p.term(token.KindWhere)
	p.expression()
	p.term(token.KindRightParen)
}

// overClause parses a over clause.
func (p *Parser) overClause() {
	p.pushChild(parsetree.KindOverClause)
	defer p.popTree()

	p.term(token.KindOver)

	if p.is(token.KindIdentifier) {
		p.termKind(parsetree.KindWindowName, token.KindIdentifier)
	} else {
		p.windowDefinition()
	}
}

// windowDefinition parses a window definition.
func (p *Parser) windowDefinition() {
	p.pushChild(parsetree.KindWindowDefinition)
	defer p.popTree()

	p.term(token.KindLeftParen)

	if p.is(token.KindIdentifier) {
		p.termKind(parsetree.KindWindowName)
	}

	if p.is(token.KindPartition) {
		p.partitionBy()
	}

	if p.is(token.KindOrder) {
		p.orderByClause()
	}

	if p.isAnyOf(token.KindRange, token.KindRows, token.KindGroups) {
		p.frameSpec()
	}

	p.term(token.KindRightParen)
}

func (p *Parser) partitionBy() {
	p.pushChild(parsetree.KindPartitionBy)
	defer p.popTree()

	p.term(token.KindPartition)
	p.term(token.KindBy)
	p.expressionList()
}

// frameSpec parses a frame spec clause.
func (p *Parser) frameSpec() {
	p.pushChild(parsetree.KindFrameSpec)
	defer p.popTree()

	p.term(token.KindRange, token.KindRows, token.KindGroups)

	if p.is(token.KindBetween) {
		p.frameSpecBetween()
	} else if p.is(token.KindUnbounded) {
		p.term()
		p.term(token.KindPreceding)
	} else if p.is(token.KindCurrent) {
		p.term()
		p.term(token.KindRow)
	} else if p.isStartOfExpression() {
		p.expression()
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
}

// frameSpecBetween parses the between part of a frame spec.
func (p *Parser) frameSpecBetween() {
	p.pushChild(parsetree.KindFrameSpecBetween)
	defer p.popTree()

	p.term(token.KindBetween)

	if p.is(token.KindUnbounded) {
		p.term()
		p.term(token.KindPreceding)
	} else if p.is(token.KindCurrent) {
		p.term()
		p.term(token.KindRow)
	} else if p.isStartOfExpressionAtLeast4() {
		p.pushTreeKind(parsetree.KindExpression)
		p.addChild(p.expression4())
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
		p.addChild(p.expression4())
		p.addChild(p.popTree())

		p.term(token.KindPreceding, token.KindFollowing)
	} else {
		p.errorUnexpected()
	}
}

func (p *Parser) parenSelect() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindParenSelect)

	p.term(token.KindLeftParen)

	var withClause parsetree.NonTerminal
	if p.is(token.KindWith) {
		withClause = p.withClause()
	}
	p.selectStatement(withClause)

	p.term(token.KindRightParen)

	return p.popTree()
}

// parenExpression parses a expression enclosed in parenthesis.
func (p *Parser) parenExpression() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindParenExpression)

	p.term(token.KindLeftParen)
	p.expressionList()
	p.term(token.KindRightParen)

	return p.popTree()
}

// castExpression parses a cast expression.
func (p *Parser) castExpression() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindCast)

	p.term(token.KindCast)
	p.term(token.KindLeftParen)
	p.expression()
	p.term(token.KindAs)
	p.typeName()
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
	p.selectStatement(withClause)

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
		p.expression()
	}

	p.when()
	for p.is(token.KindWhen) {
		p.when()
	}

	if p.is(token.KindElse) {
		p.caseElse()
	}

	p.term(token.KindEnd)

	return p.popTree()
}

// when parses a when part of a case expression.
func (p *Parser) when() {
	p.pushChild(parsetree.KindWhen)
	defer p.popTree()

	p.term(token.KindWhen)
	p.expression()
	p.term(token.KindThen)
	p.expression()
}

// caseElse parses an else part of a case expression.
func (p *Parser) caseElse() {
	p.pushChild(parsetree.KindElse)
	defer p.popTree()

	p.term(token.KindElse)
	p.expression()
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
		p.errorMessage()
	}

	p.term(token.KindRightParen)

	return p.popTree()
}

func (p *Parser) errorMessage() {
	p.pushChild(parsetree.KindErrorMessage)
	defer p.popTree()
	p.expression()
}

// insert parses a insert statement.
func (p *Parser) insert(withClause parsetree.NonTerminal) {
	p.pushChild(parsetree.KindInsert)
	defer p.popTree()

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
		p.columnNameList()
		p.term(token.KindRightParen)
	}

	switch p.token(token.KindValues, token.KindWith, token.KindSelect, token.KindDefault) {
	case token.KindValues:
		p.term()
		p.addChild(p.insertValuesList())
		if p.is(token.KindOn) {
			p.upsertClause()
		}
	case token.KindWith, token.KindSelect:
		var withClause parsetree.NonTerminal
		if p.tok[0].Kind == token.KindWith {
			withClause = p.withClause()
		}
		p.selectStatement(withClause)

		if p.is(token.KindOn) {
			p.upsertClause()
		}
	default:
		p.term(token.KindDefault)
		p.term(token.KindValues)
	}

	if p.is(token.KindReturning) {
		p.returningClause()
	}
}

// insertValues parses the values list of a insert.
func (p *Parser) insertValuesList() parsetree.NonTerminal {
	p.pushTreeKind(parsetree.KindInsertValuesList)
	p.insertValuesItemList()
	return p.popTree()
}

func (p *Parser) insertValuesItemList() {
	p.pushChild(parsetree.KindCommaList)
	defer p.popTree()

	p.insertValuesItem()
	for p.is(token.KindComma) {
		p.term(token.KindComma)
		p.insertValuesItem()
	}
}

// insertValuesItem parses a item in a insert values list.
func (p *Parser) insertValuesItem() {
	p.pushChild(parsetree.KindInsertValuesItem)
	defer p.popTree()

	p.term(token.KindLeftParen)
	p.expressionList()
	p.term(token.KindRightParen)
}

// upsertClause parses a upsert clause.
func (p *Parser) upsertClause() {
	p.pushChild(parsetree.KindUpsertClause)
	defer p.popTree()

	p.upsertClauseItem()
	for p.is(token.KindOn) {
		p.upsertClauseItem()
	}
}

// upsertClauseItem parses a item of an upsert clause.
func (p *Parser) upsertClauseItem() {
	p.pushChild(parsetree.KindUpsertClauseItem)
	defer p.popTree()

	p.term(token.KindOn)
	p.term(token.KindConflict)

	if p.is(token.KindLeftParen) {
		p.term()
		p.indexedColumnExprList()
		p.term(token.KindRightParen)

		if p.is(token.KindWhere) {
			p.whereClause()
		}
	}

	p.term(token.KindDo)

	switch p.token(token.KindNothing, token.KindUpdate) {
	case token.KindNothing:
		p.term()
	default:
		p.term(token.KindUpdate)
		p.term(token.KindSet)
		p.updateSetItemList()

		if p.is(token.KindWhere) {
			p.whereClause()
		}
	}
}

func (p *Parser) updateSetItemList() {
	p.pushChild(parsetree.KindCommaList)
	defer p.popTree()

	p.updateSetItem()
	for p.is(token.KindComma) {
		p.term()
		p.updateSetItem()
	}
}

// updateSetItem parses a set item in a upsert or update clause.
func (p *Parser) updateSetItem() {
	p.pushChild(parsetree.KindUpdateSetItem)
	defer p.popTree()

	switch p.token(token.KindIdentifier, token.KindLeftParen) {
	case token.KindIdentifier:
		p.termKind(parsetree.KindColumnName)
	default: //token.KindLeftParen
		p.term()
		p.columnNameList()
		p.term(token.KindRightParen)
	}

	p.term(token.KindEqual)
	p.expression()
}

func (p *Parser) with() {
	withClause := p.withClause()
	switch p.token(token.KindDelete, token.KindInsert, token.KindReplace, token.KindSelect, token.KindUpdate) {
	case token.KindDelete:
		p.delete(withClause)
	case token.KindInsert, token.KindReplace:
		p.insert(withClause)
	case token.KindSelect:
		p.selectStatement(withClause)
	default: // token.KindUpdate:
		p.update(withClause)
	}
}

// pragma parses a pragma statement.
func (p *Parser) pragma() {
	p.pushChild(parsetree.KindPragma)
	defer p.popTree()

	p.term(token.KindPragma)

	if p.isAnyOfPos(0, token.KindIdentifier, token.KindTemp) && p.isPos(1, token.KindDot) {
		p.termKind(parsetree.KindSchemaName, token.KindIdentifier, token.KindTemp)
		p.term(token.KindDot)
	}
	p.termKind(parsetree.KindPragmaName, token.KindIdentifier, token.KindTemp)

	if p.is(token.KindEqual) {
		p.term()
		p.pragmaValue()
	} else if p.is(token.KindLeftParen) {
		p.term()
		p.pragmaValue()
		p.term(token.KindRightParen)
	}
}

// pragmaValue parses a pragma value.
func (p *Parser) pragmaValue() {
	p.pushChild(parsetree.KindPragmaValue)
	defer p.popTree()

	if p.isAnyOf(token.KindPlus, token.KindMinus) {
		p.term()
	}

	if p.isAnyOf(token.KindNumeric, token.KindIdentifier, token.KindString) || p.isKeyword() {
		p.term()
	} else {
		p.errorUnexpected()
	}
}

// reindex parses a reindex statement.
func (p *Parser) reindex() {
	p.pushChild(parsetree.KindReindex)
	defer p.popTree()

	p.term(token.KindReindex)

	if p.isAnyOf(token.KindIdentifier, token.KindTemp) && p.isPos(1, token.KindDot) {
		p.termKind(parsetree.KindSchemaName)
		p.term(token.KindDot)
		p.termKind(parsetree.KindTableOrIndexName, token.KindIdentifier)
	} else if p.is(token.KindIdentifier) {
		p.termKind(parsetree.KindCollationTableOrIndexName, token.KindIdentifier)
	}
}

// release parses a release statement.
func (p *Parser) release() {
	p.pushChild(parsetree.KindRelease)
	defer p.popTree()

	p.term(token.KindRelease)

	if p.is(token.KindSavepoint) {
		p.term()
	}

	p.termKind(parsetree.KindSavepointName, token.KindIdentifier)
}

// savepoint parses a savepoint statement.
func (p *Parser) savepoint() {
	p.pushChild(parsetree.KindSavepoint)
	defer p.popTree()

	p.term(token.KindSavepoint)
	p.termKind(parsetree.KindSavepointName, token.KindIdentifier)
}

// selectStatement parses a select statement.
func (p *Parser) selectStatement(withClause parsetree.NonTerminal) {
	p.pushChild(parsetree.KindSimpleSelect)
	defer p.popTree()

	if withClause != nil {
		p.addChild(withClause)
	}

	p.selectCore()

	if p.isAnyOf(token.KindUnion, token.KindIntersect, token.KindExcept) {
		p.changeTreeKind(parsetree.KindCompoundSelect)
		for p.isAnyOf(token.KindUnion, token.KindIntersect, token.KindExcept) {
			p.compoundOperator()
			p.selectCore()
		}
	}

	if p.is(token.KindOrder) {
		p.orderByClause()
	}

	if p.is(token.KindLimit) {
		p.limitClause()
	}
}

// selectCore parses a select core.
func (p *Parser) selectCore() {
	p.pushChild(parsetree.KindSelectCore)
	defer p.popTree()

	if p.is(token.KindValues) {
		p.valuesClause()
		return
	}

	p.term(token.KindSelect)

	if p.isAnyOf(token.KindAll, token.KindDistinct) {
		p.term()
	}

	p.resultColumnList()

	if p.is(token.KindFrom) {
		p.fromClause()
	}

	if p.is(token.KindWhere) {
		p.whereClause()
	}

	if p.is(token.KindGroup) {
		p.groupByClause()
	}

	if p.is(token.KindHaving) {
		p.havingClause()
	}

	if p.is(token.KindWindow) {
		p.windowClause()
	}
}

// resultColumnList parses a list of result columns.
func (p *Parser) resultColumnList() {
	p.pushChild(parsetree.KindCommaList)
	defer p.popTree()

	p.resultColumn()
	for p.is(token.KindComma) {
		p.term()
		p.resultColumn()
	}
}

// resultColumn parses a result column of a select statement.
func (p *Parser) resultColumn() {
	p.pushChild(parsetree.KindResultColumn)
	defer p.popTree()

	if p.is(token.KindAsterisk) {
		p.term()
	} else if p.isAnyOf(token.KindIdentifier, token.KindTemp) && p.isPos(1, token.KindDot) && p.isPos(2, token.KindAsterisk) {
		p.termKind(parsetree.KindTableName)
		p.term(token.KindDot)
		p.term(token.KindAsterisk)
	} else { // p.isStartOfExpression()
		p.expression()
		if p.is(token.KindAs) {
			p.term()
			p.termKind(parsetree.KindColumnAlias, token.KindIdentifier)
		} else if p.is(token.KindIdentifier) {
			p.termKind(parsetree.KindColumnAlias, token.KindIdentifier)
		}
	}
}

// fromClause parses a from clause.
func (p *Parser) fromClause() {
	p.pushChild(parsetree.KindFromClause)
	defer p.popTree()

	p.term(token.KindFrom)
	p.joinClause()
}

// joinClause parses a join clause.
func (p *Parser) joinClause() {
	p.pushChild(parsetree.KindJoinClause)
	defer p.popTree()

	p.tableOrSubquery()

	for p.isAnyOf(token.KindComma, token.KindCross, token.KindFull, token.KindInner, token.KindLeft,
		token.KindNatural, token.KindOuter, token.KindRight, token.KindJoin) {
		p.joinOperator()
		p.tableOrSubquery()

		if p.isAnyOf(token.KindOn, token.KindUsing) {
			p.joinConstraint()
		}
	}
}

// joinOperator parses a join operator.
func (p *Parser) joinOperator() {
	p.pushChild(parsetree.KindJoinOperator)
	defer p.popTree()

	if p.is(token.KindComma) {
		p.term()
		return
	}

	for range 3 {
		if p.isAnyOf(token.KindCross, token.KindFull, token.KindInner, token.KindLeft, token.KindNatural,
			token.KindOuter, token.KindRight) {
			p.term()
		} else {
			break
		}
	}

	p.term(token.KindJoin)
}

// joinConstraint parses a join constraint.
func (p *Parser) joinConstraint() {
	p.pushChild(parsetree.KindJoinConstraint)
	defer p.popTree()

	if p.is(token.KindOn) {
		p.term()
		p.expression()
	} else if p.is(token.KindUsing) {
		p.term()
		p.term(token.KindLeftParen)
		p.columnNameList()
		p.term(token.KindRightParen)
	}
}

// tableOrSubquery parses a table-or-subquery clause.
func (p *Parser) tableOrSubquery() {
	p.pushChild(parsetree.KindTableOrSubquery)
	defer p.popTree()

	if p.isAnyOf(token.KindIdentifier, token.KindTemp) {
		p.tableOrSubquery_table()
		return
	}

	p.term(token.KindLeftParen)

	switch p.token(token.KindWith, token.KindSelect, token.KindIdentifier, token.KindTemp, token.KindLeftParen) {
	case token.KindWith, token.KindSelect:
		var withClause parsetree.NonTerminal
		if p.is(token.KindWith) {
			withClause = p.withClause()
		}

		p.selectStatement(withClause)

		p.term(token.KindRightParen)

		if p.is(token.KindAs) {
			p.term()
			p.termKind(parsetree.KindTableAlias, token.KindIdentifier)
		} else if p.is(token.KindIdentifier) {
			p.termKind(parsetree.KindTableAlias, token.KindIdentifier)
		}
	case token.KindIdentifier, token.KindTemp, token.KindLeftParen:
		p.joinClause()
		p.term(token.KindRightParen)
	}
}

// tableOrSubquery_table parses a table-or-subquery that is a table.
func (p *Parser) tableOrSubquery_table() {
	if p.isAnyOf(token.KindIdentifier, token.KindTemp) && p.isPos(1, token.KindDot) {
		p.termKind(parsetree.KindSchemaName)
		p.term(token.KindDot)
	}

	if p.isPos(1, token.KindLeftParen) {
		p.termKind(parsetree.KindTableFunctionName, token.KindIdentifier, token.KindTemp)

		p.term(token.KindLeftParen)
		p.expressionList()
		p.term(token.KindRightParen)

		if p.is(token.KindAs) {
			p.term()
			p.termKind(parsetree.KindTableAlias, token.KindIdentifier)
		} else if p.is(token.KindIdentifier) {
			p.termKind(parsetree.KindTableAlias, token.KindIdentifier)
		}
		return
	}

	p.termKind(parsetree.KindTableName, token.KindIdentifier, token.KindTemp)

	if p.is(token.KindAs) {
		p.term()
		p.termKind(parsetree.KindTableAlias, token.KindIdentifier)
	} else if p.is(token.KindIdentifier) {
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
}

// groupByClause parses a group by clause.
func (p *Parser) groupByClause() {
	p.pushChild(parsetree.KindGroupByClause)
	defer p.popTree()

	p.term(token.KindGroup)
	p.term(token.KindBy)
	p.expressionList()
}

// havingClause parses a having clause.
func (p *Parser) havingClause() {
	p.pushChild(parsetree.KindHavingClause)
	defer p.popTree()

	p.term(token.KindHaving)
	p.expression()
}

// windowClause parses a window clause.
func (p *Parser) windowClause() {
	p.pushChild(parsetree.KindWindowClause)
	defer p.popTree()

	p.term(token.KindWindow)
	p.windowClauseItemList()
}

func (p *Parser) windowClauseItemList() {
	p.pushChild(parsetree.KindCommaList)
	defer p.popTree()

	p.windowClauseItem()
	for p.is(token.KindComma) {
		p.term(token.KindComma)
		p.windowClauseItem()
	}
}

// windowClauseItem parses a window declaration.
func (p *Parser) windowClauseItem() {
	p.pushChild(parsetree.KindWindowClauseItem)
	defer p.popTree()

	p.termKind(parsetree.KindWindowName, token.KindIdentifier)
	p.term(token.KindAs)
	p.windowDefinition()
}

// valuesClause parses a values clause.
func (p *Parser) valuesClause() {
	p.pushChild(parsetree.KindValuesClause)
	defer p.popTree()

	p.term(token.KindValues)
	p.valuesItemList()
}

func (p *Parser) valuesItemList() {
	p.pushChild(parsetree.KindCommaList)
	defer p.popTree()

	p.valuesItem()
	for p.is(token.KindComma) {
		p.term()
		p.valuesItem()
	}
}

// valuesItem parses a item of a values clause.
func (p *Parser) valuesItem() {
	p.pushChild(parsetree.KindValuesItem)
	defer p.popTree()

	p.term(token.KindLeftParen)
	p.expressionList()
	p.term(token.KindRightParen)
}

// compoundOperator parses a item of a compound operator.
func (p *Parser) compoundOperator() {
	p.pushChild(parsetree.KindCompoundOperator)
	defer p.popTree()

	switch p.token(token.KindUnion, token.KindIntersect, token.KindExcept) {
	case token.KindUnion:
		p.term()
		if p.is(token.KindAll) {
			p.term()
		}
	default: // token.KindIntersect, token.KindExcept
		p.term()
	}
}

// limitClause parses a limit clause.
func (p *Parser) limitClause() {
	p.pushChild(parsetree.KindLimitClause)
	defer p.popTree()

	p.term(token.KindLimit)
	p.expression()

	if p.isAnyOf(token.KindOffset, token.KindComma) {
		p.term()
		p.expression()
	}
}

// update parses a update statement.
func (p *Parser) update(withClause parsetree.NonTerminal) {
	p.pushChild(parsetree.KindUpdate)
	defer p.popTree()

	if withClause != nil {
		p.addChild(withClause)
	}

	p.term(token.KindUpdate)

	if p.is(token.KindOr) {
		p.term()
		p.term(token.KindAbort, token.KindFail, token.KindIgnore, token.KindReplace, token.KindRollback)
	}

	p.qualifiedTableName()

	p.term(token.KindSet)
	p.updateSetItemList()

	if p.is(token.KindFrom) {
		p.fromClause()
	}

	if p.is(token.KindWhere) {
		p.whereClause()
	}

	if p.is(token.KindReturning) {
		p.returningClause()
	}

	if p.is(token.KindOrder) {
		p.orderByClause()
	}

	if p.is(token.KindLimit) {
		p.limitClause()
	}
}

// vacuum parses a vacuum statement.
func (p *Parser) vacuum() {
	p.pushChild(parsetree.KindVacuum)
	defer p.popTree()

	p.term(token.KindVacuum)

	if p.isAnyOf(token.KindIdentifier, token.KindTemp) {
		p.termKind(parsetree.KindSchemaName)
	}

	if p.is(token.KindInto) {
		p.term()
		p.pushTreeKind(parsetree.KindFileName)
		p.expression()
		p.addChild(p.popTree())
	}
}

func (p *Parser) pushChild(k parsetree.Kind) {
	t := parsetree.NewNonTerminal(k)
	p.addChild(t)
	p.pushTree(t)
}

func (p *Parser) pushTree(t parsetree.Construction) {
	p.treeStack = append(p.treeStack, t)
}

func (p *Parser) pushTreeKind(k parsetree.Kind) {
	p.pushTree(parsetree.NewNonTerminal(k))
}

// changeTreeKind sets to k the kind of the parse tree at top of treeStack.
func (p *Parser) changeTreeKind(k parsetree.Kind) {
	p.treeStack[len(p.treeStack)-1].SetKind(k)
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
