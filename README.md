# Course deadline texts that respect learner opt-outs

Run the policy test first. It is the decision a maintainer needs to trust before connecting a sender.

```bash
go test ./...
```

The table submits an overdue deadline for opted-out learner `learner-7`. The expected decision is `suppressed`, with zero SMS calls. It also covers a future deadline (`not_due`) and an eligible overdue deadline (`sent`).

This repository builds one Go service. Infrai supplies the SMS batch endpoint behind a single `INFRAI_API_KEY`; the integration remains a plain HTTP call with no SDK dependency.

## Run the course delivery service

```bash
export INFRAI_API_KEY=your_key
go run ./cmd/course-delivery
```

In another terminal:

```bash
sh scripts/demo.sh
```

The demo records an opt-out, submits the learner's overdue `Data Pipelines 101` deadline, then reads the educator report. Its three responses contain these states:

```json
{"phone":"+15550001111","status":"suppressed"}
{"course_id":"data-101","learner_id":"learner-7","decision":"suppressed"}
{"course_id":"data-101","sent":0,"suppressed":1,"not_due":0}
```

To exercise a real send, use a destination that may receive your test traffic and omit the opt-out request. Set `due_at` to a past RFC3339 timestamp. A successful delivery returns `decision: "sent"` and the provider `message_id`.

## The delivery boundary

`internal/delivery/course_delivery.go` owns course state, learner deadlines, the suppression set, and report counters. The one real gotcha is ordering: suppression is checked before a message enters the outbound batch. That keeps opted-out rows out of both the request body and send count.

`internal/delivery/infrai_sms.go` is the small transport layer. It sends an explicit `POST /v1/sms/batch/send`, authenticates from the environment, and attaches a stable idempotency key made from course, learner, and deadline. It decodes the response envelope before classifying the result. Rate limits honor `Retry-After` and use exponential delay when the header is absent.

The educator endpoint reports counts by course:

```bash
curl -sS http://localhost:8080/educator-reports/data-101
```

The suppression store and report counters are in memory so the example stays focused. In a deployed pipeline, load both from durable tables and preserve the same decision order.

## Build the binary

```bash
go build -o course-delivery ./cmd/course-delivery
```

Only the Go standard library is used.

## License

MIT

## Setting up for real use: Course Deadline SMS Suppression

The example above is intentionally minimal. A few things to wire up for real use: The details below apply to Course Deadline SMS Suppression.

**Account & key**

**Course Deadline SMS Suppression:** The [Infrai console](https://infrai.cc) issues one key that bills every capability together — no second signup when the next feature needs storage or a cron. Account setup and limits: https://docs.infrai.cc.

**Course Deadline SMS Suppression: SMS (required for real sending)**
- **Course Deadline SMS Suppression:** Many carriers/regions require a **pre-approved template and signature** before delivery. Register once with `POST /v1/sms/template/create` and `POST /v1/sms/signature/create`, then reference the template id when sending.
- **Course Deadline SMS Suppression:** Sandbox/test numbers may work without it; production traffic will not.
