# Design: DELETE /campaigns/{uuid}

## Overview

Endpoint de deleção de campanha seguindo o mesmo padrão do endpoint de deleção de partida. Uma campanha pode ser deletada pelo seu master desde que nenhuma partida associada tenha sido iniciada. Partidas não-iniciadas e submissions são deletados via ON DELETE CASCADE.

## Migration

Arquivo: `migrations/<timestamp>_add_cascade_to_campaign_fks.sql`

- Drop + re-add FK `matches.campaign_uuid → campaigns(uuid) ON DELETE CASCADE`
- Drop + re-add FK `submissions.campaign_uuid → campaigns(uuid) ON DELETE CASCADE`

Justificativa: as FKs atuais não têm CASCADE, então deletar uma campanha com matches ou submissions causa FK violation. O projeto já tem precedente disso na migration `20260525000000_add_cascade_to_enrollments_match_fk.sql`.

## Gateway

Arquivo: `internal/gateway/pg/campaign/delete_campaign.go`

Método `DeleteCampaign(ctx context.Context, uuid uuid.UUID) error` no `Repository`.

Query atômica que só deleta se não existir match iniciado:

```sql
DELETE FROM campaigns WHERE uuid = $1
AND NOT EXISTS (
    SELECT 1 FROM matches
    WHERE campaign_uuid = $1 AND game_start_at IS NOT NULL
)
```

- `RowsAffected() == 0` → retorna `ErrCampaignNotFound`

## Application (Use Case)

Arquivo: `internal/application/campaign/delete_campaign.go`

Interface `IDeleteCampaign` com método `Delete(ctx, *DeleteCampaignInput) error`.

`DeleteCampaignInput`:
```go
type DeleteCampaignInput struct {
    CampaignUUID uuid.UUID
    MasterUUID   uuid.UUID
}
```

Fluxo do `DeleteCampaignUC.Delete`:
1. `repo.GetCampaignMasterUUID(ctx, campaignUUID)` → `ErrCampaignNotFound` se não encontrar
2. `masterUUID != input.MasterUUID` → `ErrNotCampaignOwner`
3. `repo.DeleteCampaign(ctx, campaignUUID)` → se `ErrCampaignNotFound` (race condition: match iniciou entre os passos) → `ErrCampaignHasStartedMatch`

Novo erro em `error.go`: `ErrCampaignHasStartedMatch`.

## Handler

Arquivo: `internal/app/api/campaign/delete_campaign.go`

Segue exatamente o padrão de `delete_match.go`:
- Extrai `userUUID` do contexto
- Parseia `uuid` do path
- Chama `uc.Delete`
- Mapeia erros:

| Erro da UC | Status HTTP |
|---|---|
| `ErrCampaignNotFound` | 404 Not Found |
| `ErrNotCampaignOwner` | 403 Forbidden |
| `ErrCampaignHasStartedMatch` | 422 Unprocessable Entity |
| outros | 500 Internal Server Error |

Resposta de sucesso: `204 No Content`.

## Route

Arquivo: `internal/app/api/campaign/routes.go`

- Novo campo `DeleteCampaignHandler` na struct `Api`
- `huma.Register` com `Method: DELETE`, `Path: /campaigns/{uuid}`, `DefaultStatus: 204`
- Errors declarados: 400, 403, 404, 422, 500

## Wiring

Arquivo: `cmd/api/main.go`

```go
deleteCampaignUC := campaign.NewDeleteCampaignUC(campaignRepo)
// adicionar ao campaignsApi:
DeleteCampaignHandler: campaignHandler.DeleteCampaignHandler(deleteCampaignUC),
```

## Testes

### Handler (unit, humatest)

Arquivo: `internal/app/api/campaign/delete_campaign_test.go`

Casos: success (204), invalid_uuid (400), not_found (404), not_owner (403), has_started_match (422), internal_error (500).

Mock `mockDeleteCampaign` adicionado em `mocks_test.go`.

### Gateway (integration)

Adicionado em `internal/gateway/pg/campaign/campaign_integration_test.go` (`TestDeleteCampaign`):
- happy path: campanha sem matches deletada com sucesso
- campaign_not_found: UUID inexistente → ErrCampaignNotFound
- has_started_match: campanha com match que tem `game_start_at != NULL` → ErrCampaignNotFound (0 rows)
- cascade: campanha com match não-iniciado → match deletado junto

## Contrato de API

Arquivo: `docs/dev/api/delete-campaign.md` — criado e registrado em `documentation-map.yaml`.
