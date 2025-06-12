package sqlgen

import (
	"fmt"
	"iter"
	"strings"

	"github.com/joaobnv/mel/dit/token"
)

type syntaxGenerator interface {
	gen(stack []*syntaxGenerator, cfg *SyntaxConfig, s *syntax) iter.Seq[string]
	// firstTokenKind returns the first token kind of the previous string yielded by gen. Whitespace are ignored. The
	// kind will be nil only if the empty string was yielded.
	firstTokenKind() *token.Kind
	// firstTokenKind returns the last token kind of the previous string yielded by gen. Whitespace are ignored. The
	// kind will be nil only if the empty string was yielded.
	lastTokenKind() *token.Kind
}

type syntax struct {
	ss      *syntaxGenerator
	an      *syntaxGenerator
	bg      *syntaxGenerator
	com     *syntaxGenerator
	de      *syntaxGenerator
	di      *syntaxGenerator
	dt      *syntaxGenerator
	dg      *syntaxGenerator
	dv      *syntaxGenerator
	pg      *syntaxGenerator
	pv      *syntaxGenerator
	ri      *syntaxGenerator
	rel     *syntaxGenerator
	rol     *syntaxGenerator
	sp      *syntaxGenerator
	selStmt *syntaxGenerator
	cte     *syntaxGenerator
	co      *syntaxGenerator
	sc      *syntaxGenerator
	rcl     *syntaxGenerator
	rc      *syntaxGenerator
	tosl    *syntaxGenerator
	tos     *syntaxGenerator
	jc      *syntaxGenerator
	jo      *syntaxGenerator
	jco     *syntaxGenerator
	wdl     *syntaxGenerator
	wd      *syntaxGenerator
	wdef    *syntaxGenerator
	vac     *syntaxGenerator
	exp     *syntaxGenerator
	el      *syntaxGenerator
	fnc     *syntaxGenerator
	fa      *syntaxGenerator
	ot      *syntaxGenerator
	otl     *syntaxGenerator
	fc      *syntaxGenerator
	oc      *syntaxGenerator
	fs      *syntaxGenerator
	tn      *syntaxGenerator
	sn      *syntaxGenerator
	sl      *syntaxGenerator
}

func newSyntax() *syntax {
	s := new(syntax)
	s.build()
	return s
}

func (s *syntax) build() {
	s.ss = new(syntaxGenerator)
	s.an = new(syntaxGenerator)
	s.bg = new(syntaxGenerator)
	s.com = new(syntaxGenerator)
	s.de = new(syntaxGenerator)
	s.di = new(syntaxGenerator)
	s.dt = new(syntaxGenerator)
	s.dg = new(syntaxGenerator)
	s.dv = new(syntaxGenerator)
	s.pg = new(syntaxGenerator)
	s.pv = new(syntaxGenerator)
	s.ri = new(syntaxGenerator)
	s.rel = new(syntaxGenerator)
	s.rol = new(syntaxGenerator)
	s.sp = new(syntaxGenerator)
	s.selStmt = new(syntaxGenerator)
	s.cte = new(syntaxGenerator)
	s.co = new(syntaxGenerator)
	s.sc = new(syntaxGenerator)
	s.rcl = new(syntaxGenerator)
	s.rc = new(syntaxGenerator)
	s.tosl = new(syntaxGenerator)
	s.tos = new(syntaxGenerator)
	s.jc = new(syntaxGenerator)
	s.jo = new(syntaxGenerator)
	s.jco = new(syntaxGenerator)
	s.wdl = new(syntaxGenerator)
	s.wd = new(syntaxGenerator)
	s.wdef = new(syntaxGenerator)
	s.vac = new(syntaxGenerator)
	s.exp = new(syntaxGenerator)
	s.el = new(syntaxGenerator)
	s.fnc = new(syntaxGenerator)
	s.fa = new(syntaxGenerator)
	s.ot = new(syntaxGenerator)
	s.otl = new(syntaxGenerator)
	s.fc = new(syntaxGenerator)
	s.oc = new(syntaxGenerator)
	s.fs = new(syntaxGenerator)
	s.tn = new(syntaxGenerator)
	s.sn = new(syntaxGenerator)
	s.sl = new(syntaxGenerator)

	s.buildSqlStmt()
	s.buildAnalyze()
	s.buildBegin()
	s.buildCommit()
	s.buildDetach()
	s.buildDropIndex()
	s.buildDropTable()
	s.buildDropTrigger()
	s.buildDropView()
	s.buildPragma()
	s.buildPragmaValue()
	s.buildReindex()
	s.buildRelease()
	s.buildRollback()
	s.buildSavepoint()
	s.buildSelectStatement()
	s.buildCommonTableExpression()
	s.buildCompoundOperator()
	s.buildSelectCore()
	s.buildResultColumnList()
	s.buildResultColumn()
	s.buildTableOrSubqueryList()
	s.buildTableOrSubquery()
	s.buildJoinClause()
	s.buildJoinOperator()
	s.buildJoinConstraint()
	s.buildWindowDeclarationList()
	s.buildWindowDeclaration()
	s.buildWindowDefinition()
	s.buildVacuum()
	s.buildExpression()
	s.buildExpressionList()
	s.buildFunctionCall()
	s.buildFunctionArguments()
	s.buildOrderingTerm()
	s.buildOrderingTermList()
	s.buildFilterClause()
	s.buildOverClause()
	s.buildFrameSpec()
	s.buildTypeName()
	s.buildSignedNumber()
	s.buildSignedLiteral()
}

func (s *syntax) sqlStmt() *syntaxGenerator {
	return s.ss
}

func (s *syntax) buildSqlStmt() {
	*s.ss = *s.conc(
		s.opt(s.kw(token.KindExplain)),
		s.opt(s.conc(s.kw(token.KindQuery), s.kw(token.KindPlan))),
		s.or(
			s.analyze(), s.begin(), s.commit(), s.detach(), s.dropIndex(), s.dropTable(),
			s.dropTrigger(), s.dropView(), s.pragma(), s.reindex(), s.release(), s.rollback(),
			s.savepoint(), s.vacuum(),
		),
	)
}

func (s *syntax) analyze() *syntaxGenerator {
	return s.an
}

func (s *syntax) buildAnalyze() {
	*s.an = *s.conc(
		s.kw(token.KindAnalyze),
		s.opt(
			s.conc(
				s.schemaName(),
				s.opt(s.conc(s.oper(token.KindDot), s.id()))),
		),
	)
}

func (s *syntax) begin() *syntaxGenerator {
	return s.bg
}

func (s *syntax) buildBegin() {
	*s.bg = *s.conc(
		s.kw(token.KindBegin),
		s.or(
			s.eps(), s.kw(token.KindDeferred),
			s.kw(token.KindImmediate), s.kw(token.KindExclusive),
		),
		s.opt(s.kw(token.KindTransaction)),
	)
}

func (s *syntax) commit() *syntaxGenerator {
	return s.com
}

func (s *syntax) buildCommit() {
	*s.com = *s.conc(
		s.or(s.kw(token.KindCommit), s.kw(token.KindEnd)),
		s.opt(s.kw(token.KindTransaction)),
	)
}

func (s *syntax) detach() *syntaxGenerator {
	return s.de
}

func (s *syntax) buildDetach() {
	*s.de = *s.conc(
		s.kw(token.KindDetach),
		s.opt(s.kw(token.KindDatabase)),
		s.schemaName(),
	)
}

func (s *syntax) dropIndex() *syntaxGenerator {
	return s.di
}

