# Rules API

> Corpo em **camelCase**. Rota registrada em `internal/app/api/rules/routes.go`;
> handler em `internal/app/api/rules/get_sheet_rules.go`. Os capítulos vêm de
> `docs/game/` (embutidos no binário por `docs/embed.go`) e são carregados **uma
> vez no boot** por `internal/application/rules` (`SheetChapterSources`).

## GET /rules/character-sheet — Regras da ficha

Devolve os capítulos de regras exibidos no painel "Regras · criação de personagem"
da ficha. Conteúdo estático: só muda com um novo deploy.

**Auth:** nenhuma (rota pública). Sem body de request nem parâmetros.

### Response 200

Header: `Cache-Control: public, max-age=300`.

```json
{
  "chapters": [
    {
      "slug": "criacao",
      "label": "Criação",
      "title": "Criando seu personagem",
      "markdown": "Criar uma ficha leva cinco passos, na ordem em que aparecem na tela.\n\n1. **Preencha o perfil**: apelido, nome, data de nascimento e o restante dos dados do personagem.\n2. **Escolha uma classe**: veja [Classes](../classes.md).\n",
      "comingSoon": false
    }
  ]
}
```

(O exemplo mostra um capítulo curto; a resposta real traz os 10.)

### Campos de `chapters[]`

| Campo | Tipo | Descrição |
|---|---|---|
| `slug` | string | Identificador estável do capítulo (usado pelo front como id e prefixo de âncoras) |
| `label` | string | Rótulo curto da aba/índice |
| `title` | string | Título do capítulo (o H1 do doc de origem) |
| `markdown` | string | Corpo em Markdown, **sem H1** (já extraído para `title`) e **sem o rodapé de desenvolvedores** (`---` + `> **🔧 Para Desenvolvedores**`) |
| `comingSoon` | boolean | `true` quando o capítulo é um adiantamento ("Em breve"); hoje só `sistema-nen` |

### Ordem e slugs

A **ordem do array é significativa** (é a ordem de exibição). Hoje:
`criacao`, `classes`, `aptidoes`, `atributos`, `pericias`, `proficiencias`,
`experiencia`, `status`, `sistema-nen`, `glossario`.

### Notas sobre o Markdown

- **Links entre docs** são relativos aos arquivos de origem (ex.: `../classes.md`,
  `./atributos.md#distribuicao-de-pontos`). O front os converte em navegação entre
  capítulos.
  Os links entre capítulos podem vir com `./`, `../` ou sem prefixo nenhum (ex.:
  `pericias.md` em `proficiencias.md`). O front resolve pelo nome do arquivo sem
  `.md`: se esse nome for um dos 10 slugs de capítulo, o link vira navegação entre
  capítulos; se não for, é um link comum e abre em nova aba.
- **Âncoras** (`#progressao-em-cascata`) são o texto do título sem acento, em minúsculas,
  com hífens (o `slugify` do front). Link para outra seção do mesmo capítulo também usa o
  nome do arquivo (`./atributos.md#formulas`), nunca só `#âncora`.
- **`(em breve)`**: o texto literal `(em breve)` marca algo que existe no back mas ainda não
  aparece para os jogadores; o front o troca por um selo.
- **Fórmulas**: blocos ` ```math ` trazem LaTeX (ex.: a curva de XP em `experiencia`, dentro
  de uma citação `>`); o front os desenha como fórmula.
- **Diagramas**: blocos de código cuja info string começa com `diagram`:

  ````markdown
  ```diagram physical paint=all
  (fallback ASCII)
  ```
  ````

  - Nomes: `physical`, `mental`, `nen-principles`, `attribute-node`.
  - `paint`: `all`; ou lista separada por vírgula de nós (siglas da ficha: `RES`, `Str`,
    `RSL`, `Ken`...) e segmentos `A-B` entre nós adjacentes (`AGI-Str`). O que não é listado
    fica aceso em branco. Sem `paint`, o padrão é `all`.
  - O corpo do bloco é um fallback ASCII; nome desconhecido ou `paint` malformado deve
    renderizar esse bloco como código simples, sem quebrar.
  - Gramática completa: `docs/superpowers/specs/2026-11-11-sheet-rules-panel-design.md` §5.3.

### Respostas

| Status | Situação |
|---|---|
| 200 | Capítulos devolvidos |
| 500 | Falha interna (os capítulos são validados no boot, então na prática não ocorre) |

### Exemplo

```bash
curl -s localhost:5000/rules/character-sheet | jq '.chapters | map(.slug)'
```
