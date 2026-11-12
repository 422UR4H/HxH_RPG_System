# Painel de regras da ficha — design

**Data:** 2026-11-11 · **Status:** aprovado no brainstorm; revisão de quem implementa feita
**Repos:** `System_X_System` (back, branch `feat/sheet-rules`) e `System_X_System_React`
(front, branch `feat/sheet-rules`). As duas branches ficam em `.worktrees/sheet-rules` de cada repo.

## 1. Intenção

Os testadores se perdem ao criar a ficha: nada no site explica atributos, perícias, distribuição
de pontos ou classes. As regras já existem como docs de jogo para jogadores, em
`System_X_System/docs/game/**`. Esta feature mostra **essas docs** dentro da ficha. Assim, editar uma
doc (e publicar o back) muda o que o jogador lê. Nenhum texto de regra é duplicado no front.

O dono quer transformar `docs/game` num **livro** de regras. Por isso as docs continuam com cara de
livro: as ilustrações ricas aparecem no site, e o mesmo Markdown segue legível no GitHub ou impresso.

## 2. Objetivos

1. Um único **botão "?"** no canto superior direito da coluna ATRIBUTOS abre um **painel de regras**
   com *todas* as regras de que o jogador precisa para a ficha. Ele aparece nas quatro telas de
   ficha: ver (`/charactersheet/:id`), criar (`/charactersheet/new`), editar
   (`/charactersheet/:id/edit`) e criar NPC (`/campaigns/:campaignId/npcs/new`). As quatro usam o
   `CharacterSheetTemplate`.
2. O conteúdo do painel são as docs de jogo, servidas pelo back e renderizadas como Markdown com os
   estilos desenhados no Claude Design.
3. As ilustrações são os **diagramas do próprio site** (losango físico, losango mental, princípios
   Nen), desenhados com o novo **efeito de luz**. Elas entram nas docs por marcadores.
4. Abrir as regras nunca perde o que o jogador preencheu na ficha.
5. Corrigir o bug do Markdown no background do personagem (`DescriptionMarkdown`): títulos, negrito
   etc. aparecem como texto comum.

## 3. Fora do escopo

- A variante A do design (página de regras inteira) **não** é construída. Ela fica guardada para
  uma futura página de regras do site.
- Nada da classe escolhida aparece no painel. O texto é o mesmo para todos.
- A luz **não** vai ainda para os diagramas da própria ficha. Ela nasce reutilizável, mas por
  enquanto só o painel a usa.
- Nenhum outro botão "?" (em PERÍCIAS, no campo Classe…). O de ATRIBUTOS abre tudo.
- Não se editam regras pelo site. Não há busca.

## 4. O que seria fracasso, mesmo "funcionando"

- **Perder o que o jogador preencheu na ficha** ao abrir, ao fechar ou ao apertar o "voltar" do
  navegador.
- **Cores novas na paleta.** Só valem as entradas existentes de `palette`. Apelidos semânticos em
  `colors` apontando para cores existentes são permitidos (ver `src/styles/CLAUDE.md`).
- **Mexer no reset global de CSS** (`src/styles/ResetStyle.ts`). Ele afeta o site inteiro.
- Texto de regra fixo no front, exceto textos curtos de interface (rótulos de botão, dicas,
  mensagens de carregando e de erro).
- Conteúdo de desenvolvedor (caminhos de código, links para `docs/dev`) visível para o jogador.
- Uma luz que pisca forte. O movimento tem que ser sutil (ver §8).

## 5. Conteúdo: quais docs e em que ordem

O painel é **uma rolagem contínua** de capítulos, um capítulo por doc, nesta ordem fixa:

