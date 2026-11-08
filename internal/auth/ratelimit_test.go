package auth_test

import (
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/auth"
)

const (
	limiterIP    = "203.0.113.7"
	limiterEmail = "admin@example.com"
)

// fakeClock — управляемое время для лимитера.
type fakeClock struct{ t time.Time }

func (c *fakeClock) Now() time.Time          { return c.t }
func (c *fakeClock) Advance(d time.Duration) { c.t = c.t.Add(d) }
func newFakeClock() *fakeClock               { return &fakeClock{t: time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)} }
func exhaust(l *auth.RateLimiter, email string, n int) {
	for range n {
		l.Allow(limiterIP, email)
	}
}

func TestRateLimiter_IPLimit_BlocksEleventhAttempt(t *testing.T) {
	clock := newFakeClock()
	l := auth.NewRateLimiter(clock.Now)

	for i := range auth.IPLimit {
		_, ok := l.Allow(limiterIP, limiterEmail)
		require.True(t, ok, "попытка %d должна пройти", i+1)
	}

	retry, ok := l.Allow(limiterIP, limiterEmail)
	assert.False(t, ok)
	assert.Equal(t, auth.IPWindow, retry, "Retry-After — до выхода первой попытки из окна")
}

func TestRateLimiter_IPLimit_CountsAcrossEmails(t *testing.T) {
	l := auth.NewRateLimiter(newFakeClock().Now)

	for i := range auth.IPLimit {
		_, ok := l.Allow(limiterIP, "user"+string(rune('a'+i))+"@example.com")
		require.True(t, ok)
	}

	_, ok := l.Allow(limiterIP, "another@example.com")
	assert.False(t, ok, "перебор email с одного IP ограничен лимитом IP")
}

// Без доверенных прокси IP всех клиентов один и тот же: корзина per-IP выключена, email — нет.
func TestRateLimiter_WithoutIPLimit_OnlyEmailCounts(t *testing.T) {
	clock := newFakeClock()
	l := auth.NewRateLimiter(clock.Now, auth.WithoutIPLimit())

	for i := range auth.EmailLimit {
		_, ok := l.Allow(limiterIP, limiterEmail)
		require.True(t, ok, "попытка %d с одного IP должна пройти", i+1)
	}

	retry, ok := l.Allow(limiterIP, limiterEmail)
	assert.False(t, ok)
	assert.Equal(t, auth.EmailWindow, retry, "блок — по email, а не по IP")
	_, ok = l.Allow(limiterIP, "other@example.com")
	assert.True(t, ok, "другой email с того же IP не заблокирован")
}

func TestRateLimiter_WindowSlides(t *testing.T) {
	clock := newFakeClock()
	l := auth.NewRateLimiter(clock.Now)

	exhaust(l, limiterEmail, 1)
	clock.Advance(time.Minute)
	exhaust(l, limiterEmail, auth.IPLimit-1)

	retry, ok := l.Allow(limiterIP, limiterEmail)
	require.False(t, ok)
	assert.Equal(t, auth.IPWindow-time.Minute, retry)

	clock.Advance(auth.IPWindow - time.Minute)
	_, ok = l.Allow(limiterIP, limiterEmail)
	assert.True(t, ok, "первая попытка вышла из окна — одна попытка освободилась")
	_, ok = l.Allow(limiterIP, limiterEmail)
	assert.False(t, ok, "остальные девять ещё в окне")
}

func TestRateLimiter_BlockedAttemptIsNotRecorded(t *testing.T) {
	clock := newFakeClock()
	l := auth.NewRateLimiter(clock.Now)

	exhaust(l, limiterEmail, auth.IPLimit)
	clock.Advance(auth.IPWindow - time.Second)
	_, ok := l.Allow(limiterIP, limiterEmail)
	require.False(t, ok)

	clock.Advance(time.Second)
	_, ok = l.Allow(limiterIP, limiterEmail)
	assert.True(t, ok, "заблокированная попытка не должна продлевать окно")
}

func TestRateLimiter_EmailLimit_BlocksAcrossIPs(t *testing.T) {
	clock := newFakeClock()
	l := auth.NewRateLimiter(clock.Now)

	for i := range auth.EmailLimit {
		ip := net.IPv4(10, 0, 0, byte(i+1)).String()
		_, ok := l.Allow(ip, limiterEmail)
		require.True(t, ok, "попытка %d с нового IP должна пройти", i+1)
	}

	retry, ok := l.Allow("10.0.1.1", limiterEmail)
	assert.False(t, ok)
	assert.Equal(t, auth.EmailWindow, retry)
}

func TestRateLimiter_Reset_LiftsEmailBlockOnly(t *testing.T) {
	clock := newFakeClock()
	l := auth.NewRateLimiter(clock.Now)

	for i := range auth.EmailLimit {
		l.Allow(net.IPv4(10, 0, 0, byte(i+1)).String(), limiterEmail)
	}
	_, ok := l.Allow("10.0.1.1", limiterEmail)
	require.False(t, ok)

	l.Reset(limiterEmail)

	_, ok = l.Allow("10.0.1.1", limiterEmail)
	assert.True(t, ok, "успешный вход снимает блок по email")

	exhaust(l, "other@example.com", auth.IPLimit)
	l.Reset("other@example.com")
	_, ok = l.Allow(limiterIP, "other@example.com")
	assert.False(t, ok, "Reset не трогает счётчик IP")
}

