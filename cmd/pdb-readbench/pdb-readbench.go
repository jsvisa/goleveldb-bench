package main

import (
	"flag"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"net/http"
	_ "net/http/pprof"

	"github.com/cockroachdb/pebble"
	"github.com/cockroachdb/pebble/bloom"
	bench "github.com/fjl/goleveldb-bench"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	var (
		testflag     = flag.String("test", "", "tests to run ("+strings.Join(testnames(), ", ")+")")
		prefixflag   = flag.String("prefix", "", "test name prefix")
		sizeflag     = flag.String("size", "500mb", "total amount of value data to write")
		datasizeflag = flag.String("valuesize", "128kb", "maximum length of the value, simulate as the block or state data")
		keysizeflag  = flag.String("keysize", "32b", "size of each key")
		dirflag      = flag.String("dir", ".", "test database directory")
		logdirflag   = flag.String("logdir", ".", "test log output directory")
		keydirflag   = flag.String("keydir", ".", "test keyfile directory")
		randomflag   = flag.Float64("keyrandom", 10, "random key distribution")
		deletedbflag = flag.Bool("deletedb", false, "delete databases after test run")
		metricsAddr  = flag.String("metrics-addr", ":2112", "The address to serve metrics on")

		run []string
		cfg bench.ReadConfig
		err error
	)
	flag.Parse()

	// Start metrics server
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		log.Printf("Starting metrics server on %s", *metricsAddr)
		if err := http.ListenAndServe(*metricsAddr, nil); err != nil {
			log.Printf("Metrics server error: %v", err)
		}
	}()

	for _, t := range strings.Split(*testflag, ",") {
		if tests[t] == nil {
			log.Fatalf("unknown test %q", t)
		}
		run = append(run, t)
	}
	if len(run) == 0 {
		log.Fatal("no tests to run, use -test to select tests")
	}
	if cfg.Size, err = bench.ParseSize(*sizeflag); err != nil {
		log.Fatal("-size: ", err)
	}
	if cfg.DataSize, err = bench.ParseSize(*datasizeflag); err != nil {
		log.Fatal("-datasize: ", err)
	}
	if cfg.KeySize, err = bench.ParseSize(*keysizeflag); err != nil {
		log.Fatal("-keysize: ", err)
	}
	cfg.LogPercent = true
	cfg.RandomPercent = *randomflag

	if err := os.MkdirAll(*logdirflag, 0755); err != nil {
		log.Fatalf("can't create log dir: %v", err)
	}

	anyErr := false
	for _, name := range run {
		var (
			dbdir    string
			createdb bool
		)
		// The given dir points to an existent directory, assume it's
		// an old database for read testing.
		if isDir(*dirflag) && fileExist(filepath.Join(*keydirflag, "testing.key")) {
			dbdir = *dirflag
		} else {
			dbdir, createdb = filepath.Join(*dirflag, "testdb-"+name), true
		}
		if err := os.MkdirAll(dbdir, 0755); err != nil {
			log.Fatal("can't create keyfile dir: ", err)
		}
		if err := runTest(*logdirflag, *keydirflag, dbdir, *prefixflag, name, createdb, cfg); err != nil {
			log.Printf("test %q failed: %v", name, err)
			anyErr = true
		}
		if *deletedbflag {
			os.RemoveAll(dbdir)
		}
	}
	if anyErr {
		log.Fatal("one ore more tests failed")
	}
}

