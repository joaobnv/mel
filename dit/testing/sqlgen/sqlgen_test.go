package sqlgen

import (
	"iter"
	"math/rand/v2"
	"slices"
	"strconv"
	"testing"

	"github.com/joaobnv/mel/dit/token"
)

func TestSyntax(t *testing.T) {
	for str := range Syntax() {
		want := "explain end"
		if str != want {
			t.Errorf("got %q, want %q", str, want)
		}
		return
	}
}

func TestConcat0(t *testing.T) {
	g := newConcat()
	cfg := &Config{
		RandSource:         rand.New(rand.NewPCG(1, 1)),
		MaxTurnsInCycle:    0,
		PossibilitiesLimit: 2,
	}
	for str := range g.gen(nil, cfg) {
		t.Errorf("concat generated %q", str)
	}
}

type testCase2 struct {
	cfg    *Config
	g      generator
	pos    []int
	strs   []string
	firsts []token.Kind
	lasts  []token.Kind
}

func (tc2 *testCase2) run(t *testing.T) {
	next, stop := iter.Pull(tc2.g.gen(nil, tc2.cfg))
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
			if tc2.g.firstTokenKind() == nil {
				t.Errorf("pos %d: firstTokenKind = nil, want %s", genIndex, tc2.firsts[posIndex])
			} else if *tc2.g.firstTokenKind() != tc2.firsts[posIndex] {
				t.Errorf("pos %d: firstTokenKind = %s, want %s", genIndex, *tc2.g.firstTokenKind(), tc2.firsts[posIndex])
			}
		}
		if tc2.lasts[posIndex] == nil {
			if tc2.g.lastTokenKind() != nil {
				t.Errorf("pos %d: lastTokenKind = %s, want nil", genIndex, *tc2.g.firstTokenKind())
			}
		} else {
			if tc2.g.lastTokenKind() == nil {
				t.Errorf("pos %d: lastTokenKind = nil, want %s", genIndex, tc2.lasts[posIndex])
			} else if *tc2.g.lastTokenKind() != tc2.lasts[posIndex] {
				t.Errorf("pos %d: lastTokenKind = %s, want %s", genIndex, *tc2.g.lastTokenKind(), tc2.lasts[posIndex])
			}
		}
	}
}

type testCase struct {
	cfg   *Config
	g     generator
	strs  []string
	kinds []token.Kind
}

func (tc *testCase) run(t *testing.T) {
	next, stop := iter.Pull(tc.g.gen(nil, tc.cfg))
	defer stop()

	for i, want := range tc.strs {
		got, ok := next()
		if !ok {
			break
		}
		if want != got {
			t.Errorf("pos %d: want %q, got %q", i, want, got)
		}
		if tc.kinds[i] == nil {
			if tc.g.firstTokenKind() != nil {
				t.Errorf("pos %d: firstTokenKind = %s, want nil", i, *tc.g.firstTokenKind())
			}
			if tc.g.lastTokenKind() != nil {
				t.Errorf("pos %d: lastTokenKind = %s, want nil", i, *tc.g.lastTokenKind())
			}
		} else {
			if *tc.g.firstTokenKind() != tc.kinds[i] {
				t.Errorf("pos %d: firstTokenKind = %s, want %s", i, *tc.g.firstTokenKind(), tc.kinds[i])
			}
			if *tc.g.lastTokenKind() != tc.kinds[i] {
				t.Errorf("pos %d: lastTokenKind = %s, want %s", i, *tc.g.lastTokenKind(), tc.kinds[i])
			}
		}
	}
}

func TestSqlStmt(t *testing.T) {
	gf := newGenFactory()
	cfg := &Config{
		RandSource:         rand.New(rand.NewPCG(0, 0)),
		MaxTurnsInCycle:    0,
		PossibilitiesLimit: 3,
	}
	cases := []testCase2{
		{
			cfg:    cfg,
			g:      gf.sqlStmt(),
			strs:   []string{"explain end", "explain end transaction", "explain commit"},
			firsts: []token.Kind{token.KindExplain, token.KindExplain, token.KindExplain},
			lasts:  []token.Kind{token.KindEnd, token.KindTransaction, token.KindCommit},
		},
	}

	for i, c := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			c.run(t)
		})
	}
}

