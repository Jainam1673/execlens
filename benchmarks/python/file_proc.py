#!/usr/bin/env python3
import sys
import hashlib
import argparse

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--file", type=str, required=True, help="Path to input file")
    args = parser.parse_args()

    hasher = hashlib.sha256()
    total_bytes = 0

    with open(args.file, "rb") as f:
        while True:
            chunk = f.read(64 * 1024)
            if not chunk:
                break
            total_bytes += len(chunk)
            hasher.update(chunk)

    digest = hasher.hexdigest()
    print(f'{{"workload":"file_proc","bytes_read":{total_bytes},"sha256":"{digest}"}}')

if __name__ == "__main__":
    main()
