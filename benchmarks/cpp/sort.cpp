#include <iostream>
#include <vector>
#include <string>
#include <algorithm>
#include <cstdint>
#include <iomanip>

static int partition(std::vector<int32_t>& arr, int low, int high) {
    int mid = low + (high - low) / 2;
    if (arr[mid] < arr[low]) std::swap(arr[low], arr[mid]);
    if (arr[high] < arr[low]) std::swap(arr[low], arr[high]);
    if (arr[mid] < arr[high]) std::swap(arr[high], arr[mid]);

    int32_t pivot = arr[high];
    int i = low - 1;
    for (int j = low; j < high; ++j) {
        if (arr[j] <= pivot) {
            ++i;
            std::swap(arr[i], arr[j]);
        }
    }
    std::swap(arr[i + 1], arr[high]);
    return i + 1;
}

static void quicksort(std::vector<int32_t>& arr, int low, int high) {
    if (low < high) {
        int p = partition(arr, low, high);
        quicksort(arr, low, p - 1);
        quicksort(arr, p + 1, high);
    }
}

int main(int argc, char* argv[]) {
    int size = 200000;
    std::string mode = "quicksort";

    for (int i = 1; i < argc; ++i) {
        if (std::string(argv[i]) == "--size" && i + 1 < argc) {
            size = std::stoi(argv[i + 1]);
        } else if (std::string(argv[i]) == "--mode" && i + 1 < argc) {
            mode = argv[i + 1];
        }
    }

    std::vector<int32_t> arr(size);
    uint64_t state = 42;
    for (int i = 0; i < size; ++i) {
        state = (state * 1664525 + 1013904223) & 0xFFFFFFFFULL;
        arr[i] = static_cast<int32_t>(state);
    }

    if (mode == "stdlib") {
        std::sort(arr.begin(), arr.end());
    } else {
        quicksort(arr, 0, size - 1);
    }

    bool isSorted = true;
    uint64_t checksum = 0;
    for (size_t i = 0; i < arr.size(); ++i) {
        if (i > 0 && arr[i] < arr[i - 1]) {
            isSorted = false;
        }
        uint32_t u32 = static_cast<uint32_t>(arr[i]);
        checksum = (checksum * 31 + u32);
    }

    std::cout << "{\"workload\":\"sort\",\"size\":" << size
              << ",\"mode\":\"" << mode
              << "\",\"sorted\":" << (isSorted ? "true" : "false")
              << ",\"checksum\":\"" << std::hex << checksum << std::dec << "\"}\n";
    return 0;
}
