#include <iostream>
#include <vector>
#include <string>
#include <cmath>
#include <chrono>
#include <iomanip>
#include <cuda_runtime.h>

__inline__ __device__ float warpReduceSum(float val) {
    for (int offset = warpSize / 2; offset > 0; offset /= 2) {
        val += __shfl_down_sync(0xffffffff, val, offset);
    }
    return val;
}

__global__ void blockReduceSumKernel(const float* __restrict__ g_idata, float* __restrict__ g_odata, int n) {
    extern __shared__ float sdata[];
    unsigned int tid = threadIdx.x;
    unsigned int i = blockIdx.x * (blockDim.x * 2) + threadIdx.x;

    float sum = 0.0f;
    if (i < n) sum += g_idata[i];
    if (i + blockDim.x < n) sum += g_idata[i + blockDim.x];

    sdata[tid] = sum;
    __syncthreads();

    for (unsigned int s = blockDim.x / 2; s > 32; s >>= 1) {
        if (tid < s) {
            sdata[tid] += sdata[tid + s];
        }
        __syncthreads();
    }

    if (tid < 32) {
        float val = sdata[tid];
        if (blockDim.x >= 64) val += sdata[tid + 32];
        val = warpReduceSum(val);
        if (tid == 0) {
            g_odata[blockIdx.x] = val;
        }
    }
}

float cpuReduceSum(const float* data, int n) {
    double sum = 0.0;
    for (int i = 0; i < n; ++i) {
        sum += data[i];
    }
    return (float)sum;
}

int main(int argc, char* argv[]) {
    int n = 16777216; // 16M elements
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

    // Allocation
    auto t_alloc_start = std::chrono::high_resolution_clock::now();
    std::vector<float> h_in(n);
    for (int i = 0; i < n; ++i) {
        h_in[i] = (float)(i % 100) * 0.1f;
    }

    int blockSize = 256;
    int numBlocks = (n + (blockSize * 2) - 1) / (blockSize * 2);

    float *d_in = nullptr, *d_out = nullptr;
    cudaMalloc(&d_in, bytes);
    cudaMalloc(&d_out, numBlocks * sizeof(float));
    std::vector<float> h_out(numBlocks);

    auto t_alloc_end = std::chrono::high_resolution_clock::now();
    double alloc_ms = std::chrono::duration<double, std::milli>(t_alloc_end - t_alloc_start).count();

    // H2D Transfer
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
    cudaEvent_t ev_kern_start, ev_kern_end;
    cudaEventCreate(&ev_kern_start);
    cudaEventCreate(&ev_kern_end);

    size_t sharedMemSize = blockSize * sizeof(float);
    // Warm-up
    blockReduceSumKernel<<<numBlocks, blockSize, sharedMemSize>>>(d_in, d_out, n);
    cudaDeviceSynchronize();

    cudaEventRecord(ev_kern_start);
    blockReduceSumKernel<<<numBlocks, blockSize, sharedMemSize>>>(d_in, d_out, n);
    cudaEventRecord(ev_kern_end);
    cudaEventSynchronize(ev_kern_end);

    float kernel_ms = 0.0f;
    cudaEventElapsedTime(&kernel_ms, ev_kern_start, ev_kern_end);

    // D2H Transfer
    cudaEvent_t ev_d2h_start, ev_d2h_end;
    cudaEventCreate(&ev_d2h_start);
    cudaEventCreate(&ev_d2h_end);

    cudaEventRecord(ev_d2h_start);
    cudaMemcpy(h_out.data(), d_out, numBlocks * sizeof(float), cudaMemcpyDeviceToHost);
    cudaEventRecord(ev_d2h_end);
    cudaEventSynchronize(ev_d2h_end);

    float d2h_ms = 0.0f;
    cudaEventElapsedTime(&d2h_ms, ev_d2h_start, ev_d2h_end);

    float gpu_final_sum = 0.0f;
    for (int i = 0; i < numBlocks; ++i) {
        gpu_final_sum += h_out[i];
    }

    cudaFree(d_in);
    cudaFree(d_out);

    auto t_end = std::chrono::high_resolution_clock::now();
    double total_ms = std::chrono::duration<double, std::milli>(t_end - t_start).count();

    // CPU reference
    auto t_cpu_start = std::chrono::high_resolution_clock::now();
    float cpu_sum = cpuReduceSum(h_in.data(), n);
    auto t_cpu_end = std::chrono::high_resolution_clock::now();
    double cpu_ref_ms = std::chrono::duration<double, std::milli>(t_cpu_end - t_cpu_start).count();

    bool verified = (std::fabs(gpu_final_sum - cpu_sum) / (std::fabs(cpu_sum) + 1.0f)) < 1e-3f;

    double effective_bandwidth_gbps = (kernel_ms > 0) ? ((double)bytes) / (kernel_ms * 1e-3) / 1e9 : 0.0;
    double gflops = (kernel_ms > 0) ? ((double)n) / (kernel_ms * 1e-3) / 1e9 : 0.0;

    std::cout << std::fixed << std::setprecision(4);
    std::cout << "{"
              << "\"workload\":\"cuda_parallel_reduction\","
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
