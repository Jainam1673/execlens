#include <iostream>
#include <vector>
#include <string>
#include <iomanip>

int main(int argc, char* argv[]) {
    int n = 300;
    for (int i = 1; i < argc; ++i) {
        if (std::string(argv[i]) == "--dim" && i + 1 < argc) {
            n = std::stoi(argv[i + 1]);
        }
    }

    std::vector<double> a(n * n);
    std::vector<double> b(n * n);
    std::vector<double> c(n * n, 0.0);

    for (int i = 0; i < n; ++i) {
        for (int j = 0; j < n; ++j) {
            a[i * n + j] = static_cast<double>((i * 31 + j * 17) % 100) / 10.0;
            b[i * n + j] = static_cast<double>((i * 13 + j * 23) % 100) / 10.0;
        }
    }

    // C = A * B
    for (int i = 0; i < n; ++i) {
        int iOff = i * n;
        for (int k = 0; k < n; ++k) {
            double aik = a[iOff + k];
            int kOff = k * n;
            for (int j = 0; j < n; ++j) {
                c[iOff + j] += aik * b[kOff + j];
            }
        }
    }

    double checksum = 0.0;
    for (int i = 0; i < n * n; ++i) {
        checksum += c[i];
    }

    std::cout << "{\"workload\":\"matrix\",\"dimension\":" << n
              << ",\"checksum\":\"" << std::fixed << std::setprecision(2) << checksum << "\"}\n";
    return 0;
}