func TestAnalyze(t *testing.T) {
	gf := newGenFactory()
	cfg := &Config{
		RandSource:         rand.New(rand.NewPCG(0, 0)),
		MaxTurnsInCycle:    0,
		PossibilitiesLimit: 3,
	}
	cases := []testCase2{
		{
			cfg:    cfg,
			g:      gf.analyze(),
			strs:   []string{"analyze a", "analyze temp", "analyze"},
			firsts: []token.Kind{token.KindAnalyze, token.KindAnalyze, token.KindAnalyze},
			lasts:  []token.Kind{token.KindIdentifier, token.KindTemp, token.KindAnalyze},
		},
	}

	for i, c := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			c.run(t)
		})
	}
}

func TestBegin(t *testing.T) {
	gf := newGenFactory()
	cfg := &Config{
		RandSource:         rand.New(rand.NewPCG(0, 0)),
		MaxTurnsInCycle:    0,
		PossibilitiesLimit: 3,
	}
	cases := []testCase{
		{
			cfg:   cfg,
			g:     gf.begin(),
			strs:  []string{"begin"},
			kinds: []token.Kind{token.KindBegin},
		},
	}

	for i, c := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			c.run(t)
		})
	}
}

func TestCommit(t *testing.T) {
	gf := newGenFactory()
	cfg := &Config{
		RandSource:         rand.New(rand.NewPCG(0, 0)),
		MaxTurnsInCycle:    0,
		PossibilitiesLimit: 3,
	}
	cases := []testCase2{
		{
			cfg:    cfg,
			g:      gf.commit(),
			strs:   []string{"end", "end transaction", "commit"},
			firsts: []token.Kind{token.KindEnd, token.KindEnd, token.KindCommit},
			lasts:  []token.Kind{token.KindEnd, token.KindTransaction, token.KindCommit},
		},
	}

	for i, c := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			c.run(t)
		})
	}
}

func TestDetach(t *testing.T) {
	gf := newGenFactory()
	cfg := &Config{
		RandSource:         rand.New(rand.NewPCG(0, 0)),
		MaxTurnsInCycle:    0,
		PossibilitiesLimit: 3,
	}
	cases := []testCase2{
		{
			cfg:    cfg,
			g:      gf.detach(),
			strs:   []string{"detach database a", "detach database temp", "detach a"},
			firsts: []token.Kind{token.KindDetach, token.KindDetach, token.KindDetach},
			lasts:  []token.Kind{token.KindIdentifier, token.KindTemp, token.KindIdentifier},
		},
	}

	for i, c := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			c.run(t)
		})
	}
}

func TestDropIndex(t *testing.T) {
	gf := newGenFactory()
	cfg := &Config{
		RandSource:         rand.New(rand.NewPCG(0, 0)),
		MaxTurnsInCycle:    0,
		PossibilitiesLimit: 3,
	}
	cases := []testCase2{
		{
			cfg:    cfg,
			g:      gf.dropIndex(),
			strs:   []string{"drop index if exists a", "drop index if exists a.a", "drop index a"},
			firsts: []token.Kind{token.KindDrop, token.KindDrop, token.KindDrop},
			lasts:  []token.Kind{token.KindIdentifier, token.KindIdentifier, token.KindIdentifier},
		},
	}

	for i, c := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			c.run(t)
		})
	}
}

func TestDropTable(t *testing.T) {
	gf := newGenFactory()
	cfg := &Config{
		RandSource:         rand.New(rand.NewPCG(0, 0)),
		MaxTurnsInCycle:    0,
		PossibilitiesLimit: 3,
	}
	cases := []testCase2{
		{
			cfg:    cfg,
			g:      gf.dropTable(),
			strs:   []string{"drop table if exists a", "drop table if exists a.a", "drop table a"},
			firsts: []token.Kind{token.KindDrop, token.KindDrop, token.KindDrop},
			lasts:  []token.Kind{token.KindIdentifier, token.KindIdentifier, token.KindIdentifier},
		},
	}

	for i, c := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			c.run(t)
		})
	}
}

func TestDropTrigger(t *testing.T) {
	gf := newGenFactory()
	cfg := &Config{
		RandSource:         rand.New(rand.NewPCG(0, 0)),
		MaxTurnsInCycle:    0,
		PossibilitiesLimit: 3,
	}
	cases := []testCase2{
		{
			cfg:    cfg,
			g:      gf.dropTrigger(),
			strs:   []string{"drop trigger if exists a", "drop trigger if exists a.a", "drop trigger a"},
			firsts: []token.Kind{token.KindDrop, token.KindDrop, token.KindDrop},
			lasts:  []token.Kind{token.KindIdentifier, token.KindIdentifier, token.KindIdentifier},
		},
	}

	for i, c := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			c.run(t)
		})
	}
}

