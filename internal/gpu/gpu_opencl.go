//go:build !darwin && cgo

package gpu

/*
#cgo linux LDFLAGS: -lOpenCL
#cgo windows LDFLAGS: -lOpenCL

#ifdef __APPLE__
#include <OpenCL/cl.h>
#else
#include "clheaders/CL/cl.h"
#endif

#include <stdlib.h>
#include <string.h>
#include <stdio.h>

// SHA-256 + Base32 prefix check OpenCL kernel
static const char* kernelSource =
"// SHA-256 round constants\n"
"__constant uint K[64] = {\n"
"    0x428a2f98, 0x71374491, 0xb5c0fbcf, 0xe9b5dba5,\n"
"    0x3956c25b, 0x59f111f1, 0x923f82a4, 0xab1c5ed5,\n"
"    0xd807aa98, 0x12835b01, 0x243185be, 0x550c7dc3,\n"
"    0x72be5d74, 0x80deb1fe, 0x9bdc06a7, 0xc19bf174,\n"
"    0xe49b69c1, 0xefbe4786, 0x0fc19dc6, 0x240ca1cc,\n"
"    0x2de92c6f, 0x4a7484aa, 0x5cb0a9dc, 0x76f988da,\n"
"    0x983e5152, 0xa831c66d, 0xb00327c8, 0xbf597fc7,\n"
"    0xc6e00bf3, 0xd5a79147, 0x06ca6351, 0x14292967,\n"
"    0x27b70a85, 0x2e1b2138, 0x4d2c6dfc, 0x53380d13,\n"
"    0x650a7354, 0x766a0abb, 0x81c2c92e, 0x92722c85,\n"
"    0xa2bfe8a1, 0xa81a664b, 0xc24b8b70, 0xc76c51a3,\n"
"    0xd192e819, 0xd6990624, 0xf40e3585, 0x106aa070,\n"
"    0x19a4c116, 0x1e376c08, 0x2748774c, 0x34b0bcb5,\n"
"    0x391c0cb3, 0x4ed8aa4a, 0x5b9cca4f, 0x682e6ff3,\n"
"    0x748f82ee, 0x78a5636f, 0x84c87814, 0x8cc70208,\n"
"    0x90befffa, 0xa4506ceb, 0xbef9a3f7, 0xc67178f2\n"
"};\n"
"\n"
"__constant char B32[32] = {\n"
"    'a','b','c','d','e','f','g','h','i','j','k','l','m',\n"
"    'n','o','p','q','r','s','t','u','v','w','x','y','z',\n"
"    '2','3','4','5','6','7'\n"
"};\n"
"\n"
"uint rotr_u(uint x, uint n) { return (x >> n) | (x << (32 - n)); }\n"
"uint ch_u(uint x, uint y, uint z) { return (x & y) ^ (~x & z); }\n"
"uint maj_u(uint x, uint y, uint z) { return (x & y) ^ (x & z) ^ (y & z); }\n"
"uint ep0_u(uint x) { return rotr_u(x, 2) ^ rotr_u(x, 13) ^ rotr_u(x, 22); }\n"
"uint ep1_u(uint x) { return rotr_u(x, 6) ^ rotr_u(x, 11) ^ rotr_u(x, 25); }\n"
"uint sig0_u(uint x) { return rotr_u(x, 7) ^ rotr_u(x, 18) ^ (x >> 3); }\n"
"uint sig1_u(uint x) { return rotr_u(x, 17) ^ rotr_u(x, 19) ^ (x >> 10); }\n"
"\n"
"__kernel void vanity_search(\n"
"    __global const uchar* dest_template,\n"
"    const ulong counter_base,\n"
"    const uint prefix_len,\n"
"    __global const char* prefix,\n"
"    __global int* match_found,\n"
"    __global ulong* match_counter\n"
") {\n"
"    ulong gid = get_global_id(0);\n"
"    ulong counter = counter_base + gid;\n"
"\n"
"    if (*match_found != 0) return;\n"
"\n"
"    uchar dest[448];\n"
"    for (uint i = 0; i < 391; i++) dest[i] = dest_template[i];\n"
"\n"
"    dest[0] = (uchar)(counter);\n"
"    dest[1] = (uchar)(counter >> 8);\n"
"    dest[2] = (uchar)(counter >> 16);\n"
"    dest[3] = (uchar)(counter >> 24);\n"
"    dest[4] = (uchar)(counter >> 32);\n"
"    dest[5] = (uchar)(counter >> 40);\n"
"    dest[6] = (uchar)(counter >> 48);\n"
"    dest[7] = (uchar)(counter >> 56);\n"
"\n"
"    dest[391] = 0x80;\n"
"    for (uint i = 392; i < 440; i++) dest[i] = 0;\n"
"    dest[440] = 0; dest[441] = 0; dest[442] = 0; dest[443] = 0;\n"
"    dest[444] = 0; dest[445] = 0; dest[446] = 0x0C; dest[447] = 0x38;\n"
"\n"
"    uint h0 = 0x6a09e667, h1 = 0xbb67ae85, h2 = 0x3c6ef372, h3 = 0xa54ff53a;\n"
"    uint h4 = 0x510e527f, h5 = 0x9b05688c, h6 = 0x1f83d9ab, h7 = 0x5be0cd19;\n"
"\n"
"    for (uint block = 0; block < 7; block++) {\n"
"        uint w[64];\n"
"        uint base = block * 64;\n"
"        for (uint i = 0; i < 16; i++) {\n"
"            uint off = base + i * 4;\n"
"            w[i] = ((uint)dest[off] << 24) | ((uint)dest[off+1] << 16) |\n"
"                   ((uint)dest[off+2] << 8) | (uint)dest[off+3];\n"
"        }\n"
"        for (uint i = 16; i < 64; i++)\n"
"            w[i] = sig1_u(w[i-2]) + w[i-7] + sig0_u(w[i-15]) + w[i-16];\n"
"\n"
"        uint a = h0, b = h1, c = h2, d = h3;\n"
"        uint e = h4, f = h5, g = h6, h = h7;\n"
"        for (uint i = 0; i < 64; i++) {\n"
"            uint t1 = h + ep1_u(e) + ch_u(e, f, g) + K[i] + w[i];\n"
"            uint t2 = ep0_u(a) + maj_u(a, b, c);\n"
"            h = g; g = f; f = e; e = d + t1;\n"
"            d = c; c = b; b = a; a = t1 + t2;\n"
"        }\n"
"        h0 += a; h1 += b; h2 += c; h3 += d;\n"
"        h4 += e; h5 += f; h6 += g; h7 += h;\n"
"    }\n"
"\n"
"    uchar hash[32];\n"
"    hash[0]=(h0>>24); hash[1]=(h0>>16); hash[2]=(h0>>8); hash[3]=h0;\n"
"    hash[4]=(h1>>24); hash[5]=(h1>>16); hash[6]=(h1>>8); hash[7]=h1;\n"
"    hash[8]=(h2>>24); hash[9]=(h2>>16); hash[10]=(h2>>8); hash[11]=h2;\n"
"    hash[12]=(h3>>24); hash[13]=(h3>>16); hash[14]=(h3>>8); hash[15]=h3;\n"
"    hash[16]=(h4>>24); hash[17]=(h4>>16); hash[18]=(h4>>8); hash[19]=h4;\n"
"    hash[20]=(h5>>24); hash[21]=(h5>>16); hash[22]=(h5>>8); hash[23]=h5;\n"
"    hash[24]=(h6>>24); hash[25]=(h6>>16); hash[26]=(h6>>8); hash[27]=h6;\n"
"    hash[28]=(h7>>24); hash[29]=(h7>>16); hash[30]=(h7>>8); hash[31]=h7;\n"
"\n"
"    uint bit_offset = 0;\n"
"    int match = 1;\n"
"    for (uint i = 0; i < prefix_len; i++) {\n"
"        uint byte_idx = bit_offset / 8;\n"
"        uint bit_idx = bit_offset % 8;\n"
"        uint val;\n"
"        if (bit_idx <= 3)\n"
"            val = (hash[byte_idx] >> (3 - bit_idx)) & 0x1f;\n"
"        else\n"
"            val = ((hash[byte_idx] << (bit_idx - 3)) | (hash[byte_idx + 1] >> (11 - bit_idx))) & 0x1f;\n"
"        if (B32[val] != prefix[i]) { match = 0; break; }\n"
"        bit_offset += 5;\n"
"    }\n"
"\n"
"    if (match) {\n"
"        if (atomic_cmpxchg(match_found, 0, 1) == 0) {\n"
"            *match_counter = counter;\n"
"        }\n"
"    }\n"
"}\n";

typedef struct {
    cl_context context;
    cl_command_queue queue;
    cl_kernel kernel;
    cl_program program;
    cl_mem destBuf;
    cl_mem prefixBuf;
    cl_mem matchFoundBuf;
    cl_mem matchCounterBuf;
    cl_ulong batchSize;
    int prefixLen;
} OpenCLWorker;

static cl_device_id* g_devices = NULL;
static int g_deviceCount = 0;
static int g_initialized = 0;

static void ensureInit(void) {
    if (g_initialized) return;
    g_initialized = 1;

    cl_uint numPlatforms = 0;
    clGetPlatformIDs(0, NULL, &numPlatforms);
    if (numPlatforms == 0) return;

    cl_platform_id* platforms = (cl_platform_id*)malloc(sizeof(cl_platform_id) * numPlatforms);
    clGetPlatformIDs(numPlatforms, platforms, NULL);

    // Count all GPU devices across platforms
    int total = 0;
    for (cl_uint p = 0; p < numPlatforms; p++) {
        cl_uint nd = 0;
        clGetDeviceIDs(platforms[p], CL_DEVICE_TYPE_GPU, 0, NULL, &nd);
        total += nd;
    }
    if (total == 0) { free(platforms); return; }

    g_devices = (cl_device_id*)malloc(sizeof(cl_device_id) * total);
    int idx = 0;
    for (cl_uint p = 0; p < numPlatforms; p++) {
        cl_uint nd = 0;
        clGetDeviceIDs(platforms[p], CL_DEVICE_TYPE_GPU, 0, NULL, &nd);
        if (nd > 0) {
            clGetDeviceIDs(platforms[p], CL_DEVICE_TYPE_GPU, nd, g_devices + idx, NULL);
            idx += nd;
        }
    }
    g_deviceCount = idx;
    free(platforms);
}

int oclAvailable(void) {
    ensureInit();
    return g_deviceCount > 0 ? 1 : 0;
}

int oclDeviceCount(void) {
    ensureInit();
    return g_deviceCount;
}

char* oclDeviceName(int index) {
    ensureInit();
    if (index < 0 || index >= g_deviceCount) return strdup("Unknown");
    char name[256];
    clGetDeviceInfo(g_devices[index], CL_DEVICE_NAME, sizeof(name), name, NULL);
    return strdup(name);
}

char* oclDeviceVendor(int index) {
    ensureInit();
    if (index < 0 || index >= g_deviceCount) return strdup("Unknown");
    char vendor[256];
    clGetDeviceInfo(g_devices[index], CL_DEVICE_VENDOR, sizeof(vendor), vendor, NULL);
    return strdup(vendor);
}

void* oclNewWorker(int deviceIndex, const unsigned char* destTemplate,
                   const char* prefix, int prefixLen, unsigned long batchSize) {
    ensureInit();
    if (deviceIndex < 0 || deviceIndex >= g_deviceCount) return NULL;

    cl_device_id dev = g_devices[deviceIndex];
    cl_int err;

    cl_context ctx = clCreateContext(NULL, 1, &dev, NULL, NULL, &err);
    if (err != CL_SUCCESS) return NULL;

    cl_command_queue queue = clCreateCommandQueue(ctx, dev, 0, &err);
    if (err != CL_SUCCESS) { clReleaseContext(ctx); return NULL; }

    const char* src = kernelSource;
    size_t srcLen = strlen(kernelSource);
    cl_program prog = clCreateProgramWithSource(ctx, 1, &src, &srcLen, &err);
    if (err != CL_SUCCESS) { clReleaseCommandQueue(queue); clReleaseContext(ctx); return NULL; }

    err = clBuildProgram(prog, 1, &dev, NULL, NULL, NULL);
    if (err != CL_SUCCESS) {
        char log[4096];
        clGetProgramBuildInfo(prog, dev, CL_PROGRAM_BUILD_LOG, sizeof(log), log, NULL);
        fprintf(stderr, "OpenCL build error: %s\n", log);
        clReleaseProgram(prog);
        clReleaseCommandQueue(queue);
        clReleaseContext(ctx);
        return NULL;
    }

    cl_kernel kern = clCreateKernel(prog, "vanity_search", &err);
    if (err != CL_SUCCESS) {
        clReleaseProgram(prog);
        clReleaseCommandQueue(queue);
        clReleaseContext(ctx);
        return NULL;
    }

    cl_mem destBuf = clCreateBuffer(ctx, CL_MEM_READ_ONLY | CL_MEM_COPY_HOST_PTR,
                                    391, (void*)destTemplate, &err);
    cl_mem prefixBuf = clCreateBuffer(ctx, CL_MEM_READ_ONLY | CL_MEM_COPY_HOST_PTR,
                                      prefixLen, (void*)prefix, &err);

    int zero = 0;
    cl_mem matchFoundBuf = clCreateBuffer(ctx, CL_MEM_READ_WRITE | CL_MEM_COPY_HOST_PTR,
                                          sizeof(int), &zero, &err);
    cl_ulong zeroCounter = 0;
    cl_mem matchCounterBuf = clCreateBuffer(ctx, CL_MEM_READ_WRITE | CL_MEM_COPY_HOST_PTR,
                                            sizeof(cl_ulong), &zeroCounter, &err);

    // Set static kernel args
    clSetKernelArg(kern, 0, sizeof(cl_mem), &destBuf);
    // arg 1 (counter_base) set per batch
    cl_uint pl = (cl_uint)prefixLen;
    clSetKernelArg(kern, 2, sizeof(cl_uint), &pl);
    clSetKernelArg(kern, 3, sizeof(cl_mem), &prefixBuf);
    clSetKernelArg(kern, 4, sizeof(cl_mem), &matchFoundBuf);
    clSetKernelArg(kern, 5, sizeof(cl_mem), &matchCounterBuf);

    OpenCLWorker* w = (OpenCLWorker*)calloc(1, sizeof(OpenCLWorker));
    w->context = ctx;
    w->queue = queue;
    w->kernel = kern;
    w->program = prog;
    w->destBuf = destBuf;
    w->prefixBuf = prefixBuf;
    w->matchFoundBuf = matchFoundBuf;
    w->matchCounterBuf = matchCounterBuf;
    w->batchSize = (cl_ulong)batchSize;
    w->prefixLen = prefixLen;
    return w;
}

unsigned long oclRunBatch(void* handle, unsigned long counterStart,
                          int* matchFound, unsigned long* matchCounter) {
    OpenCLWorker* w = (OpenCLWorker*)handle;
    if (!w) return 0;

    // Reset match_found
    int zero = 0;
    clEnqueueWriteBuffer(w->queue, w->matchFoundBuf, CL_TRUE, 0, sizeof(int), &zero, 0, NULL, NULL);

    // Set counter_base arg
    cl_ulong cb = (cl_ulong)counterStart;
    clSetKernelArg(w->kernel, 1, sizeof(cl_ulong), &cb);

    // Dispatch
    size_t globalSize = (size_t)w->batchSize;
    cl_int err = clEnqueueNDRangeKernel(w->queue, w->kernel, 1, NULL, &globalSize, NULL, 0, NULL, NULL);
    if (err != CL_SUCCESS) return 0;

    clFinish(w->queue);

    // Read results
    int mf = 0;
    cl_ulong mc = 0;
    clEnqueueReadBuffer(w->queue, w->matchFoundBuf, CL_TRUE, 0, sizeof(int), &mf, 0, NULL, NULL);
    clEnqueueReadBuffer(w->queue, w->matchCounterBuf, CL_TRUE, 0, sizeof(cl_ulong), &mc, 0, NULL, NULL);

    *matchFound = mf;
    *matchCounter = (unsigned long)mc;
    return (unsigned long)w->batchSize;
}

void oclFreeWorker(void* handle) {
    OpenCLWorker* w = (OpenCLWorker*)handle;
    if (!w) return;
    clReleaseMemObject(w->matchCounterBuf);
    clReleaseMemObject(w->matchFoundBuf);
    clReleaseMemObject(w->prefixBuf);
    clReleaseMemObject(w->destBuf);
    clReleaseKernel(w->kernel);
    clReleaseProgram(w->program);
    clReleaseCommandQueue(w->queue);
    clReleaseContext(w->context);
    free(w);
}
*/
import "C"
import (
	"fmt"
	"unsafe"
)

