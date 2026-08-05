# Memória de exploração por parede + fog de nível único

**Status:** aprovado, pronto para implementação
**Escopo:** backend (modelo de memória + contrato) e frontend (renderização)
**Branch:** `feat/tactical-map-fog-polygon-10e` (nos dois repos — mesma branch/PR da 10-E)

**Supersede parcialmente** `2026-08-04-tactical-map-fog-10e-design.md`: a camada de fog
"lembrado" (alpha 0.5 por célula) descrita lá é **removida**. A borda de luz poligonal
por máscara invertida, descrita na mesma spec, **permanece** e é reaproveitada aqui.

---

## 1. Por que mudar algo que acabou de funcionar

A 10-E entregou a borda de luz lisa. Mas ela deixou dois níveis de fog: a área
lembrada ficava a 0.5, alinhada ao grid. Na tela, o contraste entre uma fronteira
poligonal lisa e um bloco de memória quadriculado ficou visualmente ruim — o
quadriculado chama atenção justamente por estar ao lado de uma borda lisa.

A decisão do dono do produto: **o terreno já visto não precisa estar na memória do
personagem.** O fog não é opaco, então o jogador já percebe o mapa por baixo dele. O
que precisa mesmo persistir é a **estrutura estática**: paredes hoje, decorações na
fase 11. Paredes dificilmente mudam enquanto o personagem não está olhando.

Peças de personagem **não** entram na memória, deliberadamente: personagens se movem o
tempo todo, então lembrar a última posição vista informa mal o jogador. O backend já
trata assim (`FilterMapState` usa `own || IsVisible(...)`, sem consultar memória).

## 2. O defeito estrutural do modelo por célula

Ao remover a célula da renderização, ela perde a última justificativa que tinha. E o
uso que sobrava — decidir se uma parede é lembrada — sempre foi um proxy com perda:

| Defeito | Como acontece hoje |
|---|---|
| **Falso negativo** | O personagem vê a parede (passa em `wallInLOS`), mas o centro da célula nunca entrou no polígono. A parede nunca é gravada, e some no instante em que o jogador se afasta. |
| **Falso positivo** | Um trecho de parede ocluído está numa célula cujo centro foi iluminado. `wallInExploredCells` a considera lembrada, e o jogador "lembra" de uma parede que nunca viu. |

Os dois têm a mesma raiz: **a célula é um proxy com perda para "eu vi esta parede"**.
Calibrar o teste de célula (marcar por área em vez de centro) reduz um e piora o outro.
Não há ajuste que resolva os dois enquanto o registro for por célula e a exibição for
por parede.

### Alternativas consideradas

| Opção | Resolve | Custo | Decisão |
|---|---|---|---|
| Calibrar o teste de célula | Nenhum dos dois de fato | ~1-2h | Rejeitada |
| **Memória por ID de parede** | Falso negativo e falso positivo | ~3-4h | **Escolhida** |
| Memória por trecho (bitmask de samples) | + a extensão da parede | ~1-1,5 dia | Adiada — compõe sem retrabalho |
| Memória geométrica (união de polígonos) | Tudo, exatamente | Vários dias | Adiada |

A escolha é oportunista e a janela é curta: **`player_fog_states` nunca foi escrita**.
`internal/gateway/pg/fog/fog_state_repository.go` tem os três métodos com `TODO`
retornando `nil` — a memória vive só em RAM, por sessão. Trocar o modelo agora custa
**zero migração de dados**. Depois que houver dado real, o mesmo trabalho vira migração.

## 3. O modelo escolhido

Registrar o objeto observado, não a região observada.

```go
type FeatureKind string
const FeatureWall FeatureKind = "wall"   // "decoration" entra na fase 11

type FeatureRef struct { Kind FeatureKind; ID string }

type PlayerMemory struct {
    PlayerID, MatchID, MapID uuid.UUID
    Seen      map[FeatureRef]struct{}
    UpdatedAt time.Time
}
```

`FeatureKind` é a única concessão ao futuro. Ela custa uma coluna e faz a fase 11 entrar
sem tocar em schema nem em assinatura. É também **onde** a memória de estado encaixa
depois (ver §6).

### O invariante que impede o falso negativo de voltar

> **O predicado que revela é o predicado que grava.**

`SeenWalls` e `FilterMapState` usam a **mesma** função não exportada `wallInLOS`, no
mesmo pacote. Não existe caminho em que uma parede apareça para o jogador e não seja
gravada, porque é literalmente o mesmo teste. Se alguém otimizar um lado e esquecer o
outro, `TestSeenWallsAgreesWithFilterMapState` quebra.

