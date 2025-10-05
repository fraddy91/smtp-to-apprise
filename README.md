[![Go Report Card](https://goreportcard.com/badge/github.com/fraddy91/smtp-to-apprise)](https://goreportcard.com/report/github.com/fraddy91/smtp-to-apprise)
[![GitHub Release](https://img.shields.io/github/v/release/fraddy91/smtp-to-apprise?logo=github)](https://github.com/fraddy91/smtp-to-apprise)
[![Docker Image Version](https://img.shields.io/docker/v/fraddy/smtp-to-apprise?logo=docker)](https://hub.docker.com/r/fraddy/smtp-to-apprise/tags)
[![Docker Image Size](https://img.shields.io/docker/image-size/fraddy/smtp-to-apprise?logo=docker)](https://hub.docker.com/r/fraddy/smtp-to-apprise/tags)

# Smtp-to-apprise overview
A minimal SMTP bridge that receives emails, selects the requested MIME part, and forwards it to Apprise destinations using configurable tags and keys. Written on Go memory footprint is as little as 3mb. It’s built for auditability, abuse resistance, and operational clarity: explicit SMTP auth, async dispatch with retries, and an admin GUI to manage mappings.
<br>It was inspired by Authelia's lack of notification providers, so the available mail provider was taken as the base to have a notification freedom, so you may redirect notifications to any/multiple destinations you want. This service can be used as notification bridge for any SMTP notifier. It can separate multipart/MIME messages by parts and forwatd it to the Apprise endpoind according to configuration (currently text/plain, text/htmp and raw message can be chosen, accordint to Authelia's standard notification template). So you can easily configure forwarding raw message to your email, text message to any messenger.

### Features
- SMTP auth (PLAIN): Require ADMIN_USER and ADMIN_PASS to send.
- MIME routing: Choose text/plain or text/html parts per record.
- Apprise dispatch: Forward to AppriseURL/Key with optional tags.
- Async queue with retry: Non-blocking enqueue, exponential backoff, bounded queue with drop-on-full.
- Admin GUI: Manage records, search, sort, and modify, dark/light theme, automatically fits system setting.
- Structured logs (optional): Clear success/error paths for audit.

### Architecture
- Backend: Stores records mapping Email + MimeType -> Key + Tags.
- SMTP server: Authenticated relay that extracts MIME parts and passes jobs to the dispatcher.
- Dispatcher: Background worker(s) that POST payloads to Apprise with retry and backoff.
- Admin GUI: HTTP server for CRUD on records and operational feedback.
flowchart LR
    <br>SMTP[SMTP Client] -->|AUTH + DATA| Bridge[SMTP-to-Apprise]
    <br>Bridge -->|extract MIME| Dispatcher[Async Dispatcher]
    <br>Dispatcher -->|POST w/ retry| Apprise[Apprise Services]
    <br>Admin[Admin GUI] -->|CRUD| BackendDB[(DB)]
    <br>Bridge --> BackendDB

## Quick start
### Service setup
```yaml
services:
  smtp-proxy:
    image: fraddy/smtp-to-apprise
    ports:
      - "8231:8080" # GUI
    environment:
      - ADMIN_USER=admin
      - ADMIN_PASS=strongpassword
      - GUI_ENABLED=true
      - LISTEN_SMTP=25
      - LISTEN_HTTP=8080
      - STORE_FILE=records.db
      - APPRISE_URL=http://apprise:8000/notify
    volumes:
      - ./data:/app/data/
    depends_on:
      - apprise
    network:
      - shim_network # Should share it with Apprise and Authelia
```

### Service configure
Open http://server.local:8231
</br> Enter preferred configuration
</br>![Example configuration 1.](/assets/gui-example.png)

### Authelia configure
```yaml
notifier:
  smtp:
    address: smtp://shim-proxy:25
    username: 'admin'
    password: 'strongpassword'
    sender: "Authelia <authelia@example.com>"
    disable_require_tls: true # Should be set to true as shim doesn't support tls
    disable_starttls: false
    disable_html_emails: false
```

### Apprise
```yaml
alice,bob,tg=tgram://<API Key>/<Chat ID>/?overflow=split&format=text
alice,mail=mailto://alice:password@example.com
bob,mail=mailto://bob:password@example.com
admin=alice,carl,den
```

#### And here you ready to go

## Configuration
### Docker Environment
- ADMIN_USER / ADMIN_PASS: SMTP PLAIN credentials required to send.
- APPRISE_URL: Base URL of Apprise (it's strongly recommended place container in common network with notifier+apprise, not exposing port) e.g., http://apprise:8000/notify.
- LISTEN_SMTP: Port the SMTP server listens on (e.g., 2525).
- LISTEN_HTTP: Port for the admin GUI (e.g., 8080).
- GUI_ENABLED: true to run the admin GUI.
- StoreFile: Filename under data/ for your SQLite store (from LoadConfig()).

### Authelia
```yaml
notifier:
  smtp:
    address: smtp://shim-proxy:2525 # This should be targeted to container's port
    timeout: '5s'
    username: 'test' # ADMIN_USER value
    password: 'password' # ADMIN_PASS value
    sender: "Authelia <authelia@example.com>" # Is required by Authelia, though is dropped anyway
    identifier: 'localhost' # Generally isn't used
    subject: "[Authelia] {title}" # This will be treated as subject for Apprise
    disable_require_tls: true # Should be set to true as shim doesn't support tls
    disable_starttls: false
    disable_html_emails: false
```

### Admin GUI
Gui contains input form with:
- email - Authelia's destination email
- key - Apprise's config key
- tags - this will be forwarded to Apprise as is
- mime-type - type to be forwarded, if message is multipart/mime (as Authelia does it) one can choose desired part, e.g. Plain text (text/plain) will be extracted and sent to Apprise

## Build
### Quick start
Binary
- Build: go build -o smtp-to-apprise ./cmd
- Run: Place the binary beside your data/ directory or let the app create it.

</br>export ADMIN_USER="admin"
</br>export ADMIN_PASS="secret"
</br>export APPRISE_URL="http://apprise:8000/notify"
</br>export LISTEN_SMTP="2525"
### Optional: GUI
export GUI_ENABLED="true"
</br>export LISTEN_HTTP="8080"

## Usage
### Add a record in the admin GUI
- Email: The recipient address that will be matched (e.g., alerts@example.com).
- Mime type: raw, text/plain or text/html to select which part to forward.
- Key: The Apprise notification key segment appended to APPRISE_URL.
- Tags: Comma/space separated list (e.g., admin,friends mail).
### Example record:
<br>Record 1
- Email: alerts@example.com
- Mime type: text/html
- Key: apprise
- Tags: admin telegram,children discord

<br>Record 2
- Email: alerts@example.com
- Mime type: Raw
- Key: apprise
- Tags: admin mail,friends mail

<br>Record 3
- Email: server-events@example.com
- Mime type: Raw
- Key: apprise
- Tags: admin

### Send an email
Use any SMTP client authenticated with ADMIN_USER/ADMIN_PASS, sending to the mapped Email. The bridge will:
- Extract: The requested MIME part from the message.
- Format: Set payload title from Subject, body from the part, tag from record tags, format inferred (text for text/plain, html otherwise).
- Enqueue: Push a dispatch job to the bounded queue.
- Retry: Worker posts to Apprise with exponential backoff; drops if queue is full.

## Additional information

### Screenshots

![GUI mobile white theme.](/assets/white-mobile-gui-example.png)
![GUI mobile black theme.](/assets/black-mobile-gui-example.png)
![GUI mobile search.](/assets/gui-monile-search-example.png)
![Telegram Authelia notification example.](/assets/notification-example.png)

#### Learn more about Apprise tags 
<a href=https://github.com/caronc/apprise/wiki/config>here</a> (https://github.com/caronc/apprise/wiki/config)
</br><a href=https://github.com/caronc/apprise/wiki/config_text>here</a> (https://github.com/caronc/apprise/wiki/config_text)
</br> <a href=https://github.com/caronc/apprise/wiki/config_text#text-based-apprise-configuration>and here</a> (https://github.com/caronc/apprise/wiki/config_text#text-based-apprise-configuration)

#### Learn more about Authelia's notification configuration 
<a href=https://www.authelia.com/configuration/notifications/smtp/>here</a> (https://www.authelia.com/configuration/notifications/smtp/)