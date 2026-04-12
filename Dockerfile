FROM golang:1.25-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY backend/ backend/

ARG BINARY
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /app ./backend/cmd/${BINARY}

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /app /app
ENTRYPOINT ["/app"]