Isso não é detalhe de implementação: é o motivo de `SeenWalls` morar em
`internal/domain/match/service/` e não numa camada acima.

### O que sobra de imperfeição

Memória de parede **inteira**: quem viu uma ponta de um vão longo lembra do vão todo. Na
tela, o passe esmaecido desenha a parede completa. É consciente e aceito — a Opção 3
resolve depois adicionando um campo de máscara ao registro, sem reescrever nada.

## 4. Renderização

Duas regras, uma para cada camada, com responsabilidades separadas:

- **Backend decide *quais* paredes o jogador pode saber que existem.** Autorização.
- **O polígono de LOS decide *quão claro* cada pixel é.** Apresentação.

Consequência direta e importante: **o cliente não precisa de nenhum dado de memória.**
Ele desenha todas as paredes que recebeu e deixa o stencil decidir o brilho.

```
Bg → Grid → decorações → Peças → Fog → Paredes
                                  ↑      ↑
                       nível único │      │ dois passes mascarados
```

| Camada | Como |
|---|---|
| Fog | Um retângulo único (tabuleiro + padding) a `alpha 0.92`, menos o polígono de LOS via máscara **invertida**. Sem célula, sem grid, sem blending. |
| Paredes — vistas agora | Máscara **normal** com o mesmo polígono: só os trechos dentro da LOS, alpha cheio. |
| Paredes — lembradas | Máscara **invertida**, `alpha 0.5`: só os trechos fora da LOS, esmaecidos. |

Paredes passam a ficar **acima** do fog. Isso é seguro porque o backend só envia parede
que o jogador tem direito de conhecer — parede atrás de outra, nunca vista, nunca chega
ao cliente. Isso inverte a decisão tomada na 10-D, onde o objetivo era o oposto
(esconder as paredes de memória); a premissa mudou por decisão do dono do produto.

Como o recorte é o mesmo polígono liso em ambos os passes, **não há pixelização** e o
resultado é per-pixel: uma parede metade dentro e metade fora da LOS sai corretamente
dividida — metade nítida, metade esmaecida. Isso resolve de forma definitiva a queixa
das "paredes cobertas pela metade", que a 10-E tinha classificado como fase futura.

`alpha 0.5` na parede lembrada foi escolhido por legibilidade (0.92 deixava a parede
quase invisível). É constante de uma linha, ajustável olhando a tela.

## 5. Contrato

`explored_cells` (em `map_full_state`) e `explored_delta` (em `visibility_updated`)
**saem do contrato**. Nenhum cliente precisa deles depois desta mudança.

`fog_mode` **permanece** e continua significativo: `live` = sem memória nenhuma
(paredes somem ao sair da LOS); `explored` = memória de estrutura ativa.

## 6. Fora de escopo, com lugar reservado

**Memória de estado.** Memória de *existência* ("sei que tem uma parede ali") é
diferente de memória de *estado* ("ela estava fechada quando vi"). Hoje
`wall_state_changed` é broadcast para todos, então um jogador que lembra de uma porta
descobre que ela abriu enquanto estava de costas. É vazamento pela mesma raiz. O modelo
acima reserva o lugar: o valor do mapa `Seen` deixa de ser `struct{}` e passa a carregar
o snapshot. Não implementar agora.

**Decorações (fase 11).** Entram como `FeatureKind("decoration")` no backend e dentro do
mesmo `<LosSplit>` no frontend. Nenhum conceito novo.

**Memória por trecho de parede.** Ver §3.

## 7. Critério de aceite visual

- [ ] Não existe mais nenhum bloco de fog quadriculado em lugar nenhum da tela.
- [ ] A borda da área iluminada continua lisa e poligonal.
- [ ] Paredes dentro da LOS aparecem nítidas e **inteiras** (não cortadas ao meio).
- [ ] Paredes já vistas e fora da LOS aparecem esmaecidas, legíveis, por cima do fog.
- [ ] Uma parede que o personagem nunca viu **não aparece de forma alguma**.
- [ ] Andar até uma parede e voltar: ela permanece esmaecida (não some) — o falso
      negativo do modelo por célula.
- [ ] O mestre continua sem fog e com todas as paredes nítidas.
- [ ] Clicar numa porta continua abrindo/fechando **uma** vez (guarda contra o listener
      duplicado — ver o plano do frontend).
