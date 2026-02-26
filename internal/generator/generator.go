package generator

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-i2p/i2p-vanitygen/internal/address"
	"github.com/go-i2p/i2p-vanitygen/internal/gpu"
)

// Result holds a successfully found vanity address.
type Result struct {
	Candidate address.Candidate
	Address   string
	Attempts  uint64
	Duration  time.Duration
}

// Stats holds progress information for the search.
type Stats struct {
	Checked    uint64
	KeysPerSec float64
	Elapsed    time.Duration
}

// Generator coordinates parallel vanity address searching.
type Generator struct {
	scheme    address.Scheme
	prefix    string
	numCores  int
	useGPU    bool
	gpuDevice int
	cancel    context.CancelFunc
	mu        sync.Mutex
}

// New creates a new vanity generator.
func New(scheme address.Scheme, prefix string, numCores int, useGPU bool, gpuDevice int) *Generator {
	return &Generator{
		scheme:    scheme,
		prefix:    strings.ToLower(prefix),
		numCores:  numCores,
		useGPU:    useGPU,
		gpuDevice: gpuDevice,
	}
}

// Start begins the parallel vanity search. Returns channels for results and stats.
func (g *Generator) Start(ctx context.Context) (<-chan Result, <-chan Stats) {
	ctx, cancel := context.WithCancel(ctx)
	g.mu.Lock()
	g.cancel = cancel
	g.mu.Unlock()

	resultCh := make(chan Result, 1)
	statsCh := make(chan Stats, 1)

	var totalChecked atomic.Uint64
	var found atomic.Bool
	startTime := time.Now()

	var workerWg sync.WaitGroup
	var statsWg sync.WaitGroup

	// Launch GPU worker if enabled and scheme supports it
	cpuWorkerOffset := 0
	if g.useGPU && g.scheme.SupportsGPU() && gpu.Available() {
		cpuWorkerOffset = 1 // reserve workerID 0 counter space for GPU
		workerWg.Add(1)
		go func() {
			defer workerWg.Done()
			g.gpuWorker(ctx, &totalChecked, &found, resultCh, startTime)
		}()
	}

	// Launch CPU worker goroutines
	for i := 0; i < g.numCores; i++ {
		workerWg.Add(1)
		go func(workerID int) {
			defer workerWg.Done()
			g.worker(ctx, workerID+cpuWorkerOffset, &totalChecked, &found, resultCh)
		}(i)
	}

	// Stats reporter
	statsWg.Add(1)
	go func() {
		defer statsWg.Done()
		ticker := time.NewTicker(250 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				checked := totalChecked.Load()
				elapsed := time.Since(startTime)
				kps := 0.0
				if elapsed.Seconds() > 0 {
					kps = float64(checked) / elapsed.Seconds()
				}
				select {
				case statsCh <- Stats{
					Checked:    checked,
					KeysPerSec: kps,
					Elapsed:    elapsed,
				}:
				default:
				}
			}
		}
	}()

	// Cleanup: wait for workers, cancel context, wait for stats, then close channels
	go func() {
		workerWg.Wait()
		cancel()
		statsWg.Wait()
		close(resultCh)
		close(statsCh)
	}()

	return resultCh, statsCh
}

// Stop cancels the running search.
func (g *Generator) Stop() {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.cancel != nil {
		g.cancel()
	}
}

func (g *Generator) gpuWorker(ctx context.Context, totalChecked *atomic.Uint64, found *atomic.Bool, resultCh chan<- Result, startTime time.Time) {
	// GPU only works with I2P scheme (needs the raw destination template)
	cand, err := g.scheme.NewCandidate()
	if err != nil {
		return
	}
	i2pCand, ok := cand.(*address.I2PCandidate)
	if !ok {
		return // GPU not supported for this scheme
	}

	batchSize := uint64(1 << 22) // ~4M hashes per dispatch
	gpuW, err := gpu.NewWorker(gpu.WorkerConfig{
		DeviceIndex:  g.gpuDevice,
		DestTemplate: i2pCand.Raw(),
		Prefix:       g.prefix,
		BatchSize:    batchSize,
	})
	if err != nil {
		return // GPU unavailable, CPU workers continue
	}
	defer gpuW.Close()

	counter := uint64(0) // GPU uses workerID 0 counter space

	for {
		if found.Load() {
			return
		}
		select {
		case <-ctx.Done():
			return
		default:
		}

		result, err := gpuW.RunBatch(counter)
		if err != nil {
			return // GPU error, stop GPU worker
		}

		totalChecked.Add(result.Checked)
		counter += result.Checked

		if result.Found {
			if found.CompareAndSwap(false, true) {
				// Reconstruct the matching destination on CPU
				i2pCand.Dest.MutateEncryptionKey(result.MatchCounter)
				resultCh <- Result{
					Candidate: i2pCand,
					Address:   i2pCand.FullAddress(),
					Attempts:  totalChecked.Load(),
					Duration:  time.Since(startTime),
				}
			}
			return
		}
	}
}

func (g *Generator) worker(ctx context.Context, workerID int, totalChecked *atomic.Uint64, found *atomic.Bool, resultCh chan<- Result) {
	startTime := time.Now()

	switch g.scheme.Network() {
	case address.NetworkI2P:
		g.i2pWorker(ctx, workerID, totalChecked, found, resultCh, startTime)
	case address.NetworkTorV3:
		g.torV3Worker(ctx, workerID, totalChecked, found, resultCh, startTime)
	}
}

func (g *Generator) i2pWorker(ctx context.Context, workerID int, totalChecked *atomic.Uint64, found *atomic.Bool, resultCh chan<- Result, startTime time.Time) {
	cand, err := g.scheme.NewCandidate()
	if err != nil {
		return
	}
	i2pCand := cand.(*address.I2PCandidate)

	baseCounter := uint64(workerID) << 48
	counter := baseCounter
	batchSize := uint64(1024)

	for {
		if found.Load() {
			return
		}
		if (counter-baseCounter)%batchSize == 0 {
			select {
			case <-ctx.Done():
				return
			default:
			}
		}

		if i2pCand.MutateAndCheck(counter, g.prefix) {
			totalChecked.Add(1)
			counter++
			if found.CompareAndSwap(false, true) {
				resultCh <- Result{
					Candidate: i2pCand,
					Address:   i2pCand.FullAddress(),
					Attempts:  totalChecked.Load(),
					Duration:  time.Since(startTime),
				}
			}
			return
		}

		counter++
		totalChecked.Add(1)
	}
}

func (g *Generator) torV3Worker(ctx context.Context, workerID int, totalChecked *atomic.Uint64, found *atomic.Bool, resultCh chan<- Result, startTime time.Time) {
	cand, err := address.NewTorV3Candidate()
	if err != nil {
		return
	}

	// Each worker starts at a different offset to avoid overlap
	if workerID > 0 {
		cand.AdvanceBy(uint64(workerID) << 48)
	}

	batchSize := uint64(1024)
	checked := uint64(0)

	for {
		if found.Load() {
			return
		}
		if checked%batchSize == 0 {
			select {
			case <-ctx.Done():
				return
			default:
			}
		}

		if cand.CheckPrefix(g.prefix) {
			totalChecked.Add(1)
			checked++
			if found.CompareAndSwap(false, true) {
				resultCh <- Result{
					Candidate: cand,
					Address:   cand.FullAddress(),
					Attempts:  totalChecked.Load(),
					Duration:  time.Since(startTime),
				}
			}
			return
		}

		cand.Advance()
		checked++
		totalChecked.Add(1)
	}
}