func TestDropView(t *testing.T) {
	gf := newGenFactory()
	cfg := &Config{
		RandSource:         rand.New(rand.NewPCG(0, 0)),
		MaxTurnsInCycle:    0,
		PossibilitiesLimit: 3,
	}
	cases := []testCase2{
		{
			cfg:    cfg,
			g:      gf.dropView(),
			strs:   []string{"drop view if exists a", "drop view if exists a.a", "drop view a"},
			firsts: []token.Kind{token.KindDrop, token.KindDrop, token.KindDrop},
			lasts:  []token.Kind{token.KindIdentifier, token.KindIdentifier, token.KindIdentifier},
		},
	}

	for i, c := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			c.run(t)
		})
	}
}

func TestPragma(t *testing.T) {
	gf := newGenFactory()
	cfg := &Config{
		RandSource:         rand.New(rand.NewPCG(0, 0)),
		MaxTurnsInCycle:    0,
		PossibilitiesLimit: 3,
	}
	cases := []testCase2{
		{
			cfg:    cfg,
			g:      gf.pragma(),
			strs:   []string{"pragma a.a(+1)", "pragma a.a(+1.5)", "pragma a.a(+1.5)"},
			firsts: []token.Kind{token.KindPragma, token.KindPragma, token.KindPragma},
			lasts:  []token.Kind{token.KindRightParen, token.KindRightParen, token.KindRightParen},
		},
	}

	for i, c := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			c.run(t)
		})
	}
}

func TestPragmaValue(t *testing.T) {
	gf := newGenFactory()
	cfg := &Config{
		RandSource:         rand.New(rand.NewPCG(0, 0)),
		MaxTurnsInCycle:    0,
		PossibilitiesLimit: 3,
	}
	cases := []testCase{
		{
			cfg:   cfg,
			g:     gf.pragmaValue(),
			strs:  []string{"'a'", "1", "1.5"},
			kinds: []token.Kind{token.KindString, token.KindNumeric, token.KindNumeric},
		},
	}

	for i, c := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			c.run(t)
		})
	}
}

func TestReindex(t *testing.T) {
	gf := newGenFactory()
	cfg := &Config{
		RandSource:         rand.New(rand.NewPCG(0, 0)),
		MaxTurnsInCycle:    0,
		PossibilitiesLimit: 3,
	}
	cases := []testCase2{
		{
			cfg:    cfg,
			g:      gf.reindex(),
			strs:   []string{"reindex a", "reindex a.a", "reindex"},
			firsts: []token.Kind{token.KindReindex, token.KindReindex, token.KindReindex},
			lasts:  []token.Kind{token.KindIdentifier, token.KindIdentifier, token.KindReindex},
		},
	}

	for i, c := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			c.run(t)
		})
	}
}

func TestRelease(t *testing.T) {
	gf := newGenFactory()
	cfg := &Config{
		RandSource:         rand.New(rand.NewPCG(0, 0)),
		MaxTurnsInCycle:    0,
		PossibilitiesLimit: 3,
	}
	cases := []testCase2{
		{
			cfg:    cfg,
			g:      gf.release(),
			strs:   []string{"release savepoint a", "release a"},
			firsts: []token.Kind{token.KindRelease, token.KindRelease},
			lasts:  []token.Kind{token.KindIdentifier, token.KindIdentifier},
		},
	}

	for i, c := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			c.run(t)
		})
	}
}

func TestRollback(t *testing.T) {
	gf := newGenFactory()
	cfg := &Config{
		RandSource:         rand.New(rand.NewPCG(0, 0)),
		MaxTurnsInCycle:    0,
		PossibilitiesLimit: 3,
	}
	cases := []testCase2{
		{
			cfg:    cfg,
			g:      gf.rollback(),
			strs:   []string{"rollback transaction", "rollback transaction to a", "rollback"},
			firsts: []token.Kind{token.KindRollback, token.KindRollback, token.KindRollback},
			lasts:  []token.Kind{token.KindTransaction, token.KindIdentifier, token.KindRollback},
		},
	}

	for i, c := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			c.run(t)
		})
	}
}

