use std::env;
use std::fs::File;
use std::io::Read;

mod sha256 {
    const K: [u32; 64] = [
        0x428a2f98, 0x71374491, 0xb5c0fbcf, 0xe9b5dba5, 0x3956c25b, 0x59f111f1, 0x923f82a4, 0xab1c5ed5,
        0xd807aa98, 0x12835b01, 0x243185be, 0x550c7dc3, 0x72be5d74, 0x80deb1fe, 0x9bdc06a7, 0xc19bf174,
        0xe49b69c1, 0xefbe4786, 0x0fc19dc6, 0x240ca1cc, 0x2de92c6f, 0x4a7484aa, 0x5cb0a9dc, 0x76f988da,
        0x983e5152, 0xa831c66d, 0xb00327c8, 0xbf597fc7, 0xc6e00bf3, 0xd5a79147, 0x06ca6351, 0x14292967,
        0x27b70a85, 0x2e1b2138, 0x4d2c6dfc, 0x53380d13, 0x650a7354, 0x766a0abb, 0x81c2c92e, 0x92722c85,
        0xa2bfe8a1, 0xa81a664b, 0xc24b8b70, 0xc76c51a3, 0xd192e819, 0xd6990624, 0xf40e3585, 0x106aa070,
        0x19a4c116, 0x1e376c08, 0x2748774c, 0x34b0bcb5, 0x391c0cb3, 0x4ed8aa4a, 0x5b9cca4f, 0x682e6ff3,
        0x748f82ee, 0x78a5636f, 0x84c87814, 0x8cc70208, 0x90befffa, 0xa4506ceb, 0xbef9a3f7, 0xc67178f2,
    ];

    #[inline(always)]
    fn ror(x: u32, n: u32) -> u32 {
        (x >> n) | (x << (32 - n))
    }

    pub struct Sha256 {
        state: [u32; 8],
        count: u64,
        buffer: [u8; 64],
    }

    impl Sha256 {
        pub fn new() -> Self {
            Sha256 {
                state: [
                    0x6a09e667, 0xbb67ae85, 0x3c6ef372, 0xa54ff53a,
                    0x510e527f, 0x9b05688c, 0x1f83d9ab, 0x5be0cd19,
                ],
                count: 0,
                buffer: [0u8; 64],
            }
        }

        fn transform(&mut self, data: &[u8; 64]) {
            let mut a = self.state[0];
            let mut b = self.state[1];
            let mut c = self.state[2];
            let mut d = self.state[3];
            let mut e = self.state[4];
            let mut f = self.state[5];
            let mut g = self.state[6];
            let mut h = self.state[7];
            let mut m = [0u32; 64];

            for i in 0..16 {
                m[i] = ((data[i * 4] as u32) << 24)
                    | ((data[i * 4 + 1] as u32) << 16)
                    | ((data[i * 4 + 2] as u32) << 8)
                    | (data[i * 4 + 3] as u32);
            }
            for i in 16..64 {
                let s0 = ror(m[i - 15], 7) ^ ror(m[i - 15], 18) ^ (m[i - 15] >> 3);
                let s1 = ror(m[i - 2], 17) ^ ror(m[i - 2], 19) ^ (m[i - 2] >> 10);
                m[i] = m[i - 16].wrapping_add(s0).wrapping_add(m[i - 7]).wrapping_add(s1);
            }

            for i in 0..64 {
                let s1 = ror(e, 6) ^ ror(e, 11) ^ ror(e, 25);
                let ch = (e & f) ^ ((!e) & g);
                let temp1 = h.wrapping_add(s1).wrapping_add(ch).wrapping_add(K[i]).wrapping_add(m[i]);
                let s0 = ror(a, 2) ^ ror(a, 13) ^ ror(a, 22);
                let maj = (a & b) ^ (a & c) ^ (b & c);
                let temp2 = s0.wrapping_add(maj);

                h = g;
                g = f;
                f = e;
                e = d.wrapping_add(temp1);
                d = c;
                c = b;
                b = a;
                a = temp1.wrapping_add(temp2);
            }

            self.state[0] = self.state[0].wrapping_add(a);
            self.state[1] = self.state[1].wrapping_add(b);
            self.state[2] = self.state[2].wrapping_add(c);
            self.state[3] = self.state[3].wrapping_add(d);
            self.state[4] = self.state[4].wrapping_add(e);
            self.state[5] = self.state[5].wrapping_add(f);
            self.state[6] = self.state[6].wrapping_add(g);
            self.state[7] = self.state[7].wrapping_add(h);
        }

        pub fn update(&mut self, data: &[u8]) {
            for &byte in data {
                self.buffer[(self.count % 64) as usize] = byte;
                self.count += 1;
                if self.count % 64 == 0 {
                    let buf = self.buffer;
                    self.transform(&buf);
                }
            }
        }

        pub fn finish(mut self) -> String {
            let total_bits = self.count * 8;
            self.update(&[0x80]);
            while self.count % 64 != 56 {
                self.update(&[0x00]);
            }
            let mut bit_bytes = [0u8; 8];
            for i in 0..8 {
                bit_bytes[i] = ((total_bits >> ((7 - i) * 8)) & 0xFF) as u8;
            }
            self.update(&bit_bytes);

            let mut out = String::with_capacity(64);
            for val in self.state.iter() {
                out.push_str(&format!("{:08x}", val));
            }
            out
        }
    }
}

fn main() {
    let mut file_path = String::new();
    let args: Vec<String> = env::args().collect();
    for i in 1..args.len() {
        if args[i] == "--file" && i + 1 < args.len() {
            file_path = args[i + 1].clone();
        }
    }

    if file_path.is_empty() {
        eprintln!("Error: --file required");
        std::process::exit(1);
    }

    let mut file = File::open(&file_path).expect("failed to open file");
    let mut hasher = sha256::Sha256::new();
    let mut buffer = [0u8; 64 * 1024];
    let mut total_bytes: u64 = 0;

    loop {
        match file.read(&mut buffer) {
            Ok(0) => break,
            Ok(n) => {
                total_bytes += n as u64;
                hasher.update(&buffer[..n]);
            }
            Err(e) => {
                eprintln!("Error reading file: {}", e);
                std::process::exit(1);
            }
        }
    }

    let digest = hasher.finish();
    println!("{{\"workload\":\"file_proc\",\"bytes_read\":{},\"sha256\":\"{}\"}}", total_bytes, digest);
}
