# WallzaOS webhook integration

BillionMail can forward email activity to a WallzaOS backend using a durable,
signed webhook queue. The mail transport, Postfix, Dovecot and Roundcube
configuration are unchanged.

## Configuration

Set these variables in the BillionMail deployment `.env` file and recreate
the `core-billionmail` service:

- `WALLZAOS_WEBHOOK_URL`: HTTPS endpoint on the WallzaOS backend.
- `WALLZAOS_WEBHOOK_SECRET`: random secret of at least 32 characters. Keep it
  private and use the same value on the receiver.
- `WALLZAOS_WEBHOOK_ALLOW_LOCAL_HTTP`: optional; set to `true` only for
  local development with a localhost URL. Production endpoints must use HTTPS.

Example:

```dotenv
WALLZAOS_WEBHOOK_URL=https://api.example.com/integrations/billionmail/events
WALLZAOS_WEBHOOK_SECRET=replace-with-a-random-secret-of-at-least-32-characters
```

The variables are read by the BillionMail core container. Do not commit a real
secret into source control. When one setting is missing or invalid, new events
are still stored in the outbox and delivery resumes after the configuration is
corrected. When both settings are empty, event forwarding remains disabled.

## Events

The queue currently emits:

- `email.delivery.updated`: Postfix delivery status changes, including sent,
  bounced and deferred statuses.
- `email.opened`: a tracked campaign open.
- `email.clicked`: a tracked campaign link click. The event includes the link
  host, not the full URL or query string.

Each JSON body has this envelope:

```json
{
  "id": "stable-event-id",
  "type": "email.delivery.updated",
  "source": "billionmail",
  "created_at": "2026-11-09T12:00:00Z",
  "data": {}
}
```

## Signature verification

BillionMail sends:

- `X-WallzaOS-Event`: event type.
- `X-WallzaOS-Delivery`: event idempotency key.
- `X-WallzaOS-Signature`: `sha256=<hex>`.

The signature is HMAC-SHA256 of the exact raw request body using
`WALLZAOS_WEBHOOK_SECRET`. Receivers should compare signatures in constant
time and deduplicate by `X-WallzaOS-Delivery`.

## Retry behavior

Events are persisted in `bm_wallzaos_webhook_deliveries`. A background timer
claims pending rows, sends them with a short timeout and retries failures after
30 seconds, 2 minutes, 10 minutes, 30 minutes and 2 hours. Claims left in
processing for more than two minutes are recovered after a restart. After six
unsuccessful attempts an event is marked
`failed`; successful events are marked `delivered`. The event queue is not
deleted when an endpoint is unavailable.

A 2xx response is considered successful. Non-2xx responses, timeouts and
connection failures are retried. The Settings page provides a searchable
delivery history, payload inspection and manual retry for failed events.

## Retention

Terminal deliveries (`delivered` and `failed`) are pruned after 30 days by
default. Set `WALLZAOS_WEBHOOK_RETENTION_DAYS` to a value from 7 to 365 to
change this period. Invalid values fall back to 30 days. Pending and processing
deliveries are never removed by this cleanup. The cleanup runs once shortly
after startup and then daily.

## Deployment note

This is a server-to-server integration for a trusted WallzaOS endpoint. The
webhook secret must never be exposed to a WallzaOS browser client. Event data
contains recipient email addresses for matching against WallzaOS contacts and
must be handled as customer data.
