# Design: Testes das Entidades de Domínio Restantes

**Data:** 2026-04-28
**Status:** Completo
**Escopo:** Adicionar cobertura de testes abrangente para os pacotes de entidades de domínio die, item, character_class, enum e match

## Declaração do Problema

Após completar os testes de character_sheet (169+ testes em 8 pacotes), os pacotes de entidades de domínio restantes tinham zero cobertura de testes. Esses pacotes contêm mecânicas críticas do jogo: rolagem de dados, sistema de armas, validação de classes de personagem, parsing de enums e ordenação de ações de combate.

## Abordagem

Testes bottom-up organizados por grafo de dependências:

- **Fase 1 (Independentes):** die → item → character_class → enum (sem dependências entre domínios)
- **Fase 2 (Domínio match):** match/action → match (depende de die, enum)

Excluídos do escopo:
- `match/scene` — importa pacote `turn` (refatoração semântica Turn/Round em andamento)
- `match/battle/blow` — struct sem métodos
- `match/round` — importa pacote `turn`

## Estratégia de Testes

### Princípios Aplicados

- Apenas `testing` da biblioteca padrão, table-driven com `t.Run()`
- Pacotes de teste externos (`package foo_test`)
- Validação de segurança de cópia (ex.: `Weapon.GetDice()` retorna cópia)
- Verificação de completude de factories (todos os enums construídos)
- Teste de aleatoriedade via validação de intervalo em múltiplas iterações
- Verificação de invariante de heap para PriorityQueue

### Resumo de Cobertura

| Pacote | Arquivo | Testes | Comportamentos Chave |
|--------|---------|--------|---------------------|
| die | die_test.go | 3 | Construção, intervalo Roll [1,N], rastreamento de estado |
| item | weapon_test.go | 5 | Getters, segurança de cópia, lógica penalidade/stamina |
| item | weapons_manager_test.go | 5 | CRUD, delegação, propagação de erros |
| item | weapons_factory_test.go | 2 | Constrói todas 40 armas, verificação de propriedades |
| character_class | character_class_test.go | 7 | Validar/Aplicar skills & proficiências, Distribution |
| character_class | character_class_factory_test.go | 3 | Constrói todas 12 classes, skills, distribuição |
| enum | enum_test.go | 7 | Parsers NameFrom, DieSides, tamanhos de coleções |
| match/action | priority_queue_test.go | 6 | Ordenação max-heap, Insert, Extract, Peek, ExtractByID |
| match/action | action_test.go | 4 | Construção, unicidade UUID, RollContext |
| match | match_test.go | 2 | Campos NewMatch, AddScene/GetScenes |
| match | game_event_test.go | 1 (3 sub) | Categorias, defaults nil, mudança de data |

**Total: 45 funções de teste em 11 arquivos**

## Descobertas Importantes

1. **RollContext.GetDiceResult** possui parâmetro não utilizado `d die.Die` — soma todos os dados no contexto independente do parâmetro
2. **CharacterClassFactory** constrói 12 classes, mas enum retorna 16 nomes (4 estão comentadas: Athlete, Tribal, Experiment, Circus)
3. **Weapon.GetDice()** retorna cópia via `copy()` — consistente com preferência do proprietário por retornos seguros
4. **PriorityQueue** inverte `Less()` para criar max-heap — maior velocidade = maior prioridade
5. **GameEvent** tem todos os campos não-exportados sem getters — testado via verificação de construção

## Documentação Criada

Documentação do jogo (PT-BR):
- `docs/game/dados.md` — Mecânicas de dados
- `docs/game/armas.md` — Sistema de armas (todas 40 armas)
- `docs/game/classes.md` — Sistema de classes de personagem
- `docs/game/combate/acoes.md` — Ações de combate, fila de prioridade, partida/eventos

## Commits

1. `ef0aded` — test(die): Testes de Die
2. `69a3f88` — test(item): Testes de Weapon, WeaponsManager, Factory
3. `ef603de` — test(character_class): Testes de CharacterClass, Factory
4. `a36df75` — test(enum): Testes de parser e coleções de enum
5. `55c4ef5` — test(match): Testes de Action, PriorityQueue, Match, GameEvent
