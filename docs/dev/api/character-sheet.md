# Character Sheet API

> Todos os corpos de request/response usam **camelCase**. Todas as respostas de
> objeto único (`create`, `get`, `update`) vêm envelopadas em
> `{ "characterSheet": {...} }`; a listagem vem envelopada em
> `{ "characterSheets": [...] }`. Rotas registradas em
> `internal/app/api/sheet/routes.go` e `internal/app/api/submission/routes.go`.

## POST /charactersheets — Criar ficha

**Auth:** JWT obrigatório

### Request

```json
{
  "campaignUuid": "uuid | null",
  "profile": {
    "nickname": "Gon",
    "fullname": "Gon Freecss",
    "alignment": "Chaotic-Good",
    "description": "Descrição longa opcional",
    "briefDescription": "Máx 255 chars",
    "birthday": "0000-05-15T00:00:00Z",
    "age": 12
  },
  "characterClass": "Hunter",
  "skillsExps": { "Vitality": 3 },
  "proficienciesExps": { "Sword": 2 },
  "attributePoints": { "Strength": 10 }
}
```

- `campaignUuid`: opcional.
  - `null`/ausente → a ficha é criada **livre**, pertencente ao usuário autenticado (`playerUuid` na resposta).
  - presente → a ficha é criada **diretamente vinculada a uma campanha**, pertencente ao mestre autenticado (`masterUuid` na resposta); o back-end valida que o usuário é de fato o mestre daquela campanha e que a campanha existe. `campaign.ErrNotCampaignOwner` e `campaign.ErrCampaignNotFound` são ambos construídos via `domain.NewValidationError` (`internal/application/campaign/error.go`) e o handler (`internal/app/api/sheet/create_character_sheet.go`) não tem `case` explícito para nenhum dos dois — caem no `case errors.Is(err, domain.ErrValidation)`, resultando em **`422`, não `403`** (ver tabela de respostas).
- `profile`: objeto `ProfileRequest` (ver validações abaixo). `avatarUrl`/`coverUrl` também podem ser enviados aqui (opcionais), embora normalmente sejam setados depois via `PATCH /charactersheets/{uuid}/profile`.
- `characterClass`: nome de uma classe válida (ex.: `Hunter`, `Swordsman`, `Ninja`...).
- `skillsExps`: `map[string]int`. Chave deve bater **exatamente** (case-sensitive) com um `SkillName` conhecido (ex.: `Vitality`, `Accuracy`, `Focus`). Valor é EXP a aplicar na skill na criação.
- `proficienciesExps`: `map[string]int`. Chave é um nome de arma (`WeaponName`, ex.: `Sword`, `Dagger`, `Bow`). **Quirk verificado no código:** o back-end força a primeira letra da chave para maiúscula antes de resolver o enum (`strings.ToUpper(k[:1]) + k[1:]`) — então `"sword"` e `"Sword"` funcionam igual, mas `skillsExps` **não** tem esse tratamento (precisa vir exatamente como o enum).
- `attributePoints`: opcional, `map[string]int`. Chave é um `AttributeName` (ex.: `Strength`, `Agility`, `Flame`). Valores `<= 0` são silenciosamente ignorados.
- **Campos que existiam no doc antigo e não existem mais no request:** `categories` (booleano) e `initialHexValue`. Estão comentados no código-fonte (`castRequest` em `create_character_sheet.go`, marcados `// TODO: move to consolidate (accept submission) flow`) — a definição da categoria de Nen (talento) acontece hoje no fluxo de consolidação da submissão, não na criação da ficha.

**Sobre `profile.birthday`:**
- Obrigatório (`Validate()` rejeita `time.Time` zero).
- Formato RFC3339.
- O front-end envia com ano `0000` (ex.: `"0000-05-15T00:00:00Z"`).
- O back-end sobrescreve o ano para `0` internamente (`time.Date(0, mês, dia, ...)`); o campo real relevante nesta etapa é dia e mês.
- O ano de nascimento definitivo é calculado e preenchido na consolidação da ficha (ver `POST /submissions/{sheet_uuid}/accept`).

