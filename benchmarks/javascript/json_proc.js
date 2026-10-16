#!/usr/bin/env node
const fs = require('fs');

function getArg(name, def) {
    const idx = process.argv.indexOf(name);
    if (idx !== -1 && idx + 1 < process.argv.length) {
        return process.argv[idx + 1];
    }
    return def;
}

const inputPath = getArg('--input');
if (!inputPath) {
    console.error('Error: --input required');
    process.exit(1);
}

const raw = fs.readFileSync(inputPath, 'utf8');
const records = JSON.parse(raw);

let activeCount = 0;
let totalAmount = 0.0;

for (let i = 0; i < records.length; i++) {
    const r = records[i];
    if (r.active) {
        activeCount++;
        totalAmount += Number(r.amount || 0);
    }
}

console.log(JSON.stringify({
    workload: 'json_proc',
    active_count: activeCount,
    total_amount: Number(totalAmount.toFixed(2))
}));
