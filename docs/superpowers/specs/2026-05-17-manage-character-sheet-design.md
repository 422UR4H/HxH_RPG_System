# Design: Gerenciar ficha de personagem

**Data:** 2026-05-17
**Status:** aprovado

## Visão geral

O jogador dono de uma ficha pode gerenciá-la diretamente na página de detalhes (`CharacterSheetPage`). Um botão compacto "Gerenciar" aparece no canto inferior esquerdo, ao lado do botão "Procurar Campanhas". Ao clicar, abre um menu inline com as ações disponíveis conforme o estado da ficha.

---

## Lógica de estado

O frontend deriva tudo a partir dos dados retornados pelo GET. Nenhum campo de estado computado é enviado pelo backend.

```
isFree = sheet.campaignUuid == null && submission == null

showGerenciar = sheet.playerUuid == user.uuid
```

O botão é **sempre visível** para o dono. O que varia é o conteúdo do menu:

| Estado          | Condição                                    | Menu                        |
|-----------------|---------------------------------------------|-----------------------------|
| FREE            | `campaignUuid == null && submission == null`| Editar (completo) + Excluir |
| PENDING         | `campaignUuid == null && submission != null`| Editar (parcial)            |
| ACCEPTED        | `campaignUuid != null`                      | Editar (parcial)            |

Não há estado "bloqueado" — a gestão de participação em partidas ativas não é tratada nesta feature (edição de campos cosméticos durante partida não gera inconsistência de domínio).

### Rotas de edição

- `isFree` → `/charactersheet/:id/edit` — edição completa (reutiliza template de criação, pré-populado)
- `!isFree` → `/charactersheet/:id/edit/profile` — edição parcial (avatar + capa + descrição)

---

## Backend

### Endpoints novos / modificados

#### `GET /charactersheets/{uuid}?include=submission`

Extensão opcional via query param. Quando `include=submission` está presente, faz JOIN com a tabela de submissions e retorna o objeto aninhado:

```json
{
  "submission": {
    "campaign_uuid": "...",
    "created_at": "..."
  }
}
```

`submission` é `null` quando não há submissão pendente. Sem o param, o comportamento atual é preservado.

#### `PATCH /charactersheets/{uuid}` *(novo)*

Edição completa. Disponível apenas quando `isFree`.

- **Auth:** owner da ficha
- **Guard:** `campaignUuid == null && sem submission pendente` → senão `422 ErrCharacterSheetNotFreeToManage`
- **Body:** mesmo payload da criação (classe, atributos, habilidades, proficiências, perfil)
- **Mecanismo:** `CharacterSheetFactory.Build` garante consistência de domínio → UPDATE em transação. Proficiências: DELETE das existentes + INSERT das novas.
- **Resposta:** `200` com a ficha atualizada (mesmo formato do GET)

#### `PATCH /charactersheets/{uuid}/profile` *(estender)*

Adiciona campo `description` ao body existente (que já aceita `avatar_url` e `cover_url`). Sem restrição de estado — campos cosméticos são sempre editáveis pelo owner.

- **Auth:** owner da ficha
- **Body:** `{ avatar_url?, cover_url?, description? }`
- **Resposta:** `200`

#### `DELETE /charactersheets/{uuid}` *(novo)*

- **Auth:** owner da ficha
- **Guard:** `isFree` → senão `422 ErrCharacterSheetNotFreeToManage`
- **Resposta:** `204 No Content`

### Validação de autorização (todos os endpoints mutantes)

1. Ficha existe → `404 ErrCharacterSheetNotFound`
2. Usuário é o `playerUuid` da ficha → `403 ErrInsufficientPermissions`
3. (PATCH full + DELETE) Estado isFree → `422 ErrCharacterSheetNotFreeToManage`

### Camadas impactadas

- `IRepository`: adicionar `DeleteCharacterSheet`, `UpdateCharacterSheet`
- `ISubmissionLookup`: `ExistsSubmittedCharacterSheet` já existe no mock (`mock_submission_repo.go`) e na implementação pg (`read_submission.go`) — sem mudança necessária
- Use cases: `DeleteCharacterSheetUC`, `UpdateCharacterSheetUC`
- Gateway pg/sheet: `deleteCharacterSheet`, `updateCharacterSheet` (UPDATE em transação)
- `routes.go`: registrar novos handlers
- `main.go`: wiring dos novos use cases
- `mock_character_sheet_repo.go`: adicionar métodos novos ao mock

---

## Frontend

### Tipos e serviços

