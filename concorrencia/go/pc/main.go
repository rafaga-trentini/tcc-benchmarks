package main

import (
    "crypto/sha256"
    "encoding/binary"
    "encoding/json"
    "flag"
    "fmt"
    "io"
    "math/rand"
    "os"
    "path/filepath"
    "runtime"
    "strconv"
    "strings"
    "sync"
    "sync/atomic"
    "syscall"
    "time"
)

type MetricasBenchmark struct {
    Problema            string  `json:"nome_problema"`
    Tamanho             int     `json:"tamanho_instancia"`
    Threads             int     `json:"quantidade_threads"`
    ParedeMs            float64 `json:"tempo_decorrido_ms"`
    CpuMs               float64 `json:"tempo_cpu_ms"`
    CpuPct              float64 `json:"percentual_uso_cpu"`
    CpuPctPorNucleo     float64 `json:"percentual_uso_cpu_por_nucleo"`
    RSSMb               float64 `json:"memoria_rss_mb"`
    ItensProcessados    int     `json:"itens_processados"`
    OperacoesRealizadas int     `json:"operacoes_realizadas"`
    IteracoesRealizadas int     `json:"iteracoes_realizadas"`
}

type amostraRecursos struct {
    momentoParede time.Time
    consumoCpuMs  float64
}

func capturarAmostraRecursos() amostraRecursos {
    return amostraRecursos{momentoParede: time.Now(), consumoCpuMs: tempoCpuEmMs()}
}

func tempoCpuEmMs() float64 {
    var ru syscall.Rusage
    if err := syscall.Getrusage(syscall.RUSAGE_SELF, &ru); err != nil {
        return 0
    }
    usuario := float64(ru.Utime.Sec)*1000.0 + float64(ru.Utime.Usec)/1000.0
    sistema := float64(ru.Stime.Sec)*1000.0 + float64(ru.Stime.Usec)/1000.0
    return usuario + sistema
}


func memoriaRssEmMb() float64 {
    status, err := os.ReadFile("/proc/self/status")
    if err == nil {
        for _, linha := range strings.Split(string(status), "\n") {
            if strings.HasPrefix(linha, "VmHWM:") {
                campos := strings.Fields(linha)
                if len(campos) >= 2 {
                    if valor, err := strconv.ParseFloat(campos[1], 64); err == nil {
                        return valor / 1024.0
                    }
                }
                break
            }
        }
    }
    dados, err := os.ReadFile("/proc/self/statm")
    if err != nil {
        return 0.0
    }
    partes := strings.Fields(string(dados))
    if len(partes) < 2 {
        return 0.0
    }
    rssPaginas, err := strconv.ParseUint(partes[1], 10, 64)
    if err != nil {
        return 0.0
    }
    tamanhoPagina := os.Getpagesize()
    return float64(rssPaginas*uint64(tamanhoPagina)) / (1024.0 * 1024.0)
}


func registrarMetricas(nomeProblema string, tamanhoBenchmark, totalThreads int, amostraInicial amostraRecursos, itensProcessados, operacoesRealizadas, iteracoesRealizadas int) {
    amostraFinal := capturarAmostraRecursos()
    tempoParede := amostraFinal.momentoParede.Sub(amostraInicial.momentoParede).Seconds() * 1000.0
    tempoCpu := amostraFinal.consumoCpuMs - amostraInicial.consumoCpuMs
    percentualCpu := 0.0
    if tempoParede > 0 {
        percentualCpu = (tempoCpu / tempoParede) * 100.0
    }
    percentualCpuPorNucleo := 0.0
    if nucleos := runtime.NumCPU(); nucleos > 0 {
        percentualCpuPorNucleo = percentualCpu / float64(nucleos)
    }
    metricas := MetricasBenchmark{
        Problema:            nomeProblema,
        Tamanho:             tamanhoBenchmark,
        Threads:             totalThreads,
        ParedeMs:            tempoParede,
        CpuMs:               tempoCpu,
        CpuPct:              percentualCpu,
        CpuPctPorNucleo:     percentualCpuPorNucleo,
        RSSMb:               memoriaRssEmMb(),
        ItensProcessados:    itensProcessados,
        OperacoesRealizadas: operacoesRealizadas,
        IteracoesRealizadas: iteracoesRealizadas,
    }
    dadosMetricas, _ := json.Marshal(metricas)
    fmt.Println(string(dadosMetricas))
}

func garantirArquivosAleatorios(diretorioDestino string, quantidadeArquivos, tamanhoArquivo int) error {
    if diretorioDestino == "" {
        return nil
    }
    if _, err := os.Stat(diretorioDestino); os.IsNotExist(err) {
        if err := os.MkdirAll(diretorioDestino, 0o755); err != nil {
            return err
        }
    }
    entradas, err := os.ReadDir(diretorioDestino)
    if err != nil {
        return err
    }
    arquivosExistentes := 0
    for _, entrada := range entradas {
        if !entrada.IsDir() && strings.HasSuffix(entrada.Name(), ".bin") {
            arquivosExistentes++
        }
    }
    if arquivosExistentes >= quantidadeArquivos {
        return nil
    }
    buffer := make([]byte, tamanhoArquivo)
    gerador := rand.New(rand.NewSource(42))
    for indice := arquivosExistentes; indice < quantidadeArquivos; indice++ {
        if _, err := gerador.Read(buffer); err != nil {
            return err
        }
        caminho := filepath.Join(diretorioDestino, fmt.Sprintf("file_%06d.bin", indice))
        if err := os.WriteFile(caminho, buffer, 0o644); err != nil {
            return err
        }
    }
    return nil
}

