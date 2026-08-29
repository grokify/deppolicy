package component

import "testing"

func TestEffectiveStatus(t *testing.T) {
	tests := []struct {
		name string
		c    Component
		want Status
	}{
		{"empty defaults to governed", Component{}, StatusGoverned},
		{"explicit governed", Component{Status: StatusGoverned}, StatusGoverned},
		{"foundation", Component{Status: StatusFoundation}, StatusFoundation},
		{"deprecated", Component{Status: StatusDeprecated}, StatusDeprecated},
		{"ungoverned", Component{Status: StatusUngoverned}, StatusUngoverned},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.c.EffectiveStatus(); got != tt.want {
				t.Errorf("EffectiveStatus() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestStatusPredicates(t *testing.T) {
	foundation := Component{Status: StatusFoundation}
	if !foundation.IsFoundation() {
		t.Error("IsFoundation() = false, want true")
	}
	if foundation.IsDeprecated() || foundation.IsUngoverned() {
		t.Error("foundation component reported as deprecated or ungoverned")
	}

	deprecated := Component{Status: StatusDeprecated}
	if !deprecated.IsDeprecated() {
		t.Error("IsDeprecated() = false, want true")
	}

	ungoverned := Component{Status: StatusUngoverned}
	if !ungoverned.IsUngoverned() {
		t.Error("IsUngoverned() = false, want true")
	}

	governed := Component{}
	if governed.IsFoundation() || governed.IsDeprecated() || governed.IsUngoverned() {
		t.Error("default component reported as non-governed")
	}
}
