#!/usr/bin/env python3
import sys
import argparse

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--dim", type=int, default=300, help="Matrix dimension N")
    args = parser.parse_args()

    n = args.dim
    a = [0.0] * (n * n)
    b = [0.0] * (n * n)
    c = [0.0] * (n * n)

    for i in range(n):
        for j in range(n):
            a[i * n + j] = float((i * 31 + j * 17) % 100) / 10.0
            b[i * n + j] = float((i * 13 + j * 23) % 100) / 10.0

    # C = A * B
    for i in range(n):
        i_off = i * n
        for k in range(n):
            aik = a[i_off + k]
            k_off = k * n
            for j in range(n):
                c[i_off + j] += aik * b[k_off + j]

    checksum = sum(c)
    print(f'{{"workload":"matrix","dimension":{n},"checksum":"{checksum:.2f}"}}')

if __name__ == "__main__":
    main()
