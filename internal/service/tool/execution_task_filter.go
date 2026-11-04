package tool

import "net/url"

// The bundled specification historically exposed status, but ZenTao API v2
// reads browseType. Preserve existing clients without changing other endpoints.
func applyExecutionTaskFilterAlias(method, path string, q url.Values) {
	if method != "GET" || (path != "/executions/:executionID/tasks" && path != "/executions/{executionID}/tasks") {
		return
	}
	if q.Get("browseType") == "" && q.Get("status") != "" {
		q.Set("browseType", q.Get("status"))
	}
	q.Del("status")
}
