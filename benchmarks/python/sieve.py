#!/usr/bin/env python3
import sys
import argparse

def main():
    parser = argparse.ArgumentParser(description="Sieve of Eratosthenes")
    parser.add_argument("--limit", type=int, default=5000000, help="Prime limit")
    args = parser.parse_args()

    limit = args.limit
    if limit < 2:
        print(f'{{"workload":"sieve","limit":{limit},"count":0,"checksum":"0"}}')
        return

    # bytearray for algorithmic parity (1 byte per candidate)
    sieve = bytearray(limit + 1)
    count = 0

    p = 2
    while p * p <= limit:
        if sieve[p] == 0:
            for i in range(p * p, limit + 1, p):
                sieve[i] = 1
        p += 1

    for p in range(2, limit + 1):
        if sieve[p] == 0:
            count += 1

    print(f'{{"workload":"sieve","limit":{limit},"count":{count},"checksum":"{count}"}}')

if __name__ == "__main__":
    main()
