package wallzaos

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/google/uuid"
)

const (
	deliveryTable        = "bm_wallzaos_webhook_deliveries"
	maxAttempts          = 6
	defaultRetentionDays = 30
	minRetentionDays     = 7
	maxRetentionDays     = 365
)

var (
	dispatchMu  sync.Mutex
	webhookHTTP = &http.Client{
		Timeout: 5 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	retryDelays = []time.Duration{
		30 * time.Second,
		2 * time.Minute,
		10 * time.Minute,
		30 * time.Minute,
		2 * time.Hour,
	}
)

type webhookConfig struct {
	URL    string
	Secret string
}

type webhookEnvelope struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Source    string                 `json:"source"`
	CreatedAt time.Time              `json:"created_at"`
	Data      map[string]interface{} `json:"data"`
}

type webhookDelivery struct {
	EventID   string `json:"event_id"`
	EventType string `json:"event_type"`
	Payload   string `json:"payload"`
	Attempts  int    `json:"attempts"`
}

type httpDoer interface {
	Do(*http.Request) (*http.Response, error)
}

func loadWebhookConfig() (webhookConfig, bool, error) {
	rawURL := strings.TrimSpace(os.Getenv("WALLZAOS_WEBHOOK_URL"))
	secret := strings.TrimSpace(os.Getenv("WALLZAOS_WEBHOOK_SECRET"))
	if rawURL == "" && secret == "" {
		return webhookConfig{}, false, nil
	}
	if rawURL == "" || secret == "" {
		return webhookConfig{}, false, errors.New("both WALLZAOS_WEBHOOK_URL and WALLZAOS_WEBHOOK_SECRET must be configured")
	}
	if len(secret) < 32 {
		return webhookConfig{}, false, errors.New("WALLZAOS_WEBHOOK_SECRET must contain at least 32 characters")
	}

	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil || parsed.Hostname() == "" || parsed.User != nil || parsed.Fragment != "" {
		return webhookConfig{}, false, errors.New("WALLZAOS_WEBHOOK_URL must be a valid absolute HTTP(S) URL without credentials or a fragment")
	}
	if parsed.Scheme != "https" {
		allowLocalHTTP := strings.EqualFold(os.Getenv("WALLZAOS_WEBHOOK_ALLOW_LOCAL_HTTP"), "true")
		hostname := strings.ToLower(parsed.Hostname())
		isLoopback := hostname == "localhost" || hostname == "127.0.0.1" || hostname == "::1"
		if parsed.Scheme != "http" || !allowLocalHTTP || !isLoopback {
			return webhookConfig{}, false, errors.New("WALLZAOS_WEBHOOK_URL must use HTTPS; HTTP is allowed only for localhost testing when WALLZAOS_WEBHOOK_ALLOW_LOCAL_HTTP=true")
		}
	}

	return webhookConfig{URL: rawURL, Secret: secret}, true, nil
}