| # | Rótulo do chip | Doc (`docs/game/…`) | Notas |
|---|---|---|---|
| 1 | Criação | `ficha-de-personagem/criacao.md` | **Doc nova**, ver §5.1 |
| 2 | Classes | `classes.md` | ajustes em §5.2 |
| 3 | Habilidades | `ficha-de-personagem/habilidades.md` | |
| 4 | Atributos | `ficha-de-personagem/atributos.md` | marcadores de diagrama, §5.3 |
| 5 | Perícias | `ficha-de-personagem/pericias.md` | |
| 6 | Proficiências | `ficha-de-personagem/proficiencias.md` | |
| 7 | Experiência | `ficha-de-personagem/experiencia.md` | |
| 8 | Vida e estamina | `ficha-de-personagem/status.md` | |
| 9 | Nen | `ficha-de-personagem/sistema-nen.md` | selo **"Em breve"**, marcador do diagrama Nen |
| 10 | Glossário | `glossario.md` | coluna Sigla, §5.2 |

`armas.md`, `dados.md`, as docs de combate e afins ficam de fora: não são regras da ficha.

As docs já são escritas para o jogador. Só o **rodapé de desenvolvedor** precisa sumir: o `---`
seguido da citação `> **🔧 Para Desenvolvedores**` que fecha a doc
(`.github/instructions/docs-workflow.instructions.md`). O rodapé é **opcional**: `glossario.md` e a
nova `criacao.md` não têm. Ele só é cortado quando existe.

### 5.1 Doc nova `criacao.md`

Um capítulo curto: cinco passos, na ordem da ficha. Cada passo leva ao capítulo que o explica.
1. Preencha o perfil (apelido, nome, aniversário…).
2. Escolha a classe → Classes.
3. Distribua os pontos de atributos físicos e mentais até **Pontos para Distribuir** zerar → Atributos.
4. Se a classe oferecer, escolha as perícias e armas que ganham experiência extra → Perícias, Proficiências.
5. Aperte **Criar Ficha**.

A base é o texto do Claude Design (`ConteudoRegras.dc.html`, seção `#resumo`).

### 5.2 Ajustes nas docs existentes (repo do back)

- **`classes.md`**
  - Apresentar a classe como **aquilo a que o personagem dedicou anos da vida para se tornar, até o
    momento da criação**.
  - Dizer que **hoje há 12 classes disponíveis**: Espadachim, Samurai, Ninja, Ladino, Netrunner,
    Pirata, Mercenário, Terrorista, Monge, Militar, Hunter e Mestre de Armas. São as que
    `CharacterClassFactory.Build()` registra, ou seja, a lista que `GET /classes` devolve aos
    jogadores.
  - Dizer que há mais a caminho, sem dar ao jogador a impressão de que falta algo.
- **`glossario.md`**
  - Coluna **Sigla** nas tabelas de atributos, com as siglas que a ficha mostra: RES, AGI, FLX,
    SEN, Str, Cel, Dex, Con, RSL, ADP, JUD, CRE.
  - Atleta, Tribal, Experimento e Circo marcadas como "(em breve)" na tabela de classes.
- **`habilidades.md`, `experiencia.md`**
  - Manter as tabelas de "Velocidade de Progressão" e a curiosidade da curva de XP.
  - A frase que as explica vira uma ou duas linhas curtas e simples: número maior = mais XP para
    subir de nível. Nada além disso.
- **`atributos.md`:** uma seção curta "Como ler um atributo" (o nó mostra *nível · pontos · poder*)
  com o marcador do nó, mais os marcadores de diagrama do §5.3.
- **`sistema-nen.md`:** o marcador do diagrama de princípios Nen.
- **Novo `internal/domain/entity/character_class/CLAUDE.md`**
  - Ao adicionar (ou tirar) uma classe de `Build()`, atualizar a contagem e a lista em
    `docs/game/classes.md` e as marcas "(em breve)" em `docs/game/glossario.md`.
  - Dizer o porquê: essas docs aparecem para o jogador no painel de regras da ficha.
- **`docs/documentation-map.yaml`:** registrar `criacao.md` e o endpoint de regras.

A Precisão (glossário e `pericias.md`) já foi corrigida nesta branch. Não mexer de novo.

### 5.3 Marcadores de diagrama no Markdown

