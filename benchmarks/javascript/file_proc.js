#!/usr/bin/env node
const fs = require('fs');
const crypto = require('crypto');

function getArg(name, def) {
    const idx = process.argv.indexOf(name);
    if (idx !== -1 && idx + 1 < process.argv.length) {
        return process.argv[idx + 1];
    }
    return def;
}

const filePath = getArg('--file');
if (!filePath) {
    console.error('Error: --file required');
    process.exit(1);
}

const hash = crypto.createHash('sha256');
let totalBytes = 0;

const fd = fs.openSync(filePath, 'r');
const buf = Buffer.alloc(64 * 1024);

while (true) {
    const bytesRead = fs.readSync(fd, buf, 0, buf.length, null);
    if (bytesRead === 0) break;
    totalBytes += bytesRead;
    hash.update(buf.subarray(0, bytesRead));
}
fs.closeSync(fd);

const digest = hash.digest('hex');
console.log(JSON.stringify({
    workload: 'file_proc',
    bytes_read: totalBytes,
    sha256: digest
}));
