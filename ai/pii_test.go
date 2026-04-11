package ai

import (
	"strings"
	"testing"
)

func TestRedactPII(t *testing.T) {
	in := "mail me at test@example.com or +1 (555) 123-4567 card 4111 1111 1111 1111"
	out := RedactPII(in)

	if out == in {
		t.Fatalf("expected redaction to change text")
	}
	if strings.Contains(out, "test@example.com") || strings.Contains(out, "555") || strings.Contains(out, "4111 1111") {
		t.Fatalf("pii should be redacted, got %q", out)
	}
}