func TestSavepoint(t *testing.T) {
	gf := newGenFactory()
	cfg := &Config{
		RandSource:         rand.New(rand.NewPCG(0, 0)),
		MaxTurnsInCycle:    0,
		PossibilitiesLimit: 1,
	}
	cases := []testCase2{
		{
			cfg:    cfg,
			g:      gf.savepoint(),
			strs:   []string{"savepoint a"},
			firsts: []token.Kind{token.KindSavepoint},
			lasts:  []token.Kind{token.KindIdentifier},
		},
	}

	for i, c := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			c.run(t)
		})
	}
}

func TestVacuum(t *testing.T) {
	gf := newGenFactory()
	cfg := &Config{
		RandSource:         rand.New(rand.NewPCG(0, 0)),
		MaxTurnsInCycle:    0,
		PossibilitiesLimit: 1,
	}
	cases := []testCase2{
		{
			cfg:    cfg,
			g:      gf.vacuum(),
			strs:   []string{"vacuum a"},
			firsts: []token.Kind{token.KindVacuum},
			lasts:  []token.Kind{token.KindIdentifier},
		},
	}

	for i, c := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			c.run(t)
		})
	}
}

func TestExpression(t *testing.T) {
	gf := newGenFactory()
	cfg := &Config{
		RandSource:         rand.New(rand.NewPCG(0, 0)),
		MaxTurnsInCycle:    0,
		PossibilitiesLimit: 1,
	}
	cases := []testCase2{
		{
			cfg:    cfg,
			g:      gf.expression(),
			strs:   []string{"raise(ignore)"},
			firsts: []token.Kind{token.KindRaise},
			lasts:  []token.Kind{token.KindRightParen},
		},
	}

	for i, c := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			c.run(t)
		})
	}
}

func TestUnOp(t *testing.T) {
	cfg := &Config{
		RandSource:         rand.New(rand.NewPCG(1, 0)),
		MaxTurnsInCycle:    0,
		PossibilitiesLimit: 4,
	}
	cases := []testCase{
		{
			cfg:   cfg,
			g:     newUnaryOperator(),
			strs:  []string{"-", "~", "+"},
			kinds: []token.Kind{token.KindMinus, token.KindTilde, token.KindPlus},
		},
	}

	for i, c := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			c.run(t)
		})
	}
}

func TestBinOp(t *testing.T) {
	cfg := &Config{
		RandSource:         rand.New(rand.NewPCG(1, 0)),
		MaxTurnsInCycle:    0,
		PossibilitiesLimit: 4,
	}
	cases := []testCase{
		{
			cfg:   cfg,
			g:     newBinaryOperator(),
			strs:  []string{">=", ">", "<="},
			kinds: []token.Kind{token.KindGreaterThanOrEqual, token.KindGreaterThan, token.KindLessThanOrEqual},
		},
	}

	for i, c := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			c.run(t)
		})
	}
}

func TestFunctionCall(t *testing.T) {
	gf := newGenFactory()
	cfg := &Config{
		RandSource:         rand.New(rand.NewPCG(1, 0)),
		MaxTurnsInCycle:    0,
		PossibilitiesLimit: 1,
	}
	cases := []testCase2{
		{
			cfg: cfg,
			g:   gf.functionCall(),
			strs: []string{
				"a() filter(where :a not null)",
			},
			firsts: []token.Kind{token.KindIdentifier},
			lasts:  []token.Kind{token.KindRightParen},
		},
	}

	for i, c := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			c.run(t)
		})
	}
}

func TestFunctionArguments(t *testing.T) {
	gf := newGenFactory()
	cfg := &Config{
		RandSource:         rand.New(rand.NewPCG(1, 2)),
		MaxTurnsInCycle:    0,
		PossibilitiesLimit: 1,
	}
	cases := []testCase2{
		{
			cfg: cfg,
			g:   gf.functionArguments(),
			strs: []string{
				"*",
			},
			firsts: []token.Kind{token.KindAsterisk},
			lasts:  []token.Kind{token.KindAsterisk},
		},
	}

	for i, c := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			c.run(t)
		})
	}
}

