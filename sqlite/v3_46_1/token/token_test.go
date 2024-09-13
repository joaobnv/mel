package token

import "testing"

// TestKind tests the Kind's MarshallText method.
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

// TestKind tests the Kind's UnmarshallText method.
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
}
