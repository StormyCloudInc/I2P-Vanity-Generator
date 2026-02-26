//go:build darwin

#import <Metal/Metal.h>
#import <Foundation/Foundation.h>
#include <stdlib.h>
#include <string.h>
#include "metal_bridge.h"

// Embedded Metal shader source (SHA-256 + base32 prefix check)
static NSString* const shaderSource = @"\n"
"#include <metal_stdlib>\n"
"using namespace metal;\n"
"\n"
"// SHA-256 round constants\n"
"constant uint K[64] = {\n"
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
"// SHA-256 helper functions\n"
"inline uint rotr(uint x, uint n) { return (x >> n) | (x << (32 - n)); }\n"
"inline uint ch(uint x, uint y, uint z) { return (x & y) ^ (~x & z); }\n"
"inline uint maj(uint x, uint y, uint z) { return (x & y) ^ (x & z) ^ (y & z); }\n"
"inline uint ep0(uint x) { return rotr(x, 2) ^ rotr(x, 13) ^ rotr(x, 22); }\n"
"inline uint ep1(uint x) { return rotr(x, 6) ^ rotr(x, 11) ^ rotr(x, 25); }\n"
"inline uint sig0(uint x) { return rotr(x, 7) ^ rotr(x, 18) ^ (x >> 3); }\n"
"inline uint sig1(uint x) { return rotr(x, 17) ^ rotr(x, 19) ^ (x >> 10); }\n"
"\n"
"// Base32 alphabet\n"
"constant char B32[32] = {\n"
"    'a','b','c','d','e','f','g','h','i','j','k','l','m',\n"
"    'n','o','p','q','r','s','t','u','v','w','x','y','z',\n"
"    '2','3','4','5','6','7'\n"
"};\n"
"\n"
"// Read a big-endian uint32 from a byte array\n"
"inline uint read_be32(const device uchar* p, uint offset) {\n"
"    return ((uint)p[offset] << 24) | ((uint)p[offset+1] << 16) |\n"
"           ((uint)p[offset+2] << 8) | (uint)p[offset+3];\n"
"}\n"
"inline uint read_be32_local(thread uchar* p, uint offset) {\n"
"    return ((uint)p[offset] << 24) | ((uint)p[offset+1] << 16) |\n"
"           ((uint)p[offset+2] << 8) | (uint)p[offset+3];\n"
"}\n"
"\n"
"struct VanityParams {\n"
"    ulong counter_base;\n"
"    uint prefix_len;\n"
"    char prefix[64];\n"
"};\n"
"\n"
"kernel void vanity_search(\n"
"    device const uchar* dest_template [[buffer(0)]],\n"
"    device const VanityParams* params [[buffer(1)]],\n"
"    device atomic_int* match_found [[buffer(2)]],\n"
"    device ulong* match_counter [[buffer(3)]],\n"
"    uint gid [[thread_position_in_grid]]\n"
") {\n"
"    ulong counter = params->counter_base + (ulong)gid;\n"
"    \n"
"    // Early exit if another thread already found a match\n"
"    if (atomic_load_explicit(match_found, memory_order_relaxed) != 0) return;\n"
"    \n"
"    // Copy destination template to thread-private memory\n"
"    uchar dest[448]; // 391 bytes + padding to 7*64=448 for SHA-256\n"
"    for (uint i = 0; i < 391; i++) {\n"
"        dest[i] = dest_template[i];\n"
"    }\n"
"    \n"
"    // Write counter (little-endian uint64) into bytes 0-7\n"
"    dest[0] = (uchar)(counter);\n"
"    dest[1] = (uchar)(counter >> 8);\n"
"    dest[2] = (uchar)(counter >> 16);\n"
"    dest[3] = (uchar)(counter >> 24);\n"
"    dest[4] = (uchar)(counter >> 32);\n"
"    dest[5] = (uchar)(counter >> 40);\n"
"    dest[6] = (uchar)(counter >> 48);\n"
"    dest[7] = (uchar)(counter >> 56);\n"
"    \n"
"    // SHA-256 padding: message is 391 bytes = 3128 bits\n"
"    // Pad: 0x80, then zeros, then 64-bit big-endian length\n"
"    // Total padded: 7 * 64 = 448 bytes\n"
"    dest[391] = 0x80;\n"
"    for (uint i = 392; i < 440; i++) {\n"
"        dest[i] = 0;\n"
"    }\n"
"    // Length in bits = 391 * 8 = 3128 = 0x0C38\n"
"    dest[440] = 0; dest[441] = 0; dest[442] = 0; dest[443] = 0;\n"
"    dest[444] = 0; dest[445] = 0; dest[446] = 0x0C; dest[447] = 0x38;\n"
"    \n"
"    // SHA-256 compression\n"
"    uint h0 = 0x6a09e667, h1 = 0xbb67ae85, h2 = 0x3c6ef372, h3 = 0xa54ff53a;\n"
"    uint h4 = 0x510e527f, h5 = 0x9b05688c, h6 = 0x1f83d9ab, h7 = 0x5be0cd19;\n"
"    \n"
"    // Process 7 blocks of 64 bytes\n"
"    for (uint block = 0; block < 7; block++) {\n"
"        uint w[64];\n"
"        uint base = block * 64;\n"
"        \n"
"        // Load message schedule W[0..15]\n"
"        for (uint i = 0; i < 16; i++) {\n"
"            w[i] = read_be32_local(dest, base + i * 4);\n"
"        }\n"
"        \n"
"        // Extend W[16..63]\n"
"        for (uint i = 16; i < 64; i++) {\n"
"            w[i] = sig1(w[i-2]) + w[i-7] + sig0(w[i-15]) + w[i-16];\n"
"        }\n"
"        \n"
"        // Compression\n"
"        uint a = h0, b = h1, c = h2, d = h3;\n"
"        uint e = h4, f = h5, g = h6, h = h7;\n"
"        \n"
"        for (uint i = 0; i < 64; i++) {\n"
"            uint t1 = h + ep1(e) + ch(e, f, g) + K[i] + w[i];\n"
"            uint t2 = ep0(a) + maj(a, b, c);\n"
"            h = g; g = f; f = e; e = d + t1;\n"
"            d = c; c = b; b = a; a = t1 + t2;\n"
"        }\n"
"        \n"
"        h0 += a; h1 += b; h2 += c; h3 += d;\n"
"        h4 += e; h5 += f; h6 += g; h7 += h;\n"
"    }\n"
"    \n"
"    // Extract hash bytes (only need enough for prefix check)\n"
"    uchar hash[32];\n"
"    hash[0] = (h0 >> 24); hash[1] = (h0 >> 16); hash[2] = (h0 >> 8); hash[3] = h0;\n"
"    hash[4] = (h1 >> 24); hash[5] = (h1 >> 16); hash[6] = (h1 >> 8); hash[7] = h1;\n"
"    hash[8] = (h2 >> 24); hash[9] = (h2 >> 16); hash[10] = (h2 >> 8); hash[11] = h2;\n"
"    hash[12] = (h3 >> 24); hash[13] = (h3 >> 16); hash[14] = (h3 >> 8); hash[15] = h3;\n"
"    hash[16] = (h4 >> 24); hash[17] = (h4 >> 16); hash[18] = (h4 >> 8); hash[19] = h4;\n"
"    hash[20] = (h5 >> 24); hash[21] = (h5 >> 16); hash[22] = (h5 >> 8); hash[23] = h5;\n"
"    hash[24] = (h6 >> 24); hash[25] = (h6 >> 16); hash[26] = (h6 >> 8); hash[27] = h6;\n"
"    hash[28] = (h7 >> 24); hash[29] = (h7 >> 16); hash[30] = (h7 >> 8); hash[31] = h7;\n"
"    \n"
"    // Base32 encode and compare prefix\n"
"    uint prefix_len = params->prefix_len;\n"
"    uint bit_offset = 0;\n"
"    bool match = true;\n"
"    for (uint i = 0; i < prefix_len && match; i++) {\n"
"        uint byte_idx = bit_offset / 8;\n"
"        uint bit_idx = bit_offset % 8;\n"
"        uint val;\n"
"        if (bit_idx <= 3) {\n"
"            val = (hash[byte_idx] >> (3 - bit_idx)) & 0x1f;\n"
"        } else {\n"
"            val = ((hash[byte_idx] << (bit_idx - 3)) | (hash[byte_idx + 1] >> (11 - bit_idx))) & 0x1f;\n"
"        }\n"
"        if (B32[val] != params->prefix[i]) {\n"
"            match = false;\n"
"        }\n"
"        bit_offset += 5;\n"
"    }\n"
"    \n"
"    if (match) {\n"
"        // Atomically signal match (only first match wins)\n"
"        int expected = 0;\n"
"        if (atomic_compare_exchange_weak_explicit(match_found, &expected, 1,\n"
"                memory_order_relaxed, memory_order_relaxed)) {\n"
"            *match_counter = counter;\n"
"        }\n"
"    }\n"
"}\n";

