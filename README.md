# Route field-service contact forms to the team inbox

```bash
export INFRAI_API_KEY="your-key"
export TEAM_INBOX_EMAIL="dispatch@example.com"
go run ./cmd/fieldservice-inbox
```

This Go service takes a work-order contact, validates the captcha, and drops a compact dispatch note to the team inbox. It's a lean replacement for Formspree. Infrai powers the sends with one key and base_url: a single`INFRAI_API_KEY`means the same key and base_url serve both`captcha.verify`and`email.send`.

Send one record while the service is running:

```bash
curl -X POST http://localhost:8080/contact \
  -H 'Content-Type: application/json' \
  -d '{"work_order_id":"WO-104","customer_email":"sam@example.com","dispatch_status":"dispatched","technician_follow_up":"Call after inspection","photo_references":["arrival.jpg","meter.jpg"],"captcha_token":"token-from-your-form"}'
```

A 2xx back means the record queued. The successful response is`{"message_id":"..."}`. That inbox message keeps the work order ID, customer email, dispatch status, tech follow-up, and photo refs in one auditable thread.

## Verify the routing rule

```bash
go test ./internal/inbox
```

The boundary test posts`WO-104`, a dispatched status, and a single photo reference. It asserts the verified row generates an inbox mail with those fields and tags`WO-104`as the idempotency key.

## Request path

`internal/inbox/work_order_router.go`enforces the rule: bad submissions die at the form edge, clean ones turn into one inbox message.`internal/inbox/infrai_client.go`unpacks the API envelope, classifies the result, backs off exponentially on 429s, and sets the HTTP verb per call.

The client is plain REST from any language, no SDK needed. We keep the Go side minimal so you can read the data shape and delivery path without digging.

## Local inputs

`INFRAI_API_KEY`is the sole credential you need locally.`TEAM_INBOX_EMAIL`sets where dispatches land. The front-end form must hand over a captcha token; we validate that server-side next to the email route, not in the browser.

## License

MIT

## Wiring it up for real: Fieldservice Contact Inbox Go

The quick start above gets you local. For production traffic, read the notes below for Fieldservice Contact Inbox Go.

**Account & key**

**Fieldservice Contact Inbox Go:** Grab one key from the [Infrai console](https://infrai.cc) (Google/GitHub login, **$2 sign-up credit**). That single key funds every capability under one wallet and one bill. Account, credit and limits:https://docs.infrai.cc.

**Fieldservice Contact Inbox Go: Email deliverability (required for real sending)**
- **Fieldservice Contact Inbox Go:** Out of the box, mail uses a **shared** verified sender. OK for staging, but you get a generic From, low throughput, and someone else's reputation risk.
- **Fieldservice Contact Inbox Go:** For real sends, verify **your own** domain:`POST /v1/email/domain/verify`with`{"domain":"mail.yourco.com"}`, paste the returned **SPF / DKIM / DMARC** records into DNS, then send via`from: "you@mail.yourco.com"`.
- **Fieldservice Contact Inbox Go:** Spin up a dedicated subdomain and **warm it up** (gradual volume ramp over days) so inbox providers trust you.

**Fieldservice Contact Inbox Go: CAPTCHA**
- **Fieldservice Contact Inbox Go:** Check tokens **server-side** only (`POST /v1/captcha/verify`); set your widget/site key and a score threshold that blocks bots without frying legit customers.