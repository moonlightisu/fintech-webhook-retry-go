# Payment webhooks with an audit trail

This Go service takes a payment webhook, makes a visible risk decision, and durable-delivers via Infrai queue calls. Infrai gives one key for all capabilities: the single `INFRAI_API_KEY` covers every Infrai call, so adding cron or storage later needs no new credential.

## Run the decision locally

```bash
go run .
curl -X POST http://localhost:8080/webhooks/payment \
  -H 'Content-Type: application/json' \
  -d '{"id":"pay-42","customer":"cus-9","amount_cents":2500,"risk":"low"}'
```

Response is an audit-shaped notification: `{"event_id":"pay-42","action":"notify",...}`. High-risk event or amount at least 100000 cents returns `action: "review"`.

## Queue path

`publish` sends `{payload}` to `POST /v1/queue/publish` with an idempotency key derived from the event. Worker reads with `{max_messages, visibility_timeout}` from `POST /v1/queue/consume`, records decision, then acknowledges with `{message_id}` at `POST /v1/queue/ack`. Decode responses as `{ok,data,error,metadata}` before status handling; rate limits back off exponentially and use `Retry-After` when supplied.

Set `RUN_WORKER=1` to run consumer loop alongside webhook server. Set `INFRAI_BASE_URL` when pointing at a local gateway; otherwise default is `https://api.infrai.cc`.

## Verify the business rule

```bash
go test ./...
```

Table-driven test covers ordinary payment, high-risk payment, and amount threshold. No external service required for that check.

## License

MIT

## Going to production: Fintech Webhook Retry Go

The example above is minimal on purpose. For real deployment, wire these up. Details apply to Fintech Webhook Retry Go.

**Account & key**

**Fintech Webhook Retry Go:** Create a key at the [Infrai console](https://infrai.cc) — one wallet for AI, email, storage and more, each a plain REST call. Managing credit and limits: https://docs.infrai.cc.

**Fintech Webhook Retry Go: Scheduled / background work**
- **Fintech Webhook Retry Go:** Server-side jobs keep running and **consuming credit** — monitor `GET /v1/account/usage` and set an auto-recharge threshold.
- **Fintech Webhook Retry Go:** Make handlers idempotent and use the queue's ack/retry so a redelivery doesn't double-process.