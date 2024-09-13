package token

import "testing"

// TestKindMarshall tests the Kind's MarshallText method.
func TestKindMarshall(t *testing.T) {
	var k Kind
	k = KindTable
	data, _ := k.MarshalText()
	if string(data) != "Table" {
		t.Errorf("expected Table, got %s", string(data))
	}

	k = Kind(-1)
	data, _ = k.MarshalText()
	if string(data) != "-1" {
		t.Errorf("expected -1, got %s", string(data))
	}

	k = Kind(1000)
	data, _ = k.MarshalText()
	if string(data) != "1000" {
		t.Errorf("expected 1000, got %s", string(data))
	}
}

// TestKindUnmarshall tests the Kind's UnmarshallText method.
func TestKindUnmarshall(t *testing.T) {
	var k Kind
	err := k.UnmarshalText([]byte("Select"))
	if err != nil {
		t.Fatal(err)
	}

	if k != KindSelect {
		t.Errorf("expected KindSelect, got Kind%s", k.String())
	}

	err = k.UnmarshalText([]byte("-1"))
	if err != nil {
		t.Fatal(err)
	}

	if k != -1 {
		t.Errorf("expected -1, got %s", k.String())
	}

	err = k.UnmarshalText([]byte("1000"))
	if err != nil {
		t.Fatal(err)
	}

	if k != 1000 {
		t.Errorf("expected 1000, got %s", k.String())
	}

	err = k.UnmarshalText([]byte("foo"))
	if err == nil {
		t.Errorf("error expected")
	}
}

// TestTokenMarshall tests the token's MarshallText method.
func TestTokenMarshall(t *testing.T) {
	tok := New([]byte("table"), KindTable)
	data, _ := tok.MarshalText()
	if string(data) != `<"table", Table>` {
		t.Errorf(`expected <"table", Table>, got %s`, string(data))
	}
}

// TestTokenUnmarshall tests the token's UnmarshallText method.
func TestTokenUnmarshall(t *testing.T) {
	var tok Token
	err := tok.UnmarshalText([]byte(`<"select", Select>`))
	if err != nil {
		t.Fatal(err)
	}

	if tok.Kind != KindSelect {
		t.Errorf("expected KindSelect, got Kind%s", tok.Kind.String())
	}

	if string(tok.Lexeme) != "select" {
		t.Errorf("expected select, got %s", string(tok.Lexeme))
	}

	if err = tok.UnmarshalText([]byte(`<"\"select\"", Identifier>`)); err != nil {
		t.Fatal(err)
	}

	if tok.Kind != KindIdentifier {
		t.Errorf("expected KindIdentifier, got Kind%s", tok.Kind.String())
	}

	if string(tok.Lexeme) != "\"select\"" {
		t.Errorf("expected \"select\", got %s", string(tok.Lexeme))
	}

	if err = tok.UnmarshalText([]byte(`invalid encoding`)); err == nil {
		t.Error("expected error, got nil")
	} else if err.Error() != "invalid token text encoding" {
		t.Errorf("expected \"invalid token text encoding\", got \"%s\"", err)
	}

	if err = tok.UnmarshalText([]byte(`<"\xselect", Identifier>`)); err == nil {
		t.Error("expected error, got nil")
	}

	if err = tok.UnmarshalText([]byte(`<"select", foo>`)); err == nil {
		t.Error("expected error, got nil")
	}
}
