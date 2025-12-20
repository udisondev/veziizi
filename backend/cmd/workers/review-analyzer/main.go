package main

import (
	// Event registration - CRITICAL for deserialization
	_ "codeberg.org/udison/veziizi/backend/internal/domain/review/events"

	"codeberg.org/udison/veziizi/backend/internal/infrastructure/handlers"
	"codeberg.org/udison/veziizi/backend/internal/pkg/factory"
	"codeberg.org/udison/veziizi/backend/internal/pkg/worker"
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