O diagrama entra como um **bloco de código cercado** cuja linha de abertura começa com `diagram`.
O corpo é um desenho em texto, que o GitHub e a impressão mostram como está; o site troca o bloco
inteiro pelo diagrama aceso.

````markdown
```diagram physical paint=all
            Resistência
   Força                Constituição
Agilidade                     Sentido
   Celeridade             Destreza
           Flexibilidade
```
````

- **Nomes:** `physical`, `mental`, `nen-principles`, `attribute-node`.
- **`paint`:**
  - `all`; ou
  - uma lista separada por vírgula de nós (siglas da ficha: `RES`, `Str`, `RSL`, `Ken`, …) e de
    trechos escritos `A-B` entre nós vizinhos (`AGI-Str`).
  - O que não estiver na lista acende **branco**.
  - Sem `paint`, vale `all`.
- **Marcadores a colocar:**

  | Onde | Marcador |
  |---|---|
  | `atributos.md`, losango físico (substitui o desenho em texto atual) | `physical paint=all` |
  | `atributos.md`, explicação dos médios | `physical paint=RES,AGI,Str,AGI-Str,Str-RES` (Força = Resistência + Agilidade) |
  | `atributos.md`, frase "um ponto em Agilidade sobe Força e Celeridade" | `physical paint=AGI,Str,Cel,AGI-Str,Cel-AGI` |
  | `atributos.md`, seção dos mentais | `mental paint=all` |
  | `atributos.md`, "Como ler um atributo" | `attribute-node` |
  | `sistema-nen.md` | `nen-principles paint=all` |

- Nome desconhecido ou `paint` mal escrito não pode quebrar a tela. Mostrar o bloco como código
  comum e registrar um aviso no modo de desenvolvimento.

### 5.4 Títulos dos capítulos e links

- O `# H1` da doc vira o título do capítulo. Dentro do painel, os níveis de título descem um degrau
  para a hierarquia da página continuar válida.
- Links entre docs de jogo (`./atributos.md`, `../classes.md`, com ou sem `#âncora`) rolam dentro do
  painel.
- Os ids dos títulos são `<slug>--<slug-do-título>` (o do capítulo é só `<slug>`), para as âncoras
  nunca colidirem entre capítulos. Um link para `atributos.md#distribuicao-de-pontos` vira
  `atributos--distribuicao-de-pontos`.
- Qualquer outro link que sobrar depois do corte abre em nova aba.

## 6. Back: servindo as docs

- **Endpoint:** `GET /rules/character-sheet`, **público** (sem login).
  - Regra não é segredo, e a futura `RulesSidebar` de campanha pode reaproveitá-lo.
  - Registrar antes de `api.UseMiddleware(authMiddleware)`, como as rotas de auth e health.
  - Resposta em camelCase:

    ```json
    { "chapters": [ { "slug": "criacao", "label": "Criação", "title": "Criando seu personagem", "markdown": "…", "comingSoon": false } ] }
    ```

    - A ordem é a do §5.
    - `slug` é o nome do arquivo sem `.md`.
    - `label` é o rótulo curto do chip, da tabela do §5, guardado na lista de capítulos do back.
      Assim o front não fixa nenhum vocabulário de regra.
    - `title` vem do H1.
    - `markdown` é o corpo **sem o H1 e sem o rodapé de desenvolvedor**.
- **Embutir:** as docs são embutidas no binário com `go:embed`. Assim o texto servido é sempre o da
  versão do código publicada.
  - `go:embed` não alcança pastas acima dele, então o arquivo Go que embute tem que ficar em
    `docs/game` ou acima.
  - `docs/game/` tem que continuar só com texto de livro (`docs-workflow.instructions.md`), então
    **nenhum `.go` lá dentro**.
  - Usar um pacote pequeno na raiz do módulo que embuta `docs/game/*.md` e
    `docs/game/ficha-de-personagem/*.md`. Nome e local exatos ficam a critério de quem implementa,
    desde que `docs/game` continue limpo.
  - **O `.dockerignore` hoje exclui `docs` e `*.md`.** É preciso reincluir `docs/game` (e seus
    `.md`), senão o build Docker do Render quebra. Provar com um `docker build` local.
