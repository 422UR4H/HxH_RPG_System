# Prévia ao vivo da ficha — pacote de back — design

> Plano do front (os dois repos): HxH_RPG_Environment_React,
> `docs/superpowers/plans/2026-11-01-sheet-live-preview.md`.
> Doc de dev: [`../../dev/character-sheet/factory.md`](../../dev/character-sheet/factory.md) ("Paridade com o front").

## Problema

Na criação e na edição da ficha, o front mostrava os valores congelados na ficha-modelo da classe:
poder, valores de perícia, princípios, nível do personagem e barras de vida/estamina/aura não
acompanhavam os pontos distribuídos nem as proficiências escolhidas.

## Decisões (dono do produto, 2026-11-01)

| # | Decisão |
|---|---|
| D1 | Duas implementações: o back (Go) continua sendo a fonte da verdade; o front repete a mesma montagem em TypeScript só para a prévia. Sem endpoint de prévia, sem WASM. |
| D2 | O front envia só os valores-chave (classe, `skillsExps`, `proficienciesExps`, `attributePoints`), como antes; o back remonta tudo a partir deles. |
| D3 | O motor do front fica desacoplado do app, para poder virar biblioteca compartilhada depois. |

## Mudanças no back

1. **`charactersheet.AssembleCharacterSheet`** — o único caminho de montagem (valida a
   distribuição, aplica sobre a classe, `Build`, `ApplyInitialAttributePoints`). Criação e edição
   usam essa função; antes cada caso de uso repetia os passos.
2. **`GET /classes` expõe `definition`** (`ClassDefinitionResponse`): os dados brutos de montagem
   da classe (`skillsExps`, `attributesExps`, `proficienciesExps`, `jointProficiencies`,
   `jointProfExps`). Os campos que já existiam são a ficha-modelo calculada; o motor do front
   precisa da entrada para repetir a montagem. `jointProficiencies` (calculado e definição) vêm
   ordenados por nome.
3. **Golden file de paridade** — `internal/app/api/sheet/sheet_parity_test.go` grava
   `testdata/sheet_parity.json`: para cada classe e alguns cenários, o envio do front e a resposta
   que o back monta, mais constantes e tabelas de EXP. Sem `-update` o teste falha se o arquivo
   estiver velho, o que obriga a regenerar sempre que uma regra da ficha muda. O front copia o
   arquivo e roda o próprio motor contra cada caso.

## Constatações

- Montar a mesma ficha 6 vezes dá o mesmo resultado: a ordem dos mapas do Go não muda a ficha (a
  cascata não depende da ordem), premissa da qual o motor do front depende.
- Com as classes de hoje, escolher proficiência nunca muda o nível de Físicos (a quantidade de
  pontos de atributo não muda), mas sobe o nível do personagem (ex.: Hunter 6 → 7).
- Na criação, o front mandava pontos em atributos físicos derivados (Força…), que o back recusa.
  Criação e edição agora usam um só montador do envio, que manda só os primários.

## Fora de escopo

`JointSkills` (nenhuma classe usa; o teste de paridade falha se alguma passar a usar),
hexágono Nen/percentuais de categoria (criação e edição montam sem eles), EXP de talento, buffs.
