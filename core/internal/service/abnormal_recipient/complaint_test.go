package abnormal_recipient

import "testing"

func TestNormalizeComplaintInput(t *testing.T) {
	got, err := normalizeComplaintInput(ComplaintInput{
		EventID:          "  event-123  ",
		Recipient:        "  CUSTOMER@Example.com ",
		Provider:         " SendGrid ",
		PostfixMessageID: " postfix-id ",
	})
	if err != nil {
		t.Fatalf("normalizeComplaintInput returned error: %v", err)
	}
	if got.EventID != "event-123" || got.Recipient != "customer@example.com" ||
		got.Provider != "sendgrid" || got.PostfixMessageID != "postfix-id" {
		t.Fatalf("unexpected normalized complaint: %+v", got)
	}
}

func TestNormalizeComplaintInputRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name  string
		input ComplaintInput
	}{
		{name: "missing event ID", input: ComplaintInput{Recipient: "a@example.com", Provider: "ses"}},
		{name: "invalid recipient", input: ComplaintInput{EventID: "evt", Recipient: "not-an-email", Provider: "ses"}},
		{name: "provider with spaces", input: ComplaintInput{EventID: "evt", Recipient: "a@example.com", Provider: "send grid"}},
		{name: "oversized event ID", input: ComplaintInput{EventID: string(make([]byte, 256)), Recipient: "a@example.com", Provider: "ses"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := normalizeComplaintInput(test.input); err == nil {
				t.Fatal("expected invalid complaint input to be rejected")
			}
		})
	}
}
