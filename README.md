# TCC Benchmarks — Concorrência e Paralelismo

Este pacote entrega **templates prontos** com 3 problemas clássicos por categoria:

## Concorrência (Go, Java, Python)
1. `pc` — Produtor–Consumidor (buffer limitado, SHA-256 em arquivos)
2. `rw` — Leitores–Escritores com `RWMutex`/locks
3. `phil` — Jantar dos Filósofos (deadlock-free)

## Paralelismo (Go, C++/OpenMP)
1. `matmul` — multiplicação de matrizes (densa, CPU-bound)
2. `stencil` — stencil 2D (5-point), memory-bound
3. `mcpi` — Monte Carlo para π (embarrassingly parallel)

### Layout dos arquivos de execução
Cada problema agora possui um ponto de entrada exclusivo por linguagem:
- Go (concorrência): `concorrencia/go/{pc, rw, phil}/main.go`
- Python: `concorrencia/python/{pc.py, rw.py, phil.py}`
- Java: `concorrencia/java/{ProdutorConsumidor.java, LeitoresEscritores.java, JantarFilosofos.java}`
- Go (paralelismo): `paralelismo/go/{matmul, stencil, mcpi}/main.go`
- C++/OpenMP: `paralelismo/cpp_omp/{matmul.cpp, stencil.cpp, mcpi.cpp}` + `common.hpp`

### Variáveis de ambiente suportadas
Os executáveis continuam aceitando variáveis de ambiente (valores padrão entre parênteses):
- `BENCH_SIZE` — tamanho/amostras da simulação (1000 para concorrência, 1024 para paralelismo)
- `BENCH_THREADS` — número de threads/goroutines (`num CPUs`)
- `BENCH_BUFFER` — capacidade do buffer no `pc` (256)
- `BENCH_DIR` — diretório usado pelo `pc` (criado automaticamente)
- `BENCH_READ_PCT` — percentual de leituras no `rw` (10)
- `BENCH_ITERS` — iterações do `stencil` (100)

### Uso via linha de comando
Formato geral (os parâmetros opcionais variam por problema):
```
<programa> --size <N> --threads <p> [--dir <path>] [--buffer <B>] [--read_pct <X>] [--iters <I>]
```
Saída (JSON, uma linha):
```
{"nome_problema":"...","tamanho_instancia":N,"quantidade_threads":p,"tempo_decorrido_ms":...,"tempo_cpu_ms":...,"percentual_uso_cpu":...,"percentual_uso_cpu_por_nucleo":...,"memoria_rss_mb":...,"itens_processados":...,"operacoes_realizadas":...,"iteracoes_realizadas":...}
```

- `nome_problema`: identificador do benchmark executado.
- `tamanho_instancia`: escala da carga processada (arquivos, chaves, dimensão, etc.).
- `quantidade_threads`: número de threads ou gorrotinas utilizados.
- `tempo_decorrido_ms`: tempo de parede total (ms) da execução.
- `tempo_cpu_ms`: tempo de CPU acumulado (usuário + kernel) em ms.
- `percentual_uso_cpu`: razão entre tempo de CPU e tempo de parede.
- `percentual_uso_cpu_por_nucleo`: percentual de utilização após normalizar pelo número de núcleos lógicos disponíveis.
- `memoria_rss_mb`: pico de memória residente observado (VmHWM) em MB.
- `itens_processados`: quantidade de unidades consumidas no benchmark Produtor-Consumidor (0 nos demais).
- `operacoes_realizadas`: total de operações concluídas no benchmark Leitores-Escritores (0 nos demais).
- `iteracoes_realizadas`: quantidade de iterações completadas no Jantar dos Filósofos (0 nos demais).

Exemplos por linguagem:

# concorrencia
## golang

- `go run ./concorrencia/go/pc --size 2000 --threads 1 --buffer 512 --dir ./concorrencia/dados_pc`
- `go run ./concorrencia/go/rw --size 2000 --threads 1 --read_pct 70`
- `go run ./concorrencia/go/phil --size 2000 --threads 1`

## python

- `python3 concorrencia/python/pc.py --size 2000 --threads 1 --buffer 512 --dir concorrencia/python/data_py`
- `python3 concorrencia/python/rw.py --size 2000 --threads 1 --read_pct 70`
- `python3 concorrencia/python/phil.py --size 2000 --threads 1`

## java

- `javac concorrencia/java/*.java`
- `java -cp concorrencia/java ProdutorConsumidor --size 2000 --threads 1 --buffer 512 --dir concorrencia/java/data`
- `java -cp concorrencia/java LeitoresEscritores --size 2000 --threads 1 --read_pct 70`
- `java -cp concorrencia/java JantarFilosofos --size 2000 --threads 1`

# paralelismo
## golang

- `taskset -c 0-11 go run ./paralelismo/go/matmul --size 1024 --threads 1`
- `taskset -c 0-11 go run ./paralelismo/go/stencil --size 2048 --iters 100 --threads 1`
- `taskset -c 0-11 go run ./paralelismo/go/mcpi --size 20000000 --threads 1`

## c++

- `cd paralelismo/cpp_omp`
- `g++ -O3 -fopenmp -march=native -std=c++17 -DNDEBUG -o matmul matmul.cpp`
- `g++ -O3 -fopenmp -march=native -std=c++17 -DNDEBUG -o stencil stencil.cpp`
- `g++ -O3 -fopenmp -march=native -std=c++17 -DNDEBUG -o mcpi mcpi.cpp`

- `OMP_NUM_THREADS=1  OMP_PROC_BIND=TRUE OMP_PLACES=cores taskset -c 0-11 ./matmul  --size 1024   --threads 1`
- `OMP_NUM_THREADS=1  OMP_PROC_BIND=TRUE OMP_PLACES=cores taskset -c 0-11 ./stencil --size 2048   --iters 100 --threads 1`
- `OMP_NUM_THREADS=1  OMP_PROC_BIND=TRUE OMP_PLACES=cores taskset -c 0-11 ./mcpi    --size 20000000 --threads 1`
