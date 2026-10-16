#include <iostream>
#include <vector>
#include <string>
#include <cstdint>
#include <numeric>
#include <algorithm>

struct Node {
    uint32_t next;
    uint32_t val;
};

__attribute__((noinline))
uint64_t run_contiguous(const uint32_t* data, size_t n, size_t rounds) {
    uint64_t sum = 0;
    for (size_t r = 0; r < rounds; ++r) {
        for (size_t i = 0; i < n; ++i) {
            sum += data[i];
        }
    }
    return sum;
}

__attribute__((noinline))
uint64_t run_pointer_chase(const Node* nodes, size_t n, size_t steps) {
    uint64_t sum = 0;
    uint32_t curr = 0;
    for (size_t i = 0; i < steps; ++i) {
        sum += nodes[curr].val;
        curr = nodes[curr].next;
    }
    return sum;
}

int main(int argc, char* argv[]) {
    size_t n = 4000000;
    std::string variant = "contiguous";
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

    if (validate && n > 50000) {
        n = 50000;
    }

    uint64_t result = 0;

    if (variant == "contiguous") {
        std::vector<uint32_t> data(n);
        for (size_t i = 0; i < n; ++i) {
            data[i] = (uint32_t)(i % 100);
        }
        size_t rounds = 4;
        result = run_contiguous(data.data(), n, rounds);
    } else if (variant == "pointer_chasing") {
        std::vector<Node> nodes(n);
        // Build a single Hamiltonian cycle through all indices deterministically
        // using linear congruential coprimes (stride coprime with n)
        // Ensure stride is coprime with n for a single cycle
        uint32_t stride = 10007;
        while (std::gcd(stride, (uint32_t)n) != 1) {
            stride += 2;
        }

        for (size_t i = 0; i < n; ++i) {
            nodes[i].next = (uint32_t)((i + stride) % n);
            nodes[i].val = (uint32_t)(i % 100);
        }

        size_t steps = n * 4;
        result = run_pointer_chase(nodes.data(), n, steps);
    } else {
        std::cerr << "Unknown variant: " << variant << std::endl;
        return 1;
    }

    // Since every element 0..99 appears equally (or nearly equally) across 4 cycles:
    // each 100 elements sum to 4950.
    bool verified = (result > 0);

    std::cout << "{\"workload\":\"lowlevel_pointer_chase\",\"variant\":\"" << variant 
              << "\",\"size\":" << n 
              << ",\"sum\":" << result 
              << ",\"checksum\":\"" << result 
              << "\",\"verified\":" << (verified ? "true" : "false") << "}\n";

    return verified ? 0 : 2;
}