// ---- Bridge implementation ----

typedef struct {
    id<MTLDevice> device;
    id<MTLCommandQueue> queue;
    id<MTLComputePipelineState> pipeline;
    id<MTLBuffer> destBuf;
    id<MTLBuffer> paramsBuf;
    id<MTLBuffer> matchFoundBuf;
    id<MTLBuffer> matchCounterBuf;
    NSUInteger batchSize;
    NSUInteger maxThreadsPerGroup;
} MetalWorker;

// Packed to match shader struct
typedef struct __attribute__((packed)) {
    uint64_t counter_base;
    uint32_t prefix_len;
    char prefix[64];
} VanityParams;

int metalAvailable(void) {
    @autoreleasepool {
        id<MTLDevice> dev = MTLCreateSystemDefaultDevice();
        return dev != nil ? 1 : 0;
    }
}

char** metalListDevices(int* count) {
    @autoreleasepool {
        NSArray<id<MTLDevice>>* devices = MTLCopyAllDevices();
        if (devices == nil || [devices count] == 0) {
            // Fall back to default device
            id<MTLDevice> dev = MTLCreateSystemDefaultDevice();
            if (dev == nil) {
                *count = 0;
                return NULL;
            }
            *count = 1;
            char** names = (char**)malloc(sizeof(char*));
            const char* name = [[dev name] UTF8String];
            names[0] = strdup(name);
            return names;
        }

        int n = (int)[devices count];
        *count = n;
        char** names = (char**)malloc(sizeof(char*) * n);
        for (int i = 0; i < n; i++) {
            const char* name = [[devices[i] name] UTF8String];
            names[i] = strdup(name);
        }
        return names;
    }
}

