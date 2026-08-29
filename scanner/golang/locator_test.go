package golang

import "testing"

func TestLocator(t *testing.T) {
	got := Locator("github.com/plexusone/omnillm")
	want := "go:github.com/plexusone/omnillm"
	if got != want {
		t.Errorf("Locator() = %q, want %q", got, want)
	}
}
