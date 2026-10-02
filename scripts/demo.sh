#!/bin/sh
set -eu

curl -sS -X POST http://localhost:8080/opt-outs \
  -H 'Content-Type: application/json' \
  -d '{"phone":"+15550001111"}'

curl -sS -X POST http://localhost:8080/course-deliveries \
  -H 'Content-Type: application/json' \
  -d '{"course_id":"data-101","learner_id":"learner-7","phone":"+15550001111","course_name":"Data Pipelines 101","due_at":"2026-09-18T08:00:00Z"}'

curl -sS http://localhost:8080/educator-reports/data-101
