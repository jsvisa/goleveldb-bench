package bench

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	writeCount = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "write_count_total",
		Help: "The total number of write operations",
	}, []string{"test"})
	writeBytes = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "write_bytes_total",
		Help: "The total number of bytes written",
	}, []string{"test"})
	writeSeconds = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "write_seconds_total",
		Help: "The total number of seconds taken to write",
	}, []string{"test"})
)

const emitInterval = 500 * 1024 // bytes

// SizeDistribution defines the distribution of sizes for keys or values
type SizeDistribution struct {
	Size1     uint64  `json:"size1"`     // First fixed size
	Size2     uint64  `json:"size2"`     // Second fixed size
	MaxRandom uint64  `json:"maxRandom"` // Maximum size for random values
	Prob1     float64 `json:"prob1"`     // Probability of size1 (0-1)
	Prob2     float64 `json:"prob2"`     // Probability of size2 (0-1)
	// Probability of random size is 1 - prob1 - prob2
}

type WriteConfig struct {
	Size      uint64           `json:"size"`      // total size of key-values to write
	KeyDist   SizeDistribution `json:"keyDist"`   // key size distribution
	ValueDist SizeDistribution `json:"valueDist"` // value size distribution

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

// getRandomSize returns a size based on the specified distribution
func getRandomSize(dist SizeDistribution) uint64 {
	r := rand.Float64()
	if r < dist.Prob1 {
		return dist.Size1
	} else if r < dist.Prob1+dist.Prob2 {
		return dist.Size2
	} else {
		return uint64(rand.Intn(int(dist.MaxRandom))) + 1
	}
}

func NewWriteEnv(output io.Writer, kw io.Writer, resetKey func(), cfg WriteConfig) *WriteEnv {
	// Calculate maximum possible sizes
	maxKeySize := max(cfg.KeyDist.Size1, cfg.KeyDist.Size2, cfg.KeyDist.MaxRandom)
	maxValueSize := max(cfg.ValueDist.Size1, cfg.ValueDist.Size2, cfg.ValueDist.MaxRandom)

	return &WriteEnv{
		cfg:      cfg,
		out:      json.NewEncoder(output),
		key:      make([]byte, maxKeySize),
		value:    make([]byte, maxValueSize),
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
		// Generate key with random size based on distribution
		keySize := getRandomSize(env.cfg.KeyDist)
		env.key = env.key[:keySize]
		env.rand.Read(env.key)

		// Generate value with random size based on distribution
		valueSize := getRandomSize(env.cfg.ValueDist)
		env.value = env.value[:valueSize]
		env.rand.Read(env.value)

		written += uint64(keySize + valueSize)
		end := written >= env.cfg.Size
		st := time.Now()
		err := write(string(env.key), string(env.value), end)
		writeCount.WithLabelValues(env.cfg.TestName).Inc()
		writeSeconds.WithLabelValues(env.cfg.TestName).Add(float64(time.Since(st).Seconds()))
		if err != nil || end {
			if env.kw != nil {
				if err == nil {
					keypool = append(keypool, copyBytes(env.key))
				}
				if len(keypool) > 0 {
					env.keych <- keypool
				}
				close(env.keych)
			}
			return err
		}
		if env.kw != nil {
			keypool = append(keypool, copyBytes(env.key))
			if len(keypool) > 1024 {
				env.keych <- keypool
				keypool = make([][]byte, 0)
			}
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

// ParseSizeDistribution parses a string in the format "size:probability,size:probability,size:probability"
// into a SizeDistribution struct. The string should contain exactly 3 parts.
// Example: "32b:0.5,65b:0.3,100b:0.2"
func ParseSizeDistribution(distStr string) (SizeDistribution, error) {
	parts := strings.Split(distStr, ",")
	if len(parts) != 3 {
		return SizeDistribution{}, fmt.Errorf("invalid distribution format, expected 3 parts")
	}

	var dist SizeDistribution
	var totalProb float64

	for i, part := range parts {
		subparts := strings.Split(part, ":")
		if len(subparts) != 2 {
			return SizeDistribution{}, fmt.Errorf("invalid part format: %s", part)
		}

		size, err := ParseSize(subparts[0])
		if err != nil {
			return SizeDistribution{}, fmt.Errorf("invalid size in part %d: %v", i, err)
		}

		prob, err := strconv.ParseFloat(subparts[1], 64)
		if err != nil {
			return SizeDistribution{}, fmt.Errorf("invalid probability in part %d: %v", i, err)
		}

		if prob < 0 || prob > 1 {
			return SizeDistribution{}, fmt.Errorf("probability must be between 0 and 1 in part %d", i)
		}

		totalProb += prob

		switch i {
		case 0:
			dist.Size1 = size
			dist.Prob1 = prob
		case 1:
			dist.Size2 = size
			dist.Prob2 = prob
		case 2:
			dist.MaxRandom = size
		}
	}

	if totalProb > 1.0 {
		return SizeDistribution{}, fmt.Errorf("total probability exceeds 1.0")
	}

	return dist, nil
}
