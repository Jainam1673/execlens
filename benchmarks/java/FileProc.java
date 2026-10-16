import java.io.FileInputStream;
import java.security.MessageDigest;

public class FileProc {
    public static void main(String[] args) throws Exception {
        String filePath = "";
        for (int i = 0; i < args.length; i++) {
            if ("--file".equals(args[i]) && i + 1 < args.length) {
                filePath = args[i + 1];
            }
        }

        if (filePath.isEmpty()) {
            System.err.println("Error: --file required");
            System.exit(1);
        }

        MessageDigest md = MessageDigest.getInstance("SHA-256");
        long totalBytes = 0;

        try (FileInputStream fis = new FileInputStream(filePath)) {
            byte[] buf = new byte[64 * 1024];
            int n;
            while ((n = fis.read(buf)) != -1) {
                totalBytes += n;
                md.update(buf, 0, n);
            }
        }

        byte[] digest = md.digest();
        StringBuilder sb = new StringBuilder();
        for (byte b : digest) {
            sb.append(String.format("%02x", b));
        }

        System.out.printf("{\"workload\":\"file_proc\",\"bytes_read\":%d,\"sha256\":\"%s\"}%n",
                totalBytes, sb.toString());
    }
}
