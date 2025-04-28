package token

import (
	"fmt"
	"testing"
)

func TestToken(t *testing.T) {
	tok := New([]byte("."), kindDot)
	expected := `<".", Dot>`
	if expected != tok.String() {
		fmt.Printf("expected %q, got %q", expected, tok.String())
		t.Fail()
	}

	k := kind(-1)
	tok = New(nil, &k)
	expected = "<-1>"
	if expected != tok.String() {
		fmt.Printf("expected %q, got %q", expected, tok.String())
		t.Fail()
	}
}

func TestIsKeyword(t *testing.T) {
	cases := []struct {
		in   Kind
		want bool
	}{
		{in: kindAbort, want: true}, {in: kindWithout, want: true},
		{in: kindCascade, want: true}, {in: kindIdentifier, want: false},
		{in: kind(-1), want: false},
	}

	for _, c := range cases {
		got := c.in.IsKeyword()
		if got != c.want {
			t.Errorf("%s.IsKeyword() = %t, want %t", c.in, got, c.want)
		}
	}
}