func TestExpressionList(t *testing.T) {
	gf := newGenFactory()
	cfg := &Config{
		RandSource:         rand.New(rand.NewPCG(0, 0)),
		MaxTurnsInCycle:    0,
		PossibilitiesLimit: 1,
	}
	cases := []testCase2{
		{
			cfg: cfg,
			g:   gf.orderingTerm(),
			strs: []string{
				"raise(ignore)",
			},
			firsts: []token.Kind{token.KindRaise},
			lasts:  []token.Kind{token.KindRightParen},
		},
	}

	for i, c := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			c.run(t)
		})
	}
}

func TestOrderingTerm(t *testing.T) {
	gf := newGenFactory()
	cfg := &Config{
		RandSource:         rand.New(rand.NewPCG(0, 0)),
		MaxTurnsInCycle:    0,
		PossibilitiesLimit: 2,
	}
	cases := []testCase2{
		{
			cfg: cfg,
			g:   gf.orderingTerm(),
			strs: []string{
				"raise(ignore)",
				"raise(ignore) asc",
				"raise(ignore) collate a desc nulls last",
			},
			firsts: []token.Kind{token.KindRaise, token.KindRaise, token.KindRaise},
			lasts:  []token.Kind{token.KindRightParen, token.KindAsc, token.KindLast},
		},
	}

	for i, c := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			c.run(t)
		})
	}
}

func TestOrderingTermList(t *testing.T) {
	gf := newGenFactory()
	cfg := &Config{
		RandSource:         rand.New(rand.NewPCG(0, 0)),
		MaxTurnsInCycle:    0,
		PossibilitiesLimit: 1,
	}
	cases := []testCase2{
		{
			cfg: cfg,
			g:   gf.orderingTermList(),
			strs: []string{
				"raise(ignore)",
				"raise(ignore) desc nulls last, cast(a.a.a as a(+1.5, 1.5)) nulls first",
				"raise(ignore) desc nulls last, cast(a.a.a as a(+1)) collate a",
			},
			firsts: []token.Kind{token.KindRaise, token.KindRaise, token.KindRaise},
			lasts:  []token.Kind{token.KindRightParen, token.KindFirst, token.KindIdentifier},
		},
	}

	for i, c := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			c.run(t)
		})
	}
}

func TestFilterClause(t *testing.T) {
	gf := newGenFactory()
	cfg := &Config{
		RandSource:         rand.New(rand.NewPCG(0, 0)),
		MaxTurnsInCycle:    0,
		PossibilitiesLimit: 3,
	}
	cases := []testCase2{
		{
			cfg: cfg,
			g:   gf.filterClause(),
			strs: []string{
				"filter(where raise(ignore))",
			},
			firsts: []token.Kind{token.KindFilter},
			lasts:  []token.Kind{token.KindRightParen},
		},
	}

	for i, c := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			c.run(t)
		})
	}
}

func TestOverClause(t *testing.T) {
	gf := newGenFactory()
	cfg := &Config{
		RandSource:         rand.New(rand.NewPCG(0, 0)),
		MaxTurnsInCycle:    0,
		PossibilitiesLimit: 2,
	}
	cases := []testCase2{
		{
			cfg: cfg,
			g:   gf.overClause(),
			strs: []string{
				"over()",
				"over(groups current row exclude ties)",
				"over(groups current row exclude current row)",
			},
			firsts: []token.Kind{token.KindOver, token.KindOver, token.KindOver},
			lasts:  []token.Kind{token.KindRightParen, token.KindRightParen, token.KindRightParen},
		},
	}

	for i, c := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			c.run(t)
		})
	}
}

func TestFrameSpec(t *testing.T) {
	gf := newGenFactory()
	cfg := &Config{
		RandSource:         rand.New(rand.NewPCG(0, 0)),
		MaxTurnsInCycle:    0,
		PossibilitiesLimit: 2,
	}
	cases := []testCase2{
		{
			cfg: cfg,
			g:   gf.frameSpec(),
			strs: []string{
				"groups temp.a.a preceding exclude current row",
				"groups temp.a.a preceding exclude ties",
				"groups a.a preceding exclude no others",
			},
			firsts: []token.Kind{token.KindGroups, token.KindGroups, token.KindGroups},
			lasts:  []token.Kind{token.KindRow, token.KindTies, token.KindOthers},
		},
	}

	for i, c := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			c.run(t)
		})
	}
}

