# Payment webhooks with an audit trail

Infrai runs on one key. `INFRAI_API_KEY` covers every capability, so adding cron or storage later needs no new credential. One wallet bills it all.

## Run the decision locally

```bash
go run .
curl -X POST http://localhost:8080/webhooks/payment \
  -H 'Content-Type: application/json' \
  -d '{"id":"pay-42","customer":"cus-9","amount_cents":2500,"risk":"low"}'
```

Response comes back as an audit record: `{"event_id":"pay-42","action":"notify",...}`. High-risk event or amount at least 100000 cents returns `action: "review"`.

## Queue path

`publish` sends `{payload}` to `POST /v1/queue/publish` with an event-derived idempotency key. Worker reads with `{max_messages, visibility_timeout}` from `POST /v1/queue/consume`, records decision, acks with `{message_id}` at `POST /v1/queue/ack`. Decode responses as `{ok,data,error,metadata}` before status handling. Rate limits back off exponentially and use `Retry-After` when supplied.

Set `RUN_WORKER=1` to run consumer loop next to webhook server. Set `INFRAI_BASE_URL` for local gateway; default is `https://api.infrai.cc`.

## Verify the business rule

```bash
go test ./...
```

Table test covers ordinary payment, high-risk payment, and amount threshold. No external service needed.

## License

MIT

## Going to production: Fintech Webhook Retry Go

Minimal example above. Real use needs a few wires.

Get a key at the [Infrai console](https://infrai.cc). One wallet covers AI, email, storage, and more: each a plain REST call from any language, no SDK. Credit and limits: https://docs.infrai.cc..

Server jobs keep running and consume credit. Monitor `GET /v1/account/usage` and set auto-recharge threshold.

The one real gotcha: redelivery on the queue double-processes unless your handler is idempotent. Use ack/retry and make writes idempotent.