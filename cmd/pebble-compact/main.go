package main

import (
	"bytes"
	"flag"
	"log"
	"runtime"
	"time"

	"github.com/cockroachdb/pebble"
	"github.com/cockroachdb/pebble/bloom"
)

var dbPath = flag.String("db", "", "path to the pebble db path")

func main() {
	flag.Parse()

	if *dbPath == "" {
		log.Fatal("db path is required")
	}

	opt := &pebble.Options{
		Cache:                       pebble.NewCache(int64(4 << 30)),
		MaxOpenFiles:                16384,
		MemTableSize:                uint64(1 << 30),
		MemTableStopWritesThreshold: 2,
		MaxConcurrentCompactions:    runtime.NumCPU,
		Levels:                      make([]pebble.LevelOptions, 7),
	}
	for i := range opt.Levels {
		l := &opt.Levels[i]
		l.FilterPolicy = bloom.FilterPolicy(10)
		if i > 0 {
			l.TargetFileSize = opt.Levels[i-1].TargetFileSize * 2
		}
		l.EnsureDefaults()
	}
	opt.Experimental.ReadSamplingMultiplier = -1

	db, err := pebble.Open(*dbPath, opt)
	if err != nil {
		log.Fatalf("failed to open db: %v", err)
	}
	limit := bytes.Repeat([]byte{0xff}, 32)

	metrics := db.Metrics()
	log.Printf("Before compaction metrics: \n%v\n", metrics)
	log.Println("Compacting the database")
	st := time.Now()
	if err := db.Compact(nil, limit, true); err != nil {
		log.Fatalf("failed to compact db: %v", err)
	}
	log.Printf("Compaction took %v\n", time.Since(st))
	metrics = db.Metrics()
	log.Printf("After compaction metrics: \n%v\n", metrics)
}
