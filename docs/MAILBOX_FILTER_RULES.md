# WallzaOS Email Service — mailbox filtering

BillionMail already installs Dovecot Pigeonhole and runs Sieve rules during
LMTP delivery. This upgrade exposes the existing ManageSieve protocol to
Roundcube so each mailbox owner can create and manage server-side filters.

## What is enabled

The Dovecot image now installs Debian's `dovecot-managesieved` package in addition to the existing `dovecot-sieve` package. The compose service builds this fork's Dovecot image as `wallza/billionmail-dovecot:1.6`.

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

## Rebuild and deploy

After updating the fork, rebuild the Dovecot image and recreate Dovecot and Roundcube:

```bash
docker compose build dovecot-billionmail
docker compose up -d --no-deps dovecot-billionmail webmail-billionmail
```

This does not publish port 4190 on the host. Roundcube reaches it over the private Docker network.

## Validation

Before production use, rebuild/recreate the Dovecot and Roundcube services
from this fork and verify:

1. Dovecot starts with `sieve` in its enabled protocol list.
2. Roundcube loads the ManageSieve plugin and connects with STARTTLS.
3. A mailbox user can create, activate, edit and remove a simple rule.
4. A test message is filed into the expected folder by LMTP.
5. IMAP/SMTP and normal delivery continue to work.

Existing Sieve script storage is not migrated or overwritten by this change.


## Shared mailboxes

Shared mailbox access is implemented with Dovecot's IMAP ACL plugin and a
separate `Shared/<mailbox-user>/` namespace. Roundcube exposes the ACL
controls to mailbox owners; the Dovecot ACL files remain the source of truth
for read, write, create and delete rights. No mailbox password is shared and
no mailbox contents are copied.

To share a mailbox, the owner grants the intended mailbox identity access
through Roundcube's folder permissions. The recipient opens the shared
mailbox under the `Shared` namespace. Revoking the ACL removes access through
IMAP; existing sessions should be disconnected before assuming revocation is
instantaneous.

The global shared-mailbox index uses Dovecot's file dictionary at
`/var/vmail/shared-mailboxes.db`. This path is on BillionMail's existing
persistent `./vmail-data:/var/vmail` volume and is writable by the `vmail:mail`
identity created by the Dovecot entrypoint. The feature is deliberately
server-side: do not publish Dovecot's internal listener or expose ACL files
through a web/static volume.

Validate in staging with two real mailbox accounts: grant read-only access,
confirm the recipient can read but cannot change messages, upgrade to
read/write, then revoke access and confirm the recipient can no longer open
the shared namespace. The automated configuration check does not replace
this two-account runtime check.
