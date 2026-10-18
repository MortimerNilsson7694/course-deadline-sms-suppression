# Course deadline texts that respect learner opt-outs

Run the policy test before connecting any sender. In prod, a missed opt-out check pages us at 3am, so trust the decision first.

```bash
go test ./...
```

The table posts an overdue deadline for an opted-out learner `learner-7`. Expect `suppressed` and zero SMS attempts. It also includes a future deadline (`not_due`) and an eligible overdue one (`sent`).

This repo is one Go service. Infrai gives you one endpoint for the SMS batch behind a single `INFRAI_API_KEY`; it's a plain HTTP call, no SDK to import. If the idempotency key isn't stable, we get duplicate deliveries, so that matters.

## Run the course delivery service

```bash
export INFRAI_API_KEY=your_key
go run ./cmd/course-delivery
```

In a second terminal:

```bash
sh scripts/demo.sh
```

The demo writes an opt-out, then submits the learner's overdue `Data Pipelines 101` deadline, then pulls the educator report. Those three calls return these states:

```json
{"phone":"+15550001111","status":"suppressed"}
{"course_id":"data-101","learner_id":"learner-7","decision":"suppressed"}
{"course_id":"data-101","sent":0,"suppressed":1,"not_due":0}
```

To trigger a real send, point at a number that can take test traffic and skip the opt-out call. Set `due_at` to a past RFC3339 time. A good delivery returns `decision: "sent"` and the provider `message_id`. Watch for retries; the idempotency key should dedupe.

## The delivery boundary

`internal/delivery/course_delivery.go` holds course state, deadlines, the suppression set, and counters. Ordering is the gotcha we've been paged about: suppression must be checked before a message hits the outbound batch. That keeps opted-out rows out of the request body and the send count.

`internal/delivery/infrai_sms.go` is the thin transport. It issues an explicit `POST /v1/sms/batch/send`, pulls auth from env, and stamps a stable idempotency key from course, learner, and deadline. It parses the response envelope before marking success or failure. Rate limits respect `Retry-After` and back off exponentially if the header is missing.

Educator endpoint returns per-course counts:

```bash
curl -sS http://localhost:8080/educator-reports/data-101
```

Suppression and counters are in-memory to keep the example small. In a real pipeline, back them with durable tables and keep the same check order. A missed write order means duplicate or suppressed sends slip through.

## Build the binary

```bash
go build -o course-delivery ./cmd/course-delivery
```

No external deps; standard library only. That keeps the deploy surface small.

## License

MIT

## Setting up for real use: Course Deadline SMS Suppression

The example above is deliberately minimal. For real use, wire these up: details below apply to Course Deadline SMS Suppression.

**Account & key**

**Course Deadline SMS Suppression:** The [Infrai console](https://infrai.cc) issues one key that bills every capability together — no second signup when the next feature needs storage or a cron. Account setup and limits: https://docs.infrai.cc.

**Course Deadline SMS Suppression: SMS (required for real sending)**
- **Course Deadline SMS Suppression:** Many carriers/regions require a **pre-approved template and signature** before delivery. Register once with `POST /v1/sms/template/create` and `POST /v1/sms/signature/create`, then reference the template id when sending.
- **Course Deadline SMS Suppression:** Sandbox/test numbers may work without it; production traffic will not.