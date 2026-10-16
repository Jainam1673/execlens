#include <iostream>
#include <unordered_map>
#include <string>
#include <cstdint>

int main(int argc, char* argv[]) {
    int operations = 300000;
    for (int i = 1; i < argc; ++i) {
        if (std::string(argv[i]) == "--operations" && i + 1 < argc) {
            operations = std::stoi(argv[i + 1]);
        }
    }

    int half = operations / 2;
    std::unordered_map<std::string, int64_t> map;
    map.reserve(half);

    for (int i = 0; i < operations; ++i) {
        std::string key = "k_" + std::to_string(i % half);
        map[key] = static_cast<int64_t>(i) * 3;
    }

    int64_t checksum = 0;
    for (int i = 0; i < operations; ++i) {
        std::string key = "k_" + std::to_string(i);
        auto it = map.find(key);
        if (it != map.end()) {
            checksum += it->second;
        } else {
            checksum -= 1;
        }
    }

    std::cout << "{\"workload\":\"hashmap\",\"operations\":" << operations
              << ",\"checksum\":\"" << checksum << "\"}\n";
    return 0;
}
