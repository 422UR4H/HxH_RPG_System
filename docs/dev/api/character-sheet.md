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
