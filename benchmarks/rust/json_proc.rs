use std::env;
use std::fs::File;
use std::io::{BufRead, BufReader};

fn main() {
    let mut input_path = String::new();
    let args: Vec<String> = env::args().collect();
    for i in 1..args.len() {
        if args[i] == "--input" && i + 1 < args.len() {
            input_path = args[i + 1].clone();
        }
    }

    if input_path.is_empty() {
        eprintln!("Error: --input required");
        std::process::exit(1);
    }

    let file = File::open(&input_path).expect("failed to open input file");
    let reader = BufReader::new(file);

    let mut active_count: usize = 0;
    let mut total_amount: f64 = 0.0;

    for line_result in reader.lines() {
        if let Ok(line) = line_result {
            if let Some(active_pos) = line.find("\"active\":") {
                let remainder = &line[active_pos..];
                let is_active = remainder.contains("true");

                if let Some(amt_pos) = line.find("\"amount\":") {
                    let amt_part = &line[amt_pos + 9..];
                    let trimmed = amt_part.trim_start();
                    let num_str: String = trimmed.chars()
                        .take_while(|c| c.is_digit(10) || *c == '.')
                        .collect();
                    if let Ok(amount) = num_str.parse::<f64>() {
                        if is_active {
                            active_count += 1;
                            total_amount += amount;
                        }
                    }
                }
            }
        }
    }

    println!("{{\"workload\":\"json_proc\",\"active_count\":{},\"total_amount\":{:.2}}}",
        active_count, total_amount);
}
