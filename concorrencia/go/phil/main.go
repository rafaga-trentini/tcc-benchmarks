package main

import (
    "crypto/sha256"
    "encoding/binary"
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

func executarJantarFilosofos(totalRodadas, totalFilosofos int) {
    if totalFilosofos < 2 {
        totalFilosofos = 2
    }
    if totalRodadas < 1 {
        totalRodadas = 1
    }
    garfosDisponiveis := make([]sync.Mutex, totalFilosofos)
    var acumuladorApetite uint64
    startSignal := make(chan struct{})
    var amostraInicial amostraRecursos
    var wg sync.WaitGroup
    for indiceFilosofo := 0; indiceFilosofo < totalFilosofos; indiceFilosofo++ {
        wg.Add(1)
        filosofoID := indiceFilosofo
        go func() {
            defer wg.Done()
            <-startSignal
            garfoEsquerdo := filosofoID
            garfoDireito := (filosofoID + 1) % totalFilosofos
            gerador := rand.New(rand.NewSource(int64(2024 + filosofoID)))
            for rodada := 0; rodada < totalRodadas; rodada++ {
                ciclosPensando := gerador.Intn(400) + 200
                somatorioLocal := uint64(0)
                for iteracao := 0; iteracao < ciclosPensando; iteracao++ {
                    somatorioLocal += uint64((iteracao + filosofoID + rodada) % 97)
                }
                if filosofoID%2 == 0 {
                    garfosDisponiveis[garfoEsquerdo].Lock()
                    garfosDisponiveis[garfoDireito].Lock()
                } else {
                    garfosDisponiveis[garfoDireito].Lock()
                    garfosDisponiveis[garfoEsquerdo].Lock()
                }
                dadosHash := make([]byte, 16)
                binary.LittleEndian.PutUint64(dadosHash[:8], uint64(filosofoID))
                binary.LittleEndian.PutUint64(dadosHash[8:], uint64(rodada))
                hashRodada := sha256.Sum256(dadosHash)
                atomic.AddUint64(&acumuladorApetite, binary.LittleEndian.Uint64(hashRodada[:8])+somatorioLocal)
                garfosDisponiveis[garfoEsquerdo].Unlock()
                garfosDisponiveis[garfoDireito].Unlock()
            }
        }()
    }
    amostraInicial = capturarAmostraRecursos()
    close(startSignal)
    wg.Wait()
    _ = acumuladorApetite
    iteracoesRealizadas := totalFilosofos * totalRodadas
    registrarMetricas("phil", totalRodadas, totalFilosofos, amostraInicial, 0, 0, iteracoesRealizadas)
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
    rodadasPadrao := obterIntEnv("BENCH_SIZE", 1000)
    filosofosPadrao := obterIntEnv("BENCH_THREADS", runtime.NumCPU())

    flags := flag.NewFlagSet("phil", flag.ExitOnError)
    rodadas := flags.Int("size", rodadasPadrao, "numero de rodadas de pensamento/refeicao")
    filosofos := flags.Int("threads", filosofosPadrao, "numero de filosofos")

    if err := flags.Parse(os.Args[1:]); err != nil {
        fmt.Println("erro:", err)
        return
    }

    runtime.GOMAXPROCS(max(1, *filosofos))
    executarJantarFilosofos(*rodadas, *filosofos)
}