func TestTypeName(t *testing.T) {
	gf := newGenFactory()
	cfg := &Config{
		RandSource:         rand.New(rand.NewPCG(1, 1)),
		MaxTurnsInCycle:    0,
		PossibilitiesLimit: 2,
	}
	cases := []testCase2{
		{
			cfg:    cfg,
			g:      gf.typeName(),
			strs:   []string{"a", "a(-1)", "a(-1.5, +1.5)"},
			firsts: []token.Kind{token.KindIdentifier, token.KindIdentifier, token.KindIdentifier},
			lasts:  []token.Kind{token.KindIdentifier, token.KindRightParen, token.KindRightParen},
		},
	}

	for i, c := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			c.run(t)
		})
	}
}

func TestSchemaName(t *testing.T) {
	cfg := &Config{
		RandSource:         rand.New(rand.NewPCG(0, 0)),
		MaxTurnsInCycle:    0,
		PossibilitiesLimit: 3,
	}
	cases := []testCase{
		{
			cfg:   cfg,
			g:     newSchemaName(),
			strs:  []string{"temp", "a"},
			kinds: []token.Kind{token.KindTemp, token.KindIdentifier},
		},
	}

	for i, c := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			c.run(t)
		})
	}
}

func TestSignedLiteral(t *testing.T) {
	gf := newGenFactory()
	cfg := &Config{
		RandSource:         rand.New(rand.NewPCG(0, 0)),
		MaxTurnsInCycle:    0,
		PossibilitiesLimit: 3,
	}
	cases := []testCase2{
		{
			cfg:    cfg,
			g:      gf.signedLiteral(),
			strs:   []string{"+1", "+'a'", "+X'ab'"},
			firsts: []token.Kind{token.KindPlus, token.KindPlus, token.KindPlus},
			lasts:  []token.Kind{token.KindNumeric, token.KindString, token.KindBlob},
		},
	}

	for i, c := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			c.run(t)
		})
	}
}

func TestSignedNumber(t *testing.T) {
	gf := newGenFactory()
	cfg := &Config{
		RandSource:         rand.New(rand.NewPCG(0, 0)),
		MaxTurnsInCycle:    0,
		PossibilitiesLimit: 3,
	}
	cases := []testCase2{
		{
			cfg:    cfg,
			g:      gf.signedNumber(),
			strs:   []string{"+1", "+1.5"},
			firsts: []token.Kind{token.KindPlus, token.KindPlus},
			lasts:  []token.Kind{token.KindNumeric, token.KindNumeric},
		},
	}

	for i, c := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			c.run(t)
		})
	}
}

func TestId(t *testing.T) {
	cfg := &Config{
		RandSource:         rand.New(rand.NewPCG(0, 0)),
		MaxTurnsInCycle:    0,
		PossibilitiesLimit: 4,
	}
	cases := []testCase{
		{
			cfg:   cfg,
			g:     newIdGen(),
			strs:  []string{"a"},
			kinds: []token.Kind{token.KindIdentifier},
		},
	}

	for i, c := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			c.run(t)
		})
	}
}

func TestLiteral(t *testing.T) {
	cfg := &Config{
		RandSource:         rand.New(rand.NewPCG(0, 0)),
		MaxTurnsInCycle:    0,
		PossibilitiesLimit: 4,
	}
	cases := []testCase{
		{
			cfg:   cfg,
			g:     newLiteral(),
			strs:  []string{"'a'", "X'ab'", "1.5", "1"},
			kinds: []token.Kind{token.KindString, token.KindBlob, token.KindNumeric, token.KindNumeric},
		},
	}

	for i, c := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			c.run(t)
		})
	}
}

func TestString(t *testing.T) {
	cfg := &Config{
		RandSource:         rand.New(rand.NewPCG(0, 0)),
		MaxTurnsInCycle:    0,
		PossibilitiesLimit: 4,
	}
	cases := []testCase{
		{
			cfg:   cfg,
			g:     newStringGen(),
			strs:  []string{"'a'"},
			kinds: []token.Kind{token.KindString},
		},
	}

	for i, c := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			c.run(t)
		})
	}
}

func TestBlob(t *testing.T) {
	cfg := &Config{
		RandSource:         rand.New(rand.NewPCG(0, 0)),
		MaxTurnsInCycle:    0,
		PossibilitiesLimit: 4,
	}
	cases := []testCase{
		{
			cfg:   cfg,
			g:     newBlobGen(),
			strs:  []string{"X'ab'"},
			kinds: []token.Kind{token.KindBlob},
		},
	}

	for i, c := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			c.run(t)
		})
	}
}

