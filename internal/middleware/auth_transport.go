package middleware

import "net/http"

type authTransport struct {
	base http.RoundTripper
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	r := req.Clone(req.Context())

	token := GetAuthorization(req.Context())
	if token == "" {
		return t.base.RoundTrip(r)
	}

	// 如果是通过自定义 token 头传入（如禅道），则转发为 token 头；
	// 否则保留原有行为，转发为 Authorization: Bearer <token>。
	if IsFromTokenHeader(req.Context()) {
		r.Header.Set("token", token)
	} else {
		r.Header.Set("Authorization", "Bearer "+token)
	}

	return t.base.RoundTrip(r)
}

// WithBearerAuth wraps an HTTP client to forward bearer tokens from context.
func WithBearerAuth(c *http.Client) *http.Client {
	base := c.Transport
	if base == nil {
		base = http.DefaultTransport
	}

	return &http.Client{
		Transport:     &authTransport{base: base},
		CheckRedirect: c.CheckRedirect,
		Jar:           c.Jar,
		Timeout:       c.Timeout,
	}
}
