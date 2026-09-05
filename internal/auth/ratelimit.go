package auth

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
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
	ips    map[string][]time.Time
	emails map[string][]time.Time
	calls  int
}

// NewRateLimiter создаёт лимитер; nil now — time.Now.
func NewRateLimiter(now func() time.Time) *RateLimiter {
	if now == nil {
		now = time.Now
	}
	return &RateLimiter{
		now:    now,
		ips:    map[string][]time.Time{},
		emails: map[string][]time.Time{},
	}
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

	ipHits := prune(l.ips[ip], now.Add(-IPWindow))
	emailHits := prune(l.emails[email], now.Add(-EmailWindow))

	if len(ipHits) >= IPLimit {
		return retryAfter(ipHits[0], IPWindow, now), false
	}
	if len(emailHits) >= EmailLimit {
		return retryAfter(emailHits[0], EmailWindow, now), false
	}

	l.ips[ip] = append(ipHits, now)
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

// IPExtractor — RemoteAddr, а X-Forwarded-For учитывается только от доверенных прокси.
// Без доверенных сетей заголовок игнорируется целиком, иначе клиент подменял бы IP для лимитера.
func IPExtractor(trusted []*net.IPNet) echo.IPExtractor {
	if len(trusted) == 0 {
		return echo.ExtractIPDirect()
	}
	opts := []echo.TrustOption{
		echo.TrustLoopback(false),
		echo.TrustLinkLocal(false),
		echo.TrustPrivateNet(false),
	}
	for _, ipNet := range trusted {
		opts = append(opts, echo.TrustIPRange(ipNet))
	}
	return echo.ExtractIPFromXFFHeader(opts...)
}
