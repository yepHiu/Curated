package fc2

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// TestFC2ConnectionRetry 用真实断开的 HTTP 连接覆盖请求 EOF 与响应体截断。
func TestFC2ConnectionRetry(t *testing.T) {
	for _, mode := range []string{"requestEOF", "bodyEOF", "persistentEOF", "403", "challenge", "404", "identity"} {
		t.Run(mode, func(t *testing.T) {
			var requests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				attempt := requests.Add(1)
				if mode == "persistentEOF" || (attempt == 1 && (mode == "requestEOF" || mode == "bodyEOF")) {
					conn, buf, err := w.(http.Hijacker).Hijack()
					if err != nil {
						t.Error(err)
						return
					}
					defer conn.Close()
					if mode == "bodyEOF" {
						fmt.Fprint(buf, "HTTP/1.1 200 OK\r\nContent-Length: 1000\r\n\r\nshort body")
						buf.Flush()
					}
					return
				}
				switch mode {
				case "403":
					w.WriteHeader(http.StatusForbidden)
				case "404":
					w.WriteHeader(http.StatusNotFound)
				case "challenge":
					fmt.Fprint(w, `<title>Just a moment...</title>`)
				case "identity":
					fmt.Fprint(w, `<link rel="canonical" href="/article/31231740/"><div class="article_title"><a>Other</a></div>`)
				default:
					fmt.Fprint(w, `<meta property="og:url" content="/article_search.php?id=3123174"><div class="article_title"><a>Recovered title</a></div><ul class="meta"><li>再生時間: 00:53:00</li></ul>`)
				}
			}))
			defer server.Close()
			client := NewClient(2 * time.Second)
			client.bases["PPVDataBank"] = server.URL
			metadata, err := client.Lookup(context.Background(), "PPVDataBank", "movie", "FC2-3123174")
			wantRequests := int32(1)
			if mode == "requestEOF" || mode == "bodyEOF" || mode == "persistentEOF" {
				wantRequests = 2
			}
			if requests.Load() != wantRequests {
				t.Fatalf("requests=%d want=%d", requests.Load(), wantRequests)
			}
			if mode == "requestEOF" || mode == "bodyEOF" {
				if err != nil || metadata.Number != "FC2-3123174" || metadata.Title != "Recovered title" || metadata.RuntimeMinutes != 53 {
					t.Fatalf("metadata=%+v err=%v", metadata, err)
				}
			} else if err == nil {
				t.Fatal("failure unexpectedly accepted")
			}
			if mode == "persistentEOF" && !errors.Is(err, io.EOF) {
				t.Fatalf("EOF lost: %v", err)
			}
		})
	}
}

// TestFC2RetryBudget 验证退避等待可取消，且两次请求不会重新获得完整超时预算。
func TestFC2RetryBudget(t *testing.T) {
	for _, mode := range []string{"cancel", "deadline"} {
		t.Run(mode, func(t *testing.T) {
			var requests atomic.Int32
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				conn, _, err := w.(http.Hijacker).Hijack()
				if err != nil {
					t.Error(err)
					return
				}
				conn.Close()
				if mode == "cancel" {
					time.AfterFunc(50*time.Millisecond, cancel)
				}
			}))
			defer server.Close()
			client := NewClient(100 * time.Millisecond)
			client.bases["PPVDataBank"] = server.URL
			_, err := client.Lookup(ctx, "PPVDataBank", "movie", "FC2-3123174")
			if mode == "cancel" && !errors.Is(err, context.Canceled) {
				t.Fatalf("unexpected cancellation error: %v", err)
			}
			if mode == "deadline" && !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("deadline lost: %v", err)
			}
			if requests.Load() != 1 {
				t.Fatalf("retried past cancellation/budget: %d", requests.Load())
			}
		})
	}
}
