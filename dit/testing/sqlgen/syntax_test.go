package sqlgen

import (
	"strings"
)

//lint:ignore U1000 maybe used in the future
type synGenSimpleString struct {
	str string
}

//lint:ignore U1000 maybe used in the future
func newSynGenSimpleString() *synGenSimpleString {
	return &synGenSimpleString{}
}

func (v *synGenSimpleString) get(g syntaxGenerator) string {
	v.str = ""
	g.accept(v)
	return v.str
}

func (v *synGenSimpleString) visitId(_ *idSynGen) {
	v.str = "id"
}

func (v *synGenSimpleString) visitString(_ *stringSynGen) {
	v.str = "str"
}

func (v *synGenSimpleString) visitBlob(_ *blobSynGen) {
	v.str = "blob"
}

func (v *synGenSimpleString) visitInt(_ *intSynGen) {
	v.str = "int"
}

func (v *synGenSimpleString) visitFloat(_ *floatSynGen) {
	v.str = "float"
}

func (v *synGenSimpleString) visitComment(_ *commentSynGen) {
	v.str = "comm"
}

func (v *synGenSimpleString) visitVariable(_ *variableSynGen) {
	v.str = "var"
}

func (v *synGenSimpleString) visitKeyword(_ *keywordSynGen) {
	v.str = "kw"
}

func (v *synGenSimpleString) visitOperator(_ *operatorSynGen) {
	v.str = "op"
}

func (v *synGenSimpleString) visitPunctuation(_ *punctuationSynGen) {
	v.str = "punct"
}

func (v *synGenSimpleString) visitEpsilon(_ *epsilonSynGen) {
	v.str = "eps"
}

func (v *synGenSimpleString) visitStar(_ *starSynGen) {
	v.str = "star"
}

func (v *synGenSimpleString) visitPlus(_ *plusSynGen) {
	v.str = "plus"
}

func (v *synGenSimpleString) visitConcat(_ *concatSynGen) {
	v.str = "conc"
}

func (v *synGenSimpleString) visitOr(_ *orSynGen) {
	v.str = "or"
}

//lint:ignore U1000 maybe used in the future
type synGenDetailedString struct {
	simpleStrVisitor *synGenSimpleString
	fatherStr        string
	childrenStr      []string
}

//lint:ignore U1000 maybe used in the future
func newSynGenDetailedString() *synGenDetailedString {
	return &synGenDetailedString{simpleStrVisitor: newSynGenSimpleString()}
}

func (v *synGenDetailedString) get(g syntaxGenerator) string {
	v.fatherStr = ""
	v.childrenStr = nil

	g.accept(v)

	str := v.fatherStr
	if len(v.childrenStr) > 0 {
		str += "(" + strings.Join(v.childrenStr, ", ") + ")"
	}
	return str
}

func (v *synGenDetailedString) addchildrenStrings(children ...*syntaxGenerator) {
	for _, child := range children {
		if child == nil { // only if the code has a bug, but we verify because we are in tests
			v.childrenStr = append(v.childrenStr, "<nil>")
		} else {
			v.childrenStr = append(v.childrenStr, v.simpleString(*child))
		}
	}
}

func (v *synGenDetailedString) simpleString(g syntaxGenerator) string {
	return v.simpleStrVisitor.get(g)
}

func (v *synGenDetailedString) visitId(isg *idSynGen) {
	v.fatherStr = v.simpleString(isg)
}

func (v *synGenDetailedString) visitString(ssg *stringSynGen) {
	v.fatherStr = v.simpleString(ssg)
}

func (v *synGenDetailedString) visitBlob(bsg *blobSynGen) {
	v.fatherStr = v.simpleString(bsg)
}

func (v *synGenDetailedString) visitInt(isg *intSynGen) {
	v.fatherStr = v.simpleString(isg)
}

func (v *synGenDetailedString) visitFloat(fsg *floatSynGen) {
	v.fatherStr = v.simpleString(fsg)
}

func (v *synGenDetailedString) visitComment(csg *commentSynGen) {
	v.fatherStr = v.simpleString(csg)
}

func (v *synGenDetailedString) visitVariable(vsg *variableSynGen) {
	v.fatherStr = v.simpleString(vsg)
}

func (v *synGenDetailedString) visitKeyword(ksg *keywordSynGen) {
	v.fatherStr = v.simpleString(ksg)
}

func (v *synGenDetailedString) visitOperator(osg *operatorSynGen) {
	v.fatherStr = v.simpleString(osg)
}

func (v *synGenDetailedString) visitPunctuation(psg *punctuationSynGen) {
	v.fatherStr = v.simpleString(psg)
}

func (v *synGenDetailedString) visitEpsilon(esg *epsilonSynGen) {
	v.fatherStr = v.simpleString(esg)
}

func (v *synGenDetailedString) visitStar(ssg *starSynGen) {
	v.fatherStr = v.simpleString(ssg)
	v.addchildrenStrings(ssg.g)
}

func (v *synGenDetailedString) visitPlus(psg *plusSynGen) {
	v.fatherStr = v.simpleString(psg)
	v.addchildrenStrings(psg.g)
}

func (v *synGenDetailedString) visitConcat(csg *concatSynGen) {
	v.fatherStr = v.simpleString(csg)
	v.addchildrenStrings(csg.gs...)
}

func (v *synGenDetailedString) visitOr(osg *orSynGen) {
	v.fatherStr = v.simpleString(osg)
	v.addchildrenStrings(osg.gs...)
}
