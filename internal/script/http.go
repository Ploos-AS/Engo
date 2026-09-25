package script

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type HTTPClient struct {
	allowed  map[string]bool
	client   *http.Client
	maxBody  int64
	lookupIP func(string) ([]net.IP, error)
}

func NewHTTPClient(hosts []string, timeout time.Duration, maxBody int64) *HTTPClient {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	if maxBody <= 0 {
		maxBody = 256 * 1024
	}
	allowed := make(map[string]bool, len(hosts))
	for _, h := range hosts {
		h = strings.ToLower(strings.TrimSpace(h))
		if h != "" {
			allowed[h] = true
		}
	}
	h := &HTTPClient{allowed: allowed, maxBody: maxBody, lookupIP: net.LookupIP}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
		if err != nil {
			return nil, err
		}
		var last error
		dialer := &net.Dialer{Timeout: timeout}
		for _, ip := range ips {
			if blockedIP(ip) {
				continue
			}
			conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
			if err == nil {
				return conn, nil
			}
			last = err
		}
		if last != nil {
			return nil, last
		}
		return nil, fmt.Errorf("HTTP destination has no permitted addresses")
	}
	h.client = &http.Client{Timeout: timeout, Transport: transport, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return fmt.Errorf("too many redirects")
		}
		return h.validateURL(req.URL)
	}}
	return h
}
func (h *HTTPClient) Enabled() bool { return len(h.allowed) > 0 }
func (h *HTTPClient) Get(raw string) (map[string]interface{}, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	if err := h.validateURL(u); err != nil {
		return nil, err
	}
	resp, err := h.client.Get(u.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, h.maxBody+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > h.maxBody {
		return nil, fmt.Errorf("HTTP response exceeds %d bytes", h.maxBody)
	}
	return map[string]interface{}{"status": int64(resp.StatusCode), "body": string(body), "content_type": resp.Header.Get("Content-Type")}, nil
}
func (h *HTTPClient) validateURL(u *url.URL) error {
	if u.Scheme != "https" {
		return fmt.Errorf("HTTP capability requires https")
	}
	if u.User != nil {
		return fmt.Errorf("HTTP URL userinfo is not allowed")
	}
	if p := u.Port(); p != "" && p != "443" {
		return fmt.Errorf("HTTP capability only allows port 443")
	}
	host := strings.ToLower(u.Hostname())
	if !h.allowed[host] {
		return fmt.Errorf("HTTP host %q is not allowed", host)
	}
	ips, err := h.lookupIP(host)
	if err != nil {
		return fmt.Errorf("resolve HTTP host: %w", err)
	}
	if len(ips) == 0 {
		return fmt.Errorf("HTTP host resolves to no addresses")
	}
	for _, ip := range ips {
		if blockedIP(ip) {
			return fmt.Errorf("HTTP host resolves to blocked address")
		}
	}
	return nil
}
func blockedIP(ip net.IP) bool {
	if v4 := ip.To4(); v4 != nil {
		ip = v4
	}
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() || ip.IsMulticast() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast()
}