**`src/types/characterSheet.ts`**
- Adicionar `uuid: string` ao tipo `CharacterSheet`
- Adicionar tipo `Submission: { campaignUuid: string; createdAt: string } | null`
- Adicionar campo `submission?: Submission` ao tipo `CharacterSheet`

**`src/services/characterSheetsService.ts`**
- `getCharacterSheetDetails`: passar `?include=submission`
- Novo: `deleteCharacterSheet(token, uuid): Promise<void>`
- Novo: `updateCharacterSheet(token, uuid, sheet): Promise<CharacterSheet>`
- `patchCharacterSheetProfile`: adicionar parâmetro `description?: string | null`

**Hooks**
- Novo: `useDeleteCharacterSheet` — mutation com invalidação de query
- Novo: `useUpdateCharacterSheet` — mutation com invalidação de query

### Componentes

#### `ManageButton` *(novo)*

Botão compacto com menu inline. Props:
```ts
interface ManageButtonProps {
  isFree: boolean;
  onEdit: () => void;
  onDelete: () => void;
  isFloating: boolean; // controlado pelo pai
}
```

- Estilo: `background: #1c1c1c; border: 1px solid #555; border-radius: 8px` (ancorado) / `border-radius: 50px` (flutuante)
- Menu abre acima do botão (`bottom: 100%`)
- Quando aberto: borda muda para `#ffa216`
- FREE: exibe "Editar" + "Excluir"; !FREE: exibe só "Editar"
- Excluir: cor `#f38ba8` (vermelho)
- Fecha ao clicar fora (listener no document)
- "Excluir" abre um `window.confirm` antes de disparar `onDelete` — prevenção de deleção acidental

#### `SheetCampaignButton` → refatorar para `SheetBottomActions`

O componente atual vira um wrapper que gerencia detecção de flutuação **uma vez** e renderiza os dois botões:

- **Ancorado:** row, `ManageButton` à esquerda (auto-width), `Procurar Campanhas` ocupa o restante (`flex: 1`)
- **Flutuante:** `ManageButton` fixado `bottom: 20px; left: 60px`; `Procurar Campanhas` fixado `bottom: 20px; right: 60px`
- Quando só `ManageButton` presente (sem `onCampaignClick`): compacto à esquerda, sem esticar

`MainContent.$hasCampaignButton` passa a se chamar `$hasBottomActions` e é `true` quando qualquer um dos dois botões está presente.

#### `CharacterSheetTemplate`

- Aceita nova prop `onManageClick?: { isFree: boolean; onEdit: () => void; onDelete: () => void }`
- Renderiza `SheetBottomActions` quando `onCampaignClick || onManageClick` estiver presente

#### `CharacterSheetPage`

Deriva o estado e passa as props:

```ts
const isFree = !charSheet.campaignUuid && !charSheet.submission;
const isOwner = charSheet.playerUuid === user.uuid;

// navegar para edit ou edit/profile
const handleEdit = () => navigate(isFree
  ? `/charactersheet/${id}/edit`
  : `/charactersheet/${id}/edit/profile`
);
```

### Novas páginas e rotas

#### `/charactersheet/:id/edit`

Edição completa — reutiliza `CharacterSheetTemplate` com `sheetMode` igual ao de criação, mas pré-populado com a ficha existente. Chama `updateCharacterSheet` no submit.

Guard de rota: redireciona para `/charactersheet/:id` se `!isFree`.

#### `/charactersheet/:id/edit/profile`

Edição parcial — página leve com apenas:
- Seletor de avatar (reutiliza lógica existente de upload)
- Seletor de capa (idem)
- Textarea de descrição breve

Chama `patchCharacterSheetProfile` no submit.

#### `App.tsx`

Adicionar as duas rotas protegidas.

---

## Fluxo de upload de imagens (edição)

Idêntico ao de criação: `getPresignedUrl` → `uploadToR2` → `patchCharacterSheetProfile` com a URL pública. Para a edição completa, o upload de avatar/capa acontece após o `updateCharacterSheet`, usando o mesmo UUID da ficha.

---

## Erros relevantes

| Código | Constante                          | Situação                                              |
|--------|------------------------------------|-------------------------------------------------------|
| 404    | `ErrCharacterSheetNotFound`        | Ficha não existe                                      |
| 403    | `ErrInsufficientPermissions`       | Usuário não é o player da ficha                       |
| 422    | `ErrCharacterSheetNotFreeToManage` | Tentativa de editar/excluir fora do estado FREE       |

---

## Fora de escopo

- Gerenciar ficha durante partida ativa (edição cosmética é segura; bloqueio não agrega valor agora)
- Soft delete (DELETE é hard delete com guard de estado)
- Histórico de edições
