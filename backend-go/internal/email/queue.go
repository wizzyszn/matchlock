package email

import (
	"context"
	"fmt"
	"log/slog"
)

// JobType identifies an outbound email template.
type JobType string

const (
	JobMagicLink   JobType = "magic_link"
	JobWagerInvite JobType = "wager_invite"
)

// Job is one asynchronous email delivery task.
type Job struct {
	Type       JobType
	To         string
	Link       string
	MakerEmail string
	MatchLabel string
	InviteURL  string
}

// Queue buffers outbound mail and delivers via a background worker.
type Queue struct {
	mailer *Mailer
	ch     chan Job
}

// NewQueue creates a buffered email queue.
func NewQueue(mailer *Mailer, buffer int) *Queue {
	if buffer < 1 {
		buffer = 64
	}
	return &Queue{
		mailer: mailer,
		ch:     make(chan Job, buffer),
	}
}

// Start runs the background worker until ctx is cancelled.
func (q *Queue) Start(ctx context.Context) {
	slog.Info("email queue worker started")
	for {
		select {
		case <-ctx.Done():
			slog.Info("email queue worker stopped")
			return
		case job := <-q.ch:
			q.deliver(job)
		}
	}
}

// Enqueue schedules an email for background delivery.
func (q *Queue) Enqueue(job Job) error {
	select {
	case q.ch <- job:
		return nil
	default:
		return fmt.Errorf("email queue full")
	}
}

func (q *Queue) deliver(job Job) {
	var err error
	switch job.Type {
	case JobMagicLink:
		err = q.mailer.SendMagicLink(job.To, job.Link)
	case JobWagerInvite:
		err = q.mailer.SendWagerInvite(job.To, job.MakerEmail, job.MatchLabel, job.InviteURL)
	default:
		err = fmt.Errorf("unknown email job type %q", job.Type)
	}
	if err != nil {
		slog.Error("email delivery failed", "type", job.Type, "to", job.To, "err", err)
		return
	}
	slog.Info("email sent", "type", job.Type, "to", job.To)
}