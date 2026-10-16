use std::env;

fn main() {
    let mut n: usize = 300;
    let args: Vec<String> = env::args().collect();
    for i in 1..args.len() {
        if args[i] == "--dim" && i + 1 < args.len() {
            if let Ok(val) = args[i + 1].parse::<usize>() {
                n = val;
            }
        }
    }

    let mut a = vec![0.0f64; n * n];
    let mut b = vec![0.0f64; n * n];
    let mut c = vec![0.0f64; n * n];

    for i in 0..n {
        for j in 0..n {
            a[i * n + j] = (((i * 31 + j * 17) % 100) as f64) / 10.0;
            b[i * n + j] = (((i * 13 + j * 23) % 100) as f64) / 10.0;
        }
    }

    // C = A * B
    for i in 0..n {
        let i_off = i * n;
        for k in 0..n {
            let aik = a[i_off + k];
            let k_off = k * n;
            for j in 0..n {
                c[i_off + j] += aik * b[k_off + j];
            }
        }
    }

    let mut checksum: f64 = 0.0;
    for i in 0..(n * n) {
        checksum += c[i];
    }

    println!("{{\"workload\":\"matrix\",\"dimension\":{},\"checksum\":\"{:.2}\"}}", n, checksum);
}
