package sqlgen

import (
	"iter"
	"slices"
	"strconv"
	"testing"

	"github.com/joaobnv/mel/dit/token"
)

func TestSyntax(t *testing.T) {
	for str := range Syntax(0) {
		want := "analyze"
		if str != want {
			t.Errorf("got %q, want %q", str, want)
		}
		return
	}
}

func TestConcat0(t *testing.T) {
	g := newConcat()
	for str := range g.gen(nil, 0) {
		t.Errorf("concat generated %q", str)
	}
}

type testCase2 struct {
	g      generator
	pos    []int
	strs   []string
	firsts []token.Kind
	lasts  []token.Kind
}

func (tc2 *testCase2) run(t *testing.T) {
	next, stop := iter.Pull(tc2.g.gen(nil, 0))
	defer stop()

	slices.Sort(tc2.pos)
	var posIndex int = -1
	var genIndex int = -1
	for _, want := range tc2.strs {
		posIndex++
		if tc2.pos != nil && posIndex >= len(tc2.pos) {
			return
		}

		var got string
		for {
			var ok bool
			got, ok = next()
			if !ok {
				return
			}
			genIndex++
			if tc2.pos == nil || tc2.pos[posIndex] == genIndex {
				break
			}
		}

		if want != got {
			t.Errorf("pos %d: want %q, got %q", genIndex, want, got)
		}
		if tc2.firsts[posIndex] == nil {
			if tc2.g.firstTokenKind() != nil {
				t.Errorf("pos %d: firstTokenKind = %s, want nil", genIndex, *tc2.g.firstTokenKind())
			}
		} else {
			if *tc2.g.firstTokenKind() != tc2.firsts[posIndex] {
				t.Errorf("pos %d: firstTokenKind = %s, want %s", genIndex, *tc2.g.firstTokenKind(), tc2.firsts[posIndex])
			}
		}
		if tc2.lasts[posIndex] == nil {
			if tc2.g.lastTokenKind() != nil {
				t.Errorf("pos %d: lastTokenKind = %s, want nil", genIndex, *tc2.g.firstTokenKind())
			}
		} else {
			if *tc2.g.lastTokenKind() != tc2.lasts[posIndex] {
				t.Errorf("pos %d: lastTokenKind = %s, want %s", genIndex, *tc2.g.lastTokenKind(), tc2.lasts[posIndex])
			}
		}
	}
}

type testCase struct {
	g     generator
	strs  []string
	kinds []token.Kind
}

func (tc *testCase) run(t *testing.T) {
	next, stop := iter.Pull(tc.g.gen(nil, 0))
	defer stop()

	for i, want := range tc.strs {
		got, _ := next()
		if want != got {
			t.Errorf("want %q, got %q", want, got)
		}
		if tc.kinds[i] == nil {
			if tc.g.firstTokenKind() != nil {
				t.Errorf("firstTokenKind = %s, want nil", *tc.g.firstTokenKind())
			}
			if tc.g.lastTokenKind() != nil {
				t.Errorf("lastTokenKind = %s, want nil", *tc.g.firstTokenKind())
			}
		} else {
			if *tc.g.firstTokenKind() != tc.kinds[i] {
				t.Errorf("firstTokenKind = %s, want %s", *tc.g.firstTokenKind(), tc.kinds[i])
			}
			if *tc.g.lastTokenKind() != tc.kinds[i] {
				t.Errorf("lastTokenKind = %s, want %s", *tc.g.lastTokenKind(), tc.kinds[i])
			}
		}
	}
}

