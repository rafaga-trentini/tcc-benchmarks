#!/usr/bin/env python3
import argparse
import hashlib
import os
import queue
import random
import threading
from pathlib import Path

import common


def garantir_arquivos_binarios(diretorio_alvo: Path, quantidade_arquivos: int, tamanho_arquivo: int) -> None:
    diretorio_alvo.mkdir(parents=True, exist_ok=True)
    arquivos_existentes = [p for p in diretorio_alvo.iterdir() if p.suffix == ".bin"]
    if len(arquivos_existentes) >= quantidade_arquivos:
        return
    gerador_aleatorio = random.Random(42)
    buffer_bytes = bytearray(tamanho_arquivo)
    for indice_arquivo in range(len(arquivos_existentes), quantidade_arquivos):
        for posicao in range(tamanho_arquivo):
            buffer_bytes[posicao] = gerador_aleatorio.getrandbits(8)
        caminho_destino = diretorio_alvo / f"file_{indice_arquivo:06d}.bin"
        caminho_destino.write_bytes(buffer_bytes)


def executar_produtor_consumidor(tamanho: int, threads: int, diretorio: Path, capacidade: int) -> None:
    garantir_arquivos_binarios(diretorio, tamanho, 64 * 1024)
    caminhos_arquivos = [p for p in diretorio.iterdir() if p.suffix == ".bin"][:tamanho]
    if not caminhos_arquivos:
        print('{"erro":"nenhum arquivo encontrado"}')
        return

    capacidade = max(1, capacidade)
    threads = max(2, threads)
    total_produtores = max(1, threads // 2)
    total_consumidores = threads - total_produtores
    if total_consumidores < 1:
        total_consumidores = 1
        total_produtores = max(1, threads - total_consumidores)
    fila_tarefas = queue.Queue(maxsize=capacidade)
    soma_hashes = 0
    trava_soma_hashes = threading.Lock()
    itens_processados = 0
    trava_itens_processados = threading.Lock()
    inicio_evento = threading.Event()

    def publicar_arquivos(lista_caminhos):
        inicio_evento.wait()
        for caminho in lista_caminhos:
            fila_tarefas.put(caminho)

    def processar_arquivo():
        nonlocal soma_hashes, itens_processados
        buffer_leitura = bytearray(1 << 20)
        inicio_evento.wait()
        while True:
            caminho = fila_tarefas.get()
            if caminho is None:
                fila_tarefas.task_done()
                break
            resumo = hashlib.sha256()
            with caminho.open("rb") as arquivo:
                while True:
                    quantidade_lida = arquivo.readinto(buffer_leitura)
                    if not quantidade_lida:
                        break
                    resumo.update(memoryview(buffer_leitura)[:quantidade_lida])
            valor_hash = int.from_bytes(resumo.digest()[:8], "little")
            with trava_soma_hashes:
                soma_hashes = (soma_hashes + valor_hash) & ((1 << 64) - 1)
            with trava_itens_processados:
                itens_processados += 1
            fila_tarefas.task_done()

    arquivos_por_produtor = (len(caminhos_arquivos) + total_produtores - 1) // total_produtores
    threads_produtoras = []
    for indice in range(total_produtores):
        inicio = indice * arquivos_por_produtor
        fim = min(inicio + arquivos_por_produtor, len(caminhos_arquivos))
        if inicio >= fim:
            break
        thread = threading.Thread(target=publicar_arquivos, args=(caminhos_arquivos[inicio:fim],))
        thread.start()
        threads_produtoras.append(thread)

    threads_consumidoras = [threading.Thread(target=processar_arquivo) for _ in range(total_consumidores)]
    for thread in threads_consumidoras:
        thread.start()

    inicio_ns, inicio_cpu = common.capturar_amostra_recursos()
    inicio_evento.set()
    for thread in threads_produtoras:
        thread.join()
    for _ in range(total_consumidores):
        fila_tarefas.put(None)
    fila_tarefas.join()
    for thread in threads_consumidoras:
        thread.join()

    print(common.montar_metricas(
        "pc",
        tamanho,
        threads,
        inicio_ns,
        inicio_cpu,
        itens_processados=itens_processados,
    ))


def main() -> None:
    tamanho_padrao = common.ler_inteiro_env("BENCH_SIZE", 1000)
    threads_padrao = max(1, common.ler_inteiro_env("BENCH_THREADS", os.cpu_count() or 4))
    buffer_padrao = common.ler_inteiro_env("BENCH_BUFFER", 256)
    diretorio_env = os.getenv("BENCH_DIR")
    diretorio_padrao = Path(diretorio_env) if diretorio_env else common.DEFAULT_DATA_DIR

    parser = argparse.ArgumentParser(description="Produtor-Consumidor em Python")
    parser.add_argument("--size", type=int, default=tamanho_padrao, help="quantidade de arquivos a processar")
    parser.add_argument("--threads", type=int, default=threads_padrao, help="numero de threads")
    parser.add_argument("--dir", type=Path, default=diretorio_padrao, help="diretorio de arquivos binarios")
    parser.add_argument("--buffer", type=int, default=buffer_padrao, help="capacidade do buffer compartilhado")
    args = parser.parse_args()

    executar_produtor_consumidor(max(1, args.size), max(2, args.threads), args.dir, max(1, args.buffer))


if __name__ == "__main__":
    main()
