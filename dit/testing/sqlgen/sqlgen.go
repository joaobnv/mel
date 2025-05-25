// This package generates strings.
package sqlgen

import (
	"errors"
	"fmt"
	"iter"
	"math/rand/v2"

	"github.com/joaobnv/mel/dit/token"
)

type Config struct {
	// RandSource is used to decide the number of turns in each cycle and the number of possibilities
	// generated in each construct that can generate multiple possibilities.
	RandSource         *rand.Rand
	MaxTurnsInCycle    uint8 // the maximum number of turns in cycles. Default 2.
	PossibilitiesLimit uint8 // the maximum number of generated results by constructs that can generate more than one. Default 3.
}

// Syntax generates a sequence of strings that respect the syntax of the SQLite dialect of SQL.
func Syntax() iter.Seq[string] {
	gf := newGenFactory()
	cfg := &Config{
		RandSource:      rand.New(rand.NewPCG(0, 0)),
		MaxTurnsInCycle: 2, PossibilitiesLimit: 3,
	}
	return gf.sqlStmt().gen(nil, cfg)
}

type generator interface {
	gen(stack []generator, cfg *Config) iter.Seq[string]
	// firstTokenKind returns the first token kind of the previous string yielded by gen. Whitespace are ignored. The
	// kind will be nil only if the empty string was yielded.
	firstTokenKind() *token.Kind
	// firstTokenKind returns the last token kind of the previous string yielded by gen. Whitespace are ignored. The
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
		newOr(
			gf.analyze(), gf.begin(), gf.commit(), gf.detach(), gf.dropIndex(), gf.dropTable(),
			gf.dropTrigger(), gf.dropView(), gf.pragma(), gf.reindex(), gf.release(), gf.rollback(),
			gf.savepoint(), gf.vacuum(),
		),
	)
}

type analyze struct {
	generator
}

func (a *analyze) build() {
	a.generator = newConcat(
		newKeywordGen(token.KindAnalyze),
		newOptional(
			newConcat(
				newSchemaName(),
				newOptional(newConcat(newOperatorGen(token.KindDot), newIdGen()))),
		),
	)
}

type begin struct {
	generator
}

func (b *begin) build() {
	b.generator = newConcat(
		newKeywordGen(token.KindBegin),
		newOr(
			newEpsilon(), newKeywordGen(token.KindDeferred),
			newKeywordGen(token.KindImmediate), newKeywordGen(token.KindExclusive),
		),
		newOptional(newKeywordGen(token.KindTransaction)),
	)
}

type commit struct {
	generator
}

func (c *commit) build() {
	c.generator = newConcat(
		newOr(newKeywordGen(token.KindCommit), newKeywordGen(token.KindEnd)),
		newOptional(newKeywordGen(token.KindTransaction)),
	)
}

type detach struct {
	generator
}

func (d *detach) build() {
	d.generator = newConcat(
		newKeywordGen(token.KindDetach),
		newOptional(newKeywordGen(token.KindDatabase)),
		newSchemaName(),
	)
}

type dropIndex struct {
	generator
}

func (di *dropIndex) build() {
	di.generator = newConcat(
		newKeywordGen(token.KindDrop),
		newKeywordGen(token.KindIndex),
		newOptional(newConcat(newKeywordGen(token.KindIf), newKeywordGen(token.KindExists))),
		newOptional(newConcat(newSchemaName(), newOperatorGen(token.KindDot))),
		newIdGen(),
	)
}

type dropTable struct {
	generator
}

func (dt *dropTable) build() {
	dt.generator = newConcat(
		newKeywordGen(token.KindDrop),
		newKeywordGen(token.KindTable),
		newOptional(newConcat(newKeywordGen(token.KindIf), newKeywordGen(token.KindExists))),
		newOptional(newConcat(newSchemaName(), newOperatorGen(token.KindDot))),
		newIdGen(),
	)
}

type dropTrigger struct {
	generator
}

func (dt *dropTrigger) build() {
	dt.generator = newConcat(
		newKeywordGen(token.KindDrop),
		newKeywordGen(token.KindTrigger),
		newOptional(newConcat(newKeywordGen(token.KindIf), newKeywordGen(token.KindExists))),
		newOptional(newConcat(newSchemaName(), newOperatorGen(token.KindDot))),
		newIdGen(),
	)
}

type dropView struct {
	generator
}

