package main

import (
    "encoding/json"
    "flag"
    "fmt"
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

func executarStencilDifusao(tamanhoGrade, totalThreads, iteracoes int) {
    if totalThreads < 1 {
        totalThreads = 1
    }
    runtime.GOMAXPROCS(totalThreads)
    gradeAtual := make([]float64, tamanhoGrade*tamanhoGrade)
    proximaGrade := make([]float64, tamanhoGrade*tamanhoGrade)
    for indice := range gradeAtual {
        gradeAtual[indice] = 1.0
    }
    calcularIndice := func(linha, coluna int) int {
        return linha*tamanhoGrade + coluna
    }
    amostraInicial := capturarAmostraRecursos()
    type tarefa struct {
        linha         int
        gradeAtual    []float64
        proximaGrade  []float64
        grupoSincronia *sync.WaitGroup
    }
    trabalhos := make(chan tarefa, totalThreads)
    for worker := 0; worker < totalThreads; worker++ {
        go func() {
            for job := range trabalhos {
                for coluna := 1; coluna < tamanhoGrade-1; coluna++ {
                    somaVizinhos := job.gradeAtual[calcularIndice(job.linha-1, coluna)] +
                        job.gradeAtual[calcularIndice(job.linha+1, coluna)] +
                        job.gradeAtual[calcularIndice(job.linha, coluna-1)] +
                        job.gradeAtual[calcularIndice(job.linha, coluna+1)]
                    job.proximaGrade[calcularIndice(job.linha, coluna)] = 0.25 * somaVizinhos
                }
                job.grupoSincronia.Done()
            }
        }()
    }
    for ciclo := 0; ciclo < iteracoes; ciclo++ {
        var grupo sync.WaitGroup
        for linha := 1; linha < tamanhoGrade-1; linha++ {
            grupo.Add(1)
            trabalhos <- tarefa{
                linha:         linha,
                gradeAtual:    gradeAtual,
                proximaGrade:  proximaGrade,
                grupoSincronia: &grupo,
            }
        }
        grupo.Wait()
        gradeAtual, proximaGrade = proximaGrade, gradeAtual
    }
    close(trabalhos)
    _ = gradeAtual[0]
    celulas := max(0, tamanhoGrade-2)
    celulas64 := int64(celulas)
    itensProcessados := celulas64 * celulas64 * int64(iteracoes)
    registrarMetricas("stencil", tamanhoGrade, totalThreads, amostraInicial, itensProcessados, 0, int64(iteracoes))
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
    tamanhoPadrao := obterIntEnv("BENCH_SIZE", 1024)
    threadsPadrao := obterIntEnv("BENCH_THREADS", runtime.NumCPU())
    iteracoesPadrao := obterIntEnv("BENCH_ITERS", 100)

    flags := flag.NewFlagSet("stencil", flag.ExitOnError)
    tamanho := flags.Int("size", tamanhoPadrao, "tamanho da grade quadrada")
    threads := flags.Int("threads", threadsPadrao, "numero de threads")
    iteracoes := flags.Int("iters", iteracoesPadrao, "numero de iteracoes")

    if err := flags.Parse(os.Args[1:]); err != nil {
        fmt.Println("erro:", err)
        return
    }

    runtime.GOMAXPROCS(max(1, *threads))
    executarStencilDifusao(max(3, *tamanho), max(1, *threads), max(1, *iteracoes))
}

func max(a, b int) int {
    if a > b {
        return a
    }
    return b
}
