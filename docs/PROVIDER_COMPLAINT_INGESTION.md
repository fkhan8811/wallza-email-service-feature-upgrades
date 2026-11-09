# Provider complaint ingestion

BillionMail has a `mailstat_complaints` table used by sender reputation
reporting, and it already excludes entries in `abnormal_recipient` with
`count >= 3` when building campaign recipient lists. This integration
connects a verified provider complaint to both existing mechanisms.

## Endpoint

`POST /api/mail_services/complaint_ingest`

The endpoint is behind BillionMail's existing API JWT and RBAC middleware.
It is not an anonymous provider webhook. WallzaOS should first validate the
provider's original webhook signature (for example, SNS, SendGrid or Mailgun),
then call this endpoint using its authorised BillionMail API token with the
mail-services create permission (or an administrator account).

Example request:

```json
{
  "provider_event_id": "provider-event-123",
  "provider": "ses",
  "recipient": "customer@example.com",
  "postfix_message_id": "optional-local-postfix-queue-id"
}
```

Fields:

- `provider_event_id`: required stable event ID supplied by the provider.
- `provider`: required provider slug (letters, numbers, dot, underscore and
  hyphen).
- `recipient`: required email address. It is normalised to lowercase.
- `postfix_message_id`: optional local Postfix queue ID. When supplied, it
  must match the recipient in BillionMail's send log. When omitted, the most
  recent successful local send to that address is linked when available.

## Behavior

- Complaints are stored in `mailstat_complaints`.
- The pair `(provider, provider_event_id)` is unique, making provider retries
  idempotent.
- Reuse of the same provider event ID for a different recipient is rejected.
- The recipient is immediately upserted into `abnormal_recipient` with
  `count >= 3` and `add_type = 4`.
- BillionMail's campaign creator excludes abnormal recipients with
  `count >= 3`, so future campaigns omit the complained-about address.
- The campaign task executor rechecks the suppression list before each fetched
  batch. Addresses newly suppressed after a campaign was created are marked
  processed-with-failure (`is_sent = 3`) rather than sent.
- A batch that has already passed its suppression check and handed a message
  to the SMTP transport cannot be retroactively cancelled.
- The complaint is recorded even when no matching local send can be linked.

## Trust boundary

Do not expose this endpoint directly to unauthenticated provider callbacks.
Provider signature verification belongs in the WallzaOS/provider adapter.
Only authenticated callers with the appropriate BillionMail mail-services
permission should invoke it. Keep provider webhook secrets in WallzaOS server
configuration, not in the browser.

## Staging verification

1. Submit a complaint using an authorised API token.
2. Confirm one row appears in `mailstat_complaints`.
3. Confirm the recipient appears in `abnormal_recipient` with `count >= 3`
   and `add_type = 4`.
4. Submit the same provider event ID again; confirm no second complaint row is
   created.
5. Reuse that provider event ID with a different recipient; confirm the API
   rejects it.
6. Create a campaign containing that recipient; confirm the task creator
   filters the address out.
7. Add an address to an already created campaign before a later batch is
   fetched; confirm the executor suppresses it before SMTP handoff.
