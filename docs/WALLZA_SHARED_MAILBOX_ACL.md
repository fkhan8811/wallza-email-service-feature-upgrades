# BillionMail shared mailbox permissions

This change adds shared mailbox visibility and access control to the existing BillionMail stack. It uses the Dovecot and Roundcube components already shipped with BillionMail; it does not add another IMAP server or a second mailbox database.

## Design

- **Permission enforcement:** Dovecot's `acl` and IMAP `imap_acl` plugins enforce mailbox-level rights.
- **Mailbox owner experience:** Roundcube's built-in `acl` plugin exposes folder permissions so a mailbox owner can grant or revoke access.
- **Shared mailbox discovery:** Dovecot exposes granted mailboxes under the `Shared/<mailbox-user>/` namespace.
- **Storage:** ACL files remain in each mailbox's `dovecot-acl` file. The shared mailbox dictionary is stored at `/var/vmail/shared-mailboxes.db`, on BillionMail's existing persistent `/var/vmail` volume.
- **Identity:** Users authenticate with their own mailbox credentials. The feature does not copy mailbox passwords or duplicate messages.

## BillionMail-specific path mapping

The existing Dovecot entrypoint generates SQL userdb settings with:

- Home: `/var/vmail/%d/%n`
- Mail location: `maildir:/var/vmail/%d/%n`
- Runtime user/group: `vmail:mail` (UID 150, GID 8)

The shared namespace therefore uses the target user's `%%h` home directory, with private index state for the logged-in user under `%h/shared/%%u`. Do not replace this with the generic `~/Maildir` example; it does not match BillionMail's existing layout.

## Grant and revoke

1. Sign in to Roundcube as the mailbox owner.
2. Open the folder's sharing/permissions controls (the ACL plugin).
3. Add the other mailbox identity and grant only the rights needed (for example, read-only before granting write/delete).
4. The invited mailbox user signs in with their own credentials and sees the granted folders beneath `Shared`.
5. Revoke the same ACL entry to remove access. Existing IMAP sessions may need to reconnect before the client reflects revocation.

Dovecot ACLs are the enforcement point. A folder that is merely listed in a client UI is not considered protected unless Dovecot rejects the corresponding IMAP operations.

## Security boundaries

- Never publish Dovecot's internal SQL or authentication sockets.
- Do not make `/var/vmail` or `dovecot-acl` files accessible over HTTP.
- Keep the shared-mailbox dictionary on the persistent mail volume; do not store it on an ephemeral container layer.
- Prefer read-only access and expand privileges only when required.
- This feature is mailbox sharing within one BillionMail instance. It is not the WallzaOS customer/tenant isolation boundary. Do not host unrelated WallzaOS organizations in a shared Mautic or email data plane without separate tenant isolation.

## Validation

The PR workflow:
- builds the existing Dovecot image;
- verifies the Dovecot ACL and IMAP ACL modules exist;
- parses the shared ACL configuration with `doveconf`;
- checks the configured mailbox home path matches the existing SQL userdb path;
- confirms the Roundcube ACL plugin is enabled;
- does not expose a new network listener.

A live two-mailbox staging test is still required: create two test mailbox accounts, grant read-only rights, prove writes fail, grant write rights, prove the allowed operation succeeds, revoke access, reconnect and prove access fails. Automated config validation is not a substitute for that runtime test.
