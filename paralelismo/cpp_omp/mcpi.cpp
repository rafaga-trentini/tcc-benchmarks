#include "common.hpp"

#include <atomic>
#include <random>
#include <stdexcept>
#include <vector>

void executarMonteCarloPi(long long totalAmostras, int threads) {
    std::atomic<long long> pontosDentro(0);
    auto amostraInicial = capturarAmostraRecursos();
    long long amostrasPorThread = (totalAmostras + threads - 1) / threads;
#ifdef _OPENMP
#pragma omp parallel num_threads(threads)
    {
        std::mt19937_64 gerador(1234 + omp_get_thread_num());
#else
    std::vector<std::mt19937_64> geradores(threads);
    for (int indice = 0; indice < threads; indice++) geradores[indice].seed(1234 + indice);
#pragma omp parallel for num_threads(1)
    for (int indice = 0; indice < threads; indice++) {
        std::mt19937_64& gerador = geradores[indice];
#endif
        std::uniform_real_distribution<double> distribuicao(0.0, 1.0);
        long long pontosLocais = 0;
        for (long long amostra = 0; amostra < amostrasPorThread; amostra++) {
            double x = distribuicao(gerador);
            double y = distribuicao(gerador);
            if (x * x + y * y <= 1.0) pontosLocais++;
        }
        pontosDentro.fetch_add(pontosLocais, std::memory_order_relaxed);
#ifdef _OPENMP
    }
#else
    }
#endif
    double piEstimado = 4.0 * static_cast<double>(pontosDentro.load()) /
                        static_cast<double>(((totalAmostras + threads - 1) / threads) * threads);
    volatile double descarte = piEstimado;
    (void)descarte;
    long long operacoes = amostrasPorThread * static_cast<long long>(threads);
    imprimirJsonMetricas(registrarMetricas("mcpi", static_cast<int>(totalAmostras), threads, amostraInicial, 0, operacoes, 0));
}

int main(int argc, char** argv) {
    long long amostras = std::max<long long>(1, lerInteiroEnv("BENCH_SIZE", 1024));
    int threads = std::max(1, lerInteiroEnv("BENCH_THREADS", 1));

    for (int indice = 1; indice < argc; ++indice) {
        std::string arg = argv[indice];
        if (arg == "--size" && indice + 1 < argc) {
            amostras = std::stoll(argv[++indice]);
        } else if (arg == "--threads" && indice + 1 < argc) {
            threads = std::stoi(argv[++indice]);
        } else {
            throw std::invalid_argument("opção desconhecida: " + arg);
        }
    }

    amostras = std::max<long long>(1, amostras);
    threads = std::max(1, threads);
#ifdef _OPENMP
    omp_set_num_threads(threads);
#endif
    executarMonteCarloPi(amostras, threads);
    return 0;
}
