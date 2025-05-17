package sqlgen

// genFactory creates generators. Note that the generators can create cycles. Then this factory handle this.
type genFactory struct {
	ss *sqlStmt
	an *analyze
	pg *pragma
	pv *pragmaValue
	sn *signedNumber
	sl *signedLiteral
}

func newGenFactory() *genFactory {
	gf := new(genFactory)

	gf.sqlStmt().build(gf)
	gf.analyze().build(gf)
	gf.pragma().build(gf)
	gf.pragmaValue().build(gf)
	gf.signedNumber().build()
	gf.signedLiteral().build()

	return gf
}

func (gf *genFactory) sqlStmt() *sqlStmt {
	if gf.ss == nil {
		gf.ss = &sqlStmt{}
	}
	return gf.ss
}

func (gf *genFactory) analyze() *analyze {
	if gf.an == nil {
		gf.an = &analyze{}
	}
	return gf.an
}

func (gf *genFactory) pragma() *pragma {
	if gf.pg == nil {
		gf.pg = &pragma{}
	}
	return gf.pg
}

func (gf *genFactory) pragmaValue() *pragmaValue {
	if gf.pv == nil {
		gf.pv = &pragmaValue{}
	}
	return gf.pv
}

func (gf *genFactory) signedNumber() *signedNumber {
	if gf.sn == nil {
		gf.sn = &signedNumber{}
	}
	return gf.sn
}

func (gf *genFactory) signedLiteral() *signedLiteral {
	if gf.sl == nil {
		gf.sl = &signedLiteral{}
	}
	return gf.sl
}