func TestFirstDifferFromLast(t *testing.T) {
	gf := newGenFactory()
	cases := []testCase2{
		{
			g:      gf.sqlStmt(),
			pos:    []int{0, 78, 117, 234, 351},
			strs:   []string{"analyze"},
			firsts: []token.Kind{token.KindAnalyze},
			lasts:  []token.Kind{token.KindAnalyze},
		}, {
			g:      gf.analyze(),
			pos:    []int{0},
			strs:   []string{"analyze", "analyze a", "analyze temp"},
			firsts: []token.Kind{token.KindAnalyze, token.KindAnalyze, token.KindAnalyze},
			lasts:  []token.Kind{token.KindAnalyze, token.KindIdentifier, token.KindTemp},
		}, {
			g:      gf.begin(),
			pos:    []int{0, 1, 2, 3, 4, 7},
			strs:   []string{"begin", "begin transaction", "begin deferred", "begin deferred transaction", "begin immediate", "begin exclusive transaction"},
			firsts: []token.Kind{token.KindBegin, token.KindBegin, token.KindBegin, token.KindBegin, token.KindBegin, token.KindBegin},
			lasts:  []token.Kind{token.KindBegin, token.KindTransaction, token.KindDeferred, token.KindTransaction, token.KindImmediate, token.KindTransaction},
		}, {
			g:    gf.pragma(),
			pos:  []int{0, 1, 20, 39, 46, 63},
			strs: []string{"pragma a", "pragma a=1", "pragma a(1)", "pragma a.a", "pragma a.a=a", "pragma a.a(+1)"},
			firsts: []token.Kind{
				token.KindPragma, token.KindPragma, token.KindPragma, token.KindPragma, token.KindPragma, token.KindPragma,
			},
			lasts: []token.Kind{
				token.KindIdentifier, token.KindNumeric, token.KindRightParen, token.KindIdentifier, token.KindIdentifier, token.KindRightParen,
			},
		}, {
			g:      gf.pragmaValue(),
			pos:    []int{0, 4, 6, 7},
			strs:   []string{"1", "+1", "a", "'a'"},
			firsts: []token.Kind{token.KindNumeric, token.KindPlus, token.KindIdentifier, token.KindString},
			lasts:  []token.Kind{token.KindNumeric, token.KindNumeric, token.KindIdentifier, token.KindString},
		}, {
			g:      gf.signedLiteral(),
			pos:    []int{0, 5, 10},
			strs:   []string{"'a'", "-X'ab'", "+1"},
			firsts: []token.Kind{token.KindString, token.KindMinus, token.KindPlus},
			lasts:  []token.Kind{token.KindString, token.KindBlob, token.KindNumeric},
		}, {
			g:      gf.signedNumber(),
			strs:   []string{"1", "1.5", "-1", "-1.5", "+1", "+1.5"},
			firsts: []token.Kind{token.KindNumeric, token.KindNumeric, token.KindMinus, token.KindMinus, token.KindPlus, token.KindPlus},
			lasts:  []token.Kind{token.KindNumeric, token.KindNumeric, token.KindNumeric, token.KindNumeric, token.KindNumeric, token.KindNumeric},
		}, {
			g:      newConcat(newOr(newIdGen(), newStringGen()), newOr(newIdGen(), newStringGen())),
			strs:   []string{"a a", "a 'a'", "'a'a", "'a' 'a'"},
			firsts: []token.Kind{token.KindIdentifier, token.KindIdentifier, token.KindString, token.KindString},
			lasts:  []token.Kind{token.KindIdentifier, token.KindString, token.KindIdentifier, token.KindString},
		},
	}

	for i, c := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			c.run(t)
		})
	}
}