func (s *syntax) buildDropIndex() {
	*s.di = *s.conc(
		s.kw(token.KindDrop),
		s.kw(token.KindIndex),
		s.opt(s.conc(s.kw(token.KindIf), s.kw(token.KindExists))),
		s.opt(s.conc(s.schemaName(), s.oper(token.KindDot))),
		s.id(),
	)
}

func (s *syntax) dropTable() *syntaxGenerator {
	return s.dt
}

func (s *syntax) buildDropTable() {
	*s.dt = *s.conc(
		s.kw(token.KindDrop),
		s.kw(token.KindTable),
		s.opt(s.conc(s.kw(token.KindIf), s.kw(token.KindExists))),
		s.opt(s.conc(s.schemaName(), s.oper(token.KindDot))),
		s.id(),
	)
}

func (s *syntax) dropTrigger() *syntaxGenerator {
	return s.dg
}

func (s *syntax) buildDropTrigger() {
	*s.dg = *s.conc(
		s.kw(token.KindDrop),
		s.kw(token.KindTrigger),
		s.opt(s.conc(s.kw(token.KindIf), s.kw(token.KindExists))),
		s.opt(s.conc(s.schemaName(), s.oper(token.KindDot))),
		s.id(),
	)
}

func (s *syntax) dropView() *syntaxGenerator {
	return s.dv
}

func (s *syntax) buildDropView() {
	*s.dv = *s.conc(
		s.kw(token.KindDrop),
		s.kw(token.KindView),
		s.opt(s.conc(s.kw(token.KindIf), s.kw(token.KindExists))),
		s.opt(s.conc(s.schemaName(), s.oper(token.KindDot))),
		s.id(),
	)
}

func (s *syntax) pragma() *syntaxGenerator {
	return s.pg
}

func (s *syntax) buildPragma() {
	*s.pg = *s.conc(
		s.kw(token.KindPragma),
		s.opt(s.conc(s.schemaName(), s.oper(token.KindDot))),
		s.id(),
		s.or(
			s.eps(),
			s.conc(s.oper(token.KindEqual), s.pragmaValue()),
			s.conc(s.punct(token.KindLeftParen), s.pragmaValue(), s.punct(token.KindRightParen)),
		),
	)
}

func (s *syntax) pragmaValue() *syntaxGenerator {
	return s.pv
}

func (s *syntax) buildPragmaValue() {
	*s.pv = *s.or(s.signedNumber(), s.id(), s.signedLiteral())
}

func (s *syntax) reindex() *syntaxGenerator {
	return s.ri
}

func (s *syntax) buildReindex() {
	*s.ri = *s.conc(
		s.kw(token.KindReindex),
		s.opt(s.conc(
			s.opt(s.conc(s.schemaName(), s.oper(token.KindDot))),
			s.id(),
		)),
	)
}

func (s *syntax) release() *syntaxGenerator {
	return s.rel
}

func (s *syntax) buildRelease() {
	*s.rel = *s.conc(
		s.kw(token.KindRelease),
		s.opt(s.kw(token.KindSavepoint)),
		s.id(),
	)
}

func (s *syntax) rollback() *syntaxGenerator {
	return s.rol
}

func (s *syntax) buildRollback() {
	*s.rol = *s.conc(
		s.kw(token.KindRollback),
		s.opt(s.kw(token.KindTransaction)),
		s.opt(s.conc(
			s.kw(token.KindTo),
			s.opt(s.kw(token.KindSavepoint)),
			s.id(),
		)),
	)
}

func (s *syntax) savepoint() *syntaxGenerator {
	return s.sp
}

func (s *syntax) buildSavepoint() {
	*s.sp = *s.conc(
		s.kw(token.KindSavepoint),
		s.id(),
	)
}

func (s *syntax) selectStatement() *syntaxGenerator {
	return s.selStmt
}

func (s *syntax) buildSelectStatement() {
	*s.selStmt = *s.conc(
		s.opt(s.conc(
			s.kw(token.KindWith),
			s.opt(s.kw(token.KindRecursive)),
			s.commonTableExpression(),
			s.star(s.conc(
				s.punct(token.KindComma),
				s.commonTableExpression(),
			)),
		)),
		s.selectCore(),
		s.star(s.conc(
			s.compoundOperator(),
			s.selectCore(),
		)),
		s.opt(s.conc(
			s.kw(token.KindOrder),
			s.kw(token.KindBy),
			s.orderingTermList(),
		)),
		s.opt(s.conc(
			s.kw(token.KindLimit),
			s.expression(),
			s.or(
				s.eps(),
				s.conc(s.kw(token.KindOffset), s.expression()),
				s.conc(s.punct(token.KindComma), s.expression()),
			),
		)),
	)
}

func (s *syntax) commonTableExpression() *syntaxGenerator {
	return s.cte
}

func (s *syntax) buildCommonTableExpression() {
	*s.cte = *s.conc(
		s.id(),
		s.opt(s.conc(
			s.punct(token.KindLeftParen),
			s.id(),
			s.star(s.conc(s.punct(token.KindComma), s.id())),
			s.punct(token.KindRightParen),
		)),
		s.kw(token.KindAs),
		s.opt(s.conc(
			s.opt(s.kw(token.KindNot)),
			s.kw(token.KindMaterialized),
		)),
		s.punct(token.KindLeftParen),
		s.selectStatement(),
		s.punct(token.KindRightParen),
	)
}

func (s *syntax) compoundOperator() *syntaxGenerator {
	return s.co
}

func (s *syntax) buildCompoundOperator() {
	*s.co = *s.or(
		s.kw(token.KindUnion),
		s.conc(s.kw(token.KindUnion), s.kw(token.KindAll)),
		s.kw(token.KindIntersect),
		s.kw(token.KindExcept),
	)
}

func (s *syntax) selectCore() *syntaxGenerator {
	return s.sc
}

func (s *syntax) buildSelectCore() {
	*s.sc = *s.or(
		s.conc(
			s.kw(token.KindSelect),
			s.or(s.eps(), s.kw(token.KindDistinct), s.kw(token.KindAll)),
			s.resultColumnList(),
			s.opt(s.conc(s.kw(token.KindFrom), s.or(s.tableOrSubqueryList(), s.joinClause()))),
			s.opt(s.conc(s.kw(token.KindWhere), s.expression())),
			s.opt(s.conc(s.kw(token.KindGroup), s.kw(token.KindBy), s.expressionList())),
			s.opt(s.conc(s.kw(token.KindHaving), s.expression())),
			s.opt(s.conc(s.kw(token.KindWindow), s.windowDeclarationList())),
		),
		s.conc(
			s.kw(token.KindValues),
			s.conc(
				s.conc(
					s.punct(token.KindLeftParen),
					s.expressionList(),
					s.punct(token.KindRightParen),
				),
				s.star(
					s.conc(
						s.punct(token.KindComma),
						s.punct(token.KindLeftParen),
						s.expressionList(),
						s.punct(token.KindRightParen),
					),
				),
			),
		),
	)
}

func (s *syntax) resultColumnList() *syntaxGenerator {
	return s.rcl
}

func (s *syntax) buildResultColumnList() {
	*s.rcl = *s.conc(
		s.resultColumn(),
		s.star(s.conc(s.punct(token.KindComma), s.resultColumn())),
	)
}

func (s *syntax) resultColumn() *syntaxGenerator {
	return s.rc
}