func TestInt(t *testing.T) {
	cfg := &Config{
		RandSource:         rand.New(rand.NewPCG(0, 0)),
		MaxTurnsInCycle:    0,
		PossibilitiesLimit: 4,
	}
	cases := []testCase{
		{
			cfg:   cfg,
			g:     newIntGen(),
			strs:  []string{"1"},
			kinds: []token.Kind{token.KindNumeric},
		},
	}

	for i, c := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			c.run(t)
		})
	}
}

func TestFloat(t *testing.T) {
	cfg := &Config{
		RandSource:         rand.New(rand.NewPCG(0, 0)),
		MaxTurnsInCycle:    0,
		PossibilitiesLimit: 4,
	}
	cases := []testCase{
		{
			cfg:   cfg,
			g:     newFloatGen(),
			strs:  []string{"1.5"},
			kinds: []token.Kind{token.KindNumeric},
		},
	}

	for i, c := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			c.run(t)
		})
	}
}

func TestComment(t *testing.T) {
	cfg := &Config{
		RandSource:         rand.New(rand.NewPCG(0, 0)),
		MaxTurnsInCycle:    0,
		PossibilitiesLimit: 4,
	}
	cases := []testCase{
		{
			cfg:   cfg,
			g:     newCommentGen(),
			strs:  []string{"/* a */", "-- a\n"},
			kinds: []token.Kind{token.KindCComment, token.KindSQLComment},
		},
	}

	for i, c := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			c.run(t)
		})
	}
}

func TestVariable(t *testing.T) {
	cfg := &Config{
		RandSource:         rand.New(rand.NewPCG(0, 0)),
		MaxTurnsInCycle:    0,
		PossibilitiesLimit: 4,
	}
	cases := []testCase{
		{
			cfg:   cfg,
			g:     newVariableGen(),
			strs:  []string{"$a::a::(a)", "?", "?1"},
			kinds: []token.Kind{token.KindDollarVariable, token.KindQuestionVariable, token.KindQuestionVariable},
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
			cfg := &Config{
				RandSource:         rand.New(rand.NewPCG(1, 1)),
				MaxTurnsInCycle:    0,
				PossibilitiesLimit: 2,
			}
			strs := slices.Collect(g.gen(nil, cfg))
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
			cfg := &Config{
				RandSource:         rand.New(rand.NewPCG(1, 1)),
				MaxTurnsInCycle:    0,
				PossibilitiesLimit: 2,
			}
			strs := slices.Collect(g.gen(nil, cfg))
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
			g := newPunctuationGen(c.kind)
			cfg := &Config{
				RandSource:         rand.New(rand.NewPCG(1, 1)),
				MaxTurnsInCycle:    0,
				PossibilitiesLimit: 2,
			}
			strs := slices.Collect(g.gen(nil, cfg))
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

	newPunctuationGen(token.KindSelect)
}

func TestNeedSpace(t *testing.T) {
	cases := []struct {
		k1, k2 *token.Kind
		want   bool
	}{
		{k1: nil, k2: &token.KindSelect, want: false},
		{k1: &token.KindSelect, k2: &token.KindSelect, want: true},
		{k1: &token.KindString, k2: &token.KindString, want: true},
		{k1: &token.KindNumeric, k2: &token.KindNumeric, want: true},
		{k1: &token.KindAtVariable, k2: &token.KindNumeric, want: true},
		{k1: &token.KindRightParen, k2: &token.KindNumeric, want: true},
		{k1: &token.KindComma, k2: &token.KindNumeric, want: true},
		{k1: &token.KindMinus, k2: &token.KindNumeric, want: false},
	}
	concat := newConcat().(*concat)
	for _, c := range cases {
		if got := concat.needSpace(c.k1, c.k2); c.want != got {
			k1Str := "nil"
			if c.k1 != nil {
				k1Str = (*c.k1).String()
			}
			k2Str := "nil"
			if c.k2 != nil {
				k2Str = (*c.k2).String()
			}
			t.Errorf("needSpace(%s, %s) = %v, want %v", k1Str, k2Str, got, c.want)
		}
	}
}

func TestStar0Break(t *testing.T) {
	cfg := &Config{
		RandSource:         rand.New(rand.NewPCG(0, 0)),
		MaxTurnsInCycle:    2,
		PossibilitiesLimit: 10,
	}
	g := newStar(newIdGen())
	g.gen(nil, cfg)(func(str string) bool {
		if str != "" {
			t.Errorf("want str = %q, got %q", "", str)
		}
		if g.firstTokenKind() != nil {
			t.Errorf("want firstTokenKind = nil, got %s", *g.firstTokenKind())
		}
		if g.lastTokenKind() != nil {
			t.Errorf("want lastTokenKind = nil, got %s", *g.lastTokenKind())
		}
		return false
	})
}

func TestStar(t *testing.T) {
	cfg := &Config{
		RandSource:         rand.New(rand.NewPCG(0, 0)),
		MaxTurnsInCycle:    2,
		PossibilitiesLimit: 10,
	}
	cases := []testCase{
		{
			cfg:   cfg,
			g:     newStar(newIdGen()),
			strs:  []string{"", "a", "a a", "a a a"},
			kinds: []token.Kind{nil, token.KindIdentifier, token.KindIdentifier, token.KindIdentifier},
		},
	}

	for i, c := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			c.run(t)
		})
	}
}

