package bench

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	writeCount = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "write_count",
		Help: "The total number of write operations",
	}, []string{"test"})
	writeBytes = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "write_bytes",
		Help: "The total number of bytes written",
	}, []string{"test"})
)

const emitInterval = 500 * 1024 // bytes

type WriteConfig struct {
	Size     uint64 `json:"size"`     // total size of values to write
	KeySize  uint64 `json:"keysize"`  // size of each key written
	DataSize uint64 `json:"datasize"` // size of each value written

	LogPercent bool   `json:"-"`
	TestName   string `json:"-"`
}

type WriteEnv struct {
	cfg WriteConfig
	// generating keys and values
	key, value []byte
	rand       *rand.Rand
	out        *json.Encoder
	kw         io.Writer
	resetKey   func()
	keych      chan [][]byte
	// reporting
	mu                   sync.Mutex
	startTime, lastTime  time.Duration
	written, lastWritten uint64
	lastPercent          int
}

func NewWriteEnv(output io.Writer, kw io.Writer, resetKey func(), cfg WriteConfig) *WriteEnv {
	return &WriteEnv{
		cfg:      cfg,
		out:      json.NewEncoder(output),
		key:      make([]byte, cfg.KeySize),
		value:    make([]byte, cfg.DataSize),
		kw:       kw,
		resetKey: resetKey,
		keych:    make(chan [][]byte, 100),
	}
}

// Run calls write repeatedly with random keys and values.
// The write function should perform a database write and call LegacyWriteProgress when
// data has actually been flushed to disk.
func (env *WriteEnv) Run(write func(key, value string, lastCall bool) error) error {
	env.start()
	written := uint64(0)

	var (
		keypool [][]byte
		wg      sync.WaitGroup
	)
	defer func() {
		wg.Wait()
	}()

	if env.kw != nil {
		wg.Add(1)
		go env.writeKey(&wg)
	}

	for {
		env.rand.Read(env.key)
		env.rand.Read(env.value)
		written += env.cfg.DataSize
		end := written >= env.cfg.Size
		err := write(string(env.key), string(env.value), end)
		if err != nil || end {
			if err == nil {
				keypool = append(keypool, copyBytes(env.key))
			}
			if len(keypool) > 0 {
				env.keych <- keypool
			}
			close(env.keych)
			return err
		}
		keypool = append(keypool, copyBytes(env.key))
		if len(keypool) > 1024 {
			env.keych <- keypool
			keypool = make([][]byte, 0)
		}
	}
}

func (env *WriteEnv) start() {
	env.written, env.lastWritten = 0, 0
	env.rand = rand.New(rand.NewSource(0x1334))
	env.startTime = mononow()
	env.lastTime = env.startTime
}

// LegacyWriteProgress writes a JSON progress event to the environment's output writer.
func (env *WriteEnv) Progress(w int) {
	writeCount.WithLabelValues(env.cfg.TestName).Inc()
	writeBytes.WithLabelValues(env.cfg.TestName).Add(float64(w))

	now := mononow()
	env.mu.Lock()
	defer env.mu.Unlock()
	env.written += uint64(w)
	d := now - env.lastTime
	dw := env.written - env.lastWritten
	if dw > 0 && dw > emitInterval {
		p := Progress{Processed: env.written, Delta: dw, Duration: d}
		env.out.Encode(&p)
		env.logPercentage()
		env.lastTime = now
		env.lastWritten = env.written
	}
}

func (env *WriteEnv) logPercentage() {
	if !env.cfg.LogPercent {
		return
	}
	pct := int((float64(env.written) / float64(env.cfg.Size)) * 100)
	if pct > env.lastPercent {
		log.Printf("%3d%%  %s\n", pct, env.cfg.TestName)
		env.lastPercent = pct
	}
}

func (env *WriteEnv) writeKey(wg *sync.WaitGroup) {
	defer wg.Done()

	for batchKeys := range env.keych {
		var buffer []byte
		for _, key := range batchKeys {
			buffer = append(buffer, key...)
		}
		if _, err := env.kw.Write(buffer); err != nil {
			panic(fmt.Sprintf("failed to write keys %v", err))
		}
	}
}
