package main

import (
    "encoding/json"
    "flag"
    "fmt"
    "math/rand"
    "os"
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

func executarLeitoresEscritores(tamanhoChaves, totalThreads, percentualLeituras int) {
    if totalThreads < 1 {
        totalThreads = 1
    }
    if percentualLeituras < 0 {
        percentualLeituras = 0
    }
    if percentualLeituras > 100 {
        percentualLeituras = 100
    }
    totalOperacoes := tamanhoChaves * 1000

    type mapaProtegido struct {
        dados         map[uint64]uint64
        sincronizador sync.RWMutex
    }

    armazenamento := &mapaProtegido{dados: make(map[uint64]uint64, 1024)}
    if totalThreads < 1 {
        totalThreads = 1
    }
    baseOperacoes := totalOperacoes / totalThreads
    restoOperacoes := totalOperacoes % totalThreads

    startSignal := make(chan struct{})
    var wg sync.WaitGroup
    var operacoesExecutadas int64

    for indice := 0; indice < totalThreads; indice++ {
        quantidadeOperacoes := baseOperacoes
        if indice < restoOperacoes {
            quantidadeOperacoes++
        }
        if quantidadeOperacoes == 0 {
            continue
        }
        wg.Add(1)
        semente := int64(1234 + indice)
        go func(seed int64, totalOperacoesThread int) {
            defer wg.Done()
            gerador := rand.New(rand.NewSource(seed))
            <-startSignal
            localExecutadas := 0
            for operacao := 0; operacao < totalOperacoesThread; operacao++ {
                localExecutadas++
                identificador := uint64(gerador.Int63n(int64(tamanhoChaves*10 + 1)))
                if gerador.Intn(100) < percentualLeituras {
                    armazenamento.sincronizador.RLock()
                    _ = armazenamento.dados[identificador]
                    armazenamento.sincronizador.RUnlock()
                    continue
                }
                novoValor := uint64(gerador.Int63())
                armazenamento.sincronizador.Lock()
                armazenamento.dados[identificador] = novoValor
                armazenamento.sincronizador.Unlock()
            }
            atomic.AddInt64(&operacoesExecutadas, int64(localExecutadas))
        }(semente, quantidadeOperacoes)
    }

    amostraInicial := capturarAmostraRecursos()
    close(startSignal)

    wg.Wait()
    registrarMetricas("rw", tamanhoChaves, totalThreads, amostraInicial, 0, int(operacoesExecutadas), 0)
}

func obterIntEnv(nome string, padrao int) int {
    if texto := strings.TrimSpace(os.Getenv(nome)); texto != "" {
        if valor, err := strconv.Atoi(texto); err == nil {
            return valor
        }
    }
    return padrao
}

func max(a, b int) int {
    if a > b {
        return a
    }
    return b
}

func main() {
    tamanhoPadrao := obterIntEnv("BENCH_SIZE", 1000)
    threadsPadrao := obterIntEnv("BENCH_THREADS", runtime.NumCPU())
    leiturasPadrao := obterIntEnv("BENCH_READ_PCT", 80)

    flags := flag.NewFlagSet("rw", flag.ExitOnError)
    tamanho := flags.Int("size", tamanhoPadrao, "tamanho da chave base")
    threads := flags.Int("threads", threadsPadrao, "numero de threads")
    percentualLeitura := flags.Int("read_pct", leiturasPadrao, "percentual de leituras")

    if err := flags.Parse(os.Args[1:]); err != nil {
        fmt.Println("erro:", err)
        return
    }

    runtime.GOMAXPROCS(max(1, *threads))
    executarLeitoresEscritores(*tamanho, *threads, *percentualLeitura)
}
