import java.nio.ByteBuffer;
import java.nio.ByteOrder;
import java.security.MessageDigest;
import java.util.Random;
import java.util.concurrent.CountDownLatch;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicLong;
import java.util.concurrent.locks.ReentrantLock;

public class JantarFilosofos {

private static void executarJantarFilosofos(int rodadas, int filosofos) throws Exception {
        final int totalFilosofos = Math.max(2, filosofos);
        final int totalRodadas = Math.max(1, rodadas);
        ReentrantLock[] garfosDisponiveis = new ReentrantLock[totalFilosofos];
        for (int indiceGarfo = 0; indiceGarfo < totalFilosofos; indiceGarfo++) {
            garfosDisponiveis[indiceGarfo] = new ReentrantLock(true);
        }
        AtomicLong apetiteTotal = new AtomicLong();
        ExecutorService pool = Executors.newFixedThreadPool(totalFilosofos);
        CountDownLatch inicio = new CountDownLatch(1);

        for (int indiceFilosofo = 0; indiceFilosofo < totalFilosofos; indiceFilosofo++) {
            final int identificador = indiceFilosofo;
            pool.submit(() -> {
                Random gerador = new Random(2024 + identificador);
                try {
                    MessageDigest hasher = MessageDigest.getInstance("SHA-256");
                    inicio.await();
                    for (int rodadaAtual = 0; rodadaAtual < totalRodadas; rodadaAtual++) {
                        long cargaPensamento = 0;
                        int ciclosPensando = gerador.nextInt(400) + 200;
                        for (int ciclo = 0; ciclo < ciclosPensando; ciclo++) {
                            cargaPensamento += (ciclo + identificador + rodadaAtual) % 97;
                        }
                        int garfoEsquerdo = identificador;
                        int garfoDireito = (identificador + 1) % totalFilosofos;
                        if (identificador % 2 == 0) {
                            garfosDisponiveis[garfoEsquerdo].lock();
                            garfosDisponiveis[garfoDireito].lock();
                        } else {
                            garfosDisponiveis[garfoDireito].lock();
                            garfosDisponiveis[garfoEsquerdo].lock();
                        }
                        hasher.reset();
                        byte[] dados = ByteBuffer.allocate(16)
                                .order(ByteOrder.LITTLE_ENDIAN)
                                .putLong(identificador)
                                .putLong(rodadaAtual)
                                .array();
                        byte[] resumo = hasher.digest(dados);
                        long valor = ByteBuffer.wrap(resumo, 0, 8).order(ByteOrder.LITTLE_ENDIAN).getLong() + cargaPensamento;
                        apetiteTotal.addAndGet(valor);
                        garfosDisponiveis[garfoEsquerdo].unlock();
                        garfosDisponiveis[garfoDireito].unlock();
                    }
                } catch (Exception ex) {
                    Thread.currentThread().interrupt();
                }
            });
        }

        BenchUtils.AmostraRecursos amostra = BenchUtils.iniciarAmostra();
        inicio.countDown();

        pool.shutdown();
        pool.awaitTermination(10, TimeUnit.MINUTES);
        if (apetiteTotal.get() == 0) {
            apetiteTotal.incrementAndGet();
        }
        int iteracoesRealizadas = totalFilosofos * totalRodadas;
        BenchUtils.imprimirMetricas("phil", totalRodadas, totalFilosofos, amostra, 0, 0, iteracoesRealizadas);
    }



    public static void main(String[] args) throws Exception {
        int rodadas = BenchUtils.lerInteiroEnv("BENCH_SIZE", 1000);
        int filosofos = Math.max(2, BenchUtils.lerInteiroEnv("BENCH_THREADS", Runtime.getRuntime().availableProcessors()));

        for (int indice = 0; indice < args.length; indice++) {
            switch (args[indice]) {
                case "--size":
                    rodadas = Integer.parseInt(args[++indice]);
                    break;
                case "--threads":
                    filosofos = Integer.parseInt(args[++indice]);
                    break;
                default:
                    throw new IllegalArgumentException("opção desconhecida: " + args[indice]);
            }
        }

        executarJantarFilosofos(Math.max(1, rodadas), Math.max(2, filosofos));
    }
}
