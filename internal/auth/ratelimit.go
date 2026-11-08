package auth

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Лимиты логина (spec 005, A-09): per-IP и per-email, скользящее окно.
const (
	IPLimit     = 10
	IPWindow    = 5 * time.Minute
	EmailLimit  = 20
	EmailWindow = time.Hour

	// cleanupEvery — каждый N-й Allow выбрасывает ключи без попыток в окне.
	cleanupEvery = 100
)

// RateLimiter — счётчик попыток логина в памяти; инстанс один, поэтому этого достаточно.
type RateLimiter struct {
	mu     sync.Mutex
	now    func() time.Time
	perIP  bool
	ips    map[string][]time.Time
	emails map[string][]time.Time
	calls  int
}

// LimiterOption настраивает RateLimiter.
type LimiterOption func(*RateLimiter)

// WithoutIPLimit оставляет только лимит по email. Нужен, когда реальный IP клиента неизвестен
// (за прокси без TRUSTED_PROXIES все клиенты — один адрес): иначе десять чужих неудач
// закрывали бы вход всей семье.
func WithoutIPLimit() LimiterOption {
	return func(l *RateLimiter) { l.perIP = false }
}

// NewRateLimiter создаёт лимитер; nil now — time.Now.
func NewRateLimiter(now func() time.Time, opts ...LimiterOption) *RateLimiter {
	if now == nil {
		now = time.Now
	}
	l := &RateLimiter{
		now:    now,
		perIP:  true,
		ips:    map[string][]time.Time{},
		emails: map[string][]time.Time{},
	}
	for _, opt := range opts {
		opt(l)
	}
	return l
}

// Allow регистрирует попытку; при блокировке попытка не записывается, retryAfter — до конца окна.
func (l *RateLimiter) Allow(ip, email string) (time.Duration, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	l.calls++
	if l.calls%cleanupEvery == 0 {
		l.cleanup(now)
	}

	emailHits := prune(l.emails[email], now.Add(-EmailWindow))
	var ipHits []time.Time
	if l.perIP {
		ipHits = prune(l.ips[ip], now.Add(-IPWindow))
	}

	if len(ipHits) >= IPLimit {
		return retryAfter(ipHits[0], IPWindow, now), false
	}
	if len(emailHits) >= EmailLimit {
		return retryAfter(emailHits[0], EmailWindow, now), false
	}

	if l.perIP {
		l.ips[ip] = append(ipHits, now)
	}
	l.emails[email] = append(emailHits, now)
	return 0, true
}

// Reset снимает счётчик email после успешного входа; счётчик IP остаётся.
func (l *RateLimiter) Reset(email string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.emails, email)
}

func (l *RateLimiter) cleanup(now time.Time) {
	for key, hits := range l.ips {
		if len(prune(hits, now.Add(-IPWindow))) == 0 {
			delete(l.ips, key)
		}
	}
	for key, hits := range l.emails {
		if len(prune(hits, now.Add(-EmailWindow))) == 0 {
			delete(l.emails, key)
		}
	}
}

// prune отбрасывает попытки старше since; hits отсортированы по возрастанию.
func prune(hits []time.Time, since time.Time) []time.Time {
	i := 0
	for i < len(hits) && !hits[i].After(since) {
		i++
	}
	return hits[i:]
}

func retryAfter(oldest time.Time, window time.Duration, now time.Time) time.Duration {
	d := oldest.Add(window).Sub(now)
	if d < time.Second {
		return time.Second
	}
	return d.Round(time.Second)
}

// ParseTrustedProxies разбирает TRUSTED_PROXIES: CIDR через запятую, пустая строка — ни одного.
func ParseTrustedProxies(raw string) ([]*net.IPNet, error) {
	var ranges []*net.IPNet
	for part := range strings.SplitSeq(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		_, ipNet, err := net.ParseCIDR(part)
		if err != nil {
			return nil, fmt.Errorf("trusted proxy %q: %w", part, err)
		}
		ranges = append(ranges, ipNet)
	}
	return ranges, nil
}

// ClientIP — RemoteAddr, а X-Forwarded-For читается справа налево только от доверенных прокси.
// Кривое значение в цепочке или недоверенный пир дают RemoteAddr.
func ClientIP(trusted []*net.IPNet) func(*http.Request) string {
	isTrusted := func(ip net.IP) bool {
		for _, n := range trusted {
			if n.Contains(ip) {
				return true
			}
		}

		return false
	}

	return func(r *http.Request) string {
		direct := remoteIP(r.RemoteAddr)
		xffs := r.Header.Values("X-Forwarded-For")
		if len(trusted) == 0 || len(xffs) == 0 {
			return direct
		}

		ips := append(strings.Split(strings.Join(xffs, ","), ","), direct)
		for i := len(ips) - 1; i >= 0; i-- {
			ips[i] = strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(ips[i]), "["), "]")
			ip := net.ParseIP(ips[i])
			if ip == nil {
				return direct
			}
			if !isTrusted(ip) {
				return ip.String()
			}
		}

		return ips[0]
	}
}

func remoteIP(addr string) string {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		if net.ParseIP(addr) != nil {
			return addr
		}

		return ""
	}

	return host
}
