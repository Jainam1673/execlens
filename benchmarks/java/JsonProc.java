import java.io.BufferedReader;
import java.io.FileReader;
import java.util.regex.Matcher;
import java.util.regex.Pattern;

public class JsonProc {
    public static void main(String[] args) throws Exception {
        String inputPath = "";
        for (int i = 0; i < args.length; i++) {
            if ("--input".equals(args[i]) && i + 1 < args.length) {
                inputPath = args[i + 1];
            }
        }

        if (inputPath.isEmpty()) {
            System.err.println("Error: --input required");
            System.exit(1);
        }

        int activeCount = 0;
        double totalAmount = 0.0;

        Pattern activePattern = Pattern.compile("\"active\"\\s*:\\s*(true|false)");
        Pattern amountPattern = Pattern.compile("\"amount\"\\s*:\\s*([0-9.]+)");

        try (BufferedReader reader = new BufferedReader(new FileReader(inputPath))) {
            String line;
            while ((line = reader.readLine()) != null) {
                Matcher mActive = activePattern.matcher(line);
                Matcher mAmount = amountPattern.matcher(line);

                if (mActive.find() && mAmount.find()) {
                    boolean active = "true".equals(mActive.group(1));
                    double amount = Double.parseDouble(mAmount.group(1));

                    if (active) {
                        activeCount++;
                        totalAmount += amount;
                    }
                }
            }
        }

        System.out.printf("{\"workload\":\"json_proc\",\"active_count\":%d,\"total_amount\":%.2f}%n",
                activeCount, totalAmount);
    }
}
