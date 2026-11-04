package tool

import "net/url"

// applyListFilterAlias keeps the advertised legacy status filter compatible with
// the API v2 list controllers. Other routes have their own parameter semantics.
func applyListFilterAlias(method, path string, query url.Values) {
	if method != "GET" || (path != "/programs" && path != "/executions") {
		return
	}
	if query.Get("browseType") == "" && query.Get("status") != "" {
		query.Set("browseType", query.Get("status"))
	}
	query.Del("status")
}
