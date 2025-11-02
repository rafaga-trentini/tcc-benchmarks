#include "common.hpp"

#include <random>
#include <stdexcept>
#include <vector>

void executarMultiplicacaoMatrizes(int tamanho, int threads) {
    std::vector<double> matrizA(tamanho * tamanho), matrizB(tamanho * tamanho), matrizC(tamanho * tamanho, 0.0);
    std::mt19937_64 gerador(42);
    std::uniform_real_distribution<double> distribuicao(0.0, 1.0);
    for (auto& elemento : matrizA) elemento = distribuicao(gerador);
    for (auto& elemento : matrizB) elemento = distribuicao(gerador);
    auto amostraInicial = capturarAmostraRecursos();
#ifdef _OPENMP
#pragma omp parallel for collapse(2) num_threads(threads) schedule(static)
#endif
    for (int linha = 0; linha < tamanho; linha++) {
        for (int coluna = 0; coluna < tamanho; coluna++) {
            double soma = 0.0;
            for (int indice = 0; indice < tamanho; indice++) soma += matrizA[linha * tamanho + indice] * matrizB[indice * tamanho + coluna];
            matrizC[linha * tamanho + coluna] = soma;
        }
    }
    volatile double descarte = matrizC[0];
    (void)descarte;
    const long long n = static_cast<long long>(tamanho);
    long long operacoes = 2LL * n * n * n;
    imprimirJsonMetricas(registrarMetricas("matmul", tamanho, threads, amostraInicial, 0, operacoes, 0));
}

int main(int argc, char** argv) {
    int tamanho = std::max(1, lerInteiroEnv("BENCH_SIZE", 1024));
    int threads = std::max(1, lerInteiroEnv("BENCH_THREADS", 1));

    for (int indice = 1; indice < argc; ++indice) {
        std::string arg = argv[indice];
        if (arg == "--size" && indice + 1 < argc) {
            tamanho = std::stoi(argv[++indice]);
        } else if (arg == "--threads" && indice + 1 < argc) {
            threads = std::stoi(argv[++indice]);
        } else {
            throw std::invalid_argument("opção desconhecida: " + arg);
        }
    }

#ifdef _OPENMP
    omp_set_num_threads(std::max(1, threads));
#endif
    executarMultiplicacaoMatrizes(std::max(1, tamanho), std::max(1, threads));
    return 0;
}
