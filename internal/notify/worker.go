package notify

import (
	"context"
	"log/slog"
	"time"
)

// Worker периодически отправляет письма из outbox.
type Worker struct {
	service  *Service
	interval time.Duration
	logger   *slog.Logger
}

// NewWorker создаёт воркер отправки.
func NewWorker(service *Service, interval time.Duration, logger *slog.Logger) *Worker {
	return &Worker{
		service:  service,
		interval: interval,
		logger:   logger,
	}
}

// Run запускает цикл отправки до отмены контекста.
func (w *Worker) Run(ctx context.Context) {
	w.process(ctx)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.process(ctx)
		}
	}
}

func (w *Worker) process(ctx context.Context) {
	processed, err := w.service.ProcessDue(ctx)
	if err != nil {
		w.logger.Error("ошибка обработки outbox", "error", err)
		return
	}
	if processed > 0 {
		w.logger.Info("обработана пачка outbox", "count", processed)
	}
}
