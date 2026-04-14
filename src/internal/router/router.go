package router

import (
	"net/http"
	"strings"
)

type Route struct {
	Method   string
	Segments []string
	Handler  http.HandlerFunc
}

type Router struct {
	routes []*Route
}

func New() *Router {
	return &Router{}
}

func (r *Router) Handle(method, pattern string, handler http.HandlerFunc) {
	r.routes = append(r.routes, &Route{
		Method:   method,
		Segments: splitPath(pattern),
		Handler:  handler,
	})
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	path := splitPath(req.URL.Path)

	for _, route := range r.routes {
		if route.Method != req.Method {
			continue
		}
		params, ok := matchRoute(route.Segments, path)
		if !ok {
			continue
		}
		ctx := withParams(req.Context(), params)
		route.Handler(w, req.WithContext(ctx))
		return
	}

	http.NotFound(w, req)
}

func matchRoute(pattern, path []string) (map[string]string, bool) {
	if len(pattern) != len(path) {
		return nil, false
	}

	params := make(map[string]string)
	for i, seg := range pattern {
		if strings.HasPrefix(seg, "{") && strings.HasSuffix(seg, "}") {
			name := seg[1 : len(seg)-1]
			params[name] = path[i]
		} else if seg != path[i] {
			return nil, false
		}
	}
	return params, true
}

func splitPath(path string) []string {
	path = strings.Trim(path, "/")
	if path == "" {
		return nil
	}
	return strings.Split(path, "/")
}
