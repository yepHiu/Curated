// Package fc2 implements context-aware, exact-number metadata adapters for FC2 sources.
package fc2

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/imroc/req/v3"

	"curated-backend/internal/proxyenv"
	"curated-backend/internal/scraper"
)

var (
	ErrNotFound        = errors.New("no results: product not found")
	ErrIdentity        = errors.New("FC2 metadata identity mismatch")
	digitsPattern      = regexp.MustCompile(`(?i)^(?:FC2[-_ ]*(?:PPV[-_ ]*)?)?([0-9]+)$`)
	javtenPathPattern  = regexp.MustCompile(`^/video/[0-9]+/id([0-9]+)(?:/|$)`)
	articlePathPattern = regexp.MustCompile(`^/article/([0-9]+)(?:/|$)`)
	officialIDPattern  = regexp.MustCompile(`(?i)^(?:Product\s*ID|商品ID)\s*[:：]\s*(.+)$`)
)

// Client keeps one browser transport while reading the current proxy for each request.
type Client struct {
	http  *req.Client
	bases map[string]string
}

// NewClient 设置 Chrome TLS/HTTP2 特征、响应上限和动态代理，不依赖浏览器会话。
func NewClient(timeout time.Duration) *Client {
	if timeout <= 0 || timeout > 20*time.Second {
		timeout = 20 * time.Second
	}
	return &Client{
		http: req.C().ImpersonateChrome().SetTimeout(timeout).SetMaxResponseSize(4<<20).
			SetProxy(proxyenv.ProxyFromEnvironment).SetRedirectPolicy(req.MaxRedirectPolicy(5), req.SameHostRedirectPolicy()),
		bases: map[string]string{
			"FC2": "https://adult.contents.fc2.com", "fc2hub": "https://javten.com",
			"PPVDataBank": "https://ppvdatabank.com", "JavDB": "https://javdb.com",
		},
	}
}

// Digits 只接受完整 FC2/PPV 番号或数字，避免近似编号与其它作品混入。
func Digits(number string) string {
	match := digitsPattern.FindStringSubmatch(strings.TrimSpace(number))
	if len(match) != 2 {
		return ""
	}
	return match[1]
}

// Supports 标识本包能完整获取并校验的来源。
func Supports(provider string) bool {
	switch provider {
	case "FC2", "fc2hub", "PPVDataBank", "JavDB":
		return true
	default:
		return false
	}
}

// CheckHealth 只探测站点主页，避免用普通番号检测 FC2 专用站点产生假成功。
func (c *Client) CheckHealth(ctx context.Context, provider string) error {
	base, ok := c.bases[provider]
	if !ok {
		return fmt.Errorf("unknown FC2 provider %q", provider)
	}
	_, _, err := c.fetch(ctx, provider, base+"/", base+"/")
	return err
}

// Lookup 读取一个来源的搜索与详情，返回可补全字段而不要求每个来源都有封面。
func (c *Client) Lookup(ctx context.Context, provider, movieID, number string) (scraper.Metadata, error) {
	digits := Digits(number)
	if digits == "" {
		return scraper.Metadata{}, fmt.Errorf("%w: invalid number %q", ErrIdentity, number)
	}
	base, ok := c.bases[provider]
	if !ok {
		return scraper.Metadata{}, fmt.Errorf("unknown FC2 provider %q", provider)
	}
	target := base + "/article/" + digits + "/"
	if provider == "fc2hub" {
		target = base + "/search?kw=" + url.QueryEscape(digits)
	}
	if provider == "JavDB" {
		target = base + "/search?q=" + url.QueryEscape("FC2-"+digits)
	}
	doc, finalURL, err := c.fetch(ctx, provider, target, base+"/")
	if err != nil {
		return scraper.Metadata{}, err
	}
	if provider == "fc2hub" || provider == "JavDB" {
		detail := findDetailURL(doc, finalURL, base, provider, digits)
		if detail == "" {
			return scraper.Metadata{}, fmt.Errorf("%s: %w for FC2-%s", provider, ErrNotFound, digits)
		}
		if !isDetailDocument(doc, provider, digits) {
			doc, finalURL, err = c.fetch(ctx, provider, detail, target)
			if err != nil {
				return scraper.Metadata{}, err
			}
		} else {
			finalURL = detail
		}
	}
	metadata, err := parseDocument(doc, provider, movieID, digits, finalURL)
	if err != nil {
		return scraper.Metadata{}, fmt.Errorf("%s: %w", provider, err)
	}
	return metadata, nil
}