func (dv *dropView) build() {
	dv.generator = newConcat(
		newKeywordGen(token.KindDrop),
		newKeywordGen(token.KindView),
		newOptional(newConcat(newKeywordGen(token.KindIf), newKeywordGen(token.KindExists))),
		newOptional(newConcat(newSchemaName(), newOperatorGen(token.KindDot))),
		newIdGen(),
	)
}

type pragma struct {
	generator
}

func (p *pragma) build(gf *genFactory) {
	p.generator = newConcat(
		newKeywordGen(token.KindPragma),
		newOptional(newConcat(newSchemaName(), newOperatorGen(token.KindDot))),
		newIdGen(),
		newOr(
			newEpsilon(),
			newConcat(newOperatorGen(token.KindEqual), gf.pragmaValue()),
			newConcat(newPunctuationGen(token.KindLeftParen), gf.pragmaValue(), newPunctuationGen(token.KindRightParen)),
		),
	)
}

type pragmaValue struct {
	generator
}

func (pv *pragmaValue) build(gf *genFactory) {
	pv.generator = newOr(gf.signedNumber(), newIdGen(), gf.signedLiteral())
}

type reindex struct {
	generator
}

func (r *reindex) build() {
	r.generator = newConcat(
		newKeywordGen(token.KindReindex),
		newOptional(newConcat(
			newOptional(newConcat(newSchemaName(), newOperatorGen(token.KindDot))),
			newIdGen(),
		)),
	)
}

type release struct {
	generator
}

func (r *release) build() {
	r.generator = newConcat(
		newKeywordGen(token.KindRelease),
		newOptional(newKeywordGen(token.KindSavepoint)),
		newIdGen(),
	)
}

type rollback struct {
	generator
}

func (r *rollback) build() {
	r.generator = newConcat(
		newKeywordGen(token.KindRollback),
		newOptional(newKeywordGen(token.KindTransaction)),
		newOptional(newConcat(
			newKeywordGen(token.KindTo),
			newOptional(newKeywordGen(token.KindSavepoint)),
			newIdGen(),
		)),
	)
}

type savepoint struct {
	generator
}

func (sp *savepoint) build() {
	sp.generator = newConcat(
		newKeywordGen(token.KindSavepoint),
		newIdGen(),
	)
}

type vacuum struct {
	generator
}

func (vac *vacuum) build(gf *genFactory) {
	vac.generator = newConcat(
		newKeywordGen(token.KindVacuum),
		newOptional(newSchemaName()),
		newOptional(newConcat(newKeywordGen(token.KindInto), gf.expression())),
	)
}

type expression struct {
	generator
}