- **Lista de capítulos:** uma lista ordenada no código Go (slug, label, arquivo, comingSoon). Um
  arquivo faltando tem que quebrar o build ou um teste.
- **Corte do rodapé:** função pura, com testes de unidade.
  - Um teste guardião garante que nenhum capítulo servido contém `internal/`, `docs/dev` ou
    `Para Desenvolvedores`.
- **Contrato:** `docs/dev/api/rules.md` (requisição, resposta, erros, auth, exemplo), registrado no
  `documentation-map.yaml`, seguindo `.claude/rules/api-contract-author.md`. O back faz merge
  primeiro: o contrato é a passagem de bastão para o front.
- **Cache:** o conteúdo só muda no deploy. Um `Cache-Control` com max-age modesto basta (a critério
  de quem implementa).

## 7. Front: o painel (variante B do Claude Design)

Os arquivos de referência foram copiados do canvas do Claude Design para
`System_X_System_React/docs/audits/2026-11-11-briefing-regras-da-ficha/claude-design/`. Canvas ao
vivo: https://claude.ai/artifact/Cz8pspo4VQRtrtbtMvRK3R. A variante escolhida e aprovada pelo
designer é a **B**: `B1`–`B4`, `BotaoAjuda`, `EstilosMarkdown`, `shared/regras.css`, mais as notas
verdes do `canvas.json`. As imagens `/_blob/…` correspondem a assets que o front já tem:
`questionmark-sm.svg`, o X vermelho de fechar dos painéis de partida e campanha, e a setinha.

**Nunca copiar as cores do Claude Design em hex.** Cada uma vira um token existente.

### 7.1 O botão "?"

- **Asset:** `questionmark-sm.svg`. 60×60 no desktop, 52×52 no celular; o alvo é o próprio ícone.
- **Posição:** absoluta, no canto superior direito da coluna ATRIBUTOS. Nos modos de distribuir
  (criar, editar, NPC), fica centralizado na altura do bloco "Pontos para Distribuir". No modo ver,
  fica no mesmo canto. Não ocupa espaço no fluxo: título e diagramas não se mexem.
- **Acessibilidade:** `aria-label` "Regras de atributos e perícias", `aria-expanded`, `aria-controls`.
- **Dica ao passar o mouse:** "Regras" com o painel fechado, **"Fechar regras" com o painel
  aberto**. Clicar de novo fecha.
- **Foco:** anel de 2px `actionHighlight`, afastado 3px.

### 7.2 Layout do painel

- **A partir de 1024px:** encaixado à direita, largura `clamp(400px, 35vw, 520px)`, altura total,
  fixo enquanto a ficha rola.
  - A ficha ocupa o resto da largura e **continua usável**: dá para pôr pontos com o painel aberto.
  - O contêiner da ficha encolhe e a ficha se reorganiza, sem ser coberta. A tipografia dela já
    escala por container queries (`cqi`). O que no template depender da largura da *janela*, e não
    do contêiner, tem que continuar funcionando na largura reduzida.
  - Conferir em 1024, 1280 e 1440, sem rolagem horizontal.
  - Não é modal (`role="complementary"`).
- **Abaixo de 1024px:** diálogo em tela cheia sobre a ficha montada (`role="dialog"`, `aria-modal`,
  foco preso no diálogo).
- **Fundo:** `space.png`, como a ficha. Cabeçalho de 52px com o título "Regras · criação de
  personagem" (no celular: "Regras") e o X vermelho vazando do cabeçalho, como em partidas e
  campanhas.
- **Chips:** um por capítulo, na ordem do §5, acima do conteúdo. O chip ativo acompanha a rolagem
  (IntersectionObserver), e clicar num chip rola até o capítulo.
- **Nen:** o título do capítulo e o chip levam o `ComingSoonBadge` que já existe.
- **Fechar:** X, `Esc`, o "?" de novo, ou o "voltar" do navegador (§7.3). O foco volta para o "?".

