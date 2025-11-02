
import java.util.Locale;

class MetricasBenchmark {
    final String problema;
    final int tamanho;
    final int threads;
    final double paredeMs;
    final double cpuMs;
    final double cpuPct;
    final double cpuPctPorNucleo;
    final double rssMb;
    final int itensProcessados;
    final int operacoesRealizadas;
    final int iteracoesRealizadas;

    MetricasBenchmark(String problema,
                      int tamanho,
                      int threads,
                      double paredeMs,
                      double cpuMs,
                      double cpuPct,
                      double cpuPctPorNucleo,
                      double rssMb,
                      int itensProcessados,
                      int operacoesRealizadas,
                      int iteracoesRealizadas) {
        this.problema = problema;
        this.tamanho = tamanho;
        this.threads = threads;
        this.paredeMs = paredeMs;
        this.cpuMs = cpuMs;
        this.cpuPct = cpuPct;
        this.cpuPctPorNucleo = cpuPctPorNucleo;
        this.rssMb = rssMb;
        this.itensProcessados = itensProcessados;
        this.operacoesRealizadas = operacoesRealizadas;
        this.iteracoesRealizadas = iteracoesRealizadas;
    }

    String paraJson() {
        return String.format(
                Locale.US,
                "{\"nome_problema\":\"%s\",\"tamanho_instancia\":%d,\"quantidade_threads\":%d,\"tempo_decorrido_ms\":%.6f,\"tempo_cpu_ms\":%.6f,\"percentual_uso_cpu\":%.6f,\"percentual_uso_cpu_por_nucleo\":%.6f,\"memoria_rss_mb\":%.3f,\"itens_processados\":%d,\"operacoes_realizadas\":%d,\"iteracoes_realizadas\":%d}",
                problema,
                tamanho,
                threads,
                paredeMs,
                cpuMs,
                cpuPct,
                cpuPctPorNucleo,
                rssMb,
                itensProcessados,
                operacoesRealizadas,
                iteracoesRealizadas);
    }
}
