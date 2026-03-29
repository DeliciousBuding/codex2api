package proxyutil

import (
	"net/http"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantMode Mode
		wantErr  bool
	}{
		{
			name:     "empty string - inherit",
			input:    "",
			wantMode: ModeInherit,
			wantErr:  false,
		},
		{
			name:     "direct keyword",
			input:    "direct",
			wantMode: ModeDirect,
			wantErr:  false,
		},
		{
			name:     "none keyword",
			input:    "none",
			wantMode: ModeDirect,
			wantErr:  false,
		},
		{
			name:     "DIRECT case insensitive",
			input:    "DIRECT",
			wantMode: ModeDirect,
			wantErr:  false,
		},
		{
			name:     "http proxy",
			input:    "http://proxy.example.com:8080",
			wantMode: ModeProxy,
			wantErr:  false,
		},
		{
			name:     "https proxy",
			input:    "https://proxy.example.com:8080",
			wantMode: ModeProxy,
			wantErr:  false,
		},
		{
			name:     "socks5 proxy",
			input:    "socks5://localhost:1080",
			wantMode: ModeProxy,
			wantErr:  false,
		},
		{
			name:     "invalid URL",
			input:    "://invalid",
			wantMode: ModeInvalid,
			wantErr:  true,
		},
		{
			name:     "unsupported scheme",
			input:    "ftp://proxy.example.com",
			wantMode: ModeInvalid,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setting, err := Parse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if setting.Mode != tt.wantMode {
				t.Errorf("Parse(%q) mode = %v, want %v", tt.input, setting.Mode, tt.wantMode)
			}
		})
	}
}

func TestBuildHTTPTransport(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantMode   Mode
		wantDirect bool
		wantErr    bool
	}{
		{
			name:       "inherit mode",
			input:      "",
			wantMode:   ModeInherit,
			wantDirect: false,
			wantErr:    false,
		},
		{
			name:       "direct mode",
			input:      "direct",
			wantMode:   ModeDirect,
			wantDirect: true,
			wantErr:    false,
		},
		{
			name:       "none mode",
			input:      "none",
			wantMode:   ModeDirect,
			wantDirect: true,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport, mode, err := BuildHTTPTransport(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildHTTPTransport(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if mode != tt.wantMode {
				t.Errorf("BuildHTTPTransport(%q) mode = %v, want %v", tt.input, mode, tt.wantMode)
			}
			if tt.wantDirect && transport != nil {
				if transport.Proxy != nil {
					t.Errorf("BuildHTTPTransport(%q) should have nil Proxy for direct mode", tt.input)
				}
			}
		})
	}
}

func TestShouldBypassProxy(t *testing.T) {
	// Save original NO_PROXY
	origNoProxy := ""
	for _, key := range []string{"NO_PROXY", "no_proxy"} {
		if v := ""; v != "" {
			continue
		}
		origNoProxy = ""
	}
	origNoProxy = ""
	defer func() {
		// Restore would be handled by test isolation
	}()

	tests := []struct {
		name       string
		noProxy    string
		targetHost string
		want       bool
	}{
		{
			name:       "no NO_PROXY set",
			noProxy:    "",
			targetHost: "api.example.com",
			want:       false,
		},
		{
			name:       "exact match",
			noProxy:    "api.example.com",
			targetHost: "api.example.com",
			want:       true,
		},
		{
			name:       "domain suffix match",
			noProxy:    ".example.com",
			targetHost: "api.example.com",
			want:       true,
		},
		{
			name:       "multiple entries - match",
			noProxy:    "localhost,.example.com,127.0.0.1",
			targetHost: "api.example.com",
			want:       true,
		},
		{
			name:       "multiple entries - no match",
			noProxy:    "localhost,.other.com,127.0.0.1",
			targetHost: "api.example.com",
			want:       false,
		},
		{
			name:       "with port - should match",
			noProxy:    ".example.com",
			targetHost: "api.example.com:443",
			want:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("NO_PROXY", tt.noProxy)
			got := ShouldBypassProxy(tt.targetHost)
			if got != tt.want {
				t.Errorf("ShouldBypassProxy(%q) with NO_PROXY=%q = %v, want %v",
					tt.targetHost, tt.noProxy, got, tt.want)
			}
		})
	}
}

func TestNewDirectTransport(t *testing.T) {
	transport := NewDirectTransport()
	if transport == nil {
		t.Fatal("NewDirectTransport() returned nil")
	}
	if transport.Proxy != nil {
		t.Error("NewDirectTransport() should have nil Proxy")
	}
}

func TestNewEnvironmentTransport(t *testing.T) {
	transport := NewEnvironmentTransport()
	if transport == nil {
		t.Fatal("NewEnvironmentTransport() returned nil")
	}
	// Should have Proxy set to ProxyFromEnvironment
	if transport.Proxy == nil {
		t.Error("NewEnvironmentTransport() should have Proxy set")
	}
}

func TestBuildDialer(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantMode Mode
		wantErr  bool
	}{
		{
			name:     "inherit mode",
			input:    "",
			wantMode: ModeInherit,
			wantErr:  false,
		},
		{
			name:     "direct mode",
			input:    "direct",
			wantMode: ModeDirect,
			wantErr:  false,
		},
		{
			name:     "none mode",
			input:    "none",
			wantMode: ModeDirect,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dialer, mode, err := BuildDialer(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildDialer(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if mode != tt.wantMode {
				t.Errorf("BuildDialer(%q) mode = %v, want %v", tt.input, mode, tt.wantMode)
			}
			if tt.input == "direct" || tt.input == "none" {
				if dialer == nil {
					t.Error("BuildDialer(direct/none) should return non-nil dialer")
				}
			}
		})
	}
}

// BenchmarkParse benchmarks the Parse function
func BenchmarkParse(b *testing.B) {
	inputs := []string{
		"",
		"direct",
		"none",
		"http://proxy.example.com:8080",
		"socks5://localhost:1080",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, input := range inputs {
			_, _ = Parse(input)
		}
	}
}

// BenchmarkShouldBypassProxy benchmarks the ShouldBypassProxy function
func BenchmarkShouldBypassProxy(b *testing.B) {
	noProxy := "localhost,127.0.0.1,.example.com,.internal.company.com,api.internal,*.wildcard.com"
	t.Setenv("NO_PROXY", noProxy)
	targetHosts := []string{
		"api.example.com",
		"sub.internal.company.com",
		"api.wildcard.com",
		"external.com",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, host := range targetHosts {
			ShouldBypassProxy(host)
		}
	}
}
