#include <iostream>
#include <vector>
#include <string>
#include <cmath>
#include <cstdint>
#include <iomanip>
#include <immintrin.h>

__attribute__((noinline))
double dot_scalar(const double* a, const double* b, size_t n) {
    double sum = 0.0;
    #if defined(__clang__)
    #pragma clang loop vectorize(disable)
    #elif defined(__GNUC__)
    #pragma GCC novector
    #endif
    for (size_t i = 0; i < n; ++i) {
        sum += a[i] * b[i];
    }
    return sum;
}

__attribute__((noinline))
double dot_autovec(const double* __restrict__ a, const double* __restrict__ b, size_t n) {
    double sum = 0.0;
    #if defined(__clang__)
    #pragma clang loop vectorize(enable)
    #elif defined(__GNUC__)
    #pragma GCC ivdep
    #endif
    for (size_t i = 0; i < n; ++i) {
        sum += a[i] * b[i];
    }
    return sum;
}

__attribute__((noinline, target("avx2,fma")))
double dot_simd_fma(const double* a, const double* b, size_t n) {
    __m256d acc0 = _mm256_setzero_pd();
    __m256d acc1 = _mm256_setzero_pd();
    __m256d acc2 = _mm256_setzero_pd();
    __m256d acc3 = _mm256_setzero_pd();

    size_t i = 0;
    for (; i + 15 < n; i += 16) {
        __m256d va0 = _mm256_loadu_pd(a + i);
        __m256d vb0 = _mm256_loadu_pd(b + i);
        acc0 = _mm256_fmadd_pd(va0, vb0, acc0);

        __m256d va1 = _mm256_loadu_pd(a + i + 4);
        __m256d vb1 = _mm256_loadu_pd(b + i + 4);
        acc1 = _mm256_fmadd_pd(va1, vb1, acc1);

        __m256d va2 = _mm256_loadu_pd(a + i + 8);
        __m256d vb2 = _mm256_loadu_pd(b + i + 8);
        acc2 = _mm256_fmadd_pd(va2, vb2, acc2);

        __m256d va3 = _mm256_loadu_pd(a + i + 12);
        __m256d vb3 = _mm256_loadu_pd(b + i + 12);
        acc3 = _mm256_fmadd_pd(va3, vb3, acc3);
    }

    acc0 = _mm256_add_pd(acc0, acc1);
    acc2 = _mm256_add_pd(acc2, acc3);
    acc0 = _mm256_add_pd(acc0, acc2);

    alignas(32) double buf[4];
    _mm256_storeu_pd(buf, acc0);
    double sum = buf[0] + buf[1] + buf[2] + buf[3];

    for (; i < n; ++i) {
        sum += a[i] * b[i];
    }
    return sum;
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

    std::vector<double> a(n);
    std::vector<double> b(n);
    for (size_t i = 0; i < n; ++i) {
        a[i] = (double)(i % 50) * 0.02;
        b[i] = (double)((i + 1) % 50) * 0.02;
    }

    double sum = 0.0;
    if (variant == "scalar") {
        sum = dot_scalar(a.data(), b.data(), n);
    } else if (variant == "autovec") {
        sum = dot_autovec(a.data(), b.data(), n);
    } else if (variant == "simd_fma") {
        sum = dot_simd_fma(a.data(), b.data(), n);
    } else {
        std::cerr << "Unknown variant: " << variant << std::endl;
        return 1;
    }

    // Baseline calculation to verify within floating tolerance
    double baseline = dot_scalar(a.data(), b.data(), n);
    bool verified = std::fabs(sum - baseline) < 1e-2;

    std::cout << std::fixed << std::setprecision(4);
    std::cout << "{\"workload\":\"lowlevel_dot_product\",\"variant\":\"" << variant 
              << "\",\"size\":" << n 
              << ",\"sum\":" << sum 
              << ",\"checksum\":\"" << (int64_t)std::round(sum) 
              << "\",\"verified\":" << (verified ? "true" : "false") << "}\n";

    return verified ? 0 : 2;
}
