#include <iostream>
#include <fstream>
#include <string>
#include <iomanip>

int main(int argc, char* argv[]) {
    std::string inputPath = "";
    for (int i = 1; i < argc; ++i) {
        if (std::string(argv[i]) == "--input" && i + 1 < argc) {
            inputPath = argv[i + 1];
        }
    }

    if (inputPath.empty()) {
        std::cerr << "Error: --input required\n";
        return 1;
    }

    std::ifstream file(inputPath);
    if (!file.is_open()) {
        std::cerr << "Error opening file\n";
        return 1;
    }

    int activeCount = 0;
    double totalAmount = 0.0;

    std::string line;
    while (std::getline(file, line)) {
        size_t activePos = line.find("\"active\":");
        if (activePos != std::string::npos) {
            bool isActive = (line.find("true", activePos) != std::string::npos &&
                             line.find("true", activePos) < activePos + 16);
            size_t amtPos = line.find("\"amount\":");
            if (amtPos != std::string::npos) {
                size_t numStart = line.find_first_of("0123456789", amtPos);
                if (numStart != std::string::npos) {
                    size_t numEnd = line.find_first_not_of("0123456789.", numStart);
                    double amount = std::stod(line.substr(numStart, numEnd - numStart));
                    if (isActive) {
                        activeCount++;
                        totalAmount += amount;
                    }
                }
            }
        }
    }

    std::cout << "{\"workload\":\"json_proc\",\"active_count\":" << activeCount
              << ",\"total_amount\":" << std::fixed << std::setprecision(2) << totalAmount << "}\n";
    return 0;
}
