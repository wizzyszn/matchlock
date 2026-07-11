package email

import (
	"context"
	"testing"
	"time"
)

func TestEnqueueRejectsWhenBufferFull(t *testing.T) {
	mailer := NewMailer(Config{APIKey: "test-key"})
	q := NewQueue(mailer, 1)
	q.ch <- Job{Type: JobMagicLink, To: "a@test.com", Link: "http://x"}

	err := q.Enqueue(Job{Type: JobMagicLink, To: "b@test.com", Link: "http://y"})
	if err == nil {
		t.Fatal("expected queue full error")
	}
}

func TestStartProcessesJobs(t *testing.T) {
	mailer := NewMailer(Config{APIKey: "test-key"})
	q := NewQueue(mailer, 4)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go q.Start(ctx)

	if err := q.Enqueue(Job{Type: JobMagicLink, To: "user@test.com", Link: "http://localhost/auth"}); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	time.Sleep(50 * time.Millisecond)
	cancel()
}
