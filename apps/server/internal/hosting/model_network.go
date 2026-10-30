package hosting

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"
)

var blockedModelRanges = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"), netip.MustParsePrefix("10.0.0.0/8"),
	netip.MustParsePrefix("100.64.0.0/10"), netip.MustParsePrefix("127.0.0.0/8"),
	netip.MustParsePrefix("169.254.0.0/16"), netip.MustParsePrefix("172.16.0.0/12"),
	netip.MustParsePrefix("192.0.0.0/24"), netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("192.88.99.0/24"),
	netip.MustParsePrefix("192.168.0.0/16"), netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("198.51.100.0/24"), netip.MustParsePrefix("203.0.113.0/24"),
	netip.MustParsePrefix("224.0.0.0/4"), netip.MustParsePrefix("240.0.0.0/4"),
	netip.MustParsePrefix("::/128"), netip.MustParsePrefix("::1/128"),
	netip.MustParsePrefix("64:ff9b::/96"),
	netip.MustParsePrefix("64:ff9b:1::/48"), netip.MustParsePrefix("100::/64"),
	netip.MustParsePrefix("2001::/32"), netip.MustParsePrefix("2001:db8::/32"),
	netip.MustParsePrefix("2002::/16"),
	netip.MustParsePrefix("fc00::/7"), netip.MustParsePrefix("fe80::/10"),
	netip.MustParsePrefix("ff00::/8"),
}

func publicModelIP(ip netip.Addr) bool {
	ip = ip.Unmap()
	if !ip.IsValid() || !ip.IsGlobalUnicast() || ip.IsPrivate() {
		return false
	}
	for _, prefix := range blockedModelRanges {
		if prefix.Contains(ip) {
			return false
		}
	}
	return true
}

func modelChatURL(base string) (*url.URL, error) {
	u, err := url.Parse(base)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Opaque != "" || u.RawPath != "" || len(base) > 2048 {
		return nil, fail(400, "invalid_model_base", "Enter a public HTTPS API base URL")
	}
	host := strings.TrimSuffix(strings.ToLower(u.Hostname()), ".")
	if host == "" || host == "localhost" || strings.HasSuffix(host, ".localhost") || strings.HasSuffix(host, ".local") || strings.Contains(host, "%") {
		return nil, fail(400, "invalid_model_base", "Enter a public HTTPS API base URL")
	}
	if port := u.Port(); port != "" {
		n, err := strconv.Atoi(port)
		if err != nil || n < 1 || n > 65535 {
			return nil, fail(400, "invalid_model_base", "Enter a public HTTPS API base URL")
		}
	}
	if addr, err := netip.ParseAddr(host); err == nil {
		if !publicModelIP(addr) {
			return nil, fail(400, "invalid_model_base", "Enter a public HTTPS API base URL")
		}
	} else if !domainPattern.MatchString(host) || !strings.Contains(host, ".") {
		return nil, fail(400, "invalid_model_base", "Enter a public HTTPS API base URL")
	}
	if strings.ContainsAny(u.Path, "\\%") || path.Clean(u.Path) != u.Path && strings.TrimSuffix(path.Clean(u.Path), "/") != strings.TrimSuffix(u.Path, "/") {
		return nil, fail(400, "invalid_model_base", "Enter a public HTTPS API base URL")
	}
	for _, part := range strings.Split(strings.Trim(u.Path, "/"), "/") {
		if part == "." || part == ".." {
			return nil, fail(400, "invalid_model_base", "Enter a public HTTPS API base URL")
		}
	}
	u.Path = strings.TrimSuffix(u.Path, "/") + "/chat/completions"
	return u, nil
}

type modelResolver interface {
	LookupNetIP(context.Context, string, string) ([]netip.Addr, error)
}

func resolveModelHost(ctx context.Context, resolver modelResolver, host string) ([]netip.Addr, error) {
	if ip, err := netip.ParseAddr(host); err == nil {
		if !publicModelIP(ip) {
			return nil, errors.New("model target is not public")
		}
		return []netip.Addr{ip}, nil
	}
	lookup, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	ips, err := resolver.LookupNetIP(lookup, "ip", host)
	if err != nil || len(ips) == 0 {
		return nil, errors.New("model DNS lookup failed")
	}
	for _, ip := range ips {
		if !publicModelIP(ip) {
			return nil, errors.New("model DNS returned a restricted address")
		}
	}
	return ips, nil
}

func modelHTTPClient(resolver modelResolver) *http.Client {
	transport := &http.Transport{
		Proxy:                  nil,
		DisableKeepAlives:      true,
		TLSHandshakeTimeout:    5 * time.Second,
		ResponseHeaderTimeout:  20 * time.Second,
		MaxResponseHeaderBytes: 32 << 10,
	}
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		ips, err := resolveModelHost(ctx, resolver, host)
		if err != nil {
			return nil, err
		}
		dialer := net.Dialer{Timeout: 5 * time.Second}
		var last error
		for _, ip := range ips {
			conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(ip.String(), port))
			if err != nil {
				last = err
				continue
			}
			remote, ok := conn.RemoteAddr().(*net.TCPAddr)
			var remoteIP netip.Addr
			var valid bool
			if ok {
				remoteIP, valid = netip.AddrFromSlice(remote.IP)
			}
			if !ok || !valid || !publicModelIP(remoteIP) {
				conn.Close()
				return nil, errors.New("model connection reached a restricted address")
			}
			return conn, nil
		}
		return nil, fmt.Errorf("model connection failed: %w", last)
	}
	return &http.Client{Transport: transport, Timeout: 2 * time.Minute, CheckRedirect: func(*http.Request, []*http.Request) error {
		return errors.New("model redirects are not allowed")
	}}
}

func modelResponseBody(response *http.Response, limit int64) ([]byte, error) {
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, errors.New("model response exceeds limit")
	}
	return data, nil
}

func newModelChatRequest(ctx context.Context, base, key string, body []byte) (*http.Request, error) {
	if len(body) == 0 || len(body) > 1<<20 {
		return nil, fail(413, "model_request_too_large", "Model request exceeds limit")
	}
	u, err := modelChatURL(base)
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json, text/event-stream")
	request.Header.Set("Authorization", "Bearer "+key)
	return request, nil
}

func openModelChat(ctx context.Context, resolver modelResolver, base, key string, body []byte) (*http.Response, error) {
	request, err := newModelChatRequest(ctx, base, key, body)
	if err != nil {
		return nil, err
	}
	return modelHTTPClient(resolver).Do(request)
}

func requestModelChat(ctx context.Context, resolver modelResolver, base, key string, body []byte) (int, []byte, error) {
	response, err := openModelChat(ctx, resolver, base, key, body)
	if err != nil {
		return 0, nil, err
	}
	result, err := modelResponseBody(response, 4<<20)
	if err != nil {
		return 0, nil, err
	}
	return response.StatusCode, result, nil
}
