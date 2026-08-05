# Fase 10-E — Fog de polígono (borda de luz suave)

> **Parcialmente superseded por** `2026-08-05-tactical-map-wall-memory-design.md`.
> A borda de luz poligonal por máscara invertida descrita aqui **permanece válida**.
> A camada de fog "lembrado" (alpha 0.5 por célula) foi **removida**: na tela, o bloco
> quadriculado ao lado de uma fronteira lisa ficou visualmente ruim, e a memória passou
> a registrar estrutura estática (paredes por id) em vez de região do mapa.

**Status:** aprovado, pronto para implementação
**Escopo:** frontend apenas (o backend entra só com a correção de lint da 10-D)
**Branch:** `feat/tactical-map-fog-polygon-10e` (nos dois repos)
**Antecessor:** Fase 10-D (fog de guerra funcional, PRs #53 back / #51 front)

---

## 1. Problema

O fog do jogador é **quadriculado**: a fronteira entre o que ele vê e o que está
escondido segue as bordas das células do grid. O Foundry e VTTs equivalentes mostram
uma **borda de luz poligonal** — raios saindo das quinas das paredes, com arestas
retas em ângulos arbitrários.

## 2. A geometria já existe

Esta é a informação central desta fase, e ela contraria a intuição de que "fog por
polígono" é um trabalho grande.

O backend **já** calcula um polígono de visibilidade real. `ComputeVisibilityPolygon`
(`internal/domain/match/service/visibility.go`) faz varredura angular lançando raios
para as quinas de cada parede (com ±epsilon para dobrar cantos) e para os limites do
tabuleiro (`BoundaryLOSWalls`, adicionado na 10-D). Na partida real de teste o polígono
sai com **98 vértices**, em coordenadas de mundo, e já viaja no `map_full_state` no
campo `visible_polygons`.

Ou seja: **a geometria correta já chega no cliente**. O quadriculado é uma decisão de
renderização tomada na 10-D, não uma limitação do modelo.

### Por que a 10-D quadriculou

Na 10-D o `FogLayer` desenhava a área iluminada com `blendMode="erase"` dentro de um
`isRenderGroup`. Isso não funciona: `isRenderGroup` **não** cria um render target
isolado, então o erase perfurava o framebuffer principal até a cor de limpeza do canvas
— a área iluminada saía preta sólida em vez de revelar o mapa.

A correção foi abandonar blending e desenhar **regiões disjuntas**: cada célula é
pintada no máximo uma vez, com o alpha do seu nível. Regiões disjuntas exigem
classificar por célula, e classificar por célula produz borda quadriculada.

## 3. Solução: máscara invertida (stencil)

Pixi v8 expõe `container.setMask({ mask, inverse: true })`. Com `inverse: true`, o
conteúdo do container é renderizado **onde a máscara NÃO está**. Aplicando o polígono
de visibilidade como máscara invertida sobre a camada de fog, o fog some exatamente
dentro do polígono — com a borda real, sem blending e sem classificar célula alguma.

### O que foi verificado na fonte do Pixi (não inferido)

Os fatos abaixo foram lidos em `node_modules/pixi.js/lib/` na versão instalada
(`pixi.js ^8.18.1`). A 10-D custou um mês por causa de inferência sobre composição no
Pixi; esta fase só avança sobre comportamento verificado.

| Fato | Onde | Consequência |
|---|---|---|
| `MaskOptions.inverse` é API pública, documentada em `setMask` | `scene/container/container-mixins/effectsMixin.d.ts` | não é gambiarra nem API interna |
| `StencilMaskPipe` lê `_container._maskOptions.inverse` e emite `INVERSE_MASK_ACTIVE` | `rendering/mask/stencil/StencilMaskPipe.mjs` | inverse é honrado de fato pelo pipeline de stencil |
| `INVERSE_MASK_ACTIVE` = `compare: "not-equal"` sobre o índice da pilha de máscara | `rendering/renderers/gpu/state/GpuStencilModesToPixi.mjs` | renderiza fora da máscara — exatamente o que queremos |
| `RENDERING_MASK_ADD` = `compare: "equal"` + `increment-clamp` | idem | **polígonos sobrepostos na máscara se unem, não se cancelam** (não é XOR) |
| `StencilMaskPipe.push` faz `maskContainer.includeInBuild = true` → coleta → `= false` | `StencilMaskPipe.mjs` | a máscara pode ser filha do container mascarado sem ser desenhada como conteúdo |
| `_maskOptions` é objeto **compartilhado no protótipo** do mixin; `setMask` cria cópia própria | `effectsMixin.mjs` | mutar `container._maskOptions.inverse` direto vaza para todo container que nunca chamou `setMask` |

O último item é uma armadilha real: a única forma correta de ligar o inverse é
`setMask({ mask, inverse: true })`.

### Por que máscara e não `Graphics.cut()`

`cut()` também existe e funciona, mas tem duas propriedades que o desqualificam aqui,
ambas confirmadas na fonte:

1. `buildContextBatches.mjs` triangula furos com earcut. Earcut não define
   comportamento para **furos sobrepostos**. Um jogador com duas peças próximas gera
   dois polígonos de visibilidade que se sobrepõem — caso comum, resultado quebrado.
2. `GraphicsContext.cut()` percorre as **duas** últimas instruções e, quando a última
   já tem furo, chama `addPath` **sem `break`** — então o segundo furo em diante
   também é anexado à instrução anterior. Com mais de um `fill()` no mesmo `Graphics`,
   os furos vazam para a camada de baixo.

A máscara invertida não tem nenhum dos dois problemas: sobreposição vira união pelo
stencil, e não há interação entre camadas de fill.

## 4. Arquitetura resultante

```
<pixiContainer label="fog-layer">        ← recebe setMask({ mask, inverse: true })
   <pixiGraphics draw={drawFogTiers} />  ← níveis de fog, por célula, disjuntos
   <pixiGraphics draw={drawLosMask} />   ← polígonos de LOS (a máscara)
</pixiContainer>
```

A máscara é **filha** do container mascarado. O `StencilMaskPipe` marca
`includeInBuild = false` depois de coletá-la, então ela não é desenhada como conteúdo.

### Os níveis de fog ficam mais simples

Hoje `fogTiers` roda point-in-polygon em `cols × rows × vértices` (35×35×98 ≈ 120 mil
operações a cada mudança de fog) só para excluir as células visíveis. Com a máscara,
excluir a área visível deixa de ser trabalho do classificador: a máscara faz isso na
GPU, com a geometria certa.

`fogTiers` passa a responder apenas "esta célula é lembrada ou desconhecida?":

| Nível | Alpha | Origem |
|---|---|---|
| desconhecido | 0.92 | célula fora de `exploredCells` (ou modo `live`) |
| lembrado | 0.50 | célula em `exploredCells`, modo `explored` |
| visível | 0 | **não é pintado** — a máscara invertida remove |

O resultado visual é exatamente o do Foundry: **a fronteira da visão atual é lisa e
poligonal**; a área de memória permanece quadriculada, porque memória é acumulada por
célula no backend (`player_fog_states`).

## 5. Fora de escopo (deliberado)

- **Área explorada lisa.** Exigiria acumular histórico como geometria ou textura em vez
  de células, mudando o formato de persistência. 1–2 dias. Fase futura.
- **Paredes cobertas pela metade.** Uma parede que bloqueia a visão fica exatamente
  sobre a borda do polígono, então metade da sua espessura cai do lado escuro. A fase
  10-E não muda isso, mas **facilita** a correção futura: a borda do fog passa a ser a
  própria geometria de LOS, então dilatar o polígono por meia espessura de parede
  resolve o caso de forma uniforme.
- **Divergência `applyTransform` Go × TS.** O Go faz squash antes da rotação e ignora
  `originX`/`originY`; o TS rotaciona em torno do centro do grid, depois faz squash, e
  soma a origem. As duas são idênticas para `rotation: 0`, `skewRatio: 1`,
  `origin: 0` — o caso de todos os mapas atuais. Os polígonos já são comparados contra
  coordenadas de mundo TS hoje (em `fogTiers`), então desenhá-los cru **não introduz**
  regressão. Convergir as duas implementações é trabalho próprio, não desta fase.

## 6. Como validar

Três níveis, todos obrigatórios antes de declarar a fase concluída.

1. **Unit, contra payload real.** As funções de desenho são puras e recebem um alvo
   estrutural, não um `Graphics` do Pixi. Os testes gravam as chamadas e verificam
   geometria e alphas usando `__tests__/fixtures/realFogPayload.json` — exportado da
   partida real pelo smoke test do backend. Guarda de regressão explícita: **todos os
   vértices do polígono ficam dentro dos limites do tabuleiro** (foi o bug do polígono
   de 7505×9734 num tabuleiro de 3360×3360, corrigido na 10-D por `BoundaryLOSWalls`).
2. **Falha ruidosa.** Se `Container.setMask` não existir no runtime, o `FogLayer`
   lança. O modo de falha silencioso — fog cobrindo o mapa inteiro sem erro — foi
   exatamente o que escondeu os bugs da 10-D por semanas.
3. **Verificação visual** (obrigatória pelo `CLAUDE.md` do projeto). Ver seção 7.

## 7. Critério de aceite visual

Com `./dev-checkout.sh feat/tactical-map-fog-polygon-10e` e duas janelas (mestre e
jogador) na mesma partida:

- [ ] A área iluminada do jogador é um **leque de arestas retas** saindo das quinas das
      paredes, em ângulos arbitrários — não segue as bordas do grid.
- [ ] Não existe retângulo preto sólido em lugar nenhum (a regressão do `erase`).
- [ ] As paredes e portas continuam visíveis para o jogador (ganho da 10-D preservado).
- [ ] Em modo `explored`, a área lembrada continua cinza médio e quadriculada — e a
      transição para a área iluminada corta as células ao meio, sem degrau de grid.
- [ ] O mestre continua sem fog nenhum.
- [ ] Mover uma peça reacende o leque na posição nova sem piscar preto.
