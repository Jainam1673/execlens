#include <iostream>
#include <vector>
#include <string>
#include <cmath>
#include <chrono>
#include <iomanip>
#include <cuda_runtime.h>

#define MASK_RADIUS 4
#define MASK_WIDTH (2 * MASK_RADIUS + 1)
#define BLOCK_SIZE 256

__constant__ float c_mask[MASK_WIDTH];

__global__ void conv1DSharedKernel(const float* __restrict__ in, float* __restrict__ out, int n) {
    __shared__ float tile[BLOCK_SIZE + 2 * MASK_RADIUS];

    int g_idx = blockIdx.x * blockDim.x + threadIdx.x;
    int l_idx = threadIdx.x + MASK_RADIUS;

    // Load center
    tile[l_idx] = (g_idx < n) ? in[g_idx] : 0.0f;

    // Load left halo
    if (threadIdx.x < MASK_RADIUS) {
        int left_idx = g_idx - MASK_RADIUS;
        tile[threadIdx.x] = (left_idx >= 0) ? in[left_idx] : 0.0f;
    }

    // Load right halo
    if (threadIdx.x >= blockDim.x - MASK_RADIUS) {
        int right_idx = g_idx + MASK_RADIUS;
        tile[l_idx + MASK_RADIUS] = (right_idx < n) ? in[right_idx] : 0.0f;
    }

    __syncthreads();

    if (g_idx < n) {
        float sum = 0.0f;
        #pragma unroll
        for (int k = 0; k < MASK_WIDTH; ++k) {
            sum += tile[threadIdx.x + k] * c_mask[k];
        }
        out[g_idx] = sum;
    }
}

void cpuConv1D(const float* in, const float* mask, float* out, int n) {
    for (int i = 0; i < n; ++i) {
        float sum = 0.0f;
        for (int k = 0; k < MASK_WIDTH; ++k) {
            int in_idx = i - MASK_RADIUS + k;
            if (in_idx >= 0 && in_idx < n) {
                sum += in[in_idx] * mask[k];
            }
        }
        out[i] = sum;
    }
}

int main(int argc, char* argv[]) {
    int n = 8388608; // 8M elements
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
    auto t_start = std::chrono::high_resolution_clock::now();

    // Setup mask
    std::vector<float> h_mask(MASK_WIDTH);
    for (int k = 0; k < MASK_WIDTH; ++k) {
        h_mask[k] = 1.0f / (float)MASK_WIDTH;
    }
    cudaMemcpyToSymbol(c_mask, h_mask.data(), MASK_WIDTH * sizeof(float));

    // Allocation
    auto t_alloc_start = std::chrono::high_resolution_clock::now();
    std::vector<float> h_in(n);
    std::vector<float> h_out(n);
    std::vector<float> h_ref_out(n);

    for (int i = 0; i < n; ++i) {
        h_in[i] = (float)(i % 50) * 0.1f;
    }

    float *d_in = nullptr, *d_out = nullptr;
    cudaMalloc(&d_in, bytes);
    cudaMalloc(&d_out, bytes);

    auto t_alloc_end = std::chrono::high_resolution_clock::now();
    double alloc_ms = std::chrono::duration<double, std::milli>(t_alloc_end - t_alloc_start).count();

    // H2D
    cudaEvent_t ev_h2d_start, ev_h2d_end;
    cudaEventCreate(&ev_h2d_start);
    cudaEventCreate(&ev_h2d_end);

    cudaEventRecord(ev_h2d_start);
    cudaMemcpy(d_in, h_in.data(), bytes, cudaMemcpyHostToDevice);
    cudaEventRecord(ev_h2d_end);
    cudaEventSynchronize(ev_h2d_end);

    float h2d_ms = 0.0f;
    cudaEventElapsedTime(&h2d_ms, ev_h2d_start, ev_h2d_end);

    // Kernel execution
    int numBlocks = (n + BLOCK_SIZE - 1) / BLOCK_SIZE;

    cudaEvent_t ev_kern_start, ev_kern_end;
    cudaEventCreate(&ev_kern_start);
    cudaEventCreate(&ev_kern_end);

    conv1DSharedKernel<<<numBlocks, BLOCK_SIZE>>>(d_in, d_out, n);
    cudaDeviceSynchronize();

    cudaEventRecord(ev_kern_start);
    conv1DSharedKernel<<<numBlocks, BLOCK_SIZE>>>(d_in, d_out, n);
    cudaEventRecord(ev_kern_end);
    cudaEventSynchronize(ev_kern_end);

    float kernel_ms = 0.0f;
    cudaEventElapsedTime(&kernel_ms, ev_kern_start, ev_kern_end);

    // D2H
    cudaEvent_t ev_d2h_start, ev_d2h_end;
    cudaEventCreate(&ev_d2h_start);
    cudaEventCreate(&ev_d2h_end);

    cudaEventRecord(ev_d2h_start);
    cudaMemcpy(h_out.data(), d_out, bytes, cudaMemcpyDeviceToHost);
    cudaEventRecord(ev_d2h_end);
    cudaEventSynchronize(ev_d2h_end);

    float d2h_ms = 0.0f;
    cudaEventElapsedTime(&d2h_ms, ev_d2h_start, ev_d2h_end);

    cudaFree(d_in);
    cudaFree(d_out);

    auto t_end = std::chrono::high_resolution_clock::now();
    double total_ms = std::chrono::duration<double, std::milli>(t_end - t_start).count();

    // CPU reference
    auto t_cpu_start = std::chrono::high_resolution_clock::now();
    cpuConv1D(h_in.data(), h_mask.data(), h_ref_out.data(), n);
    auto t_cpu_end = std::chrono::high_resolution_clock::now();
    double cpu_ref_ms = std::chrono::duration<double, std::milli>(t_cpu_end - t_cpu_start).count();

    bool verified = true;
    for (int i = 0; i < std::min(n, 10000); ++i) {
        if (std::fabs(h_out[i] - h_ref_out[i]) > 1e-3f) {
            verified = false;
            break;
        }
    }

    double gflops = (kernel_ms > 0) ? ((double)n * MASK_WIDTH * 2.0) / (kernel_ms * 1e-3) / 1e9 : 0.0;
    double effective_bandwidth_gbps = (kernel_ms > 0) ? ((double)bytes * 2.0) / (kernel_ms * 1e-3) / 1e9 : 0.0;

    std::cout << std::fixed << std::setprecision(4);
    std::cout << "{"
              << "\"workload\":\"cuda_convolution_1d\","
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
