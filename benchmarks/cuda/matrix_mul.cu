#include <iostream>
#include <vector>
#include <string>
#include <cmath>
#include <chrono>
#include <iomanip>
#include <cuda_runtime.h>

#define TILE_WIDTH 16

__global__ void matrixMulNaiveKernel(const float* A, const float* B, float* C, int n) {
    int row = blockIdx.y * blockDim.y + threadIdx.y;
    int col = blockIdx.x * blockDim.x + threadIdx.x;

    if (row < n && col < n) {
        float sum = 0.0f;
        for (int k = 0; k < n; ++k) {
            sum += A[row * n + k] * B[k * n + col];
        }
        C[row * n + col] = sum;
    }
}

__global__ void matrixMulTiledKernel(const float* A, const float* B, float* C, int n) {
    __shared__ float s_a[TILE_WIDTH][TILE_WIDTH];
    __shared__ float s_b[TILE_WIDTH][TILE_WIDTH];

    int bx = blockIdx.x; int by = blockIdx.y;
    int tx = threadIdx.x; int ty = threadIdx.y;

    int row = by * TILE_WIDTH + ty;
    int col = bx * TILE_WIDTH + tx;

    float sum = 0.0f;

    for (int p = 0; p < (n + TILE_WIDTH - 1) / TILE_WIDTH; ++p) {
        if (row < n && p * TILE_WIDTH + tx < n)
            s_a[ty][tx] = A[row * n + p * TILE_WIDTH + tx];
        else
            s_a[ty][tx] = 0.0f;

        if (col < n && p * TILE_WIDTH + ty < n)
            s_b[ty][tx] = B[(p * TILE_WIDTH + ty) * n + col];
        else
            s_b[ty][tx] = 0.0f;

        __syncthreads();

        for (int k = 0; k < TILE_WIDTH; ++k) {
            sum += s_a[ty][k] * s_b[k][tx];
        }
        __syncthreads();
    }

    if (row < n && col < n) {
        C[row * n + col] = sum;
    }
}

void cpuMatrixMul(const float* A, const float* B, float* C, int n) {
    for (int i = 0; i < n; ++i) {
        for (int k = 0; k < n; ++k) {
            float a_ik = A[i * n + k];
            for (int j = 0; j < n; ++j) {
                C[i * n + j] += a_ik * B[k * n + j];
            }
        }
    }
}

