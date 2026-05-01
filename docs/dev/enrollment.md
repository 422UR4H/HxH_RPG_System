# Submissão & Enrollment de Fichas

Documentação técnica dos fluxos de submissão de fichas a campanhas e enrollment
em partidas — foco nas cadeias de validação, regras de negócio não óbvias e
relacionamentos cross-entity.

---

## 1. Fluxo de Submissão

Um jogador submete sua ficha de personagem a uma campanha para solicitar participação.

### Cadeia de validação (`SubmitCharacterSheetUC.Submit`)

```
1. GetCharacterSheetPlayerUUID(sheetUUID)
   └─ Ficha não existe? → ErrCharacterSheetNotFound

2. playerUUID != userUUID?
   └─ → ErrNotCharacterSheetOwner

3. ExistsSubmittedCharacterSheet(sheetUUID)
   └─ Já submetida? → ErrCharacterAlreadySubmitted

4. GetCampaignMasterUUID(campaignUUID)
   └─ Campanha não existe? → ErrCampaignNotFound

5. playerUUID == masterUUID?
   └─ → ErrMasterCannotSubmitOwnSheet

6. SubmitCharacterSheet(sheetUUID, campaignUUID, time.Now())
```

**Regra de negócio principal:** O master de uma campanha **não pode** submeter
sua própria ficha àquela campanha. A validação compara o `playerUUID` da ficha
com o `masterUUID` da campanha — se forem iguais, a operação é bloqueada.

**Ponto de atenção:** A verificação de duplicidade usa `ExistsSubmittedCharacterSheet`
com apenas o `sheetUUID`. Isso significa que uma ficha só pode estar submetida
a **uma única campanha** por vez. Não há validação por par `(sheetUUID, campaignUUID)`.

---

## 2. Fluxo de Aceitação / Rejeição

O master da campanha aceita ou rejeita fichas submetidas. Ambos os fluxos seguem
o mesmo padrão de validação:

### Cadeia de validação (Accept e Reject)

```
1. GetSubmissionCampaignUUIDBySheetUUID(sheetUUID)
   └─ Submissão não existe? → ErrSubmissionNotFound

2. GetCampaignMasterUUID(campaignUUID)
   └─ Campanha não existe? → ErrCampaignNotFound

3. campaignMasterUUID != masterUUID?
   └─ → ErrNotCampaignMaster

4a. Accept → AcceptCharacterSheetSubmission(sheetUUID, campaignUUID)
4b. Reject → RejectCharacterSheetSubmission(sheetUUID)
```

**Diferença entre Accept e Reject:**
- `AcceptCharacterSheetSubmission` recebe `(sheetUUID, campaignUUID)` — precisa
  de ambos para vincular a ficha à campanha no banco.
- `RejectCharacterSheetSubmission` recebe apenas `(sheetUUID)` — remove o registro
  de submissão sem criar vínculo.

> **TODO preservado do código-fonte (presente em ambos os UCs):**
> `// TODO: optimize that 2 calls to db to only 1`
>
> Atualmente são feitas duas queries sequenciais: buscar o `campaignUUID` pela
> submissão e depois buscar o `masterUUID` pela campanha. Isso poderia ser um
> único JOIN no repositório.

---

## 3. Fluxo de Enrollment

Após a ficha ser aceita em uma campanha, o jogador a inscreve (enroll) em uma
partida específica (`Match`) daquela campanha.

### Cadeia de validação (`EnrollCharacterInMatchUC.Enroll`)

```
1. GetCharacterSheetRelationshipUUIDs(sheetUUID)
   └─ Ficha não existe? → ErrCharacterSheetNotFound

2. sheetRelationship.PlayerUUID == nil || *PlayerUUID != playerUUID?
   └─ → ErrNotCharacterSheetOwner

3. ExistsEnrolledCharacterSheet(sheetUUID, matchUUID)
   └─ Já inscrita nesta partida? → ErrCharacterAlreadyEnrolled

4. GetMatchCampaignUUID(matchUUID)
   └─ Partida não existe? → ErrMatchNotFound

5. sheetRelationship.CampaignUUID == nil || *CampaignUUID != campaignUUID?
   └─ → ErrCharacterNotInCampaign

6. EnrollCharacterSheet(matchUUID, sheetUUID)
```

