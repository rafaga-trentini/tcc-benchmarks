#include "common.hpp"

#include <stdexcept>
#include <utility>
#include <vector>

void executarStencilDifusao(int tamanho, int threads, int iteracoes) {
    std::vector<double> gradeAtual(tamanho * tamanho, 1.0), proximaGrade(tamanho * tamanho, 0.0);
    auto indice = [tamanho](int linha, int coluna) { return linha * tamanho + coluna; };
    auto amostraInicial = capturarAmostraRecursos();
    for (int passo = 0; passo < iteracoes; ++passo) {
#ifdef _OPENMP
#pragma omp parallel for num_threads(threads) schedule(static)
#endif
        for (int linha = 1; linha < tamanho - 1; linha++) {
            for (int coluna = 1; coluna < tamanho - 1; coluna++) {
                proximaGrade[indice(linha, coluna)] = 0.25 * (
                    gradeAtual[indice(linha - 1, coluna)] + gradeAtual[indice(linha + 1, coluna)] +
                    gradeAtual[indice(linha, coluna - 1)] + gradeAtual[indice(linha, coluna + 1)]);
            }
        }
        std::swap(gradeAtual, proximaGrade);
    }
    volatile double descarte = gradeAtual[0];
    (void)descarte;
    const int interior = std::max(0, tamanho - 2);
    const long long interior64 = static_cast<long long>(interior);
    long long itensProcessados = interior64 * interior64 * static_cast<long long>(iteracoes);
    imprimirJsonMetricas(registrarMetricas("stencil", tamanho, threads, amostraInicial, itensProcessados, 0, static_cast<long long>(iteracoes)));
}

int main(int argc, char** argv) {
    int tamanho = std::max(3, lerInteiroEnv("BENCH_SIZE", 1024));
    int threads = std::max(1, lerInteiroEnv("BENCH_THREADS", 1));
    int iteracoes = std::max(1, lerInteiroEnv("BENCH_ITERS", 100));

    for (int indice = 1; indice < argc; ++indice) {
        std::string arg = argv[indice];
        if (arg == "--size" && indice + 1 < argc) {
            tamanho = std::stoi(argv[++indice]);
        } else if (arg == "--threads" && indice + 1 < argc) {
            threads = std::stoi(argv[++indice]);
        } else if (arg == "--iters" && indice + 1 < argc) {
            iteracoes = std::stoi(argv[++indice]);
        } else {
            throw std::invalid_argument("opção desconhecida: " + arg);
        }
    }

    tamanho = std::max(3, tamanho);
    threads = std::max(1, threads);
    iteracoes = std::max(1, iteracoes);
#ifdef _OPENMP
    omp_set_num_threads(threads);
#endif
    executarStencilDifusao(tamanho, threads, iteracoes);
    return 0;
}
