package fc2

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const javtenFixture = `<html><head><script type="application/ld+json">{"@graph":[{"@type":["Movie"],"name":"Fixture title","identifier":["FC2-PPV-3977618"],"image":"/cover.jpg","datePublished":"2024/03/01","duration":"PT1H20M1S","actor":[{"name":"Actor A"}],"genre":["Tag A"],"aggregateRating":{"ratingValue":"4.7"}}]}</script></head><body><h1 class="fc2-id">FC2-PPV-3977618</h1><div class="col des">Fixture plot<br>■出演<br>名前　Actor B<br>年齢　20</div><div class="card"><div class="card-header">売り手情報</div><div class="col-8">Seller<span class="badge">100</span></div></div><a data-fancybox="gallery" href="/sample.jpg">Sample</a></body></html>`

// TestJavtenSearchLinks 验证非跳转搜索、完整番号匹配、JSON-LD 和简介演员补全。
func TestJavtenSearchLinks(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 本地页面复现搜索列表与对应详情，并验证浏览器请求头。
		requests.Add(1)
		if !strings.Contains(r.UserAgent(), "Chrome/") {
			t.Errorf("missing browser user agent: %s", r.UserAgent())
		}
		if r.URL.Path == "/search" {
			fmt.Fprint(w, `<a href="/video/1/id39776180/wrong">Wrong prefix</a><a href="/video/2/id3977618/right">Right</a>`)
			return
		}
		if r.URL.Path != "/video/2/id3977618/right" {
			t.Errorf("unexpected detail %s", r.URL.Path)
		}
		if !strings.Contains(r.Referer(), "/search?kw=3977618") {
			t.Errorf("unexpected referer %s", r.Referer())
		}
		fmt.Fprint(w, javtenFixture)
	}))
	defer server.Close()
	client := NewClient(time.Second)
	client.bases["fc2hub"] = server.URL
	metadata, err := client.Lookup(context.Background(), "fc2hub", "movie", "FC2-PPV-3977618")
	if err != nil {
		t.Fatal(err)
	}
	if metadata.Number != "FC2-3977618" || metadata.Title != "Fixture title" || metadata.Studio != "Seller" || metadata.RuntimeMinutes != 80 || metadata.ReleaseDate != "2024-03-01" || metadata.Rating != 4.7 {
		t.Fatalf("unexpected metadata %+v", metadata)
	}
	if strings.Join(metadata.Actors, ",") != "Actor A,Actor B" || metadata.CoverURL != server.URL+"/cover.jpg" || len(metadata.PreviewImages) != 1 {
		t.Fatalf("missing enriched fields %+v", metadata)
	}
	if requests.Load() != 2 {
		t.Fatalf("expected search and detail, got %d requests", requests.Load())
	}
}

// TestJavtenDirectHit 验证跳转和 HTML 元信息直达详情不会重复下载。
func TestJavtenDirectHit(t *testing.T) {
	for _, mode := range []string{"redirect", "canonical", "og"} {
		t.Run(mode, func(t *testing.T) {
			// 每种直达机制分别计数实际网络请求。
			var requests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// 按模式返回重定向或已含详情的搜索响应。
				requests.Add(1)
				if mode == "redirect" && r.URL.Path == "/search" {
					http.Redirect(w, r, "/video/2/id3977618/right", http.StatusFound)
					return
				}
				meta := `<link rel="canonical" href="/video/2/id3977618/right">`
				if mode == "og" {
					meta = `<meta property="og:url" content="/video/2/id3977618/right">`
				}
				fmt.Fprint(w, strings.Replace(javtenFixture, "<head>", "<head>"+meta, 1))
			}))
			defer server.Close()
			client := NewClient(time.Second)
			client.bases["fc2hub"] = server.URL
			if _, err := client.Lookup(context.Background(), "fc2hub", "movie", "FC2-3977618"); err != nil {
				t.Fatal(err)
			}
			want := int32(1)
			if mode == "redirect" {
				want = 2
			}
			if requests.Load() != want {
				t.Fatalf("got %d requests, want %d", requests.Load(), want)
			}
		})
	}
}

// TestFC2AccessErrors 确保访问验证、商品不存在与身份错配不再混为无结果。
func TestFC2AccessErrors(t *testing.T) {
	cases := []struct {
		name, provider, body string
		status               int
		sentinel             error
		message              string
	}{
		{"403", "fc2hub", `<title>Just a moment...</title>`, 403, nil, "HTTP 403 forbidden"},
		{"challenge200", "fc2hub", `<title>Just a moment...</title>`, 200, nil, "browser verification"},
		{"officialNotFound", "FC2", `<title>お探しの商品が見つかりませんでした</title>`, 200, ErrNotFound, "product not found"},
		{"officialWrongID", "FC2", `<div data-section="userInfo"><h3>Wrong official</h3></div><div class="items_article_softDevice"><p>商品ID: 39776180</p></div>`, 200, ErrIdentity, "identity mismatch"},
		{"identity", "fc2hub", strings.ReplaceAll(javtenFixture, "3977618", "39776180"), 200, ErrIdentity, "identity mismatch"},
		{"unrelatedLink", "fc2hub", `<a href="/video/1/id39776180/other">Other</a>`, 200, ErrNotFound, "product not found"},
		{"databankCanonicalMismatch", "PPVDataBank", `<link rel="canonical" href="/article/39776180/"><div class="article_title"><a>Wrong movie</a></div>`, 200, ErrIdentity, "identity mismatch"},
		{"javdbRelatedNumber", "JavDB", `<link rel="canonical" href="/v/wrong"><h2 class="title"><strong class="current-title">Wrong movie</strong></h2><div class="movie-panel-info"><div class="panel-block"><strong>番號:</strong><span class="value">FC2-39776180</span></div><div class="panel-block">Related FC2-3977618</div></div>`, 200, ErrIdentity, "identity mismatch"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			// 错误页面只通过本地服务器注入，不依赖站点状态。
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// identity 用精确 canonical 加错误详情身份，验证解析阶段再次核对。
				w.WriteHeader(test.status)
				body := test.body
				if test.name == "identity" {
					body = `<link rel="canonical" href="/video/2/id3977618/right">` + body
				}
				fmt.Fprint(w, body)
			}))
			defer server.Close()
			client := NewClient(time.Second)
			client.bases[test.provider] = server.URL
			_, err := client.Lookup(context.Background(), test.provider, "movie", "FC2-3977618")
			if err == nil || !strings.Contains(err.Error(), test.message) {
				t.Fatalf("wrong error %v", err)
			}
			if test.sentinel != nil && !errors.Is(err, test.sentinel) {
				t.Fatalf("missing sentinel %v", err)
			}
			if strings.Contains(test.name, "challenge") || test.name == "403" {
				if errors.Is(err, ErrNotFound) {
					t.Fatalf("access denied became not found: %v", err)
				}
			}
		})
	}
}