func (e *expression) build(gf *genFactory) {
	e.generator = newOr(
		newLiteral(),
		newVariableGen(),
		newConcat(
			newOptional(newConcat(newSchemaName(), newOperatorGen(token.KindDot))),
			newOptional(newConcat(newIdGen(), newOperatorGen(token.KindDot))),
			newIdGen(),
		),
		newConcat(newUnaryOperator(), gf.expression()),
		newConcat(gf.expression(), newBinaryOperator(), gf.expression()),
		gf.functionCall(),
		newConcat(newPunctuationGen(token.KindLeftParen), gf.expressionList(), newPunctuationGen(token.KindRightParen)),
		newConcat(
			newKeywordGen(token.KindCast),
			newPunctuationGen(token.KindLeftParen),
			gf.expression(), newKeywordGen(token.KindAs), gf.typeName(),
			newPunctuationGen(token.KindRightParen),
		),
		newConcat(gf.expression(), newKeywordGen(token.KindCollate), newIdGen()),
		newConcat(
			gf.expression(), newOptional(newKeywordGen(token.KindNot)),
			newOr(
				newConcat(newKeywordGen(token.KindLike), gf.expression(), newOptional(newConcat(newKeywordGen(token.KindEscape), gf.expression()))),
				newConcat(newKeywordGen(token.KindGlob), gf.expression()),
				newConcat(newKeywordGen(token.KindRegexp), gf.expression()),
				newConcat(newKeywordGen(token.KindMatch), gf.expression()),
			),
		),
		newConcat(gf.expression(), newOr(
			newKeywordGen(token.KindIsnull),
			newKeywordGen(token.KindNotnull),
			newConcat(newKeywordGen(token.KindNot), newKeywordGen(token.KindNull)),
		)),
		newConcat(gf.expression(), newKeywordGen(token.KindIs), newOptional(newKeywordGen(token.KindNot)),
			newOptional(newConcat(newKeywordGen(token.KindDistinct), newKeywordGen(token.KindFrom))),
			gf.expression(),
		),
		newConcat(gf.expression(), newOptional(newKeywordGen(token.KindNot)), newKeywordGen(token.KindBetween),
			gf.expression(), newKeywordGen(token.KindAnd), gf.expression(),
		),
		newConcat(gf.expression(), newOptional(newKeywordGen(token.KindNot)), newKeywordGen(token.KindIn),
			newOr(
				newConcat(newPunctuationGen(token.KindLeftParen),
					// TODO: implement the select-stmt
					newOr(gf.expressionList()),
					newPunctuationGen(token.KindRightParen),
				),
				newConcat(
					newOptional(newConcat(newSchemaName(), newOperatorGen(token.KindDot))),
					newOr(
						newIdGen(),
						newConcat(newIdGen(), newPunctuationGen(token.KindLeftParen), gf.expressionList(), newPunctuationGen(token.KindRightParen)),
					),
				),
			),
		),
		newConcat(
			newOptional(newKeywordGen(token.KindNot)), newOptional(newKeywordGen(token.KindExists)),
			newPunctuationGen(token.KindLeftParen),
			// TODO: implement the select stmt
			newPunctuationGen(token.KindRightParen),
		),
		newConcat(
			newKeywordGen(token.KindCase), newOptional(gf.expression()),
			newPlus(newConcat(newKeywordGen(token.KindCase), gf.expression(), newKeywordGen(token.KindThen), gf.expression())),
			newOptional(newConcat(newKeywordGen(token.KindElse), gf.expression())),
			newKeywordGen(token.KindEnd),
		),
		newConcat(
			newKeywordGen(token.KindRaise),
			newPunctuationGen(token.KindLeftParen),
			newOr(
				newKeywordGen(token.KindIgnore),
				newConcat(
					newOr(newKeywordGen(token.KindRollback), newKeywordGen(token.KindAbort), newKeywordGen(token.KindFail)),
					newPunctuationGen(token.KindComma), gf.expression(),
				),
			),
			newPunctuationGen(token.KindRightParen),
		),
	)
}

type unaryOperator struct {
	generator
}

func newUnaryOperator() generator {
	return &unaryOperator{
		generator: newOr(newOperatorGen(token.KindTilde), newOperatorGen(token.KindPlus), newOperatorGen(token.KindMinus)),
	}
}

type binaryOperator struct {
	generator
}

func newBinaryOperator() generator {
	return &binaryOperator{generator: newOr(
		newOperatorGen(token.KindMinus),
		newOperatorGen(token.KindMinusGreaterThan),
		newOperatorGen(token.KindMinusGreaterThanGreaterThan),
		newOperatorGen(token.KindPlus),
		newOperatorGen(token.KindAsterisk),
		newOperatorGen(token.KindSlash),
		newOperatorGen(token.KindPercent),
		newOperatorGen(token.KindEqual),
		newOperatorGen(token.KindEqualEqual),
		newOperatorGen(token.KindLessThanOrEqual),
		newOperatorGen(token.KindLessThanGreaterThan),
		newOperatorGen(token.KindLessThanLessThan),
		newOperatorGen(token.KindLessThan),
		newOperatorGen(token.KindGreaterThanOrEqual),
		newOperatorGen(token.KindGreaterThanGreaterThan),
		newOperatorGen(token.KindGreaterThan),
		newOperatorGen(token.KindExclamationEqual),
		newOperatorGen(token.KindAmpersand),
		newOperatorGen(token.KindPipe),
		newOperatorGen(token.KindPipePipe),
	),
	}
}

type functionCall struct {
	generator
}

func (fc *functionCall) build(gf *genFactory) {
	fc.generator = newConcat(
		newIdGen(),
		newPunctuationGen(token.KindLeftParen), gf.functionArguments(), newPunctuationGen(token.KindRightParen),
		newOptional(gf.filterClause()),
		newOptional(gf.overClause()),
	)
}

type functionArguments struct {
	generator
}

func (fa *functionArguments) build(gf *genFactory) {
	fa.generator = newOptional(
		newOr(
			newConcat(
				newOptional(newKeywordGen(token.KindDistinct)),
				gf.expressionList(),
				newOptional(
					newConcat(newKeywordGen(token.KindOrder), newKeywordGen(token.KindBy), gf.orderingTermList()),
				),
			),
			newOperatorGen(token.KindAsterisk),
		),
	)
}

