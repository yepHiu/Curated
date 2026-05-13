package server

import (
	"context"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestListenAndServeWithReadyReportsBoundAddress(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	readyCh := make(chan string, 1)
	errCh := make(chan error, 1)

	go func() {
		errCh <- ListenAndServeWithReady(
			ctx,
			"127.0.0.1:0",
			http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			}),
			nil,
			func(addr string) {
				readyCh <- addr
			},
		)
	}()

	var addr string
	select {
	case addr = <-readyCh:
	case err := <-errCh:
		t.Fatalf("server exited before ready callback: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for ready callback")
	}

	if !strings.HasPrefix(addr, "127.0.0.1:") {
		t.Fatalf("ready addr = %q, want loopback TCP address", addr)
	}
	if strings.HasSuffix(addr, ":0") {
		t.Fatalf("ready addr = %q, want resolved port", addr)
	}

	resp, err := http.Get("http://" + addr + "/health-check")
	if err != nil {
		t.Fatalf("GET bound server: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusNoContent)
	}

	cancel()
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("server shutdown error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for shutdown")
	}
}

func TestListenAndServeWithReadyDoesNotSignalWhenBindFails(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve port: %v", err)
	}
	defer listener.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	readyCalled := false
	err = ListenAndServeWithReady(ctx, listener.Addr().String(), http.NewServeMux(), nil, func(string) {
		readyCalled = true
	})
	if err == nil {
		t.Fatal("ListenAndServeWithReady returned nil, want bind error")
	}
	if readyCalled {
		t.Fatal("ready callback was called after bind failure")
	}
}
