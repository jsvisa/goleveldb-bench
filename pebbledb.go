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

	tableCacheCount = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pebble_table_cache_count",
		Help: "The number of table cache entries",
	})
	tableCacheSize = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pebble_table_cache_size",
		Help: "The size of the table cache",
	})
	tableCacheHits = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pebble_table_cache_hits",
		Help: "The number of table cache hits",
	})
	tableCacheMiss = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pebble_table_cache_miss",
		Help: "The number of table cache misses",
	})
	blockCacheCount = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pebble_block_cache_count",
		Help: "The number of block cache entries",
	})
	blockCacheSize = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pebble_block_cache_size",
		Help: "The size of the block cache",
	})
	blockCacheHits = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pebble_block_cache_hits",
		Help: "The number of block cache hits",
	})
	blockCacheMiss = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pebble_block_cache_miss",
		Help: "The number of block cache misses",
	})
	filterHits = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pebble_filter_hits",
		Help: "The number of filter hits",
	})
	filterMiss = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pebble_filter_miss",
		Help: "The number of filter misses",
	})

	memtableCount = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pebble_memtable_count",
		Help: "The number of memtables",
	})
	memtableSize = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pebble_memtable_size",
		Help: "The size of the memtable",
	})
	memtableZombieSize = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pebble_memtable_zombie_size",
		Help: "The size of the zombie memtable",
	})
	memtableZombieCount = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pebble_memtable_zombie_count",
		Help: "The number of zombie memtables",
	})
	tableObsoleteSize = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pebble_table_obsolete_size",
		Help: "The size of the obsolete tables",
	})
	tableObsoleteCount = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pebble_table_obsolete_count",
		Help: "The number of obsolete tables",
	})
	tableZombieSize = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pebble_table_zombie_size",
		Help: "The size of the zombie tables",
	})
	tableZombieCount = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pebble_table_zombie_count",
		Help: "The number of zombie tables",
	})
	tableBackingTableCount = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pebble_table_backing_table_count",
		Help: "The number of backing tables",
	})
	tableBackingTableSize = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pebble_table_backing_table_size",
		Help: "The size of the backing tables",
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
			tableCacheCount.Set(float64(stats.TableCache.Count))
			tableCacheSize.Set(float64(stats.TableCache.Size))
			tableCacheHits.Set(float64(stats.TableCache.Hits))
			tableCacheMiss.Set(float64(stats.TableCache.Misses))
			blockCacheCount.Set(float64(stats.BlockCache.Count))
			blockCacheSize.Set(float64(stats.BlockCache.Size))
			blockCacheHits.Set(float64(stats.BlockCache.Hits))
			blockCacheMiss.Set(float64(stats.BlockCache.Misses))
			filterHits.Set(float64(stats.Filter.Hits))
			filterMiss.Set(float64(stats.Filter.Misses))

			memtableCount.Set(float64(stats.MemTable.Count))
			memtableSize.Set(float64(stats.MemTable.Size))
			memtableZombieCount.Set(float64(stats.MemTable.ZombieCount))
			memtableZombieSize.Set(float64(stats.MemTable.ZombieSize))
			tableObsoleteCount.Set(float64(stats.Table.ObsoleteCount))
			tableObsoleteSize.Set(float64(stats.Table.ObsoleteSize))
			tableZombieCount.Set(float64(stats.Table.ZombieCount))
			tableZombieSize.Set(float64(stats.Table.ZombieSize))
			tableBackingTableCount.Set(float64(stats.Table.BackingTableCount))
			tableBackingTableSize.Set(float64(stats.Table.BackingTableSize))
		}
	}
}
