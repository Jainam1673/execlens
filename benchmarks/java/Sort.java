import java.util.Arrays;

public class Sort {
    private static void quicksort(int[] arr, int low, int high) {
        if (low < high) {
            int p = partition(arr, low, high);
            quicksort(arr, low, p - 1);
            quicksort(arr, p + 1, high);
        }
    }

    private static int partition(int[] arr, int low, int high) {
        int mid = low + (high - low) / 2;
        if (arr[mid] < arr[low]) {
            int t = arr[low]; arr[low] = arr[mid]; arr[mid] = t;
        }
        if (arr[high] < arr[low]) {
            int t = arr[low]; arr[low] = arr[high]; arr[high] = t;
        }
        if (arr[mid] < arr[high]) {
            int t = arr[high]; arr[high] = arr[mid]; arr[mid] = t;
        }

        int pivot = arr[high];
        int i = low - 1;
        for (int j = low; j < high; j++) {
            if (arr[j] <= pivot) {
                i++;
                int t = arr[i]; arr[i] = arr[j]; arr[j] = t;
            }
        }
        int t = arr[i + 1]; arr[i + 1] = arr[high]; arr[high] = t;
        return i + 1;
    }

    public static void main(String[] args) {
        int size = 200000;
        String mode = "quicksort";

        for (int i = 0; i < args.length; i++) {
            if ("--size".equals(args[i]) && i + 1 < args.length) {
                size = Integer.parseInt(args[i + 1]);
            } else if ("--mode".equals(args[i]) && i + 1 < args.length) {
                mode = args[i + 1];
            }
        }

        int[] arr = new int[size];
        long state = 42L;
        for (int i = 0; i < size; i++) {
            state = (state * 1664525L + 1013904223L) & 0xFFFFFFFFL;
            arr[i] = (int) state;
        }

        if ("stdlib".equals(mode)) {
            Arrays.sort(arr);
        } else {
            quicksort(arr, 0, arr.length - 1);
        }

        boolean isSorted = true;
        long checksum = 0L;
        for (int i = 0; i < arr.length; i++) {
            if (i > 0 && arr[i] < arr[i - 1]) {
                isSorted = false;
            }
            long u32 = Integer.toUnsignedLong(arr[i]);
            checksum = (checksum * 31L + u32);
        }

        System.out.printf("{\"workload\":\"sort\",\"size\":%d,\"mode\":\"%s\",\"sorted\":%b,\"checksum\":\"%x\"}%n",
                size, mode, isSorted, checksum);
    }
}