**Outras validações de `profile` (`CharacterProfile.Validate()`):**
- `nickname`: 3–10 caracteres.
- `fullname`: 6–32 caracteres.
- `briefDescription`: máx. 255 caracteres.
- `age`: `>= 0`.
- `alignment`: opcional; se enviado, deve seguir o formato `"<Eixo1>-<Eixo2>"` com `Eixo1 ∈ {Lawful, Neutral, Chaotic}` e `Eixo2 ∈ {Good, Neutral, Evil}` (ex.: `"Chaotic-Good"`).

### Response 201

Envelope `{ "characterSheet": {...} }` — ver shape completo em [Shape de `characterSheet`](#shape-de-charactersheet) abaixo.

### Respostas

| Status | Situação |
|--------|----------|
| 201 | Ficha criada com sucesso |
| 400 | UUID da campanha inválido, enum inválido (classe/skill/proficiência/atributo) |
| 409 | Nickname já existe |
| 404 | Classe de personagem não encontrada |
| 403 | Limite de fichas atingido (jogador) |
| 422 | Perfil inválido (birthday ausente, nickname/fullname fora do tamanho, alignment mal formatado, etc.) / usuário não é o mestre da campanha informada (`campaign.ErrNotCampaignOwner`) / campanha não encontrada (`campaign.ErrCampaignNotFound`) — ambos caem no fallback `domain.ErrValidation` do handler |

---

## POST /submissions/charactersheets/submit — Submeter ficha

**Auth:** JWT (player dono da ficha)

**Request:**
```json
{
  "sheetUuid": "uuid-v4",
  "campaignUuid": "uuid-v4"
}
```

### Respostas

| Status | Situação |
|--------|----------|
| 201 | Ficha submetida com sucesso |
| 403 | Usuário não é dono da ficha / mestre não pode submeter a própria ficha |
| 404 | Ficha ou campanha não encontrada |
| 409 | Ficha já submetida / nick já existe nesta campanha (aceito ou pendente) |

---

## POST /submissions/{sheet_uuid}/accept — Consolidar ficha

**Auth:** JWT (mestre da campanha)

Valida unicidade de nick na campanha (aceitos + submissions pendentes) antes de consolidar. Sem corpo de request.

Calcula e persiste o ano de nascimento (`CalcBirthYear` em `internal/application/submission/accept_sheet_submission.go`):

```
ref = campaign.storyCurrentAt ?? campaign.storyStartAt
birthYear = ref.year - age
se (birthday.month, birthday.day) > (ref.month, ref.day) [comparação lexicográfica]: birthYear -= 1
```

### Respostas

| Status | Situação |
|--------|----------|
| 200 | Ficha consolidada; birthday atualizado com ano correto |
| 403 | Usuário não é o mestre da campanha |
| 404 | Submissão ou campanha não encontrada |
| 409 | Nick já existe nesta campanha (outro personagem aceito ou pendente) |

---

## POST /submissions/{sheet_uuid}/reject — Rejeitar submissão

**Auth:** JWT (mestre da campanha)

Sem corpo de request.

### Respostas

| Status | Situação |
|--------|----------|
| 204 | Submissão rejeitada, sem corpo na resposta |
| 403 | Usuário não é o mestre da campanha |
| 404 | Submissão ou campanha não encontrada |

---

## POST /upload/presigned-url

**Auth:** Bearer JWT obrigatório

**Request:**
```json
{
  "fileType": "avatar",
  "sheetUuid": "uuid-v4"
}
```

- `fileType`: `"avatar"` | `"cover"` | `"map_bg"`.
- Para `"avatar"`/`"cover"`: exige `sheetUuid`.
- Para `"map_bg"`: exige `mapUuid` no lugar de `sheetUuid` (usado pelo domínio de mapas — fora do escopo deste documento, ver doc de mapas).

**Response 200:**
```json
{
  "uploadUrl": "https://...r2.cloudflarestorage.com/...",
  "publicUrl": "https://pub.r2.dev/avatar/uuid.webp"
}
```

A chave do objeto no R2 usa um UUID novo gerado no servidor (não o UUID da ficha), para garantir cache-busting no CDN a cada upload.

**Erros:**
- `400` - `sheetUuid`/`mapUuid` ausente ou inválido para o `fileType` informado
- `422` - `fileType` inválido
- `401` - Unauthorized
- `500` - Internal Server Error

---

## GET /charactersheets/{uuid}

**Auth:** Bearer JWT obrigatório

### Parâmetros de Query

- `include` (opcional): lista separada por vírgula. Se contiver `submission`, adiciona campo `submission` à resposta.

### Response 200

Envelope `{ "characterSheet": {...} }`.

<a id="shape-de-charactersheet"></a>
### Shape de `characterSheet`

```json
{
  "uuid": "...",
  "playerUuid": "...",
  "masterUuid": null,
  "campaignUuid": null,
  "submission": null,

  "characterClass": "Hunter",
  "categoryName": "",
  "profile": {
    "nickname": "Gon",
    "fullname": "Gon Freecss",
    "alignment": "Chaotic-Good",
    "description": "Descrição longa opcional",
    "briefDescription": "Máx 255 chars",
    "avatarUrl": "https://pub.r2.dev/avatar/uuid.webp",
    "coverUrl": "https://pub.r2.dev/cover/uuid.webp",
    "birthday": "0000-05-15T00:00:00Z",
    "age": 12
  },

  "characterExp": { "level": 1, "exp": 0, "currExp": 0, "nextLvlBaseExp": 100, "points": 0 },
  "talent": { "level": 0, "exp": 0, "currExp": 0, "nextLvlBaseExp": 100 },
  "nenHexValue": null,

  "abilities": {
    "Physicals": { "level": 1, "exp": 0, "currExp": 0, "nextLvlBaseExp": 100, "bonus": 0 }
  },
  "physicalAttributes": {
    "Strength": { "level": 1, "exp": 0, "currExp": 0, "nextLvlBaseExp": 100, "points": 10, "value": 5, "power": 1 }
  },
  "mentalAttributes": {
    "Resilience": { "level": 1, "exp": 0, "currExp": 0, "nextLvlBaseExp": 100, "points": 0, "value": 0, "power": 0 }
  },
  "spiritualAttributes": {
    "Flame": { "level": 1, "exp": 0, "currExp": 0, "nextLvlBaseExp": 100, "power": 0 }
  },
  "physicalSkills": {
    "Vitality": { "level": 1, "exp": 3, "currExp": 3, "nextLvlBaseExp": 100, "value": 1 }
  },
  "mentalSkills": {},
  "spiritualSkills": {
    "Focus": { "level": 1, "exp": 0, "currExp": 0, "nextLvlBaseExp": 100, "value": 0 }
  },
  "principles": {
    "Ten": { "level": 1, "exp": 0, "currExp": 0, "nextLvlBaseExp": 100, "value": 0 }
  },
  "categories": {
    "Reinforcement": { "level": 1, "exp": 0, "currExp": 0, "nextLvlBaseExp": 100, "value": 0, "percent": 0 }
  },
  "commonProficiencies": {
    "Sword": { "level": 1, "exp": 2, "currExp": 2, "nextLvlBaseExp": 100 }
  },
  "jointProficiencies": {
    "<jointProficiencyName>": {
      "level": 1, "exp": 0, "currExp": 0, "nextLvlBaseExp": 100,
      "weapons": ["Sword", "Longsword"]
    }
  },
  "status": {
    "health": { "min": 0, "current": 20, "max": 20 },
    "stamina": { "min": 0, "current": 20, "max": 20 },
    "aura": { "min": 0, "current": 20, "max": 20 }
  }
}
```

Notas sobre o shape:
- `playerUuid` **xor** `masterUuid` fica presente (`omitempty`) conforme a origem da ficha — ver seção de criação acima.
- `campaignUuid` só aparece quando a ficha está vinculada a uma campanha.
- `nenHexValue` só aparece quando o hexágono de Nen da ficha já tem valor definido (`*int`, `omitempty`).
- As chaves de `abilities`/`physicalAttributes`/`mentalAttributes`/`spiritualAttributes`/`physicalSkills`/`mentalSkills`/`spiritualSkills`/`principles`/`categories`/`commonProficiencies` são os nomes reais dos enums (`AbilityName`, `AttributeName`, `SkillName`, `PrincipleName`, `CategoryName`, `WeaponName`), com a capitalização exata dos enums Go.
- `status` usa chaves em minúsculo (`health`, `stamina`, `aura` — via `strings.ToLower`), diferente de todos os outros mapas acima.
- Toda entrada de "experiência" (abilities, atributos, skills, principles, categories, proficiências) compartilha o mesmo bloco base `{ level, exp, currExp, nextLvlBaseExp }`, cada tipo adicionando campos próprios (`bonus`, `points`/`value`/`power`, `percent`, `weapons`).
- A chave de `jointProficiencies` é um nome de grupo livre (string, não vinculado a um enum fixo) definido no domínio de proficiências conjuntas — não fabricamos um valor real aqui; consulte `internal/domain/entity/character_sheet/proficiency/proficiency_manager.go` para a lógica de criação desses grupos.

Quando `include=submission` é enviado e existe submissão pendente:

```json
{
  "...": "campos anteriores",
  "submission": {
    "campaignUuid": "...",
    "createdAt": "2026-05-17T12:00:00Z"
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

## GET /charactersheets — Listar fichas do usuário

**Auth:** Bearer JWT obrigatório

Retorna as fichas do jogador autenticado, no shape de summary privado (ver [Shapes de summary](#shapes-de-summary)).

### Response 200

```json
{
  "characterSheets": [
    {
      "uuid": "...",
      "playerUuid": "...",
      "nickName": "Gon",
      "avatarUrl": "https://pub.r2.dev/avatar/uuid.webp",
      "createdAt": "2026-05-17T12:00:00Z",
      "updatedAt": "2026-05-17T12:00:00Z",
      "fullName": "Gon Freecss",
      "alignment": "Chaotic-Good",
      "characterClass": "Hunter",
      "birthday": "0000-05-15",
      "categoryName": "",
      "level": 1,
      "points": 0,
      "currExp": 0,
      "nextLvlBaseExp": 100,
      "talentLvl": 0,
      "physicalsLvl": 1,
      "mentalsLvl": 1,
      "spiritualsLvl": 1,
      "skillsLvl": 1,
      "stamina": { "min": 0, "current": 20, "max": 20 },
      "health": { "min": 0, "current": 20, "max": 20 }
    }
  ]
}
```

### Erros

| Status | Situação |
|--------|----------|
| 200 | Lista retornada (vazia se não houver fichas) |
| 400 | Bad Request |
| 401 | Unauthorized |
| 500 | Internal Server Error |

---

## PATCH /charactersheets/{uuid}

**Auth:** Bearer JWT (apenas o dono da ficha)

**Pré-condição:** Ficha deve estar livre (sem campanha e sem submissão pendente). Caso contrário, retorna `422 ErrCharacterSheetNotFreeToManage`.

### Request

**Mesmo formato exato que `POST /charactersheets`** — o handler reutiliza o mesmo `CreateCharacterSheetRequestBody` (verificado em `internal/app/api/sheet/update_character_sheet.go`: `Body CreateCharacterSheetRequestBody`). Isso inclui `campaignUuid`, `profile`, `characterClass`, `skillsExps`, `proficienciesExps` e `attributePoints` — todos com as mesmas regras de validação do create.

**Exceção: `campaignUuid` é ignorado no update.** `UpdateCharacterSheetUC.UpdateCharacterSheet` (`internal/application/character_sheet/update_character_sheet.go`) nunca lê `input.CampaignUUID` — a ficha atualizada é reconstruída com `rel.CampaignUUID`, obtido do registro existente no banco (`rel, err := uc.repo.GetCharacterSheetRelationshipUUIDs(...)`, ~linha 83). Ou seja, qualquer valor de `campaignUuid` enviado no corpo do PATCH é silenciosamente descartado — **PATCH não move uma ficha entre campanhas**.

```json
{
  "campaignUuid": null,
  "profile": {
    "nickname": "Gon",
    "fullname": "Gon Freecss",
    "alignment": "Chaotic-Good",
    "description": "Descrição longa opcional",
    "briefDescription": "Máx 255 chars",
    "birthday": "0000-05-15T00:00:00Z",
    "age": 12
  },
  "characterClass": "Hunter",
  "skillsExps": {},
  "proficienciesExps": {},
  "attributePoints": {}
}
```

### Response 200

Retorna a ficha atualizada, envelopada em `{ "characterSheet": {...} }` (mesmo shape completo de `GET /charactersheets/{uuid}`, sem `submission`).

### Erros

| Status | Situação |
|--------|----------|
| 200 | Ficha atualizada |
| 400 | UUID ou dados inválidos |
| 403 | Acesso negado (não é o dono) |
| 404 | Ficha ou classe de personagem não encontrada / campanha não encontrada |
| 422 | Ficha não está livre para editar (está em campanha ou com submissão pendente) / perfil inválido |
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
  "avatarUrl": "https://...",
  "coverUrl": "https://...",
  "briefDescription": "Máx 255 chars"
}
```

> **⚠️ Atenção — este endpoint NÃO faz partial update de verdade.** O
> repositório (`internal/gateway/pg/sheet/update_character_sheet_profile.go`)
> executa `UPDATE ... SET avatar_url = $1, cover_url = $2, brief_description = $3`
> **incondicionalmente**, sem `COALESCE` nem checagem de presença de campo. Como
> os três campos do request body são ponteiros (`*string` com `omitempty`
> apenas no *marshal*), o Go não distingue "campo omitido" de "campo enviado
> como `null`" no *unmarshal* — ambos chegam como `nil` no handler. Ou seja: se
> você enviar `{ "avatarUrl": "..." }` sem `coverUrl` nem `briefDescription`,
> o back-end vai **zerar (NULL) os outros dois campos no banco**, não apenas
> deixá-los como estavam. **O front-end deve sempre enviar os três campos
> juntos** (com os valores atuais para os que não mudaram, ou `null`
> deliberadamente para limpar).

**Response:** 204 No Content

**Erros:**
- `400` - Bad Request (UUID inválido)
- `401` - Unauthorized
- `404` - ficha não encontrada ou não pertence ao usuário
- `500` - Internal Server Error

---

<a id="shapes-de-summary"></a>
## Shapes de summary

Os endpoints que retornam listas/summaries de fichas (`GET /charactersheets`,
`GET /campaigns/{uuid}`, enrollments e participants de partida — ver
`internal/app/api/campaign/campaign_response.go`,
`internal/app/api/match/list_match_enrollments.go` e
`internal/app/api/match/get_match_participants.go`) usam os tipos definidos
em `internal/app/api/sheet/character_sheet_sumary_response.go`:

### `CharacterBaseSummaryResponse` (campos públicos, comuns a todos os summaries)

```json
{
  "uuid": "...",
  "playerUuid": "...",
  "masterUuid": null,
  "campaignUuid": null,
  "nickName": "Gon",
  "avatarUrl": "https://pub.r2.dev/avatar/uuid.webp",
  "coverUrl": "https://pub.r2.dev/cover/uuid.webp",
  "storyStartAt": "2026-01-01",
  "storyCurrentAt": "2026-03-10",
  "deadAt": null,
  "createdAt": "2026-05-17T12:00:00Z",
  "updatedAt": "2026-05-17T12:00:00Z"
}
```

Note que a chave é **`nickName`** (N maiúsculo) aqui, diferente de `nickname`
usado em `profile` no shape completo da ficha — são DTOs diferentes,
verificados separadamente no código. `avatarUrl`, `coverUrl`, `playerUuid`,
`masterUuid`, `campaignUuid`, `storyStartAt`, `storyCurrentAt` e `deadAt` são
todos `omitempty` — ausentes quando não aplicável (ex.: personagem sem
imagem, ficha ainda sem campanha, campanha sem data de história corrente).

### `CharacterPrivateOnlyResponse` (campos privados — dono da ficha / mestre da partida)

```json
{
  "fullName": "Gon Freecss",
  "alignment": "Chaotic-Good",
  "characterClass": "Hunter",
  "birthday": "0000-05-15",
  "categoryName": "",
  "currHexValue": null,
  "level": 3,
  "points": 2,
  "currExp": 450,
  "nextLvlBaseExp": 800,
  "talentLvl": 1,
  "physicalsLvl": 2,
  "mentalsLvl": 1,
  "spiritualsLvl": 1,
  "skillsLvl": 2,
  "stamina": { "min": 0, "current": 18, "max": 20 },
  "health": { "min": 0, "current": 20, "max": 20 }
}
```

- `CharacterPrivateSummaryResponse` = `CharacterBaseSummaryResponse` + `CharacterPrivateOnlyResponse` (embutidos, achatados no JSON) — é o shape usado hoje por `GET /charactersheets`.
- `CharacterPublicSummaryResponse` = apenas `CharacterBaseSummaryResponse`.
- `CharacterSheetWithVisibilityResponse` = `CharacterBaseSummaryResponse` + campo `"private"` (opcional, `*CharacterPrivateOnlyResponse`) — usado onde a visibilidade dos campos privados depende de quem está pedindo (ex.: participantes de partida).

### `currExp` / `nextLvlBaseExp` nos summaries

Os summaries privados incluem `currExp` e `nextLvlBaseExp` para alimentar a
barra de EXP nas telas de listagem.

#### Fonte dos dados

| Contexto | Fonte |
|----------|-------|
| Ficha completa (`GET /charactersheets/{uuid}`) | `characterExp.currExp` / `characterExp.nextLvlBaseExp`, calculados pelo domínio (`GetCurrentExp()`/`GetNextLvlBaseExp()`) após rebuild completo da ficha. **Esta é a fonte da verdade.** |
| Summary de lista (`GET /charactersheets`, campanhas, enrollments, participants) | derivados de `charExp` (coluna desnormalizada em `character_sheets`) via `deriveCurrExp`/`deriveNxtLvlBaseExp` + `charExpTable` — apenas para exibição. `level` também é recalculado a partir de `charExp` (não do `Level` armazenado), para evitar inconsistência entre os dois. |

#### Por que desnormalizar?

O `charExp` real é o acúmulo de todas as cascatas de XP (skills → atributos →
habilidades → personagem). Computar esse valor a partir do zero exigiria
restaurar a ficha inteira para cada item da lista. A coluna `char_exp` (no
banco) armazena esse valor já computado no momento do create/update da ficha.

**Regra:** Nunca use os campos derivados de `char_exp` nos summaries para
lógica de jogo. Qualquer operação que dependa de XP correto deve usar o shape
completo da ficha (`characterExp` em `GET /charactersheets/{uuid}`).
