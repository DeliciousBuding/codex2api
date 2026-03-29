package proxyutil

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"

	"golang.org/x/net/proxy"
)

// Mode describes how a proxy setting should be interpreted.
type Mode int

const (
	// ModeInherit means no explicit proxy behavior was configured.
	ModeInherit Mode = iota
	// ModeDirect means outbound requests must bypass proxies explicitly.
	ModeDirect
	// ModeProxy means a concrete proxy URL was configured.
	ModeProxy
	// ModeInvalid means the proxy setting is present but malformed or unsupported.
	ModeInvalid
)

// Setting is the normalized interpretation of a proxy configuration value.
type Setting struct {
	Raw  string
	Mode Mode
	URL  *url.URL
}

// Parse normalizes a proxy configuration value into inherit, direct, or proxy modes.
// Supports "direct" and "none" keywords to explicitly bypass proxy.
func Parse(raw string) (Setting, error) {
	trimmed := strings.TrimSpace(raw)
	setting := Setting{Raw: trimmed}

	if trimmed == "" {
		setting.Mode = ModeInherit
		return setting, nil
	}

	// Support direct/none keywords to bypass proxy explicitly
	if strings.EqualFold(trimmed, "direct") || strings.EqualFold(trimmed, "none") {
		setting.Mode = ModeDirect
		return setting, nil
	}

	parsedURL, errParse := url.Parse(trimmed)
	if errParse != nil {
		setting.Mode = ModeInvalid
		return setting, fmt.Errorf("parse proxy URL failed: %w", errParse)
	}
	if parsedURL.Scheme == "" || parsedURL.Host == "" {
		setting.Mode = ModeInvalid
		return setting, fmt.Errorf("proxy URL missing scheme/host")
	}

	switch parsedURL.Scheme {
	case "socks5", "http", "https":
		setting.Mode = ModeProxy
		setting.URL = parsedURL
		return setting, nil
	default:
		setting.Mode = ModeInvalid
		return setting, fmt.Errorf("unsupported proxy scheme: %s", parsedURL.Scheme)
	}
}

// ParseWithFallback parses proxy setting with environment variable fallback.
// Environment proxy variables (HTTP_PROXY, HTTPS_PROXY, NO_PROXY) are respected.
func ParseWithFallback(raw string) (Setting, error) {
	// First try explicit setting
	setting, err := Parse(raw)
	if err != nil {
		return setting, err
	}

	// If explicit mode is not inherit, use it
	if setting.Mode != ModeInherit {
		return setting, nil
	}

	// Check NO_PROXY environment variable first
	noProxy := strings.TrimSpace(os.Getenv("NO_PROXY"))
	if noProxy == "" {
		noProxy = strings.TrimSpace(os.Getenv("no_proxy"))
	}
	if noProxy != "" {
		// NO_PROXY is set - caller should check if target is excluded
		// This signals that environment proxy might be bypassed for certain hosts
		setting.Mode = ModeInherit
	}

	return setting, nil
}

func cloneDefaultTransport() *http.Transport {
	if transport, ok := http.DefaultTransport.(*http.Transport); ok && transport != nil {
		return transport.Clone()
	}
	return &http.Transport{
		Proxy: http.ProxyFromEnvironment,
	}
}

// NewDirectTransport returns a transport that bypasses environment proxies.
func NewDirectTransport() *http.Transport {
	clone := cloneDefaultTransport()
	clone.Proxy = nil
	return clone
}

// NewEnvironmentTransport returns a transport that respects environment proxy settings.
func NewEnvironmentTransport() *http.Transport {
	clone := cloneDefaultTransport()
	clone.Proxy = http.ProxyFromEnvironment
	return clone
}

// BuildHTTPTransport constructs an HTTP transport for the provided proxy setting.
// Supports direct/none keywords, environment proxy inheritance, and explicit proxy URLs.
func BuildHTTPTransport(raw string) (*http.Transport, Mode, error) {
	setting, errParse := Parse(raw)
	if errParse != nil {
		return nil, setting.Mode, errParse
	}

	switch setting.Mode {
	case ModeInherit:
		// Inherit from environment - return nil to use default transport behavior
		return nil, setting.Mode, nil
	case ModeDirect:
		// Explicitly bypass proxy
		return NewDirectTransport(), setting.Mode, nil
	case ModeProxy:
		if setting.URL.Scheme == "socks5" {
			var proxyAuth *proxy.Auth
			if setting.URL.User != nil {
				username := setting.URL.User.Username()
				password, _ := setting.URL.User.Password()
				proxyAuth = &proxy.Auth{User: username, Password: password}
			}
			dialer, errSOCKS5 := proxy.SOCKS5("tcp", setting.URL.Host, proxyAuth, proxy.Direct)
			if errSOCKS5 != nil {
				return nil, setting.Mode, fmt.Errorf("create SOCKS5 dialer failed: %w", errSOCKS5)
			}
			transport := cloneDefaultTransport()
			transport.Proxy = nil
			transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
				return dialer.Dial(network, addr)
			}
			return transport, setting.Mode, nil
		}
		transport := cloneDefaultTransport()
		transport.Proxy = http.ProxyURL(setting.URL)
		return transport, setting.Mode, nil
	default:
		return nil, setting.Mode, nil
	}
}

// BuildDialer constructs a proxy dialer for settings that operate at the connection layer.
func BuildDialer(raw string) (proxy.Dialer, Mode, error) {
	setting, errParse := Parse(raw)
	if errParse != nil {
		return nil, setting.Mode, errParse
	}

	switch setting.Mode {
	case ModeInherit:
		return nil, setting.Mode, nil
	case ModeDirect:
		return proxy.Direct, setting.Mode, nil
	case ModeProxy:
		dialer, errDialer := proxy.FromURL(setting.URL, proxy.Direct)
		if errDialer != nil {
			return nil, setting.Mode, fmt.Errorf("create proxy dialer failed: %w", errDialer)
		}
		return dialer, setting.Mode, nil
	default:
		return nil, setting.Mode, nil
	}
}

// ShouldBypassProxy checks if a target host should bypass proxy based on NO_PROXY setting.
func ShouldBypassProxy(targetHost string) bool {
	noProxy := strings.TrimSpace(os.Getenv("NO_PROXY"))
	if noProxy == "" {
		noProxy = strings.TrimSpace(os.Getenv("no_proxy"))
	}
	if noProxy == "" {
		return false
	}

	// Parse target host (remove port if present)
	host := targetHost
	if idx := strings.LastIndex(targetHost, ":"); idx > 0 {
		host = targetHost[:idx]
	}

	// Check each NO_PROXY entry
	entries := strings.Split(noProxy, ",")
	for _, entry := range entries {
		entry = strings.TrimSpace(strings.ToLower(entry))
		if entry == "" {
			continue
		}

		// Exact match
		if strings.EqualFold(host, entry) {
			return true
		}

		// Domain suffix match (e.g., .example.com matches sub.example.com)
		if strings.HasPrefix(entry, ".") && strings.HasSuffix(strings.ToLower(host), entry) {
			return true
		}

		// Wildcard domain match
		if strings.HasPrefix(entry, "*.") {
			suffix := entry[1:] // Remove *
			if strings.HasSuffix(strings.ToLower(host), suffix) {
				return true
			}
		}
	}

	return false
}
