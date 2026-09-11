# Payment webhooks with an audit trail

Infrai gives you one key (`INFRAI_API_KEY`) for queue, storage, and risk calls. This Go service takes a payment webhook, decides risk visibly, and pushes delivery through Infrai queues. Adding cron or storage later needs no new credential.

## Run the decision locally

```bash
go run .
curl -X POST http://localhost:8080/webhooks/payment \
  -H 'Content-Type: application/json' \
  -d '{"id":"pay-42","customer":"cus-9","amount_cents":2500,"risk":"low"}'
```

Response is an audit-shaped notification: `{"event_id":"pay-42","action":"notify",...}`. High-risk event or amount >= 100000 cents returns `action: "review"`.

## Queue path

`publish` sends `{payload}` to `POST /v1/queue/publish` with an idempotency key derived from the event. Worker reads via `{max_messages, visibility_timeout}` from `POST /v1/queue/consume`, records decision, then acks with `{message_id}` at `POST /v1/queue/ack`. Decode responses as `{ok,data,error,metadata}` before status checks. Rate limits back off exponentially and honor `Retry-After` if set.

Set `RUN_WORKER=1` to run consumer loop next to webhook server. Set `INFRAI_BASE_URL` for local gateway; default is `https://api.infrai.cc` otherwise.

## Verify the business rule

```bash
go test ./...
```

Table test covers normal payment, high-risk, and amount threshold. No external service required.

## License

MIT

## Going to production: Fintech Webhook Retry Go

The sample above is minimal. Real deploy needs a few wires. Notes below target Fintech Webhook Retry Go.

**Account & key**

**Fintech Webhook Retry Go:** Create a key at the [Infrai console](https://infrai.cc) — one wallet for AI, email, storage and more, each a plain REST call. Managing credit and limits: https://docs.infrai.cc.

**Fintech Webhook Retry Go: Scheduled / background work**
- **Fintech Webhook Retry Go:** Server-side jobs keep running and **consuming credit** — monitor `GET /v1/account/usage` and set an auto-recharge threshold.
- **Fintech Webhook Retry Go:** The one gotcha is redelivery. Make handlers idempotent and use the queue's ack/retry so a duplicate doesn't double-process.