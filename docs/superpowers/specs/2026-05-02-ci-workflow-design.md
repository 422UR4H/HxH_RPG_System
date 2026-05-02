# CI Workflow Design

## Problema

Cada execução manual de `go vet` + `go test` no chat consome tokens
significativos. Automatizar via GitHub Actions reduz esse custo a quase zero
(só verificar status ✅/❌) e garante que todo push/PR seja validado.

## Decisões

| Decisão | Escolha | Justificativa |
|---------|---------|---------------|
| Estrutura | Job único sequencial | Mais rápido em wall-clock; o desenvolvedor sempre abre os logs |
| Trigger | `push` + `pull_request` | Cobertura máxima |
| Verificações | vet, golangci-lint, build, unit tests, integration tests | Cobertura completa |
| Banco de dados | PostgreSQL service container | Efêmero, gratuito, sem infra externa |
| Custo | Gratuito | Repo público → minutos ilimitados |

## Arquivo

`.github/workflows/ci.yml`

## Workflow

**Nome:** `CI`

**Triggers:**
- `push` (todas as branches)
- `pull_request` (todas as branches)

**Job: `ci`**

Roda em `ubuntu-latest` com PostgreSQL service container.

### Service Container: PostgreSQL

- Imagem: `postgres:17`
- Credenciais: `user=test_user`, `password=test_pass`, `db=test_db`
- Health check: aguarda o banco estar pronto antes dos steps
- Porta: 5432 mapeada para o host

### Steps (sequenciais)

1. **Checkout** — `actions/checkout@v4`
2. **Setup Go** — `actions/setup-go@v5` com Go `1.25.x` (cache embutido)
3. **Download dependencies** — `go mod download`
4. **go vet** — `go vet ./...`
5. **golangci-lint** — `golangci/golangci-lint-action@v7`
6. **go build** — `go build ./...`
7. **Unit tests** — `go test ./...`
8. **Integration tests** — `go test -tags=integration -p 1 ./internal/gateway/pg/...`

### Variáveis de ambiente

```yaml
env:
  TEST_DATABASE_URL: postgres://test_user:test_pass@localhost:5432/test_db?sslmode=disable
```

Os testes de integração usam `TEST_DATABASE_URL` (via `pgtest.GetDatabaseURL()`)
para conectar ao banco. As migrações são executadas automaticamente pelo helper
`SetupTestDB` — não é necessário um step separado para rodar goose.

## Pontos de atenção

- **Go 1.25.x**: versão minor com patch flexível para receber bugfixes
  automáticos
- **`-p 1`** nos testes de integração: serializa execução para evitar conflitos
  no banco compartilhado
- **Migrações automáticas**: `pgtest.SetupTestDB` já executa goose migrations
  programaticamente — nenhum step extra necessário
- **golangci-lint**: usa configuração padrão inicialmente; pode ser customizado
  com `.golangci.yml` no futuro
- **Cache**: `actions/setup-go@v5` já faz cache automático dos módulos Go

## Fora de escopo

- Deploy automático
- Coverage reports / badges
- Configuração customizada do golangci-lint
- Matrix de múltiplas versões de Go
- Notificações (Slack, email, etc.)
