#!/bin/bash

# Run all backend services
# This script is used by air for hot-reload

# Trap to kill all background processes on exit
cleanup() {
    echo "Shutting down all services..."
    kill $(jobs -p) 2>/dev/null
    wait
}
trap cleanup EXIT INT TERM

# Start all workers in background
./tmp/worker-members &
./tmp/worker-invitations &
./tmp/worker-pending-organizations &
./tmp/worker-organizations &
./tmp/worker-freight-requests &
./tmp/worker-orders &
./tmp/worker-order-creator &
./tmp/worker-review-receiver &
./tmp/worker-review-analyzer &
./tmp/worker-reviews-projection &
./tmp/worker-review-activator &
./tmp/worker-fraudster-handler &
./tmp/worker-order-fraud-analyzer &
./tmp/worker-notification-dispatcher &
./tmp/worker-telegram-sender &
./tmp/telegram-bot &

# Run API in foreground (so air can capture output and restart)
./tmp/api
