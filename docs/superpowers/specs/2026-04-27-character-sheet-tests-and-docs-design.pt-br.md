# Design: Testes da Ficha de Personagem e Documentação do Jogo

**Data:** 27/04/2026
**Status:** Aprovado
**Escopo:** Adicionar cobertura abrangente de testes para as entidades de domínio da ficha de personagem (character_sheet) + criar documentação escalável das regras do jogo

## Declaração do Problema

O back-end do HxH RPG System possui um modelo de domínio rico para fichas de personagem com mais de 50 tipos de entidades, um sistema sofisticado de cascata de experiência, um sistema espiritual Nen e mecânicas de combate. Atualmente existe apenas 1 arquivo de teste (engine de turno, atualmente quebrado). Antes de qualquer refatoração ser feita com segurança (simplificação do fluxo de experiência, engines → domain services, POO → padrões idiomáticos do Go), precisamos de uma rede de segurança abrangente com testes.

Além disso, as regras do RPG existem apenas implicitamente no código — elas precisam ser documentadas para jogadores (em português) e para agentes de IA que navegam no código-fonte.

## Abordagem: Testes + Docs Juntos (Abordagem A)

Escrever testes bottom-up (das peças menores para as maiores) para cada sub-pacote, documentando as regras do jogo correspondentes conforme cada pacote é compreendido durante a escrita dos testes. Isso maximiza a eficiência de contexto (escrever docs enquanto o código está fresco no contexto) e garante que cada pacote sai "completo" (testado + documentado).

## Estrutura de Documentação

```
docs/
├── game/                              # Regras do jogo (PT-BR, para jogadores + IA)
│   ├── glossario.md                   # Glossário: PT-BR ↔ EN de palavras-chave
│   ├── ficha-de-personagem/           # Regras da ficha de personagem
│   │   ├── habilidades.md             # Habilidades (Abilities)
│   │   ├── atributos.md               # Atributos (Attributes)
│   │   ├── pericias.md                # Perícias (Skills)
│   │   ├── proficiencias.md           # Proficiências (Proficiencies)
│   │   ├── sistema-nen.md             # Nen: Princípios, Categorias, Hexágono
│   │   ├── experiencia.md             # Sistema de XP e cascata de evolução
│   │   └── status.md                  # Barras de Status (HP, Stamina, Aura)
│   └── classes/                       # Uma doc por classe (futuro)
├── architecture/                      # Docs técnicas (EN, para devs + IA)
│   └── overview.md                    # Visão geral da arquitetura
AGENTS.md                             # Na raiz: documento de contexto para agentes de IA
```

### Convenções de Documentação

- Docs do jogo em português com termos em inglês entre parênteses: "Vitalidade (Vitality)"
- Primeira ocorrência em cada documento: forma bilíngue completa
- Ocorrências seguintes no mesmo doc: pode usar apenas o português
- Todas as palavras-chave do jogo (habilidades, perícias, atributos, proficiências, categorias, classes, princípios, status, armas, modos de turno) seguem essa convenção
- O glossário é a fonte única de verdade para traduções
- O criador do sistema pode sugerir traduções alternativas; o glossário e todas as referências devem ser atualizados conforme necessário

## Estratégia de Testes

### Princípios

- **Apenas biblioteca padrão**: pacote `testing` do Go, sem frameworks de teste externos
- **Table-driven tests**: convenção idiomática do Go usando `[]struct` + `t.Run()` — é uma forma de definir vários casos de teste como uma lista de structs e iterar sobre eles
- **Mocks manuais**: quando necessário, escritos à mão em Go (sem frameworks de mock)
- **Helpers de teste (fixtures)**: funções factory que constroem objetos pré-configurados para testes — similares a builders/factories
- **Estratégia mista**: testes unitários bottom-up primeiro, depois testes de cascata/integração usando objetos reais compostos (sem mocks para os testes de cascata)

### Localização dos Arquivos de Teste

Cada `_test.go` fica ao lado do arquivo que testa (convenção Go):
```
internal/domain/entity/character_sheet/<pacote>/<arquivo>_test.go
```

### Ordem Bottom-Up (seguindo o grafo de dependências)

1. **`experience/`** — ExpTable → Exp → CharacterExp
   - Fundação de todo o sistema de progressão
   - Testes: limiares de subida de nível, acúmulo de XP, gatilhos de cascata

2. **`ability/`** — Talent → Ability → Manager
   - Depende de: experience
   - Testes: cálculo de bônus, propagação de upgrade em cascata, pontos de personagem

