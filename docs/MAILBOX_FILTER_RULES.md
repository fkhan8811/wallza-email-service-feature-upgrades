# WallzaOS Email Service — mailbox filtering

BillionMail already installs Dovecot Pigeonhole and runs Sieve rules during
LMTP delivery. This upgrade exposes the existing ManageSieve protocol to
Roundcube so each mailbox owner can create and manage server-side filters.

## What is enabled

- Dovecot ManageSieve on the standard internal port 4190.
- Roundcube's built-in `managesieve` plugin.
- Roundcube connects to `tls://dovecot:4190`, which negotiates STARTTLS.
- Sieve scripts remain in each mailbox user's existing Dovecot script storage
  (`file:~/sieve;active=~/.dovecot.sieve`).
- The existing LMTP Sieve plugin remains enabled, so rules are applied when
  mail is delivered.

## Network boundary

Port 4190 is not added to the host's Docker port mappings. Only services on
BillionMail's private Docker network can reach the ManageSieve listener. Do
not publish this port directly to the public internet.

## User experience

Mailbox users manage filters inside Roundcube Webmail. This reuses the
existing mail client; it does not add another WallzaOS sidebar page or another
mail server.

## Validation

Before production use, rebuild/recreate the Dovecot and Roundcube services
from this fork and verify:

1. Dovecot starts with `sieve` in its enabled protocol list.
2. Roundcube loads the ManageSieve plugin and connects with STARTTLS.
3. A mailbox user can create, activate, edit and remove a simple rule.
4. A test message is filed into the expected folder by LMTP.
5. IMAP/SMTP and normal delivery continue to work.

Existing Sieve script storage is not migrated or overwritten by this change.
