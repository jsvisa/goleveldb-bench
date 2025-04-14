package bench

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/cockroachdb/pebble"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/syndtr/goleveldb/leveldb"
)

var (
	readCount = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "read_count_total",
		Help: "The total number of read operations",
	}, []string{"test", "status"})
	readBytes = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "read_bytes_total",
		Help: "The total number of bytes readed",
	}, []string{"test"})
	readSeconds = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "read_seconds_total",
		Help: "The total number of seconds taken to read",
	}, []string{"test", "status"})
)

type ReadConfig struct {
	Size          uint64  `json:"size"`           // testing dataset size(pre-constructed)
	KeySize       uint64  `json:"keysize"`        // size of each testing key
	DataSize      uint64  `json:"datasize"`       // size of each testing value
	RandomPercent float64 `json:"random_percent"` // percentage of random keys (0-100)

	LogPercent bool   `json:"-"`
	TestName   string `json:"-"`
}

type ReadEnv struct {
	cfg ReadConfig

	// generating keys and values
	key, value []byte
	rand       *rand.Rand
	log        *json.Encoder
	kw         io.Writer
	kr         io.Reader
	resetKey   func()

	// reporting
	mu                  sync.Mutex
	startTime, lastTime time.Duration
	read, lastRead      uint64
	lastReadPercent     int

	written, lastWritten uint64
	lastWrittenPercent   int
}

func NewReadEnv(log io.Writer, kr io.Reader, kw io.Writer, resetKey func(), cfg ReadConfig) *ReadEnv {
	return &ReadEnv{
		cfg:      cfg,
		log:      json.NewEncoder(log),
		kr:       kr,
		kw:       kw,
		resetKey: resetKey,
		key:      make([]byte, cfg.KeySize),
		value:    make([]byte, cfg.DataSize),
	}
}

// Run calls write repeatedly with random keys and values.
// The write function should perform a database write and call LegacyWriteProgress when
// data has actually been flushed to disk.
func (env *ReadEnv) Run(write func(key, value string, lastCall bool) error, read func(key string) error) error {
	env.start()

	var (
		err      error
		wg       sync.WaitGroup
		shutdown = make(chan struct{})
		result   = make(chan [][]byte, 100)
	)
	defer func() {
		close(shutdown)
		wg.Wait()
	}()

	if env.kw != nil {
		wg.Add(1)
		go env.sideWrite(&wg, write, shutdown)
	}

	wg.Add(1)
	go env.readKey(result, shutdown, &wg)

stageTwo:
	for keybatch := range result {
		for _, key := range keybatch {
			st := time.Now()
			err = read(string(key))
			notfound := err == pebble.ErrNotFound || err == leveldb.ErrNotFound
			status := "200"
			if notfound {
				err = nil
				status = "404"
			}
			readCount.WithLabelValues(env.cfg.TestName, status).Inc()
			readSeconds.WithLabelValues(env.cfg.TestName, status).Add(float64(time.Since(st).Seconds()))
			if err != nil {
				break stageTwo
			}
		}
	}
	if err != nil {
		return err
	}
	return nil
}

func (env *ReadEnv) writeKey(batchKeys [][]byte) {
	var buffer []byte
	for _, key := range batchKeys {
		buffer = append(buffer, key...)
	}
	if _, err := env.kw.Write(buffer); err != nil {
		panic(fmt.Sprintf("failed to write keys %v", err))
	}
}

func (env *ReadEnv) sideWrite(wg *sync.WaitGroup, write func(key, value string, lastCall bool) error, shutdown chan struct{}) {
	defer wg.Done()

	// geth write qps: 100
	timer := time.NewTicker(10 * time.Millisecond)
	defer timer.Stop()

	keypool := make([][]byte, 0, 1024)

stageOne:
	for {
		select {
		case <-shutdown:
			break stageOne
		case <-timer.C:
			env.rand.Read(env.key)
			env.rand.Read(env.value)

			st := time.Now()
			err := write(string(env.key), string(env.value), false)
			writeCount.WithLabelValues(env.cfg.TestName).Inc()
			writeSeconds.WithLabelValues(env.cfg.TestName).Add(float64(time.Since(st).Seconds()))
			if err != nil {
				break stageOne
			}
			keypool = append(keypool, copyBytes(env.key))
			if len(keypool) > 1024 {
				env.writeKey(keypool)
				keypool = make([][]byte, 0)
			}
		}
	}
	if len(keypool) > 0 {
		env.writeKey(keypool)
	}
}

func (env *ReadEnv) readKey(result chan [][]byte, shutdown chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()

	var buffer = make([]byte, env.cfg.KeySize*1024)
	if env.resetKey != nil {
		env.resetKey()
	}
	// Create a new random source for selecting random keys
	randSource := rand.New(rand.NewSource(time.Now().UnixNano()))
	for {
		read, err := env.kr.Read(buffer)
		if read == 0 && env.resetKey != nil {
			// reset the key to read more
			env.resetKey()
			continue
		}
		env.mu.Lock()
		end := env.read >= env.cfg.Size
		env.mu.Unlock()
		if read == 0 || end {
			close(result)
			return
		}
		var batchKey = make([][]byte, read/int(env.cfg.KeySize))
		for i := 0; i+int(env.cfg.KeySize) <= read; i += int(env.cfg.KeySize) {
			if randSource.Float64()*100 < env.cfg.RandomPercent {
				// Generate a random key for the random percentage
				randomKey := make([]byte, env.cfg.KeySize)
				randSource.Read(randomKey)
				batchKey[i/int(env.cfg.KeySize)] = randomKey
			} else {
				batchKey[i/int(env.cfg.KeySize)] = copyBytes(buffer[i : i+int(env.cfg.KeySize)])
			}
		}
		select {
		case result <- batchKey:
		case <-shutdown:
			return
		}
		if err != nil {
			close(result)
			return
		}
	}
}

func (env *ReadEnv) start() {
	env.rand = rand.New(rand.NewSource(0x1334))
	env.startTime = mononow()
	env.lastTime = env.startTime
}

// Progress writes a JSON progress event to the environment's output writer.
func (env *ReadEnv) Progress(w int) {
	readBytes.WithLabelValues(env.cfg.TestName).Add(float64(w))
	now := mononow()
	env.mu.Lock()
	defer env.mu.Unlock()
	env.read += uint64(w)
	d := now - env.lastTime
	dw := env.read - env.lastRead
	if dw > 0 && dw > emitInterval {
		p := Progress{Processed: env.read, Delta: dw, Duration: d}
		env.log.Encode(&p)
		env.logReadPercentage()
		env.lastTime = now
		env.lastRead = env.read
	}
}

func (env *ReadEnv) logReadPercentage() {
	if !env.cfg.LogPercent {
		return
	}
	pct := int((float64(env.read) / float64(env.cfg.Size)) * 100)
	if pct > env.lastReadPercent {
		log.Printf("[Reading] %3d%%  %s\n", pct, env.cfg.TestName)
		env.lastReadPercent = pct
	}
}
