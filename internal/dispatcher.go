package internal

import (
	"database/sql"

	"github.com/fraddy91/smtp-to-apprise/logger"
)

type Backend struct {
	Db         *sql.DB
	AppriseURL string
	Dispatcher *Dispatcher
}
type Dispatcher struct {
	queue chan *dispatchJob
}

type dispatchJob struct {
	url  string
	data []byte
	rec  *Record
}

func NewDispatcher(size int) *Dispatcher {
	d := &Dispatcher{queue: make(chan *dispatchJob, size)}
	numWorkers := 5
	for i := 0; i < numWorkers; i++ {
		go d.worker()
	}
	return d
}

func (d *Dispatcher) worker() {
	for job := range d.queue {
		if err := postWithRetry(job.url, job.data, 5); err != nil {
			logger.Errorf("Failed to deliver to %s/%s: %v", job.rec.Email, job.rec.Key, err)
		} else {
			logger.Infof("Delivered %s (%s) for %s to Apprise key %s",
				job.rec.MimeType, job.rec.Tags, job.rec.Email, job.rec.Key)
		}
	}
}

func (d *Dispatcher) Enqueue(url string, data []byte, rec *Record) {
	select {
	case d.queue <- &dispatchJob{url, data, rec}:
	default:
		logger.Warnf("Queue full, dropping message for %s/%s", rec.Email, rec.Key)
	}
}
