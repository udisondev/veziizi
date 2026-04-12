FROM golang:1.25-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY backend/ backend/

RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /bin/api ./backend/cmd/api && \
    CGO_ENABLED=0 go build -ldflags="-s -w" -o /bin/migrator ./backend/cmd/migrator && \
    CGO_ENABLED=0 go build -ldflags="-s -w" -o /bin/telegram-bot ./backend/cmd/telegram-bot && \
    CGO_ENABLED=0 go build -ldflags="-s -w" -o /bin/create-admin ./backend/cmd/tools/create-admin && \
    CGO_ENABLED=0 go build -ldflags="-s -w" -o /bin/seed-geo ./backend/cmd/tools/seed-geo && \
    for w in members invitations pending-organizations organizations freight-requests \
             review-receiver review-analyzer reviews-projection review-activator \
             fraudster-handler notification-dispatcher telegram-sender email-sender \
             support-tickets rate-limiter-cleanup; do \
      CGO_ENABLED=0 go build -ldflags="-s -w" -o "/bin/worker-${w}" "./backend/cmd/workers/${w}"; \
    done

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /bin/ /usr/local/bin/
CMD ["api"]
