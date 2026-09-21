// Package wishlistassets 保存数据库旁的不可变原图和可再生缩略图。
package wishlistassets

import (
	"bytes"
	"context"
	"crypto/sha256"
	"curated-backend/internal/config"
	"curated-backend/internal/proxyenv"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/disintegration/imaging"
	_ "golang.org/x/image/webp"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const MaxBytes = 12 << 20

// File 是已校验且提交成功的图片，不含外部用户可控的绝对路径。
type File struct{ Path, ThumbnailPath, Hash string }

// publicIP 判断图片目标，拒绝本机、私网与非单播地址。
func publicIP(ip net.IP) bool {
	return ip.IsGlobalUnicast() && !ip.IsPrivate() && !ip.IsLoopback() && !ip.IsLinkLocalUnicast() && !(ip.To4() != nil && ip.To4()[0] == 100 && ip.To4()[1] >= 64 && ip.To4()[1] <= 127)
}

// validateURL 在每个请求与跳转前验证 HTTPS 目标及当前 DNS 解析。
func validateURL(ctx context.Context, u *url.URL) error {
	if u == nil || u.Scheme != "https" || u.User != nil || u.Hostname() == "" {
		return errors.New("invalid image URL")
	}
	ips, e := net.DefaultResolver.LookupIPAddr(ctx, u.Hostname())
	if e != nil {
		return e
	}
	if len(ips) == 0 {
		return errors.New("empty image DNS")
	}
	for _, ip := range ips {
		if !publicIP(ip.IP) {
			return errors.New("private image target")
		}
	}
	return nil
}

// Download 使用当前代理配置获取有界图片，每次重定向重新验证。
func Download(ctx context.Context, root, id, source string, proxy config.ProxyConfig) (File, error) {
	client, e := proxyenv.NewHTTPClientForProxy(proxy, 25*time.Second)
	if e != nil {
		return File{}, e
	}
	defer client.CloseIdleConnections()
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error { // 跳转不继承私网访问许可。
		if len(via) > 4 {
			return errors.New("too many redirects")
		}
		return validateURL(req.Context(), req.URL)
	}
	if !proxy.Enabled {
		tr := client.Transport.(*http.Transport)
		tr.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) { // 直连使用经验证的 IP 防止 DNS 校验和连接之间重绑定。
			host, port, e := net.SplitHostPort(address)
			if e != nil {
				return nil, e
			}
			ips, e := net.DefaultResolver.LookupIPAddr(ctx, host)
			if e != nil {
				return nil, e
			}
			for _, ip := range ips {
				if !publicIP(ip.IP) {
					return nil, errors.New("private image address")
				}
			}
			var last error
			for _, ip := range ips {
				conn, e := (&net.Dialer{Timeout: 10 * time.Second}).DialContext(ctx, network, net.JoinHostPort(ip.IP.String(), port))
				if e == nil {
					return conn, nil
				}
				last = e
			}
			return nil, last
		}
	}
	req, e := http.NewRequestWithContext(ctx, "GET", source, nil)
	if e != nil {
		return File{}, e
	}
	if e = validateURL(ctx, req.URL); e != nil {
		return File{}, e
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Referer", req.URL.Scheme+"://"+req.URL.Host+"/")
	resp, e := client.Do(req)
	if e != nil {
		return File{}, e
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return File{}, fmt.Errorf("image HTTP %d", resp.StatusCode)
	}
	b, e := io.ReadAll(io.LimitReader(resp.Body, MaxBytes+1))
	if e != nil {
		return File{}, e
	}
	return Save(root, id, b)
}

// Save 解码验证后按内容寻址保存原图和 JPEG 缩略图，已有文件不覆盖。
func Save(root, id string, b []byte) (File, error) {
	if len(b) > MaxBytes {
		return File{}, errors.New("image too large")
	}
	if id == "" || strings.ContainsAny(id, "/\\.") {
		return File{}, errors.New("invalid item id")
	}
	cfg, format, e := image.DecodeConfig(bytes.NewReader(b))
	if e != nil {
		return File{}, e
	}
	if cfg.Width < 1 || cfg.Height < 1 || int64(cfg.Width)*int64(cfg.Height) > 24_000_000 {
		return File{}, errors.New("image dimensions too large")
	}
	ext := format
	if ext == "jpeg" {
		ext = "jpg"
	}
	switch ext {
	case "jpg", "png", "gif", "webp":
	default:
		return File{}, errors.New("unsupported image")
	}
	h := sha256.Sum256(b)
	hash := hex.EncodeToString(h[:])
	f := File{Path: filepath.ToSlash(filepath.Join(id, "images", hash+"."+ext)), ThumbnailPath: filepath.ToSlash(filepath.Join(id, "derived", hash+"-thumb-v1.jpg")), Hash: hash}
	if e = writeImmutable(filepath.Join(root, filepath.FromSlash(f.Path)), b); e != nil {
		return File{}, e
	}
	img, _, e := image.Decode(bytes.NewReader(b))
	if e != nil {
		return File{}, e
	}
	thumb := imaging.Fit(img, 400, 600, imaging.Lanczos)
	var encoded bytes.Buffer
	if e = jpeg.Encode(&encoded, thumb, &jpeg.Options{Quality: 82}); e != nil {
		return File{}, e
	}
	if e = writeImmutable(filepath.Join(root, filepath.FromSlash(f.ThumbnailPath)), encoded.Bytes()); e != nil {
		return File{}, e
	}
	return f, nil
}

// writeImmutable 先写临时文件，再发布完整内容；已经存在的哈希文件保持不变。
func writeImmutable(path string, b []byte) error {
	if existing, e := os.ReadFile(path); e == nil {
		if bytes.Equal(existing, b) {
			return nil
		}
		return errors.New("immutable asset content mismatch")
	}
	if e := os.MkdirAll(filepath.Dir(path), 0700); e != nil {
		return e
	}
	f, e := os.CreateTemp(filepath.Dir(path), ".pending-*")
	if e != nil {
		return e
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if _, e = f.Write(b); e != nil {
		f.Close()
		return e
	}
	if e = f.Close(); e != nil {
		return e
	}
	return os.Rename(tmp, path)
}