type expressionList struct {
	generator
}

func (el *expressionList) build(gf *genFactory) {
	el.generator = newConcat(
		gf.expression(),
		newStar(newConcat(newPunctuationGen(token.KindComma), gf.expression())),
	)
}

type orderingTerm struct {
	generator
}

func (ot *orderingTerm) build(gf *genFactory) {
	ot.generator = newConcat(
		gf.expression(),
		newOptional(newConcat(newKeywordGen(token.KindCollate), newIdGen())),
		newOptional(newOr(newKeywordGen(token.KindAsc), newKeywordGen(token.KindDesc))),
		newOptional(newOr(
			newConcat(newKeywordGen(token.KindNulls), newKeywordGen(token.KindLast)),
			newConcat(newKeywordGen(token.KindNulls), newKeywordGen(token.KindFirst)),
		)),
	)
}

type orderingTermList struct {
	generator
}

func (otl *orderingTermList) build(gf *genFactory) {
	otl.generator = newConcat(
		gf.orderingTerm(),
		newStar(newConcat(newPunctuationGen(token.KindComma), gf.orderingTerm())),
	)
}

type filterClause struct {
	generator
}

func (fc *filterClause) build(gf *genFactory) {
	fc.generator = newConcat(
		newKeywordGen(token.KindFilter),
		newPunctuationGen(token.KindLeftParen),
		newKeywordGen(token.KindWhere),
		gf.expression(),
		newPunctuationGen(token.KindRightParen),
	)
}

type overClause struct {
	generator
}

func (oc *overClause) build(gf *genFactory) {
	oc.generator = newConcat(
		newKeywordGen(token.KindOver),
		newOr(
			newIdGen(),
			newConcat(
				newPunctuationGen(token.KindLeftParen),
				newOptional(newIdGen()),
				newOptional(newConcat(newKeywordGen(token.KindPartition), newKeywordGen(token.KindBy), gf.expressionList())),
				newOptional(newConcat(newKeywordGen(token.KindOrder), newKeywordGen(token.KindBy), gf.orderingTermList())),
				newOptional(gf.frameSpec()),
				newPunctuationGen(token.KindRightParen),
			),
		),
	)
}

type frameSpec struct {
	generator
}

func (fs *frameSpec) build(gf *genFactory) {
	fs.generator = newConcat(
		newOr(newKeywordGen(token.KindRange), newKeywordGen(token.KindRows), newKeywordGen(token.KindGroups)),
		newOr(
			newConcat(
				newKeywordGen(token.KindBetween),
				newOr(
					newConcat(newKeywordGen(token.KindUnbounded), newKeywordGen(token.KindPreceding)),
					newConcat(gf.expression(), newKeywordGen(token.KindPreceding)),
					newConcat(newKeywordGen(token.KindCurrent), newKeywordGen(token.KindRow)),
					newConcat(gf.expression(), newKeywordGen(token.KindFollowing)),
				),
				newKeywordGen(token.KindAnd),
				newOr(
					newConcat(gf.expression(), newKeywordGen(token.KindPreceding)),
					newConcat(newKeywordGen(token.KindCurrent), newKeywordGen(token.KindRow)),
					newConcat(gf.expression(), newKeywordGen(token.KindFollowing)),
					newConcat(newKeywordGen(token.KindUnbounded), newKeywordGen(token.KindFollowing)),
				),
			),
			newConcat(newKeywordGen(token.KindUnbounded), newKeywordGen(token.KindPreceding)),
			newConcat(gf.expression(), newKeywordGen(token.KindPreceding)),
			newConcat(newKeywordGen(token.KindCurrent), newKeywordGen(token.KindRow)),
		),
		newOr(
			newConcat(newKeywordGen(token.KindExclude), newKeywordGen(token.KindNo), newKeywordGen(token.KindOthers)),
			newConcat(newKeywordGen(token.KindExclude), newKeywordGen(token.KindCurrent), newKeywordGen(token.KindRow)),
			newConcat(newKeywordGen(token.KindExclude), newKeywordGen(token.KindGroup)),
			newConcat(newKeywordGen(token.KindExclude), newKeywordGen(token.KindTies)),
			newEpsilon(),
		),
	)
}

type typeName struct {
	generator
}

