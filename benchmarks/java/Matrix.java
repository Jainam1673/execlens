public class Matrix {
    public static void main(String[] args) {
        int n = 300;
        for (int i = 0; i < args.length; i++) {
            if ("--dim".equals(args[i]) && i + 1 < args.length) {
                n = Integer.parseInt(args[i + 1]);
            }
        }

        double[] a = new double[n * n];
        double[] b = new double[n * n];
        double[] c = new double[n * n];

        for (int i = 0; i < n; i++) {
            for (int j = 0; j < n; j++) {
                a[i * n + j] = ((i * 31 + j * 17) % 100) / 10.0;
                b[i * n + j] = ((i * 13 + j * 23) % 100) / 10.0;
            }
        }

        // C = A * B
        for (int i = 0; i < n; i++) {
            int iOff = i * n;
            for (int k = 0; k < n; k++) {
                double aik = a[iOff + k];
                int kOff = k * n;
                for (int j = 0; j < n; j++) {
                    c[iOff + j] += aik * b[kOff + j];
                }
            }
        }

        double checksum = 0.0;
        for (int i = 0; i < n * n; i++) {
            checksum += c[i];
        }

        System.out.printf("{\"workload\":\"matrix\",\"dimension\":%d,\"checksum\":\"%.2f\"}%n", n, checksum);
    }
}
