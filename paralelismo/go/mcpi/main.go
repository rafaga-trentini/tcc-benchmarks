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
    ItensProcessados    int64   `json:"itens_processados"`
    OperacoesRealizadas int64   `json:"operacoes_realizadas"`
    IteracoesRealizadas int64   `json:"iteracoes_realizadas"`
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


func registrarMetricas(nomeProblema string, tamanhoBenchmark, totalThreads int, amostraInicial amostraRecursos, itensProcessados, operacoesRealizadas, iteracoesRealizadas int64) {
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

func executarMonteCarloPi(totalAmostras, totalThreads int) {
    if totalThreads < 1 {
        totalThreads = 1
    }
    runtime.GOMAXPROCS(totalThreads)
    var pontosDentro uint64
    var grupo sync.WaitGroup
    amostrasPorThread := (totalAmostras + totalThreads - 1) / totalThreads
    amostraInicial := capturarAmostraRecursos()
    var mutex sync.Mutex
    for indiceThread := 0; indiceThread < totalThreads; indiceThread++ {
        grupo.Add(1)
        semente := int64(1234 + indiceThread)
        go func(seed int64) {
            defer grupo.Done()
            gerador := rand.New(rand.NewSource(seed))
            pontosInternosLocais := 0
            for amostra := 0; amostra < amostrasPorThread; amostra++ {
                x := gerador.Float64()
                y := gerador.Float64()
                if x*x+y*y <= 1.0 {
                    pontosInternosLocais++
                }
            }
            mutex.Lock()
            pontosDentro += uint64(pontosInternosLocais)
            mutex.Unlock()
        }(semente)
    }
    grupo.Wait()
    _ = pontosDentro
    operacoes := int64(amostrasPorThread) * int64(totalThreads)
    registrarMetricas("mcpi", totalAmostras, totalThreads, amostraInicial, 0, operacoes, 0)
}

func obterIntEnv(nome string, padrao int) int {
    if texto := os.Getenv(nome); texto != "" {
        if valor, err := strconv.Atoi(texto); err == nil {
            return valor
        }
    }
    return padrao
}

func main() {
    amostrasPadrao := obterIntEnv("BENCH_SIZE", 1024)
    threadsPadrao := obterIntEnv("BENCH_THREADS", runtime.NumCPU())

    flags := flag.NewFlagSet("mcpi", flag.ExitOnError)
    amostras := flags.Int("size", amostrasPadrao, "total de amostras")
    threads := flags.Int("threads", threadsPadrao, "numero de threads")

    if err := flags.Parse(os.Args[1:]); err != nil {
        fmt.Println("erro:", err)
        return
    }

    runtime.GOMAXPROCS(max(1, *threads))
    executarMonteCarloPi(max(1, *amostras), max(1, *threads))
}

func max(a, b int) int {
    if a > b {
        return a
    }
    return b
}
