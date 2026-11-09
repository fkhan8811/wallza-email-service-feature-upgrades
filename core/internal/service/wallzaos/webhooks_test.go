package wallzaos

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSignatureIsStableAndPrefixed(t *testing.T) {
	got := signature("a-secret-with-enough-entropy", []byte(`{"hello":"world"}`))
	if !strings.HasPrefix(got, "sha256=") {
		t.Fatalf("signature prefix = %q", got)
	}
	if got != signature("a-secret-with-enough-entropy", []byte(`{"hello":"world"}`)) {
		t.Fatal("signature should be deterministic")
	}
	if got == signature("different-secret-with-enough-entropy", []byte(`{"hello":"world"}`)) {
		t.Fatal("different secrets must produce different signatures")
	}
}

func TestSendWebhookSignsExactPayload(t *testing.T) {
	payload := `{"id":"evt-1","type":"email.opened"}`
	secret := "wallza-test-secret-with-more-than-32-characters"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if string(body) != payload {
			t.Fatalf("body = %q", string(body))
		}
		if r.Header.Get("X-WallzaOS-Signature") != signature(secret, []byte(payload)) {
			t.Fatal("signature header does not match request body")
		}
		if r.Header.Get("X-WallzaOS-Delivery") != "evt-1" {
			t.Fatal("missing delivery id")
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	err := sendWebhook(context.Background(), server.Client(), webhookConfig{URL: server.URL, Secret: secret}, webhookDelivery{
		EventID: "evt-1", EventType: "email.opened", Payload: payload,
	})
	if err != nil {
		t.Fatalf("sendWebhook returned error: %v", err)
	}
}

func TestSendWebhookReturnsNon2xxAsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("upstream unavailable"))
	}))
	defer server.Close()

	err := sendWebhook(context.Background(), server.Client(), webhookConfig{
		URL: server.URL, Secret: "wallza-test-secret-with-more-than-32-characters",
	}, webhookDelivery{EventID: "evt-2", EventType: "email.clicked", Payload: `{}`})
	if err == nil || !strings.Contains(err.Error(), "502") {
		t.Fatalf("expected HTTP 502 error, got %v", err)
	}
}

func TestLoadWebhookConfigRequiresHTTPS(t *testing.T) {
	t.Setenv("WALLZAOS_WEBHOOK_URL", "http://example.com/events")
	t.Setenv("WALLZAOS_WEBHOOK_SECRET", "wallza-test-secret-with-more-than-32-characters")
	t.Setenv("WALLZAOS_WEBHOOK_ALLOW_LOCAL_HTTP", "true")

	_, enabled, err := loadWebhookConfig()
	if err == nil || enabled {
		t.Fatalf("expected non-local HTTP URL to be rejected, enabled=%v err=%v", enabled, err)
	}
}

func TestLoadWebhookConfigAllowsOptedInLocalHTTP(t *testing.T) {
	t.Setenv("WALLZAOS_WEBHOOK_URL", "http://127.0.0.1:8080/events")
	t.Setenv("WALLZAOS_WEBHOOK_SECRET", "wallza-test-secret-with-more-than-32-characters")
	t.Setenv("WALLZAOS_WEBHOOK_ALLOW_LOCAL_HTTP", "true")

	cfg, enabled, err := loadWebhookConfig()
	if err != nil || !enabled || cfg.URL == "" {
		t.Fatalf("expected localhost HTTP config to be enabled, config=%+v enabled=%v err=%v", cfg, enabled, err)
	}
}

func TestSendWebhookDoesNotFollowRedirects(t *testing.T) {
	var redirected bool
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		redirected = true
		w.WriteHeader(http.StatusOK)
	}))
	defer target.Close()

	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusFound)
	}))
	defer redirect.Close()

	err := sendWebhook(context.Background(), webhookHTTP, webhookConfig{
		URL: redirect.URL, Secret: "wallza-test-secret-with-more-than-32-characters",
	}, webhookDelivery{EventID: "evt-redirect", EventType: "email.opened", Payload: `{}`})
	if err == nil || !strings.Contains(err.Error(), "302") {
		t.Fatalf("expected HTTP 302 error, got %v", err)
	}
	if redirected {
		t.Fatal("webhook client must not forward signed event bodies to redirect targets")
	}
}
