package delivery

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type Deadline struct {
	CourseID   string    `json:"course_id"`
	LearnerID  string    `json:"learner_id"`
	Phone      string    `json:"phone"`
	CourseName string    `json:"course_name"`
	DueAt      time.Time `json:"due_at"`
}

type Decision string

const (
	Sent       Decision = "sent"
	Suppressed Decision = "suppressed"
	NotDue     Decision = "not_due"
)

type Result struct {
	CourseID  string   `json:"course_id"`
	LearnerID string   `json:"learner_id"`
	Decision  Decision `json:"decision"`
	MessageID string   `json:"message_id,omitempty"`
}

type Report struct {
	CourseID   string `json:"course_id"`
	Sent       int    `json:"sent"`
	Suppressed int    `json:"suppressed"`
	NotDue     int    `json:"not_due"`
}

type BatchMessage struct {
	To   string `json:"to"`
	Text string `json:"text"`
}

type Sender interface {
	SendBatch(context.Context, []BatchMessage, string) (string, error)
}

type Service struct {
	sender Sender
	now    func() time.Time

	mu          sync.RWMutex
	suppression map[string]struct{}
	reports     map[string]Report
}

func NewService(sender Sender, now func() time.Time) *Service {
	return &Service{
		sender:      sender,
		now:         now,
		suppression: make(map[string]struct{}),
		reports:     make(map[string]Report),
	}
}

func (s *Service) OptOut(phone string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.suppression[phone] = struct{}{}
}

func (s *Service) Deliver(ctx context.Context, deadline Deadline) (Result, error) {
	result := Result{CourseID: deadline.CourseID, LearnerID: deadline.LearnerID}
	if s.isSuppressed(deadline.Phone) {
		result.Decision = Suppressed
		s.record(result)
		return result, nil
	}
	if deadline.DueAt.After(s.now()) {
		result.Decision = NotDue
		s.record(result)
		return result, nil
	}

	text := fmt.Sprintf("Course deadline: %s is due. Open your course workspace for details.", deadline.CourseName)
	key := fmt.Sprintf("course:%s:learner:%s:due:%s", deadline.CourseID, deadline.LearnerID, deadline.DueAt.UTC().Format(time.RFC3339))
	messageID, err := s.sender.SendBatch(ctx, []BatchMessage{{To: deadline.Phone, Text: text}}, key)
	if err != nil {
		return Result{}, err
	}
	result.Decision = Sent
	result.MessageID = messageID
	s.record(result)
	return result, nil
}

func (s *Service) Report(courseID string) Report {
	s.mu.RLock()
	defer s.mu.RUnlock()
	report := s.reports[courseID]
	report.CourseID = courseID
	return report
}

func (s *Service) isSuppressed(phone string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, found := s.suppression[phone]
	return found
}

func (s *Service) record(result Result) {
	s.mu.Lock()
	defer s.mu.Unlock()
	report := s.reports[result.CourseID]
	report.CourseID = result.CourseID
	switch result.Decision {
	case Sent:
		report.Sent++
	case Suppressed:
		report.Suppressed++
	case NotDue:
		report.NotDue++
	}
	s.reports[result.CourseID] = report
}
