use std::env;

fn partition(arr: &mut [i32], low: usize, high: usize) -> usize {
    let mid = low + (high - low) / 2;
    if arr[mid] < arr[low] { arr.swap(low, mid); }
    if arr[high] < arr[low] { arr.swap(low, high); }
    if arr[mid] < arr[high] { arr.swap(high, mid); }

    let pivot = arr[high];
    let mut i = low;
    for j in low..high {
        if arr[j] <= pivot {
            arr.swap(i, j);
            i += 1;
        }
    }
    arr.swap(i, high);
    i
}

fn quicksort(arr: &mut [i32], low: usize, high: usize) {
    if low < high {
        let p = partition(arr, low, high);
        if p > 0 {
            quicksort(arr, low, p - 1);
        }
        quicksort(arr, p + 1, high);
    }
}

fn main() {
    let mut size: usize = 200000;
    let mut mode = String::from("quicksort");

    let args: Vec<String> = env::args().collect();
    for i in 1..args.len() {
        if args[i] == "--size" && i + 1 < args.len() {
            if let Ok(val) = args[i + 1].parse::<usize>() {
                size = val;
            }
        } else if args[i] == "--mode" && i + 1 < args.len() {
            mode = args[i + 1].clone();
        }
    }

    let mut arr = vec![0i32; size];
    let mut state = 42u64;
    for i in 0..size {
        state = (state.wrapping_mul(1664525).wrapping_add(1013904223)) & 0xFFFFFFFF;
        arr[i] = state as i32;
    }

    if mode == "stdlib" {
        arr.sort_unstable();
    } else {
        if size > 0 {
            quicksort(&mut arr, 0, size - 1);
        }
    }

    let mut is_sorted = true;
    let mut checksum = 0u64;
    for i in 0..arr.len() {
        if i > 0 && arr[i] < arr[i - 1] {
            is_sorted = false;
        }
        let u32_val = arr[i] as u32 as u64;
        checksum = checksum.wrapping_mul(31).wrapping_add(u32_val);
    }

    println!("{{\"workload\":\"sort\",\"size\":{},\"mode\":\"{}\",\"sorted\":{},\"checksum\":\"{:x}\"}}",
        size, mode, is_sorted, checksum);
}