func TestFirstEqualsLast(t *testing.T) {
	cases := []testCase{
		{g: newLiteral(),
			strs:  []string{"'a'", "X'ab'", "1", "1.5"},
			kinds: []token.Kind{token.KindString, token.KindBlob, token.KindNumeric, token.KindNumeric},
		}, {g: newEpsilon(),
			strs:  []string{""},
			kinds: []token.Kind{nil},
		}, {g: newStar(newIdGen()),
			strs:  []string{},
			kinds: []token.Kind{},
		}, {g: newStar(newIdGen()),
			strs:  []string{"", "a", "a a", "a a a"},
			kinds: []token.Kind{nil, token.KindIdentifier, token.KindIdentifier, token.KindIdentifier},
		}, {g: newPlus(newIdGen()),
			strs:  []string{"a", "a a", "a a a"},
			kinds: []token.Kind{token.KindIdentifier, token.KindIdentifier, token.KindIdentifier},
		}, {g: newConcat(),
			strs:  []string{},
			kinds: []token.Kind{},
		}, {g: newConcat(newIntGen(), newEpsilon()),
			strs:  []string{"1"},
			kinds: []token.Kind{token.KindNumeric},
		}, {g: newConcat(newIntGen(), newOptional(newIntGen()), newOptional(newIntGen())),
			strs:  []string{"1", "1 1", "1 1", "1 1 1"},
			kinds: []token.Kind{token.KindNumeric, token.KindNumeric, token.KindNumeric, token.KindNumeric},
		}, {g: newOr(newIdGen(), newStringGen()),
			strs:  []string{"a", "'a'"},
			kinds: []token.Kind{token.KindIdentifier, token.KindString},
		}, {g: newOptional(newIdGen()),
			strs:  []string{"", "a"},
			kinds: []token.Kind{nil, token.KindIdentifier},
		}, {g: newIdGen(),
			strs:  []string{"a"},
			kinds: []token.Kind{token.KindIdentifier},
		}, {g: newStringGen(),
			strs:  []string{"'a'"},
			kinds: []token.Kind{token.KindString},
		}, {g: newBlobGen(),
			strs:  []string{"X'ab'"},
			kinds: []token.Kind{token.KindBlob},
		}, {g: newIntGen(),
			strs:  []string{"1"},
			kinds: []token.Kind{token.KindNumeric},
		}, {g: newFloatGen(),
			strs:  []string{"1.5"},
			kinds: []token.Kind{token.KindNumeric},
		}, {g: newCommentGen(),
			strs:  []string{"-- a\n", "/* a */"},
			kinds: []token.Kind{token.KindSQLComment, token.KindCComment},
		}, {g: newCommentGen(),
			strs:  []string{"-- a\n"},
			kinds: []token.Kind{token.KindSQLComment},
		}, {g: newVariableGen(),
			strs: []string{"?1", "?", ":a", "@a", "$a", "$a::", "$a::a::(a)", "$a::(a)"},
			kinds: []token.Kind{
				token.KindQuestionVariable, token.KindQuestionVariable, token.KindColonVariable, token.KindAtVariable,
				token.KindDollarVariable, token.KindDollarVariable, token.KindDollarVariable, token.KindDollarVariable,
			},
		}, {g: newVariableGen(),
			strs: []string{"?1", "?", ":a", "@a"},
			kinds: []token.Kind{
				token.KindQuestionVariable, token.KindQuestionVariable, token.KindColonVariable, token.KindAtVariable,
			},
		},
	}

	for i, c := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			c.run(t)
		})
	}
}

