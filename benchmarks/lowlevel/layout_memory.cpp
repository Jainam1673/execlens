#include <iostream>
#include <vector>
#include <string>
#include <cmath>
#include <cstdint>
#include <iomanip>

struct ParticleAoS {
    float x, y, z;
    float vx, vy, vz;
    float mass;
    float charge;
};

__attribute__((noinline))
double run_aos(size_t n, size_t steps) {
    std::vector<ParticleAoS> p(n);
    for (size_t i = 0; i < n; ++i) {
        p[i].x = (float)(i % 100);
        p[i].y = (float)((i + 1) % 100);
        p[i].z = (float)((i + 2) % 100);
        p[i].vx = 0.5f;
        p[i].vy = 0.25f;
        p[i].vz = 0.125f;
        p[i].mass = 1.0f;
        p[i].charge = -1.0f;
    }

    const float dt = 0.01f;
    double kinetic_energy = 0.0;

    for (size_t s = 0; s < steps; ++s) {
        for (size_t i = 0; i < n; ++i) {
            p[i].x += p[i].vx * dt;
            p[i].y += p[i].vy * dt;
            p[i].z += p[i].vz * dt;
            kinetic_energy += 0.5 * (double)p[i].mass * 
                ((double)p[i].vx * (double)p[i].vx + 
                 (double)p[i].vy * (double)p[i].vy + 
                 (double)p[i].vz * (double)p[i].vz);
        }
    }
    return kinetic_energy;
}

__attribute__((noinline))
double run_soa(size_t n, size_t steps) {
    std::vector<float> x(n), y(n), z(n);
    std::vector<float> vx(n), vy(n), vz(n);
    std::vector<float> mass(n), charge(n);

    for (size_t i = 0; i < n; ++i) {
        x[i] = (float)(i % 100);
        y[i] = (float)((i + 1) % 100);
        z[i] = (float)((i + 2) % 100);
        vx[i] = 0.5f;
        vy[i] = 0.25f;
        vz[i] = 0.125f;
        mass[i] = 1.0f;
        charge[i] = -1.0f;
    }

    const float dt = 0.01f;
    double kinetic_energy = 0.0;

    float* __restrict__ px = x.data();
    float* __restrict__ py = y.data();
    float* __restrict__ pz = z.data();
    const float* __restrict__ pvx = vx.data();
    const float* __restrict__ pvy = vy.data();
    const float* __restrict__ pvz = vz.data();
    const float* __restrict__ pmass = mass.data();

    for (size_t s = 0; s < steps; ++s) {
        #pragma GCC ivdep
        #pragma clang loop vectorize(enable)
        for (size_t i = 0; i < n; ++i) {
            px[i] += pvx[i] * dt;
            py[i] += pvy[i] * dt;
            pz[i] += pvz[i] * dt;
            kinetic_energy += 0.5 * (double)pmass[i] * 
                ((double)pvx[i] * (double)pvx[i] + 
                 (double)pvy[i] * (double)pvy[i] + 
                 (double)pvz[i] * (double)pvz[i]);
        }
    }
    return kinetic_energy;
}

int main(int argc, char* argv[]) {
    size_t n = 5000000;
    size_t steps = 2;
    std::string variant = "soa";
    bool validate = false;

    for (int i = 1; i < argc; ++i) {
        std::string arg = argv[i];
        if (arg == "--size" && i + 1 < argc) {
            n = std::stoull(argv[++i]);
        } else if (arg == "--steps" && i + 1 < argc) {
            steps = std::stoull(argv[++i]);
        } else if (arg == "--variant" && i + 1 < argc) {
            variant = argv[++i];
        } else if (arg == "--validate") {
            validate = true;
        }
    }

    if (validate && n > 100000) {
        n = 100000;
        steps = 1;
    }

    double ke = 0.0;
    if (variant == "aos") {
        ke = run_aos(n, steps);
    } else if (variant == "soa") {
        ke = run_soa(n, steps);
    } else {
        std::cerr << "Unknown variant: " << variant << std::endl;
        return 1;
    }

    // Expected KE: 0.5 * 1.0 * (0.25 + 0.0625 + 0.015625) = 0.5 * 0.328125 = 0.1640625 per particle per step
    double expected = (double)n * (double)steps * 0.1640625;
    bool verified = std::fabs(ke - expected) < 1e-2;

    std::cout << std::fixed << std::setprecision(4);
    std::cout << "{\"workload\":\"lowlevel_layout\",\"variant\":\"" << variant 
              << "\",\"particles\":" << n 
              << ",\"steps\":" << steps 
              << ",\"kinetic_energy\":" << ke 
              << ",\"checksum\":\"" << (int64_t)std::round(ke) 
              << "\",\"verified\":" << (verified ? "true" : "false") << "}\n";

    return verified ? 0 : 2;
}