// Available returns true if at least one OpenCL GPU device is detected.
func Available() bool {
	return C.oclAvailable() != 0
}

// ListDevices enumerates OpenCL GPU devices.
func ListDevices() ([]Device, error) {
	count := int(C.oclDeviceCount())
	if count == 0 {
		return nil, nil
	}

	devices := make([]Device, count)
	for i := 0; i < count; i++ {
		cName := C.oclDeviceName(C.int(i))
		cVendor := C.oclDeviceVendor(C.int(i))
		devices[i] = Device{
			Name:    C.GoString(cName),
			Vendor:  C.GoString(cVendor),
			Backend: "OpenCL",
		}
		C.free(unsafe.Pointer(cName))
		C.free(unsafe.Pointer(cVendor))
	}
	return devices, nil
}

// NewWorker creates a GPU worker using OpenCL compute.
func NewWorker(cfg WorkerConfig) (*Worker, error) {
	if !Available() {
		return nil, fmt.Errorf("no OpenCL GPU available")
	}

	cPrefix := C.CString(cfg.Prefix)
	defer C.free(unsafe.Pointer(cPrefix))

	handle := C.oclNewWorker(
		C.int(cfg.DeviceIndex),
		(*C.uchar)(unsafe.Pointer(&cfg.DestTemplate[0])),
		cPrefix,
		C.int(len(cfg.Prefix)),
		C.ulong(cfg.BatchSize),
	)
	if handle == nil {
		return nil, fmt.Errorf("failed to create OpenCL compute pipeline")
	}

	return &Worker{
		impl: &openclWorker{handle: handle},
	}, nil
}

type openclWorker struct {
	handle unsafe.Pointer
}

func (w *openclWorker) runBatch(counterStart uint64) (BatchResult, error) {
	var matchFound C.int
	var matchCounter C.ulong

	checked := C.oclRunBatch(w.handle, C.ulong(counterStart), &matchFound, &matchCounter)
	if checked == 0 {
		return BatchResult{}, fmt.Errorf("OpenCL kernel execution failed")
	}

	return BatchResult{
		Found:        matchFound != 0,
		MatchCounter: uint64(matchCounter),
		Checked:      uint64(checked),
	}, nil
}

func (w *openclWorker) close() {
	if w.handle != nil {
		C.oclFreeWorker(w.handle)
		w.handle = nil
	}
}