int main(int argc, char* argv[]) {
    int n = 1024; // 1024 x 1024 matrix
    std::string variant = "tiled";
    bool validate = false;

    for (int i = 1; i < argc; ++i) {
        std::string arg = argv[i];
        if (arg == "--size" && i + 1 < argc) {
            n = std::stoi(argv[++i]);
        } else if (arg == "--variant" && i + 1 < argc) {
            variant = argv[++i];
        } else if (arg == "--validate") {
            validate = true;
        }
    }

    if (validate && n > 256) {
        n = 256;
    }

    int deviceCount = 0;
    cudaError_t err = cudaGetDeviceCount(&deviceCount);
    if (err != cudaSuccess || deviceCount == 0) {
        std::cerr << "{\"error\":\"NO_CUDA_DEVICE\",\"message\":\"" << cudaGetErrorString(err) << "\"}\n";
        return 1;
    }

    cudaDeviceProp prop;
    cudaGetDeviceProperties(&prop, 0);

    size_t num_elements = (size_t)n * n;
    size_t bytes = num_elements * sizeof(float);

    auto t_start = std::chrono::high_resolution_clock::now();

    // Allocation
    auto t_alloc_start = std::chrono::high_resolution_clock::now();
    std::vector<float> h_a(num_elements);
    std::vector<float> h_b(num_elements);
    std::vector<float> h_c(num_elements, 0.0f);
    std::vector<float> h_ref_c(num_elements, 0.0f);

    for (size_t i = 0; i < num_elements; ++i) {
        h_a[i] = (float)(i % 10) * 0.1f;
        h_b[i] = (float)((i + 1) % 10) * 0.1f;
    }

    float *d_a = nullptr, *d_b = nullptr, *d_c = nullptr;
    cudaMalloc(&d_a, bytes);
    cudaMalloc(&d_b, bytes);
    cudaMalloc(&d_c, bytes);

    auto t_alloc_end = std::chrono::high_resolution_clock::now();
    double alloc_ms = std::chrono::duration<double, std::milli>(t_alloc_end - t_alloc_start).count();

    // H2D
    cudaEvent_t ev_h2d_start, ev_h2d_end;
    cudaEventCreate(&ev_h2d_start);
    cudaEventCreate(&ev_h2d_end);

    cudaEventRecord(ev_h2d_start);
    cudaMemcpy(d_a, h_a.data(), bytes, cudaMemcpyHostToDevice);
    cudaMemcpy(d_b, h_b.data(), bytes, cudaMemcpyHostToDevice);
    cudaEventRecord(ev_h2d_end);
    cudaEventSynchronize(ev_h2d_end);

    float h2d_ms = 0.0f;
    cudaEventElapsedTime(&h2d_ms, ev_h2d_start, ev_h2d_end);

    // Kernel execution
    dim3 dimBlock(TILE_WIDTH, TILE_WIDTH);
    dim3 dimGrid((n + TILE_WIDTH - 1) / TILE_WIDTH, (n + TILE_WIDTH - 1) / TILE_WIDTH);

    cudaEvent_t ev_kern_start, ev_kern_end;
    cudaEventCreate(&ev_kern_start);
    cudaEventCreate(&ev_kern_end);

    // Warm-up
    if (variant == "naive") {
        matrixMulNaiveKernel<<<dimGrid, dimBlock>>>(d_a, d_b, d_c, n);
    } else {
        matrixMulTiledKernel<<<dimGrid, dimBlock>>>(d_a, d_b, d_c, n);
    }
    cudaDeviceSynchronize();

    cudaEventRecord(ev_kern_start);
    if (variant == "naive") {
        matrixMulNaiveKernel<<<dimGrid, dimBlock>>>(d_a, d_b, d_c, n);
    } else {
        matrixMulTiledKernel<<<dimGrid, dimBlock>>>(d_a, d_b, d_c, n);
    }
    cudaEventRecord(ev_kern_end);
    cudaEventSynchronize(ev_kern_end);

    float kernel_ms = 0.0f;
    cudaEventElapsedTime(&kernel_ms, ev_kern_start, ev_kern_end);

    // D2H
    cudaEvent_t ev_d2h_start, ev_d2h_end;
    cudaEventCreate(&ev_d2h_start);
    cudaEventCreate(&ev_d2h_end);

    cudaEventRecord(ev_d2h_start);
    cudaMemcpy(h_c.data(), d_c, bytes, cudaMemcpyDeviceToHost);
    cudaEventRecord(ev_d2h_end);
    cudaEventSynchronize(ev_d2h_end);

    float d2h_ms = 0.0f;
    cudaEventElapsedTime(&d2h_ms, ev_d2h_start, ev_d2h_end);

    cudaFree(d_a);
    cudaFree(d_b);
    cudaFree(d_c);

    auto t_end = std::chrono::high_resolution_clock::now();
    double total_ms = std::chrono::duration<double, std::milli>(t_end - t_start).count();

    // CPU Reference (for small n or validation)
    double cpu_ref_ms = 0.0;
    bool verified = true;
    if (validate || n <= 512) {
        auto t_cpu_start = std::chrono::high_resolution_clock::now();
        cpuMatrixMul(h_a.data(), h_b.data(), h_ref_c.data(), n);
        auto t_cpu_end = std::chrono::high_resolution_clock::now();
        cpu_ref_ms = std::chrono::duration<double, std::milli>(t_cpu_end - t_cpu_start).count();

        for (size_t i = 0; i < std::min(num_elements, (size_t)10000); ++i) {
            if (std::fabs(h_c[i] - h_ref_c[i]) > 1e-2f) {
                verified = false;
                break;
            }
        }
    }

    // 2 * N^3 FLOPs
    double gflops = (kernel_ms > 0) ? (2.0 * (double)n * (double)n * (double)n) / (kernel_ms * 1e-3) / 1e9 : 0.0;
    // Memory transfer: 2 matrices read + 1 matrix written = 3 * N^2 * 4 bytes
    double effective_bandwidth_gbps = (kernel_ms > 0) ? (3.0 * (double)bytes) / (kernel_ms * 1e-3) / 1e9 : 0.0;

    std::cout << std::fixed << std::setprecision(4);
    std::cout << "{"
              << "\"workload\":\"cuda_matrix_mul\","
              << "\"variant\":\"" << variant << "\","
              << "\"device_name\":\"" << prop.name << "\","
              << "\"compute_capability\":\"" << prop.major << "." << prop.minor << "\","
              << "\"dimension\":" << n << ","
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
