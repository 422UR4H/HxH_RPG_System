# Match API

## POST /matches — Criar partida

**Auth:** JWT (master da campanha)

### Request

```json
{
  "campaign_uuid": "uuid-v4",
  "title": "Greed Island - Session 1",
  "brief_initial_description": "Os heróis chegam à ilha (máx 64 chars)",
  "description": "Descrição completa apenas para o mestre",
  "is_public": true,
  "game_scheduled_at": "2026-06-15T19:30:00Z",
  "story_start_at": "2026-06-15"
}
```

| Campo | Regra |
|---|---|
| `title` | obrigatório, 5–32 chars |
| `brief_initial_description` | ≤ 64 chars |
| `game_scheduled_at` | ISO 8601, futuro, ≤ +1 ano |
| `story_start_at` | YYYY-MM-DD, dentro da janela da campanha |
| `is_public` | default `true` |

### Respostas

| Status | Situação |
|---|---|
| 201 | Partida criada, retorna `{ "match": MatchResponse }` |
| 400 | Body malformado |
| 401 | Sem JWT |
| 403 | Usuário não é o mestre da campanha |
| 404 | Campanha não encontrada |
| 422 | Validação (título fora do tamanho, datas fora da janela, etc.) |
| 500 | Erro interno |

---

## GET /matches/{uuid} — Obter partida

**Auth:** JWT obrigatório

### Visibilidade

- **Mestre da partida:** sempre vê.
- **Partida pública:** qualquer usuário autenticado vê.
- **Partida privada:** apenas participantes (jogadores com personagem na campanha) veem; demais recebem 403.

### Response 200

```json
{
  "match": {
    "uuid": "...",
    "master_uuid": "...",
    "campaign_uuid": "...",
    "title": "Greed Island - Session 1",
    "brief_initial_description": "...",
    "brief_final_description": null,
    "description": "...",
    "is_public": true,
    "game_scheduled_at": "2026-06-15T19:30:00Z",
    "game_start_at": null,
    "story_start_at": "2026-06-15",
    "story_end_at": null,
    "created_at": "...",
    "updated_at": "..."
  }
}
```

### Erros

| Status | Situação |
|---|---|
| 200 | Partida retornada |
| 400 | UUID inválido |
| 403 | Partida privada e usuário não é mestre nem participante |
| 404 | Partida não encontrada |
| 500 | Erro interno |

---

## PATCH /matches/{uuid} — Editar partida

**Auth:** JWT (apenas o mestre)

**Pré-condição:** `game_start_at IS NULL && story_end_at IS NULL`. Após
`StartMatch` ou encerramento, qualquer PATCH retorna 422.

### Request — todos os campos opcionais

```json
{
  "title": "Novo título",
  "brief_initial_description": "Novo brief",
  "description": "Nova descrição",
  "is_public": false,
  "game_scheduled_at": "2026-07-20T19:30:00Z",
  "story_start_at": "2026-07-20"
}
```

| Campo | Regra (se enviado) |
|---|---|
| `title` | 5–32 chars |
| `brief_initial_description` | ≤ 64 chars |
| `game_scheduled_at` | ISO 8601, futuro, ≤ +1 ano |
| `story_start_at` | YYYY-MM-DD, dentro da janela da campanha |

`campaign_uuid` é imutável — não enviar.

Body vazio (`{}`) é no-op idempotente, retorna 200 com a partida atual.

### Response 200

Mesmo formato de `GET /matches/{uuid}` — partida com campos atualizados.

### Erros

| Status | Situação |
|---|---|
| 200 | Atualizado (ou no-op) |
| 400 | Body malformado |
| 401 | Sem JWT |
| 403 | Usuário não é o mestre |
| 404 | Partida ou campanha não encontrada |
| 422 | Validação (tamanho, data fora de janela) **ou** partida já iniciada/encerrada |
| 500 | Erro interno |

---

## DELETE /matches/{uuid} — Excluir partida

**Auth:** JWT (apenas o mestre da partida)

**Pré-condição:** `game_start_at IS NULL`. Se a partida já foi iniciada, retorna 422. Inscrições são removidas automaticamente via `ON DELETE CASCADE` na FK `enrollments.match_uuid`.

### Response

| Status | Situação |
|---|---|
| 204 | Partida e inscrições excluídas |
| 400 | UUID inválido |
| 401 | Sem JWT |
| 403 | Usuário não é o mestre da partida |
| 404 | Partida não encontrada |
| 422 | Partida já iniciada (`game_start_at IS NOT NULL`) |
| 500 | Erro interno |

---

## GET /matches — Listar partidas do mestre

**Auth:** JWT obrigatório

Retorna todas as partidas em que o usuário é mestre, ordenadas por `story_start_at` ASC.

### Response 200

```json
{
  "matches": [
    {
      "uuid": "...",
      "campaign_uuid": "...",
      "title": "...",
      "brief_initial_description": "...",
      "brief_final_description": null,
      "is_public": true,
      "game_scheduled_at": "...",
      "game_start_at": null,
      "story_start_at": "...",
      "story_end_at": null,
      "created_at": "...",
      "updated_at": "..."
    }
  ]
}
```

### Erros

| Status | Situação |
|---|---|
| 200 | Lista (pode ser vazia) |
| 400 | Token inválido |
| 401 | Sem JWT |
| 500 | Erro interno |

---

## GET /public/matches — Partidas públicas futuras

**Auth:** JWT obrigatório

Retorna partidas públicas com `game_scheduled_at > now()`, ordenadas
ASC, **excluindo** as do próprio mestre autenticado.

### Response 200

Mesmo formato de `GET /matches`.

### Erros

| Status | Situação |
|---|---|
| 200 | Lista (pode ser vazia) |
| 401 | Sem JWT |
| 500 | Erro interno |

---

## GET /matches/{uuid}/enrollments — Inscrições

**Auth:** JWT obrigatório

Visibilidade por linha:

- **Mestre da partida:** vê resumo privado de cada ficha inscrita.
- **Dono da ficha inscrita:** vê o resumo privado da própria; demais linhas, resumo base.
- **Outros:** apenas resumo base de todas as fichas.

### Response 200

```json
{
  "enrollments": [
    {
      "uuid": "...",
      "status": "pending | accepted | rejected",
      "created_at": "...",
      "character_sheet": {
        "uuid": "...",
        "nick_name": "Gon",
        "avatar_url": "...",
        "cover_url": "...",
        "private": { }
      },
      "player": { "uuid": "...", "nick": "..." }
    }
  ]
}
```

### Erros

| Status | Situação |
|---|---|
| 200 | Lista de enrollments |
| 403 | Acesso negado |
| 404 | Partida não encontrada |
| 500 | Erro interno |

---

## GET /matches/{uuid}/participants — Participantes

**Auth:** JWT obrigatório

Lista personagens que entraram na partida (snapshot a partir de `StartMatch`).

### Response 200

```json
{
  "participants": [
    {
      "uuid": "...",
      "joined_at": "...",
      "left_at": null,
      "character_sheet": {
        "uuid": "...",
        "nick_name": "Killua",
        "avatar_url": "...",
        "cover_url": "...",
        "private": { }
      }
    }
  ]
}
```

### Erros

| Status | Situação |
|---|---|
| 200 | Lista (pode ser vazia) |
| 403 | Acesso negado |
| 404 | Partida não encontrada |
| 500 | Erro interno |
