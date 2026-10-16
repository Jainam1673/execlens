#!/usr/bin/env node

function getArg(name, def) {
    const idx = process.argv.indexOf(name);
    if (idx !== -1 && idx + 1 < process.argv.length) {
        return process.argv[idx + 1];
    }
    return def;
}

function quicksort(arr, low, high) {
    const stack = [[low, high]];
    while (stack.length > 0) {
        const [l, h] = stack.pop();
        if (l < h) {
            const p = partition(arr, l, h);
            if ((p - 1 - l) < (h - (p + 1))) {
                stack.push([p + 1, h]);
                stack.push([l, p - 1]);
            } else {
                stack.push([l, p - 1]);
                stack.push([p + 1, h]);
            }
        }
    }
}

function partition(arr, low, high) {
    const mid = low + ((high - low) >> 1);
    if (arr[mid] < arr[low]) {
        const t = arr[low]; arr[low] = arr[mid]; arr[mid] = t;
    }
    if (arr[high] < arr[low]) {
        const t = arr[low]; arr[low] = arr[high]; arr[high] = t;
    }
    if (arr[mid] < arr[high]) {
        const t = arr[high]; arr[high] = arr[mid]; arr[mid] = t;
    }

    const pivot = arr[high];
    let i = low - 1;
    for (let j = low; j < high; j++) {
        if (arr[j] <= pivot) {
            i++;
            const t = arr[i]; arr[i] = arr[j]; arr[j] = t;
        }
    }
    const t = arr[i + 1]; arr[i + 1] = arr[high]; arr[high] = t;
    return i + 1;
}

const size = parseInt(getArg('--size', '200000'), 10);
const mode = getArg('--mode', 'quicksort');

const arr = new Int32Array(size);
let state = 42n;
for (let i = 0; i < size; i++) {
    state = (state * 1664525n + 1013904223n) & 0xFFFFFFFFn;
    // Map to signed 32-bit integer
    const u32 = Number(state);
    arr[i] = u32 > 0x7FFFFFFF ? u32 - 0x100000000 : u32;
}

if (mode === 'stdlib') {
    arr.sort();
} else {
    quicksort(arr, 0, arr.length - 1);
}

let isSorted = true;
let checksum = 0n;
for (let i = 0; i < arr.length; i++) {
    if (i > 0 && arr[i] < arr[i - 1]) {
        isSorted = false;
    }
    const u32 = BigInt(arr[i] >>> 0);
    checksum = (checksum * 31n + u32) & 0xFFFFFFFFFFFFFFFFn;
}

console.log(JSON.stringify({
    workload: 'sort',
    size,
    mode,
    sorted: isSorted,
    checksum: checksum.toString(16)
}));