func (tn *typeName) build(gf *genFactory) {
	tn.generator = newConcat(
		newPlus(newIdGen()),
		newOr(
			newEpsilon(),
			newConcat(newPunctuationGen(token.KindLeftParen), gf.signedNumber(),
				newOr(newEpsilon(), newConcat(newPunctuationGen(token.KindComma), gf.signedNumber())),
				newPunctuationGen(token.KindRightParen)),
		))
}

type schemaName struct {
	generator
}

func newSchemaName() generator {
	return &schemaName{newOr(newIdGen(), newKeywordGen(token.KindTemp))}
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

type idGen struct{}

func newIdGen() generator {
	return &idGen{}
}

func (ig *idGen) gen(stack []generator, cfg *Config) iter.Seq[string] {
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

func (sg *stringGen) gen(stack []generator, cfg *Config) iter.Seq[string] {
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

func (bg *blobGen) gen(stack []generator, cfg *Config) iter.Seq[string] {
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

func (ig *intGen) gen(stack []generator, cfg *Config) iter.Seq[string] {
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

func (fg *floatGen) gen(stack []generator, cfg *Config) iter.Seq[string] {
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

// TODO: remove the lint:ignore

//lint:ignore U1000 this function may be used in the future
func newCommentGen() generator {
	return &commentGen{}
}

//lint:ignore U1000 this function may be used in the future
func (cg *commentGen) gen(stack []generator, cfg *Config) iter.Seq[string] {
	return func(yield func(string) bool) {
		strs := []string{"-- a\n", "/* a */"}
		kinds := []token.Kind{token.KindSQLComment, token.KindCComment}

		cfg.RandSource.Shuffle(
			len(strs),
			func(i, j int) {
				strs[i], strs[j] = strs[j], strs[i]
				kinds[i], kinds[j] = kinds[j], kinds[i]
			},
		)
		n := int(uint8(cfg.RandSource.Int())%min(cfg.PossibilitiesLimit, uint8(len(strs))) + 1)
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

func newVariableGen() generator {
	return &variableGen{}
}

func (vg *variableGen) gen(stack []generator, cfg *Config) iter.Seq[string] {
	return func(yield func(string) bool) {
		strs := []string{"?1", "?", ":a", "@a", "$a", "$a::", "$a::a::(a)", "$a::(a)"}
		kinds := []token.Kind{
			token.KindQuestionVariable, token.KindQuestionVariable, token.KindColonVariable, token.KindAtVariable,
			token.KindDollarVariable, token.KindDollarVariable, token.KindDollarVariable, token.KindDollarVariable,
		}
		cfg.RandSource.Shuffle(
			len(strs),
			func(i, j int) {
				strs[i], strs[j] = strs[j], strs[i]
				kinds[i], kinds[j] = kinds[j], kinds[i]
			},
		)
		n := int(uint8(cfg.RandSource.Int())%min(cfg.PossibilitiesLimit, uint8(len(strs))) + 1)
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

func (vg *variableGen) firstTokenKind() *token.Kind {
	return vg.ltk
}

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

func (kg *keywordGen) gen(stack []generator, cfg *Config) iter.Seq[string] {
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

func (og *operatorGen) gen(stack []generator, cfg *Config) iter.Seq[string] {
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

func newPunctuationGen(kind token.Kind) generator {
	switch kind {
	case token.KindLeftParen, token.KindRightParen, token.KindSemicolon, token.KindComma:
		return &punctuationGen{kind: kind}
	}
	panic(fmt.Errorf("unknown punctuation %s", kind))
}

func (pg *punctuationGen) gen(stack []generator, cfg *Config) iter.Seq[string] {
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

type epsilon struct{}

func newEpsilon() generator {
	return &epsilon{}
}

func (e *epsilon) gen(stack []generator, cfg *Config) iter.Seq[string] {
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
	g   generator
	ltk *token.Kind
	ftk *token.Kind
}

func newStar(g generator) generator {
	return &star{g: g}
}

func (s *star) gen(stack []generator, cfg *Config) iter.Seq[string] {
	return func(yield func(string) bool) {
		stack = append(stack, s)
		limit := 1 + uint8(cfg.RandSource.Int())%cfg.PossibilitiesLimit
		for i := range limit {
			if i == 0 {
				s.ftk = nil
				s.ltk = nil
				if !yield("") {
					return
				}
				continue
			}
			var gs []generator
			for range i {
				gs = append(gs, s.g)
			}
			c := newConcat(gs...)
			for str := range c.gen(stack, cfg) {
				s.ftk = c.firstTokenKind()
				s.ltk = c.lastTokenKind()
				if !yield(str) {
					return
				}
			}
		}
	}
}

func (s *star) firstTokenKind() *token.Kind {
	return s.ftk
}

func (s *star) lastTokenKind() *token.Kind {
	return s.ltk
}

type plus struct {
	g   generator
	ltk *token.Kind
	ftk *token.Kind
}

func newPlus(g generator) generator {
	return &plus{g: g}
}

func (p *plus) gen(stack []generator, cfg *Config) iter.Seq[string] {
	return func(yield func(string) bool) {
		stack = append(stack, p)
		limit := 1 + uint8(cfg.RandSource.Int())%cfg.PossibilitiesLimit
		for i := range limit {
			var gs []generator
			for range i + 1 {
				gs = append(gs, p.g)
			}
			c := newConcat(gs...)
			for str := range c.gen(stack, cfg) {
				p.ftk = c.firstTokenKind()
				p.ltk = c.lastTokenKind()
				if !yield(str) {
					return
				}
			}
		}
	}
}

func (p *plus) firstTokenKind() *token.Kind {
	return p.ftk
}

func (p *plus) lastTokenKind() *token.Kind {
	return p.ltk
}

type concat struct {
	gs  []generator
	ftk *token.Kind
	ltk *token.Kind
}

func newConcat(gs ...generator) generator {
	return &concat{gs: gs}
}

func (c *concat) gen(stack []generator, cfg *Config) iter.Seq[string] {
	return func(yield func(string) bool) {
		stack = append(stack, c)
		if len(c.gs) == 0 {
			return
		}
		if len(c.gs) == 1 {
			for str := range c.gs[0].gen(stack, cfg) {
				c.ftk = c.gs[0].firstTokenKind()
				c.ltk = c.gs[0].lastTokenKind()
				if !yield(str) {
					break
				}
			}
			return
		}
		first := c.gs[0]
		rem := newConcat(c.gs[1:]...)
		for strFirst := range first.gen(stack, cfg) {
			firstFtk := first.firstTokenKind()
			for strRem := range rem.gen(stack, cfg) {
				var str string
				needSpace := c.needSpace(first.lastTokenKind(), rem.firstTokenKind())
				if needSpace {
					str = strFirst + " " + strRem
				} else {
					str = strFirst + strRem
				}

				c.ftk = firstFtk
				if c.ftk == nil {
					c.ftk = rem.firstTokenKind()
				}
				c.ltk = rem.lastTokenKind()
				if c.ltk == nil {
					c.ltk = first.lastTokenKind()
				}
				if !yield(str) {
					return
				}
			}
		}
	}
}

func (c *concat) firstTokenKind() *token.Kind {
	return c.ftk
}

func (c *concat) lastTokenKind() *token.Kind {
	return c.ltk
}

func (s *concat) needSpace(k1, k2 *token.Kind) bool {
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

func (o *or) gen(stack []generator, cfg *Config) iter.Seq[string] {
	return func(yield func(string) bool) {
		stack = append(stack, o)
		indices := o.toGen(stack, cfg)
		for _, ind := range indices {
			for str := range o.gs[ind].gen(stack, cfg) {
				o.ftk = o.gs[ind].firstTokenKind()
				o.ltk = o.gs[ind].lastTokenKind()
				if !yield(str) {
					return
				}
			}
		}
	}
}

func (o *or) toGen(stack []generator, cfg *Config) (result []int) {
	for i := range o.gs {
		if count(stack, o.gs[i]) > int(cfg.MaxTurnsInCycle) {
			continue
		}
		result = append(result, i)
	}
	if len(result) == 0 {
		// we do not treat all types of LL grammars
		panic(errors.New(`invalid grammar: or didn't generate anything`))
	}

	cfg.RandSource.Shuffle(len(result), func(i, j int) { result[i], result[j] = result[j], result[i] })
	n := int(uint8(cfg.RandSource.Int())%min(cfg.PossibilitiesLimit, uint8(len(result))) + 1)
	return result[:n]
}

func (o *or) firstTokenKind() *token.Kind {
	return o.ftk
}

func (o *or) lastTokenKind() *token.Kind {
	return o.ltk
}

// count returns the number of times that e is in s.
func count(s []generator, e generator) (n int) {
	for i := range s {
		if s[i] == e {
			n++
		}
	}

	return n
}