func TestKeyword(t *testing.T) {
	cases := []struct {
		kind token.Kind
		want string
	}{
		{kind: token.KindAbort, want: "abort"},
		{kind: token.KindAction, want: "action"},
		{kind: token.KindAdd, want: "add"},
		{kind: token.KindAfter, want: "after"},
		{kind: token.KindAll, want: "all"},
		{kind: token.KindAlter, want: "alter"},
		{kind: token.KindAlways, want: "always"},
		{kind: token.KindAnalyze, want: "analyze"},
		{kind: token.KindAnd, want: "and"},
		{kind: token.KindAs, want: "as"},
		{kind: token.KindAsc, want: "asc"},
		{kind: token.KindAttach, want: "attach"},
		{kind: token.KindAutoincrement, want: "autoincrement"},
		{kind: token.KindBefore, want: "before"},
		{kind: token.KindBegin, want: "begin"},
		{kind: token.KindBetween, want: "between"},
		{kind: token.KindBy, want: "by"},
		{kind: token.KindCascade, want: "cascade"},
		{kind: token.KindCase, want: "case"},
		{kind: token.KindCast, want: "cast"},
		{kind: token.KindCheck, want: "check"},
		{kind: token.KindCollate, want: "collate"},
		{kind: token.KindColumn, want: "column"},
		{kind: token.KindCommit, want: "commit"},
		{kind: token.KindConflict, want: "conflict"},
		{kind: token.KindConstraint, want: "constraint"},
		{kind: token.KindCreate, want: "create"},
		{kind: token.KindCross, want: "cross"},
		{kind: token.KindCurrent, want: "current"},
		{kind: token.KindCurrentDate, want: "currentdate"},
		{kind: token.KindCurrentTime, want: "currenttime"},
		{kind: token.KindCurrentTimestamp, want: "currenttimestamp"},
		{kind: token.KindDatabase, want: "database"},
		{kind: token.KindDefault, want: "default"},
		{kind: token.KindDeferrable, want: "deferrable"},
		{kind: token.KindDeferred, want: "deferred"},
		{kind: token.KindDelete, want: "delete"},
		{kind: token.KindDesc, want: "desc"},
		{kind: token.KindDetach, want: "detach"},
		{kind: token.KindDistinct, want: "distinct"},
		{kind: token.KindDo, want: "do"},
		{kind: token.KindDrop, want: "drop"},
		{kind: token.KindEach, want: "each"},
		{kind: token.KindElse, want: "else"},
		{kind: token.KindEnd, want: "end"},
		{kind: token.KindEscape, want: "escape"},
		{kind: token.KindExcept, want: "except"},
		{kind: token.KindExclude, want: "exclude"},
		{kind: token.KindExclusive, want: "exclusive"},
		{kind: token.KindExists, want: "exists"},
		{kind: token.KindExplain, want: "explain"},
		{kind: token.KindFail, want: "fail"},
		{kind: token.KindFilter, want: "filter"},
		{kind: token.KindFirst, want: "first"},
		{kind: token.KindFollowing, want: "following"},
		{kind: token.KindFor, want: "for"},
		{kind: token.KindForeign, want: "foreign"},
		{kind: token.KindFrom, want: "from"},
		{kind: token.KindFull, want: "full"},
		{kind: token.KindGenerated, want: "generated"},
		{kind: token.KindGlob, want: "glob"},
		{kind: token.KindGroup, want: "group"},
		{kind: token.KindGroups, want: "groups"},
		{kind: token.KindHaving, want: "having"},
		{kind: token.KindIf, want: "if"},
		{kind: token.KindIgnore, want: "ignore"},
		{kind: token.KindImmediate, want: "immediate"},
		{kind: token.KindIn, want: "in"},
		{kind: token.KindIndex, want: "index"},
		{kind: token.KindIndexed, want: "indexed"},
		{kind: token.KindInitially, want: "initially"},
		{kind: token.KindInner, want: "inner"},
		{kind: token.KindInsert, want: "insert"},
		{kind: token.KindInstead, want: "instead"},
		{kind: token.KindIntersect, want: "intersect"},
		{kind: token.KindInto, want: "into"},
		{kind: token.KindIs, want: "is"},
		{kind: token.KindIsnull, want: "isnull"},
		{kind: token.KindJoin, want: "join"},
		{kind: token.KindKey, want: "key"},
		{kind: token.KindLast, want: "last"},
		{kind: token.KindLeft, want: "left"},
		{kind: token.KindLike, want: "like"},
		{kind: token.KindLimit, want: "limit"},
		{kind: token.KindMatch, want: "match"},
		{kind: token.KindMaterialized, want: "materialized"},
		{kind: token.KindNatural, want: "natural"},
		{kind: token.KindNo, want: "no"},
		{kind: token.KindNot, want: "not"},
		{kind: token.KindNothing, want: "nothing"},
		{kind: token.KindNotnull, want: "notnull"},
		{kind: token.KindNull, want: "null"},
		{kind: token.KindNulls, want: "nulls"},
		{kind: token.KindOf, want: "of"},
		{kind: token.KindOffset, want: "offset"},
		{kind: token.KindOn, want: "on"},
		{kind: token.KindOr, want: "or"},
		{kind: token.KindOrder, want: "order"},
		{kind: token.KindOthers, want: "others"},
		{kind: token.KindOuter, want: "outer"},
		{kind: token.KindOver, want: "over"},
		{kind: token.KindPartition, want: "partition"},
		{kind: token.KindPlan, want: "plan"},
		{kind: token.KindPragma, want: "pragma"},
		{kind: token.KindPreceding, want: "preceding"},
		{kind: token.KindPrimary, want: "primary"},
		{kind: token.KindQuery, want: "query"},
		{kind: token.KindRaise, want: "raise"},
		{kind: token.KindRange, want: "range"},
		{kind: token.KindRecursive, want: "recursive"},
		{kind: token.KindReferences, want: "references"},
		{kind: token.KindRegexp, want: "regexp"},
		{kind: token.KindReindex, want: "reindex"},
		{kind: token.KindRelease, want: "release"},
		{kind: token.KindRename, want: "rename"},
		{kind: token.KindReplace, want: "replace"},
		{kind: token.KindRestrict, want: "restrict"},
		{kind: token.KindReturning, want: "returning"},
		{kind: token.KindRight, want: "right"},
		{kind: token.KindRollback, want: "rollback"},
		{kind: token.KindRow, want: "row"},
		{kind: token.KindRowId, want: "rowid"},
		{kind: token.KindRows, want: "rows"},
		{kind: token.KindSavepoint, want: "savepoint"},
		{kind: token.KindSelect, want: "select"},
		{kind: token.KindSet, want: "set"},
		{kind: token.KindStrict, want: "strict"},
		{kind: token.KindTable, want: "table"},
		{kind: token.KindTemp, want: "temp"},
		{kind: token.KindTemporary, want: "temporary"},
		{kind: token.KindThen, want: "then"},
		{kind: token.KindTies, want: "ties"},
		{kind: token.KindTo, want: "to"},
		{kind: token.KindTransaction, want: "transaction"},
		{kind: token.KindTrigger, want: "trigger"},
		{kind: token.KindUnbounded, want: "unbounded"},
		{kind: token.KindUnion, want: "union"},
		{kind: token.KindUnique, want: "unique"},
		{kind: token.KindUpdate, want: "update"},
		{kind: token.KindUsing, want: "using"},
		{kind: token.KindVacuum, want: "vacuum"},
		{kind: token.KindValues, want: "values"},
		{kind: token.KindView, want: "view"},
		{kind: token.KindVirtual, want: "virtual"},
		{kind: token.KindWhen, want: "when"},
		{kind: token.KindWhere, want: "where"},
		{kind: token.KindWindow, want: "window"},
		{kind: token.KindWith, want: "with"},
		{kind: token.KindWithout, want: "without"},
	}

	for i, c := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			g := newKeywordGen(c.kind)
			strs := slices.Collect(g.gen(nil, 0))
			if len(strs) != 1 {
				t.Errorf("want %d strings, got %d", 1, len(strs))
				return
			}
			if c.want != strs[0] {
				t.Errorf("want %q, got %q", c.want, strs[0])
			}
			if *g.firstTokenKind() != c.kind {
				t.Errorf("firstTokenKind = %s, want %s", *g.firstTokenKind(), c.kind)
			}
			if *g.lastTokenKind() != c.kind {
				t.Errorf("lastTokenKind = %s, want %s", *g.lastTokenKind(), c.kind)
			}
		})
	}
}

