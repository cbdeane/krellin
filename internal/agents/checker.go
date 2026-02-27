package agents

import (
	"context"
	"net"
	"net/url"
	"strings"
	"time"
)

type Checker interface {
	Check(ctx context.Context, provider Provider) string
}

type DefaultChecker struct {
	Timeout time.Duration
}

func (c DefaultChecker) Check(ctx context.Context, provider Provider) string {
	endpoint := provider.BaseURL
	if endpoint == "" {
		endpoint = defaultBaseURL(provider.Type)
	}
	if endpoint == "" {
		return "ready"
	}
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Host == "" {
		return "unreachable"
	}
	host := parsed.Host
	if !strings.Contains(host, ":") {
		switch strings.ToLower(parsed.Scheme) {
		case "http":
			host = host + ":80"
		default:
			host = host + ":443"
		}
	}
	timeout := c.Timeout
	if timeout == 0 {
		timeout = 800 * time.Millisecond
	}
	dialer := net.Dialer{Timeout: timeout}
	conn, err := dialer.DialContext(ctx, "tcp", host)
	if err != nil {
		return "unreachable"
	}
	_ = conn.Close()
	return "ready"
}

func defaultBaseURL(pt ProviderType) string {
	switch pt {
	case ProviderOpenAI:
		return "https://api.openai.com"
	case ProviderAnthropic:
		return "https://api.anthropic.com"
	case ProviderGrok:
		return "https://api.x.ai"
	case ProviderGemini:
		return "https://generativelanguage.googleapis.com"
	default:
		return ""
	}
}