func TestRateLimiter_Cleanup_KeepsBlockingWithinWindow(t *testing.T) {
	clock := newFakeClock()
	l := auth.NewRateLimiter(clock.Now)

	exhaust(l, limiterEmail, auth.IPLimit)
	// Много вызовов с других ключей прогоняют уборку; заблокированный IP остаётся в окне.
	for i := range 250 {
		l.Allow(net.IPv4(192, 0, 2, byte(i%200)).String(), "x@example.com")
	}

	_, ok := l.Allow(limiterIP, limiterEmail)
	assert.False(t, ok)
}

// Retry-After никогда не 0: за миллисекунду до конца окна ответ — секунда.
func TestRateLimiter_RetryAfter_ClampsToOneSecond(t *testing.T) {
	clock := newFakeClock()
	l := auth.NewRateLimiter(clock.Now)

	exhaust(l, limiterEmail, auth.IPLimit)
	clock.Advance(auth.IPWindow - time.Millisecond)

	retry, ok := l.Allow(limiterIP, limiterEmail)
	require.False(t, ok)
	assert.Equal(t, time.Second, retry)

	clock.Advance(-time.Second - 400*time.Millisecond)
	retry, ok = l.Allow(limiterIP, limiterEmail)
	require.False(t, ok)
	assert.Equal(t, time.Second, retry, "1.4s округляется до секунды")
}

// Карты лимитера под мьютексом: параллельные Allow с одного IP пропускают ровно IPLimit попыток.
func TestRateLimiter_Concurrent(t *testing.T) {
	l := auth.NewRateLimiter(nil)
	const workers = 50

	var wg sync.WaitGroup
	var allowed atomic.Int32
	for i := range workers {
		wg.Go(func() {
			if _, ok := l.Allow(limiterIP, "user"+strconv.Itoa(i)+"@example.com"); ok {
				allowed.Add(1)
			}
			l.Reset(limiterEmail)
		})
	}
	wg.Wait()

	assert.Equal(t, int32(auth.IPLimit), allowed.Load())
}

func TestParseTrustedProxies(t *testing.T) {
	ranges, err := auth.ParseTrustedProxies(" 10.0.0.0/8, 172.16.0.0/12 ,,")
	require.NoError(t, err)
	require.Len(t, ranges, 2)
	assert.Equal(t, "10.0.0.0/8", ranges[0].String())

	empty, err := auth.ParseTrustedProxies("")
	require.NoError(t, err)
	assert.Empty(t, empty)

	_, err = auth.ParseTrustedProxies("10.0.0.1")
	require.Error(t, err, "адрес без маски — не CIDR")
}

func TestClientIP(t *testing.T) {
	trusted, err := auth.ParseTrustedProxies("10.0.0.0/8")
	require.NoError(t, err)

	cases := map[string]struct {
		trusted []*net.IPNet
		remote  string
		xff     []string
		want    string
	}{
		"no trusted proxies ignores XFF": {nil, "10.0.0.5:4321", []string{"198.51.100.9"}, "10.0.0.5"},
		"no XFF":                         {trusted, "10.0.0.5:4321", nil, "10.0.0.5"},
		"XFF from trusted proxy":         {trusted, "10.0.0.5:4321", []string{"198.51.100.9"}, "198.51.100.9"},
		"XFF from untrusted peer":        {trusted, "192.168.1.5:4321", []string{"198.51.100.9"}, "192.168.1.5"},
		"chain stops at untrusted hop": {
			trusted, "10.0.0.5:4321", []string{"198.51.100.9, 192.168.1.5"}, "192.168.1.5",
		},
		"several headers are one chain": {
			trusted, "10.0.0.5:4321", []string{"198.51.100.9", "10.0.0.7"}, "198.51.100.9",
		},
		"untrusted hop in the middle": {
			trusted, "10.0.0.5:4321", []string{"203.0.113.1, 198.51.100.9, 10.0.0.7"}, "198.51.100.9",
		},
		"malformed value":             {trusted, "10.0.0.5:4321", []string{"not-an-ip"}, "10.0.0.5"},
		"all hops trusted, leftmost":  {trusted, "10.0.0.5:4321", []string{"10.0.0.2, 10.0.0.3"}, "10.0.0.2"},
		"private outside CIDR":        {trusted, "10.0.0.5:4321", []string{"192.168.7.7, 10.0.0.9"}, "192.168.7.7"},
		"bracketed IPv6":              {trusted, "10.0.0.5:4321", []string{"[2001:db8::1]"}, "2001:db8::1"},
		"RemoteAddr without a port":   {nil, "203.0.113.4", nil, "203.0.113.4"},
		"RemoteAddr that is no IP":    {nil, "garbage", nil, ""},
		"IPv6 peer without trust set": {nil, "[2001:db8::2]:80", nil, "2001:db8::2"},
		"empty XFF":                   {trusted, "10.0.0.5:4321", []string{""}, "10.0.0.5"},
		"comma-only XFF":              {trusted, "10.0.0.5:4321", []string{","}, "10.0.0.5"},
		"spoofed leftmost entries": {
			trusted, "10.0.0.5:4321", []string{"6.6.6.6, 7.7.7.7, 198.51.100.9, 10.0.0.2"}, "198.51.100.9",
		},
		"IPv4-mapped trusted peer": {
			trusted, "[::ffff:10.0.0.5]:4321", []string{"198.51.100.9"}, "198.51.100.9",
		},
		"IPv4-mapped untrusted peer": {
			trusted, "[::ffff:192.168.1.5]:4321", []string{"6.6.6.6"}, "192.168.1.5",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tc.remote
			for _, v := range tc.xff {
				req.Header.Add("X-Forwarded-For", v)
			}
			assert.Equal(t, tc.want, auth.ClientIP(tc.trusted)(req))
		})
	}
}
