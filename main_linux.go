package main

import (
	"context"
	"os"
	"os/signal"
	"time"

	"github.com/kazufusa/go-service-example/llog"
)

func main() {
	f := Feature{elog: llog.NewLogger()}
	go func() {
		f.Start()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	select {
	case <-sigCh:
		f.Shutdown(ctx)
	}
}
