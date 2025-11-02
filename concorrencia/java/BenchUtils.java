import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.time.Duration;
import java.util.List;

final class BenchUtils {
    private BenchUtils() {}

    static final class AmostraRecursos {
        final long inicioNs;
        final double cpuInicialMs;

        AmostraRecursos(long inicioNs, double cpuInicialMs) {
            this.inicioNs = inicioNs;
            this.cpuInicialMs = cpuInicialMs;
        }
    }

    static AmostraRecursos iniciarAmostra() {
        return new AmostraRecursos(System.nanoTime(), tempoCpuEmMs());
    }

    static void imprimirMetricas(String problema,
                                     int tamanho,
                                     int threads,
                                     AmostraRecursos amostra,
                                     int itensProcessados,
                                     int operacoesRealizadas,
                                     int iteracoesRealizadas) {
        long paredeNs = System.nanoTime() - amostra.inicioNs;
        double paredeMs = paredeNs / 1_000_000.0;
        double cpuMs = tempoCpuEmMs() - amostra.cpuInicialMs;
        double cpuPct = paredeMs > 0.0 ? (cpuMs / paredeMs) * 100.0 : 0.0;
        int nucleos = Math.max(1, Runtime.getRuntime().availableProcessors());
        double cpuPctPorNucleo = cpuPct / nucleos;
        MetricasBenchmark metricas = new MetricasBenchmark(
                problema,
                tamanho,
                threads,
                paredeMs,
                cpuMs,
                cpuPct,
                cpuPctPorNucleo,
                obterMemoriaRssMb(),
                itensProcessados,
                operacoesRealizadas,
                iteracoesRealizadas);
        System.out.println(metricas.paraJson());
    }

    static int lerInteiroEnv(String nome, int padrao) {
        String valor = System.getenv(nome);
        if (valor == null) {
            return padrao;
        }
        valor = valor.trim();
        if (valor.isEmpty()) {
            return padrao;
        }
        try {
            return Integer.parseInt(valor);
        } catch (NumberFormatException ignored) {
            return padrao;
        }
    }

    private static double tempoCpuEmMs() {
        return ProcessHandle.current()
                .info()
                .totalCpuDuration()
                .map(Duration::toNanos)
                .orElse(0L) / 1_000_000.0;
    }

    private static double obterMemoriaRssMb() {
        double picoKb = -1.0;
        double atualKb = -1.0;
        try {
            List<String> linhas = Files.readAllLines(Path.of("/proc/self/status"));
            for (String linha : linhas) {
                if (linha.startsWith("VmHWM:")) {
                    String[] partes = linha.trim().split("\\s+");
                    if (partes.length >= 2) {
                        picoKb = Double.parseDouble(partes[1]);
                        break;
                    }
                }
                if (linha.startsWith("VmRSS:")) {
                    String[] partes = linha.trim().split("\\s+");
                    if (partes.length >= 2) {
                        atualKb = Double.parseDouble(partes[1]);
                    }
                }
            }
        } catch (IOException ignored) {
        }
        double kb = picoKb >= 0.0 ? picoKb : Math.max(atualKb, 0.0);
        return kb > 0.0 ? kb / 1024.0 : 0.0;
    }

}
