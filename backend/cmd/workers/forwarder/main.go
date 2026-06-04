package main

import (
	"github.com/udisondev/veziizi/backend/internal/pkg/worker"
)

// forwarder — единственный потребитель Postgres outbox-топика: разворачивает
// forwarder-envelope и публикует сообщения в Redis-стримы. Работает строго в
// одном инстансе. Blank-импорты events не нужны: forwarder не десериализует
// payload, он перекладывает сообщения как есть.
func main() {
	worker.RunForwarder()
}
