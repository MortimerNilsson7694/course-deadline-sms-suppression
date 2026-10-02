package delivery

import (
	"context"
	"testing"
	"time"
)

type recordingSender struct {
	calls int
}

func (s *recordingSender) SendBatch(context.Context, []BatchMessage, string) (string, error) {
	s.calls++
	return "message-42", nil
}

func TestDeadlineDecision(t *testing.T) {
	now := time.Date(2026, time.September, 18, 9, 0, 0, 0, time.UTC)
	tests := []struct {
		name      string
		optedOut  bool
		dueAt     time.Time
		want      Decision
		wantCalls int
	}{
		{name: "opted out learner is suppressed", optedOut: true, dueAt: now.Add(-time.Hour), want: Suppressed, wantCalls: 0},
		{name: "future deadline is not due", dueAt: now.Add(time.Hour), want: NotDue, wantCalls: 0},
		{name: "due learner is sent", dueAt: now.Add(-time.Minute), want: Sent, wantCalls: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sender := &recordingSender{}
			service := NewService(sender, func() time.Time { return now })
			if tt.optedOut {
				service.OptOut("+15550001111")
			}
			result, err := service.Deliver(context.Background(), Deadline{
				CourseID: "data-101", LearnerID: "learner-7", Phone: "+15550001111",
				CourseName: "Data Pipelines 101", DueAt: tt.dueAt,
			})
			if err != nil {
				t.Fatalf("Deliver() error = %v", err)
			}
			if result.Decision != tt.want {
				t.Fatalf("decision = %q, want %q", result.Decision, tt.want)
			}
			if sender.calls != tt.wantCalls {
				t.Fatalf("send calls = %d, want %d", sender.calls, tt.wantCalls)
			}
		})
	}
}
