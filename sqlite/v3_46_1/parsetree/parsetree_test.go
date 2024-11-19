package parsetree

import (
	"errors"
	"fmt"
	"slices"
	"testing"

	"github.com/joaobnv/mel/sqlite/v3_46_1/token"
)

func TestParseTree(t *testing.T) {
	tr := NewNonTerminal(KindSQLStatement)
	tr.AddChild(NewNonTerminal(KindAlterTable))
	tr.AddChild(NewTerminal(KindToken, token.New([]byte("SELECT"), token.KindSelect)))
	tr.AddChild(NewError(KindErrorMissing, errors.New("test error")))

	if tr.NumberOfChildren() != 3 {
		fmt.Printf("want 3 children, got %d", tr.NumberOfChildren())
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
	if string(cs[1].(Terminal).Token().Lexeme) != "SELECT" {
		fmt.Printf("want %q, got %q", "SELECT", string(cs[1].(Terminal).Token().Lexeme))
		t.Fail()
	}
	if cs[2].Kind() != KindErrorMissing {
		fmt.Printf("want %s, got %s", KindErrorMissing, cs[2].Kind())
		t.Fail()
	}
	if cs[2].(Error).Error() != "test error" {
		fmt.Printf("want %q, got %q", "test error", cs[2].(Error).Error())
		t.Fail()
	}

	// for the case of yield returning false on the iterator.
	for range tr.Children {
		break
	}
}

func TestKindString(t *testing.T) {
	if KindSQLStatement.String() != "SQLStatement" {
		fmt.Printf("want %s, got %s", "SQLStatement", KindSQLStatement)
		t.Fail()
	}

	k := Kind(-1)
	if k.String() != "-1" {
		fmt.Printf("want %s, got %s", "-1", k)
		t.Fail()
	}
}
