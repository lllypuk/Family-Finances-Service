package metrics

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// StateReader считает состояние базы на скрейпе.
type StateReader interface {
	ActiveSessions(ctx context.Context, now time.Time) (int, error)
	UsersByActive(ctx context.Context) (active, inactive int, err error)
	Transactions(ctx context.Context) (int, error)
	SetupComplete(ctx context.Context) (bool, error)
}

// stateCollector отдаёт размер файлов БД и счётчики строк на момент скрейпа.
type stateCollector struct {
	reader        StateReader
	dbPath        string
	dbSize        *prometheus.Desc
	sessions      *prometheus.Desc
	users         *prometheus.Desc
	transactions  *prometheus.Desc
	setupComplete *prometheus.Desc
}

// NewStateCollector собирает состояние базы: размер файлов, сессии, пользователи, операции, setup.
func NewStateCollector(reader StateReader, dbPath string) prometheus.Collector {
	return &stateCollector{
		reader: reader,
		dbPath: dbPath,
		dbSize: prometheus.NewDesc(
			namespace+"_db_size_bytes",
			"Размер файлов базы в байтах; отсутствующий файл — 0.",
			[]string{"file"}, nil,
		),
		sessions: prometheus.NewDesc(
			namespace+"_sessions_active",
			"Число непросроченных сессий.",
			nil, nil,
		),
		users: prometheus.NewDesc(
			namespace+"_users",
			"Число пользователей по признаку активности.",
			[]string{"active"}, nil,
		),
		transactions: prometheus.NewDesc(
			namespace+"_transactions",
			"Число операций в базе.",
			nil, nil,
		),
		setupComplete: prometheus.NewDesc(
			namespace+"_setup_complete",
			"1 — семья создана (cmd/server setup), 0 — нет.",
			nil, nil,
		),
	}
}

func (c *stateCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.dbSize
	ch <- c.sessions
	ch <- c.users
	ch <- c.transactions
	ch <- c.setupComplete
}

func (c *stateCollector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(context.Background(), ScrapeTimeout)
	defer cancel()

	c.collectDBSize(ch)
	c.collectSessions(ctx, ch)
	c.collectUsers(ctx, ch)
	c.collectTransactions(ctx, ch)
	c.collectSetup(ctx, ch)
}

func (c *stateCollector) collectDBSize(ch chan<- prometheus.Metric) {
	for _, f := range []struct{ label, path string }{
		{"main", c.dbPath},
		{"wal", c.dbPath + "-wal"},
	} {
		size, err := fileSize(f.path)
		if err != nil {
			ch <- prometheus.NewInvalidMetric(c.dbSize, err)
			continue
		}
		ch <- prometheus.MustNewConstMetric(c.dbSize, prometheus.GaugeValue, size, f.label)
	}
}

func (c *stateCollector) collectSessions(ctx context.Context, ch chan<- prometheus.Metric) {
	count, err := c.reader.ActiveSessions(ctx, time.Now())
	if err != nil {
		ch <- prometheus.NewInvalidMetric(c.sessions, err)
		return
	}
	ch <- prometheus.MustNewConstMetric(c.sessions, prometheus.GaugeValue, float64(count))
}

func (c *stateCollector) collectUsers(ctx context.Context, ch chan<- prometheus.Metric) {
	active, inactive, err := c.reader.UsersByActive(ctx)
	if err != nil {
		ch <- prometheus.NewInvalidMetric(c.users, err)
		return
	}
	ch <- prometheus.MustNewConstMetric(c.users, prometheus.GaugeValue, float64(active), "true")
	ch <- prometheus.MustNewConstMetric(c.users, prometheus.GaugeValue, float64(inactive), "false")
}

func (c *stateCollector) collectTransactions(ctx context.Context, ch chan<- prometheus.Metric) {
	count, err := c.reader.Transactions(ctx)
	if err != nil {
		ch <- prometheus.NewInvalidMetric(c.transactions, err)
		return
	}
	ch <- prometheus.MustNewConstMetric(c.transactions, prometheus.GaugeValue, float64(count))
}

func (c *stateCollector) collectSetup(ctx context.Context, ch chan<- prometheus.Metric) {
	complete, err := c.reader.SetupComplete(ctx)
	if err != nil {
		ch <- prometheus.NewInvalidMetric(c.setupComplete, err)
		return
	}
	var value float64
	if complete {
		value = 1
	}
	ch <- prometheus.MustNewConstMetric(c.setupComplete, prometheus.GaugeValue, value)
}

// fileSize: отсутствующий файл — не ошибка, WAL появляется только под нагрузкой.
func fileSize(path string) (float64, error) {
	info, err := os.Stat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return float64(info.Size()), nil
}
