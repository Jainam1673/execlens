#!/usr/bin/env node

function getArg(name, def) {
    const idx = process.argv.indexOf(name);
    if (idx !== -1 && idx + 1 < process.argv.length) {
        return process.argv[idx + 1];
    }
    return def;
}

const n = parseInt(getArg('--dim', '300'), 10);
const a = new Float64Array(n * n);
const b = new Float64Array(n * n);
const c = new Float64Array(n * n);

for (let i = 0; i < n; i++) {
    for (let j = 0; j < n; j++) {
        a[i * n + j] = ((i * 31 + j * 17) % 100) / 10.0;
        b[i * n + j] = ((i * 13 + j * 23) % 100) / 10.0;
    }
}

// C = A * B
for (let i = 0; i < n; i++) {
    const iOff = i * n;
    for (let k = 0; k < n; k++) {
        const aik = a[iOff + k];
        const kOff = k * n;
        for (let j = 0; j < n; j++) {
            c[iOff + j] += aik * b[kOff + j];
        }
    }
}

let checksum = 0.0;
for (let i = 0; i < n * n; i++) {
    checksum += c[i];
}

console.log(JSON.stringify({
    workload: 'matrix',
    dimension: n,
    checksum: checksum.toFixed(2)
}));
