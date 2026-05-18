package token

import "testing"

func TestNew(t *testing.T) {
	value, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if len(value) <= len("tsk_live_") {
		t.Fatalf("token too short: %q", value)
	}
	if value[:9] != "tsk_live_" {
		t.Fatalf("unexpected prefix: %q", value)
	}
}
