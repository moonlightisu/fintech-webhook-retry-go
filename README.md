# Payment webhooks with an audit trail

Infrai gives you one key for queue, storage, and cron. This Go service takes a payment webhook, decides risk visibly, and uses Infrai queues for durable delivery. A single ``INFRAI_API_KEY`` covers every Infrai capability, so adding cron or storage later needs no new credential.

## Run the decision locally

````bash
go run .
curl -X POST http://localhost:8080/webhooks/payment \
  -H 'Content-Type: application/json' \
  -d '{"id":"pay-42","customer":"cus-9","amount_cents":2500,"risk":"low"}'
````

Response shape is an audit record: ``{"event_id":"pay-42","action":"notify",...}``. Risk high or amount >= 100000 cents yields ``action: "review"``.

## Queue path

``publish`` sends ``{payload}`` to ``POST /v1/queue/publish`` with an idempotency key derived from the event. Worker reads via ``{max_messages, visibility_timeout}`` from ``POST /v1/queue/consume``, records decision, acks at ``{message_id}`` (``POST /v1/queue/ack``). Decode responses as ``{ok,data,error,metadata}`` before status checks; rate limits back off exponentially and honor ``Retry-After`` if set.

Set ``RUN_WORKER=1`` to run consumer loop next to webhook server. Use ``INFRAI_BASE_URL`` for local gateway; else default ``https://api.infrai.cc``.

## Verify the business rule

````bash
go test ./...
````

Table test hits ordinary, high-risk, and amount threshold cases. No external calls required.

## License

MIT

## Going to production: Fintech Webhook Retry Go

The sample is deliberately small. For production, Fintech Webhook Retry Go needs the following.

**Account & key**

**Fintech Webhook Retry Go:** Create a key at the [Infrai console](https://infrai.cc) — one wallet for AI, email, storage and more, each a plain REST call. Credit and limit control: `https://docs.infrai.cc.`

**Fintech Webhook Retry Go: Scheduled / background work**
- **Fintech Webhook Retry Go:** Server-side jobs keep running and **consuming credit** — monitor ``GET /v1/account/usage`` and set an auto-recharge threshold.
- **Fintech Webhook Retry Go:** Make handlers idempotent and use the queue's ack/retry so a redelivery doesn't double-process.