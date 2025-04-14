package main

import (
	"context"
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	_ "net/http/pprof"

	"github.com/cockroachdb/pebble"
	bench "github.com/fjl/goleveldb-bench"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/sync/errgroup"
)

const (
	MiB = bench.MiB
	GiB = bench.GiB
)

func main() {
	var (
		testflag      = flag.String("test", "", "tests to run ("+strings.Join(testnames(), ", ")+")")
		prefixflag    = flag.String("prefix", "", "test name prefix")
		sizeflag      = flag.String("size", "500mb", "total amount of value data to write")
		valuedistflag = flag.String("value-dist", "3b:0.31,4b:0.16,1b:0.05,33b:0.05,83b:0.05,128b:0.32", "value size distribution (size:prob,size:prob,...)")
		keydistflag   = flag.String("key-dist", "33b:0.4,65b:0.2,38b:0.1,39b:0.1,65b:0.2", "key size distribution (size:prob,size:prob,...)")
		dirflag       = flag.String("dir", ".", "test database directory")
		logdirflag    = flag.String("logdir", ".", "test log output directory")
		keydirflag    = flag.String("keydir", "", "test keyfile directory")
		deletedbflag  = flag.Bool("deletedb", false, "delete databases after test run")
		metricsAddr   = flag.String("metrics-addr", ":2112", "The address to serve metrics on")

		run []string
		cfg bench.WriteConfig
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
		t = strings.TrimSpace(t)
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
	if cfg.ValueDist, err = bench.ParseSizeDistribution(*valuedistflag); err != nil {
		log.Fatal("-value-dist: ", err)
	}
	if cfg.KeyDist, err = bench.ParseSizeDistribution(*keydistflag); err != nil {
		log.Fatal("-key-dist: ", err)
	}
	cfg.LogPercent = true

	if err := os.MkdirAll(*logdirflag, 0755); err != nil {
		log.Fatal("can't create log dir: ", err)
	}

	anyErr := false
	for _, name := range run {
		dbdir := filepath.Join(*dirflag, "testdb-"+name)
		if err := runTest(*logdirflag, *keydirflag, dbdir, *prefixflag, name, cfg); err != nil {
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

func runTest(logdir, keydir, dbdir, prefix, name string, cfg bench.WriteConfig) error {
	cfg.TestName = prefix + name
	logfile, err := os.Create(filepath.Join(logdir, name+".json"))
	if err != nil {
		return err
	}
	defer logfile.Close()
	log.Printf("== running %q", name)

	var kw io.Writer
	if keydir != "" {
		if err := os.MkdirAll(keydir, 0755); err != nil {
			return err
		}

		keyfile := path.Join(keydir, name, "testing.key")
		// create the keyfile directory
		if err := os.MkdirAll(path.Join(keydir, name), 0755); err != nil {
			return err
		}
		var f *os.File
		if _, iErr := os.Stat(keyfile); os.IsNotExist(iErr) {
			f, err = os.Create(keyfile)
		} else {
			f, err = os.OpenFile(keyfile, os.O_APPEND|os.O_WRONLY, 0644)
		}
		if err != nil {
			return err
		}
		defer f.Close()
		kw = f
	}

	env := bench.NewWriteEnv(logfile, kw, nil, cfg)
	return tests[name].Benchmark(dbdir, env)
}

type Benchmarker interface {
	Benchmark(dir string, env *bench.WriteEnv) error
}

var tests = map[string]Benchmarker{
	"nobatch":            seqWrite{},
	"nobatch-nosync":     seqWrite{wOptions: pebble.NoSync},
	"batch-100kb":        &batchWrite{BatchSize: 100 * bench.KiB},
	"batch-1mb":          &batchWrite{BatchSize: bench.MiB},
	"batch-5mb":          &batchWrite{BatchSize: 5 * bench.MiB},
	"batch-100kb-nosync": &batchWrite{BatchSize: 100 * bench.KiB, wOptions: pebble.NoSync},
	"concurrent":         concurrentWrite{N: 8},

	"batch-100kb-mt-004mb-cache-1gb": newBatchWrite(4*MiB, 1*GiB),
	"batch-100kb-mt-008mb-cache-1gb": newBatchWrite(8*MiB, 1*GiB),
	"batch-100kb-mt-016mb-cache-1gb": newBatchWrite(16*MiB, 1*GiB),
	"batch-100kb-mt-064mb-cache-1gb": newBatchWrite(64*MiB, 1*GiB),
	"batch-100kb-mt-256mb-cache-1gb": newBatchWrite(256*MiB, 1*GiB),
	"batch-100kb-mt-512mb-cache-1gb": newBatchWrite(512*MiB, 1*GiB),
	"batch-100kb-mt-064mb-cache-4gb": newBatchWrite(64*MiB, 4*GiB),
	"batch-100kb-mt-256mb-cache-4gb": newBatchWrite(256*MiB, 4*GiB),
	"batch-100kb-mt-512mb-cache-4gb": newBatchWrite(512*MiB, 4*GiB),

	"batch-100kb-mt-1gb-cache-01gb": newBatchWrite(1*GiB, 1*GiB),
	"batch-100kb-mt-1gb-cache-04gb": newBatchWrite(1*GiB, 4*GiB),
	"batch-100kb-mt-1gb-cache-08gb": newBatchWrite(1*GiB, 8*GiB),
	"batch-100kb-mt-2gb-cache-08gb": newBatchWrite(2*GiB, 8*GiB),
	"batch-100kb-mt-4gb-cache-08gb": newBatchWrite(4*GiB, 8*GiB),
	"batch-100kb-mt-1gb-cache-16gb": newBatchWrite(1*GiB, 16*GiB),
	"batch-100kb-mt-2gb-cache-16gb": newBatchWrite(2*GiB, 16*GiB),
	"batch-100kb-mt-4gb-cache-16gb": newBatchWrite(4*GiB, 16*GiB),
	"batch-100kb-mt-1gb-cache-32gb": newBatchWrite(1*GiB, 32*GiB),
	"batch-100kb-mt-2gb-cache-32gb": newBatchWrite(2*GiB, 32*GiB),
	"batch-100kb-mt-4gb-cache-32gb": newBatchWrite(4*GiB, 32*GiB),

	"batch-100kb-mt-1gb-cache-04gb-stopwrite-4": newBatchWrite(1*GiB, 4*GiB).With(func(opt *pebble.Options) { opt.MemTableStopWritesThreshold = 4 }),
	"batch-100kb-mt-1gb-cache-04gb-compact-4":   newBatchWrite(1*GiB, 4*GiB).With(func(opt *pebble.Options) { opt.MaxConcurrentCompactions = func() int { return 4 } }),
	"batch-100kb-mt-1gb-cache-04gb-openfd-10k":  newBatchWrite(1*GiB, 4*GiB).With(func(opt *pebble.Options) { opt.MaxOpenFiles = 10_000 }),

	"batch-100kb-mt-1gb-cache-04gb-bytespersync-1mb":    newBatchWrite(1*GiB, 4*GiB).With(func(opt *pebble.Options) { opt.BytesPerSync = 1 * MiB }),
	"batch-100kb-mt-1gb-cache-04gb-walbytespersync-1mb": newBatchWrite(1*GiB, 4*GiB).With(func(opt *pebble.Options) { opt.WALBytesPerSync = 1 * MiB }),
	"batch-100kb-mt-1gb-cache-04gb-maxopenfiles-10k":    newBatchWrite(1*GiB, 4*GiB).With(func(opt *pebble.Options) { opt.MaxOpenFiles = 10_000 }),
	"batch-100kb-mt-1gb-cache-04gb-lbasemaxbytes-128mb": newBatchWrite(1*GiB, 4*GiB).With(func(opt *pebble.Options) { opt.LBaseMaxBytes = 128 * MiB }),
}

func testnames() (n []string) {
	for name := range tests {
		n = append(n, name)
	}
	sort.Strings(n)
	return n
}

type seqWrite struct {
	Options  pebble.Options
	wOptions *pebble.WriteOptions
}

func (b seqWrite) Benchmark(dir string, env *bench.WriteEnv) error {
	db, err := pebble.Open(dir, &b.Options)
	if err != nil {
		return err
	}
	defer db.Close()

	return env.Run(func(key, value string, lastCall bool) error {
		if err := db.Set([]byte(key), []byte(value), b.wOptions); err != nil {
			return err
		}
		env.Progress(len(key) + len(value))
		return nil
	})
}

type batchWrite struct {
	Options   *pebble.Options
	wOptions  *pebble.WriteOptions
	BatchSize int
}

func newBatchWrite(mem, cache int) *batchWrite {
	// MemTableSize must be < 4.0GB
	if mem >= 4*GiB {
		mem -= 2
	}
	return &batchWrite{
		BatchSize: 100 * bench.KiB,
		Options: &pebble.Options{
			Cache:        pebble.NewCache(int64(cache)),
			MemTableSize: uint64(mem),
		},
	}
}

func (b *batchWrite) With(customize func(opt *pebble.Options)) *batchWrite {
	customize(b.Options)
	return b
}

func (b *batchWrite) Benchmark(dir string, env *bench.WriteEnv) error {
	// l := pebble.MakeLoggingEventListener(nil)
	// b.Options.EventListener = &l
	db, err := pebble.Open(dir, b.Options)
	if err != nil {
		return err
	}
	defer db.Close()

	batch := db.NewBatch()
	bsize := 0
	return env.Run(func(key, value string, lastCall bool) error {
		batch.Set([]byte(key), []byte(value), nil)
		bsize += len(key) + len(value)
		if bsize >= b.BatchSize || lastCall {
			if err := batch.Commit(b.wOptions); err != nil {
				return err
			}
			env.Progress(bsize)
			bsize = 0
			batch.Reset()
		}
		return nil
	})
}

type kv struct{ k, v string }

type concurrentWrite struct {
	Options pebble.Options
	N       int
}

func (b concurrentWrite) Benchmark(dir string, env *bench.WriteEnv) error {
	db, err := pebble.Open(dir, &b.Options)
	if err != nil {
		return err
	}
	defer db.Close()

	var (
		write            = make(chan kv, b.N)
		outerCtx, cancel = context.WithCancel(context.Background())
		eg, ctx          = errgroup.WithContext(outerCtx)
	)
	for range b.N {
		eg.Go(func() error {
			for {
				select {
				case kv := <-write:
					if err := db.Set([]byte(kv.k), []byte(kv.v), nil); err != nil {
						return err
					}
					env.Progress(len(kv.k) + len(kv.v))
				case <-ctx.Done():
					return nil
				}
			}
		})
	}

	return env.Run(func(key, value string, lastCall bool) error {
		select {
		case write <- kv{k: key, v: value}:
		case <-ctx.Done():
			lastCall = true
		}
		if lastCall {
			cancel()
			return eg.Wait()
		}
		return nil
	})
}