func TestInvalidKeyword(t *testing.T) {
	defer func() {
		value := recover()
		if value == nil {
			t.Fatal("not panicked")
		}
		err, ok := value.(error)
		if !ok {
			t.Fatalf("panic value of type %T, want type error", value)
		}
		wantMsg := "unknown keyword Dot"
		if err.Error() != wantMsg {
			t.Fatalf("got %q, want %q", err.Error(), wantMsg)
		}
	}()

	newKeywordGen(token.KindDot)
}

func TestOperator(t *testing.T) {
	cases := []struct {
		kind token.Kind
		want string
	}{
		{kind: token.KindMinus, want: "-"},
		{kind: token.KindMinusGreaterThan, want: "->"},
		{kind: token.KindMinusGreaterThanGreaterThan, want: "->>"},
		{kind: token.KindPlus, want: "+"},
		{kind: token.KindAsterisk, want: "*"},
		{kind: token.KindSlash, want: "/"},
		{kind: token.KindPercent, want: "%"},
		{kind: token.KindEqual, want: "="},
		{kind: token.KindEqualEqual, want: "=="},
		{kind: token.KindLessThanOrEqual, want: "<="},
		{kind: token.KindLessThanGreaterThan, want: "<>"},
		{kind: token.KindLessThanLessThan, want: "<<"},
		{kind: token.KindLessThan, want: "<"},
		{kind: token.KindGreaterThanOrEqual, want: ">="},
		{kind: token.KindGreaterThanGreaterThan, want: ">>"},
		{kind: token.KindGreaterThan, want: ">"},
		{kind: token.KindExclamationEqual, want: "!="},
		{kind: token.KindAmpersand, want: "&"},
		{kind: token.KindTilde, want: "~"},
		{kind: token.KindPipe, want: "|"},
		{kind: token.KindPipePipe, want: "||"},
		{kind: token.KindDot, want: "."},
	}

	for i, c := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			g := newOperatorGen(c.kind)
			strs := slices.Collect(g.gen(nil, 0))
			if len(strs) != 1 {
				t.Errorf("want %d strings, got %d", 1, len(strs))
				return
			}
			if c.want != strs[0] {
				t.Errorf("want %q, got %q", c.want, strs[0])
			}
			if *g.firstTokenKind() != c.kind {
				t.Errorf("firstTokenKind = %s, want %s", *g.firstTokenKind(), c.kind)
			}
			if *g.lastTokenKind() != c.kind {
				t.Errorf("lastTokenKind = %s, want %s", *g.lastTokenKind(), c.kind)
			}
		})
	}
}

