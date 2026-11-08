// Package router — роутер на net/http: статический сегмент важнее параметрического, промахи уходят в fallback группы.
package router

import (
	"context"
	"net/http"
	"slices"
	"strings"
)

// Handler — хендлер, возвращающий ошибку; её превращает в ответ ErrorHandler.
type Handler func(w http.ResponseWriter, r *http.Request) error

// Middleware оборачивает Handler; первый в списке — самый внешний.
type Middleware func(Handler) Handler

// ErrorHandler пишет ответ на ошибку хендлера или middleware.
type ErrorHandler func(w http.ResponseWriter, r *http.Request, err error)

// HTTPError — ошибка с заранее известным ответом; пустые Code и Message выводятся из Status.
type HTTPError struct {
	Status  int
	Code    string
	Message string
	// Err — внутренняя причина: уходит только в лог.
	Err error
}

func (e *HTTPError) Error() string {
	if e.Err != nil {
		return http.StatusText(e.Status) + ": " + e.Err.Error()
	}

	return http.StatusText(e.Status)
}

func (e *HTTPError) Unwrap() error { return e.Err }

// Route — зарегистрированный маршрут; Path в шаблоне с `:param`.
type Route struct {
	Method string
	Path   string
}

// AnyRoute — метка route для пути, не принадлежащего ни одной группе.
const AnyRoute = "/*"

type labelKey struct{}

// LabelHolder получает шаблон выбранного маршрута (или fallback'а) до middleware.
type LabelHolder struct{ route string }

// Route — записанный шаблон; AnyRoute, пока роутер не отработал.
func (h *LabelHolder) Route() string { return h.route }

// WithLabelHolder кладёт в контекст держатель метки, который заполнит Router.
func WithLabelHolder(ctx context.Context) (context.Context, *LabelHolder) {
	h := &LabelHolder{route: AnyRoute}

	return context.WithValue(ctx, labelKey{}, h), h
}

type segment struct {
	name  string
	param bool
}

type endpoint struct {
	template string
	segs     []segment
	key      string
	leaf     bool
	handlers map[string]Handler
}

type fallback struct {
	template string
	prefix   string
	exact    bool
	handler  Handler
}

// Router — диспетчер по методу поверх собственного сопоставления путей.
type Router struct {
	onError   ErrorHandler
	endpoints []*endpoint
	fallbacks []*fallback
	routes    []Route
}

// New строит роутер с корневым fallback'ом `/*` (JSON 404 без middleware).
func New(onError ErrorHandler) *Router {
	rt := &Router{onError: onError}
	rt.fallbacks = append(rt.fallbacks, &fallback{template: AnyRoute, handler: notFound})

	return rt
}

func notFound(http.ResponseWriter, *http.Request) error {
	return &HTTPError{Status: http.StatusNotFound}
}

// Group — префикс с общей цепочкой middleware.
type Group struct {
	rt     *Router
	prefix string
	chain  []Middleware
}

// Group открывает группу; с непустой цепочкой регистрирует её fallback'и `prefix` и `prefix/*`.
func (rt *Router) Group(prefix string, mw ...Middleware) *Group {
	g := &Group{rt: rt, prefix: prefix, chain: mw}
	g.registerFallbacks()

	return g
}

// Group открывает вложенную группу: цепочка и префикс наследуются.
func (g *Group) Group(prefix string, mw ...Middleware) *Group {
	sub := &Group{rt: g.rt, prefix: g.prefix + prefix, chain: joinChain(g.chain, mw)}
	sub.registerFallbacks()

	return sub
}

// Register добавляет маршрут группы; путь — prefix+template.
func (g *Group) Register(method, template string, h Handler, mw ...Middleware) {
	g.rt.Register(method, g.prefix+template, h, joinChain(g.chain, mw)...)
}

func joinChain(a, b []Middleware) []Middleware {
	return append(slices.Clip(a), b...)
}

func (g *Group) registerFallbacks() {
	if len(g.chain) == 0 {
		return
	}
	h := wrap(notFound, g.chain)
	g.rt.fallbacks = append(g.rt.fallbacks,
		&fallback{template: g.prefix, prefix: g.prefix, exact: true, handler: h},
		&fallback{template: g.prefix + "/*", prefix: g.prefix, handler: h})
}

func wrap(h Handler, chain []Middleware) Handler {
	for i := len(chain) - 1; i >= 0; i-- {
		h = chain[i](h)
	}

	return h
}

// Register добавляет маршрут вне групп.
func (rt *Router) Register(method, template string, h Handler, mw ...Middleware) {
	if !strings.HasPrefix(template, "/") {
		panic("router: template must start with /: " + template)
	}

	var ep *endpoint
	for _, e := range rt.endpoints {
		if e.template == template {
			ep = e

			break
		}
	}
	if ep == nil {
		segs := parseTemplate(template)
		ep = &endpoint{template: template, segs: segs, key: shapeKey(segs), handlers: map[string]Handler{}}
		rt.endpoints = append(rt.endpoints, ep)
		slices.SortStableFunc(rt.endpoints, (*endpoint).compare)
		rt.markLeaves()
	}
	if _, dup := ep.handlers[method]; dup {
		panic("router: duplicate route " + method + " " + template)
	}
	ep.handlers[method] = wrap(h, mw)
	rt.routes = append(rt.routes, Route{Method: method, Path: template})
}

// Routes — все зарегистрированные маршруты в порядке регистрации.
func (rt *Router) Routes() []Route {
	return append([]Route(nil), rt.routes...)
}

