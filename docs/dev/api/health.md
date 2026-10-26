# Health API

Dois endpoints de saúde, ambos **sem autenticação**.

| Endpoint | Pergunta que responde | Toca o banco? |
|---|---|---|
| `GET /liveness` | O processo está de pé? | Não |
| `GET /readiness` | Consegue atender requisições (dependências ok)? | Sim (`Ping`) |

## GET /liveness

**Auth:** nenhuma. Sem body de request.

| Status | Situação | Body |
|---|---|---|
| 200 | Processo respondendo | `{"message": "Service is healthy"}` |

## GET /readiness

**Auth:** nenhuma. Sem body de request.

Faz `Ping` no Postgres com prazo de **2 segundos** sobre o contexto da
requisição. O erro do banco não vaza no corpo.

| Status | Situação | Body |
|---|---|---|
| 200 | Ping ok | `{"message": "Service is ready"}` |
| 503 | Ping falhou ou estourou os 2 s | `{"message": "Database unavailable"}` |

### Exemplos

```bash
curl -s -w ' %{http_code}' localhost:5000/readiness
# "{\"message\": \"Service is ready\"}" 200
```

> Nota: o corpo hoje é uma *string JSON* contendo o texto acima (o handler
> devolve `Body` como string), não um objeto. O front deve usar só o status.

### Uso

- O front chama `/readiness` em segundo plano para acordar, com uma única
  requisição, a API (Render free tier) e o banco (Neon). Um 503 durante o
  despertar é esperado; o front pode tentar de novo.
- Health check de plataforma (Render etc.) deve usar `/liveness`: se usasse
  `/readiness`, uma oscilação do banco faria a plataforma reiniciar uma API
  saudável.
