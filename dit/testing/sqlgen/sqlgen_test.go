package sqlgen

import (
	"fmt"
	"iter"
	"math/rand/v2"
	"slices"
	"strconv"
	"strings"
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
	s := newSyntax()
	g := *s.conc()
	cfg := &SyntaxConfig{
		Rand:               rand.New(rand.NewPCG(1, 1)),
		TurnsInCycleLimit:  0,
		PossibilitiesLimit: 2,
	}
	for str := range g.gen(nil, cfg, s) {
		t.Errorf("concat generated %q", str)
	}
}

type testCase2 struct {
	cfg    *SyntaxConfig
	s      *syntax
	g      syntaxGenerator
	pos    []int
	strs   []string
	firsts []token.Kind
	lasts  []token.Kind
}

func (tc2 *testCase2) run(t *testing.T) {
	next, stop := iter.Pull(tc2.g.gen(nil, tc2.cfg, tc2.s))
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
	cfg   *SyntaxConfig
	s     *syntax
	g     syntaxGenerator
	strs  []string
	kinds []token.Kind
}

func (tc *testCase) run(t *testing.T) {
	next, stop := iter.Pull(tc.g.gen(nil, tc.cfg, tc.s))
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
	s := newSyntax()
	cfg := &SyntaxConfig{
		Rand:               rand.New(rand.NewPCG(0, 0)),
		TurnsInCycleLimit:  0,
		PossibilitiesLimit: 3,
	}
	cases := []testCase2{
		{
			cfg:    cfg,
			s:      s,
			g:      *s.sqlStmt(),
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
	s := newSyntax()
	cfg := &SyntaxConfig{
		Rand:               rand.New(rand.NewPCG(0, 0)),
		TurnsInCycleLimit:  0,
		PossibilitiesLimit: 3,
	}
	cases := []testCase2{
		{
			cfg:    cfg,
			s:      s,
			g:      *s.analyze(),
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
	s := newSyntax()
	cfg := &SyntaxConfig{
		Rand:               rand.New(rand.NewPCG(0, 0)),
		TurnsInCycleLimit:  0,
		PossibilitiesLimit: 3,
	}
	cases := []testCase{
		{
			cfg:   cfg,
			s:     s,
			g:     *s.begin(),
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
	s := newSyntax()
	cfg := &SyntaxConfig{
		Rand:               rand.New(rand.NewPCG(0, 0)),
		TurnsInCycleLimit:  0,
		PossibilitiesLimit: 3,
	}
	cases := []testCase2{
		{
			cfg:    cfg,
			s:      s,
			g:      *s.commit(),
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
	s := newSyntax()
	cfg := &SyntaxConfig{
		Rand:               rand.New(rand.NewPCG(0, 0)),
		TurnsInCycleLimit:  0,
		PossibilitiesLimit: 3,
	}
	cases := []testCase2{
		{
			cfg:    cfg,
			s:      s,
			g:      *s.detach(),
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
	s := newSyntax()
	cfg := &SyntaxConfig{
		Rand:               rand.New(rand.NewPCG(0, 0)),
		TurnsInCycleLimit:  0,
		PossibilitiesLimit: 3,
	}
	cases := []testCase2{
		{
			cfg:    cfg,
			s:      s,
			g:      *s.dropIndex(),
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
	s := newSyntax()
	cfg := &SyntaxConfig{
		Rand:               rand.New(rand.NewPCG(0, 0)),
		TurnsInCycleLimit:  0,
		PossibilitiesLimit: 3,
	}
	cases := []testCase2{
		{
			cfg:    cfg,
			s:      s,
			g:      *s.dropTable(),
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
	s := newSyntax()
	cfg := &SyntaxConfig{
		Rand:               rand.New(rand.NewPCG(0, 0)),
		TurnsInCycleLimit:  0,
		PossibilitiesLimit: 3,
	}
	cases := []testCase2{
		{
			cfg:    cfg,
			s:      s,
			g:      *s.dropTrigger(),
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
	s := newSyntax()
	cfg := &SyntaxConfig{
		Rand:               rand.New(rand.NewPCG(0, 0)),
		TurnsInCycleLimit:  0,
		PossibilitiesLimit: 3,
	}
	cases := []testCase2{
		{
			cfg:    cfg,
			s:      s,
			g:      *s.dropView(),
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
	s := newSyntax()
	cfg := &SyntaxConfig{
		Rand:               rand.New(rand.NewPCG(0, 0)),
		TurnsInCycleLimit:  0,
		PossibilitiesLimit: 3,
	}
	cases := []testCase2{
		{
			cfg:    cfg,
			s:      s,
			g:      *s.pragma(),
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
	s := newSyntax()
	cfg := &SyntaxConfig{
		Rand:               rand.New(rand.NewPCG(0, 0)),
		TurnsInCycleLimit:  0,
		PossibilitiesLimit: 3,
	}
	cases := []testCase{
		{
			cfg:   cfg,
			s:     s,
			g:     *s.pragmaValue(),
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
	s := newSyntax()
	cfg := &SyntaxConfig{
		Rand:               rand.New(rand.NewPCG(0, 0)),
		TurnsInCycleLimit:  0,
		PossibilitiesLimit: 3,
	}
	cases := []testCase2{
		{
			cfg:    cfg,
			s:      s,
			g:      *s.reindex(),
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
	s := newSyntax()
	cfg := &SyntaxConfig{
		Rand:               rand.New(rand.NewPCG(0, 0)),
		TurnsInCycleLimit:  0,
		PossibilitiesLimit: 3,
	}
	cases := []testCase2{
		{
			cfg:    cfg,
			s:      s,
			g:      *s.release(),
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
	s := newSyntax()
	cfg := &SyntaxConfig{
		Rand:               rand.New(rand.NewPCG(0, 0)),
		TurnsInCycleLimit:  0,
		PossibilitiesLimit: 3,
	}
	cases := []testCase2{
		{
			cfg:    cfg,
			s:      s,
			g:      *s.rollback(),
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
	s := newSyntax()
	cfg := &SyntaxConfig{
		Rand:               rand.New(rand.NewPCG(0, 0)),
		TurnsInCycleLimit:  0,
		PossibilitiesLimit: 1,
	}
	cases := []testCase2{
		{
			cfg:    cfg,
			s:      s,
			g:      *s.savepoint(),
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

func TestSelectStatement(t *testing.T) {
	s := newSyntax()
	cfg := &SyntaxConfig{
		Rand:               rand.New(rand.NewPCG(5, 0)),
		TurnsInCycleLimit:  0,
		PossibilitiesLimit: 1,
	}
	cases := []testCase2{
		{
			cfg:    cfg,
			s:      s,
			g:      *s.selectStatement(),
			strs:   []string{"with recursive a as(values(raise(ignore))) select all a.*window a as(a) order by(exists()) asc"},
			firsts: []token.Kind{token.KindWith},
			lasts:  []token.Kind{token.KindAsc},
		},
	}

	for i, c := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			c.run(t)
		})
	}
}

func TestCommonTableExpression(t *testing.T) {
	s := newSyntax()
	cfg := &SyntaxConfig{
		Rand:               rand.New(rand.NewPCG(2, 50)),
		TurnsInCycleLimit:  0,
		PossibilitiesLimit: 1,
	}
	cases := []testCase2{
		{
			cfg:    cfg,
			s:      s,
			g:      *s.commonTableExpression(),
			strs:   []string{"a(a) as materialized(values(raise(ignore)))"},
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

func TestCompoundOperator(t *testing.T) {
	s := newSyntax()
	cfg := &SyntaxConfig{
		Rand:               rand.New(rand.NewPCG(2, 50)),
		TurnsInCycleLimit:  0,
		PossibilitiesLimit: 1,
	}
	cases := []testCase2{
		{
			cfg:    cfg,
			s:      s,
			g:      *s.compoundOperator(),
			strs:   []string{"union all"},
			firsts: []token.Kind{token.KindUnion},
			lasts:  []token.Kind{token.KindAll},
		},
	}

	for i, c := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			c.run(t)
		})
	}
}

func TestSelectCore(t *testing.T) {
	s := newSyntax()
	cfg := &SyntaxConfig{
		Rand:               rand.New(rand.NewPCG(2, 50)),
		TurnsInCycleLimit:  0,
		PossibilitiesLimit: 1,
	}
	cases := []testCase2{
		{
			cfg:    cfg,
			s:      s,
			g:      *s.selectCore(),
			strs:   []string{"values('a' regexp a())"},
			firsts: []token.Kind{token.KindValues},
			lasts:  []token.Kind{token.KindRightParen},
		},
	}

	for i, c := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			c.run(t)
		})
	}
}

func TestResultColumnList(t *testing.T) {
	s := newSyntax()
	cfg := &SyntaxConfig{
		Rand:               rand.New(rand.NewPCG(1, 0)),
		TurnsInCycleLimit:  0,
		PossibilitiesLimit: 2,
	}
	cases := []testCase2{
		{
			cfg:    cfg,
			s:      s,
			g:      *s.resultColumnList(),
			strs:   []string{"*"},
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

func TestResultColumn(t *testing.T) {
	s := newSyntax()
	cfg := &SyntaxConfig{
		Rand:               rand.New(rand.NewPCG(1, 0)),
		TurnsInCycleLimit:  0,
		PossibilitiesLimit: 2,
	}
	cases := []testCase2{
		{
			cfg:    cfg,
			s:      s,
			g:      *s.resultColumn(),
			strs:   []string{"*"},
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

func TestTableOrSubqueryList(t *testing.T) {
	s := newSyntax()
	cfg := &SyntaxConfig{
		Rand:               rand.New(rand.NewPCG(5, 0)),
		TurnsInCycleLimit:  0,
		PossibilitiesLimit: 1,
	}
	cases := []testCase2{
		{
			cfg:    cfg,
			s:      s,
			g:      *s.tableOrSubqueryList(),
			strs:   []string{"(temp.a)"},
			firsts: []token.Kind{token.KindLeftParen},
			lasts:  []token.Kind{token.KindRightParen},
		},
	}

	for i, c := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			c.run(t)
		})
	}
}

func TestTableOrSubquery(t *testing.T) {
	s := newSyntax()
	cfg := &SyntaxConfig{
		Rand:               rand.New(rand.NewPCG(5, 0)),
		TurnsInCycleLimit:  0,
		PossibilitiesLimit: 2,
	}
	cases := []testCase2{
		{
			cfg:    cfg,
			s:      s,
			g:      *s.tableOrSubquery(),
			strs:   []string{"(temp.a)", "(temp.a not indexed)"},
			firsts: []token.Kind{token.KindLeftParen, token.KindLeftParen},
			lasts:  []token.Kind{token.KindRightParen, token.KindRightParen},
		},
	}

	for i, c := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			c.run(t)
		})
	}
}

func TestJoinClause(t *testing.T) {
	s := newSyntax()
	cfg := &SyntaxConfig{
		Rand:               rand.New(rand.NewPCG(5, 0)),
		TurnsInCycleLimit:  0,
		PossibilitiesLimit: 2,
	}
	cases := []testCase2{
		{
			cfg:    cfg,
			s:      s,
			g:      *s.joinClause(),
			strs:   []string{"(temp.a)"},
			firsts: []token.Kind{token.KindLeftParen},
			lasts:  []token.Kind{token.KindRightParen},
		},
	}

	for i, c := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			c.run(t)
		})
	}
}

func TestJoinOperator(t *testing.T) {
	s := newSyntax()
	cfg := &SyntaxConfig{
		Rand:               rand.New(rand.NewPCG(0, 1)),
		TurnsInCycleLimit:  0,
		PossibilitiesLimit: 2,
	}
	cases := []testCase2{
		{
			cfg:    cfg,
			s:      s,
			g:      *s.joinOperator(),
			strs:   []string{"inner natural join", "inner outer join", "inner outer left join"},
			firsts: []token.Kind{token.KindInner, token.KindInner, token.KindInner},
			lasts:  []token.Kind{token.KindJoin, token.KindJoin, token.KindJoin},
		},
	}

	for i, c := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			c.run(t)
		})
	}
}

func TestJoinConstraint(t *testing.T) {
	s := newSyntax()
	cfg := &SyntaxConfig{
		Rand:               rand.New(rand.NewPCG(0, 2)),
		TurnsInCycleLimit:  0,
		PossibilitiesLimit: 2,
	}
	cases := []testCase2{
		{
			cfg:    cfg,
			s:      s,
			g:      *s.joinConstraint(),
			strs:   []string{"on ?1", "using(a)", "using(a, a)"},
			firsts: []token.Kind{token.KindOn, token.KindUsing, token.KindUsing},
			lasts:  []token.Kind{token.KindQuestionVariable, token.KindRightParen, token.KindRightParen},
		},
	}

	for i, c := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			c.run(t)
		})
	}
}

func TestWindowDeclarationList(t *testing.T) {
	s := newSyntax()
	cfg := &SyntaxConfig{
		Rand:               rand.New(rand.NewPCG(0, 2)),
		TurnsInCycleLimit:  0,
		PossibilitiesLimit: 2,
	}
	cases := []testCase2{
		{
			cfg: cfg,
			s:   s,
			g:   *s.windowDeclarationList(),
			strs: []string{
				"a as(rows between unbounded preceding and() preceding)",
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

func TestWindowDeclaration(t *testing.T) {
	s := newSyntax()
	cfg := &SyntaxConfig{
		Rand:               rand.New(rand.NewPCG(1, 1)),
		TurnsInCycleLimit:  0,
		PossibilitiesLimit: 3,
	}
	cases := []testCase2{
		{
			cfg: cfg,
			s:   s,
			g:   *s.windowDeclaration(),
			strs: []string{
				"a as(a partition by a.a collate a)",
				"a as(a partition by a.a collate a groups current row)",
				"a as(a partition by a.a collate a groups current row exclude group)",
			},
			firsts: []token.Kind{token.KindIdentifier, token.KindIdentifier, token.KindIdentifier},
			lasts:  []token.Kind{token.KindRightParen, token.KindRightParen, token.KindRightParen},
		},
	}

	for i, c := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			c.run(t)
		})
	}
}

func TestWindowDefinition(t *testing.T) {
	s := newSyntax()
	cfg := &SyntaxConfig{
		Rand:               rand.New(rand.NewPCG(1, 1)),
		TurnsInCycleLimit:  0,
		PossibilitiesLimit: 3,
	}
	cases := []testCase2{
		{
			cfg: cfg,
			s:   s,
			g:   *s.windowDefinition(),
			strs: []string{
				"(a partition by a.a collate a)",
				"(a partition by a.a collate a groups current row)",
				"(a partition by a.a collate a groups current row exclude group)",
			},
			firsts: []token.Kind{token.KindLeftParen, token.KindLeftParen, token.KindLeftParen},
			lasts:  []token.Kind{token.KindRightParen, token.KindRightParen, token.KindRightParen},
		},
	}

	for i, c := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			c.run(t)
		})
	}
}

func TestVacuum(t *testing.T) {
	s := newSyntax()
	cfg := &SyntaxConfig{
		Rand:               rand.New(rand.NewPCG(0, 0)),
		TurnsInCycleLimit:  0,
		PossibilitiesLimit: 1,
	}
	cases := []testCase2{
		{
			cfg:    cfg,
			s:      s,
			g:      *s.vacuum(),
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
	s := newSyntax()
	cfg := &SyntaxConfig{
		Rand:               rand.New(rand.NewPCG(0, 0)),
		TurnsInCycleLimit:  0,
		PossibilitiesLimit: 1,
	}
	cases := []testCase2{
		{
			cfg:    cfg,
			s:      s,
			g:      *s.expression(),
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
	s := newSyntax()
	cfg := &SyntaxConfig{
		Rand:               rand.New(rand.NewPCG(1, 0)),
		TurnsInCycleLimit:  0,
		PossibilitiesLimit: 4,
	}
	cases := []testCase{
		{
			cfg:   cfg,
			s:     s,
			g:     *s.unaryOper(),
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
	s := newSyntax()
	cfg := &SyntaxConfig{
		Rand:               rand.New(rand.NewPCG(1, 0)),
		TurnsInCycleLimit:  0,
		PossibilitiesLimit: 4,
	}
	cases := []testCase{
		{
			cfg:   cfg,
			s:     s,
			g:     *s.binaryOper(),
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
	s := newSyntax()
	cfg := &SyntaxConfig{
		Rand:               rand.New(rand.NewPCG(1, 0)),
		TurnsInCycleLimit:  0,
		PossibilitiesLimit: 1,
	}
	cases := []testCase2{
		{
			cfg: cfg,
			s:   s,
			g:   *s.functionCall(),
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
	s := newSyntax()
	cfg := &SyntaxConfig{
		Rand:               rand.New(rand.NewPCG(1, 2)),
		TurnsInCycleLimit:  0,
		PossibilitiesLimit: 1,
	}
	cases := []testCase2{
		{
			cfg: cfg,
			s:   s,
			g:   *s.functionArguments(),
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
	s := newSyntax()
	cfg := &SyntaxConfig{
		Rand:               rand.New(rand.NewPCG(0, 0)),
		TurnsInCycleLimit:  0,
		PossibilitiesLimit: 1,
	}
	cases := []testCase2{
		{
			cfg: cfg,
			s:   s,
			g:   *s.orderingTerm(),
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
	s := newSyntax()
	cfg := &SyntaxConfig{
		Rand:               rand.New(rand.NewPCG(0, 0)),
		TurnsInCycleLimit:  0,
		PossibilitiesLimit: 2,
	}
	cases := []testCase2{
		{
			cfg: cfg,
			s:   s,
			g:   *s.orderingTerm(),
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
	s := newSyntax()
	cfg := &SyntaxConfig{
		Rand:               rand.New(rand.NewPCG(0, 0)),
		TurnsInCycleLimit:  0,
		PossibilitiesLimit: 1,
	}
	cases := []testCase2{
		{
			cfg: cfg,
			s:   s,
			g:   *s.orderingTermList(),
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
	s := newSyntax()
	cfg := &SyntaxConfig{
		Rand:               rand.New(rand.NewPCG(0, 0)),
		TurnsInCycleLimit:  0,
		PossibilitiesLimit: 3,
	}
	cases := []testCase2{
		{
			cfg: cfg,
			s:   s,
			g:   *s.filterClause(),
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
	s := newSyntax()
	cfg := &SyntaxConfig{
		Rand:               rand.New(rand.NewPCG(0, 0)),
		TurnsInCycleLimit:  0,
		PossibilitiesLimit: 2,
	}
	cases := []testCase2{
		{
			cfg: cfg,
			s:   s,
			g:   *s.overClause(),
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
	s := newSyntax()
	cfg := &SyntaxConfig{
		Rand:               rand.New(rand.NewPCG(0, 0)),
		TurnsInCycleLimit:  0,
		PossibilitiesLimit: 2,
	}
	cases := []testCase2{
		{
			cfg: cfg,
			s:   s,
			g:   *s.frameSpec(),
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
	s := newSyntax()
	cfg := &SyntaxConfig{
		Rand:               rand.New(rand.NewPCG(1, 1)),
		TurnsInCycleLimit:  0,
		PossibilitiesLimit: 2,
	}
	cases := []testCase2{
		{
			cfg:    cfg,
			s:      s,
			g:      *s.typeName(),
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
	s := newSyntax()
	cfg := &SyntaxConfig{
		Rand:               rand.New(rand.NewPCG(0, 0)),
		TurnsInCycleLimit:  0,
		PossibilitiesLimit: 3,
	}
	cases := []testCase{
		{
			cfg:   cfg,
			s:     s,
			g:     *s.schemaName(),
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
	s := newSyntax()
	cfg := &SyntaxConfig{
		Rand:               rand.New(rand.NewPCG(0, 0)),
		TurnsInCycleLimit:  0,
		PossibilitiesLimit: 3,
	}
	cases := []testCase2{
		{
			cfg:    cfg,
			s:      s,
			g:      *s.signedLiteral(),
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
	s := newSyntax()
	cfg := &SyntaxConfig{
		Rand:               rand.New(rand.NewPCG(0, 0)),
		TurnsInCycleLimit:  0,
		PossibilitiesLimit: 3,
	}
	cases := []testCase2{
		{
			cfg:    cfg,
			s:      s,
			g:      *s.signedNumber(),
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
	s := newSyntax()
	cfg := &SyntaxConfig{
		Rand:               rand.New(rand.NewPCG(0, 0)),
		TurnsInCycleLimit:  0,
		PossibilitiesLimit: 4,
	}
	cases := []testCase{
		{
			cfg:   cfg,
			s:     s,
			g:     *s.id(),
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
	s := newSyntax()
	cfg := &SyntaxConfig{
		Rand:               rand.New(rand.NewPCG(0, 0)),
		TurnsInCycleLimit:  0,
		PossibilitiesLimit: 4,
	}
	cases := []testCase{
		{
			cfg:   cfg,
			s:     s,
			g:     *s.literal(),
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
	s := newSyntax()
	cfg := &SyntaxConfig{
		Rand:               rand.New(rand.NewPCG(0, 0)),
		TurnsInCycleLimit:  0,
		PossibilitiesLimit: 4,
	}
	cases := []testCase{
		{
			cfg:   cfg,
			s:     s,
			g:     *s.string(),
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
	s := newSyntax()
	cfg := &SyntaxConfig{
		Rand:               rand.New(rand.NewPCG(0, 0)),
		TurnsInCycleLimit:  0,
		PossibilitiesLimit: 4,
	}
	cases := []testCase{
		{
			cfg:   cfg,
			s:     s,
			g:     *s.blob(),
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
	s := newSyntax()
	cfg := &SyntaxConfig{
		Rand:               rand.New(rand.NewPCG(0, 0)),
		TurnsInCycleLimit:  0,
		PossibilitiesLimit: 4,
	}
	cases := []testCase{
		{
			cfg:   cfg,
			s:     s,
			g:     *s.int(),
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
	s := newSyntax()
	cfg := &SyntaxConfig{
		Rand:               rand.New(rand.NewPCG(0, 0)),
		TurnsInCycleLimit:  0,
		PossibilitiesLimit: 4,
	}
	cases := []testCase{
		{
			cfg:   cfg,
			s:     s,
			g:     *s.float(),
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
	s := newSyntax()
	cfg := &SyntaxConfig{
		Rand:               rand.New(rand.NewPCG(0, 0)),
		TurnsInCycleLimit:  0,
		PossibilitiesLimit: 4,
	}
	cases := []testCase{
		{
			cfg:   cfg,
			s:     s,
			g:     *s.comment(),
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
	s := newSyntax()
	cfg := &SyntaxConfig{
		Rand:               rand.New(rand.NewPCG(0, 0)),
		TurnsInCycleLimit:  0,
		PossibilitiesLimit: 4,
	}
	cases := []testCase{
		{
			cfg:   cfg,
			s:     s,
			g:     *s.variable(),
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

	s := newSyntax()
	for i, c := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			g := s.kw(c.kind)
			cfg := &SyntaxConfig{
				Rand:               rand.New(rand.NewPCG(1, 1)),
				TurnsInCycleLimit:  0,
				PossibilitiesLimit: 2,
			}
			strs := slices.Collect((*g).gen(nil, cfg, s))
			if len(strs) != 1 {
				t.Errorf("want %d strings, got %d", 1, len(strs))
				return
			}
			if c.want != strs[0] {
				t.Errorf("want %q, got %q", c.want, strs[0])
			}
			if *(*g).firstTokenKind() != c.kind {
				t.Errorf("firstTokenKind = %s, want %s", *(*g).firstTokenKind(), c.kind)
			}
			if *(*g).lastTokenKind() != c.kind {
				t.Errorf("lastTokenKind = %s, want %s", *(*g).lastTokenKind(), c.kind)
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

	s := newSyntax()
	s.kw(token.KindDot)
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

	s := newSyntax()
	for i, c := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			g := s.oper(c.kind)
			cfg := &SyntaxConfig{
				Rand:               rand.New(rand.NewPCG(1, 1)),
				TurnsInCycleLimit:  0,
				PossibilitiesLimit: 2,
			}
			strs := slices.Collect((*g).gen(nil, cfg, s))
			if len(strs) != 1 {
				t.Errorf("want %d strings, got %d", 1, len(strs))
				return
			}
			if c.want != strs[0] {
				t.Errorf("want %q, got %q", c.want, strs[0])
			}
			if *(*g).firstTokenKind() != c.kind {
				t.Errorf("firstTokenKind = %s, want %s", *(*g).firstTokenKind(), c.kind)
			}
			if *(*g).lastTokenKind() != c.kind {
				t.Errorf("lastTokenKind = %s, want %s", *(*g).lastTokenKind(), c.kind)
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

	s := newSyntax()
	s.oper(token.KindSelect)
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

	s := newSyntax()
	for i, c := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			g := s.punct(c.kind)
			cfg := &SyntaxConfig{
				Rand:               rand.New(rand.NewPCG(1, 1)),
				TurnsInCycleLimit:  0,
				PossibilitiesLimit: 2,
			}
			strs := slices.Collect((*g).gen(nil, cfg, s))
			if len(strs) != 1 {
				t.Errorf("want %d strings, got %d", 1, len(strs))
				return
			}
			if c.want != strs[0] {
				t.Errorf("want %q, got %q", c.want, strs[0])
			}
			if *(*g).firstTokenKind() != c.kind {
				t.Errorf("firstTokenKind = %s, want %s", *(*g).firstTokenKind(), c.kind)
			}
			if *(*g).lastTokenKind() != c.kind {
				t.Errorf("lastTokenKind = %s, want %s", *(*g).lastTokenKind(), c.kind)
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

	s := newSyntax()
	s.punct(token.KindSelect)
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
	s := newSyntax()
	concat := (*s.conc()).(*concatSynGen)
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

func TestEpsilon(t *testing.T) {
	s := newSyntax()

	cfg := &SyntaxConfig{
		Rand:               rand.New(rand.NewPCG(0, 0)),
		TurnsInCycleLimit:  2,
		PossibilitiesLimit: 10,
	}

	cases := []testCase2{
		{
			cfg:    cfg,
			g:      *s.eps(),
			strs:   []string{""},
			firsts: []token.Kind{nil},
			lasts:  []token.Kind{nil},
		},
	}

	for i, c := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			c.run(t)
		})
	}
}

func TestStar0Break(t *testing.T) {
	cfg := &SyntaxConfig{
		Rand:               rand.New(rand.NewPCG(0, 0)),
		TurnsInCycleLimit:  2,
		PossibilitiesLimit: 10,
	}
	s := newSyntax()
	g := s.star(s.id())
	(*g).gen(nil, cfg, s)(func(str string) bool {
		if str != "" {
			t.Errorf("want str = %q, got %q", "", str)
		}
		if (*g).firstTokenKind() != nil {
			t.Errorf("want firstTokenKind = nil, got %s", *(*g).firstTokenKind())
		}
		if (*g).lastTokenKind() != nil {
			t.Errorf("want lastTokenKind = nil, got %s", *(*g).lastTokenKind())
		}
		return false
	})
}

func TestStar(t *testing.T) {
	s := newSyntax()
	cfg := &SyntaxConfig{
		Rand:               rand.New(rand.NewPCG(0, 0)),
		TurnsInCycleLimit:  2,
		PossibilitiesLimit: 10,
	}
	cases := []testCase{
		{
			cfg:   cfg,
			g:     *s.star(s.id()),
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
	s := newSyntax()
	cfg := &SyntaxConfig{
		Rand:               rand.New(rand.NewPCG(0, 0)),
		TurnsInCycleLimit:  2,
		PossibilitiesLimit: 10,
	}
	cases := []testCase{
		{
			cfg:   cfg,
			g:     *s.plus(s.id()),
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

func TestConcat(t *testing.T) {
	s := newSyntax()
	cfg := &SyntaxConfig{
		Rand:               rand.New(rand.NewPCG(0, 0)),
		TurnsInCycleLimit:  2,
		PossibilitiesLimit: 10,
	}
	cases := []testCase2{
		{
			cfg:    cfg,
			g:      *s.conc(s.eps(), s.id()),
			strs:   []string{"a"},
			firsts: []token.Kind{token.KindIdentifier},
			lasts:  []token.Kind{token.KindIdentifier},
		}, {
			cfg:    cfg,
			g:      *s.conc(s.id(), s.eps()),
			strs:   []string{"a"},
			firsts: []token.Kind{token.KindIdentifier},
			lasts:  []token.Kind{token.KindIdentifier},
		},
	}

	for i, c := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			c.run(t)
		})
	}
}

func TestOr(t *testing.T) {
	s := newSyntax()

	cfg := &SyntaxConfig{
		Rand:               rand.New(rand.NewPCG(0, 0)),
		TurnsInCycleLimit:  2,
		PossibilitiesLimit: 10,
	}

	var rec syntaxGenerator = &orSynGen{}
	rec.(*orSynGen).gs = append(rec.(*orSynGen).gs, s.id(), &rec)

	cases := []testCase2{
		{
			cfg: cfg,
			g: *s.or(
				s.id(), s.kw(token.KindSelect),
				s.oper(token.KindEqual), s.savepoint(),
			),
			strs:   []string{"a", "=", "savepoint a"},
			firsts: []token.Kind{token.KindIdentifier, token.KindEqual, token.KindSavepoint},
			lasts:  []token.Kind{token.KindIdentifier, token.KindEqual, token.KindIdentifier},
		}, {
			cfg:    cfg,
			g:      rec,
			strs:   []string{"a", "a"},
			firsts: []token.Kind{token.KindIdentifier, token.KindIdentifier},
			lasts:  []token.Kind{token.KindIdentifier, token.KindIdentifier},
		},
	}

	for i, c := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			c.run(t)
		})
	}
}

func TestCount(t *testing.T) {
	s := newSyntax()
	cases := []struct {
		s     []*syntaxGenerator
		e     *syntaxGenerator
		count int
	}{
		{e: s.expression()},
		{s: []*syntaxGenerator{s.expression()}, e: s.expression(), count: 1},
		{s: []*syntaxGenerator{s.typeName(), s.expression()}, e: s.typeName(), count: 1},
		{s: []*syntaxGenerator{s.expression(), s.expression()}, e: s.expression(), count: 2},
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

func TestPrintStack(t *testing.T) {
	s := newSyntax()
	var stack []*syntaxGenerator

	stack = append(stack,
		s.sqlStmt(), s.analyze(), s.begin(), s.commit(), s.detach(), s.dropIndex(), s.dropTable(),
		s.dropTrigger(), s.dropView(), s.pragma(), s.pragmaValue(), s.reindex(), s.release(),
		s.rollback(), s.savepoint(), s.selectStatement(), s.selectCore(), s.commonTableExpression(),
		s.compoundOperator(), s.resultColumnList(), s.resultColumn(), s.tableOrSubqueryList(), s.tableOrSubquery(),
		s.joinClause(), s.joinOperator(), s.joinConstraint(), s.windowDeclarationList(), s.windowDeclaration(),
		s.windowDefinition(), s.vacuum(), s.expression(), s.expressionList(), s.functionCall(), s.functionArguments(),
		s.orderingTerm(), s.orderingTermList(), s.filterClause(), s.overClause(), s.frameSpec(), s.typeName(),
		s.signedNumber(), s.signedLiteral(), s.eps(),
	)
	var want strings.Builder
	for _, str := range []string{
		"sqlStmt", "analyze", "begin", "commit", "detach", "dropIndex", "dropTable", "dropTrigger", "dropView", "pragma",
		"pragmaValue", "reindex", "release", "rollback", "savepoint", "selectStatement", "selectCore", "commonTableExpression",
		"compoundOperator", "resultColumnList", "resultColumn", "tableOrSubqueryList", "tableOrSubquery", "joinClause",
		"joinOperator", "joinConstraint", "windowDeclarationList", "windowDeclaration", "windowDefinition", "vacuum",
		"expression", "expressionList", "functionCall", "functionArguments", "orderingTerm", "orderingTermList", "filterClause",
		"overClause", "frameSpec", "typeName", "signedNumber", "signedLiteral", "...",
	} {
		fmt.Fprintln(&want, str)
	}

	var b strings.Builder
	s.printStack(&b, stack)

	if b.String() != want.String() {
		t.Errorf("want %q, got %q", want.String(), b.String())
	}
}
