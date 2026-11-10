package abnormal_recipient

import (
	"context"
	"database/sql"
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

var complaintProviderPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,63}$`)

type ComplaintInput struct {
	EventID          string
	Recipient        string
	Provider         string
	PostfixMessageID string
}

func normalizeComplaintInput(input ComplaintInput) (ComplaintInput, error) {
	input.EventID = strings.TrimSpace(input.EventID)
	input.Recipient = strings.ToLower(strings.TrimSpace(input.Recipient))
	input.Provider = strings.ToLower(strings.TrimSpace(input.Provider))
	input.PostfixMessageID = strings.TrimSpace(input.PostfixMessageID)

	if input.EventID == "" || len(input.EventID) > 255 {
		return ComplaintInput{}, fmt.Errorf("provider event ID must contain 1 to 255 characters")
	}
	if input.Provider == "" || !complaintProviderPattern.MatchString(input.Provider) {
		return ComplaintInput{}, fmt.Errorf("provider must use 1 to 64 letters, numbers, dots, underscores or hyphens")
	}
	if len(input.PostfixMessageID) > 255 {
		return ComplaintInput{}, fmt.Errorf("Postfix message ID must not exceed 255 characters")
	}

	parsed, err := mail.ParseAddress(input.Recipient)
	if err != nil || parsed.Address != input.Recipient || !strings.Contains(input.Recipient, "@") {
		return ComplaintInput{}, fmt.Errorf("recipient must be a valid email address")
	}
	return input, nil
}

// RecordComplaint stores a provider feedback-loop complaint and immediately
// adds the address to the existing recipient suppression list. Provider event
// IDs are unique per provider so retries cannot create duplicate complaint rows.
func RecordComplaint(ctx context.Context, input ComplaintInput) error {
	input, err := normalizeComplaintInput(input)
	if err != nil {
		return err
	}

	return g.DB().Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		postfixMessageID := input.PostfixMessageID
		if postfixMessageID != "" {
			// An explicit Postfix ID must belong to this recipient.
			var matched struct {
				PostfixMessageID string `json:"postfix_message_id"`
			}
			err := tx.Model("mailstat_send_mails").
				Where("postfix_message_id", postfixMessageID).
				Where("recipient", input.Recipient).
				Fields("postfix_message_id").
				Scan(&matched)
			if err != nil {
				return fmt.Errorf("could not verify Postfix message ID: %w", err)
			}
			if matched.PostfixMessageID == "" {
				return fmt.Errorf("Postfix message ID does not match the supplied recipient")
			}
		} else {
			// When an upstream relay does not expose our Postfix queue ID, link
			// the complaint to the most recent successful send to this address.
			var latest struct {
				PostfixMessageID string `json:"postfix_message_id"`
			}
			err := tx.Model("mailstat_send_mails").
				Where("recipient", input.Recipient).
				Where("status", "sent").
				Where("dsn LIKE ?", "2.%").
				Fields("postfix_message_id").
				OrderDesc("log_time_millis").
				Limit(1).
				Scan(&latest)
			if err != nil && err != sql.ErrNoRows {
				return fmt.Errorf("could not find the latest successful delivery: %w", err)
			}
			postfixMessageID = latest.PostfixMessageID
		}

		inserted, err := tx.Model("mailstat_complaints").Data(g.Map{
			"provider_event_id":  input.EventID,
			"provider":           input.Provider,
			"postfix_message_id": postfixMessageID,
			"recipient":          input.Recipient,
			"log_time_millis":    time.Now().UnixMilli(),
		}).InsertIgnore()
		if err != nil {
			return fmt.Errorf("failed to store provider complaint: %w", err)
		}

		affected, err := inserted.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to verify complaint insert: %w", err)
		}
		if affected == 0 {
			var existing struct {
				Recipient string `json:"recipient"`
			}
			err = tx.Model("mailstat_complaints").
				Where("provider", input.Provider).
				Where("provider_event_id", input.EventID).
				Fields("recipient").
				Scan(&existing)
			if err != nil {
				return fmt.Errorf("failed to verify duplicate provider event: %w", err)
			}
			if !strings.EqualFold(existing.Recipient, input.Recipient) {
				return fmt.Errorf("provider event ID has already been used for a different recipient")
			}
		}

		// The existing campaign task creator excludes every abnormal recipient
		// with count >= 3. A complaint is a definitive signal, so suppress at once.
		description := "Abuse complaint received from " + input.Provider
		_, err = tx.Model("abnormal_recipient").Data(g.Map{
			"recipient":   input.Recipient,
			"count":       3,
			"add_type":    4,
			"description": description,
			"create_time": time.Now().Unix(),
		}).OnConflict("recipient").OnDuplicate(g.Map{
			"count":       gdb.Raw("GREATEST(abnormal_recipient.count, excluded.count)"),
			"add_type":    4,
			"description": description,
		}).Save()
		if err != nil {
			return fmt.Errorf("failed to add recipient to suppression list: %w", err)
		}

		g.Log().Infof(ctx, "Recorded email abuse complaint from %s for %s", input.Provider, input.Recipient)
		return nil
	})
}