func signature(secret string, payload []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(payload)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// Enqueue adds a WallzaOS event to the durable delivery queue.
// Events are ignored when webhook delivery is not configured.
func Enqueue(ctx context.Context, eventType string, data map[string]interface{}) {
	EnqueueOnce(ctx, uuid.NewString(), eventType, data)
}

// EnqueueOnce adds an event with a caller-supplied idempotency key.
// Reprocessing the same Postfix log entry therefore does not create duplicate deliveries.
func EnqueueOnce(ctx context.Context, eventID, eventType string, data map[string]interface{}) {
	if strings.TrimSpace(eventType) == "" || strings.TrimSpace(eventID) == "" {
		return
	}
	_, enabled, err := loadWebhookConfig()
	if err != nil {
		g.Log().Warning(ctx, "WallzaOS webhook is not configured correctly; preserving event until configuration is fixed: ", err)
		// Do not lose mail events because a partially configured endpoint is invalid.
		// Keep them in the outbox so the dispatcher can deliver them after correction.
		if strings.TrimSpace(os.Getenv("WALLZAOS_WEBHOOK_URL")) == "" &&
			strings.TrimSpace(os.Getenv("WALLZAOS_WEBHOOK_SECRET")) == "" {
			return
		}
	} else if !enabled {
		return
	}

	envelope := webhookEnvelope{
		ID:        eventID,
		Type:      eventType,
		Source:    "billionmail",
		CreatedAt: time.Now().UTC(),
		Data:      data,
	}
	payload, err := json.Marshal(envelope)
	if err != nil {
		g.Log().Warning(ctx, "Could not encode WallzaOS webhook event: ", err)
		return
	}

	now := time.Now().Unix()
	_, err = g.DB().Model(deliveryTable).Ctx(ctx).InsertIgnore(g.Map{
		"event_id":        eventID,
		"event_type":      eventType,
		"payload":         string(payload),
		"status":          "pending",
		"attempts":        0,
		"next_attempt_at": now,
		"created_at":      now,
		"delivered_at":    0,
		"last_error":      "",
	})
	if err != nil {
		g.Log().Warning(ctx, "Could not queue WallzaOS webhook event: ", err)
	}
}

// DispatchPending sends queued events. A row is claimed before network I/O so
// overlapping timer invocations cannot send the same pending row concurrently.
func DispatchPending(ctx context.Context, limit int) {
	if !dispatchMu.TryLock() {
		return
	}
	defer dispatchMu.Unlock()

	cfg, enabled, err := loadWebhookConfig()
	if err != nil {
		g.Log().Warning(ctx, "WallzaOS webhook dispatcher disabled: ", err)
		return
	}
	if !enabled {
		return
	}
	if limit < 1 {
		limit = 1
	}
	if limit > 25 {
		limit = 25
	}

	now := time.Now().Unix()
	// Recover claims left behind by a process restart or crash.
	_, recoverErr := g.DB().Model(deliveryTable).Ctx(ctx).
		Where("status", "processing").
		Where("processing_started_at < ?", now-120).
		Update(g.Map{"status": "pending", "processing_started_at": 0})
	if recoverErr != nil {
		g.Log().Warning(ctx, "Could not recover stale WallzaOS webhook claims: ", recoverErr)
	}

	var rows []webhookDelivery
	err = g.DB().Model(deliveryTable).Ctx(ctx).
		Fields("event_id, event_type, payload, attempts").
		Where("status", "pending").
		Where("next_attempt_at <= ?", time.Now().Unix()).
		OrderAsc("created_at").
		Limit(limit).
		Scan(&rows)
	if err != nil {
		g.Log().Warning(ctx, "Could not read WallzaOS webhook queue: ", err)
		return
	}

	for _, row := range rows {
		if ctx.Err() != nil {
			return
		}

		result, claimErr := g.DB().Model(deliveryTable).Ctx(ctx).
			Where("event_id", row.EventID).
			Where("status", "pending").
			Update(g.Map{"status": "processing", "processing_started_at": time.Now().Unix()})
		if claimErr != nil {
			g.Log().Warning(ctx, "Could not claim WallzaOS webhook delivery: ", claimErr)
			continue
		}
		claimed, claimErr := result.RowsAffected()
		if claimErr != nil || claimed == 0 {
			continue
		}

		sendErr := sendWebhook(ctx, webhookHTTP, cfg, row)
		attempts := row.Attempts + 1
		if sendErr == nil {
			_, updateErr := g.DB().Model(deliveryTable).Ctx(ctx).
				Where("event_id", row.EventID).
				Update(g.Map{
					"status":                "delivered",
					"attempts":              attempts,
					"delivered_at":          time.Now().Unix(),
					"processing_started_at": 0,
					"last_error":            "",
				})
			if updateErr != nil {
				g.Log().Warning(ctx, "Could not mark WallzaOS webhook as delivered: ", updateErr)
			}
			continue
		}

		nextStatus := "pending"
		nextAttempt := time.Now().Add(retryDelays[len(retryDelays)-1]).Unix()
		if attempts >= maxAttempts {
			nextStatus = "failed"
			nextAttempt = 0
		} else {
			nextAttempt = time.Now().Add(retryDelays[attempts-1]).Unix()
		}
		_, updateErr := g.DB().Model(deliveryTable).Ctx(ctx).
			Where("event_id", row.EventID).
			Update(g.Map{
				"status":                nextStatus,
				"attempts":              attempts,
				"next_attempt_at":       nextAttempt,
				"processing_started_at": 0,
				"last_error":            truncateError(sendErr.Error(), 1000),
			})
		if updateErr != nil {
			g.Log().Warning(ctx, "Could not update failed WallzaOS webhook delivery: ", updateErr)
		}
	}
}

func sendWebhook(ctx context.Context, client httpDoer, cfg webhookConfig, delivery webhookDelivery) error {
	payload := []byte(delivery.Payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.URL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "BillionMail-WallzaOS-Webhooks/1.0")
	req.Header.Set("X-WallzaOS-Event", delivery.EventType)
	req.Header.Set("X-WallzaOS-Delivery", delivery.EventID)
	req.Header.Set("X-WallzaOS-Signature", signature(cfg.Secret, payload))

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
	return fmt.Errorf("endpoint returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
}

func truncateError(value string, max int) string {
	if len(value) <= max {
		return value
	}
	return value[:max]
}

func ConfiguredRetentionDays() int {
	raw := strings.TrimSpace(os.Getenv("WALLZAOS_WEBHOOK_RETENTION_DAYS"))
	if raw == "" {
		return defaultRetentionDays
	}
	days, err := strconv.Atoi(raw)
	if err != nil || days < minRetentionDays || days > maxRetentionDays {
		g.Log().Warning(context.Background(), "Invalid WALLZAOS_WEBHOOK_RETENTION_DAYS; using default retention of ", defaultRetentionDays, " days")
		return defaultRetentionDays
	}
	return days
}

// PruneTerminalDeliveries removes only completed or permanently failed events.
// Pending and processing deliveries are retained regardless of age.
func PruneTerminalDeliveries(ctx context.Context) {
	days := ConfiguredRetentionDays()
	cutoff := time.Now().Add(-time.Duration(days) * 24 * time.Hour).Unix()
	result, err := g.DB().Model(deliveryTable).Ctx(ctx).
		WhereIn("status", []string{"delivered", "failed"}).
		Where("created_at < ?", cutoff).
		Delete()
	if err != nil {
		g.Log().Warning(ctx, "Could not prune expired WallzaOS webhook deliveries: ", err)
		return
	}
	deleted, err := result.RowsAffected()
	if err == nil && deleted > 0 {
		g.Log().Info(ctx, "Pruned expired WallzaOS webhook deliveries: ", deleted, " records; retention days: ", days)
	}
}