### 7.3 Estado, URL e o "voltar"

- **Ao abrir,** entra um item no histórico com um parâmetro na URL, ex.: `?regras=atributos`, o
  slug do capítulo em que abriu. Rolar entre capítulos **nunca reescreve a URL**; só atualiza o
  último capítulo, guardado na memória.
  - O **"voltar" do navegador fecha o painel** e mantém o jogador na ficha.
  - Fechar pelo X, `Esc` ou "?", quando o item do histórico é nosso, chama `history.back()`, para não
    deixar item morto.
  - Uma URL que já chega com `?regras=<slug>` abre o painel naquele capítulo, o que permite mandar o
    link. Fechar um painel aberto assim faz um `replace` que tira o parâmetro, já que não há item
    nosso para voltar.
- **Obrigatório:** mudar esse parâmetro não pode remontar a ficha nem zerar o formulário. Conferir
  nas quatro telas, inclusive em criar com pontos já distribuídos.
- **Abre sozinho no primeiro personagem.**
  - Em `/charactersheet/new`, o painel abre automaticamente em **Criação** quando a lista de fichas
    do jogador (`useCharacterSheets`) **terminou de carregar com sucesso e está vazia**. É o caso
    para onde a home já redireciona (`CharacterSheetsPage`).
  - Acontece uma vez por montagem da página. É uma abertura comum: entra o item no histórico, e o
    "voltar" fecha.
  - **Armadilha:** nunca tratar "ainda não carregou" como vazia. Com a lista carregando,
    recarregando ou com erro, o painel não abre sozinho.
  - A página de NPC nunca abre sozinha.
- **Onde o "?" abre:**
  - Em **Atributos**, onde o botão fica.
  - Na mesma visita à página, reabrir volta ao último capítulo mostrado. Esse estado fica só na
    memória; **nada é guardado no navegador**.
  - O link direto `?regras=<slug>` vence os dois.

### 7.4 Carregamento

- **Busca:** um hook do React Query para `GET /rules/character-sheet`, com chave sem token (é
  público), `staleTime: Infinity` na sessão e `retry: 1`. Pré-carregar ao passar o mouse ou focar o
  "?" é bem-vindo.
- **Carregando:** esqueleto ou indicador dentro do painel. O painel abre na hora.
- **Erro:** no estilo de erro existente (`errorBox`: caixa escura + borda de 1px + red600), com o
  botão "Tentar de novo".

### 7.5 Renderização do Markdown e a correção do reset

- **Causa do bug do background:** o `ResetStyle.ts` aplica `font: inherit` (e afins) em títulos,
  `strong`, `em`, listas… e o `DescriptionMarkdown` nunca devolve esses estilos.
- **Correção:** **não** mexer no reset. Criar **uma** base de estilo de Markdown que devolve, *só
  dentro do próprio contêiner*, tamanho e peso dos títulos, negrito, itálico, marcadores e
  espaçamento de listas, e o estilo das tabelas. Ela é usada no `DescriptionMarkdown` (o background)
  e no painel de regras. Nada fora desses contêineres muda.
- **Estilos do painel sobre essa base,** conforme `EstilosMarkdown.dc.html` / `regras.css`:
  - hierarquia h2/h3;
  - linhas de 60–75 caracteres;
  - `code` inline como chip de "sigla", **só no painel** (o background mantém o estilo base);
  - **citação de fórmula** (citação cujo conteúdo é só itálico, como as docs escrevem as fórmulas);
  - **caixa de nota** (citação que começa com um rótulo em negrito, como `**Nota:**`, ou com um
    título "Curiosidade");
  - tabelas que nunca rolam na horizontal e viram linhas rótulo/valor em telas estreitas.

  Quem implementa escolhe a regra para distinguir fórmula de nota e a documenta no componente.
- Os marcadores do §5.3 entram por um renderizador de `code` para a linha de abertura `diagram`.

### 7.6 Acessibilidade e responsividade

