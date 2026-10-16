#!/usr/bin/env node

function getArg(name, def) {
    const idx = process.argv.indexOf(name);
    if (idx !== -1 && idx + 1 < process.argv.length) {
        return process.argv[idx + 1];
    }
    return def;
}

const operations = parseInt(getArg('--operations', '300000'), 10);
const half = Math.floor(operations / 2);
const map = new Map();

for (let i = 0; i < operations; i++) {
    const key = `k_${i % half}`;
    map.set(key, i * 3);
}

let checksum = 0n;
for (let i = 0; i < operations; i++) {
    const key = `k_${i}`;
    const val = map.get(key);
    if (val !== undefined) {
        checksum += BigInt(val);
    } else {
        checksum -= 1n;
    }
}

console.log(JSON.stringify({
    workload: 'hashmap',
    operations,
    checksum: checksum.toString()
}));
