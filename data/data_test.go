package data

import "testing"

func TestIDTextMarshaler(t *testing.T) {
	r := GenerateRandomID()
	text, err := r.MarshalText()
	if err != nil {
		t.Fatal(err)
	}
	r2 := new(ID)
	err = r2.UnmarshalText(text)
	if err != nil {
		t.Fatal(err)
	}
	if r.Epoch != r2.Epoch {
		t.Fatalf("epoch mismatch: r=%d and r2=%d", r.Epoch, r2.Epoch)
	}
	if r.Random != r2.Random {
		t.Fatal("random mismatch")
	}
}