- Contraste de 4,5:1 no texto.
- Alvos de toque de 44px.
- Sem rolagem horizontal da página em 390px.
- Títulos numa hierarquia válida.
- Diagramas com `role="img"` e um `aria-label` em português que diz o que está em destaque (ex.:
  "Força acende junto com Resistência e Agilidade").

## 8. A luz — o módulo de diagramas

Referência aprovada, para abrir no navegador:
`System_X_System_React/docs/audits/2026-11-11-briefing-regras-da-ficha/brilho-referencia.html`
(protótipo v7). A implementação tem que ficar igual a ela. As regras que a definem:

1. **A luz é uma só.** Cada camada de brilho é **um** grupo SVG desfocado com *todas* as peças do
   diagrama (linhas, bolinhas, arcos, setas). A opacidade vai no grupo, nunca em cada peça, para
   bolinhas e linhas se fundirem sem emendas. A referência usa quatro camadas (desfoque
   55/30/18/8, largura de linha 60/40/18/9, raio da bolinha ×3,4/2,4/1,6/1,35, opacidades
   ~,3/,6/,8/,9).
2. **Filamento branco contínuo.** A ordem do desenho é:
   - o corpo da bolinha: degradê radial com centro quase branco, depois a cor, depois sumindo até
     transparente na borda;
   - depois os fios brancos das linhas (4px, com um brilho branco suave de 7px por baixo), que
     *atravessam* a bolinha até o centro;
   - depois um miolo branco de borda suave (≈0,62 do raio).

   O branco da linha e o branco da bolinha formam um único filamento, sem quebra.
3. **Bolinhas pelo tamanho.** As das arestas (médios) brilham menos que as dos vértices, na
   proporção do raio.
4. **Pintado = cor; o resto = branco.**
   - O que está pintado usa a cor do diagrama; o que não está acende **branco** (nem apagado, nem
     transparente).
   - Em cada camada, as peças brancas vêm primeiro e as coloridas por cima, um pouco mais largas
     (×1,3 na linha, ×1,2 na bolinha), para a cor vazar sobre o branco.
5. **Regra do vazamento (padrão do módulo).** Quando a pintura chega numa bolinha, a cor segue
   ~32% de cada linha vizinha não pintada e vai sumindo (degradê linear da bolinha para fora). É o
   comportamento padrão sempre que "o progresso atinge uma bolinha".
6. **Cores, só da paleta existente:**

   | Diagrama | Cor |
   |---|---|
   | Físico | `red600` |
   | Mental | `purple500` (o roxo da parede mágica) |
   | Princípios Nen | `blue400` ("mana") |
   | Luz: fios das linhas e miolo das bolinhas | `neutral0` (branco) |
   | Luz: halos não pintados | `neutral300` |

   - Com os mesmos valores de camada, um halo branco parece brilhar muito mais que um colorido.
     Por isso os halos não pintados usam `neutral300`, e o corpo da bolinha branca vai
     `neutral0` → `neutral100` → `neutral200` → transparente. Fios e miolos continuam branco puro.
     A referência (v7) mostra o equilíbrio a seguir.
   - O `red500` pode aparecer dentro do degradê da bolinha vermelha. Criar apelidos semânticos em
   `colors` se precisar; nada novo em `palette`.
7. **Movimento:**
   - Ao aparecer, o diagrama "liga": as linhas se desenham (~0,7s) e as bolinhas acendem em
     sequência.
   - Depois, cada camada respira devagar (4–6s), e cada bolinha muda de tamanho bem pouco, no seu
     ritmo (no máximo ±5%).
   - A cada ~2s, uma bolinha abre o halo devagar (~2,4s, no máximo +13%) e volta.
   - **Sem flashes.**
   - Respeitar `prefers-reduced-motion`: mostrar direto o estado aceso, sem ignição e sem
     respiração.
8. **A geometria** vem dos assets existentes (`physical_lozange.svg`, `mental_lozenge.svg`,
   `nen_principles.svg`), para os diagramas baterem com a ficha. A referência já tem as
   coordenadas; as setas e arcos do Nen são os próprios caminhos do asset.