// TestFC2WrongArticleRedirect 验证官网或数据银行跳转到其它作品时不会采用其标题。
func TestFC2WrongArticleRedirect(t *testing.T) {
	for _, provider := range []string{"FC2", "PPVDataBank"} {
		t.Run(provider, func(t *testing.T) { // 两个直达 article 来源共享跳转身份约束。
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// 精确请求被站点重定向到错误作品，详情仍具有有效标题。
				if r.URL.Path == "/article/3977618/" {
					http.Redirect(w, r, "/article/39776180/", http.StatusFound)
					return
				}
				fmt.Fprint(w, `<div data-section="userInfo"><h3>Wrong official</h3></div><div class="article_title"><a>Wrong databank</a></div>`)
			}))
			defer server.Close()
			client := NewClient(time.Second)
			client.bases[provider] = server.URL
			if _, err := client.Lookup(context.Background(), provider, "movie", "FC2-3977618"); !errors.Is(err, ErrIdentity) {
				t.Fatalf("redirect identity lost: %v", err)
			}
		})
	}
}

// TestFC2Sources 验证官方新页面、PPVDataBank 与 JavDB 的有效字段和番号检查。
func TestFC2Sources(t *testing.T) {
	cases := []struct{ provider, body string }{
		{"FC2", `<div data-section="userInfo"><h3>Official<span style="display:none">spam</span></h3><a href="/users/seller">Seller</a></div><div class="items_article_MainitemThumb"><img src="/cover.jpg"><p class="items_article_info">01:20:01</p></div><div class="items_article_softDevice"><p>販売日: 2024/03/01</p><p>Product ID: 3977618</p></div>`},
		{"PPVDataBank", `<meta property="og:url" content="/article_search.php?id=3977618"><div class="article_title"><a href="https://contents.fc2.com/aff.php?aid=3977618">Databank</a></div><div class="thumb"><img src="/cover.jpg"></div><ul class="meta"><li>販売者: Seller</li><li>販売日: 2024/03/01</li><li>再生時間: 01:20:01</li></ul>`},
		{"JavDB", `<link rel="canonical" href="/v/right"><h2 class="title"><strong class="current-title">JavDB title</strong></h2><img class="video-cover" src="/cover.jpg"><div class="movie-panel-info"><div class="panel-block"><strong>番號:</strong><span class="value">FC2-3977618</span></div><div class="panel-block"><strong>日期:</strong><span class="value">2024-03-01</span></div><div class="panel-block"><strong>時長:</strong><span class="value">80分鐘</span></div><div class="panel-block"><strong>片商:</strong><span class="value">Seller</span></div></div>`},
	}
	for _, test := range cases {
		t.Run(test.provider, func(t *testing.T) {
			// 同一页面可复现直达详情和正常解析。
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { // 返回指定来源 fixture。
				fmt.Fprint(w, test.body)
			}))
			defer server.Close()
			client := NewClient(time.Second)
			client.bases[test.provider] = server.URL
			metadata, err := client.Lookup(context.Background(), test.provider, "movie", "FC2-3977618")
			if err != nil {
				t.Fatal(err)
			}
			if metadata.Studio != "Seller" || metadata.RuntimeMinutes != 80 || metadata.ReleaseDate != "2024-03-01" || metadata.CoverURL != server.URL+"/cover.jpg" {
				t.Fatalf("unexpected fields %+v", metadata)
			}
			if strings.Contains(metadata.Title, "spam") {
				t.Fatalf("watermark retained %q", metadata.Title)
			}
		})
	}
}

// TestFC2RequestCancellation 确认取消任务会终止已发出的请求。
func TestFC2RequestCancellation(t *testing.T) {
	started := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { // 直到客户端取消才退出响应。
		close(started)
		<-r.Context().Done()
	}))
	defer server.Close()
	client := NewClient(time.Second)
	client.bases["FC2"] = server.URL
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { // 把真实阻塞请求放在独立 goroutine，主测试控制取消时机。
		_, err := client.Lookup(ctx, "FC2", "movie", "FC2-3977618")
		done <- err
	}()
	<-started
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancellation lost: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("request did not stop")
	}
}
