#!/usr/bin/env python3
import argparse
import os
import random
import threading

import common


class LeitoresEscritoresLock:
    def __init__(self) -> None:
        self._leitores_ativos = 0
        self._mutex_leitura = threading.Lock()
        self._mutex_escrita = threading.Lock()

    def adquirir_leitura(self) -> None:
        with self._mutex_leitura:
            self._leitores_ativos += 1
            if self._leitores_ativos == 1:
                self._mutex_escrita.acquire()

    def liberar_leitura(self) -> None:
        with self._mutex_leitura:
            self._leitores_ativos -= 1
            if self._leitores_ativos == 0:
                self._mutex_escrita.release()

    def adquirir_escrita(self) -> None:
        self._mutex_escrita.acquire()

    def liberar_escrita(self) -> None:
        self._mutex_escrita.release()


def executar_leitores_escritores(tamanho: int, threads: int, leitura_pct: int) -> None:
    operacoes_totais = tamanho * 1000
    armazenamento_compartilhado = {}
    sincronizador = LeitoresEscritoresLock()
    threads = max(1, threads)
    leitura_pct = max(0, min(100, leitura_pct))
    base_operacoes = operacoes_totais // threads
    resto_operacoes = operacoes_totais % threads
    inicio_evento = threading.Event()
    operacoes_executadas = 0
    trava_operacoes = threading.Lock()

    def simular_operacoes(semente: int, quantidade: int) -> None:
        nonlocal operacoes_executadas
        gerador = random.Random(semente)
        inicio_evento.wait()
        realizadas_local = 0
        for _ in range(quantidade):
            realizadas_local += 1
            chave = gerador.randrange(tamanho * 10 + 1)
            if gerador.randrange(100) < leitura_pct:
                sincronizador.adquirir_leitura()
                _ = armazenamento_compartilhado.get(chave)
                sincronizador.liberar_leitura()
            else:
                valor = gerador.getrandbits(32)
                sincronizador.adquirir_escrita()
                armazenamento_compartilhado[chave] = valor
                sincronizador.liberar_escrita()
        with trava_operacoes:
            operacoes_executadas += realizadas_local

    threads_trabalhadoras = []
    for indice in range(threads):
        quantidade = base_operacoes + (1 if indice < resto_operacoes else 0)
        if quantidade == 0:
            continue
        thread = threading.Thread(target=simular_operacoes, args=(1234 + indice, quantidade))
        threads_trabalhadoras.append(thread)

    for thread in threads_trabalhadoras:
        thread.start()
    inicio_ns, inicio_cpu = common.capturar_amostra_recursos()
    inicio_evento.set()
    for thread in threads_trabalhadoras:
        thread.join()

    print(common.montar_metricas(
        "rw",
        tamanho,
        threads,
        inicio_ns,
        inicio_cpu,
        operacoes_realizadas=operacoes_executadas,
    ))


def main() -> None:
    tamanho_padrao = common.ler_inteiro_env("BENCH_SIZE", 1000)
    threads_padrao = max(1, common.ler_inteiro_env("BENCH_THREADS", os.cpu_count() or 4))
    leitura_padrao = common.ler_inteiro_env("BENCH_READ_PCT", 80)

    parser = argparse.ArgumentParser(description="Leitores-Escritores em Python")
    parser.add_argument("--size", type=int, default=tamanho_padrao, help="escala de chaves")
    parser.add_argument("--threads", type=int, default=threads_padrao, help="numero de threads")
    parser.add_argument("--read_pct", type=int, default=leitura_padrao, help="percentual de leituras")
    args = parser.parse_args()

    executar_leitores_escritores(max(1, args.size), max(1, args.threads), args.read_pct)


if __name__ == "__main__":
    main()
