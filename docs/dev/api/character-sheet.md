# Character Sheet API

## POST /character-sheets — Criar ficha

**Auth:** JWT obrigatório

### Request

```json
{
  "campaign_uuid": "uuid | null",
  "profile": {
    "nickname": "Gon",
    "fullname": "Gon Freecss",
    "alignment": "Chaotic-Good",
    "description": "Descrição longa opcional",
    "brief_description": "Máx 255 chars",
    "birthday": "0000-05-15T00:00:00Z",
    "age": 12
  },
  "character_class": "Hunter",
  "skills_exps": {},
  "proficiencies_exps": {},
  "categories": { "Reinforcement": true },
  "initial_hex_value": null
}
```

**Sobre `birthday`:**
- Obrigatório. Formato RFC3339.
- O front-end envia com ano `0000` (ex: `"0000-05-15T00:00:00Z"`).
- O back-end sobrescreve o ano para `0` internamente; o campo real é dia e mês.
- O ano de nascimento é calculado e preenchido na consolidação da ficha (ver `POST /submissions/:sheet_uuid/accept`).

### Respostas

| Status | Situação |
|--------|----------|
| 201 | Ficha criada com sucesso |
| 400 | UUID da campanha inválido ou enum inválido |
| 409 | Nickname já existe |
| 404 | Classe de personagem não encontrada |
| 403 | Limite de fichas atingido |
| 422 | Perfil inválido (birthday ausente, nickname/fullname fora do tamanho, etc.) |

---

## POST /submissions/:sheet_uuid/accept — Consolidar ficha

**Auth:** JWT (master da campanha)

Calcula e persiste o ano de nascimento:

```
ref = campaign.story_current_at ?? campaign.story_start_at
birth_year = ref.year - age
se (birthday.month, birthday.day) > (ref.month, ref.day): birth_year -= 1
```

### Respostas

| Status | Situação |
|--------|----------|
| 200 | Ficha consolidada; birthday atualizado com ano correto |
| 404 | Submissão ou campanha não encontrada |
| 403 | Usuário não é o mestre da campanha |

---

## POST /upload/presigned-url

**Auth:** Bearer JWT obrigatório

**Request:**
```json
{
  "file_type": "avatar",
  "sheet_uuid": "uuid-v4"
}
```

**Response 200:**
```json
{
  "upload_url": "https://...r2.cloudflarestorage.com/...",
  "public_url": "https://pub.r2.dev/avatar/uuid.webp"
}
```

**Erros:**
- `400` - sheet_uuid inválido
- `422` - file_type inválido
- `401` - Unauthorized
- `500` - Internal Server Error

---

## GET /charactersheets/{uuid}

**Auth:** Bearer JWT obrigatório

### Parâmetros de Query

- `include` (opcional): Se `include=submission`, adiciona campo `submission` à resposta

### Response 200

```json
{
  "uuid": "...",
  "user_uuid": "...",
  "nickname": "Gon",
  "fullname": "Gon Freecss",
  "alignment": "Chaotic-Good",
  "description": "Descrição longa opcional",
  "brief_description": "Máx 255 chars",
  "birthday": "0000-05-15T00:00:00Z",
  "age": 12,
  "character_class": "Hunter",
  "skills_exps": {},
  "proficiencies_exps": {},
  "categories": { "Reinforcement": true },
  "avatar_url": "https://...",
  "cover_url": "https://..."
}
```

Quando `include=submission` é enviado:

```json
{
  "...": "campos anteriores",
  "submission": {
    "campaign_uuid": "...",
    "created_at": "2026-05-17T12:00:00Z"
  }
}
```

ou `"submission": null` se nenhuma submissão está pendente.

### Erros

| Status | Situação |
|--------|----------|
| 200 | Ficha retornada (com ou sem submission) |
| 400 | UUID inválido |
| 403 | Acesso negado (não é o dono) |
| 404 | Ficha não encontrada |
| 500 | Internal Server Error |

---

## PATCH /charactersheets/{uuid}

**Auth:** Bearer JWT (apenas o dono da ficha)

**Pré-condição:** Ficha deve estar livre (sem campanha e sem submissão pendente). Caso contrário, retorna `422 ErrCharacterSheetNotFreeToManage`.

### Request

```json
{
  "character_class": "Hunter",
  "skills_exps": {},
  "proficiencies_exps": {},
  "attribute_points": 10,
  "profile": {
    "nickname": "Gon",
    "fullname": "Gon Freecss",
    "alignment": "Chaotic-Good",
    "description": "Descrição longa opcional",
    "brief_description": "Máx 255 chars",
    "birthday": "0000-05-15T00:00:00Z",
    "age": 12
  }
}
```

Todos os campos obrigatórios (mesmo formato que `POST /character-sheets`).

### Response 200

Retorna a ficha atualizada (mesmo formato que `GET /charactersheets/{uuid}`).

### Erros

| Status | Situação |
|--------|----------|
| 200 | Ficha atualizada |
| 400 | UUID ou dados inválidos |
| 403 | Acesso negado (não é o dono) |
| 404 | Ficha ou classe de personagem não encontrada |
| 422 | Ficha não está livre para editar (está em campanha ou com submissão pendente) |
| 500 | Internal Server Error |

---

## DELETE /charactersheets/{uuid}

**Auth:** Bearer JWT (apenas o dono da ficha)

**Pré-condição:** Ficha deve estar livre (sem campanha e sem submissão pendente). Caso contrário, retorna `422 ErrCharacterSheetNotFreeToManage`.

### Response 204

Ficha deletada com sucesso. Nenhum corpo na resposta.

### Erros

| Status | Situação |
|--------|----------|
| 204 | Ficha deletada |
| 400 | UUID inválido |
| 403 | Acesso negado (não é o dono) |
| 404 | Ficha não encontrada |
| 422 | Ficha não está livre para deletar (está em campanha ou com submissão pendente) |
| 500 | Internal Server Error |

---

## PATCH /charactersheets/{uuid}/profile

**Auth:** Bearer JWT (apenas o dono da ficha)

**Request:**
```json
{
  "avatar_url": "https://...",
  "cover_url": "https://...",
  "brief_description": "Máx 255 chars"
}
```

Todos opcionais. Enviar `null` para limpar `avatar_url` ou `cover_url`. Campo `brief_description` também opcional.

**Response:** 204 No Content

**Erros:**
- `400` - Bad Request
- `401` - Unauthorized
- `404` - ficha não encontrada ou não pertence ao usuário
- `500` - Internal Server Error

---

## Campos de imagem nos summaries

Os endpoints que retornam listas de fichas (`GET /charactersheets`,
`GET /campaigns/:id`, enrollments e participants de partida) incluem
`avatar_url` e `cover_url` opcionais em cada summary de ficha:

```json
{
  "uuid": "...",
  "nick_name": "Gon",
  "avatar_url": "https://pub.r2.dev/avatar/uuid.webp",
  "cover_url": "https://pub.r2.dev/cover/uuid.webp"
}
```

Ambos são `omitempty` — ausentes quando o personagem ainda não tem imagem.
