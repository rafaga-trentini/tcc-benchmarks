import java.util.Random;
import java.util.concurrent.CountDownLatch;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.LongAdder;
import java.util.concurrent.locks.ReentrantReadWriteLock;

public class LeitoresEscritores {

private static void executarLeitoresEscritores(int tamanho, int threads, int leituraPct) throws Exception {
        final int threadsAjustados = Math.max(1, threads);
        final int leituraAjustada = Math.max(0, Math.min(100, leituraPct));
        final int operacoesTotais = tamanho * 1000;
        final java.util.concurrent.ConcurrentHashMap<Long, Long> armazenamentoCompartilhado = new java.util.concurrent.ConcurrentHashMap<>();
        final ReentrantReadWriteLock sincronizador = new ReentrantReadWriteLock();

        ExecutorService pool = Executors.newFixedThreadPool(threadsAjustados);
        CountDownLatch inicio = new CountDownLatch(1);
        LongAdder operacoesExecutadas = new LongAdder();

        int baseOperacoes = operacoesTotais / threadsAjustados;
        int restoOperacoes = operacoesTotais % threadsAjustados;

        for (int indiceThread = 0; indiceThread < threadsAjustados; indiceThread++) {
            final long semente = 1234L + indiceThread;
            final int quantidadeOperacoes = baseOperacoes + (indiceThread < restoOperacoes ? 1 : 0);
            if (quantidadeOperacoes == 0) {
                continue;
            }
            pool.submit(() -> {
                Random gerador = new Random(semente);
                try {
                    inicio.await();
                } catch (InterruptedException ex) {
                    Thread.currentThread().interrupt();
                    return;
                }
                for (int iteracao = 0; iteracao < quantidadeOperacoes; iteracao++) {
                    long chave = gerador.nextInt(tamanho * 10 + 1);
                    if (gerador.nextInt(100) < leituraAjustada) {
                        sincronizador.readLock().lock();
                        try {
                            armazenamentoCompartilhado.get(chave);
                        } finally {
                            sincronizador.readLock().unlock();
                        }
                    } else {
                        long valor = gerador.nextLong();
                        sincronizador.writeLock().lock();
                        try {
                            armazenamentoCompartilhado.put(chave, valor);
                        } finally {
                            sincronizador.writeLock().unlock();
                        }
                    }
                }
                operacoesExecutadas.add(quantidadeOperacoes);
            });
        }

        BenchUtils.AmostraRecursos amostra = BenchUtils.iniciarAmostra();
        inicio.countDown();

        pool.shutdown();
        pool.awaitTermination(10, TimeUnit.MINUTES);
        BenchUtils.imprimirMetricas("rw", tamanho, threadsAjustados, amostra, 0, operacoesExecutadas.intValue(), 0);
    }



    public static void main(String[] args) throws Exception {
        int tamanho = BenchUtils.lerInteiroEnv("BENCH_SIZE", 1000);
        int threads = Math.max(1, BenchUtils.lerInteiroEnv("BENCH_THREADS", Runtime.getRuntime().availableProcessors()));
        int leituraPct = BenchUtils.lerInteiroEnv("BENCH_READ_PCT", 80);

        for (int indice = 0; indice < args.length; indice++) {
            switch (args[indice]) {
                case "--size":
                    tamanho = Integer.parseInt(args[++indice]);
                    break;
                case "--threads":
                    threads = Integer.parseInt(args[++indice]);
                    break;
                case "--read_pct":
                    leituraPct = Integer.parseInt(args[++indice]);
                    break;
                default:
                    throw new IllegalArgumentException("opção desconhecida: " + args[indice]);
            }
        }

        executarLeitoresEscritores(Math.max(1, tamanho), Math.max(1, threads), leituraPct);
    }
}
