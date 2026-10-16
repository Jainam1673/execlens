import java.util.HashMap;

public class Hashmap {
    public static void main(String[] args) {
        int operations = 300000;
        for (int i = 0; i < args.length; i++) {
            if ("--operations".equals(args[i]) && i + 1 < args.length) {
                operations = Integer.parseInt(args[i + 1]);
            }
        }

        int half = operations / 2;
        HashMap<String, Long> map = new HashMap<>(half);

        for (int i = 0; i < operations; i++) {
            String key = "k_" + (i % half);
            map.put(key, (long) i * 3);
        }

        long checksum = 0L;
        for (int i = 0; i < operations; i++) {
            String key = "k_" + i;
            Long val = map.get(key);
            if (val != null) {
                checksum += val;
            } else {
                checksum -= 1L;
            }
        }

        System.out.printf("{\"workload\":\"hashmap\",\"operations\":%d,\"checksum\":\"%d\"}%n", operations, checksum);
    }
}
