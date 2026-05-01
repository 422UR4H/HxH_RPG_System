# Autenticação & Sessões — Guia de Desenvolvimento

> Documentação técnica do fluxo de autenticação, gerenciamento de sessões e middleware do sistema de RPG HxH.

---

## 1. Register — Caso de Uso

O registro de usuário segue uma **ordem estrita de validação** antes de qualquer acesso ao banco de dados. As validações falham rapidamente (fail-fast), retornando o primeiro erro encontrado.

### Ordem de validação

```
1. Nick vazio?         → ErrMissingNick
2. Nick length [3,20]? → ErrInvalidNickLength
3. Email vazio?        → ErrMissingEmail
4. Email length [12,64]? → ErrInvalidEmailLength
5. Password vazio?     → ErrMissingPassword
6. ConfirmPass vazio?  → ErrMissingConfirmPass
7. Password length ≥ 8? → ErrPasswordMinLenght
8. Password length ≤ 64? → ErrPasswordMaxLenght
9. Password == ConfirmPass? → ErrMismatchPassword
```

> **Nota:** `ErrPasswordMinLenght` e `ErrPasswordMaxLenght` preservam o typo original no código-fonte ("Lenght" em vez de "Length").

### Restrições de comprimento

| Campo | Mínimo | Máximo |
|-------|--------|--------|
| Nick | 3 | 20 |
| Email | 12 | 64 |
| Password | 8 | 64 |

### Verificações de unicidade (banco de dados)

Após validação local, duas queries separadas verificam unicidade:

1. `ExistsUserWithNick(nick)` → `ErrNickAlreadyExists`
2. `ExistsUserWithEmail(email)` → `ErrEmailAlreadyExists`

> **TODO preservado do código-fonte:** `// TODO: improve validation unifying these 2 or 3 db calls` — as verificações de unicidade são feitas em chamadas separadas ao banco. Idealmente seriam unificadas em uma única query para reduzir round-trips.

### Criação do usuário

Se todas as validações passam, o use case cria o `User` com:

- **UUID:** gerado via `uuid.New()` (Google UUID v4)
- **CreatedAt / UpdatedAt:** `time.Now()` no momento do registro
- **Persistência:** `repo.CreateUser()` recebe a entidade completa

O hashing de senha **não acontece no use case** — deve ocorrer na camada de repositório/gateway antes da persistência.

---

## 2. Login — Caso de Uso

O login autentica o usuário e estabelece uma sessão dual (memória + banco).

### Fluxo completo

```
1. Validar campos (email, password) — mesmas regras de comprimento do registro
2. Buscar usuário por email: repo.GetUserByEmail(email)
   └─ Se ErrEmailNotFound → retorna ErrUnauthorized (não revela se email existe)
3. Comparar senha: bcrypt.CompareHashAndPassword(hash, plaintext)
   └─ Se erro → retorna ErrUnauthorized (mesmo erro para email/senha incorretos)
4. Gerar JWT: auth.GenerateToken(userUUID)
5. Armazenar sessão em memória: sessions.Store(userUUID, token)
6. Persistir sessão no banco: sessionRepo.CreateSession(userUUID, token)
   └─ Se falha → log de erro via fmt.Println (não bloqueia o login!)
7. Retornar LoginOutput{Token, User}
```

### Decisões de design

- **Erro genérico (`ErrUnauthorized`):** tanto email inexistente quanto senha incorreta retornam o mesmo erro, prevenindo enumeração de usuários.
- **Sessão dual:** `sync.Map` (cache em memória) + PostgreSQL (persistência durável). A sessão é criada em ambos, mas a falha no banco **não impede o login** — apenas emite um log.
- **`sync.Map` como cache:** compartilhada entre `LoginUC` e o middleware via injeção de dependência. Chave: `userUUID`, valor: `token`.

### Dependências

O `LoginUC` recebe três dependências:

| Dependência | Tipo | Responsabilidade |
|-------------|------|------------------|
| `sessions` | `*sync.Map` | Cache de sessões em memória |
| `repo` | `IRepository` | Repositório de usuários (busca por email) |
| `sessionRepo` | `session.IRepository` | Repositório de sessões (persistência) |

---

## 3. Middleware — Autenticação por Request

O middleware `AuthMiddlewareProvider` retorna uma closure compatível com `huma.Middleware` que protege rotas autenticadas.

### Pipeline de verificação

```
Request com header "Authorization: Bearer <token>"
│
├─ 1. Extrair token do header Authorization
│     └─ Sem header ou formato inválido → 401 Unauthorized
│
├─ 2. Validar JWT: jwtAuth.ValidateToken(tokenStr)
│     └─ Token expirado/inválido → 401 Unauthorized
│     └─ Extrair claims (UserID)
│
├─ 3. Verificar sessão (cascata de 3 níveis):
│     │
│     ├─ 3a. Cache em memória: sessions.Load(claims.UserID)
│     │     └─ Hit + token confere → ✅ sessão válida
│     │
│     ├─ 3b. Fallback DB: sessionRepo.GetSessionTokenByUserUUID(userID)
│     │     └─ Hit + token confere → ✅ sessão válida
│     │
│     └─ 3c. Re-validação DB: sessionRepo.ValidateSession(userID, token)
│           └─ Válido → ✅ sessão válida
│           └─ Inválido → 401 Unauthorized
│
└─ 4. Injetar UserID no contexto: huma.WithValue(ctx, UserIDKey, claims.UserID)
      └─ next(ctx) — prosseguir para o handler
```