func (s *syntax) buildResultColumn() {
	*s.rc = *s.or(
		s.conc(s.expression(), s.opt(s.conc(s.opt(s.kw(token.KindAs)), s.id()))),
		s.oper(token.KindAsterisk),
		s.conc(s.id(), s.oper(token.KindDot), s.oper(token.KindAsterisk)),
	)
}

func (s *syntax) tableOrSubqueryList() *syntaxGenerator {
	return s.tosl
}

func (s *syntax) buildTableOrSubqueryList() {
	*s.tosl = *s.conc(
		s.tableOrSubquery(),
		s.star(s.conc(s.punct(token.KindComma), s.tableOrSubquery())),
	)
}

func (s *syntax) tableOrSubquery() *syntaxGenerator {
	return s.tos
}

func (s *syntax) buildTableOrSubquery() {
	*s.tos = *s.or(
		s.conc(
			s.opt(s.conc(s.schemaName(), s.oper(token.KindDot))),
			s.or(
				s.conc(
					s.id(),
					s.opt(s.conc(s.opt(s.kw(token.KindAs)), s.id())),
					s.opt(
						s.or(
							s.conc(s.kw(token.KindIndexed), s.kw(token.KindBy), s.id()),
							s.conc(s.kw(token.KindNot), s.kw(token.KindIndexed)),
						),
					),
				),
				s.conc(
					s.id(),
					s.punct(token.KindLeftParen),
					s.expressionList(),
					s.punct(token.KindRightParen),
					s.opt(s.conc(s.opt(s.kw(token.KindAs)), s.id())),
				),
			),
		),
		s.conc(
			s.punct(token.KindLeftParen),
			s.selectStatement(),
			s.punct(token.KindRightParen),
		),
		s.conc(
			s.punct(token.KindLeftParen),
			s.or(s.tableOrSubqueryList(), s.joinClause()),
			s.punct(token.KindRightParen),
		),
	)
}

func (s *syntax) joinClause() *syntaxGenerator {
	return s.jc
}

func (s *syntax) buildJoinClause() {
	*s.jc = *s.conc(
		s.tableOrSubquery(),
		s.star(s.conc(s.joinOperator(), s.tableOrSubquery(), s.joinConstraint())),
	)
}

func (s *syntax) joinOperator() *syntaxGenerator {
	return s.jo
}

func (s *syntax) buildJoinOperator() {
	*s.jo = *s.or(
		s.punct(token.KindComma),
		s.conc(
			s.opt(s.joinOperatorAttribute()),
			s.opt(s.joinOperatorAttribute()),
			s.opt(s.joinOperatorAttribute()),
			s.kw(token.KindJoin),
		),
	)
}

// attribute is the "blah"
func (s *syntax) joinOperatorAttribute() *syntaxGenerator {
	return s.or(
		s.kw(token.KindCross),
		s.kw(token.KindFull),
		s.kw(token.KindInner),
		s.kw(token.KindLeft),
		s.kw(token.KindNatural),
		s.kw(token.KindOuter),
		s.kw(token.KindRight),
	)
}

func (s *syntax) joinConstraint() *syntaxGenerator {
	return s.jco
}

func (s *syntax) buildJoinConstraint() {
	*s.jco = *s.or(
		s.conc(s.kw(token.KindOn), s.expression()),
		s.conc(s.kw(token.KindUsing), s.punct(token.KindLeftParen), s.idList(), s.punct(token.KindRightParen)),
	)
}

func (s *syntax) windowDeclarationList() *syntaxGenerator {
	return s.wdl
}

func (s *syntax) buildWindowDeclarationList() {
	*s.wdl = *s.conc(
		s.windowDeclaration(),
		s.star(s.conc(s.punct(token.KindComma), s.windowDeclaration())),
	)
}

func (s *syntax) windowDeclaration() *syntaxGenerator {
	return s.wd
}

func (s *syntax) buildWindowDeclaration() {
	*s.wd = *s.conc(
		s.id(), s.kw(token.KindAs), s.windowDefinition(),
	)
}

func (s *syntax) windowDefinition() *syntaxGenerator {
	return s.wdef
}

func (s *syntax) buildWindowDefinition() {
	*s.wdef = *s.conc(
		s.punct(token.KindLeftParen),
		s.opt(s.id()),
		s.opt(s.conc(s.kw(token.KindPartition), s.kw(token.KindBy), s.expressionList())),
		s.opt(s.conc(s.kw(token.KindOrder), s.kw(token.KindBy), s.orderingTermList())),
		s.opt(s.frameSpec()),
		s.punct(token.KindRightParen),
	)
}

func (s *syntax) vacuum() *syntaxGenerator {
	return s.vac
}

func (s *syntax) buildVacuum() {
	*s.vac = *s.conc(
		s.kw(token.KindVacuum),
		s.opt(s.schemaName()),
		s.opt(s.conc(s.kw(token.KindInto), s.expression())),
	)
}

func (s *syntax) expression() *syntaxGenerator {
	return s.exp
}

func (s *syntax) buildExpression() {
	*s.exp = *s.or(
		s.literal(),
		s.variable(),
		s.conc(
			s.opt(s.conc(s.schemaName(), s.oper(token.KindDot))),
			s.opt(s.conc(s.id(), s.oper(token.KindDot))),
			s.id(),
		),
		s.conc(s.unaryOper(), s.expression()),
		s.conc(s.expression(), s.binaryOper(), s.expression()),
		s.functionCall(),
		s.conc(s.punct(token.KindLeftParen), s.expressionList(), s.punct(token.KindRightParen)),
		s.conc(
			s.kw(token.KindCast),
			s.punct(token.KindLeftParen),
			s.expression(), s.kw(token.KindAs), s.typeName(),
			s.punct(token.KindRightParen),
		),
		s.conc(s.expression(), s.kw(token.KindCollate), s.id()),
		s.conc(
			s.expression(), s.opt(s.kw(token.KindNot)),
			s.or(
				s.conc(s.kw(token.KindLike), s.expression(), s.opt(s.conc(s.kw(token.KindEscape), s.expression()))),
				s.conc(s.kw(token.KindGlob), s.expression()),
				s.conc(s.kw(token.KindRegexp), s.expression()),
				s.conc(s.kw(token.KindMatch), s.expression()),
			),
		),
		s.conc(s.expression(), s.or(
			s.kw(token.KindIsnull),
			s.kw(token.KindNotnull),
			s.conc(s.kw(token.KindNot), s.kw(token.KindNull)),
		)),
		s.conc(s.expression(), s.kw(token.KindIs), s.opt(s.kw(token.KindNot)),
			s.opt(s.conc(s.kw(token.KindDistinct), s.kw(token.KindFrom))),
			s.expression(),
		),
		s.conc(s.expression(), s.opt(s.kw(token.KindNot)), s.kw(token.KindBetween),
			s.expression(), s.kw(token.KindAnd), s.expression(),
		),
		s.conc(s.expression(), s.opt(s.kw(token.KindNot)), s.kw(token.KindIn),
			s.or(
				s.conc(
					s.punct(token.KindLeftParen),
					s.or(s.selectStatement(), s.expressionList()),
					s.punct(token.KindRightParen),
				),
				s.conc(
					s.opt(s.conc(s.schemaName(), s.oper(token.KindDot))),
					s.or(
						s.id(),
						s.conc(s.id(), s.punct(token.KindLeftParen), s.expressionList(), s.punct(token.KindRightParen)),
					),
				),
			),
		),
		s.conc(
			s.opt(s.kw(token.KindNot)), s.opt(s.kw(token.KindExists)),
			s.punct(token.KindLeftParen),
			s.selectStatement(),
			s.punct(token.KindRightParen),
		),
		s.conc(
			s.kw(token.KindCase), s.opt(s.expression()),
			s.plus(s.conc(s.kw(token.KindCase), s.expression(), s.kw(token.KindThen), s.expression())),
			s.opt(s.conc(s.kw(token.KindElse), s.expression())),
			s.kw(token.KindEnd),
		),
		s.conc(
			s.kw(token.KindRaise),
			s.punct(token.KindLeftParen),
			s.or(
				s.kw(token.KindIgnore),
				s.conc(
					s.or(s.kw(token.KindRollback), s.kw(token.KindAbort), s.kw(token.KindFail)),
					s.punct(token.KindComma), s.expression(),
				),
			),
			s.punct(token.KindRightParen),
		),
	)
}

