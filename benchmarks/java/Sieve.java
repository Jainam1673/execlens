public class Sieve {
    public static void main(String[] args) {
        int limit = 5000000;
        for (int i = 0; i < args.length; i++) {
            if ("--limit".equals(args[i]) && i + 1 < args.length) {
                limit = Integer.parseInt(args[i + 1]);
            }
        }

        if (limit < 2) {
            System.out.printf("{\"workload\":\"sieve\",\"limit\":%d,\"count\":0,\"checksum\":\"0\"}%n", limit);
            return;
        }

        // byte[] for algorithmic parity (1 byte per candidate)
        byte[] sieve = new byte[limit + 1];
        int count = 0;

        for (long p = 2; p * p <= limit; p++) {
            if (sieve[(int)p] == 0) {
                for (long i = p * p; i <= limit; i += p) {
                    sieve[(int)i] = 1;
                }
            }
        }

        for (int p = 2; p <= limit; p++) {
            if (sieve[p] == 0) {
                count++;
            }
        }

        System.out.printf("{\"workload\":\"sieve\",\"limit\":%d,\"count\":%d,\"checksum\":\"%d\"}%n", limit, count, count);
    }
}
