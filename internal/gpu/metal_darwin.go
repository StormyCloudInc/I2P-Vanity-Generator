//go:build darwin

package gpu

/*
#cgo LDFLAGS: -framework Metal -framework Foundation -framework CoreGraphics
#include <stdlib.h>
#include "metal_bridge.h"
*/
import "C"
import (
	"fmt"
	"unsafe"
)

// Available returns true if at least one Metal GPU device is detected.
func Available() bool {
	return C.metalAvailable() != 0
}

// ListDevices enumerates Metal GPU devices.
func ListDevices() ([]Device, error) {
	var count C.int
	names := C.metalListDevices(&count)
	if count == 0 {
		return nil, nil
	}
	defer C.free(unsafe.Pointer(names))

	// names is a C array of C strings, count entries
	nameSlice := unsafe.Slice((**C.char)(unsafe.Pointer(names)), int(count))
	devices := make([]Device, int(count))
	for i := 0; i < int(count); i++ {
		devices[i] = Device{
			Name:    C.GoString(nameSlice[i]),
			Backend: "Metal",
		}
		C.free(unsafe.Pointer(nameSlice[i]))
	}
	return devices, nil
}

// NewWorker creates a GPU worker using Metal compute shaders.
func NewWorker(cfg WorkerConfig) (*Worker, error) {
	if !Available() {
		return nil, fmt.Errorf("no Metal GPU available")
	}

	cPrefix := C.CString(cfg.Prefix)
	defer C.free(unsafe.Pointer(cPrefix))

	handle := C.metalNewWorker(
		C.int(cfg.DeviceIndex),
		(*C.uchar)(unsafe.Pointer(&cfg.DestTemplate[0])),
		cPrefix,
		C.int(len(cfg.Prefix)),
		C.ulong(cfg.BatchSize),
	)
	if handle == nil {
		return nil, fmt.Errorf("failed to create Metal compute pipeline")
	}

	return &Worker{
		impl: &metalWorker{handle: handle},
	}, nil
}

type metalWorker struct {
	handle unsafe.Pointer
}

func (w *metalWorker) runBatch(counterStart uint64) (BatchResult, error) {
	var matchFound C.int
	var matchCounter C.ulong

	checked := C.metalRunBatch(w.handle, C.ulong(counterStart), &matchFound, &matchCounter)
	if checked == 0 {
		return BatchResult{}, fmt.Errorf("Metal kernel execution failed")
	}

	return BatchResult{
		Found:        matchFound != 0,
		MatchCounter: uint64(matchCounter),
		Checked:      uint64(checked),
	}, nil
}

func (w *metalWorker) close() {
	if w.handle != nil {
		C.metalFreeWorker(w.handle)
		w.handle = nil
	}
}
