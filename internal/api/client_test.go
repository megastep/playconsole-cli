package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"google.golang.org/api/androidpublisher/v3"
	"google.golang.org/api/option"
)

func newTestEdit(t *testing.T, handler http.HandlerFunc) *Edit {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	service, err := androidpublisher.NewService(
		context.Background(),
		option.WithHTTPClient(server.Client()),
		option.WithEndpoint(server.URL+"/"),
		option.WithoutAuthentication(),
	)
	if err != nil {
		t.Fatalf("create service: %v", err)
	}

	return &Edit{
		client: &Client{
			service:     service,
			packageName: "com.example.app",
			timeout:     time.Second,
		},
		editID: "edit-123",
		ctx:    context.Background(),
		cancel: func() {},
	}
}

func TestCommitWithOptionsDefaultDoesNotStage(t *testing.T) {
	edit := newTestEdit(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("changesNotSentForReview"); got != "" {
			t.Fatalf("unexpected changesNotSentForReview query: %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"edit-123"}`))
	})

	if err := edit.Commit(); err != nil {
		t.Fatalf("Commit() error = %v", err)
	}
}

func TestCommitWithOptionsStagesChanges(t *testing.T) {
	edit := newTestEdit(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("changesNotSentForReview"); got != "true" {
			t.Fatalf("changesNotSentForReview query = %q, want true", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"edit-123"}`))
	})

	if err := edit.CommitWithOptions(CommitOptions{ChangesNotSentForReview: true}); err != nil {
		t.Fatalf("CommitWithOptions() error = %v", err)
	}
}
