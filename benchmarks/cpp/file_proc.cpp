#include <iostream>
#include <fstream>
#include <vector>
#include <string>
#include <cstdint>
#include <iomanip>
#include <sstream>

// Standard SHA-256 implementation
namespace sha256 {
    static const uint32_t K[64] = {
        0x428a2f98, 0x71374491, 0xb5c0fbcf, 0xe9b5dba5, 0x3956c25b, 0x59f111f1, 0x923f82a4, 0xab1c5ed5,
        0xd807aa98, 0x12835b01, 0x243185be, 0x550c7dc3, 0x72be5d74, 0x80deb1fe, 0x9bdc06a7, 0xc19bf174,
        0xe49b69c1, 0xefbe4786, 0x0fc19dc6, 0x240ca1cc, 0x2de92c6f, 0x4a7484aa, 0x5cb0a9dc, 0x76f988da,
        0x983e5152, 0xa831c66d, 0xb00327c8, 0xbf597fc7, 0xc6e00bf3, 0xd5a79147, 0x06ca6351, 0x14292967,
        0x27b70a85, 0x2e1b2138, 0x4d2c6dfc, 0x53380d13, 0x650a7354, 0x766a0abb, 0x81c2c92e, 0x92722c85,
        0xa2bfe8a1, 0xa81a664b, 0xc24b8b70, 0xc76c51a3, 0xd192e819, 0xd6990624, 0xf40e3585, 0x106aa070,
        0x19a4c116, 0x1e376c08, 0x2748774c, 0x34b0bcb5, 0x391c0cb3, 0x4ed8aa4a, 0x5b9cca4f, 0x682e6ff3,
        0x748f82ee, 0x78a5636f, 0x84c87814, 0x8cc70208, 0x90befffa, 0xa4506ceb, 0xbef9a3f7, 0xc67178f2
    };

    static inline uint32_t ror(uint32_t x, uint32_t n) { return (x >> n) | (x << (32 - n)); }

    struct Context {
        uint32_t state[8];
        uint64_t count;
        uint8_t buffer[64];

        Context() {
            state[0] = 0x6a09e667; state[1] = 0xbb67ae85; state[2] = 0x3c6ef372; state[3] = 0xa54ff53a;
            state[4] = 0x510e527f; state[5] = 0x9b05688c; state[6] = 0x1f83d9ab; state[7] = 0x5be0cd19;
            count = 0;
        }

        void transform(const uint8_t data[]) {
            uint32_t a = state[0], b = state[1], c = state[2], d = state[3];
            uint32_t e = state[4], f = state[5], g = state[6], h = state[7];
            uint32_t m[64];

            for (int i = 0; i < 16; ++i) {
                m[i] = (static_cast<uint32_t>(data[i * 4]) << 24) |
                       (static_cast<uint32_t>(data[i * 4 + 1]) << 16) |
                       (static_cast<uint32_t>(data[i * 4 + 2]) << 8) |
                       (static_cast<uint32_t>(data[i * 4 + 3]));
            }
            for (int i = 16; i < 64; ++i) {
                uint32_t s0 = ror(m[i - 15], 7) ^ ror(m[i - 15], 18) ^ (m[i - 15] >> 3);
                uint32_t s1 = ror(m[i - 2], 17) ^ ror(m[i - 2], 19) ^ (m[i - 2] >> 10);
                m[i] = m[i - 16] + s0 + m[i - 7] + s1;
            }

            for (int i = 0; i < 64; ++i) {
                uint32_t S1 = ror(e, 6) ^ ror(e, 11) ^ ror(e, 25);
                uint32_t ch = (e & f) ^ ((~e) & g);
                uint32_t temp1 = h + S1 + ch + K[i] + m[i];
                uint32_t S0 = ror(a, 2) ^ ror(a, 13) ^ ror(a, 22);
                uint32_t maj = (a & b) ^ (a & c) ^ (b & c);
                uint32_t temp2 = S0 + maj;

                h = g; g = f; f = e; e = d + temp1;
                d = c; c = b; b = a; a = temp1 + temp2;
            }

            state[0] += a; state[1] += b; state[2] += c; state[3] += d;
            state[4] += e; state[5] += f; state[6] += g; state[7] += h;
        }

        void update(const uint8_t* data, size_t len) {
            for (size_t i = 0; i < len; ++i) {
                buffer[count % 64] = data[i];
                count++;
                if (count % 64 == 0) {
                    transform(buffer);
                }
            }
        }

        std::string finalHex() {
            uint64_t totalBits = count * 8;
            update((const uint8_t*)"\x80", 1);
            while (count % 64 != 56) {
                update((const uint8_t*)"\x00", 1);
            }
            uint8_t bitLenBytes[8];
            for (int i = 0; i < 8; ++i) {
                bitLenBytes[i] = static_cast<uint8_t>((totalBits >> ((7 - i) * 8)) & 0xFF);
            }
            update(bitLenBytes, 8);

            std::ostringstream oss;
            for (int i = 0; i < 8; ++i) {
                oss << std::hex << std::setfill('0') << std::setw(8) << state[i];
            }
            return oss.str();
        }
    };
}

int main(int argc, char* argv[]) {
    std::string filePath = "";
    for (int i = 1; i < argc; ++i) {
        if (std::string(argv[i]) == "--file" && i + 1 < argc) {
            filePath = argv[i + 1];
        }
    }

    if (filePath.empty()) {
        std::cerr << "Error: --file required\n";
        return 1;
    }

    std::ifstream file(filePath, std::ios::binary);
    if (!file.is_open()) {
        std::cerr << "Error opening file\n";
        return 1;
    }

    sha256::Context ctx;
    std::vector<char> buf(64 * 1024);
    uint64_t totalBytes = 0;

    while (file.read(buf.data(), buf.size()) || file.gcount() > 0) {
        std::streamsize bytesRead = file.gcount();
        totalBytes += bytesRead;
        ctx.update(reinterpret_cast<const uint8_t*>(buf.data()), bytesRead);
    }

    std::string digest = ctx.finalHex();
    std::cout << "{\"workload\":\"file_proc\",\"bytes_read\":" << totalBytes
              << ",\"sha256\":\"" << digest << "\"}\n";
    return 0;
}
