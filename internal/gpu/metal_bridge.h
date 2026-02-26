#ifndef METAL_BRIDGE_H
#define METAL_BRIDGE_H

#include <stdint.h>

// Returns 1 if a Metal GPU device is available, 0 otherwise.
int metalAvailable(void);

// Returns an array of device name strings. Sets *count to the number of devices.
// Caller must free each string and the array itself.
char** metalListDevices(int* count);

// Creates a new Metal compute worker. Returns an opaque handle, or NULL on failure.
// destTemplate: 391-byte I2P destination template
// prefix: target base32 prefix string
// prefixLen: length of prefix
// batchSize: number of hashes per dispatch
void* metalNewWorker(int deviceIndex, const unsigned char* destTemplate,
                     const char* prefix, int prefixLen, unsigned long batchSize);

// Runs one batch starting at counterStart.
// Sets *matchFound to 1 if a match was found, *matchCounter to the matching counter.
// Returns the number of hashes computed (batchSize), or 0 on error.
unsigned long metalRunBatch(void* handle, unsigned long counterStart,
                            int* matchFound, unsigned long* matchCounter);

// Releases all GPU resources.
void metalFreeWorker(void* handle);

#endif
