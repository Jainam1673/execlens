#!/usr/bin/env python3
import sys
import argparse

def quicksort(arr, low, high):
    # Iterative stack or tail call optimization to avoid recursion depth limit on large arrays
    stack = [(low, high)]
    while stack:
        l, h = stack.pop()
        if l < h:
            p = partition(arr, l, h)
            # Push smaller partition first to keep stack small
            if (p - 1 - l) < (h - (p + 1)):
                stack.append((p + 1, h))
                stack.append((l, p - 1))
            else:
                stack.append((l, p - 1))
                stack.append((p + 1, h))

def partition(arr, low, high):
    mid = low + (high - low) // 2
    if arr[mid] < arr[low]:
        arr[low], arr[mid] = arr[mid], arr[low]
    if arr[high] < arr[low]:
        arr[low], arr[high] = arr[high], arr[low]
    if arr[mid] < arr[high]:
        arr[mid], arr[high] = arr[high], arr[mid]

    pivot = arr[high]
    i = low - 1
    for j in range(low, high):
        if arr[j] <= pivot:
            i += 1
            arr[i], arr[j] = arr[j], arr[i]
    arr[i + 1], arr[high] = arr[high], arr[i + 1]
    return i + 1

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--size", type=int, default=200000)
    parser.add_argument("--mode", type=str, default="quicksort")
    args = parser.parse_args()

    size = args.size
    mode = args.mode

    # Deterministic LCG generation
    state = 42
    arr = [0] * size
    for i in range(size):
        state = (state * 1664525 + 1013904223) & 0xFFFFFFFF
        # Convert to signed 32-bit int
        val = state if state < 0x80000000 else state - 0x100000000
        arr[i] = val

    if mode == "stdlib":
        arr.sort()
    else:
        quicksort(arr, 0, len(arr) - 1)

    is_sorted = True
    checksum = 0
    for i in range(len(arr)):
        if i > 0 and arr[i] < arr[i - 1]:
            is_sorted = False
        u32_val = arr[i] & 0xFFFFFFFF
        checksum = (checksum * 31 + u32_val) & 0xFFFFFFFFFFFFFFFF

    sorted_str = "true" if is_sorted else "false"
    print(f'{{"workload":"sort","size":{size},"mode":"{mode}","sorted":{sorted_str},"checksum":"{checksum:x}"}}')

if __name__ == "__main__":
    main()
