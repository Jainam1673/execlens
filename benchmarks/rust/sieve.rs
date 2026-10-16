use std::env;

fn main() {
    let mut limit: usize = 5000000;
    let args: Vec<String> = env::args().collect();
    for i in 1..args.len() {
        if args[i] == "--limit" && i + 1 < args.len() {
            if let Ok(val) = args[i + 1].parse::<usize>() {
                limit = val;
            }
        }
    }

    if limit < 2 {
        println!("{{\"workload\":\"sieve\",\"limit\":{},\"count\":0,\"checksum\":\"0\"}}", limit);
        return;
    }

    // Vec<u8> for 1 byte per candidate (algorithmic parity)
    let mut sieve = vec![0u8; limit + 1];
    let mut count: usize = 0;

    let mut p = 2usize;
    while p * p <= limit {
        if sieve[p] == 0 {
            let mut i = p * p;
            while i <= limit {
                sieve[i] = 1;
                i += p;
            }
        }
        p += 1;
    }

    for i in 2..=limit {
        if sieve[i] == 0 {
            count += 1;
        }
    }

    println!("{{\"workload\":\"sieve\",\"limit\":{},\"count\":{},\"checksum\":\"{}\"}}", limit, count, count);
}
