package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"example.com/course-deadline-sms/internal/delivery"
)

func main() {
	apiKey := os.Getenv("INFRAI_API_KEY")
	if apiKey == "" {
		log.Fatal("INFRAI_API_KEY is required")
	}
	service := delivery.NewService(&delivery.InfraiSMS{
		APIKey: apiKey,
		Client: &http.Client{Timeout: 15 * time.Second},
		Sleep:  delivery.SleepContext,
	}, time.Now)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /opt-outs", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Phone string `json:"phone"`
		}
		if err := decodeJSON(r, &input); err != nil || strings.TrimSpace(input.Phone) == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "phone is required"})
			return
		}
		service.OptOut(input.Phone)
		writeJSON(w, http.StatusOK, map[string]string{"phone": input.Phone, "status": "suppressed"})
	})
	mux.HandleFunc("POST /course-deliveries", func(w http.ResponseWriter, r *http.Request) {
		var deadline delivery.Deadline
		if err := decodeJSON(r, &deadline); err != nil || deadline.CourseID == "" || deadline.LearnerID == "" || deadline.Phone == "" || deadline.CourseName == "" || deadline.DueAt.IsZero() {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "course_id, learner_id, phone, course_name, and due_at are required"})
			return
		}
		result, err := service.Deliver(r.Context(), deadline)
		if err != nil {
			status := http.StatusBadGateway
			var apiErr *delivery.InfraiError
			if errors.As(err, &apiErr) && apiErr.Status >= 400 && apiErr.Status < 500 {
				status = apiErr.Status
			}
			writeJSON(w, status, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, result)
	})
	mux.HandleFunc("GET /educator-reports/{courseID}", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, service.Report(r.PathValue("courseID")))
	})

	address := ":8080"
	log.Printf("course delivery service listening on %s", address)
	log.Fatal(http.ListenAndServe(address, mux))
}

func decodeJSON(r *http.Request, target any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("encode response: %v", err)
	}
}
