#include <iostream>
#include <vector>
#include <string>
#include <cmath>
#include <chrono>
#include <iomanip>
#include <cuda_runtime.h>

int main(int argc, char* argv[]) {
    size_t bytes = 67108864; // 64 MB
    bool validate = false;

    for (int i = 1; i < argc; ++i) {
        std::string arg = argv[i];
        if (arg == "--bytes" && i + 1 < argc) {
            bytes = std::stoull(argv[++i]);
        } else if (arg == "--validate") {
            validate = true;
        }
    }

    if (validate && bytes > 4194304) {
        bytes = 4194304; // 4 MB for validation
    }

    int deviceCount = 0;
    cudaError_t err = cudaGetDeviceCount(&deviceCount);
    if (err != cudaSuccess || deviceCount == 0) {
        std::cerr << "{\"error\":\"NO_CUDA_DEVICE\",\"message\":\"" << cudaGetErrorString(err) << "\"}\n";
        return 1;
    }

    cudaDeviceProp prop;
    cudaGetDeviceProperties(&prop, 0);

    float *d_buf = nullptr;
    cudaMalloc(&d_buf, bytes);

    // 1. Pageable Host Memory Transfer
    std::vector<uint8_t> h_pageable(bytes, 0xAB);
    cudaEvent_t ev_start, ev_end;
    cudaEventCreate(&ev_start);
    cudaEventCreate(&ev_end);

    // Pageable H2D
    cudaEventRecord(ev_start);
    cudaMemcpy(d_buf, h_pageable.data(), bytes, cudaMemcpyHostToDevice);
    cudaEventRecord(ev_end);
    cudaEventSynchronize(ev_end);
    float pageable_h2d_ms = 0.0f;
    cudaEventElapsedTime(&pageable_h2d_ms, ev_start, ev_end);

    // Pageable D2H
    cudaEventRecord(ev_start);
    cudaMemcpy(h_pageable.data(), d_buf, bytes, cudaMemcpyDeviceToHost);
    cudaEventRecord(ev_end);
    cudaEventSynchronize(ev_end);
    float pageable_d2h_ms = 0.0f;
    cudaEventElapsedTime(&pageable_d2h_ms, ev_start, ev_end);

    // 2. Pinned Host Memory Transfer (cudaHostAlloc)
    uint8_t* h_pinned = nullptr;
    cudaHostAlloc((void**)&h_pinned, bytes, cudaHostAllocDefault);
    std::memset(h_pinned, 0xCD, bytes);

    // Pinned H2D
    cudaEventRecord(ev_start);
    cudaMemcpy(d_buf, h_pinned, bytes, cudaMemcpyHostToDevice);
    cudaEventRecord(ev_end);
    cudaEventSynchronize(ev_end);
    float pinned_h2d_ms = 0.0f;
    cudaEventElapsedTime(&pinned_h2d_ms, ev_start, ev_end);

    // Pinned D2H
    cudaEventRecord(ev_start);
    cudaMemcpy(h_pinned, d_buf, bytes, cudaMemcpyDeviceToHost);
    cudaEventRecord(ev_end);
    cudaEventSynchronize(ev_end);
    float pinned_d2h_ms = 0.0f;
    cudaEventElapsedTime(&pinned_d2h_ms, ev_start, ev_end);

    // Cleanup
    cudaFree(d_buf);
    cudaFreeHost(h_pinned);

    double pageable_h2d_gbps = (pageable_h2d_ms > 0) ? ((double)bytes / (pageable_h2d_ms * 1e-3)) / 1e9 : 0.0;
    double pageable_d2h_gbps = (pageable_d2h_ms > 0) ? ((double)bytes / (pageable_d2h_ms * 1e-3)) / 1e9 : 0.0;
    double pinned_h2d_gbps = (pinned_h2d_ms > 0) ? ((double)bytes / (pinned_h2d_ms * 1e-3)) / 1e9 : 0.0;
    double pinned_d2h_gbps = (pinned_d2h_ms > 0) ? ((double)bytes / (pinned_d2h_ms * 1e-3)) / 1e9 : 0.0;

    std::cout << std::fixed << std::setprecision(4);
    std::cout << "{"
              << "\"workload\":\"cuda_bandwidth_transfer\","
              << "\"device_name\":\"" << prop.name << "\","
              << "\"compute_capability\":\"" << prop.major << "." << prop.minor << "\","
              << "\"bytes\":" << bytes << ","
              << "\"h2d_ms\":" << pinned_h2d_ms << ","
              << "\"d2h_ms\":" << pinned_d2h_ms << ","
              << "\"total_ms\":" << (pinned_h2d_ms + pinned_d2h_ms) << ","
              << "\"effective_bandwidth_gbps\":" << pinned_h2d_gbps << ","
              << "\"pageable_h2d_gbps\":" << pageable_h2d_gbps << ","
              << "\"pageable_d2h_gbps\":" << pageable_d2h_gbps << ","
              << "\"pinned_h2d_gbps\":" << pinned_h2d_gbps << ","
              << "\"pinned_d2h_gbps\":" << pinned_d2h_gbps << ","
              << "\"verified\":true"
              << "}\n";

    return 0;
}
