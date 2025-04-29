/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package kahuna

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/kowabunga-cloud/kowabunga/kowabunga/common/klog"
)

const (
	// If timeouts limits are reached, an empty reply will be sent
	HttpMaxHeaderBytes      = 1048576 // 1 MiB
	HttpReadTimeoutSeconds  = 1800
	HttpWriteTimeoutSeconds = 3600
	HttpIdleTimeoutSeconds  = 60
	HttpGraceTimeoutSeconds = 60
)

type HTTPServer struct {
	shutdown     chan struct{}
	shuttingDown int32
	interrupted  bool
	interrupt    chan os.Signal
	httpServer   *http.Server
}

func NewHTTPServer(ke *KahunaEngine, address string, port int) *HTTPServer {
	rt := NewRouter(ke)
	s := HTTPServer{
		shutdown:  make(chan struct{}),
		interrupt: make(chan os.Signal, 1),
		httpServer: &http.Server{
			Handler:        rt,
			Addr:           fmt.Sprintf("%s:%d", address, port),
			MaxHeaderBytes: HttpMaxHeaderBytes,
			ReadTimeout:    HttpReadTimeoutSeconds * time.Second,
			WriteTimeout:   HttpWriteTimeoutSeconds * time.Second,
			IdleTimeout:    HttpIdleTimeoutSeconds * time.Second,
		},
	}
	s.httpServer.SetKeepAlivesEnabled(true)

	return &s
}

func (s *HTTPServer) handleShutdown(wg *sync.WaitGroup) {
	defer wg.Done()

	<-s.shutdown

	ctx, cancel := context.WithTimeout(context.TODO(), HttpGraceTimeoutSeconds*time.Second)
	defer cancel()

	shutdownChan := make(chan bool)
	go func() {
		var success bool
		defer func() {
			shutdownChan <- success
		}()
		if err := s.httpServer.Shutdown(ctx); err != nil {
			// Error from closing listeners, or context timeout:
			klog.Errorf("HTTP server Shutdown: %v", err)
		} else {
			success = true
		}
	}()
}

func handleInterrupt(once *sync.Once, s *HTTPServer) {
	once.Do(func() {
		for range s.interrupt {
			if s.interrupted {
				klog.Info("Server already shutting down")
				continue
			}
			s.interrupted = true
			klog.Infof("Shutting down... ")
			if err := s.Shutdown(); err != nil {
				klog.Errorf("HTTP server Shutdown: %v", err)
			}
		}
	})
}

func signalNotify(interrupt chan<- os.Signal) {
	signal.Notify(interrupt, syscall.SIGINT, syscall.SIGTERM)
}

func (s *HTTPServer) Serve() (err error) {
	wg := new(sync.WaitGroup)
	once := new(sync.Once)
	signalNotify(s.interrupt)
	go handleInterrupt(once, s)

	wg.Add(1)
	klog.Infof("Serving kowabunga at http://%s", s.httpServer.Addr)
	go func() {
		defer wg.Done()
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			klog.Fatalf("%v", err)
		}
		klog.Infof("Stopped serving kowabunga at http://%s", s.httpServer.Addr)
	}()

	wg.Add(1)
	go s.handleShutdown(wg)

	wg.Wait()
	return nil
}

// Shutdown server and clean up resources
func (s *HTTPServer) Shutdown() error {
	if atomic.CompareAndSwapInt32(&s.shuttingDown, 0, 1) {
		close(s.shutdown)
	}
	return nil
}
