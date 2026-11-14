package parsetree

import (
	"fmt"
	"slices"
	"testing"

	"github.com/joaobnv/mel/sqlite/v3_46_1/token"
)

func TestParseTree(t *testing.T) {
	tr := NewNonTerminal(KindSQLStatement)
	tr.AddChild(NewNonTerminal(KindAlterTable))
	tr.AddChild(NewTerminal(KindToken, token.New([]byte("SELECT"), token.KindSelect)))

	if tr.Kind() != KindSQLStatement {
		t.Logf("tr.Kind = %s, want %s", tr.Kind(), KindSQLStatement)
		t.Fail()
	}
	
	tr.SetKind(KindAnd)
	if tr.Kind() != KindAnd {
		t.Logf("tr.Kind = %s, want %s", tr.Kind(), KindAnd)
		t.Fail()
	}

	if tr.NumberOfChildren() != 2 {
		fmt.Printf("want 2 children, got %d", tr.NumberOfChildren())
		t.Fail()
	}

	cs := slices.Collect(tr.Children)

	if cs[0].Kind() != KindAlterTable {
		fmt.Printf("want %s, got %s", KindAlterTable, cs[0].Kind())
		t.Fail()
	}
	if cs[1].Kind() != KindToken {
		fmt.Printf("want %s, got %s", KindToken, cs[1].Kind())
		t.Fail()
	}
	cs[1].SetKind(KindAnd)
	if cs[1].Kind() != KindAnd {
		t.Logf("cs[1].Kind = %s, want %s", tr.Kind(), KindAnd)
		t.Fail()
	}
	if string(cs[1].(Terminal).Token().Lexeme) != "SELECT" {
		fmt.Printf("want %q, got %q", "SELECT", string(cs[1].(Terminal).Token().Lexeme))
		t.Fail()
	}

	// for the case of yield returning false on the iterator.
	for range tr.Children {
		break
	}
}

func TestKindString(t *testing.T) {
	cases := []struct {
		k   Kind
		str string
	}{
		{KindAdd, "Add"}, {KindDivide, "Divide"}, {KindFunctionName, "FunctionName"},
		{KindParenExpression, "ParenExpression"}, {KindTableOrIndexName, "TableOrIndexName"},
		{KindTypeName, "TypeName"}, {KindUpsertClause, "UpsertClause"}, {KindWindowName, "WindowName"},
		{KindWithClause, "WithClause"}, {-1, "-1"},
	}

	for _, c := range cases {
		if c.k.String() != c.str {
			t.Errorf("want %s, got %s", c.str, c.k)
		}
	}
}

func TestKindStringsIsSorted(t *testing.T) {
	if !slices.IsSorted(kindStrings) {
		t.Error("kindStrings isn't sorted")
		for i := range kindStrings[:len(kindStrings)-1] {
			if kindStrings[i] > kindStrings[i+1] {
				t.Logf("%q > %q", kindStrings[i], kindStrings[i+1])
				break
			}
		}
	}
}
