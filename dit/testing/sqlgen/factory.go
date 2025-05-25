package sqlgen

// genFactory creates generators. Note that the generators can create cycles. Then this factory handle this.
type genFactory struct {
	ss  *sqlStmt
	an  *analyze
	bg  *begin
	com *commit
	de  *detach
	di  *dropIndex
	dt  *dropTable
	dg  *dropTrigger
	dv  *dropView
	pg  *pragma
	pv  *pragmaValue
	ri  *reindex
	rel *release
	rol *rollback
	sp  *savepoint
	vac *vacuum
	exp *expression
	el  *expressionList
	fnc *functionCall
	fa  *functionArguments
	ot  *orderingTerm
	otl *orderingTermList
	fc  *filterClause
	oc  *overClause
	fs  *frameSpec
	tn  *typeName
	sn  *signedNumber
	sl  *signedLiteral
}

func newGenFactory() *genFactory {
	gf := new(genFactory)

	gf.ss = &sqlStmt{}
	gf.an = &analyze{}
	gf.bg = &begin{}
	gf.com = &commit{}
	gf.de = &detach{}
	gf.di = &dropIndex{}
	gf.dt = &dropTable{}
	gf.dg = &dropTrigger{}
	gf.dv = &dropView{}
	gf.pg = &pragma{}
	gf.pv = &pragmaValue{}
	gf.ri = &reindex{}
	gf.rel = &release{}
	gf.rol = &rollback{}
	gf.sp = &savepoint{}
	gf.vac = &vacuum{}
	gf.exp = &expression{}
	gf.el = &expressionList{}
	gf.fnc = &functionCall{}
	gf.fa = &functionArguments{}
	gf.ot = &orderingTerm{}
	gf.otl = &orderingTermList{}
	gf.fc = &filterClause{}
	gf.oc = &overClause{}
	gf.fs = &frameSpec{}
	gf.tn = &typeName{}
	gf.sn = &signedNumber{}
	gf.sl = &signedLiteral{}

	gf.sqlStmt().build(gf)
	gf.analyze().build()
	gf.begin().build()
	gf.commit().build()
	gf.detach().build()
	gf.dropIndex().build()
	gf.dropTable().build()
	gf.dropTrigger().build()
	gf.dropView().build()
	gf.pragma().build(gf)
	gf.pragmaValue().build(gf)
	gf.reindex().build()
	gf.release().build()
	gf.rollback().build()
	gf.savepoint().build()
	gf.vacuum().build(gf)
	gf.expression().build(gf)
	gf.expressionList().build(gf)
	gf.functionCall().build(gf)
	gf.functionArguments().build(gf)
	gf.orderingTerm().build(gf)
	gf.orderingTermList().build(gf)
	gf.filterClause().build(gf)
	gf.overClause().build(gf)
	gf.frameSpec().build(gf)
	gf.typeName().build(gf)
	gf.signedNumber().build()
	gf.signedLiteral().build()

	return gf
}

func (gf *genFactory) sqlStmt() *sqlStmt {
	return gf.ss
}

func (gf *genFactory) analyze() *analyze {
	return gf.an
}

func (gf *genFactory) begin() *begin {
	return gf.bg
}

func (gf *genFactory) commit() *commit {
	return gf.com
}

func (gf *genFactory) detach() *detach {
	return gf.de
}

func (gf *genFactory) dropIndex() *dropIndex {
	return gf.di
}

func (gf *genFactory) dropTable() *dropTable {
	return gf.dt
}

func (gf *genFactory) dropTrigger() *dropTrigger {
	return gf.dg
}

func (gf *genFactory) dropView() *dropView {
	return gf.dv
}

func (gf *genFactory) pragma() *pragma {
	return gf.pg
}

func (gf *genFactory) pragmaValue() *pragmaValue {
	return gf.pv
}

func (gf *genFactory) reindex() *reindex {
	return gf.ri
}

func (gf *genFactory) release() *release {
	return gf.rel
}

func (gf *genFactory) rollback() *rollback {
	return gf.rol
}

func (gf *genFactory) savepoint() *savepoint {
	return gf.sp
}

func (gf *genFactory) vacuum() *vacuum {
	return gf.vac
}

func (gf *genFactory) expression() *expression {
	return gf.exp
}

func (gf *genFactory) expressionList() *expressionList {
	return gf.el
}

func (gf *genFactory) functionCall() *functionCall {
	return gf.fnc
}

func (gf *genFactory) functionArguments() *functionArguments {
	return gf.fa
}

func (gf *genFactory) orderingTerm() *orderingTerm {
	return gf.ot
}

func (gf *genFactory) orderingTermList() *orderingTermList {
	return gf.otl
}

func (gf *genFactory) filterClause() *filterClause {
	return gf.fc
}

func (gf *genFactory) overClause() *overClause {
	return gf.oc
}

func (gf *genFactory) frameSpec() *frameSpec {
	return gf.fs
}

func (gf *genFactory) typeName() *typeName {
	return gf.tn
}

func (gf *genFactory) signedNumber() *signedNumber {
	return gf.sn
}

func (gf *genFactory) signedLiteral() *signedLiteral {
	return gf.sl
}
