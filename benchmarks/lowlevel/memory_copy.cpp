#include <iostream>
#include <vector>
#include <string>
#include <cstring>
#include <cstdint>
#include <immintrin.h>

__attribute__((noinline))
void copy_libc(void* __restrict__ dst, const void* __restrict__ src, size_t n) {
    std::memcpy(dst, src, n);
}

__attribute__((noinline))
void copy_scalar(uint8_t* __restrict__ dst, const uint8_t* __restrict__ src, size_t n) {
    #if defined(__clang__)
    #pragma clang loop vectorize(disable)
    #elif defined(__GNUC__)
    #pragma GCC novector
    #endif
    for (size_t i = 0; i < n; ++i) {
        dst[i] = src[i];
    }
}

__attribute__((noinline))
void copy_unroll64(uint64_t* __restrict__ dst, const uint64_t* __restrict__ src, size_t count64) {
    size_t i = 0;
    for (; i + 3 < count64; i += 4) {
        dst[i + 0] = src[i + 0];
        dst[i + 1] = src[i + 1];
        dst[i + 2] = src[i + 2];
        dst[i + 3] = src[i + 3];
    }
    for (; i < count64; ++i) {
        dst[i] = src[i];
    }
}

__attribute__((noinline, target("avx2")))
void copy_simd_streaming(uint8_t* dst, const uint8_t* src, size_t n) {
    size_t chunks = n / 32;
    const __m256i* s = (const __m256i*)src;
    __m256i* d = (__m256i*)dst;

    for (size_t i = 0; i < chunks; ++i) {
        __m256i v = _mm256_loadu_si256(s + i);
        // Non-temporal streaming store: bypasses CPU cache hierarchy
        _mm256_stream_si256(d + i, v);
    }
    _mm_sfence();

    size_t rem = n % 32;
    if (rem > 0) {
        std::memcpy(dst + chunks * 32, src + chunks * 32, rem);
    }
}

int main(int argc, char* argv[]) {
    size_t bytes = 64000000; // 64 MB
    std::string variant = "libc_memcpy";
    bool validate = false;

    for (int i = 1; i < argc; ++i) {
        std::string arg = argv[i];
        if (arg == "--bytes" && i + 1 < argc) {
            bytes = std::stoull(argv[++i]);
        } else if (arg == "--variant" && i + 1 < argc) {
            variant = argv[++i];
        } else if (arg == "--validate") {
            validate = true;
        }
    }

    if (validate && bytes > 1000000) {
        bytes = 1000000;
    }

    // Allocate 32-byte aligned buffers using posix_memalign
    size_t alloc_bytes = ((bytes + 63) / 64) * 64;
    void* src_raw = nullptr;
    void* dst_raw = nullptr;
    if (posix_memalign(&src_raw, 64, alloc_bytes) != 0 || posix_memalign(&dst_raw, 64, alloc_bytes) != 0) {
        std::cerr << "posix_memalign failed\n";
        return 1;
    }
    uint8_t* src = (uint8_t*)src_raw;
    uint8_t* dst = (uint8_t*)dst_raw;
    std::memset(dst, 0, alloc_bytes);

    for (size_t i = 0; i < bytes; ++i) {
        src[i] = (uint8_t)(i & 0xFF);
    }

    if (variant == "libc_memcpy" || variant == "memcpy") {
        copy_libc(dst, src, bytes);
    } else if (variant == "scalar_loop" || variant == "scalar") {
        copy_scalar(dst, src, bytes);
    } else if (variant == "unroll64") {
        size_t count64 = bytes / 8;
        copy_unroll64((uint64_t*)dst, (const uint64_t*)src, count64);
        size_t rem = bytes % 8;
        if (rem > 0) {
            std::memcpy(dst + count64 * 8, src + count64 * 8, rem);
        }
    } else if (variant == "simd_streaming" || variant == "avx2_stream") {
        copy_simd_streaming(dst, src, bytes);
    } else {
        std::cerr << "Unknown variant: " << variant << std::endl;
        free(src_raw);
        free(dst_raw);
        return 1;
    }

    // Verify copy correctness
    bool verified = (std::memcmp(src, dst, bytes) == 0);

    uint64_t checksum = 0;
    for (size_t i = 0; i < std::min((size_t)1024, bytes); ++i) {
        checksum += dst[i];
    }

    std::cout << "{\"workload\":\"lowlevel_memcpy\",\"variant\":\"" << variant 
              << "\",\"bytes\":" << bytes 
              << ",\"checksum\":\"" << checksum 
              << "\",\"verified\":" << (verified ? "true" : "false") << "}\n";

    free(src_raw);
    free(dst_raw);

    return verified ? 0 : 2;
}