// validateDetailIdentity 校验详情地址与公开的 canonical，防止跳转到其它作品后误补全。
func validateDetailIdentity(doc *goquery.Document, provider, digits, homepage string) error {
	pattern := articlePathPattern
	if provider == "fc2hub" {
		pattern = javtenPathPattern
	}
	if provider == "JavDB" {
		if Digits(javDBNumber(doc)) != digits {
			return ErrIdentity
		}
		return nil
	}
	for index, raw := range []string{homepage, attr(doc, "link[rel='canonical']", "href"), attr(doc, "meta[property='og:url']", "content")} {
		if raw == "" {
			continue
		}
		resolved := absoluteURL(homepage, raw)
		pageURL, err := url.Parse(resolved)
		if err != nil {
			return ErrIdentity
		}
		match := pattern.FindStringSubmatch(pageURL.Path)
		identity := ""
		if len(match) == 2 {
			identity = match[1]
		} else if provider == "PPVDataBank" && pageURL.Path == "/article_search.php" {
			// PPVDataBank 的 og:url 保留旧式查询地址，与 article 路径等价。
			identity = Digits(pageURL.Query().Get("id"))
		}
		if identity == "" && index > 0 {
			// 站点通用的社交主页地址不提供作品身份，不作为错号依据。
			continue
		}
		if identity != digits {
			return ErrIdentity
		}
	}
	if provider == "PPVDataBank" {
		link, err := url.Parse(attr(doc, ".article_title a", "href"))
		if err == nil && link.Query().Get("aid") != "" && Digits(link.Query().Get("aid")) != digits {
			return ErrIdentity
		}
	}
	if provider == "FC2" {
		mismatch := false
		doc.Find(".items_article_softDevice p").Each(func(_ int, node *goquery.Selection) {
			// 官网公开商品 ID 时必须与请求番号一致，不能只凭 URL 判断。
			match := officialIDPattern.FindStringSubmatch(strings.TrimSpace(node.Text()))
			if len(match) == 2 && Digits(match[1]) != digits {
				mismatch = true
			}
		})
		if mismatch {
			return ErrIdentity
		}
	}
	return nil
}

// fetch 保留 403/验证页错误，限制读取大小，并使所有请求随任务取消。
func (c *Client) fetch(ctx context.Context, provider, target, referer string) (*goquery.Document, string, error) {
	response, err := c.http.R().SetContext(ctx).SetHeader("Referer", referer).
		SetHeader("Accept-Language", "ja,zh-TW;q=0.9,en;q=0.8").DisableAutoReadResponse().Get(target)
	if err != nil {
		return nil, "", fmt.Errorf("%s request failed: %w", provider, err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, (4<<20)+1))
	if err != nil {
		return nil, "", fmt.Errorf("%s response read failed: %w", provider, err)
	}
	if len(body) > 4<<20 {
		return nil, "", fmt.Errorf("%s parser failed: response exceeds 4 MiB", provider)
	}
	challenge := response.Header.Get("Cf-Mitigated") == "challenge" || isChallengePage(string(body))
	if challenge || response.StatusCode == http.StatusForbidden {
		return nil, "", fmt.Errorf("%s HTTP %d forbidden: browser verification/access denied at %s", provider, response.StatusCode, target)
	}
	if response.StatusCode == http.StatusNotFound {
		return nil, "", fmt.Errorf("%s: %w at %s", provider, ErrNotFound, target)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, "", fmt.Errorf("%s HTTP %d at %s", provider, response.StatusCode, target)
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return nil, "", fmt.Errorf("%s parser failed: %w", provider, err)
	}
	return doc, response.Response.Request.URL.String(), nil
}

// isChallengePage 识别验证页面的结构与标题，不把正文提及 Cloudflare 误判成拦截。
func isChallengePage(body string) bool {
	lower := strings.ToLower(body)
	return strings.Contains(lower, "<title>just a moment") || strings.Contains(lower, "<title>access denied") ||
		strings.Contains(lower, "cf-chl-") || strings.Contains(lower, "/cdn-cgi/challenge-platform/")
}

// findDetailURL 优先使用已命中的页面，继而读取 canonical/og:url 与搜索链接。
func findDetailURL(doc *goquery.Document, finalURL, base, provider, digits string) string {
	candidates := []string{finalURL, attr(doc, "link[rel='canonical']", "href"), attr(doc, "meta[property='og:url']", "content")}
	if provider == "fc2hub" {
		doc.Find("a[href*='/video/']").Each(func(_ int, node *goquery.Selection) {
			// 收集搜索页所有候选，后续按完整数字身份过滤。
			value, _ := node.Attr("href")
			candidates = append(candidates, value)
		})
	} else {
		doc.Find("a.box").Each(func(_ int, node *goquery.Selection) {
			// JavDB 使用卡片展示番号，先验证番号再接受不透明详情 ID。
			if numberInText(node.Find(".video-title strong, .video-title, .uid").Text(), digits) {
				value, _ := node.Attr("href")
				candidates = append(candidates, value)
			}
		})
	}
	for _, raw := range candidates {
		resolved := sameSiteURL(base, raw)
		if resolved == "" {
			continue
		}
		parsed, _ := url.Parse(resolved)
		if provider == "fc2hub" {
			match := javtenPathPattern.FindStringSubmatch(parsed.Path)
			if len(match) == 2 && match[1] == digits {
				return resolved
			}
		} else if strings.HasPrefix(parsed.Path, "/v/") {
			return resolved
		}
	}
	return ""
}

// sameSiteURL 限制搜索跳转到配置的站点，防止网页中的外站链接进入详情获取。
func sameSiteURL(base, raw string) string {
	root, err := url.Parse(base)
	if err != nil || strings.TrimSpace(raw) == "" {
		return ""
	}
	ref, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return ""
	}
	resolved := root.ResolveReference(ref)
	if !strings.EqualFold(resolved.Host, root.Host) || (resolved.Scheme != "http" && resolved.Scheme != "https") {
		return ""
	}
	if root.Scheme == "https" {
		resolved.Scheme = "https"
	}
	return resolved.String()
}

// isDetailDocument 判断搜索响应是否已含完整标题，避免重复下载直达详情页。
func isDetailDocument(doc *goquery.Document, provider, digits string) bool {
	if provider == "fc2hub" {
		return text(doc, "h1.card-text.fc2-title") != "" || movieJSONLD(doc) != nil
	}
	return text(doc, "h2.title strong.current-title") != "" && Digits(javDBNumber(doc)) == digits
}
