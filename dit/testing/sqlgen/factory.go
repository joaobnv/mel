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

func (gf *genFactory) signedNumber() *signedNumber {
	return gf.sn
}

func (gf *genFactory) signedLiteral() *signedLiteral {
	return gf.sl
}
