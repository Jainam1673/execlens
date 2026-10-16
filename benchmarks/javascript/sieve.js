#!/usr/bin/env node

function getArg(name, def) {
    const idx = process.argv.indexOf(name);
    if (idx !== -1 && idx + 1 < process.argv.length) {
        return process.argv[idx + 1];
    }
    return def;
}

const limit = parseInt(getArg('--limit', '5000000'), 10);
if (limit < 2) {
    console.log(JSON.stringify({ workload: 'sieve', limit, count: 0, checksum: '0' }));
    process.exit(0);
}

// Uint8Array for algorithmic parity (1 byte per candidate)
const sieve = new Uint8Array(limit + 1);
let count = 0;

for (let p = 2; p * p <= limit; p++) {
    if (sieve[p] === 0) {
        for (let i = p * p; i <= limit; i += p) {
            sieve[i] = 1;
        }
    }
}

for (let p = 2; p <= limit; p++) {
    if (sieve[p] === 0) {
        count++;
    }
}

console.log(JSON.stringify({ workload: 'sieve', limit, count, checksum: count.toString() }));
