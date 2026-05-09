package tool

import (
	"net/url"
	"strings"
)

const redactedValue = "[REDACTED]"

var sensitiveKeys = map[string]struct{}{
	"access_token":  {},
	"api_key":       {},
	"apikey":        {},
	"authorization": {},
	"password":      {},
	"passwd":        {},
	"refresh_token": {},
	"secret":        {},
	"token":         {},
}

func redactInput(in map[string]any) map[string]any {
	redacted := make(map[string]any, len(in))

	for key, value := range in {
		if isSensitiveKey(key) {
			redacted[key] = redactedValue

			continue
		}

		redacted[key] = value
	}

	return redacted
}

func redactQueryParams(values url.Values) map[string][]string {
	redacted := make(map[string][]string, len(values))

	for key, vals := range values {
		copied := append([]string(nil), vals...)
		if isSensitiveKey(key) {
			for i := range copied {
				copied[i] = redactedValue
			}
		}

		redacted[key] = copied
	}

	return redacted
}

func redactURL(u *url.URL) string {
	redacted := *u
	q := redacted.Query()

	for key := range q {
		if !isSensitiveKey(key) {
			continue
		}

		values := q[key]
		for i := range values {
			values[i] = redactedValue
		}
		q[key] = values
	}

	redacted.RawQuery = q.Encode()

	return redacted.String()
}

func isSensitiveKey(key string) bool {
	_, ok := sensitiveKeys[strings.ToLower(key)]

	return ok
}