func parseTemplate(template string) []segment {
	parts := strings.Split(template[1:], "/")
	segs := make([]segment, len(parts))
	for i, p := range parts {
		if name, ok := strings.CutPrefix(p, ":"); ok {
			segs[i] = segment{name: name, param: true}
		} else {
			segs[i] = segment{name: p}
		}
	}

	return segs
}

// shapeKey — шаблон без имён параметров: узел echo не различает `/:id` и `/:name`.
func shapeKey(segs []segment) string {
	var b strings.Builder
	for _, s := range segs {
		b.WriteByte('/')
		if s.param {
			b.WriteByte(':')
		} else {
			b.WriteString(s.name)
		}
	}

	return b.String()
}

// markLeaves: параметрический узел echo — лист, если ни один шаблон (любого метода) не продолжает его.
func (rt *Router) markLeaves() {
	for _, e := range rt.endpoints {
		e.leaf = !slices.ContainsFunc(rt.endpoints, func(o *endpoint) bool {
			return strings.HasPrefix(o.key, e.key+"/")
		})
	}
}

// match как у echo: параметр — один сегмент и может быть пустым, но последний непустой параметр-лист забирает остаток с `/`.
// Значения не декодируются: Path уже декодирован, RawPath (его выбирает ServeHTTP) остаётся как есть.
func (e *endpoint) match(parts []string) (map[string]string, bool, bool) {
	n := len(e.segs)
	last := e.segs[n-1]
	if len(parts) < n || len(parts) > n && (!last.param || !e.leaf) {
		return nil, false, false
	}
	greedy := len(parts) > n
	var values map[string]string
	for i, s := range e.segs {
		v := parts[i]
		if i == n-1 {
			v = strings.Join(parts[i:], "/")
		}
		if !s.param {
			if v != s.name {
				return nil, false, false
			}

			continue
		}
		if v == "" && i == n-1 {
			return nil, false, false
		}
		if values == nil {
			values = map[string]string{}
		}
		values[s.name] = v
	}

	return values, true, greedy
}

// compare — порядок перебора: короче раньше, на первом различии статический сегмент раньше параметра.
func (e *endpoint) compare(o *endpoint) int {
	if d := len(e.segs) - len(o.segs); d != 0 {
		return d
	}
	for i := range e.segs {
		if e.segs[i].param != o.segs[i].param {
			if e.segs[i].param {
				return 1
			}

			return -1
		}
	}

	return 0
}

// lookup: при промахе по методу идёт к менее специфичным маршрутам (`/x/bulk` без PUT доходит до `/x/:id`);
// маршруты, где конечный параметр забрал лишние сегменты, пробуются после точных по числу сегментов.
func (rt *Router) lookup(r *http.Request, path string) (Handler, string, *endpoint) {
	if !strings.HasPrefix(path, "/") {
		return nil, "", nil
	}
	parts := strings.Split(path[1:], "/")
	var first *endpoint
	for _, wantGreedy := range []bool{false, true} {
		for _, ep := range rt.endpoints {
			values, ok, greedy := ep.match(parts)
			if !ok || greedy != wantGreedy {
				continue
			}
			if first == nil {
				first = ep
			}
			if hh := ep.handlers[r.Method]; hh != nil {
				for k, v := range values {
					r.SetPathValue(k, v)
				}

				return hh, ep.template, first
			}
		}
	}

	return nil, "", first
}

// ServeHTTP диспетчеризует запрос; без совпадения отвечает fallback группы.
func (rt *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if r.URL.RawPath != "" {
		path = r.URL.RawPath
	}
	h, label, first := rt.lookup(r, path)
	if h == nil {
		var fb *fallback
		if first != nil {
			fb = rt.ancestorFallback(first.template)
		} else {
			fb = rt.pathFallback(path)
		}
		h, label = fb.handler, fb.template
	}

	if holder, ok := r.Context().Value(labelKey{}).(*LabelHolder); ok {
		holder.route = label
	}

	tw := &trackingWriter{ResponseWriter: w}
	if err := h(tw, r); err != nil {
		rt.onError(tw, r, err)
	}
}

func (rt *Router) ancestorFallback(template string) *fallback {
	var best *fallback
	for _, fb := range rt.fallbacks {
		if fb.exact || !strings.HasPrefix(template, fb.prefix+"/") {
			continue
		}
		if best == nil || len(fb.prefix) > len(best.prefix) {
			best = fb
		}
	}

	return best
}

func (rt *Router) pathFallback(path string) *fallback {
	best := rt.fallbacks[0]
	for _, fb := range rt.fallbacks {
		owns := path == fb.prefix && fb.exact || !fb.exact && strings.HasPrefix(path, fb.prefix+"/")
		if !owns {
			continue
		}
		if len(fb.prefix) > len(best.prefix) {
			best = fb
		}
	}

	return best
}

// trackingWriter помнит, начат ли ответ; Unwrap нужен http.ResponseController.
type trackingWriter struct {
	http.ResponseWriter

	started bool
}

func (t *trackingWriter) WriteHeader(code int) {
	t.started = true
	t.ResponseWriter.WriteHeader(code)
}

func (t *trackingWriter) Write(b []byte) (int, error) {
	t.started = true

	return t.ResponseWriter.Write(b)
}

func (t *trackingWriter) Unwrap() http.ResponseWriter { return t.ResponseWriter }

// Committed — начат ли ответ на writer'е, выданном роутером.
func Committed(w http.ResponseWriter) bool {
	for w != nil {
		if tw, ok := w.(*trackingWriter); ok {
			return tw.started
		}
		u, ok := w.(interface{ Unwrap() http.ResponseWriter })
		if !ok {
			return false
		}
		w = u.Unwrap()
	}

	return false
}
