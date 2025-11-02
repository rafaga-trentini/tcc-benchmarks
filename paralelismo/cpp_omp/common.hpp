#pragma once

#include <algorithm>
#include <chrono>
#include <cctype>
#include <cstdint>
#include <cstdlib>
#include <cstring>
#include <iomanip>
#include <iostream>
#include <sstream>
#include <string>
#include <thread>
#include <sys/resource.h>

#ifdef _OPENMP
#include <omp.h>
#endif

struct MetricasBenchmark {
    std::string problema;
    int tamanho;
    int threads;
    double parede_ms;
    double cpu_ms;
    double cpu_pct;
    double cpu_pct_por_nucleo;
    double rss_mb;
    long long itens_processados;
    long long operacoes_realizadas;
    long long iteracoes_realizadas;
};

struct AmostraRecursos {
    std::chrono::steady_clock::time_point instante_parede;
    double cpu_ms;
};

inline double tempoCpuEmMs() {
    rusage ru{};
    if (getrusage(RUSAGE_SELF, &ru) != 0) return 0.0;
    double usuario = ru.ru_utime.tv_sec * 1000.0 + ru.ru_utime.tv_usec / 1000.0;
    double sistema = ru.ru_stime.tv_sec * 1000.0 + ru.ru_stime.tv_usec / 1000.0;
    return usuario + sistema;
}

inline double memoriaRssMb() {
    rusage ru{};
    if (getrusage(RUSAGE_SELF, &ru) != 0) return 0.0;
#ifdef __APPLE__
    return ru.ru_maxrss / (1024.0 * 1024.0);
#else
    return ru.ru_maxrss / 1024.0;
#endif
}

inline AmostraRecursos capturarAmostraRecursos() {
    return {std::chrono::steady_clock::now(), tempoCpuEmMs()};
}

inline MetricasBenchmark registrarMetricas(const std::string& problema,
                                           int tamanho,
                                           int threads,
                                           const AmostraRecursos& inicio,
                                           long long itens_processados = 0,
                                           long long operacoes_realizadas = 0,
                                           long long iteracoes_realizadas = 0) {
    auto fim = capturarAmostraRecursos();
    double parede_ms = std::chrono::duration<double, std::milli>(fim.instante_parede - inicio.instante_parede).count();
    double cpu_ms = fim.cpu_ms - inicio.cpu_ms;
    double cpu_pct = parede_ms > 0.0 ? (cpu_ms / parede_ms) * 100.0 : 0.0;
    unsigned int nucleos = std::max(1u, std::thread::hardware_concurrency());
    double cpu_pct_por_nucleo = cpu_pct / static_cast<double>(nucleos);
    return {problema, tamanho, threads, parede_ms, cpu_ms, cpu_pct, cpu_pct_por_nucleo, memoriaRssMb(), itens_processados, operacoes_realizadas, iteracoes_realizadas};
}

inline void imprimirJsonMetricas(const MetricasBenchmark& metricas) {
    std::ostringstream fluxo;
    fluxo.setf(std::ios::fixed);
    fluxo << std::setprecision(6)
          << "{\"nome_problema\":\"" << metricas.problema << "\",\"tamanho_instancia\":" << metricas.tamanho
          << ",\"quantidade_threads\":" << metricas.threads
          << ",\"tempo_decorrido_ms\":" << metricas.parede_ms
          << ",\"tempo_cpu_ms\":" << metricas.cpu_ms
          << ",\"percentual_uso_cpu\":" << metricas.cpu_pct
          << ",\"percentual_uso_cpu_por_nucleo\":" << metricas.cpu_pct_por_nucleo
          << ",\"memoria_rss_mb\":" << std::setprecision(3) << metricas.rss_mb << std::setprecision(6)
          << ",\"itens_processados\":" << metricas.itens_processados
          << ",\"operacoes_realizadas\":" << metricas.operacoes_realizadas
          << ",\"iteracoes_realizadas\":" << metricas.iteracoes_realizadas << "}";
    std::cout << fluxo.str() << '\n';
}

inline int lerInteiroEnv(const char* nome, int padrao) {
    if (const char* valor = std::getenv(nome)) {
        try {
            return std::stoi(valor);
        } catch (...) {
            return padrao;
        }
    }
    return padrao;
}

inline std::string lerStringEnv(const char* nome, const std::string& padrao = std::string()) {
    if (const char* valor = std::getenv(nome)) {
        std::string texto(valor);
        texto.erase(std::remove_if(texto.begin(), texto.end(), [](unsigned char c) { return std::isspace(c); }), texto.end());
        return texto;
    }
    return padrao;
}