func executarProdutorConsumidor(totalArquivos, totalThreads, capacidadeBuffer int, diretorioDados string) {
    if diretorioDados == "" {
        diretorioDados = defaultDataDir
    }
    if capacidadeBuffer < 1 {
        capacidadeBuffer = 1
    }
    if totalThreads < 2 {
        totalThreads = 2
    }
    if err := garantirArquivosAleatorios(diretorioDados, totalArquivos, 64*1024); err != nil {
        fmt.Println(`{"erro":"nao foi possivel gerar dados"}`)
        return
    }
    caminhosArquivos := make([]string, 0, totalArquivos)
    _ = filepath.WalkDir(diretorioDados, func(caminho string, entrada os.DirEntry, err error) error {
        if err == nil && !entrada.IsDir() && strings.HasSuffix(entrada.Name(), ".bin") {
            caminhosArquivos = append(caminhosArquivos, caminho)
        }
        return nil
    })
    if len(caminhosArquivos) == 0 {
        fmt.Println(`{"erro":"nenhum arquivo encontrado"}`)
        return
    }
    if totalArquivos < len(caminhosArquivos) {
        caminhosArquivos = caminhosArquivos[:totalArquivos]
    }
    produtores := totalThreads / 2
    if produtores < 1 {
        produtores = 1
    }
    consumidores := totalThreads - produtores
    if consumidores < 1 {
        consumidores = 1
        produtores = max(1, totalThreads-consumidores)
    }
    filaTarefas := make(chan string, capacidadeBuffer)
    var totalProduzido int64
    var totalConsumido int64
    var somaHashes uint64
    startSignal := make(chan struct{})
    var amostraInicial amostraRecursos

    var produtoresWG sync.WaitGroup
    arquivosPorProdutor := (len(caminhosArquivos) + produtores - 1) / produtores
    for indiceProdutor := 0; indiceProdutor < produtores; indiceProdutor++ {
        inicio := indiceProdutor * arquivosPorProdutor
        fim := inicio + arquivosPorProdutor
        if inicio >= len(caminhosArquivos) {
            break
        }
        if fim > len(caminhosArquivos) {
            fim = len(caminhosArquivos)
        }
        lote := append([]string(nil), caminhosArquivos[inicio:fim]...)
        produtoresWG.Add(1)
        go func() {
            defer produtoresWG.Done()
            <-startSignal
            for _, caminho := range lote {
                filaTarefas <- caminho
                atomic.AddInt64(&totalProduzido, 1)
            }
        }()
    }

    var consumidoresWG sync.WaitGroup
    for indiceConsumidor := 0; indiceConsumidor < consumidores; indiceConsumidor++ {
        consumidoresWG.Add(1)
        go func() {
            defer consumidoresWG.Done()
            bufferLeitura := make([]byte, 1<<20)
            <-startSignal
            for caminhoArquivo := range filaTarefas {
                arquivo, err := os.Open(caminhoArquivo)
                if err != nil {
                    continue
                }
                hashArquivo := sha256.New()
                for {
                    bytesLidos, er := arquivo.Read(bufferLeitura)
                    if bytesLidos > 0 {
                        hashArquivo.Write(bufferLeitura[:bytesLidos])
                    }
                    if er == io.EOF {
                        break
                    }
                    if er != nil {
                        break
                    }
                }
                arquivo.Close()
                resumo := hashArquivo.Sum(nil)
                if len(resumo) >= 8 {
                    atomic.AddUint64(&somaHashes, binary.LittleEndian.Uint64(resumo[:8]))
                }
                atomic.AddInt64(&totalConsumido, 1)
            }
        }()
    }

    amostraInicial = capturarAmostraRecursos()
    close(startSignal)

    go func() {
        produtoresWG.Wait()
        close(filaTarefas)
    }()
    consumidoresWG.Wait()
    _ = somaHashes

    itensProcessados := int(atomic.LoadInt64(&totalConsumido))
    registrarMetricas("pc", totalArquivos, totalThreads, amostraInicial, itensProcessados, 0, 0)
}

func obterIntEnv(nome string, padrao int) int {
    if texto := strings.TrimSpace(os.Getenv(nome)); texto != "" {
        if valor, err := strconv.Atoi(texto); err == nil {
            return valor
        }
    }
    return padrao
}

var defaultDataDir string

func init() {
    _, arquivo, _, ok := runtime.Caller(0)
    if ok {
        defaultDataDir = filepath.Join(filepath.Dir(arquivo), "..", "..", "dados_pc")
    } else {
        defaultDataDir = "../dados_pc"
    }
}

func main() {
    tamanhoPadrao := obterIntEnv("BENCH_SIZE", 1000)
    threadsPadrao := obterIntEnv("BENCH_THREADS", runtime.NumCPU())
    bufferPadrao := obterIntEnv("BENCH_BUFFER", 256)
    diretorioPadrao := strings.TrimSpace(os.Getenv("BENCH_DIR"))
    if diretorioPadrao == "" {
        diretorioPadrao = defaultDataDir
    }

    flags := flag.NewFlagSet("pc", flag.ExitOnError)
    tamanho := flags.Int("size", tamanhoPadrao, "tamanho/escala do benchmark")
    threads := flags.Int("threads", threadsPadrao, "numero de threads/gorrotinas")
    diretorio := flags.String("dir", diretorioPadrao, "diretorio de arquivos (padrao: data do projeto)")
    buffer := flags.Int("buffer", bufferPadrao, "capacidade do buffer")

    if err := flags.Parse(os.Args[1:]); err != nil {
        fmt.Println("erro:", err)
        return
    }

    runtime.GOMAXPROCS(max(1, *threads))
    executarProdutorConsumidor(*tamanho, *threads, *buffer, *diretorio)
}

func max(a, b int) int {
    if a > b {
        return a
    }
    return b
}