func (s *syntax) expressionList() *syntaxGenerator {
	return s.el
}

func (s *syntax) buildExpressionList() {
	*s.el = *s.conc(
		s.expression(),
		s.star(s.conc(s.punct(token.KindComma), s.expression())),
	)
}

func (s *syntax) functionCall() *syntaxGenerator {
	return s.fnc
}

func (s *syntax) buildFunctionCall() {
	*s.fnc = *s.conc(
		s.id(),
		s.punct(token.KindLeftParen), s.functionArguments(), s.punct(token.KindRightParen),
		s.opt(s.filterClause()),
		s.opt(s.overClause()),
	)
}

func (s *syntax) functionArguments() *syntaxGenerator {
	return s.fa
}

func (s *syntax) buildFunctionArguments() {
	*s.fa = *s.opt(
		s.or(
			s.conc(
				s.opt(s.kw(token.KindDistinct)),
				s.expressionList(),
				s.opt(
					s.conc(s.kw(token.KindOrder), s.kw(token.KindBy), s.orderingTermList()),
				),
			),
			s.oper(token.KindAsterisk),
		),
	)
}

func (s *syntax) orderingTerm() *syntaxGenerator {
	return s.ot
}

func (s *syntax) buildOrderingTerm() {
	*s.ot = *s.conc(
		s.expression(),
		s.opt(s.conc(s.kw(token.KindCollate), s.id())),
		s.opt(s.or(s.kw(token.KindAsc), s.kw(token.KindDesc))),
		s.opt(s.or(
			s.conc(s.kw(token.KindNulls), s.kw(token.KindLast)),
			s.conc(s.kw(token.KindNulls), s.kw(token.KindFirst)),
		)),
	)
}

func (s *syntax) orderingTermList() *syntaxGenerator {
	return s.otl
}

func (s *syntax) buildOrderingTermList() {
	*s.otl = *s.conc(
		s.orderingTerm(),
		s.star(s.conc(s.punct(token.KindComma), s.orderingTerm())),
	)
}

func (s *syntax) filterClause() *syntaxGenerator {
	return s.fc
}

func (s *syntax) buildFilterClause() {
	*s.fc = *s.conc(
		s.kw(token.KindFilter),
		s.punct(token.KindLeftParen),
		s.kw(token.KindWhere),
		s.expression(),
		s.punct(token.KindRightParen),
	)
}

func (s *syntax) overClause() *syntaxGenerator {
	return s.oc
}

func (s *syntax) buildOverClause() {
	*s.oc = *s.conc(
		s.kw(token.KindOver),
		s.or(
			s.id(),
			s.conc(
				s.punct(token.KindLeftParen),
				s.opt(s.id()),
				s.opt(s.conc(s.kw(token.KindPartition), s.kw(token.KindBy), s.expressionList())),
				s.opt(s.conc(s.kw(token.KindOrder), s.kw(token.KindBy), s.orderingTermList())),
				s.opt(s.frameSpec()),
				s.punct(token.KindRightParen),
			),
		),
	)
}

func (s *syntax) frameSpec() *syntaxGenerator {
	return s.fs
}

func (s *syntax) buildFrameSpec() {
	*s.fs = *s.conc(
		s.or(s.kw(token.KindRange), s.kw(token.KindRows), s.kw(token.KindGroups)),
		s.or(
			s.conc(
				s.kw(token.KindBetween),
				s.or(
					s.conc(s.kw(token.KindUnbounded), s.kw(token.KindPreceding)),
					s.conc(s.expression(), s.kw(token.KindPreceding)),
					s.conc(s.kw(token.KindCurrent), s.kw(token.KindRow)),
					s.conc(s.expression(), s.kw(token.KindFollowing)),
				),
				s.kw(token.KindAnd),
				s.or(
					s.conc(s.expression(), s.kw(token.KindPreceding)),
					s.conc(s.kw(token.KindCurrent), s.kw(token.KindRow)),
					s.conc(s.expression(), s.kw(token.KindFollowing)),
					s.conc(s.kw(token.KindUnbounded), s.kw(token.KindFollowing)),
				),
			),
			s.conc(s.kw(token.KindUnbounded), s.kw(token.KindPreceding)),
			s.conc(s.expression(), s.kw(token.KindPreceding)),
			s.conc(s.kw(token.KindCurrent), s.kw(token.KindRow)),
		),
		s.or(
			s.conc(s.kw(token.KindExclude), s.kw(token.KindNo), s.kw(token.KindOthers)),
			s.conc(s.kw(token.KindExclude), s.kw(token.KindCurrent), s.kw(token.KindRow)),
			s.conc(s.kw(token.KindExclude), s.kw(token.KindGroup)),
			s.conc(s.kw(token.KindExclude), s.kw(token.KindTies)),
			s.eps(),
		),
	)
}

func (s *syntax) typeName() *syntaxGenerator {
	return s.tn
}

func (s *syntax) buildTypeName() {
	*s.tn = *s.conc(
		s.plus(s.id()),
		s.or(
			s.eps(),
			s.conc(s.punct(token.KindLeftParen), s.signedNumber(),
				s.or(s.eps(), s.conc(s.punct(token.KindComma), s.signedNumber())),
				s.punct(token.KindRightParen)),
		))
}

func (s *syntax) signedNumber() *syntaxGenerator {
	return s.sn
}

func (s *syntax) buildSignedNumber() {
	*s.sn = *s.conc(
		s.or(s.eps(), s.oper(token.KindMinus), s.oper(token.KindPlus)),
		s.or(s.int(), s.float()))
}

func (s *syntax) signedLiteral() *syntaxGenerator {
	return s.sl
}

func (s *syntax) buildSignedLiteral() {
	*s.sl = *s.conc(
		s.or(s.eps(), s.oper(token.KindMinus), s.oper(token.KindPlus)),
		s.literal(),
	)
}

func (s *syntax) literal() *syntaxGenerator {
	return s.or(s.string(), s.blob(), s.int(), s.float())
}

func (s *syntax) unaryOper() *syntaxGenerator {
	return s.or(s.oper(token.KindTilde), s.oper(token.KindPlus), s.oper(token.KindMinus))
}

func (s *syntax) binaryOper() *syntaxGenerator {
	return s.or(
		s.oper(token.KindMinus),
		s.oper(token.KindMinusGreaterThan),
		s.oper(token.KindMinusGreaterThanGreaterThan),
		s.oper(token.KindPlus),
		s.oper(token.KindAsterisk),
		s.oper(token.KindSlash),
		s.oper(token.KindPercent),
		s.oper(token.KindEqual),
		s.oper(token.KindEqualEqual),
		s.oper(token.KindLessThanOrEqual),
		s.oper(token.KindLessThanGreaterThan),
		s.oper(token.KindLessThanLessThan),
		s.oper(token.KindLessThan),
		s.oper(token.KindGreaterThanOrEqual),
		s.oper(token.KindGreaterThanGreaterThan),
		s.oper(token.KindGreaterThan),
		s.oper(token.KindExclamationEqual),
		s.oper(token.KindAmpersand),
		s.oper(token.KindPipe),
		s.oper(token.KindPipePipe),
	)
}

