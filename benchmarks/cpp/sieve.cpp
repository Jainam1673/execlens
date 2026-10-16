#include <iostream>
#include <vector>
#include <string>
#include <cstdint>

int main(int argc, char* argv[]) {
    int64_t limit = 5000000;
    for (int i = 1; i < argc; ++i) {
        if (std::string(argv[i]) == "--limit" && i + 1 < argc) {
            limit = std::stoll(argv[i + 1]);
        }
    }

    if (limit < 2) {
        std::cout << "{\"workload\":\"sieve\",\"limit\":" << limit << ",\"count\":0,\"checksum\":\"0\"}\n";
        return 0;
    }

    // uint8_t vector for 1 byte per candidate (algorithmic parity)
    std::vector<uint8_t> sieve(limit + 1, 0);
    int64_t count = 0;

    for (int64_t p = 2; p * p <= limit; ++p) {
        if (sieve[p] == 0) {
            for (int64_t i = p * p; i <= limit; i += p) {
                sieve[i] = 1;
            }
        }
    }

    for (int64_t p = 2; p <= limit; ++p) {
        if (sieve[p] == 0) {
            count++;
        }
    }

    std::cout << "{\"workload\":\"sieve\",\"limit\":" << limit << ",\"count\":" << count << ",\"checksum\":\"" << count << "\"}\n";
    return 0;
}
