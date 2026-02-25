# Golang Policy Inference Case

Serviço de inferência de políticas declarativas representadas como DAG (Graphviz DOT).

## O que o sistema faz
Recebe uma policy em DOT e um payload de variáveis, executa a inferência de forma determinística e retorna o output enriquecido.

Endpoint principal:
- `POST /infer`

Request:
```json
{
  "policy_dot": "digraph { start [result=\"\"]; ok [result=\"approved=true\"]; no [result=\"approved=false\"]; start -> ok [cond=\"age>=18\"]; start -> no [cond=\"age<18\"]; }",
  "input": {"age": 20},
  "policy_id": "credit",
  "policy_version": "v1",
  "debug": true
}
```

Response:
```json
{
  "output": {"age": 20, "approved": true},
  "policy": {"id": "credit", "version": "v1", "hash": "..."},
  "trace": {
    "start_node": "start",
    "visited_path": ["start", "ok"],
    "terminated": "leaf"
  }
}
```

- `policy` é metadata de versionamento.
- `trace` só aparece com `debug=true`.

## Semântica de execução
- nó inicial: `start`
- aplica `result` do nó atual
- avalia arestas em ordem
- segue a primeira condição verdadeira
- repete até folha ou ausência de transição válida

## Decisões arquiteturais

### 1. Separação por camadas
- `internal/policy`: domínio (compiler, engine, eval, cache, trace, observability)
- `internal/app`: usecase/orquestração
- `internal/transport/*`: adapter de protocolo (HTTP/Lambda)
- `cmd/*`: bootstrap

Motivo: reduzir acoplamento e facilitar teste/evolução.

### 2. Modelo de domínio próprio + parser DOT consolidado
- parsing DOT com `gographviz`
- transformação para modelo próprio (`Policy`, `Node`, `Edge`)

Motivo: manter robustez de parser e controlar semântica no domínio.

### 3. Execução determinística
Engine sempre segue a primeira aresta válida na ordem.

Motivo: Requisito e previsibilidade de resultado.

### 4. Condição pré-compilada
Condições são compiladas no compile (`eval.Compile`) e reutilizadas no runtime.

Motivo: reduzir custo no hot path.

### 5. Validação de DAG no compile
Detecta ciclos via DFS e aplica failfast

Motivo: evitar loops em runtime e retornar erro semântico claro.

### 6. Versionamento de policy
`policy_id` + `policy_version` opcionais (sempre em par), com `hash` da policy no retorno.

Motivo: auditoria, rastreabilidade e reprodutibilidade.

### 7. Trace opcional de execução
`debug=true` habilita retorno detalhado do caminho (`visited_path`, `steps`, `terminated`).

Motivo: debug.

### 8. Observability de latência por nó
Observer de latência por node, com implementação async (`AsyncNodeLatencyObserver`).

Motivo: visibilidade sem bloquear o caminho crítico.

### 9. Cache concorrente com dedupe por chave
`GetOrCompute` usa inflight por chave, evita contenção global e deduplica compilação concorrente.

Motivo: throughput melhor sob carga concorrente.

## Estrutura de pastas
```text
cmd/
  http/
  lambda/
  loadtest/
internal/
  app/
  config/
  integration/
  policy/
  transport/
```

## Configuração
Arquivo `.env` (local):
```env
HTTP_ADDR=:8080
POLICY_CACHE_MAX_ITEMS=1024
POLICY_MAX_STEPS=10000
POLICY_OBS_BUFFER=4096
```

Variáveis usadas:
- `HTTP_ADDR`: endereço do servidor HTTP
- `POLICY_CACHE_MAX_ITEMS`: tamanho máximo do cache de policy compilada
- `POLICY_MAX_STEPS`: limite de passos por execução
- `POLICY_OBS_BUFFER`: buffer do observer assíncrono

## Como rodar

### HTTP local
```bash
go run ./cmd/http
```

### Docker
```bash
make docker-run
```

### SAM Local (Lambda)
Se necessário, exporte socket do Docker:
```bash
export DOCKER_HOST=unix:///Users/$USER/.docker/run/docker.sock
```

Rodar SAM padrão:
```bash
make sam
```

Rodar SAM com containers sem cold start:
```bash
make sam-warm
```

## Qualidade e testes

### Testes
```bash
make test
make test-race
make test-cover
```

### Integração
Pacote `internal/integration` cobre:
- fluxo compile+engine
- e2e HTTP
- caminhos approved/review/rejected
- erros de entrada
- debug trace
- versionamento válido/inválido
- ciclo no DOT
- concorrência básica

## Performance

### Benchmark
```bash
make bench
```

### Profiling
```bash
make bench-profile
```

### Load test
Suba o servidor e rode:
```bash
make load-test
```

Critério de aceite no load test:
- `achieved_rps >= 98%` do alvo
- `p90 < 30ms`
- `errors = 0`
- `non_2xx = 0`

## Trade-offs e limitações
- cache é in-memory (por processo), sem persistência distribuída
- observer assíncrono pode dropar eventos sob saturação (preferência por não bloquear)
- evaluator usa `expr` com validação restritiva (não há parser próprio)

## Comandos úteis
```bash
make test
make bench
make load-test
make docker-run
make sam
make sam-warm
```