func (s *syntax) schemaName() *syntaxGenerator {
	return s.or(s.id(), s.kw(token.KindTemp))
}

func (s *syntax) idList() *syntaxGenerator {
	return s.conc(
		s.id(),
		s.star(s.conc(s.punct(token.KindComma), s.id())),
	)
}

func (s *syntax) id() *syntaxGenerator {
	var g syntaxGenerator = &idSynGen{}
	return &g
}

func (s *syntax) string() *syntaxGenerator {
	var g syntaxGenerator = &stringSynGen{}
	return &g
}

func (s *syntax) blob() *syntaxGenerator {
	var g syntaxGenerator = &blobSynGen{}
	return &g
}

func (s *syntax) int() *syntaxGenerator {
	var g syntaxGenerator = &intSynGen{}
	return &g
}

func (s *syntax) float() *syntaxGenerator {
	var g syntaxGenerator = &floatSynGen{}
	return &g
}

//lint:ignore U1000 this function may be used in the future
func (s *syntax) comment() *syntaxGenerator {
	var g syntaxGenerator = &commentSynGen{}
	return &g
}

func (s *syntax) variable() *syntaxGenerator {
	var g syntaxGenerator = &variableSynGen{}
	return &g
}

func (s *syntax) kw(kind token.Kind) *syntaxGenerator {
	if !kind.IsKeyword() {
		panic(fmt.Errorf("unknown keyword %s", kind))
	}
	var g syntaxGenerator = &keywordSynGen{kind: kind}
	return &g
}

func (s *syntax) oper(kind token.Kind) *syntaxGenerator {
	switch kind {
	case token.KindMinus, token.KindMinusGreaterThan, token.KindMinusGreaterThanGreaterThan,
		token.KindPlus, token.KindAsterisk, token.KindSlash, token.KindPercent, token.KindEqual,
		token.KindEqualEqual, token.KindLessThanOrEqual, token.KindLessThanGreaterThan,
		token.KindLessThanLessThan, token.KindLessThan, token.KindGreaterThanOrEqual,
		token.KindGreaterThanGreaterThan, token.KindGreaterThan, token.KindExclamationEqual,
		token.KindAmpersand, token.KindTilde, token.KindPipe, token.KindPipePipe, token.KindDot:
		var g syntaxGenerator = &operatorSynGen{kind: kind}
		return &g
	}
	panic(fmt.Errorf("unknown operator %s", kind))
}

func (s *syntax) punct(kind token.Kind) *syntaxGenerator {
	switch kind {
	case token.KindLeftParen, token.KindRightParen, token.KindSemicolon, token.KindComma:
		var g syntaxGenerator = &punctuationSynGen{kind: kind}
		return &g
	}
	panic(fmt.Errorf("unknown punctuation %s", kind))
}

func (s *syntax) eps() *syntaxGenerator {
	var g syntaxGenerator = &epsilonSynGen{}
	return &g
}

func (s *syntax) star(g *syntaxGenerator) *syntaxGenerator {
	var gs syntaxGenerator = &starSynGen{g: g}
	return &gs
}

func (s *syntax) plus(g *syntaxGenerator) *syntaxGenerator {
	var gp syntaxGenerator = &plusSynGen{g: g}
	return &gp
}

func (s *syntax) conc(gs ...*syntaxGenerator) *syntaxGenerator {
	var gc syntaxGenerator = &concatSynGen{gs: gs}
	return &gc
}

func (s *syntax) opt(g *syntaxGenerator) *syntaxGenerator {
	return s.or(s.eps(), g)
}

func (s *syntax) or(gs ...*syntaxGenerator) *syntaxGenerator {
	var gor syntaxGenerator = &orSynGen{gs: gs}
	return &gor
}

type idSynGen struct{}

func (ig *idSynGen) gen(stack []*syntaxGenerator, cfg *SyntaxConfig, s *syntax) iter.Seq[string] {
	return func(yield func(string) bool) {
		yield("a")
	}
}

func (ig *idSynGen) firstTokenKind() *token.Kind {
	return &token.KindIdentifier
}

func (ig *idSynGen) lastTokenKind() *token.Kind {
	return &token.KindIdentifier
}

type stringSynGen struct{}

func (sg *stringSynGen) gen(stack []*syntaxGenerator, cfg *SyntaxConfig, s *syntax) iter.Seq[string] {
	return func(yield func(string) bool) {
		yield("'a'")
	}
}

func (sg *stringSynGen) firstTokenKind() *token.Kind {
	return &token.KindString
}

func (sg *stringSynGen) lastTokenKind() *token.Kind {
	return &token.KindString
}

type blobSynGen struct{}

func (bg *blobSynGen) gen(stack []*syntaxGenerator, cfg *SyntaxConfig, s *syntax) iter.Seq[string] {
	return func(yield func(string) bool) {
		yield("X'ab'")
	}
}

func (bg *blobSynGen) firstTokenKind() *token.Kind {
	return &token.KindBlob
}

func (bg *blobSynGen) lastTokenKind() *token.Kind {
	return &token.KindBlob
}

type intSynGen struct{}

func (ig *intSynGen) gen(stack []*syntaxGenerator, cfg *SyntaxConfig, s *syntax) iter.Seq[string] {
	return func(yield func(string) bool) {
		yield("1")
	}
}

func (ig *intSynGen) firstTokenKind() *token.Kind {
	return &token.KindNumeric
}

func (ig *intSynGen) lastTokenKind() *token.Kind {
	return &token.KindNumeric
}

type floatSynGen struct{}

func (fg *floatSynGen) gen(stack []*syntaxGenerator, cfg *SyntaxConfig, s *syntax) iter.Seq[string] {
	return func(yield func(string) bool) {
		yield("1.5")
	}
}

func (fg *floatSynGen) firstTokenKind() *token.Kind {
	return &token.KindNumeric
}

func (fg *floatSynGen) lastTokenKind() *token.Kind {
	return &token.KindNumeric
}

type commentSynGen struct {
	ftk *token.Kind
	ltk *token.Kind
}

// TODO: remove the lint:ignore

//lint:ignore U1000 this function may be used in the future
func (cg *commentSynGen) gen(stack []*syntaxGenerator, cfg *SyntaxConfig, s *syntax) iter.Seq[string] {
	return func(yield func(string) bool) {
		strs := []string{"-- a\n", "/* a */"}
		kinds := []token.Kind{token.KindSQLComment, token.KindCComment}

		cfg.Rand.Shuffle(
			len(strs),
			func(i, j int) {
				strs[i], strs[j] = strs[j], strs[i]
				kinds[i], kinds[j] = kinds[j], kinds[i]
			},
		)
		n := int(uint8(cfg.Rand.Int())%min(cfg.PossibilitiesLimit, uint8(len(strs))) + 1)
		strs = strs[:n]
		kinds = kinds[:n]

		for i, str := range strs {
			cg.ftk = &kinds[i]
			cg.ltk = &kinds[i]
			if !yield(str) {
				return
			}
		}
	}
}

//lint:ignore U1000 this function may be used in the future
func (cg *commentSynGen) firstTokenKind() *token.Kind {
	return cg.ftk
}

