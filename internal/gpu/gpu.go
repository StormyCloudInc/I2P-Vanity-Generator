package gpu

// Device represents a detected GPU compute device.
type Device struct {
	Name          string
	Vendor        string
	MaxWorkGroups int
	Backend       string // "Metal" or "OpenCL"
}

// WorkerConfig configures a GPU vanity search worker.
type WorkerConfig struct {
	DeviceIndex  int
	DestTemplate [391]byte // the 391-byte I2P destination to mutate
	Prefix       string    // target base32 prefix
	BatchSize    uint64    // hashes per kernel dispatch (e.g. 1<<22)
}

// BatchResult holds the outcome of one GPU batch.
type BatchResult struct {
	Found        bool
	MatchCounter uint64 // counter value that produced the matching prefix
	Checked      uint64 // number of hashes computed in this batch
}

// Worker represents an active GPU compute session.
// Created by NewWorker, must be closed with Close.
type Worker struct {
	impl workerImpl
}

// RunBatch dispatches one batch of work to the GPU starting at counterStart.
// Blocks until the GPU finishes.
func (w *Worker) RunBatch(counterStart uint64) (BatchResult, error) {
	return w.impl.runBatch(counterStart)
}

// Close releases all GPU resources.
func (w *Worker) Close() {
	w.impl.close()
}

// workerImpl is the platform-specific backend interface.
type workerImpl interface {
	runBatch(counterStart uint64) (BatchResult, error)
	close()
}
