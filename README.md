# Referenzzinssatz

A small web service that emails subscribers when the Swiss mortgage reference interest rate
(*hypothekarischer Referenzzinssatz*) changes.

Once a day it loads the [BWO's published table](https://www.bwo.admin.ch/de/entwicklung-referenzzinssatz-und-durchschnittszinssatz)
in headless Chromium and stores any new entries. If the newest rate is different afterwards, every
confirmed subscriber gets an email. Sign-up uses double opt-in and is protected by reCAPTCHA Enterprise.

## Running it

```sh
cp conf.example.json conf.json   # then fill in your values
mkdir -p data && sudo chown 10001:10001 data
docker compose up -d
```

The container runs as uid `10001`, so the data directory must be writable by that user. If you are
upgrading from an image that ran as root, `chown` the existing directory the same way.

### docker-compose.yml

```yaml
services:
  app:
    container_name: referenzzinssatz
    image: kalinkasolutions/referenzzinssatz:latest
    ports:
      - "8080:8000"
    volumes:
      - ./conf.json:/app/conf.json:ro
      - ./data:/app/data
    restart: always
```

## Configuration

`conf.json` holds credentials. It is ignored by git; never commit it.

| Key | Meaning |
| --- | --- |
| `Domain`, `Ssl` | Public host name used in the links inside emails, e.g. `referenzzinssatz.example.ch` with `Ssl: true`. It must be reachable from the internet. |
| `Port` | Port the app listens on inside the container. |
| `DatabasePath`, `DatabaseName` | Location of the SQLite database. |
| `SMTP_*` | Outgoing mail server. `SMTP_Username` is also the sender address and the contact address shown to users. |
| `ReCaptchaSiteKey`, `RecaptchaGoogleCloudApiKey`, `ReCaptchaProjectID` | A score-based [reCAPTCHA Enterprise](https://cloud.google.com/recaptcha) key and the Google Cloud project it belongs to. The API key needs access to the reCAPTCHA Enterprise API. |
| `RecaptchaMinScore` | Minimum score (0–1) required to subscribe; `0.7` is a reasonable start. |
| `TrustedProxies` | Addresses of reverse proxies whose `X-Forwarded-For` header is trusted. |
| `ReferenzZinssatzUrl` | Optional. Defaults to the BWO page above. |
| `Debug` | Enables debug-level logging and gin's debug mode. |
| `Loki` | Optional log shipping to [Grafana Loki](https://grafana.com/oss/loki/), see below. |

## Development

Requires Go and, for the scraper tests, Chrome or Chromium on the `PATH`. Without a browser those
tests are skipped.

```sh
go test ./...
go run . -configPath ./conf.json
```

Templates and static files live in `web/` and are embedded into the binary.

## Logging

Logs are written with `log/slog` to stdout. Warnings and errors are also kept in the `Logs` table
for 90 days, with their fields as JSON in the `Attributes` column. Subscribers appear in logs by id,
never by mail address.

With `Loki.Url` set, every log line from Info up (Debug up with `Debug: true`) is also pushed to
Loki as JSON, every two seconds. Lines are dropped rather than queued when Loki is unreachable.

| `Loki` key | Meaning |
| --- | --- |
| `Url` | Push endpoint, e.g. `https://loki.example.ch/loki/api/v1/push`. Empty turns shipping off. |
| `Username`, `Password` | Basic auth. For Grafana Cloud: the Loki user id and an access policy token. |
| `TenantId` | Sent as `X-Scope-OrgID`, for multi-tenant Loki. |
| `Labels` | Extra stream labels. `service_name` is `referenzzinssatz` unless set here. |

In Grafana, `{service_name="referenzzinssatz"} | json` shows the fields, e.g.
`| json | msg="HTTP request" | status >= 400`.