3. **`status/`** — Bar → HealthBar/StaminaBar/AuraBar → Manager
   - Depende de: experience (para mecânica de upgrade)
   - Testes: aumento/diminuição de barras, limites (mín/máx), upgrade ao subir de nível

4. **`attribute/`** — PrimaryAttribute → MiddleAttribute → Manager → CharacterAttributes
   - Depende de: experience, ability
   - Testes: separação físico/mental/espiritual, pontos distribuíveis, cálculo de poder

5. **`skill/`** — CommonSkill → SpecialSkill → JointSkill → Manager → CharacterSkills
   - Depende de: attribute, ability, experience
   - Testes: cálculo de valor para teste, composição de joint skill, gatilhos de cascata

6. **`proficiency/`** — Proficiency → JointProficiency → Manager
   - Depende de: skill, ability, experience
   - Testes: valores específicos por arma, buffs de joint proficiency, gatilhos de cascata

7. **`spiritual/`** — NenPrinciple → NenCategory → NenHexagon → Manager
   - Depende de: attribute, experience
   - Testes: níveis de princípios Nen, porcentagens de categorias, distribuição de valores no hexágono

8. **`sheet/`** — CharacterProfile → CharacterSheet (raiz do agregado)
   - Depende de: TODOS os anteriores
   - Testes: validação de perfil, testes de cascata do agregado (XP em perícia → subida de nível de habilidade → subida de nível do personagem)

### Testes de Cascata (no nível da sheet)

Após todos os sub-pacotes terem testes unitários, escrevemos testes no estilo integração no nível da CharacterSheet que exercitam a cascata completa de experiência:
- XP de Perícia → subida de nível da Habilidade → subida de nível do Personagem
- Distribuição de atributos → mudanças no valor das perícias
- Mudanças no hexágono Nen → efeitos nas porcentagens das categorias
- Upgrades de barras de status ao subir de nível

Esses testes usam objetos reais compostos (sem mocks) para validar que o sistema funciona de ponta a ponta.

## Estrutura do AGENTS.md

Documento compacto (~200-400 linhas) na raiz do projeto:
1. Visão geral do projeto — O que é, contexto HxH RPG
2. Arquitetura — Camadas (entity → usecase → app → gateway), convenções de pacotes
3. Mapa do domínio — Onde encontrar cada conceito
4. Convenções de código — Go idiomático, engines como domain services, interfaces implícitas, sem frameworks de teste
5. Glossário rápido — Referência ao `docs/game/glossario.md` + top 20 termos inline
6. Estado atual — O que está estável, o que está em WIP (Turn/Round), o que precisa de refatoração
7. Como testar — `go test ./...`, convenções de table-driven tests
8. Como buildar — `make build`, `make run-dev`

## Fluxo de Trabalho por Sub-Pacote

```
1. Ler o código do sub-pacote
2. Escrever testes (table-driven, helpers reutilizáveis)
3. Rodar testes → garantir que passam (cobrindo o comportamento ATUAL)
4. Documentar as regras descobertas em docs/game/
5. Atualizar glossário se novos termos aparecerem
6. Seguir para o próximo sub-pacote
```

## Limites de Escopo

### Dentro do escopo
- Testes para todos os 8 sub-pacotes da character_sheet
- Testes de cascata no nível do agregado CharacterSheet
- Documentação das regras do jogo (docs/game/)
- Glossário (docs/game/glossario.md)
- Visão geral da arquitetura (docs/architecture/overview.md)
- AGENTS.md

### Fora do escopo
- ❌ Refatoração de código de qualquer tipo
- ❌ Pacotes Turn/Round/Action (atualmente quebrados/em andamento)
- ❌ Testes da camada de usecases
- ❌ Testes da camada de gateway/repositório
- ❌ Testes da camada de app/API
- ❌ Pacotes de teste de terceiros
- ❌ Testes da entidade de classes de personagem (não faz parte dos sub-pacotes da character_sheet)
- ❌ Testes das entidades Die, Item, Match

## Critérios de Sucesso

1. Todos os 8 sub-pacotes da character_sheet possuem arquivos de teste abrangentes
2. `go test ./internal/domain/entity/character_sheet/...` passa com 0 falhas
3. Testes de cascata no nível da sheet validam a propagação de XP de ponta a ponta
4. Glossário cobre todas as palavras-chave do jogo com mapeamento PT-BR ↔ EN
5. Cada sub-pacote possui documentação correspondente das regras do jogo
6. AGENTS.md fornece contexto completo do projeto para agentes de IA
7. Visão geral da arquitetura documenta a estrutura de camadas
