package metrics

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// ScrapeTimeout ограничивает работу коллектора на скрейпе: HTTP-таймаут promhttp
// не прерывает ни SQL, ни чтение каталога внутри Collect.
const ScrapeTimeout = 5 * time.Second

// BackupLister отдаёт времена публикации всех копий в каталоге бэкапов.
type BackupLister interface {
	ListBackupTimes(ctx context.Context) ([]time.Time, error)
}

// backupDirCollector читает каталог бэкапов на скрейпе — так он видит и копии cron-процесса.
type backupDirCollector struct {
	lister BackupLister
	latest *prometheus.Desc
	files  *prometheus.Desc
}

// NewBackupDirCollector собирает свежесть и число файлов в каталоге бэкапов.
func NewBackupDirCollector(lister BackupLister) prometheus.Collector {
	return &backupDirCollector{
		lister: lister,
		latest: prometheus.NewDesc(
			namespace+"_backup_latest_file_timestamp_seconds",
			"Время новейшего файла копии (unix-секунды); 0 — файлов нет.",
			nil, nil,
		),
		files: prometheus.NewDesc(
			namespace+"_backup_files",
			"Число файлов копий в каталоге бэкапов.",
			nil, nil,
		),
	}
}

func (c *backupDirCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.latest
	ch <- c.files
}

func (c *backupDirCollector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(context.Background(), ScrapeTimeout)
	defer cancel()

	times, err := c.lister.ListBackupTimes(ctx)
	if err != nil {
		ch <- prometheus.NewInvalidMetric(c.latest, err)
		ch <- prometheus.NewInvalidMetric(c.files, err)
		return
	}

	var latest float64
	for _, t := range times {
		if seconds := float64(t.Unix()); seconds > latest {
			latest = seconds
		}
	}

	ch <- prometheus.MustNewConstMetric(c.latest, prometheus.GaugeValue, latest)
	ch <- prometheus.MustNewConstMetric(c.files, prometheus.GaugeValue, float64(len(times)))
}
