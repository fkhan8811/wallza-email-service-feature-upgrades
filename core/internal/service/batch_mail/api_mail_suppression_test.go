package batch_mail

import "testing"

func TestNormalizeSuppressionRecipient(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "lowercases and trims", input: "  Customer@Example.COM ", expected: "customer@example.com"},
		{name: "already normalized", input: "customer@example.com", expected: "customer@example.com"},
		{name: "empty stays empty", input: "  ", expected: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeSuppressionRecipient(tt.input); got != tt.expected {
				t.Fatalf("normalizeSuppressionRecipient(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}
