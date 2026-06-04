package main

import (
	"github.com/ThreeDotsLabs/watermill/components/cqrs"

	_ "github.com/udisondev/veziizi/backend/internal/domain/review/events"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/handlers"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/messaging"
	"github.com/udisondev/veziizi/backend/internal/pkg/factory"
	"github.com/udisondev/veziizi/backend/internal/pkg/worker"
)

func main() {
	worker.Run(worker.Config{
		Name:          "review-analyzer",
		Topic:         messaging.TopicReviewEvents,
		ConsumerGroup: "review_analyzer",
		LogFile:       "review-analyzer-worker.log",
		Setup: func(f *factory.Factory, ep *cqrs.EventGroupProcessor) error {
			h := handlers.NewReviewAnalyzerHandler(f.ReviewService(), f.ReviewAnalyzer())
			return ep.AddHandlersGroup("review-analyzer", handlers.ReviewAnalyzerGroupHandlers(h)...)
		},
	})
}