9. **No painel,** o diagrama aceso substitui a imagem de fundo do diagrama da ficha, sob as mesmas
   siglas, para ficar com a cara da ficha. Os números e o centro da habilidade (ex.:
   "PHYSICAL 3 4.5") ficam **escondidos**; quem explica os números é a figura `attribute-node`.
   - A figura `attribute-node` segue a ideia do `NoAtributo` do Claude Design: o nó mais três
     cartões (Nível · Pontos · Poder).
   - Ela tem que ficar igual ao nó real da ficha (mesmos tamanhos de fonte e os pontos em verde), e
     não à versão distorcida do Claude Design.
   - Reaproveitar a marcação real do nó, as peças de `AttributeMarker` dentro de
     `PhysicalsDiagram.tsx`, extraindo para uma peça compartilhada quando ficar limpo (regra de
     promover em vez de duplicar). Ela recebe números de exemplo fixos.
10. **Reutilizável:** um módulo independente, algo como um `GlowDiagram` que recebe geometria,
    pintura e cor, mais uma função pura `resolvePaint(spec)` com testes de unidade. O dono vai
    ampliá-lo depois (pintura na ficha, progresso).

## 9. Decisões minhas, abertas à revisão do dono

- **Endpoint:** público; caminho `/rules/character-sheet`; o formato de resposta do §6.
- **Marcadores:** blocos de código `diagram` com desenho em texto de reserva; números escondidos
  nos diagramas do painel.
- **Cor do Nen:** `blue400`.
- **Capítulos:** os dez do §5, nessa ordem; o glossário aparece inteiro.
- **Onde o painel abre:**
  - Abre sozinho em Criação para quem não tem ficha, na página de criar, e só depois que se sabe
    que a lista está vazia.
  - O "?" abre em Atributos.
  - O último capítulo é lembrado só durante a visita à página, sem guardar nada no navegador. É a
    sugestão do dono, refinada para evitar a armadilha da lista ainda não carregada.
- **Da revisão de quem implementa:**
  - rodapé opcional;
  - campo `label`;
  - formato dos ids de âncora;
  - `replace` da URL ao fechar um link direto, e nenhuma reescrita da URL ao rolar;
  - a ficha se reorganiza ao lado do painel encaixado;
  - a figura do nó reaproveita a marcação real;
  - o chip de sigla só no painel.
- **Sem Docker:** se não houver `docker build` na máquina, trocar por um teste que prove que os
  arquivos foram embutidos, e dizer isso no PR.

## 10. A critério de quem implementa

- Local do arquivo Go e do pacote que embute as docs.
- Valor do `Cache-Control`.
- Regra que distingue fórmula de nota.
- Nomes de componentes e organização de arquivos, seguindo `src/components/CLAUDE.md` (camadas
  Atomic, regra de promover em vez de duplicar) e `src/features/sheet/CLAUDE.md`.
- Se o estado de aberto/fechado do painel mora num contexto ou no template.
- Visual do carregando.

## 11. Verificação (da seção "Entrega" do CLAUDE.md da raiz)

- **Back:**
  - testes de unidade (corte do rodapé, lista de capítulos, guardião);
  - teste do handler via `humatest`;
  - `curl` no endpoint local;
  - `docker build` local provando que as docs foram embutidas.
- **Front:**
  - testes de:
    - abrir/fechar, "voltar", retorno do foco;
    - regra de abrir sozinho: lista vazia carregada → abre; carregando, recarregando, com erro ou
      com fichas → não abre;
    - resolução da pintura e mapeamento do Markdown;
  - depois, o navegador em 1440 e 390 nas quatro telas, inclusive criar com pontos já distribuídos
    (abrir, ler, fechar, "voltar": nada se perde);
  - o Markdown do background mostrando títulos e negrito;
  - `prefers-reduced-motion` respeitado.
- **PRs:** um por repo, com link cruzado; back primeiro.
