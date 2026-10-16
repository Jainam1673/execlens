#include <iostream>
#include <vector>
#include <string>
#include <cmath>
#include <chrono>
#include <iomanip>
#include <cuda_runtime.h>

#define NUM_BINS 256
#define HIST_BLOCK_SIZE 256

__global__ void histogramSharedKernel(const uint8_t* __restrict__ data, uint32_t* __restrict__ hist, int n) {
    __shared__ uint32_t s_hist[NUM_BINS];

    int tid = threadIdx.x;
    s_hist[tid] = 0;
    __syncthreads();

    int g_idx = blockIdx.x * blockDim.x + threadIdx.x;
    int stride = blockDim.x * gridDim.x;

    while (g_idx < n) {
        atomicAdd(&s_hist[data[g_idx]], 1);
        g_idx += stride;
    }

    __syncthreads();

    // Accumulate to global memory
    atomicAdd(&hist[tid], s_hist[tid]);
}

void cpuHistogram(const uint8_t* data, uint32_t* hist, int n) {
    for (int i = 0; i < NUM_BINS; ++i) {
        hist[i] = 0;
    }
    for (int i = 0; i < n; ++i) {
        hist[data[i]]++;
    }
}

int main(int argc, char* argv[]) {
    int n = 16777216; // 16M bytes
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

    size_t bytes = (size_t)n * sizeof(uint8_t);
    auto t_start = std::chrono::high_resolution_clock::now();

    // Allocation
    auto t_alloc_start = std::chrono::high_resolution_clock::now();
    std::vector<uint8_t> h_in(n);
    std::vector<uint32_t> h_hist(NUM_BINS, 0);
    std::vector<uint32_t> h_ref_hist(NUM_BINS, 0);

    for (int i = 0; i < n; ++i) {
        h_in[i] = (uint8_t)(i & 0xFF);
    }

    uint8_t* d_in = nullptr;
    uint32_t* d_hist = nullptr;
    cudaMalloc(&d_in, bytes);
    cudaMalloc(&d_hist, NUM_BINS * sizeof(uint32_t));

    auto t_alloc_end = std::chrono::high_resolution_clock::now();
    double alloc_ms = std::chrono::duration<double, std::milli>(t_alloc_end - t_alloc_start).count();

    // H2D
    cudaEvent_t ev_h2d_start, ev_h2d_end;
    cudaEventCreate(&ev_h2d_start);
    cudaEventCreate(&ev_h2d_end);

    cudaEventRecord(ev_h2d_start);
    cudaMemcpy(d_in, h_in.data(), bytes, cudaMemcpyHostToDevice);
    cudaMemset(d_hist, 0, NUM_BINS * sizeof(uint32_t));
    cudaEventRecord(ev_h2d_end);
    cudaEventSynchronize(ev_h2d_end);

    float h2d_ms = 0.0f;
    cudaEventElapsedTime(&h2d_ms, ev_h2d_start, ev_h2d_end);

    // Kernel execution
    int numBlocks = (n + HIST_BLOCK_SIZE - 1) / HIST_BLOCK_SIZE;
    if (numBlocks > 1024) numBlocks = 1024;

    cudaEvent_t ev_kern_start, ev_kern_end;
    cudaEventCreate(&ev_kern_start);
    cudaEventCreate(&ev_kern_end);

    // Warmup
    histogramSharedKernel<<<numBlocks, HIST_BLOCK_SIZE>>>(d_in, d_hist, n);
    cudaDeviceSynchronize();
    cudaMemset(d_hist, 0, NUM_BINS * sizeof(uint32_t));

    cudaEventRecord(ev_kern_start);
    histogramSharedKernel<<<numBlocks, HIST_BLOCK_SIZE>>>(d_in, d_hist, n);
    cudaEventRecord(ev_kern_end);
    cudaEventSynchronize(ev_kern_end);

    float kernel_ms = 0.0f;
    cudaEventElapsedTime(&kernel_ms, ev_kern_start, ev_kern_end);

    // D2H
    cudaEvent_t ev_d2h_start, ev_d2h_end;
    cudaEventCreate(&ev_d2h_start);
    cudaEventCreate(&ev_d2h_end);

    cudaEventRecord(ev_d2h_start);
    cudaMemcpy(h_hist.data(), d_hist, NUM_BINS * sizeof(uint32_t), cudaMemcpyDeviceToHost);
    cudaEventRecord(ev_d2h_end);
    cudaEventSynchronize(ev_d2h_end);

    float d2h_ms = 0.0f;
    cudaEventElapsedTime(&d2h_ms, ev_d2h_start, ev_d2h_end);

    cudaFree(d_in);
    cudaFree(d_hist);

    auto t_end = std::chrono::high_resolution_clock::now();
    double total_ms = std::chrono::duration<double, std::milli>(t_end - t_start).count();

    // CPU reference
    auto t_cpu_start = std::chrono::high_resolution_clock::now();
    cpuHistogram(h_in.data(), h_ref_hist.data(), n);
    auto t_cpu_end = std::chrono::high_resolution_clock::now();
    double cpu_ref_ms = std::chrono::duration<double, std::milli>(t_cpu_end - t_cpu_start).count();

    bool verified = true;
    for (int i = 0; i < NUM_BINS; ++i) {
        if (h_hist[i] != h_ref_hist[i]) {
            verified = false;
            break;
        }
    }

    double effective_bandwidth_gbps = (kernel_ms > 0) ? ((double)bytes) / (kernel_ms * 1e-3) / 1e9 : 0.0;
    double gflops = (kernel_ms > 0) ? ((double)n) / (kernel_ms * 1e-3) / 1e9 : 0.0;

    std::cout << std::fixed << std::setprecision(4);
    std::cout << "{"
              << "\"workload\":\"cuda_histogram_scan\","
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