**Insight chave — `CharacterSheetRelationshipUUIDs`:**

Este é um modelo de gateway (não uma entidade de domínio) que agrega os UUIDs de
relacionamento da ficha:

```go
type CharacterSheetRelationshipUUIDs struct {
    CampaignUUID *uuid.UUID  // nil se a ficha não está vinculada a nenhuma campanha
    PlayerUUID   *uuid.UUID  // nil se a ficha não tem jogador (ex: NPC)
    MasterUUID   *uuid.UUID  // nil se a ficha não tem master associado
}
```

Todos os campos são ponteiros (`*uuid.UUID`), o que significa que:
- A validação de ownership precisa checar `nil` antes de comparar valores.
- A validação de pertencimento à campanha precisa checar `nil` antes de comparar
  com o `campaignUUID` da partida.
- Fichas sem `PlayerUUID` (potencialmente NPCs) não podem ser enrolled por jogadores.

> **TODO preservado do código-fonte:**
> `// TODO: treat if the request was made by a master too`
>
> Atualmente apenas jogadores podem fazer enrollment. Há intenção de permitir que
> o master também inscreva fichas em partidas, mas isso ainda não foi implementado.

**Nota:** O bloqueio de mestre inscrever sua própria ficha é feito no fluxo de
`SubmitCharacterSheet`, que impede a submissão da ficha à campanha. Como o
enrollment já valida se a ficha pertence à campanha da partida, esse cenário
é coberto indiretamente.

---

## 4. Diagrama de Relacionamentos

```
  User (Player)
    │
    │ owns (PlayerUUID)
    ▼
  CharacterSheet ──submitted to──▶ Campaign ◀── belongs to ── Match
    │                                  │                         │
    │  (accept)                        │ (MasterUUID)            │
    │  CampaignUUID ← set             │                         │
    │                                  ▼                         │
    └──────── enroll ─────────────▶ Match ◀──────────────────────┘
              (requires sheet.CampaignUUID == match.CampaignUUID)
```

### Fluxo completo de vida de uma ficha em jogo:

```
1. Player cria CharacterSheet
2. Player submete ficha → Campaign  (submission)
3. Master aceita submissão          (accept → vincula CampaignUUID na ficha)
4. Player inscreve ficha → Match    (enrollment → valida CampaignUUID)
5. Ficha participa da partida
```

**Validação cross-entity no enrollment:** O sistema garante integridade verificando
que `sheetRelationship.CampaignUUID == match.CampaignUUID`. Isso impede que uma
ficha aceita na Campanha A seja inscrita em uma partida da Campanha B.

A verificação de duplicidade no enrollment é por par `(sheetUUID, matchUUID)` — diferente
da submissão, que é apenas por `sheetUUID`. Isso permite que a mesma ficha participe
de múltiplas partidas dentro da mesma campanha.

---

## Referências de Código

| Arquivo                                              | Responsabilidade                              |
|------------------------------------------------------|-----------------------------------------------|
| `internal/domain/submission/submit_character_sheet.go`  | UC de submissão de ficha a campanha        |
| `internal/domain/submission/accept_sheet_submission.go` | UC de aceitação pelo master                |
| `internal/domain/submission/reject_sheet_submission.go` | UC de rejeição pelo master                 |
| `internal/domain/submission/error.go`                   | Erros de domínio de Submission             |
| `internal/domain/submission/i_repository.go`            | Interface de repositório Submission        |
| `internal/domain/enrollment/enroll_character_sheet.go`  | UC de enrollment em partida                |
| `internal/domain/enrollment/error.go`                   | Erros de domínio de Enrollment             |
| `internal/domain/enrollment/i_repository.go`            | Interface de repositório Enrollment        |
| `internal/domain/campaign/i_repository.go`              | `GetCampaignMasterUUID` — usado por ambos  |
