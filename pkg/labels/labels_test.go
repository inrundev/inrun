package labels_test

import (
	"testing"

	"github.com/inrundev/inrun/pkg/labels"
)

func TestWithDeletionProtection(t *testing.T) {
	base := map[string]string{
		"app.kubernetes.io/name": "inrun",
		"app.kubernetes.io/tag":  "inrun-internal",
	}

	got := labels.WithDeletionProtection(base)

	if got[labels.DeletionProtectionLabel] != "true" {
		t.Errorf("expected %s=true, got %q", labels.DeletionProtectionLabel, got[labels.DeletionProtectionLabel])
	}

	// Original map must not be mutated.
	if _, ok := base[labels.DeletionProtectionLabel]; ok {
		t.Error("WithDeletionProtection mutated the input map")
	}

	// Existing labels must be preserved.
	for k, v := range base {
		if got[k] != v {
			t.Errorf("key %q: expected %q, got %q", k, v, got[k])
		}
	}
}

func TestWithDeletionProtectionNilInput(t *testing.T) {
	got := labels.WithDeletionProtection(nil)
	if got[labels.DeletionProtectionLabel] != "true" {
		t.Errorf("expected label on nil input, got %v", got)
	}
}