void* metalNewWorker(int deviceIndex, const unsigned char* destTemplate,
                     const char* prefix, int prefixLen, unsigned long batchSize) {
    @autoreleasepool {
        // Get device
        id<MTLDevice> device = nil;
        NSArray<id<MTLDevice>>* devices = MTLCopyAllDevices();
        if (devices != nil && [devices count] > 0 && deviceIndex < (int)[devices count]) {
            device = devices[deviceIndex];
        } else {
            device = MTLCreateSystemDefaultDevice();
        }
        if (device == nil) return NULL;

        // Compile shader
        NSError* error = nil;
        MTLCompileOptions* options = [[MTLCompileOptions alloc] init];
        options.fastMathEnabled = YES;
        id<MTLLibrary> library = [device newLibraryWithSource:shaderSource options:options error:&error];
        if (library == nil) {
            NSLog(@"Metal shader compile error: %@", error);
            return NULL;
        }

        id<MTLFunction> function = [library newFunctionWithName:@"vanity_search"];
        if (function == nil) {
            NSLog(@"Metal function 'vanity_search' not found");
            return NULL;
        }

        id<MTLComputePipelineState> pipeline = [device newComputePipelineStateWithFunction:function error:&error];
        if (pipeline == nil) {
            NSLog(@"Metal pipeline error: %@", error);
            return NULL;
        }

        // Create command queue
        id<MTLCommandQueue> queue = [device newCommandQueue];
        if (queue == nil) return NULL;

        // Create buffers
        id<MTLBuffer> destBuf = [device newBufferWithBytes:destTemplate
                                                    length:391
                                                   options:MTLResourceStorageModeShared];

        VanityParams params;
        memset(&params, 0, sizeof(params));
        params.counter_base = 0;
        params.prefix_len = (uint32_t)prefixLen;
        memcpy(params.prefix, prefix, prefixLen < 64 ? prefixLen : 63);

        id<MTLBuffer> paramsBuf = [device newBufferWithBytes:&params
                                                      length:sizeof(VanityParams)
                                                     options:MTLResourceStorageModeShared];

        int32_t zero = 0;
        id<MTLBuffer> matchFoundBuf = [device newBufferWithBytes:&zero
                                                          length:sizeof(int32_t)
                                                         options:MTLResourceStorageModeShared];

        uint64_t zeroCounter = 0;
        id<MTLBuffer> matchCounterBuf = [device newBufferWithBytes:&zeroCounter
                                                            length:sizeof(uint64_t)
                                                           options:MTLResourceStorageModeShared];

        // Allocate worker struct
        MetalWorker* worker = (MetalWorker*)calloc(1, sizeof(MetalWorker));
        worker->device = device;
        worker->queue = queue;
        worker->pipeline = pipeline;
        worker->destBuf = destBuf;
        worker->paramsBuf = paramsBuf;
        worker->matchFoundBuf = matchFoundBuf;
        worker->matchCounterBuf = matchCounterBuf;
        worker->batchSize = (NSUInteger)batchSize;
        worker->maxThreadsPerGroup = [pipeline maxTotalThreadsPerThreadgroup];

        // Retain Objective-C objects
        CFRetain((__bridge CFTypeRef)device);
        CFRetain((__bridge CFTypeRef)queue);
        CFRetain((__bridge CFTypeRef)pipeline);
        CFRetain((__bridge CFTypeRef)destBuf);
        CFRetain((__bridge CFTypeRef)paramsBuf);
        CFRetain((__bridge CFTypeRef)matchFoundBuf);
        CFRetain((__bridge CFTypeRef)matchCounterBuf);

        return worker;
    }
}