### Cache-miss e rehidratação

Quando o token não está no `sync.Map` (ex.: após restart do servidor), o middleware busca no PostgreSQL. Isso garante que sessões válidas sobrevivam a reinicializações sem forçar re-login, enquanto a `sync.Map` serve como cache rápido para requests subsequentes.

### Contexto injetado

Handlers autenticados acessam o ID do usuário via:

```go
userID := ctx.Context().Value(UserIDKey)
```

---

## 4. Persistência de Sessões

O sistema de sessões opera em duas camadas, priorizando performance sem sacrificar durabilidade.

### Arquitetura dual

```
┌─────────────────┐     ┌─────────────────────┐
│   sync.Map      │     │   PostgreSQL         │
│  (cache L1)     │     │  (store durável)     │
│                 │     │                      │
│  uuid → token   │     │  uuid, token,        │
│                 │     │  created_at, ...     │
└────────┬────────┘     └──────────┬───────────┘
         │                         │
         └────── leitura ──────────┘
              (cache-miss pattern)
```

### Fluxo de escrita (Login)

1. `sessions.Store(uuid, token)` — cache imediato
2. `sessionRepo.CreateSession(uuid, token)` — persistência assíncrona (falha tolerada)

### Fluxo de leitura (Middleware)

1. `sessions.Load(uuid)` — O(1), sem I/O
2. Se miss: `sessionRepo.GetSessionTokenByUserUUID(uuid)` — query PostgreSQL
3. Se inconsistente: `sessionRepo.ValidateSession(uuid, token)` — re-validação completa

### Trade-offs

| Aspecto | Decisão | Consequência |
|---------|---------|--------------|
| Cache não invalidado explicitamente | Tokens antigos podem permanecer no `sync.Map` | Logout precisa limpar ambos os stores |
| Falha de persistência tolerada no login | Sessão existe em memória mas não no banco | Restart do servidor invalida essa sessão |
| Sem TTL no `sync.Map` | Entradas crescem monotonicamente | Potencial leak de memória em produção com muitos usuários |

---

## 5. Mapeamento de Erros — Códigos HTTP

O handler na camada `app/api/auth/` traduz erros de domínio para códigos HTTP apropriados.

### POST /auth/register

| Código | Erros de domínio | Cenário |
|--------|-------------------|---------|
| **400** | `ErrMissingNick`, `ErrMissingEmail`, `ErrMissingPassword`, `ErrMissingConfirmPass` | Campo obrigatório ausente |
| **409** | `ErrNickAlreadyExists`, `ErrEmailAlreadyExists` | Conflito de unicidade |
| **422** | `ErrInvalidNickLength`, `ErrInvalidEmailLength`, `ErrPasswordMinLenght`, `ErrPasswordMaxLenght`, `ErrMismatchPassword` | Validação de formato/comprimento |
| **500** | Qualquer outro erro | Erro interno inesperado |

### POST /auth/login

| Código | Erros de domínio | Cenário |
|--------|-------------------|---------|
| **400** | `ErrMissingEmail`, `ErrMissingPassword` | Campo obrigatório ausente |
| **401** | `ErrUnauthorized` | Email não encontrado ou senha incorreta |
| **422** | `ErrInvalidEmailLength`, `ErrPasswordMinLenght`, `ErrPasswordMaxLenght` | Validação de formato/comprimento |
| **500** | Qualquer outro erro | Erro interno inesperado |

### Princípio de segurança

O login **nunca diferencia** entre "email não encontrado" e "senha incorreta" — ambos retornam `401 ErrUnauthorized`. Isso previne ataques de enumeração de contas.

---

## Referências de Código

| Conceito | Arquivo | Pacote |
|----------|---------|--------|
| RegisterUC (caso de uso) | `internal/domain/auth/register.go` | `auth` |
| LoginUC (caso de uso) | `internal/domain/auth/login.go` | `auth` |
| Auth IRepository | `internal/domain/auth/i_repository.go` | `auth` |
| Auth errors | `internal/domain/auth/error.go` | `auth` |
| User entity | `internal/domain/entity/user/` | `user` |
| User errors | `internal/domain/entity/user/error.go` | `user` |
| Session IRepository | `internal/domain/session/` | `session` |
| JWT generation/validation | `pkg/auth/` | `auth` |
| Auth handler (HTTP) | `internal/app/api/auth/handler.go` | `auth` |
| Auth middleware | `internal/app/api/auth/middleware.go` | `auth` |
| PG user repository | `internal/gateway/pg/user/` | `pgUser` |
