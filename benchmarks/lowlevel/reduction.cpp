#include <iostream>
#include <vector>
#include <string>
#include <cmath>
#include <cstdint>
#include <iomanip>
#include <immintrin.h>

#if defined(_OPENMP) && __has_include(<omp.h>)
#include <omp.h>
#define HAVE_OPENMP 1
#endif

// Scalar reduction loop with vectorization disabled
__attribute__((noinline))
double reduce_scalar(const double* data, size_t n) {
    double sum = 0.0;
    #if defined(__clang__)
    #pragma clang loop vectorize(disable)
    #pragma clang loop interleave(disable)
    #elif defined(__GNUC__)
    #pragma GCC novector
    #endif
    for (size_t i = 0; i < n; ++i) {
        sum += data[i];
    }
    return sum;
}

// Loop designed for compiler auto-vectorization
__attribute__((noinline))
double reduce_autovec(const double* __restrict__ data, size_t n) {
    double sum = 0.0;
    #if defined(__clang__)
    #pragma clang loop vectorize(enable)
    #elif defined(__GNUC__)
    #pragma GCC ivdep
    #endif
    for (size_t i = 0; i < n; ++i) {
        sum += data[i];
    }
    return sum;
}

// Explicit AVX2 SIMD intrinsics with 4-way accumulator unrolling
__attribute__((noinline, target("avx2,fma")))
double reduce_simd_avx2(const double* data, size_t n) {
    __m256d ymm0 = _mm256_setzero_pd();
    __m256d ymm1 = _mm256_setzero_pd();
    __m256d ymm2 = _mm256_setzero_pd();
    __m256d ymm3 = _mm256_setzero_pd();

    size_t i = 0;
    for (; i + 15 < n; i += 16) {
        __m256d v0 = _mm256_loadu_pd(data + i);
        __m256d v1 = _mm256_loadu_pd(data + i + 4);
        __m256d v2 = _mm256_loadu_pd(data + i + 8);
        __m256d v3 = _mm256_loadu_pd(data + i + 12);

        ymm0 = _mm256_add_pd(ymm0, v0);
        ymm1 = _mm256_add_pd(ymm1, v1);
        ymm2 = _mm256_add_pd(ymm2, v2);
        ymm3 = _mm256_add_pd(ymm3, v3);
    }

    ymm0 = _mm256_add_pd(ymm0, ymm1);
    ymm2 = _mm256_add_pd(ymm2, ymm3);
    ymm0 = _mm256_add_pd(ymm0, ymm2);

    alignas(32) double buf[4];
    _mm256_storeu_pd(buf, ymm0);
    double sum = buf[0] + buf[1] + buf[2] + buf[3];

    for (; i < n; ++i) {
        sum += data[i];
    }
    return sum;
}

// Parallel reduction using OpenMP
__attribute__((noinline))
double reduce_openmp(const double* data, size_t n) {
    double sum = 0.0;
#ifdef HAVE_OPENMP
    #pragma omp parallel for reduction(+:sum) schedule(static)
    for (size_t i = 0; i < n; ++i) {
        sum += data[i];
    }
#else
    for (size_t i = 0; i < n; ++i) {
        sum += data[i];
    }
#endif
    return sum;
}

// Handwritten x86_64 AVX inline assembly reduction
__attribute__((noinline, target("avx")))
double reduce_asm(const double* data, size_t n) {
    double result = 0.0;
    size_t vec_chunks = n / 4;
    size_t remainder = n % 4;

    if (vec_chunks > 0) {
        alignas(32) double temp[4];
        __asm__ volatile (
            "vxorpd %%ymm0, %%ymm0, %%ymm0\n\t"
            "1:\n\t"
            "vmovupd (%[ptr]), %%ymm1\n\t"
            "vaddpd %%ymm1, %%ymm0, %%ymm0\n\t"
            "addq $32, %[ptr]\n\t"
            "decq %[count]\n\t"
            "jnz 1b\n\t"
            "vmovupd %%ymm0, (%[out])\n\t"
            : [ptr] "+r" (data), [count] "+r" (vec_chunks)
            : [out] "r" (temp)
            : "ymm0", "ymm1", "memory", "cc"
        );
        result = temp[0] + temp[1] + temp[2] + temp[3];
    }

    const double* rem_ptr = data + (n - remainder);
    for (size_t i = 0; i < remainder; ++i) {
        result += rem_ptr[i];
    }
    return result;
}

int main(int argc, char* argv[]) {
    size_t n = 10000000;
    std::string variant = "autovec";
    bool validate = false;

    for (int i = 1; i < argc; ++i) {
        std::string arg = argv[i];
        if (arg == "--size" && i + 1 < argc) {
            n = std::stoull(argv[++i]);
        } else if (arg == "--variant" && i + 1 < argc) {
            variant = argv[++i];
        } else if (arg == "--validate") {
            validate = true;
        }
    }

    if (validate && n > 100000) {
        n = 100000;
    }

    // Allocate 32-byte aligned memory for SIMD efficiency
    std::vector<double> data(n);
    for (size_t i = 0; i < n; ++i) {
        data[i] = (double)(i % 100) * 0.1;
    }

    double sum = 0.0;
    if (variant == "scalar") {
        sum = reduce_scalar(data.data(), n);
    } else if (variant == "autovec") {
        sum = reduce_autovec(data.data(), n);
    } else if (variant == "simd_avx2") {
        sum = reduce_simd_avx2(data.data(), n);
    } else if (variant == "openmp") {
        sum = reduce_openmp(data.data(), n);
    } else if (variant == "asm") {
        sum = reduce_asm(data.data(), n);
    } else {
        std::cerr << "Unknown variant: " << variant << std::endl;
        return 1;
    }

    // Calculate expected sum deterministically
    size_t full_chunks = n / 100;
    size_t rem = n % 100;
    double expected = (double)full_chunks * 495.0;
    for (size_t i = 0; i < rem; ++i) {
        expected += (double)i * 0.1;
    }

    bool verified = std::fabs(sum - expected) < 1e-3;

    std::cout << std::fixed << std::setprecision(4);
    std::cout << "{\"workload\":\"lowlevel_reduction\",\"variant\":\"" << variant 
              << "\",\"size\":" << n 
              << ",\"sum\":" << sum 
              << ",\"checksum\":\"" << (int64_t)std::round(sum) 
              << "\",\"verified\":" << (verified ? "true" : "false") << "}\n";

    return verified ? 0 : 2;
}