func TestInvalidOperator(t *testing.T) {
	defer func() {
		value := recover()
		if value == nil {
			t.Fatal("not panicked")
		}
		err, ok := value.(error)
		if !ok {
			t.Fatalf("panic value of type %T, want type error", value)
		}
		wantMsg := "unknown operator Select"
		if err.Error() != wantMsg {
			t.Fatalf("got %q, want %q", err.Error(), wantMsg)
		}
	}()

	newOperatorGen(token.KindSelect)
}

func TestPunctuation(t *testing.T) {
	cases := []struct {
		kind token.Kind
		want string
	}{
		{kind: token.KindLeftParen, want: "("},
		{kind: token.KindRightParen, want: ")"},
		{kind: token.KindSemicolon, want: ";"},
		{kind: token.KindComma, want: ","},
	}

	for i, c := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			g := newPuctuationGen(c.kind)
			strs := slices.Collect(g.gen(nil, 0))
			if len(strs) != 1 {
				t.Errorf("want %d strings, got %d", 1, len(strs))
				return
			}
			if c.want != strs[0] {
				t.Errorf("want %q, got %q", c.want, strs[0])
			}
			if *g.firstTokenKind() != c.kind {
				t.Errorf("firstTokenKind = %s, want %s", *g.firstTokenKind(), c.kind)
			}
			if *g.lastTokenKind() != c.kind {
				t.Errorf("lastTokenKind = %s, want %s", *g.lastTokenKind(), c.kind)
			}
		})
	}
}

func TestInvalidPunctuation(t *testing.T) {
	defer func() {
		value := recover()
		if value == nil {
			t.Fatal("not panicked")
		}
		err, ok := value.(error)
		if !ok {
			t.Fatalf("panic value of type %T, want type error", value)
		}
		wantMsg := "unknown punctuation Select"
		if err.Error() != wantMsg {
			t.Fatalf("got %q, want %q", err.Error(), wantMsg)
		}
	}()

	newPuctuationGen(token.KindSelect)
}
