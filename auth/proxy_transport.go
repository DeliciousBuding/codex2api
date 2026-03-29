package auth

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/codex2api/proxyutil"
	xproxy "golang.org/x/net/proxy"
)

// ConfigureTransportProxy applies HTTP(S) or SOCKS5 proxy settings to a transport.
// Enhanced to support CPA features:
//   - "direct" and "none" keywords to bypass proxy
//   - Environment proxy inheritance
func ConfigureTransportProxy(transport *http.Transport, rawProxyURL string, baseDialer *net.Dialer) error {
	if transport == nil {
		return nil
	}

	// Use proxyutil to parse proxy setting
	setting, err := proxyutil.Parse(rawProxyURL)
	if err != nil {
		return fmt.Errorf("parse proxy url: %w", err)
	}

	switch setting.Mode {
	case proxyutil.ModeInherit:
		// Inherit from environment - use default transport behavior
		if baseDialer == nil {
			baseDialer = &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
		}
		transport.Proxy = http.ProxyFromEnvironment
		transport.DialContext = baseDialer.DialContext
		return nil

	case proxyutil.ModeDirect:
		// Explicitly bypass proxy
		transport.Proxy = nil
		if baseDialer == nil {
			baseDialer = &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
		}
		transport.DialContext = baseDialer.DialContext
		return nil

	case proxyutil.ModeProxy:
		if setting.URL == nil {
			return fmt.Errorf("proxy URL is nil")
		}

		if baseDialer == nil {
			baseDialer = &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
		}

		switch setting.URL.Scheme {
		case "http", "https":
			transport.Proxy = http.ProxyURL(setting.URL)
			transport.DialContext = baseDialer.DialContext
			return nil

		case "socks5", "socks5h":
			var auth *xproxy.Auth
			if setting.URL.User != nil {
				password, _ := setting.URL.User.Password()
				auth = &xproxy.Auth{User: setting.URL.User.Username(), Password: password}
			}

			dialer, err := xproxy.SOCKS5("tcp", setting.URL.Host, auth, baseDialer)
			if err != nil {
				return fmt.Errorf("build socks5 dialer: %w", err)
			}
			if cd, ok := dialer.(contextDialer); ok {
				transport.DialContext = cd.DialContext
				transport.Proxy = nil
				return nil
			}

			transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
				type result struct {
					conn net.Conn
					err  error
				}
				done := make(chan result, 1)
				go func() {
					conn, err := dialer.Dial(network, address)
					done <- result{conn: conn, err: err}
				}()
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case out := <-done:
					return out.conn, out.err
				}
			}
			transport.Proxy = nil
			return nil

		default:
			return fmt.Errorf("unsupported proxy scheme: %s", setting.URL.Scheme)
		}

	case proxyutil.ModeInvalid:
		return fmt.Errorf("invalid proxy setting: %s", rawProxyURL)

	default:
		return fmt.Errorf("unknown proxy mode: %v", setting.Mode)
	}
}

type contextDialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}