//lint:ignore U1000 this function may be used in the future
func (cg *commentSynGen) lastTokenKind() *token.Kind {
	return cg.ltk
}

type variableSynGen struct {
	ftk *token.Kind
	ltk *token.Kind
}

func (vg *variableSynGen) gen(stack []*syntaxGenerator, cfg *SyntaxConfig, s *syntax) iter.Seq[string] {
	return func(yield func(string) bool) {
		strs := []string{"?1", "?", ":a", "@a", "$a", "$a::", "$a::a::(a)", "$a::(a)"}
		kinds := []token.Kind{
			token.KindQuestionVariable, token.KindQuestionVariable, token.KindColonVariable, token.KindAtVariable,
			token.KindDollarVariable, token.KindDollarVariable, token.KindDollarVariable, token.KindDollarVariable,
		}
		cfg.Rand.Shuffle(
			len(strs),
			func(i, j int) {
				strs[i], strs[j] = strs[j], strs[i]
				kinds[i], kinds[j] = kinds[j], kinds[i]
			},
		)
		n := int(uint8(cfg.Rand.Int())%min(cfg.PossibilitiesLimit, uint8(len(strs))) + 1)
		strs = strs[:n]
		kinds = kinds[:n]
		for i, str := range strs {
			vg.ftk = &kinds[i]
			vg.ltk = &kinds[i]
			if !yield(str) {
				return
			}
		}
	}
}

func (vg *variableSynGen) firstTokenKind() *token.Kind {
	return vg.ltk
}

func (vg *variableSynGen) lastTokenKind() *token.Kind {
	return vg.ltk
}

type keywordSynGen struct {
	kind token.Kind
	ftk  *token.Kind
	ltk  *token.Kind
}

func (kg *keywordSynGen) gen(stack []*syntaxGenerator, cfg *SyntaxConfig, s *syntax) iter.Seq[string] {
	return func(yield func(string) bool) {
		kg.ftk = &kg.kind
		kg.ltk = &kg.kind
		switch kg.kind {
		case token.KindAbort:
			yield("abort")
		case token.KindAction:
			yield("action")
		case token.KindAdd:
			yield("add")
		case token.KindAfter:
			yield("after")
		case token.KindAll:
			yield("all")
		case token.KindAlter:
			yield("alter")
		case token.KindAlways:
			yield("always")
		case token.KindAnalyze:
			yield("analyze")
		case token.KindAnd:
			yield("and")
		case token.KindAs:
			yield("as")
		case token.KindAsc:
			yield("asc")
		case token.KindAttach:
			yield("attach")
		case token.KindAutoincrement:
			yield("autoincrement")
		case token.KindBefore:
			yield("before")
		case token.KindBegin:
			yield("begin")
		case token.KindBetween:
			yield("between")
		case token.KindBy:
			yield("by")
		case token.KindCascade:
			yield("cascade")
		case token.KindCase:
			yield("case")
		case token.KindCast:
			yield("cast")
		case token.KindCheck:
			yield("check")
		case token.KindCollate:
			yield("collate")
		case token.KindColumn:
			yield("column")
		case token.KindCommit:
			yield("commit")
		case token.KindConflict:
			yield("conflict")
		case token.KindConstraint:
			yield("constraint")
		case token.KindCreate:
			yield("create")
		case token.KindCross:
			yield("cross")
		case token.KindCurrent:
			yield("current")
		case token.KindCurrentDate:
			yield("currentdate")
		case token.KindCurrentTime:
			yield("currenttime")
		case token.KindCurrentTimestamp:
			yield("currenttimestamp")
		case token.KindDatabase:
			yield("database")
		case token.KindDefault:
			yield("default")
		case token.KindDeferrable:
			yield("deferrable")
		case token.KindDeferred:
			yield("deferred")
		case token.KindDelete:
			yield("delete")
		case token.KindDesc:
			yield("desc")
		case token.KindDetach:
			yield("detach")
		case token.KindDistinct:
			yield("distinct")
		case token.KindDo:
			yield("do")
		case token.KindDrop:
			yield("drop")
		case token.KindEach:
			yield("each")
		case token.KindElse:
			yield("else")
		case token.KindEnd:
			yield("end")
		case token.KindEscape:
			yield("escape")
		case token.KindExcept:
			yield("except")
		case token.KindExclude:
			yield("exclude")
		case token.KindExclusive:
			yield("exclusive")
		case token.KindExists:
			yield("exists")
		case token.KindExplain:
			yield("explain")
		case token.KindFail:
			yield("fail")
		case token.KindFilter:
			yield("filter")
		case token.KindFirst:
			yield("first")
		case token.KindFollowing:
			yield("following")
		case token.KindFor:
			yield("for")
		case token.KindForeign:
			yield("foreign")
		case token.KindFrom:
			yield("from")
		case token.KindFull:
			yield("full")
		case token.KindGenerated:
			yield("generated")
		case token.KindGlob:
			yield("glob")
		case token.KindGroup:
			yield("group")
		case token.KindGroups:
			yield("groups")
		case token.KindHaving:
			yield("having")
		case token.KindIf:
			yield("if")
		case token.KindIgnore:
			yield("ignore")
		case token.KindImmediate:
			yield("immediate")
		case token.KindIn:
			yield("in")
		case token.KindIndex:
			yield("index")
		case token.KindIndexed:
			yield("indexed")
		case token.KindInitially:
			yield("initially")
		case token.KindInner:
			yield("inner")
		case token.KindInsert:
			yield("insert")
		case token.KindInstead:
			yield("instead")
		case token.KindIntersect:
			yield("intersect")
		case token.KindInto:
			yield("into")
		case token.KindIs:
			yield("is")
		case token.KindIsnull:
			yield("isnull")
		case token.KindJoin:
			yield("join")
		case token.KindKey:
			yield("key")
		case token.KindLast:
			yield("last")
		case token.KindLeft:
			yield("left")
		case token.KindLike:
			yield("like")
		case token.KindLimit:
			yield("limit")
		case token.KindMatch:
			yield("match")
		case token.KindMaterialized:
			yield("materialized")
		case token.KindNatural:
			yield("natural")
		case token.KindNo:
			yield("no")
		case token.KindNot:
			yield("not")
		case token.KindNothing:
			yield("nothing")
		case token.KindNotnull:
			yield("notnull")
		case token.KindNull:
			yield("null")
		case token.KindNulls:
			yield("nulls")
		case token.KindOf:
			yield("of")
		case token.KindOffset:
			yield("offset")
		case token.KindOn:
			yield("on")
		case token.KindOr:
			yield("or")
		case token.KindOrder:
			yield("order")
		case token.KindOthers:
			yield("others")
		case token.KindOuter:
			yield("outer")
		case token.KindOver:
			yield("over")
		case token.KindPartition:
			yield("partition")
		case token.KindPlan:
			yield("plan")
		case token.KindPragma:
			yield("pragma")
		case token.KindPreceding:
			yield("preceding")
		case token.KindPrimary:
			yield("primary")
		case token.KindQuery:
			yield("query")
		case token.KindRaise:
			yield("raise")
		case token.KindRange:
			yield("range")
		case token.KindRecursive:
			yield("recursive")
		case token.KindReferences:
			yield("references")
		case token.KindRegexp:
			yield("regexp")
		case token.KindReindex:
			yield("reindex")
		case token.KindRelease:
			yield("release")
		case token.KindRename:
			yield("rename")
		case token.KindReplace:
			yield("replace")
		case token.KindRestrict:
			yield("restrict")
		case token.KindReturning:
			yield("returning")
		case token.KindRight:
			yield("right")
		case token.KindRollback:
			yield("rollback")
		case token.KindRow:
			yield("row")
		case token.KindRowId:
			yield("rowid")
		case token.KindRows:
			yield("rows")
		case token.KindSavepoint:
			yield("savepoint")
		case token.KindSelect:
			yield("select")
		case token.KindSet:
			yield("set")
		case token.KindStrict:
			yield("strict")
		case token.KindTable:
			yield("table")
		case token.KindTemp:
			yield("temp")
		case token.KindTemporary:
			yield("temporary")
		case token.KindThen:
			yield("then")
		case token.KindTies:
			yield("ties")
		case token.KindTo:
			yield("to")
		case token.KindTransaction:
			yield("transaction")
		case token.KindTrigger:
			yield("trigger")
		case token.KindUnbounded:
			yield("unbounded")
		case token.KindUnion:
			yield("union")
		case token.KindUnique:
			yield("unique")
		case token.KindUpdate:
			yield("update")
		case token.KindUsing:
			yield("using")
		case token.KindVacuum:
			yield("vacuum")
		case token.KindValues:
			yield("values")
		case token.KindView:
			yield("view")
		case token.KindVirtual:
			yield("virtual")
		case token.KindWhen:
			yield("when")
		case token.KindWhere:
			yield("where")
		case token.KindWindow:
			yield("window")
		case token.KindWith:
			yield("with")
		case token.KindWithout:
			yield("without")
		}
	}
}

