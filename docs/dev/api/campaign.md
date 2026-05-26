# Campaign API

## DELETE /campaigns/{uuid} — Deletar campanha

**Auth:** JWT (master da campanha)

### Path Parameters

| Parâmetro | Tipo | Descrição |
|---|---|---|
| `uuid` | UUID v4 | UUID da campanha a deletar |

### Request

Sem body.

### Respostas

| Status | Situação |
|---|---|
| 204 | Campanha deletada com sucesso. Partidas não-iniciadas e submissions cascadeiam. |
| 400 | UUID inválido no path |
| 401 | Sem JWT |
| 403 | Usuário não é o master da campanha |
| 404 | Campanha não encontrada |
| 422 | Campanha possui ao menos uma partida que já foi iniciada (`game_start_at IS NOT NULL`) |
| 500 | Erro interno |

### Notas

- A deleção é atômica: se qualquer partida associada tiver `game_start_at != null`, a query não deleta e retorna 422.
- Partidas não-iniciadas e suas submissions são removidas via `ON DELETE CASCADE`.
- O check de `game_start_at IS NOT NULL` é feito diretamente no SQL (`NOT EXISTS` subquery), garantindo atomicidade contra race conditions onde uma partida pode iniciar entre a verificação de ownership e a deleção.
