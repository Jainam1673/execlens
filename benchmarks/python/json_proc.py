#!/usr/bin/env python3
import sys
import json
import argparse

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--input", type=str, required=True, help="Path to input JSON file")
    args = parser.parse_args()

    with open(args.input, "r", encoding="utf-8") as f:
        records = json.load(f)

    active_count = 0
    total_amount = 0.0

    for r in records:
        if r.get("active", False):
            active_count += 1
            total_amount += float(r.get("amount", 0.0))

    print(f'{{"workload":"json_proc","active_count":{active_count},"total_amount":{total_amount:.2f}}}')

if __name__ == "__main__":
    main()
