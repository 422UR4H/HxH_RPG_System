# Deploy — Render + Neon + R2

Hospedagem gratuita do projeto. O repositório é **público**, então nenhum segredo vai para o código:
tudo entra por variáveis de ambiente configuradas na plataforma.

> TODO: preencher as URLs públicas (front, API, game) quando o deploy estiver no ar.

## Topologia

| Peça | Onde | Observação |
|---|---|---|
| Front (Vite/React) | Render — static site | build com `VITE_*` |
| `hxh-rpg-api` | Render — web service (Docker) | `CMD` padrão: `scripts/start-api.sh` |
| `hxh-rpg-game` | Render — web service (Docker) | comando `./game` (sobrescreve o `CMD`) |
| Postgres | Neon free, AWS `us-east-1` | TLS obrigatório; mesma área do Render (Virginia) para manter a latência baixa |
| Imagens | Cloudflare R2 | inalterado |

As duas imagens de serviço são a **mesma** (`Dockerfile` na raiz): contém `api`, `game`, o CLI `goose`
e `migrations/`.

## Variáveis de ambiente

| Variável | Serviço | Valor / nota |
|---|---|---|
| `PG_DB_HOST`, `PG_DB_PORT`, `PG_DB_NAME`, `PG_DB_USER`, `PG_DB_PASS` | API + game | credenciais do Neon; host **direto** (sem `-pooler`): o goose e o pgx contam com sessão, que o pooler em modo transação não garante |
| `PG_DB_SSLMODE` | API + game | `require` (também é o default de `pkg/config.go` e do start script) |
| `PG_DB_POOL_MIN_SIZE` | API + game | `0` — pool ocioso não mantém o Neon acordado |
| `JWT_SECRET` | API + game | **idêntico nos dois**: a API emite o token, o game valida |
| `SERVER_ADDR` | API | `0.0.0.0:10000` |
| `ALLOWED_ORIGINS` | API | origem do front |
| `R2_*` | API | credenciais do bucket |
| `GAME_SERVER_ADDR` | game | `0.0.0.0:10000` |
| `VITE_API_URL` | build do front | URL pública da API |
| `VITE_WS_URL` | build do front | URL pública do game, com `wss://` |

Dentro de um container o bind precisa ser `0.0.0.0`. O default do código (`localhost:5000`) não é
alcançável de fora, por isso `SERVER_ADDR` e `GAME_SERVER_ADDR` são definidos na plataforma — o
default do código não muda.

## Como o deploy acontece

1. Push/merge em `main` → Render reconstrói a imagem.
2. `scripts/start-api.sh` monta o DSN (formato `key=value`, evita problemas de URL-encoding na senha) a
   partir das mesmas `PG_DB_*`, roda `goose -dir ./migrations postgres "<dsn>" up` e só então faz
   `exec ./api`.
3. O game server **nunca** roda migrations (evita dois processos disputando o schema).

## Comportamento do free tier

- API e game dormem após 15 min sem tráfego; acordar leva cerca de 1 min.
- As salas em memória do game server se perdem quando ele dorme ou reinicia; elas são reidratadas a
  partir do banco.
- `JWT_SECRET` ausente = chave de desenvolvimento **insegura** (`pkg/auth/jwt.go`), com um aviso no log
  de ambos os binários. Nunca deixe assim em produção: qualquer um forjaria tokens.
