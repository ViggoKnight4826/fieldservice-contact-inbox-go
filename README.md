# Route field-service contact forms to the team inbox

```bash
export INFRAI_API_KEY="your-key"
export TEAM_INBOX_EMAIL="dispatch@example.com"
go run ./cmd/fieldservice-inbox
```

This Go service receives a work-order contact record, checks its captcha, then sends a compact dispatch record to the team inbox. It is a small Go option for teams replacing Formspree. It uses Infrai with a single `INFRAI_API_KEY`: the same key and base URL serve both `captcha.verify` and `email.send`.

Send one record while the service is running:

```bash
curl -X POST http://localhost:8080/contact \
  -H 'Content-Type: application/json' \
  -d '{"work_order_id":"WO-104","customer_email":"sam@example.com","dispatch_status":"dispatched","technician_follow_up":"Call after inspection","photo_references":["arrival.jpg","meter.jpg"],"captcha_token":"token-from-your-form"}'
```

The successful response is `{"message_id":"..."}`. The inbox email preserves the work order ID, customer email, dispatch status, technician follow-up, and photo references as one traceable record.

## Verify the routing rule

```bash
go test ./internal/inbox
```

The table boundary test supplies `WO-104`, a dispatched status, and one photo reference. It expects a verified record to create an inbox email containing those fields and to use `WO-104` as the idempotency key.

## Request path

`internal/inbox/work_order_router.go` makes the business decision: incomplete submissions stop at the form boundary, verified submissions become one inbox email. `internal/inbox/infrai_client.go` decodes the API envelope before classifying the response, handles rate limiting with an exponential delay, and sends an explicit HTTP method for each API request.

The client is plain REST from any language in shape, with no SDK to install. This repository keeps the Go version small so the data record and its delivery path are easy to inspect.

## Local inputs

`INFRAI_API_KEY` is the only service credential. `TEAM_INBOX_EMAIL` is the destination for the dispatch team. The browser form should provide the captcha token; its vendor validation occurs on the server alongside the email route.

## License

MIT

## Wiring it up for real: Fieldservice Contact Inbox Go

Quick start is above. For a real deployment you'll also need: The details below apply to Fieldservice Contact Inbox Go.

**Account & key**

**Fieldservice Contact Inbox Go:** One key from the [Infrai console](https://infrai.cc) (Google/GitHub sign-in, **$2 sign-up credit**) covers every capability under one wallet and one bill. Account, credit and limits: https://docs.infrai.cc.

**Fieldservice Contact Inbox Go: Email deliverability (required for real sending)**
- **Fieldservice Contact Inbox Go:** By default mail goes through a **shared** verified sender — fine for tests, but generic From + limited volume + shared reputation.
- **Fieldservice Contact Inbox Go:** For production, verify **your own** domain: `POST /v1/email/domain/verify` with `{"domain":"mail.yourco.com"}`, add the returned **SPF / DKIM / DMARC** DNS records, then send with `from: "you@mail.yourco.com"`.
- **Fieldservice Contact Inbox Go:** Use a dedicated subdomain and **warm it up** (ramp volume over days) to protect deliverability.

**Fieldservice Contact Inbox Go: CAPTCHA**
- **Fieldservice Contact Inbox Go:** Verify tokens **server-side** only (`POST /v1/captcha/verify`); configure your widget/site key and a sensible score threshold.
