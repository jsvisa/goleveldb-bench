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

func main() {
	var (
		testflag     = flag.String("test", "", "tests to run ("+strings.Join(testnames(), ", ")+")")
		prefixflag   = flag.String("prefix", "", "test name prefix")
		sizeflag     = flag.String("size", "500mb", "total amount of value data to write")
		datasizeflag = flag.String("valuesize", "100b", "size of each value")
		keysizeflag  = flag.String("keysize", "32b", "size of each key")
		dirflag      = flag.String("dir", ".", "test database directory")
		logdirflag   = flag.String("logdir", ".", "test log output directory")
		keydirflag   = flag.String("keydir", "", "test keyfile directory")
		deletedbflag = flag.Bool("deletedb", false, "delete databases after test run")
		metricsAddr  = flag.String("metrics-addr", ":2112", "The address to serve metrics on")

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
	if cfg.DataSize, err = bench.ParseSize(*datasizeflag); err != nil {
		log.Fatal("-datasize: ", err)
	}
	if cfg.KeySize, err = bench.ParseSize(*keysizeflag); err != nil {
		log.Fatal("-keysize: ", err)
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
		if _, err := os.Stat(keyfile); os.IsNotExist(err) {
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
	"nobatch":        seqWrite{},
	"nobatch-nosync": seqWrite{wOptions: pebble.NoSync},
	"batch-100kb":    batchWrite{BatchSize: 100 * bench.KiB},
	"batch-1mb":      batchWrite{BatchSize: bench.MiB},
	"batch-5mb":      batchWrite{BatchSize: 5 * bench.MiB},
	"batch-100kb-mt-64mb-cache-1gb": batchWrite{
		BatchSize: 100 * bench.KiB,
		Options: pebble.Options{
			Cache:        pebble.NewCache(int64(1024 * bench.MiB)),
			MemTableSize: 64 * bench.MiB,
		},
	},
	"batch-100kb-mt-256mb-cache-1gb": batchWrite{
		BatchSize: 100 * bench.KiB,
		Options: pebble.Options{
			Cache:        pebble.NewCache(int64(1024 * bench.MiB)),
			MemTableSize: 256 * bench.MiB,
		},
	},
	"batch-100kb-mt-512mb-cache-1gb": batchWrite{
		BatchSize: 100 * bench.KiB,
		Options: pebble.Options{
			Cache:        pebble.NewCache(int64(1024 * bench.MiB)),
			MemTableSize: 512 * bench.MiB,
		},
	},
	"batch-100kb-nosync": batchWrite{
		BatchSize: 100 * bench.KiB,
		wOptions:  pebble.NoSync,
	},
	"batch-100kb-mt-512mb-cache-1gb-nosync": batchWrite{
		BatchSize: 100 * bench.KiB,
		wOptions:  pebble.NoSync,
		Options: pebble.Options{
			// These settings approximate what geth is doing.
			Cache:        pebble.NewCache(int64(1024 * bench.MiB)),
			MemTableSize: 512 * bench.MiB,
		},
	},
	"batch-100kb-mt-1gb-cache-1gb-nosync": batchWrite{
		BatchSize: 100 * bench.KiB,
		wOptions:  pebble.NoSync,
		Options: pebble.Options{
			Cache:        pebble.NewCache(int64(1 * bench.GiB)),
			MemTableSize: 1 * bench.GiB,
		},
	},
	"batch-100kb-mt-512mb-cache-4gb-nosync": batchWrite{
		BatchSize: 100 * bench.KiB,
		wOptions:  pebble.NoSync,
		Options: pebble.Options{
			Cache:        pebble.NewCache(int64(4 * bench.GiB)),
			MemTableSize: 512 * bench.MiB,
		},
	},
	"batch-100kb-mt-4gb-cache-16gb-nosync": batchWrite{
		BatchSize: 100 * bench.KiB,
		wOptions:  pebble.NoSync,
		Options: pebble.Options{
			// These settings approximate what geth is doing.
			Cache:        pebble.NewCache(int64(16 * bench.GiB)),
			MemTableSize: 4*bench.GiB - 1 - 1,
		},
	},
	"batch-100kb-mt-4gb-cache-32gb-nosync": batchWrite{
		BatchSize: 100 * bench.KiB,
		wOptions:  pebble.NoSync,
		Options: pebble.Options{
			Cache:        pebble.NewCache(int64(32 * bench.GiB)),
			MemTableSize: 4*bench.GiB - 1 - 1,
		},
	},
	// "batch-100kb-ctable-64mb": batchWrite{
	// 	BatchSize: 100 * 1024,
	// 	Options:   pebble.Options{CompactionTableSize: 64 * MiB},
	// },
	// "batch-100kb-ctable-64mb-nosync": batchWrite{
	// 	BatchSize: 100 * 1024,
	// 	Options:   pebble.Options{NoSync: true, CompactionTableSize: 64 * MiB},
	// },
	// "batch-100kb-ctable-64mb-wb-512mb-cache-1gb": batchWrite{
	// 	BatchSize: 100 * 1024,
	// 	Options: pebble.Options{
	// 		BlockCacheCapacity:  1024 * MiB,
	// 		WriteBuffer:         512 * MiB,
	// 		CompactionTableSize: 64 * MiB,
	// 	},
	// },
	// "batch-100kb-notx": batchWrite{
	// 	BatchSize: 1024 * 1024,
	// 	Options:   pebble.Options{DisableLargeBatchTransaction: true},
	// },
	// "batch-1mb-notx": batchWrite{
	// 	BatchSize: 1024 * 1024,
	// 	Options:   pebble.Options{DisableLargeBatchTransaction: true},
	// },
	// "batch-5mb-notx": batchWrite{
	// 	BatchSize: 5 * 1024 * 1024,
	// 	Options:   pebble.Options{DisableLargeBatchTransaction: true},
	// },
	"concurrent": concurrentWrite{N: 8},
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
	Options   pebble.Options
	wOptions  *pebble.WriteOptions
	BatchSize int
}

func (b batchWrite) Benchmark(dir string, env *bench.WriteEnv) error {
	db, err := pebble.Open(dir, &b.Options)
	if err != nil {
		return err
	}
	defer db.Close()

	batch := db.NewBatch()
	bsize := 0
	return env.Run(func(key, value string, lastCall bool) error {
		batch.Set([]byte(key), []byte(value), nil)
		bsize += len(value)
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
	for i := 0; i < b.N; i++ {
		eg.Go(func() error {
			for {
				select {
				case kv := <-write:
					if err := db.Set([]byte(kv.k), []byte(kv.v), nil); err != nil {
						return err
					}
					env.Progress(len(kv.v))
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