func TestPlus(t *testing.T) {
	cfg := &Config{
		RandSource:         rand.New(rand.NewPCG(0, 0)),
		MaxTurnsInCycle:    2,
		PossibilitiesLimit: 10,
	}
	cases := []testCase{
		{
			cfg:   cfg,
			g:     newPlus(newIdGen()),
			strs:  []string{"a", "a a", "a a a"},
			kinds: []token.Kind{token.KindIdentifier, token.KindIdentifier, token.KindIdentifier},
		},
	}

	for i, c := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			c.run(t)
		})
	}
}

func TestOr(t *testing.T) {
	gf := newGenFactory()

	cfg := &Config{
		RandSource:         rand.New(rand.NewPCG(0, 0)),
		MaxTurnsInCycle:    2,
		PossibilitiesLimit: 10,
	}

	rec := &or{}
	rec.gs = append(rec.gs, newIdGen(), rec)

	cases := []testCase2{
		{
			cfg: cfg,
			g: newOr(
				newIdGen(), newKeywordGen(token.KindSelect),
				newOperatorGen(token.KindEqual), gf.savepoint(),
			),
			strs:   []string{"a", "select", "savepoint a"},
			firsts: []token.Kind{token.KindIdentifier, token.KindSelect, token.KindSavepoint},
			lasts:  []token.Kind{token.KindIdentifier, token.KindSelect, token.KindIdentifier},
		}, {
			cfg:    cfg,
			g:      rec,
			strs:   []string{"a", "a", "a"},
			firsts: []token.Kind{token.KindIdentifier, token.KindIdentifier, token.KindIdentifier},
			lasts:  []token.Kind{token.KindIdentifier, token.KindIdentifier, token.KindIdentifier},
		},
	}

	for i, c := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			c.run(t)
		})
	}
}

func TestOrInvalidGrammar(t *testing.T) {
	defer func() {
		value := recover()
		if value == nil {
			t.Fatal("not panicked")
		}
		err, ok := value.(error)
		if !ok {
			t.Fatalf("panic value of type %T, want type error", value)
		}
		wantMsg := "invalid grammar: or didn't generate anything"
		if err.Error() != wantMsg {
			t.Fatalf("got %q, want %q", err.Error(), wantMsg)
		}
	}()

	g := &or{}
	g.gs = append(g.gs, g)

	cfg := &Config{
		RandSource:         rand.New(rand.NewPCG(0, 0)),
		MaxTurnsInCycle:    2,
		PossibilitiesLimit: 10,
	}

	for range g.gen(nil, cfg) {
	}
}

func TestCount(t *testing.T) {
	gf := newGenFactory()
	cases := []struct {
		s     []generator
		e     generator
		count int
	}{
		{e: gf.expression()},
		{s: []generator{gf.expression()}, e: gf.expression(), count: 1},
		{s: []generator{gf.typeName(), gf.expression()}, e: gf.typeName(), count: 1},
		{s: []generator{gf.expression(), gf.expression()}, e: gf.expression(), count: 2},
	}

	for i, c := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			got := count(c.s, c.e)
			if got != c.count {
				t.Errorf("got %d, want %d", got, c.count)
			}
		})
	}
}
