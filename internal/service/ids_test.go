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

func TestBuildEventIDRejectsInvalidValues(t *testing.T) {
	testCases := []struct {
		name     string
		prefix   string
		sequence int64
	}{
		{
			name:     "prefix inválido",
			prefix:   "2026",
			sequence: 1,
		}, {
			name:     "sequencia zero",
			prefix:   "202609",
			sequence: 0,
		}, {
			name:     "sequência negativa",
			prefix:   "202609",
			sequence: -1,
		}, {
			name:     "sequência acima do limite",
			prefix:   "202609",
			sequence: 10000,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if _, err := BuildEventID(testCase.prefix, testCase.sequence); err == nil {
				t.Fatalf("expected error for prefix=%q sequence=%d", testCase.prefix, testCase.sequence)
			}
		})
	}
}

func TestNormalizeEventIDPreservesLeadingZeros(t *testing.T) {
	normalizedID, err := NormalizeEventID("00001234")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if normalizedID != "00001234" {
		t.Fatalf("expected leadin zeros to be preserved, got %q", normalizedID)
	}
}

func TestNormalizeEventIDRejectsEmptyValue(t *testing.T) {
	if _, err := NormalizeEventID("    "); err == nil {
		t.Fatal("expected error for empty id")
	}
}