unsigned long metalRunBatch(void* handle, unsigned long counterStart,
                            int* matchFound, unsigned long* matchCounter) {
    @autoreleasepool {
        MetalWorker* worker = (MetalWorker*)handle;
        if (!worker) return 0;

        // Update params with new counter_base
        VanityParams* params = (VanityParams*)[worker->paramsBuf contents];
        params->counter_base = (uint64_t)counterStart;

        // Reset match_found to 0
        int32_t* found = (int32_t*)[worker->matchFoundBuf contents];
        *found = 0;

        // Create command buffer and encoder
        id<MTLCommandBuffer> cmdBuf = [worker->queue commandBuffer];
        if (cmdBuf == nil) return 0;

        id<MTLComputeCommandEncoder> encoder = [cmdBuf computeCommandEncoder];
        if (encoder == nil) return 0;

        [encoder setComputePipelineState:worker->pipeline];
        [encoder setBuffer:worker->destBuf offset:0 atIndex:0];
        [encoder setBuffer:worker->paramsBuf offset:0 atIndex:1];
        [encoder setBuffer:worker->matchFoundBuf offset:0 atIndex:2];
        [encoder setBuffer:worker->matchCounterBuf offset:0 atIndex:3];

        // Calculate grid size
        NSUInteger threadGroupSize = worker->maxThreadsPerGroup;
        if (threadGroupSize > 256) threadGroupSize = 256;
        MTLSize gridSize = MTLSizeMake(worker->batchSize, 1, 1);
        MTLSize groupSize = MTLSizeMake(threadGroupSize, 1, 1);

        [encoder dispatchThreads:gridSize threadsPerThreadgroup:groupSize];
        [encoder endEncoding];

        [cmdBuf commit];
        [cmdBuf waitUntilCompleted];

        if ([cmdBuf status] == MTLCommandBufferStatusError) {
            NSLog(@"Metal command buffer error: %@", [cmdBuf error]);
            return 0;
        }

        // Read results
        *matchFound = *(int32_t*)[worker->matchFoundBuf contents];
        *matchCounter = *(uint64_t*)[worker->matchCounterBuf contents];

        return (unsigned long)worker->batchSize;
    }
}

void metalFreeWorker(void* handle) {
    @autoreleasepool {
        MetalWorker* worker = (MetalWorker*)handle;
        if (!worker) return;

        CFRelease((__bridge CFTypeRef)worker->matchCounterBuf);
        CFRelease((__bridge CFTypeRef)worker->matchFoundBuf);
        CFRelease((__bridge CFTypeRef)worker->paramsBuf);
        CFRelease((__bridge CFTypeRef)worker->destBuf);
        CFRelease((__bridge CFTypeRef)worker->pipeline);
        CFRelease((__bridge CFTypeRef)worker->queue);
        CFRelease((__bridge CFTypeRef)worker->device);
        free(worker);
    }
}
