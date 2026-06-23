package app

import "testing"

func TestEscapeLikePattern(t *testing.T) {
	got := escapeLikePattern(`a%b_c\d`)
	want := `a\%b\_c\\d`
	if got != want {
		t.Fatalf("escapeLikePattern() = %q, want %q", got, want)
	}
}
