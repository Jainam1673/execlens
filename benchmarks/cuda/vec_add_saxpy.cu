#include <iostream>
#include <vector>
#include <string>
#include <cmath>
#include <chrono>
#include <iomanip>
#include <cuda_runtime.h>

__global__ void saxpyKernel(int n, float a, const float* __restrict__ x, float* __restrict__ y) {
    int i = blockIdx.x * blockDim.x + threadIdx.x;
    if (i < n) {
        y[i] = a * x[i] + y[i];
    }
}

void cpuSaxpy(int n, float a, const float* x, float* y) {
    for (int i = 0; i < n; ++i) {
        y[i] = a * x[i] + y[i];
    }
}

int main(int argc, char* argv[]) {
    int n = 16777216; // 16M elements (~64 MB per vector)
    float alpha = 2.5f;
    bool validate = false;

    for (int i = 1; i < argc; ++i) {
        std::string arg = argv[i];
        if (arg == "--size" && i + 1 < argc) {
            n = std::stoi(argv[++i]);
        } else if (arg == "--validate") {
            validate = true;
        }
    }

    if (validate && n > 1000000) {
        n = 1000000;
    }

    int deviceCount = 0;
    cudaError_t err = cudaGetDeviceCount(&deviceCount);
    if (err != cudaSuccess || deviceCount == 0) {
        std::cerr << "{\"error\":\"NO_CUDA_DEVICE\",\"message\":\"" << cudaGetErrorString(err) << "\"}\n";
        return 1;
    }

    cudaDeviceProp prop;
    cudaGetDeviceProperties(&prop, 0);

    size_t bytes = (size_t)n * sizeof(float);

    // Timing breakdown
    auto t_start = std::chrono::high_resolution_clock::now();

    // Allocation
    auto t_alloc_start = std::chrono::high_resolution_clock::now();
    std::vector<float> h_x(n);
    std::vector<float> h_y(n);
    std::vector<float> h_ref_y(n);
    for (int i = 0; i < n; ++i) {
        h_x[i] = (float)(i % 100) * 0.1f;
        h_y[i] = (float)((i + 1) % 100) * 0.2f;
        h_ref_y[i] = h_y[i];
    }

    float *d_x = nullptr, *d_y = nullptr;
    cudaMalloc(&d_x, bytes);
    cudaMalloc(&d_y, bytes);
    auto t_alloc_end = std::chrono::high_resolution_clock::now();
    double alloc_ms = std::chrono::duration<double, std::milli>(t_alloc_end - t_alloc_start).count();

    // Host-to-Device transfer
    cudaEvent_t ev_h2d_start, ev_h2d_end;
    cudaEventCreate(&ev_h2d_start);
    cudaEventCreate(&ev_h2d_end);

    cudaEventRecord(ev_h2d_start);
    cudaMemcpy(d_x, h_x.data(), bytes, cudaMemcpyHostToDevice);
    cudaMemcpy(d_y, h_y.data(), bytes, cudaMemcpyHostToDevice);
    cudaEventRecord(ev_h2d_end);
    cudaEventSynchronize(ev_h2d_end);

    float h2d_ms = 0.0f;
    cudaEventElapsedTime(&h2d_ms, ev_h2d_start, ev_h2d_end);

    // Pure Kernel execution
    cudaEvent_t ev_kern_start, ev_kern_end;
    cudaEventCreate(&ev_kern_start);
    cudaEventCreate(&ev_kern_end);

    int blockSize = 256;
    int numBlocks = (n + blockSize - 1) / blockSize;

    // Warm-up kernel
    saxpyKernel<<<numBlocks, blockSize>>>(n, alpha, d_x, d_y);
    cudaDeviceSynchronize();

    cudaEventRecord(ev_kern_start);
    saxpyKernel<<<numBlocks, blockSize>>>(n, alpha, d_x, d_y);
    cudaEventRecord(ev_kern_end);
    cudaEventSynchronize(ev_kern_end);

    float kernel_ms = 0.0f;
    cudaEventElapsedTime(&kernel_ms, ev_kern_start, ev_kern_end);

    // Device-to-Host transfer
    cudaEvent_t ev_d2h_start, ev_d2h_end;
    cudaEventCreate(&ev_d2h_start);
    cudaEventCreate(&ev_d2h_end);

    cudaEventRecord(ev_d2h_start);
    cudaMemcpy(h_y.data(), d_y, bytes, cudaMemcpyDeviceToHost);
    cudaEventRecord(ev_d2h_end);
    cudaEventSynchronize(ev_d2h_end);

    float d2h_ms = 0.0f;
    cudaEventElapsedTime(&d2h_ms, ev_d2h_start, ev_d2h_end);

    // Free device memory
    cudaFree(d_x);
    cudaFree(d_y);

    auto t_end = std::chrono::high_resolution_clock::now();
    double total_ms = std::chrono::duration<double, std::milli>(t_end - t_start).count();

    // CPU Reference benchmark
    auto t_cpu_start = std::chrono::high_resolution_clock::now();
    cpuSaxpy(n, alpha, h_x.data(), h_ref_y.data());
    auto t_cpu_end = std::chrono::high_resolution_clock::now();
    double cpu_ref_ms = std::chrono::duration<double, std::milli>(t_cpu_end - t_cpu_start).count();

    // Validation
    bool verified = true;
    for (int i = 0; i < std::min(n, 10000); ++i) {
        if (std::fabs(h_y[i] - h_ref_y[i]) > 1e-3f) {
            verified = false;
            break;
        }
    }

    // SAXPY performs 3 memory transfers per element (read x, read y, write y) -> 12 bytes
    // and 2 FLOPs (1 multiply, 1 add)
    double effective_bandwidth_gbps = (kernel_ms > 0) ? ((double)n * 12.0) / (kernel_ms * 1e-3) / 1e9 : 0.0;
    double gflops = (kernel_ms > 0) ? ((double)n * 2.0) / (kernel_ms * 1e-3) / 1e9 : 0.0;

    std::cout << std::fixed << std::setprecision(4);
    std::cout << "{"
              << "\"workload\":\"cuda_vec_add_saxpy\","
              << "\"device_name\":\"" << prop.name << "\","
              << "\"compute_capability\":\"" << prop.major << "." << prop.minor << "\","
              << "\"size\":" << n << ","
              << "\"alloc_ms\":" << alloc_ms << ","
              << "\"h2d_ms\":" << h2d_ms << ","
              << "\"kernel_ms\":" << kernel_ms << ","
              << "\"d2h_ms\":" << d2h_ms << ","
              << "\"total_ms\":" << total_ms << ","
              << "\"effective_bandwidth_gbps\":" << effective_bandwidth_gbps << ","
              << "\"gflops\":" << gflops << ","
              << "\"cpu_ref_ms\":" << cpu_ref_ms << ","
              << "\"verified\":" << (verified ? "true" : "false")
              << "}\n";

    return verified ? 0 : 2;
}
