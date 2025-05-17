// This package generates strings.
package sqlgen

import (
	"fmt"
	"iter"

	"github.com/joaobnv/mel/dit/token"
)

// Syntax generates a sequence of strings that respect the syntax of the SQLite dialect of SQL.
func Syntax(maxTurnsInCycle int) iter.Seq[string] {
	gf := newGenFactory()
	return gf.sqlStmt().gen(nil, maxTurnsInCycle)
}

// genFactory creates generators. Note that the generators can create cycles. Then this factory handle this.
type genFactory struct {
	ss *sqlStmt
	pg *pragma
	pv *pragmaValue
	sn *signedNumber
	sl *signedLiteral
}

func newGenFactory() *genFactory {
	gf := new(genFactory)

	gf.sqlStmt().build(gf)
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

type generator interface {
	gen(stack []generator, maxTurnsInCycle int) iter.Seq[string]
	// firstTokenKind returns the first token kind of the previous string yielded by gen. The
	// kind will be nil only if the empty string was yielded.
	firstTokenKind() *token.Kind
	// firstTokenKind returns the last token kind of the previous string yielded by gen. The
	// kind will be nil only if the empty string was yielded.
	lastTokenKind() *token.Kind
}

type sqlStmt struct {
	generator
}

func (ss *sqlStmt) build(gf *genFactory) {
	ss.generator = newConcat(
		newOptional(newKeywordGen(token.KindExplain)),
		newOptional(newConcat(newKeywordGen(token.KindQuery), newKeywordGen(token.KindPlan))),
		newOr(gf.pragma()),
	)
}

type pragma struct {
	generator
}

func (p *pragma) build(gf *genFactory) {
	p.generator = newConcat(
		newKeywordGen(token.KindPragma),
		newOptional(newConcat(newIdGen(), newOperatorGen(token.KindDot))),
		newIdGen(),
		newOr(
			newEpsilon(),
			newConcat(newOperatorGen(token.KindEqual), gf.pragmaValue()),
			newConcat(newPuctuationGen(token.KindLeftParen), gf.pragmaValue(), newPuctuationGen(token.KindRightParen)),
		),
	)
}

type pragmaValue struct {
	generator
}

func (pv *pragmaValue) build(gf *genFactory) {
	pv.generator = newOr(gf.signedNumber(), newIdGen(), gf.signedLiteral())
}

type signedLiteral struct {
	generator
}

func (sl *signedLiteral) build() {
	sl.generator = newConcat(
		newOr(newEpsilon(), newOperatorGen(token.KindMinus), newOperatorGen(token.KindPlus)),
		newLiteral(),
	)
}

type signedNumber struct {
	generator
}

func (sn *signedNumber) build() {
	sn.generator = newConcat(
		newOr(newEpsilon(), newOperatorGen(token.KindMinus), newOperatorGen(token.KindPlus)),
		newOr(newIntGen(), newFloatGen()))
}

type epsilon struct{}

func newEpsilon() generator {
	return &epsilon{}
}

func (e *epsilon) gen(stack []generator, maxTurnsInCycle int) iter.Seq[string] {
	return func(yield func(string) bool) {
		yield("")
	}
}

func (e *epsilon) firstTokenKind() *token.Kind {
	return nil
}

func (e *epsilon) lastTokenKind() *token.Kind {
	return nil
}

type star struct {
	generator
}

//lint:ignore U1000 this function may be used in the future
func newStar(g generator) generator {
	return &star{newOr(newEpsilon(), newConcat(g), newConcat(g, g), newConcat(g, g, g))}
}

type plus struct {
	generator
}

//lint:ignore U1000 this function may be used in the future
func newPlus(g generator) generator {
	return &plus{newConcat(g, newStar(g))}
}

type concat struct {
	gs  []generator
	ftk *token.Kind
	ltk *token.Kind
}

func newConcat(gs ...generator) generator {
	return &concat{gs: gs}
}

func (s *concat) gen(stack []generator, maxTurnsInCycle int) iter.Seq[string] {
	return func(yield func(string) bool) {
		stack = append(stack, s)
		if len(s.gs) == 0 {
			return
		}
		if len(s.gs) == 1 {
			for str := range s.gs[0].gen(stack, maxTurnsInCycle) {
				s.ftk = s.gs[0].firstTokenKind()
				s.ltk = s.gs[0].lastTokenKind()
				if !yield(str) {
					break
				}
			}
			return
		}
		first := s.gs[0]
		rem := newConcat(s.gs[1:]...)
		for strFirst := range first.gen(stack, maxTurnsInCycle) {
			for strRem := range rem.gen(stack, maxTurnsInCycle) {
				var str string
				needSpace := s.needSpace(first.lastTokenKind(), rem.firstTokenKind())
				if needSpace {
					str = strFirst + " " + strRem
				} else {
					str = strFirst + strRem
				}

				s.ftk = first.firstTokenKind()
				if s.ftk == nil {
					s.ftk = rem.firstTokenKind()
				}
				s.ltk = rem.lastTokenKind()
				if s.ltk == nil {
					s.ltk = first.lastTokenKind()
				}
				if !yield(str) {
					return
				}
			}
		}
	}
}

func (s *concat) firstTokenKind() *token.Kind {
	return s.ftk
}

func (s *concat) lastTokenKind() *token.Kind {
	return s.ltk
}

func (s *concat) needSpace(k1, k2 *token.Kind) bool {
	if k1 == nil || k2 == nil {
		return false
	}
	if (*k1).IsKeyword() || (*k1) == token.KindIdentifier {
		return (*k2).IsKeyword() || (*k2) == token.KindIdentifier || (*k2) == token.KindNumeric ||
			(*k2) == token.KindString || (*k2) == token.KindBlob
	}
	if (*k1) == token.KindString && (*k2) == token.KindString {
		return true
	}
	if (*k1) == token.KindNumeric && (*k2) == token.KindNumeric {
		return true
	}
	return false
}

type optional struct {
	generator
}

func newOptional(g generator) generator {
	return &optional{newOr(newEpsilon(), g)}
}

type or struct {
	gs  []generator
	ftk *token.Kind
	ltk *token.Kind
}

func newOr(gs ...generator) generator {
	return &or{gs: gs}
}

func (o *or) gen(stack []generator, maxTurnsInCycle int) iter.Seq[string] {
	return func(yield func(string) bool) {
		stack = append(stack, o)
		for i := range o.gs {
			for str := range o.gs[i].gen(stack, maxTurnsInCycle) {
				o.ftk = o.gs[i].firstTokenKind()
				o.ltk = o.gs[i].lastTokenKind()
				if !yield(str) {
					return
				}
			}
		}
	}
}

func (o *or) firstTokenKind() *token.Kind {
	return o.ftk
}

func (o *or) lastTokenKind() *token.Kind {
	return o.ltk
}

type idGen struct{}

func newIdGen() generator {
	return &idGen{}
}

func (ig *idGen) gen(stack []generator, maxTurnsInCycle int) iter.Seq[string] {
	return func(yield func(string) bool) {
		yield("a")
	}
}

func (ig *idGen) firstTokenKind() *token.Kind {
	return &token.KindIdentifier
}

func (ig *idGen) lastTokenKind() *token.Kind {
	return &token.KindIdentifier
}

type literal struct {
	generator
}

func newLiteral() generator {
	return &literal{
		newOr(
			newStringGen(), newBlobGen(), newIntGen(), newFloatGen(),
		),
	}
}

type stringGen struct{}

func newStringGen() generator {
	return &stringGen{}
}

func (sg *stringGen) gen(stack []generator, maxTurnsInCycle int) iter.Seq[string] {
	return func(yield func(string) bool) {
		yield("'a'")
	}
}

func (sg *stringGen) firstTokenKind() *token.Kind {
	return &token.KindString
}

func (sg *stringGen) lastTokenKind() *token.Kind {
	return &token.KindString
}

type blobGen struct{}

func newBlobGen() generator {
	return &blobGen{}
}

func (bg *blobGen) gen(stack []generator, maxTurnsInCycle int) iter.Seq[string] {
	return func(yield func(string) bool) {
		yield("X'ab'")
	}
}

func (bg *blobGen) firstTokenKind() *token.Kind {
	return &token.KindBlob
}

func (bg *blobGen) lastTokenKind() *token.Kind {
	return &token.KindBlob
}

type intGen struct{}

func newIntGen() generator {
	return &intGen{}
}

func (ig *intGen) gen(stack []generator, maxTurnsInCycle int) iter.Seq[string] {
	return func(yield func(string) bool) {
		yield("1")
	}
}

func (ig *intGen) firstTokenKind() *token.Kind {
	return &token.KindNumeric
}

func (ig *intGen) lastTokenKind() *token.Kind {
	return &token.KindNumeric
}

type floatGen struct{}

func newFloatGen() generator {
	return &floatGen{}
}

func (fg *floatGen) gen(stack []generator, maxTurnsInCycle int) iter.Seq[string] {
	return func(yield func(string) bool) {
		yield("1.5")
	}
}

func (fg *floatGen) firstTokenKind() *token.Kind {
	return &token.KindNumeric
}

func (fg *floatGen) lastTokenKind() *token.Kind {
	return &token.KindNumeric
}

type commentGen struct {
	ftk *token.Kind
	ltk *token.Kind
}

//lint:ignore U1000 this function may be used in the future
func newCommentGen() generator {
	return &commentGen{}
}

//lint:ignore U1000 this function may be used in the future
func (cg *commentGen) gen(stack []generator, maxTurnsInCycle int) iter.Seq[string] {
	return func(yield func(string) bool) {
		strs := []string{"-- a\n", "/* a */"}
		kinds := []token.Kind{token.KindSQLComment, token.KindCComment}
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
func (cg *commentGen) firstTokenKind() *token.Kind {
	return cg.ftk
}

//lint:ignore U1000 this function may be used in the future
func (cg *commentGen) lastTokenKind() *token.Kind {
	return cg.ltk
}

type variableGen struct {
	ftk *token.Kind
	ltk *token.Kind
}

//lint:ignore U1000 this function may be used in the future
func newVariableGen() generator {
	return &variableGen{}
}

//lint:ignore U1000 this function may be used in the future
func (vg *variableGen) gen(stack []generator, maxTurnsInCycle int) iter.Seq[string] {
	return func(yield func(string) bool) {
		strs := []string{"?1", "?", ":a", "@a", "$a", "$a::", "$a::a::(a)", "$a::(a)"}
		kinds := []token.Kind{
			token.KindQuestionVariable, token.KindQuestionVariable, token.KindColonVariable, token.KindAtVariable,
			token.KindDollarVariable, token.KindDollarVariable, token.KindDollarVariable, token.KindDollarVariable,
		}
		for i, str := range strs {
			vg.ftk = &kinds[i]
			vg.ltk = &kinds[i]
			if !yield(str) {
				return
			}
		}
	}
}

//lint:ignore U1000 this function may be used in the future
func (vg *variableGen) firstTokenKind() *token.Kind {
	return vg.ltk
}

//lint:ignore U1000 this function may be used in the future
func (vg *variableGen) lastTokenKind() *token.Kind {
	return vg.ltk
}

type keywordGen struct {
	kind token.Kind
	ftk  *token.Kind
	ltk  *token.Kind
}

func newKeywordGen(kind token.Kind) generator {
	if !kind.IsKeyword() {
		panic(fmt.Errorf("unknown keyword %s", kind))
	}
	return &keywordGen{kind: kind}
}

func (kg *keywordGen) gen(stack []generator, maxTurnsInCycle int) iter.Seq[string] {
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

func (kg *keywordGen) firstTokenKind() *token.Kind {
	return kg.ftk
}

func (kg *keywordGen) lastTokenKind() *token.Kind {
	return kg.ltk
}

type operatorGen struct {
	kind token.Kind
	ftk  *token.Kind
	ltk  *token.Kind
}

func newOperatorGen(kind token.Kind) generator {
	switch kind {
	case token.KindMinus, token.KindMinusGreaterThan, token.KindMinusGreaterThanGreaterThan,
		token.KindPlus, token.KindAsterisk, token.KindSlash, token.KindPercent, token.KindEqual,
		token.KindEqualEqual, token.KindLessThanOrEqual, token.KindLessThanGreaterThan,
		token.KindLessThanLessThan, token.KindLessThan, token.KindGreaterThanOrEqual,
		token.KindGreaterThanGreaterThan, token.KindGreaterThan, token.KindExclamationEqual,
		token.KindAmpersand, token.KindTilde, token.KindPipe, token.KindPipePipe, token.KindDot:
		return &operatorGen{kind: kind}
	}
	panic(fmt.Errorf("unknown operator %s", kind))
}

func (og *operatorGen) gen(stack []generator, maxTurnsInCycle int) iter.Seq[string] {
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

func (og *operatorGen) firstTokenKind() *token.Kind {
	return og.ftk
}

func (og *operatorGen) lastTokenKind() *token.Kind {
	return og.ltk
}

type punctuationGen struct {
	kind token.Kind
	ftk  *token.Kind
	ltk  *token.Kind
}

func newPuctuationGen(kind token.Kind) generator {
	switch kind {
	case token.KindLeftParen, token.KindRightParen, token.KindSemicolon, token.KindComma:
		return &punctuationGen{kind: kind}
	}
	panic(fmt.Errorf("unknown punctuation %s", kind))
}

func (pg *punctuationGen) gen(stack []generator, maxTurnsInCycle int) iter.Seq[string] {
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

func (pg *punctuationGen) firstTokenKind() *token.Kind {
	return pg.ltk
}

func (pg *punctuationGen) lastTokenKind() *token.Kind {
	return pg.ltk
}