func (kg *keywordSynGen) firstTokenKind() *token.Kind {
	return kg.ftk
}

func (kg *keywordSynGen) lastTokenKind() *token.Kind {
	return kg.ltk
}

type operatorSynGen struct {
	kind token.Kind
	ftk  *token.Kind
	ltk  *token.Kind
}

func (og *operatorSynGen) gen(stack []*syntaxGenerator, cfg *SyntaxConfig, s *syntax) iter.Seq[string] {
	return func(yield func(string) bool) {
		og.ftk = &og.kind
		og.ltk = &og.kind
		switch og.kind {
		case token.KindMinus:
			yield("-")
		case token.KindMinusGreaterThan:
			yield("->")
		case token.KindMinusGreaterThanGreaterThan:
			yield("->>")
		case token.KindPlus:
			yield("+")
		case token.KindAsterisk:
			yield("*")
		case token.KindSlash:
			yield("/")
		case token.KindPercent:
			yield("%")
		case token.KindEqual:
			yield("=")
		case token.KindEqualEqual:
			yield("==")
		case token.KindLessThanOrEqual:
			yield("<=")
		case token.KindLessThanGreaterThan:
			yield("<>")
		case token.KindLessThanLessThan:
			yield("<<")
		case token.KindLessThan:
			yield("<")
		case token.KindGreaterThanOrEqual:
			yield(">=")
		case token.KindGreaterThanGreaterThan:
			yield(">>")
		case token.KindGreaterThan:
			yield(">")
		case token.KindExclamationEqual:
			yield("!=")
		case token.KindAmpersand:
			yield("&")
		case token.KindTilde:
			yield("~")
		case token.KindPipe:
			yield("|")
		case token.KindPipePipe:
			yield("||")
		case token.KindDot:
			yield(".")
		}
	}
}

func (og *operatorSynGen) firstTokenKind() *token.Kind {
	return og.ftk
}

func (og *operatorSynGen) lastTokenKind() *token.Kind {
	return og.ltk
}

type punctuationSynGen struct {
	kind token.Kind
	ftk  *token.Kind
	ltk  *token.Kind
}

func (pg *punctuationSynGen) gen(stack []*syntaxGenerator, cfg *SyntaxConfig, s *syntax) iter.Seq[string] {
	return func(yield func(string) bool) {
		pg.ltk = &pg.kind
		pg.ftk = &pg.kind
		switch pg.kind {
		case token.KindLeftParen:
			yield("(")
		case token.KindRightParen:
			yield(")")
		case token.KindSemicolon:
			yield(";")
		case token.KindComma:
			yield(",")
		}
	}
}

func (pg *punctuationSynGen) firstTokenKind() *token.Kind {
	return pg.ltk
}

func (pg *punctuationSynGen) lastTokenKind() *token.Kind {
	return pg.ltk
}

type epsilonSynGen struct{}

func (e *epsilonSynGen) gen(stack []*syntaxGenerator, cfg *SyntaxConfig, s *syntax) iter.Seq[string] {
	return func(yield func(string) bool) {
		yield("")
	}
}

func (e *epsilonSynGen) firstTokenKind() *token.Kind {
	return nil
}

func (e *epsilonSynGen) lastTokenKind() *token.Kind {
	return nil
}

// TODO: make star and plus less deterministic

type starSynGen struct {
	g   *syntaxGenerator
	ltk *token.Kind
	ftk *token.Kind
}

func (sg *starSynGen) gen(stack []*syntaxGenerator, cfg *SyntaxConfig, s *syntax) iter.Seq[string] {
	return func(yield func(string) bool) {
		var ptr syntaxGenerator = sg
		stack = append(stack, &ptr)
		limit := 1 + uint8(cfg.Rand.Int())%cfg.PossibilitiesLimit
		for i := range limit {
			if i == 0 {
				sg.ftk = nil
				sg.ltk = nil
				if !yield("") {
					return
				}
				continue
			}
			var gs []*syntaxGenerator
			for range i {
				gs = append(gs, sg.g)
			}
			c := s.conc(gs...)
			for str := range (*c).gen(stack, cfg, s) {
				sg.ftk = (*c).firstTokenKind()
				sg.ltk = (*c).lastTokenKind()
				if !yield(str) {
					return
				}
			}
		}
	}
}

func (s *starSynGen) firstTokenKind() *token.Kind {
	return s.ftk
}

func (s *starSynGen) lastTokenKind() *token.Kind {
	return s.ltk
}

type plusSynGen struct {
	g   *syntaxGenerator
	ltk *token.Kind
	ftk *token.Kind
}

func (p *plusSynGen) gen(stack []*syntaxGenerator, cfg *SyntaxConfig, s *syntax) iter.Seq[string] {
	return func(yield func(string) bool) {
		var ptr syntaxGenerator = p
		stack = append(stack, &ptr)
		limit := 1 + uint8(cfg.Rand.Int())%cfg.PossibilitiesLimit
		for i := range limit {
			var gs []*syntaxGenerator
			for range i + 1 {
				gs = append(gs, p.g)
			}
			c := s.conc(gs...)
			for str := range (*c).gen(stack, cfg, s) {
				p.ftk = (*c).firstTokenKind()
				p.ltk = (*c).lastTokenKind()
				if !yield(str) {
					return
				}
			}
		}
	}
}

func (p *plusSynGen) firstTokenKind() *token.Kind {
	return p.ftk
}

func (p *plusSynGen) lastTokenKind() *token.Kind {
	return p.ltk
}

type concatSynGen struct {
	gs  []*syntaxGenerator
	ftk *token.Kind
	ltk *token.Kind
}

