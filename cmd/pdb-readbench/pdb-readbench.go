package main

import (
	"flag"
	"io"
	"log"
	"os"
	"path/filepath"
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
		sizeflag     = flag.String("size", "500mb", "total amount of value data to write")
		datasizeflag = flag.String("valuesize", "100b", "size of each value")
		keysizeflag  = flag.String("keysize", "32b", "size of each key")
		dirflag      = flag.String("dir", ".", "test database directory")
		logdirflag   = flag.String("logdir", ".", "test log output directory")
		keydirflag   = flag.String("keydir", ".", "test keyfile directory")
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
		if err := runTest(*logdirflag, *keydirflag, dbdir, name, createdb, cfg); err != nil {
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

func runTest(logdir, keydir, dbdir, name string, createdb bool, cfg bench.ReadConfig) error {
	cfg.TestName = name
	logfile, err := os.Create(filepath.Join(logdir, name+time.Now().Format(".2006-01-02-15:04:05")+".json"))
	if err != nil {
		return err
	}
	defer logfile.Close()

	var (
		kw    io.Writer
		kr    io.Reader
		reset func()
		kfile = filepath.Join(keydir, "testing.key")
	)
	if !createdb {
		keyfile, err := os.Open(kfile)
		if err != nil {
			return err
		}
		defer keyfile.Close()
		kr = keyfile
	} else {
		keyfile, err := os.Create(kfile)
		if err != nil {
			return err
		}
		defer keyfile.Close()
		kw, kr = keyfile, keyfile
		reset = func() {
			keyfile.Seek(0, io.SeekStart)
		}
	}

	log.Printf("== running %q", name)
	env := bench.NewReadEnv(logfile, kr, kw, reset, cfg)
	return tests[name].Benchmark(dbdir, env)
}

type Benchmarker interface {
	Benchmark(dir string, env *bench.ReadEnv) error
}

var tests = map[string]Benchmarker{
	"random-read": randomRead{},
	"random-read-filter": randomRead{Options: pebble.Options{
		Levels: []pebble.LevelOptions{{TargetFileSize: 2 * 1024 * 1024, FilterPolicy: bloom.FilterPolicy(10)}},
	}},
	"random-read-bigcache": randomRead{Options: pebble.Options{
		Cache: pebble.NewCache(int64(10 * bench.GiB)),
	}},
	"random-read-bigcache-filter": randomRead{Options: pebble.Options{
		Levels: []pebble.LevelOptions{{TargetFileSize: 2 * 1024 * 1024, FilterPolicy: bloom.FilterPolicy(10)}},
		Cache:  pebble.NewCache(int64(10 * bench.GiB)),
	}},
}

func testnames() (n []string) {
	for name := range tests {
		n = append(n, name)
	}
	sort.Strings(n)
	return n
}

type randomRead struct {
	Options pebble.Options
}

func (b randomRead) Benchmark(dir string, env *bench.ReadEnv) error {
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
		if err != nil && err != pebble.ErrNotFound {
			return err
		}
		if err := closer.Close(); err != nil {
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
