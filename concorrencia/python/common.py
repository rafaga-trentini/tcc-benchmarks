from __future__ import annotations

import json
import os
import resource
import time
from pathlib import Path
from typing import Tuple

BASE_DIR = Path(__file__).resolve().parent
DEFAULT_DATA_DIR = BASE_DIR.parent / "dados_pc"


def obter_rss_mb() -> float:
    rss_kb = resource.getrusage(resource.RUSAGE_SELF).ru_maxrss
    if os.name == "posix" and os.uname().sysname == "Darwin":
        return rss_kb / (1024.0 * 1024.0)
    return rss_kb / 1024.0


def capturar_amostra_recursos() -> Tuple[int, float]:
    uso_processador = resource.getrusage(resource.RUSAGE_SELF)
    return time.perf_counter_ns(), uso_processador.ru_utime + uso_processador.ru_stime


def montar_metricas(
    problema: str,
    tamanho: int,
    threads: int,
    inicio_ns: int,
    inicio_cpu: float,
    itens_processados: int = 0,
    operacoes_realizadas: int = 0,
    iteracoes_realizadas: int = 0,
) -> str:
    tempo_parede_ms = (time.perf_counter_ns() - inicio_ns) / 1_000_000.0
    uso_atual = resource.getrusage(resource.RUSAGE_SELF)
    tempo_cpu_ms = ((uso_atual.ru_utime + uso_atual.ru_stime) - inicio_cpu) * 1000.0
    percentual_cpu = (tempo_cpu_ms / tempo_parede_ms * 100.0) if tempo_parede_ms > 0 else 0.0
    nucleos = max(1, os.cpu_count() or 1)
    metricas = {
        "nome_problema": problema,
        "tamanho_instancia": tamanho,
        "quantidade_threads": threads,
        "tempo_decorrido_ms": tempo_parede_ms,
        "tempo_cpu_ms": tempo_cpu_ms,
        "percentual_uso_cpu": percentual_cpu,
        "percentual_uso_cpu_por_nucleo": percentual_cpu / nucleos,
        "memoria_rss_mb": obter_rss_mb(),
    }
    metricas["itens_processados"] = itens_processados
    metricas["operacoes_realizadas"] = operacoes_realizadas
    metricas["iteracoes_realizadas"] = iteracoes_realizadas
    return json.dumps(metricas)


def ler_inteiro_env(nome: str, padrao: int) -> int:
    valor = os.getenv(nome)
    if not valor:
        return padrao
    try:
        return int(valor)
    except ValueError:
        return padrao
