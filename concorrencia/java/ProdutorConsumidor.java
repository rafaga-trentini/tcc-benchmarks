import java.io.BufferedInputStream;
import java.io.IOException;
import java.io.InputStream;
import java.nio.ByteBuffer;
import java.nio.ByteOrder;
import java.nio.file.DirectoryStream;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.security.MessageDigest;
import java.util.ArrayList;
import java.util.List;
import java.util.Random;
import java.util.concurrent.ArrayBlockingQueue;
import java.util.concurrent.CountDownLatch;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicInteger;
import java.util.concurrent.atomic.AtomicLong;

public class ProdutorConsumidor {
    private static final Path SENTINELA_FILA = Paths.get("__sentinela__");
    private static final Path DEFAULT_DATA_DIR = Paths.get("..", "dados_pc");

    private static Path resolverDiretorio(String argumento) {
        if (argumento != null && !argumento.trim().isEmpty()) {
            return Paths.get(argumento.trim());
        }
        return DEFAULT_DATA_DIR;
    }

    private static void garantirArquivosAleatorios(Path diretorioAlvo, int quantidadeArquivos, int tamanhoArquivo) throws IOException {
        if (!Files.exists(diretorioAlvo)) {
            Files.createDirectories(diretorioAlvo);
        }
        int arquivosExistentes = 0;
        try (DirectoryStream<Path> fluxo = Files.newDirectoryStream(diretorioAlvo)) {
            for (Path caminho : fluxo) {
                if (Files.isRegularFile(caminho) && caminho.getFileName().toString().endsWith(".bin")) {
                    arquivosExistentes++;
                }
            }
        }
        if (arquivosExistentes >= quantidadeArquivos) {
            return;
        }
        Random gerador = new Random(42);
        byte[] conteudo = new byte[tamanhoArquivo];
        for (int indice = arquivosExistentes; indice < quantidadeArquivos; indice++) {
            gerador.nextBytes(conteudo);
            Path destino = diretorioAlvo.resolve(String.format("file_%06d.bin", indice));
            Files.write(destino, conteudo);
        }
    }


private static void executarProdutorConsumidor(int tamanho, int threads, int capacidadeBuffer, String diretorioStr) throws Exception {
    Path diretorio = resolverDiretorio(diretorioStr);
    garantirArquivosAleatorios(diretorio, tamanho, 64 * 1024);
    List<Path> caminhosArquivos = new ArrayList<>();
    try (DirectoryStream<Path> fluxo = Files.newDirectoryStream(diretorio)) {
        for (Path caminho : fluxo) {
            if (Files.isRegularFile(caminho) && caminho.toString().endsWith(".bin")) {
                caminhosArquivos.add(caminho);
                if (caminhosArquivos.size() == tamanho) {
                    break;
                }
            }
        }
    }
    if (caminhosArquivos.isEmpty()) {
        System.out.println("{\"erro\":\"nenhum arquivo encontrado\"}");
        return;
    }

    threads = Math.max(2, threads);
    capacidadeBuffer = Math.max(1, capacidadeBuffer);
    int totalProdutores = Math.max(1, threads / 2);
    int totalConsumidores = threads - totalProdutores;
    if (totalConsumidores < 1) {
        totalConsumidores = 1;
        totalProdutores = Math.max(1, threads - totalConsumidores);
    }

    ArrayBlockingQueue<Path> filaArquivos = new ArrayBlockingQueue<>(capacidadeBuffer);
    AtomicLong somaHashes = new AtomicLong();
    AtomicInteger itensProcessados = new AtomicInteger();

    ExecutorService poolProdutores = Executors.newFixedThreadPool(totalProdutores);
    ExecutorService poolConsumidores = Executors.newFixedThreadPool(totalConsumidores);
    CountDownLatch inicio = new CountDownLatch(1);

    int arquivosPorProdutor = (caminhosArquivos.size() + totalProdutores - 1) / totalProdutores;
    for (int indiceProdutor = 0; indiceProdutor < totalProdutores; indiceProdutor++) {
        final int inicioIndice = indiceProdutor * arquivosPorProdutor;
        final int fimIndice = Math.min(inicioIndice + arquivosPorProdutor, caminhosArquivos.size());
        if (inicioIndice >= fimIndice) {
            break;
        }
        poolProdutores.submit(() -> {
            try {
                inicio.await();
                for (int indiceArquivo = inicioIndice; indiceArquivo < fimIndice; indiceArquivo++) {
                    filaArquivos.put(caminhosArquivos.get(indiceArquivo));
                }
            } catch (InterruptedException ex) {
                Thread.currentThread().interrupt();
            }
        });
    }

    for (int indiceConsumidor = 0; indiceConsumidor < totalConsumidores; indiceConsumidor++) {
        poolConsumidores.submit(() -> {
            byte[] bufferLeitura = new byte[1 << 20];
            try {
                MessageDigest hasher = MessageDigest.getInstance("SHA-256");
                inicio.await();
                while (true) {
                    Path arquivo = filaArquivos.take();
                    if (arquivo == SENTINELA_FILA) {
                        break;
                    }
                    try (InputStream entrada = new BufferedInputStream(Files.newInputStream(arquivo))) {
                        hasher.reset();
                        int bytesLidos;
                        while ((bytesLidos = entrada.read(bufferLeitura)) > 0) {
                            hasher.update(bufferLeitura, 0, bytesLidos);
                        }
                        byte[] digest = hasher.digest();
                        long hashTruncado = ByteBuffer.wrap(digest, 0, 8).order(ByteOrder.LITTLE_ENDIAN).getLong();
                        somaHashes.addAndGet(hashTruncado);
                    }
                    itensProcessados.incrementAndGet();
                }
            } catch (Exception ex) {
                Thread.currentThread().interrupt();
            }
        });
    }

    BenchUtils.AmostraRecursos amostra = BenchUtils.iniciarAmostra();
    inicio.countDown();

    poolProdutores.shutdown();
    poolProdutores.awaitTermination(10, TimeUnit.MINUTES);
    for (int indiceConsumidor = 0; indiceConsumidor < totalConsumidores; indiceConsumidor++) {
        filaArquivos.put(SENTINELA_FILA);
    }
    poolConsumidores.shutdown();
    poolConsumidores.awaitTermination(10, TimeUnit.MINUTES);

    if (somaHashes.get() == 0) {
        somaHashes.incrementAndGet();
    }
    BenchUtils.imprimirMetricas("pc", tamanho, threads, amostra, itensProcessados.get(), 0, 0);
}


    public static void main(String[] args) throws Exception {
        int tamanho = BenchUtils.lerInteiroEnv("BENCH_SIZE", 1000);
        int threads = Math.max(1, BenchUtils.lerInteiroEnv("BENCH_THREADS", Runtime.getRuntime().availableProcessors()));
        int capacidade = BenchUtils.lerInteiroEnv("BENCH_BUFFER", 256);
        String diretorio = System.getenv("BENCH_DIR");

        for (int indice = 0; indice < args.length; indice++) {
            switch (args[indice]) {
                case "--size":
                    tamanho = Integer.parseInt(args[++indice]);
                    break;
                case "--threads":
                    threads = Integer.parseInt(args[++indice]);
                    break;
                case "--dir":
                    diretorio = args[++indice];
                    break;
                case "--buffer":
                    capacidade = Integer.parseInt(args[++indice]);
                    break;
                default:
                    throw new IllegalArgumentException("opção desconhecida: " + args[indice]);
            }
        }

        executarProdutorConsumidor(Math.max(1, tamanho), Math.max(1, threads), Math.max(1, capacidade), diretorio);
    }
}
