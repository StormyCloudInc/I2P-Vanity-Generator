//go:build !darwin && !cgo

package gpu

import "fmt"

// Available returns false when GPU support is not compiled in.
func Available() bool { return false }

// ListDevices returns nil when GPU support is not compiled in.
func ListDevices() ([]Device, error) { return nil, nil }

// NewWorker returns an error when GPU support is not compiled in.
func NewWorker(cfg WorkerConfig) (*Worker, error) {
	return nil, fmt.Errorf("GPU support not available (built without CGo)")
}
