package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"
)

type HttpServer struct {
	Server   *http.Server
	shutdown chan struct{}
}

func (s *HttpServer) ListenAndServe() (err error) {
	if s.shutdown == nil {
		s.shutdown = make(chan struct{})
	}

	err = s.Server.ListenAndServe()
	if err == http.ErrServerClosed {
		// expected error after calling Server.Shutdown().
		err = nil
	} else if err != nil {
		err = fmt.Errorf("unexpected error from ListenAndServe: %w", err)
		return
	}

	log.Debugln("waiting for shutdown finishing...")
	<-s.shutdown
	log.Debugln("shutdown finished")

	return
}

func (s *HttpServer) WaitExitSignal(timeout time.Duration) {
	waiter := make(chan os.Signal, 1) // buffered channel
	// signal.Notify(waiter, syscall.SIGTERM, syscall.SIGINT)

	// blocks here until there's a signal
	<-waiter

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	err := s.Server.Shutdown(ctx)
	if err != nil {
		log.Errorln("shutting down: " + err.Error())
	} else {
		log.Debugln("shutdown processed successfully")
		close(s.shutdown)
	}
}
