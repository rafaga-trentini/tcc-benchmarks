#!/usr/bin/env python3
import argparse
import hashlib
import os
import random
import threading

import common


def executar_filosofos(rodadas: int, filosofos: int) -> None:
    filosofos = max(2, filosofos)
    rodadas = max(1, rodadas)
    garfos_disponiveis = [threading.Lock() for _ in range(filosofos)]
    apetite_total = 0
    trava_apetite = threading.Lock()
    inicio_evento = threading.Event()

    def simular_filosofo(identificador: int) -> None:
        nonlocal apetite_total
        gerador = random.Random(2024 + identificador)
        inicio_evento.wait()
        for rodada in range(rodadas):
            carga_pensamento = 0
            ciclos_pensando = gerador.randint(200, 600)
            for ciclo in range(ciclos_pensando):
                carga_pensamento += (ciclo + identificador + rodada) % 97
            garfo_esquerdo = identificador
            garfo_direito = (identificador + 1) % filosofos
            primeiro, segundo = (
                (garfo_esquerdo, garfo_direito)
                if identificador % 2 == 0
                else (garfo_direito, garfo_esquerdo)
            )
            with garfos_disponiveis[primeiro]:
                with garfos_disponiveis[segundo]:
                    resumo = hashlib.sha256(
                        f"{identificador}:{rodada}:{carga_pensamento}".encode()
                    ).digest()
                    valor = int.from_bytes(resumo[:8], "little") + carga_pensamento
                    with trava_apetite:
                        apetite_total = (apetite_total + valor) & ((1 << 64) - 1)

    threads_filosofos = [
        threading.Thread(target=simular_filosofo, args=(indice,)) for indice in range(filosofos)
    ]
    for thread in threads_filosofos:
        thread.start()
    inicio_ns, inicio_cpu = common.capturar_amostra_recursos()
    inicio_evento.set()
    for thread in threads_filosofos:
        thread.join()

    print(common.montar_metricas(
        "phil",
        rodadas,
        filosofos,
        inicio_ns,
        inicio_cpu,
        iteracoes_realizadas=filosofos * rodadas,
    ))


def main() -> None:
    rodadas_padrao = common.ler_inteiro_env("BENCH_SIZE", 1000)
    filosofos_padrao = max(2, common.ler_inteiro_env("BENCH_THREADS", os.cpu_count() or 4))

    parser = argparse.ArgumentParser(description="Jantar dos Filosofos em Python")
    parser.add_argument("--size", type=int, default=rodadas_padrao, help="numero de rodadas")
    parser.add_argument("--threads", type=int, default=filosofos_padrao, help="numero de filosofos")
    args = parser.parse_args()

    executar_filosofos(max(1, args.size), max(2, args.threads))


if __name__ == "__main__":
    main()
