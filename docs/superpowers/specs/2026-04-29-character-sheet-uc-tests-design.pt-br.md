# Fase 4: Testes dos Use Cases de CharacterSheet — Spec de Design

**Data:** 29/04/2026  
**Escopo:** Testes unitários para os 4 use cases de domínio do CharacterSheet  
**Branch:** `feat/character-sheet-uc-tests`

## Problema

O pacote de domínio CharacterSheet possui 4 use cases sem cobertura de testes:
1. `CreateCharacterSheetUC` — construção complexa de entidade com validação multi-etapa
2. `GetCharacterSheetUC` — verificação de permissões + hidratação de model para entidade
3. `ListCharacterSheetsUC` — delegação simples para o repositório
4. `UpdateNenHexagonValueUC` — despacho de método + persistência

Foram adiados da Fase 3 devido à complexidade (200+ linhas para Create, 130+ para Get).

## Abordagem

### Estratégia de Testes

Usar **objetos de domínio reais** (factory, character classes) combinados com **repositórios mockados**:

- `CharacterSheetFactory` é uma função pura (sem I/O) — usar diretamente
- Instâncias de `CharacterClass` são construídas pela factory existente (ex: `BuildSwordsman()`)
- Interações com repositório são mockadas via `testutil.MockCharacterSheetRepo` e `testutil.MockCampaignRepo` existentes
- `sync.Map` é populado no setup de teste com dados reais de classe

### Helper de Teste

Um helper compartilhado em `internal/domain/character_sheet/testutil_test.go` (pacote de teste externo) fornecendo:
- `newTestFactory()` — retorna um `*sheet.CharacterSheetFactory`
- `newTestClassMap()` — retorna um `*sync.Map` pré-carregado com a classe Swordsman
- `newValidInput()` — retorna um `*CreateCharacterSheetInput` que passa em toda validação
- `newValidModelSheet()` — retorna um `*model.CharacterSheet` para hidratação do GetCharacterSheet

## Casos de Teste

### CreateCharacterSheetUC (13 casos)

| # | Caso | Esperado |
|---|------|----------|
| 1 | Caminho feliz (player, sem campanha) | Retorna `*CharacterSheet`, sem erro |
| 2 | Caminho feliz (master, sem campanha) | Retorna `*CharacterSheet`, sem erro |
| 3 | Caminho feliz (player, com campanha) | Retorna `*CharacterSheet`, sem erro |
| 4 | Classe não encontrada | `ErrCharacterClassNotFound` (wrapped) |
| 5 | Distribuição de skills inválida | Erro de validação da CharacterClass |
| 6 | Distribuição de proficiencies inválida | Erro de validação da CharacterClass |
| 7 | Nickname igual ao nome de uma classe | `ErrNicknameNotAllowed` (wrapped) |
| 8 | Limite de fichas por player excedido (>=20) | `ErrMaxCharacterSheetsLimit` |
| 9 | Erro do repo ao contar fichas | Erro propagado |
| 10 | Campanha não encontrada | `domainCampaign.ErrCampaignNotFound` |
| 11 | Erro do repo de campanha | Erro propagado |
| 12 | Não é dono da campanha | `domainCampaign.ErrNotCampaignOwner` |
| 13 | Erro do repo ao criar ficha | Erro propagado |

### GetCharacterSheetUC (9 casos)

| # | Caso | Esperado |
|---|------|----------|
| 1 | Caminho feliz — usuário é master | Retorna `*CharacterSheet` hidratado |
| 2 | Caminho feliz — usuário é player | Retorna `*CharacterSheet` hidratado |
| 3 | Caminho feliz — usuário é master da campanha | Retorna `*CharacterSheet` hidratado |
| 4 | Ficha não encontrada | `ErrCharacterSheetNotFound` |
| 5 | Erro do repo ao buscar | Erro propagado |
| 6 | Permissão insuficiente (sem campanha) | `auth.ErrInsufficientPermissions` |
| 7 | Permissão insuficiente (não é master da campanha) | `auth.ErrInsufficientPermissions` |
| 8 | Campanha não encontrada ao verificar permissão | `domainCampaign.ErrCampaignNotFound` |
| 9 | Erro do repo de campanha ao verificar permissão | Erro propagado |

### ListCharacterSheetsUC (3 casos)

| # | Caso | Esperado |
|---|------|----------|
| 1 | Caminho feliz — retorna lista | `[]model.CharacterSheetSummary`, sem erro |
| 2 | Caminho feliz — lista vazia | Slice vazio, sem erro |
| 3 | Erro do repo | Erro propagado |

### UpdateNenHexagonValueUC (5 casos)

| # | Caso | Esperado |
|---|------|----------|
| 1 | Caminho feliz — increase | Retorna `*NenHexagonUpdateResult` |
| 2 | Caminho feliz — decrease | Retorna `*NenHexagonUpdateResult` |
| 3 | Método inválido | `ErrInvalidUpdateHexValMethod` |
| 4 | Erro da entidade no increase/decrease | Erro propagado |
| 5 | Erro do repo ao atualizar | `domain.DBError` encapsulando erro do repo |

**Total: 30 casos de teste**

## Estrutura de Arquivos

```
internal/domain/character_sheet/
├── create_character_sheet_test.go   (13 casos)
├── get_character_sheet_test.go      (9 casos)
├── list_character_sheets_test.go    (3 casos)
├── update_nen_hexagon_value_test.go (5 casos)
└── testutil_test.go                 (helpers compartilhados)
```

## Decisões-Chave de Design

1. **Pacote de teste externo** (`charactersheet_test`) — testes importam o pacote como consumidor
2. **Factory real** — evita mockar construção complexa de entidades; garante correção de integração
3. **Classe Swordsman** — mais simples (sem Distribution), então `SkillsExps` e `ProficienciesExps` devem ser vazios para validação passar
4. **Classe Ninja para testes de distribuição** — tem `Distribution` com proficiencies/pontos específicos permitidos, habilitando testes de erro de validação
5. **Importações de erros do gateway** — testes importam pacotes de erro `pgCampaign` e `pgSheet` para acionar caminhos de tradução de erros nos UCs
6. **Testes table-driven** com sub-testes `t.Run()` — consistente com padrão da Fase 3

## Dependências

- Mocks existentes: `testutil.MockCharacterSheetRepo`, `testutil.MockCampaignRepo`
- Reais: `sheet.CharacterSheetFactory`, `characterclass.BuildSwordsman()`, `characterclass.BuildNinja()`
- Erros do gateway: `pgCampaign.ErrCampaignNotFound`, `pgSheet.ErrCharacterSheetNotFound`
