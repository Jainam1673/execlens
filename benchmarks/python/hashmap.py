#!/usr/bin/env python3
import sys
import argparse

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--operations", type=int, default=300000)
    args = parser.parse_args()

    ops = args.operations
    half = ops // 2
    d = {}

    for i in range(ops):
        key = f"k_{i % half}"
        d[key] = i * 3

    checksum = 0
    for i in range(ops):
        key = f"k_{i}"
        if key in d:
            checksum += d[key]
        else:
            checksum -= 1

    print(f'{{"workload":"hashmap","operations":{ops},"checksum":"{checksum}"}}')

if __name__ == "__main__":
    main()
