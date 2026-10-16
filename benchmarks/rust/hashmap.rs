use std::collections::HashMap;
use std::env;

fn main() {
    let mut operations: usize = 300000;
    let args: Vec<String> = env::args().collect();
    for i in 1..args.len() {
        if args[i] == "--operations" && i + 1 < args.len() {
            if let Ok(val) = args[i + 1].parse::<usize>() {
                operations = val;
            }
        }
    }

    let half = operations / 2;
    let mut map: HashMap<String, i64> = HashMap::with_capacity(half);

    for i in 0..operations {
        let key = format!("k_{}", i % half);
        map.insert(key, (i as i64) * 3);
    }

    let mut checksum: i64 = 0;
    for i in 0..operations {
        let key = format!("k_{}", i);
        if let Some(&val) = map.get(&key) {
            checksum += val;
        } else {
            checksum -= 1;
        }
    }

    println!("{{\"workload\":\"hashmap\",\"operations\":{},\"checksum\":\"{}\"}}", operations, checksum);
}