func runTest(logdir, keydir, dbdir, prefix, name string, createdb bool, cfg bench.ReadConfig) error {
	cfg.TestName = prefix + name
	logfile, err := os.Create(filepath.Join(logdir, name+time.Now().Format(".2006-01-02-15:04:05")+".json"))
	if err != nil {
		return err
	}
	defer logfile.Close()

	var (
		kw    io.Writer
		kr    io.Reader
		kfile = filepath.Join(keydir, "testing.key")
	)
	kf, err := os.OpenFile(kfile, os.O_APPEND|os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer kf.Close()
	kw = kf

	kf, err = os.OpenFile(kfile, os.O_RDONLY, 0644)
	if err != nil {
		return err
	}
	defer kf.Close()
	kr = kf
	reset := func() { kf.Seek(0, io.SeekStart) }

	log.Printf("== running %q", name)
	env := bench.NewReadEnv(logfile, kr, kw, reset, cfg)
	return tests[name].Benchmark(dbdir, env)
}

type Benchmarker interface {
	Benchmark(dir string, env *bench.ReadEnv) error
}

var tests = map[string]Benchmarker{
	"random-read": &randomRead{},
	"random-read-filter": &randomRead{Options: &pebble.Options{
		Levels: []pebble.LevelOptions{{TargetFileSize: 2 * 1024 * 1024, FilterPolicy: bloom.FilterPolicy(10)}},
	}},
	"random-read-bigcache-filter": &randomRead{Options: &pebble.Options{
		Levels: []pebble.LevelOptions{{TargetFileSize: 2 * 1024 * 1024, FilterPolicy: bloom.FilterPolicy(10)}},
		Cache:  pebble.NewCache(int64(10 * bench.GiB)),
	}},

	"geth-read-cache-02gb": newGethRead(2 * bench.GiB),
	"geth-read-cache-20gb": newGethRead(20 * bench.GiB),

	"random-read-cache-02gb": newRandomRead(2 * bench.GiB),
	"random-read-cache-04gb": newRandomRead(4 * bench.GiB),
	"random-read-cache-08gb": newRandomRead(8 * bench.GiB),
	"random-read-cache-10gb": newRandomRead(10 * bench.GiB),
	"random-read-cache-20gb": newRandomRead(20 * bench.GiB),
	"random-read-cache-30gb": newRandomRead(30 * bench.GiB),

	"pebble-read": newPebbleRead(1 * bench.GiB),

	"pebble-read-cache-02gb": newPebbleRead(2 * bench.GiB),
	"pebble-read-cache-04gb": newPebbleRead(4 * bench.GiB),
	"pebble-read-cache-08gb": newPebbleRead(8 * bench.GiB),
	"pebble-read-cache-10gb": newPebbleRead(10 * bench.GiB),
	"pebble-read-cache-20gb": newPebbleRead(20 * bench.GiB),
	"pebble-read-cache-30gb": newPebbleRead(30 * bench.GiB),
}

func testnames() (n []string) {
	for name := range tests {
		n = append(n, name)
	}
	sort.Strings(n)
	return n
}

type randomRead struct {
	Options *pebble.Options
}

func newGethRead(cache int) *randomRead {
	opt := &pebble.Options{
		// Pebble has a single combined cache area and the write
		// buffers are taken from this too. Assign all available
		// memory allowance for cache.
		Cache: pebble.NewCache(int64(cache)),

		// The size of memory table(as well as the write buffer).
		// Note, there may have more than two memory tables in the system.
		MemTableSize: uint64(4*bench.GiB - 2),

		// MemTableStopWritesThreshold places a hard limit on the size
		// of the existent MemTables(including the frozen one).
		// Note, this must be the number of tables not the size of all memtables
		// according to https://github.com/cockroachdb/pebble/blob/master/options.go#L738-L742
		// and to https://github.com/cockroachdb/pebble/blob/master/db.go#L1892-L1903.
		MemTableStopWritesThreshold: 2,

		// The default compaction concurrency(1 thread),
		// Here use all available CPUs for faster compaction.
		MaxConcurrentCompactions: runtime.NumCPU,

		// Per-level options. Options for at least one level must be specified. The
		// options for the last level are used for all subsequent levels.
		Levels: []pebble.LevelOptions{
			{TargetFileSize: 2 * 1024 * 1024, FilterPolicy: bloom.FilterPolicy(10)},
			{TargetFileSize: 4 * 1024 * 1024, FilterPolicy: bloom.FilterPolicy(10)},
			{TargetFileSize: 8 * 1024 * 1024, FilterPolicy: bloom.FilterPolicy(10)},
			{TargetFileSize: 16 * 1024 * 1024, FilterPolicy: bloom.FilterPolicy(10)},
			{TargetFileSize: 32 * 1024 * 1024, FilterPolicy: bloom.FilterPolicy(10)},
			{TargetFileSize: 64 * 1024 * 1024, FilterPolicy: bloom.FilterPolicy(10)},
			{TargetFileSize: 128 * 1024 * 1024, FilterPolicy: bloom.FilterPolicy(10)},
		},
	}
	opt.Experimental.ReadSamplingMultiplier = -1
	return &randomRead{
		Options: opt,
	}
}

func newPebbleRead(cache int) *randomRead {
	return newRandomRead(cache).With(func(opt *pebble.Options) {
		opt.L0CompactionThreshold = 2
		opt.L0StopWritesThreshold = 1000
		opt.LBaseMaxBytes = 64 << 20 // 64 MB
		opt.Levels = make([]pebble.LevelOptions, 7)
		opt.MaxOpenFiles = 16384
		opt.MemTableSize = 64 << 20
		opt.MemTableStopWritesThreshold = 4
		opt.MaxConcurrentCompactions = func() int { return 3 }

		for i := 0; i < len(opt.Levels); i++ {
			l := &opt.Levels[i]
			l.BlockSize = 32 << 10       // 32 KB
			l.IndexBlockSize = 256 << 10 // 256 KB
			l.FilterPolicy = bloom.FilterPolicy(10)
			l.FilterType = pebble.TableFilter
			if i > 0 {
				l.TargetFileSize = opt.Levels[i-1].TargetFileSize * 2
			}
			l.EnsureDefaults()
		}
		opt.Levels[6].FilterPolicy = nil
		opt.FlushSplitBytes = opt.Levels[0].TargetFileSize
		opt.EnsureDefaults()
	})
}

func newRandomRead(cache int) *randomRead {
	return &randomRead{
		Options: &pebble.Options{
			Cache: pebble.NewCache(int64(cache)),
		},
	}
}

func (b *randomRead) With(customize func(opt *pebble.Options)) *randomRead {
	customize(b.Options)
	return b
}

func (b *randomRead) Benchmark(dir string, env *bench.ReadEnv) error {
	db, err := pebble.Open(dir, b.Options)
	if err != nil {
		return err
	}
	defer db.Close()

	// limit := bytes.Repeat([]byte{0xff}, 32)
	// db.Compact(nil, limit, true)

	done := make(chan struct{})
	go bench.CollectPebbleMetrics(db, done)
	defer func() { done <- struct{}{} }()

	batch := db.NewBatch()
	bsize := 0
	return env.Run(func(key, value string, lastCall bool) error {
		batch.Set([]byte(key), []byte(value), nil)
		bsize += len(value)
		if bsize >= 100*bench.KiB || lastCall {
			if err := batch.Commit(nil); err != nil {
				return err
			}
			bsize = 0
			batch.Reset()
		}
		return nil
	}, func(key string) error {
		value, closer, err := db.Get([]byte(key))
		if err != nil {
			return err
		} else if err := closer.Close(); err != nil {
			return err
		}
		env.Progress(len(value))
		return nil
	})
}

func fileExist(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func isDir(name string) bool {
	f, err := os.Stat(name)
	if err != nil {
		return false
	}
	return f.Mode().IsDir()
}
