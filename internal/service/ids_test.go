package service

import "testing"

func TestRoundTrip(t *testing.T) {
	id, err := BuildEventID("202609", 1)
	if err != nil || id != "2026090001" {
		t.Fatalf("got %q,%v", id, err)
	}
	got, err := NormalizeEventID(id)
	if err != nil || got != id {
		t.Fatalf("normalize failed: %q,%v", got, err)
	}
	if _, err := NormalizeEventID(int64(42)); err != nil {
		t.Fatal(err)
	}
	got, _ = NormalizeEventID(" 0012 ")
	if got != "0012" {
		t.Fatalf("trim/zero failed: %q", got)
	}
}
