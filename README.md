# Fernmail

Fernmail is a self-hosted shared email application derived from LibreDesk. This fork keeps email receiving and sending, threading, attachments, rich replies, drafts, search, statuses, snoozing, views, private notes, and mentions.

The contacts/CRM, assignment, priorities, automation, macros, SLAs, business hours, surveys, reporting, public help center, website chat, AI assistants, and copilot, tags, and the old notification center have been removed. The app opens directly into the mailbox; mailbox settings are available from the settings icon beside the user name.

Email runs over IMAP and SMTP, including Google/Microsoft OAuth. This is a conversation-based shared mailbox, not a full bidirectional IMAP folder client.

Unread mail has a green edge and contributes to the browser tab count. Optional browser notifications show the sender and a message preview while Fernmail is open.

The new binary, module, and container image are named Fernmail. Existing `LIBREDESK_` configuration variables, database identifiers, and container names remain compatible with earlier installations. Custom site names and favicons are preserved when upgrading.

See [the mail-only upgrade notes](docs/mail-only.md) before upgrading an existing database.

## Installation

### Docker

The latest image is available in GitHub Container Registry at [`ghcr.io/jakedolan443/fernmail:latest`](https://github.com/jakedolan443/fernmail/pkgs/container/fernmail).

```shell
# Download the compose file and sample config file in the current directory.
curl -LO https://github.com/jakedolan443/fernmail/raw/main/docker-compose.yml
curl -LO https://github.com/jakedolan443/fernmail/raw/main/config.sample.toml

# Copy the config.sample.toml to config.toml and edit it as needed.
cp config.sample.toml config.toml

# Run the services in the background.
docker compose up -d

# Setting System user password.
docker exec -it libredesk_app ./fernmail --set-system-user-password
```

Go to `http://localhost:9000` and login with username `System` and the password you set using the `--set-system-user-password` command.

See [installation docs](https://docs.libredesk.io/getting-started/installation)

__________________

### Binary
- Download the [latest release](https://github.com/jakedolan443/fernmail/releases) and extract the `fernmail` binary.
- Edit config.toml as needed.
- `./fernmail --install` to setup the Postgres DB.
- Run `./fernmail --set-system-user-password` to set the password for the System user.
- Run `./fernmail` and visit `http://localhost:9000` and login with email `System` and the password you set using the --set-system-user-password command.

See [installation docs](https://docs.libredesk.io/getting-started/installation)
__________________

## Developers

For local development and setup, refer to the [developer setup](https://docs.libredesk.io/contributing/developer-setup).

The backend is written in Go and the frontend is Vue.js 3 with Shadcn UI.

### Multiple mailboxes and access

Add each email address in **Admin → Inboxes** with its own IMAP/SMTP connection. Each permitted address appears as a separate sidebar inbox; **All** combines mail from the addresses the signed-in user can access.

Open an inbox's settings and use **Inbox access** to choose **All users** or **Selected users and roles**. A direct user grant or membership in any selected role provides access. With no selections, a restricted inbox is visible only to administrators. Administrators always retain access; existing role permissions continue to control actions such as replying.

Access restrictions apply on the server to conversation reads and writes, lists, saved Views, search, drafts, counts, attachments and live message notifications. Existing inboxes remain unrestricted after upgrading. Upgrade existing databases before starting this build (`--upgrade`); fresh installations include the access schema automatically.
