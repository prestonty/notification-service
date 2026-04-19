package event

import (
	"errors"
	"testing"
)

func TestLookup(t *testing.T) {
	t.Run("known event returns template", func(t *testing.T) {
		tmpl, err := Lookup(PRMerged)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tmpl.Subject == "" {
			t.Error("expected non-empty subject")
		}
		if tmpl.Body == "" {
			t.Error("expected non-empty body")
		}
		if len(tmpl.Required) == 0 {
			t.Error("expected non-empty required fields")
		}
	})

	t.Run("unknown event returns error", func(t *testing.T) {
		_, err := Lookup("DOES_NOT_EXIST")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, ErrUnknownEvent) {
			t.Errorf("expected ErrUnknownEvent, got: %v", err)
		}
	})

	t.Run("all registered events have required fields", func(t *testing.T) {
		events := []Type{PRMerged, DeploySucceeded, DeployFailed, OrderReady}
		for _, e := range events {
			tmpl, err := Lookup(e)
			if err != nil {
				t.Errorf("event %q: unexpected error: %v", e, err)
				continue
			}
			if len(tmpl.Required) == 0 {
				t.Errorf("event %q: expected required fields", e)
			}
		}
	})
}