func (c *concatSynGen) gen(stack []*syntaxGenerator, cfg *SyntaxConfig, s *syntax) iter.Seq[string] {
	return func(yield func(string) bool) {
		var ptr syntaxGenerator = c
		stack = append(stack, &ptr)
		if count(stack, &ptr) > int(cfg.TurnsInCycleLimit) {
			return
		}
		if len(c.gs) == 1 {
			for str := range (*c.gs[0]).gen(stack, cfg, s) {
				c.ftk = (*c.gs[0]).firstTokenKind()
				c.ltk = (*c.gs[0]).lastTokenKind()
				if !yield(str) {
					break
				}
			}
			return
		}
		first := c.gs[0]
		rem := s.conc(c.gs[1:]...)
		for strFirst := range (*first).gen(stack, cfg, s) {
			firstFtk := (*first).firstTokenKind()
			for strRem := range (*rem).gen(stack, cfg, s) {
				var str string
				needSpace := c.needSpace((*first).lastTokenKind(), (*rem).firstTokenKind())
				if needSpace {
					str = strFirst + " " + strRem
				} else {
					str = strFirst + strRem
				}

				c.ftk = firstFtk
				if c.ftk == nil {
					c.ftk = (*rem).firstTokenKind()
				}
				c.ltk = (*rem).lastTokenKind()
				if c.ltk == nil {
					c.ltk = (*first).lastTokenKind()
				}
				if !yield(str) {
					return
				}
			}
		}
	}
}

func (c *concatSynGen) firstTokenKind() *token.Kind {
	return c.ftk
}

func (c *concatSynGen) lastTokenKind() *token.Kind {
	return c.ltk
}

func (s *concatSynGen) needSpace(k1, k2 *token.Kind) bool {
	if k1 == nil || k2 == nil {
		return false
	}
	if (*k1).IsKeyword() || (*k1) == token.KindIdentifier {
		return (*k2).IsKeyword() || (*k2) == token.KindIdentifier || (*k2) == token.KindNumeric ||
			(*k2) == token.KindString || (*k2) == token.KindBlob || (*k2) == token.KindAtVariable ||
			(*k2) == token.KindColonVariable || (*k2) == token.KindDollarVariable || (*k2) == token.KindQuestionVariable
	}
	if (*k1) == token.KindString && (*k2) == token.KindString {
		return true
	}
	if (*k1) == token.KindNumeric && (*k2) == token.KindNumeric {
		return true
	}
	if *k1 == token.KindAtVariable || *k1 == token.KindColonVariable ||
		*k1 == token.KindDollarVariable || *k1 == token.KindQuestionVariable {
		return true
	}
	if *k1 == token.KindRightParen {
		return *k2 != token.KindRightParen && *k2 != token.KindComma
	}
	if *k1 == token.KindComma {
		return true
	}
	return false
}

type orSynGen struct {
	gs  []*syntaxGenerator
	ftk *token.Kind
	ltk *token.Kind
}

func (o *orSynGen) gen(stack []*syntaxGenerator, cfg *SyntaxConfig, s *syntax) iter.Seq[string] {
	return func(yield func(string) bool) {
		var ptr syntaxGenerator = o
		stack = append(stack, &ptr)
		if count(stack, &ptr) > int(cfg.TurnsInCycleLimit) {
			return
		}

		RHSYieldLimit := cfg.Rand.IntN(int(cfg.PossibilitiesLimit))
		var amountRHSYielded int

		perm := cfg.Rand.Perm(len(o.gs))
		for _, ind := range perm {
			if amountRHSYielded >= RHSYieldLimit {
				break
			}
			var yielded bool
			for str := range (*o.gs[ind]).gen(stack, cfg, s) {
				o.ftk = (*o.gs[ind]).firstTokenKind()
				o.ltk = (*o.gs[ind]).lastTokenKind()
				if !yield(str) {
					return
				}
				yielded = true
			}

			if yielded {
				amountRHSYielded++
			}
		}
	}
}

func (o *orSynGen) firstTokenKind() *token.Kind {
	return o.ftk
}

func (o *orSynGen) lastTokenKind() *token.Kind {
	return o.ltk
}

//lint:ignore U1000 this function is to be used in debugging
func (s *syntax) printStack(w *strings.Builder, stack []*syntaxGenerator) {
	for i := range stack {
		switch *stack[i] {
		case *s.sqlStmt():
			fmt.Fprintln(w, "sqlStmt")
		case *s.analyze():
			fmt.Fprintln(w, "analyze")
		case *s.begin():
			fmt.Fprintln(w, "begin")
		case *s.commit():
			fmt.Fprintln(w, "commit")
		case *s.detach():
			fmt.Fprintln(w, "detach")
		case *s.dropIndex():
			fmt.Fprintln(w, "dropIndex")
		case *s.dropTable():
			fmt.Fprintln(w, "dropTable")
		case *s.dropTrigger():
			fmt.Fprintln(w, "dropTrigger")
		case *s.dropView():
			fmt.Fprintln(w, "dropView")
		case *s.pragma():
			fmt.Fprintln(w, "pragma")
		case *s.pragmaValue():
			fmt.Fprintln(w, "pragmaValue")
		case *s.reindex():
			fmt.Fprintln(w, "reindex")
		case *s.release():
			fmt.Fprintln(w, "release")
		case *s.rollback():
			fmt.Fprintln(w, "rollback")
		case *s.savepoint():
			fmt.Fprintln(w, "savepoint")
		case *s.selectStatement():
			fmt.Fprintln(w, "selectStatement")
		case *s.selectCore():
			fmt.Fprintln(w, "selectCore")
		case *s.commonTableExpression():
			fmt.Fprintln(w, "commonTableExpression")
		case *s.compoundOperator():
			fmt.Fprintln(w, "compoundOperator")
		case *s.resultColumnList():
			fmt.Fprintln(w, "resultColumnList")
		case *s.resultColumn():
			fmt.Fprintln(w, "resultColumn")
		case *s.tableOrSubqueryList():
			fmt.Fprintln(w, "tableOrSubqueryList")
		case *s.tableOrSubquery():
			fmt.Fprintln(w, "tableOrSubquery")
		case *s.joinClause():
			fmt.Fprintln(w, "joinClause")
		case *s.joinOperator():
			fmt.Fprintln(w, "joinOperator")
		case *s.joinConstraint():
			fmt.Fprintln(w, "joinConstraint")
		case *s.windowDeclarationList():
			fmt.Fprintln(w, "windowDeclarationList")
		case *s.windowDeclaration():
			fmt.Fprintln(w, "windowDeclaration")
		case *s.windowDefinition():
			fmt.Fprintln(w, "windowDefinition")
		case *s.vacuum():
			fmt.Fprintln(w, "vacuum")
		case *s.expression():
			fmt.Fprintln(w, "expression")
		case *s.expressionList():
			fmt.Fprintln(w, "expressionList")
		case *s.functionCall():
			fmt.Fprintln(w, "functionCall")
		case *s.functionArguments():
			fmt.Fprintln(w, "functionArguments")
		case *s.orderingTerm():
			fmt.Fprintln(w, "orderingTerm")
		case *s.orderingTermList():
			fmt.Fprintln(w, "orderingTermList")
		case *s.filterClause():
			fmt.Fprintln(w, "filterClause")
		case *s.overClause():
			fmt.Fprintln(w, "overClause")
		case *s.frameSpec():
			fmt.Fprintln(w, "frameSpec")
		case *s.typeName():
			fmt.Fprintln(w, "typeName")
		case *s.signedNumber():
			fmt.Fprintln(w, "signedNumber")
		case *s.signedLiteral():
			fmt.Fprintln(w, "signedLiteral")
		default:
			fmt.Fprintln(w, "...")
		}
	}
}

// count returns the number of times that e is in s.
func count(s []*syntaxGenerator, e *syntaxGenerator) (n int) {
	for i := range s {
		if *s[i] == *e {
			n++
		}
	}

	return n
}
