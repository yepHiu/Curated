// Package browserheaders holds outbound HTTP headers aligned with common browser probes.
package browserheaders

// UserAgentChrome120 matches settings proxy ping and asset downloads (CDN / hotlink behavior).
const UserAgentChrome120 = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

// AcceptLikeChrome matches settings proxy ping.
const AcceptLikeChrome = "text/html,application/xhtml+xml;q=0.9,*/*;q=0.8"
