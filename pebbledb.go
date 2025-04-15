package bench

import (
	"time"

	"github.com/cockroachdb/pebble"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	compCount = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "compact_count",
		Help: "The total number of compactions",
	})
	compReadCount = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "compact_read_count",
		Help: "The total number of compaction reads",
	})
	compMoveCount = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "compact_move_count",
		Help: "The total number of compaction moves",
	})
	compRewriteCount = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "compact_rewrite_count",
		Help: "The total number of compaction rewrites",
	})
	compMultilevelCount = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "compact_multilevel_count",
		Help: "The total number of compaction moves",
	})
	compEstimatedDebt = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "compact_estimated_debt",
		Help: "The estimated debt of the compaction",
	})
	compMarkedFiles = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "compact_marked_files",
		Help: "The number of marked files",
	})
	compDuration = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "compact_duration",
		Help: "The total duration of compactions",
	})
	diskSize = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pebble_disk_size",
		Help: "The size of the database on disk",
	})
)

func CollectPebbleMetrics(db *pebble.DB, done chan struct{}) {
	timer := time.NewTicker(5 * time.Second)
	defer timer.Stop()

	for {
		select {
		case <-done:
			return
		case <-timer.C:
			stats := db.Metrics()
			compCount.Set(float64(stats.Compact.Count))
			compReadCount.Set(float64(stats.Compact.ReadCount))
			compMoveCount.Set(float64(stats.Compact.MoveCount))
			compRewriteCount.Set(float64(stats.Compact.RewriteCount))
			compMultilevelCount.Set(float64(stats.Compact.MultiLevelCount))
			compEstimatedDebt.Set(float64(stats.Compact.EstimatedDebt))
			compMarkedFiles.Set(float64(stats.Compact.MarkedFiles))
			compDuration.Set(float64(stats.Compact.Duration.Seconds()))
			diskSize.Set(float64(stats.DiskSpaceUsage()))
		}
	}
}
