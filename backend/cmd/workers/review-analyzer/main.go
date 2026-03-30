package main

import (
	// Event registration - CRITICAL for deserialization
	_ "github.com/udisondev/veziizi/backend/internal/domain/review/events"

	"github.com/udisondev/veziizi/backend/internal/infrastructure/handlers"
	"github.com/udisondev/veziizi/backend/internal/pkg/factory"
	"github.com/udisondev/veziizi/backend/internal/pkg/worker"
	"github.com/ThreeDotsLabs/watermill/message"
)

func main() {
	worker.Run(worker.Config{
		Name:          "review-analyzer",
		Topic:         "review.events",
		ConsumerGroup: "review_analyzer",
		LogFile:       "review-analyzer-worker.log",
		Handler: func(f *factory.Factory) message.NoPublishHandlerFunc {
			return handlers.NewReviewAnalyzerHandler(
				f.ReviewService(),
				f.ReviewAnalyzer(),
			).Handle
		},
	})
}
