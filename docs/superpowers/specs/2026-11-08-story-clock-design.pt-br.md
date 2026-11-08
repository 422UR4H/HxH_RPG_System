# Relógio da história, fim de combate e encerrar partida — design

> 2026-11-08. Brainstorm com o dono do produto (sessão de 07→08/11). Origem:
> `System_X_System_Project/docs/handoffs/2026-11-07-contador-de-tempo.md` e
> `.../2026-10-26-encerrar-partida.md` (raiz do projeto, não versionada).
> Decisões do dono estão marcadas **(dono)**; o que eu decidi sozinho está marcado **(minha)** —
> ele dormiu confiando que eu levaria até os PRs, então essas são as que ele vai conferir.

## 1. Para que serve

1. **Saber que horas são na história.** Hoje ninguém sabe quanto tempo passou dentro da partida:
   ela guarda só a *data* de início na história (`matches.story_start_at`, `DATE`). O relógio é o
   tempo do jogo, nunca o do mundo real.
2. **Destravar o "encerrar partida"**, que pede o momento exato do fim na história — e que hoje
   não existe em lugar nenhum (nem back, nem front).
3. **Converter combate em tempo.** Cada round disputado vale 1 segundo, e o fechamento do combate
   mostra ao mestre os *hiatos* (pausas sem disputa) para ele somar o tempo que eles duraram.
4. **Base para os status** (sessão paralela): sangramento e outros efeitos "por round" precisam
   responder "quantos rounds equivalentes se passaram" fora do combate. Como 1 round = 1 s, a
   resposta é a diferença em segundos entre dois instantes do relógio.

**Para quem:** o mestre controla; os jogadores só veem — e **toda mudança feita pelo mestre fica
explícita para eles (dono)**, nunca passa batida.

## 2. Vocabulário

| Termo | Significado |
|---|---|
| **Relógio** | O instante atual da história da partida (data + hora até o segundo). Um por partida. |
| **Tempo real (play)** | O relógio correndo junto com o relógio do mundo, para cenas de roleplay. |
| **Limite** (piso) | O instante antes do qual o relógio não pode voltar. |
| **Round contado** | Round que teve ao menos um turno aberto no regime disputado. Vale 1 s. |
| **Hiato** | Round disputado que terminou com a fila vazia, até o primeiro turno do round seguinte. |

## 3. Regras do relógio (back, fonte da verdade)

### 3.1 Começo
- A partida ganha uma **hora de início opcional** na história, ao lado da data (dono). Criar e
  editar partida aceitam esse campo; editar continua recusado depois do início (regra atual).
- Ao iniciar a partida (`start_match`), se a partida tem data **e** hora, o relógio nasce nesse
  instante, **pausado** (dono: "ele sempre começa pausado"). Limite = esse instante.
- Se só existe a data, o relógio nasce **sem hora definida**. O mestre recebe na mesa, logo que
  entra, um diálogo **obrigatório** para informar a hora (a data é a da partida, fixa) (dono).
  Enquanto não houver hora, o servidor recusa com `clock_not_set` os comandos do mestre que mexem
  na linha do tempo: `open_next_action`, `pull_action`, `change_round_mode`, `change_scene`,
  `end_match` e os de relógio (exceto `set_clock_start`). Jogadores continuam podendo declarar, e
  o mestre continua podendo mexer no tabuleiro (master actions não andam no tempo).
  **(minha)** — é a única forma de os limites abaixo valerem desde o primeiro turno.
  `set_clock_start` com a hora já definida → `clock_already_set`. Um mestre que recarrega a página
  sem ter definido a hora recebe o diálogo de novo (o `match_full_state` traz `storyAt: null`).
- Partidas que já começaram antes desta feature (dados de dev) caem no mesmo caso: sem hora,
  diálogo obrigatório.

### 3.2 O limite (nunca voltar antes de…) (dono)
O relógio nunca volta para antes de:
- a última ação que **de fato** aconteceu — o limite sobe quando um turno **abre** e quando fecha;
- a última troca de cena;
- a última troca de regime;
- (implícito) o início da partida e cada round contado.

Cada um desses momentos grava `limite = relógio atual` com o **motivo** (`start`, `turn`, `scene`,
`roundMode`, `round`), e a UI diz ao mestre qual é o limite e por quê. Tempo real rodando **não**
sobe o limite: o mestre pode voltar o tempo que deixou correr (dono), até o limite.

**O limite de um turno aberto segue a regra da casa** (minha, resolvendo um conflito apontado na
revisão): "dentro de um turno aberto nada é gravado antes do fechamento". Na **abertura**, o
limite sobe só em memória e o turno guarda o instante (`Turn.storyAt`); o fechamento grava os dois
na transação do turno (`turns.story_at` = instante da abertura). Um reinício no meio do turno o
perde inteiro, e o limite volta junto para o último gravado — coerente.

Um comando que acontece com turno aberto e já é gravado na hora (`change_round_mode`, os comandos
de relógio) grava o limite em memória daquele momento — inclusive o que o turno aberto subiu.

### 3.3 Mudanças do mestre
- **Ajuste relativo** (`adjust_clock`, em segundos, positivo ou negativo). Recusado se o resultado
  ficar abaixo do limite (`clock_below_limit`) ou se `delta` for 0.
- **Ir para um instante** (`set_clock`, absoluto) — a edição direta da UI. Mesmo limite.
- **Mais de 2 dias para frente exige confirmação** (dono): o servidor recusa avanço > 172 800 s
  sem `confirmLongSkip: true` (`clock_long_skip_unconfirmed`). A UI pergunta antes ("os
  personagens perdem os dias de treino desse período"). Vale para todo comando que move o relógio
  para frente (`adjust_clock`, `set_clock`, `change_scene` com `storyAt`, `end_match`). A base da
  conta é sempre o **relógio corrente antes do comando** (sem o segundo automático do round).
- Funcionam com o relógio rodando: aplicam-se ao valor corrente e ele continua rodando.

### 3.4 Tempo real (play)
- Só existe no **regime livre** (dono). `play_clock` no disputado → `clock_play_in_race`.
- **Sempre começa pausado**; o mestre liga explicitamente (dono).
- **Pausa sozinho** (dono) em: troca de regime, troca de cena, mestre sai da mesa (última conexão
  do mestre cai), encerrar partida.
- **Mestre sai = pausa imediata (minha)**, no instante da queda, sem carência: uma reconexão
  instável custa ao mestre um clique em ▶; um relógio rodando sem ninguém olhando é exatamente o
  que o dono quer evitar. A sala já sabe quando a conexão do mestre sai (`unregister`; uma conta
  tem uma conexão só, B4 — a substituída por outra aba **não** conta como saída).
- Enquanto roda, o servidor grava um **checkpoint a cada 60 s** (minha). Se o servidor cair, a
  sala que renasce encontra o relógio "rodando" e o **pausa no último checkpoint** (perde no máximo
  1 min, nunca soma horas de servidor parado) (minha).
- O valor corrente é `storyAt + (agora − runningSince)`, em segundos inteiros.

### 3.5 Combate: 1 round = 1 segundo (dono)
- Cada **round contado** (teve ao menos um turno aberto enquanto estava no disputado) soma **1 s**
  ao relógio **quando fecha** — por qualquer caminho: o fim automático do `open_next_action`, o
  `change_scene` e o `end_match`. Uma troca de regime no meio do round não desfaz a marca: o round
  que já abriu turno no disputado conta, mesmo que termine no livre. Isso acontece ao vivo:
  o relógio anda junto com o combate, e o limite sobe junto (motivo `round`). Não pode ser desfeito
  pelo mestre (dono: "não pode voltar atrás desse cálculo mínimo").
- Round sem turno nenhum não conta (e hoje nem fecha: `ErrRoundHasNoTurns`).
- Os jogadores **não** recebem aviso por round: só veem o número mudar (dono).

### 3.6 Hiato (dono, definição confirmada)
Um **hiato** é um round **disputado** que terminou pelo `open_next_action` com a **fila vazia** —
todos já agiram e ninguém declarou a próxima ação —, seguido de um round com turno na mesma cena.
Um round que termina com ações ainda na fila (sem barra para pagar) **não** é hiato. Para cada
hiato, o resumo do combate dá: o **último turno antes** (do round que esvaziou) e o **primeiro
turno depois** (do próximo round com turno), cada um com número do round na cena, quem agiu e o
tipo de ação. O sistema não sabe o que houve no hiato; o mestre se lembra e soma o tempo.
Um round que esvaziou e **não teve sucessor com turno** (o mestre trocou de cena logo depois) não
é hiato: o combate acabou ali. `rounds.drained` registra só o fato (fila vazia ao fechar); quem
decide se virou hiato é o resumo, olhando o round seguinte.

## 4. Fim de combate

Combate = **cena de categoria `battle`**. O combate termina trocando de cena.

- Quando o mestre abre "Nova cena" e a cena atual é de combate, o front pede o **resumo do
  combate** (`get_combat_summary`) e o diálogo vira **"Encerrar combate"** com:
  - quantos rounds foram contados e quantos segundos isso deu — **já somados no relógio** (o
    round em andamento aparece como "+1 s ao encerrar" quando conta);
  - a lista de hiatos (último turno antes / primeiro depois) e a mensagem: *"Nesses intervalos
    ninguém disputou ações — os personagens podem ter conversado, se reposicionado ou procurado uma
    brecha. Some ao relógio o tempo que eles duraram."*;
  - o **editor de relógio** começando no mínimo (relógio atual + o segundo do round em andamento,
    se ele conta), sem poder descer dele: é a **hora exata do fim do combate** (dono);
  - **descrição final da cena** (opcional, 255) — `scenes.brief_final_description`, que já existe
    e nunca foi preenchida;
  - os campos da nova cena (categoria, descrição inicial), como hoje.
- Trocar de cena de **roleplay** ganha só a descrição final opcional da cena que fecha (dono: o
  campo já existe, então entra). Sem resumo e sem editor de relógio.
- `change_scene` ganha `briefFinalDescription?` e `storyAt?` (+ `confirmLongSkip?`). O servidor:
  recusa turno aberto (como hoje); fecha o round (somando 1 s se contado); valida `storyAt ≥`
  mínimo; move o relógio; pausa o tempo real; grava tudo numa transação (cena e round fechados,
  relógio, eventos); limite = novo relógio, motivo `scene`.
- **Não dá para encerrar a partida durante um combate** (dono): é preciso trocar de cena antes.

## 5. Encerrar partida

- **Só o mestre**, pela mesa, via WS `end_match` (minha: é a sala em memória que precisa parar e
  avisar quem está conectado, como `start_match`; quem está offline descobre pelo `GET` da partida).
- **Recusado** com: cena atual de combate (`match_in_battle`), turno aberto
  (`cannot close round: current turn is still open`, o mesmo do `change_scene`), relógio sem hora
  (`clock_not_set`), partida já encerrada.
- **Fluxo (dono):** 1) confirmação avisando que **não dá para desfazer**; 2) diálogo com o
  **editor de relógio** (começa no relógio atual; mínimo = limite) para a **hora exata do fim** —
  obrigatória (dono: "vamos exigir isso do mestre, por enquanto") — e o **resumo final** opcional
  (`matches.brief_final_description`, 255), seguindo o padrão "(opcional)" dos formulários.
- O mínimo do editor é o maior entre o limite e "relógio atual + 1 s" quando o round em andamento
  conta (o mesmo cálculo do Encerrar combate: um round disputado numa cena de roleplay também
  vale 1 s). O resumo de combate não aparece aqui — a cena atual nunca é de combate.
- **No servidor**, numa transação: pausa o relógio e o põe no instante final; fecha round e cena;
  `matches.game_end_at = agora()` (instante **real**, coluna nova, no par de `game_start_at`);
  `matches.story_end_at = data do instante final`; `brief_final_description`; evento de histórico;
  e **`campaigns.story_current_at` avança para o instante final** (dono, 08/11: "essa é a ideia").
  A coluna é `TIMESTAMP` e já tem a regra "não regride" (`ErrCannotRegressStoryCurrentAt`): grava
  `GREATEST(COALESCE(story_current_at, instante), instante)` — uma partida jogada fora de ordem não
  puxa a campanha para trás (minha).
- **Todas as fichas participantes** (jogadores e NPCs, `match_participants`) avançam
  `character_sheets.story_current_at` (`DATE`) para a data do instante final (dono, 08/11: "as
  fichas participantes também! todas!!"), pela mesma regra de nunca regredir, na mesma transação.
  A fila e as reações em memória são descartadas. Depois: `match_ended` para a mesa inteira, a
  sala fecha as conexões com o **código 4002 `match ended`** e para.
- **O que marca "encerrada":** `game_end_at != nil` (minha, hipótese do handoff confirmada pelo
  modelo: `story_end_at` volta a ser só a data da história). Um método `Match.IsFinished()`
  substitui todos os guardas que hoje olham `StoryEndAt` (iniciar, inscrições, NPC, editar
  campanha/partida) — correção com o refactor junto.
- **Reconexão** numa partida encerrada (mesa ou lobby): o handler fecha com 4002 logo depois do
  upgrade, como o 4001 do lobby.
- **Front:** todos vão para a página da partida com `navigate(..., { replace: true })` — a `/game`
  sai do histórico do navegador, então "Voltar" na página da partida **nunca** volta para a mesa
  (dono). O jogador vê "A partida foi encerrada pelo mestre." `/game` e `/lobby` de uma partida
  encerrada redirecionam sozinhos (`replace`), o que cobre quem estava offline e quem digitar a URL.
  A página da partida encerrada esconde os botões de abrir mesa/lobby e usa `gameEndAt` no selo
  "ENCERRADA".
- **Fora desta entrega (dono):** mostrar o histórico inteiro na página da partida encerrada — é a
  próxima sessão. Fichas (HP, EXP) não são persistidas por encerrar além do que já é hoje.

## 6. Modelo de dados (migrations só aditivas)

| Onde | O quê |
|---|---|
| `matches` | `story_start_time TIME NULL` (hora de início opcional); `game_end_at TIMESTAMPTZ NULL` |
| `match_clocks` (nova) | `match_uuid PK FK`, `story_at TIMESTAMP NULL` (NULL = hora não definida), `limit_at TIMESTAMP NULL`, `limit_reason VARCHAR(16)`, `running_since TIMESTAMPTZ NULL`, `updated_at TIMESTAMPTZ` |
| `scenes` | `story_started_at TIMESTAMP NULL`, `story_finished_at TIMESTAMP NULL` |
| `rounds` | `counted BOOLEAN NOT NULL DEFAULT false` (round contado), `drained BOOLEAN NOT NULL DEFAULT false` (terminou com a fila vazia) |
| `turns` | `story_at TIMESTAMP NULL` (o relógio quando o turno abriu) |
| `match_events` | novos `kind`: `clockStarted`, `clockAdjusted`, `clockRealTime` (payloads na seção 7) |

**Tempo da história é ingênuo** (sem fuso): `TIMESTAMP`, nunca `TIMESTAMPTZ`; no Go, `time.Time`
em UTC tratado como "relógio de parede"; no fio, string `"2000-01-01T08:30:00"` **sem `Z` e sem
offset**. O front **nunca** passa isso por `new Date(str)` (aplicaria o fuso do navegador):
um util próprio quebra em partes e formata. Esse é o gotcha nº 1 desta feature.

**Gravação:** "um comando do mestre, uma transação" (regra da casa). O relógio vai na transação do
fechamento de turno/round (`TurnCloseData`/`PersistRoundClose`), do `change_scene` e do
`end_match`; os comandos de relógio têm a sua. O estado do relógio é um retrato (upsert): uma
gravação que falha é curada pela próxima; o evento de histórico perdido fica no log.
`match_events` exige `scene_uuid`/`round_uuid`: os eventos de relógio garantem a linha do par
ativo antes (`EnsureSceneAndRound`).

**Agrupamento no histórico (minha):** cliques seguidos do mestre viram **um** evento: um
`adjust_clock` que chega até 10 s depois do anterior, sem outro evento no meio, atualiza o evento
anterior (`from` fica, `to` e `deltaSeconds` acumulam) em vez de criar outro.

O front já relê o histórico quando o WS avisa ("WS avisa, REST busca"): o `clock_changed`
entra na lista de avisos que invalidam o `GET /history`, e o evento atualizado no lugar aparece
na releitura.

**Busca obrigatória antes de trocar o guarda:** todo uso de `storyEndAt`/`story_end_at` como
"encerrada" no back (queries de lista, filtros de campanha) e no front (selo ENCERRADA,
`getMatchStatus`, cards de partida) passa a olhar `gameEndAt`.

**Status (sessão paralela):** um helper de domínio `RoundEquivalents(from, to) int` (= segundos)
documenta a conversão. Nada de status é implementado aqui.

## 7. Protocolo (contrato completo vai em `docs/dev/api/match-clock.md`)

**Estado** (em `match_full_state.clock` e em todo `clock_changed`):
```json
{ "storyAt": "2000-01-01T08:30:00", "running": false,
  "limitAt": "2000-01-01T08:12:40", "limitReason": "turn" }
```
`storyAt: null` = hora não definida. Com `running: true`, o cliente conta a partir do momento em
que recebeu a mensagem.

**Cliente → servidor (só mestre, `forbidden` para os outros):**
`set_clock_start {time:"08:30:00"}` · `adjust_clock {deltaSeconds, confirmLongSkip?}` ·
`set_clock {storyAt, confirmLongSkip?}` · `play_clock {}` · `pause_clock {}` ·
`get_combat_summary {}` · `change_scene` (campos novos) · `end_match {storyAt, briefFinalDescription?, confirmLongSkip?}`.

**Servidor → cliente:**
- `clock_changed {clock, change}` — mesa inteira. `change.kind` ∈ `start` | `adjust` | `set` |
  `play` | `pause` | `round` | `scene` | `end` | `limit` (`limit`: só o limite subiu, ao abrir
  turno — o painel do mestre precisa dele; nenhum aviso); `change.from`, `change.to`, `change.deltaSeconds`;
  em `pause`, `change.reason` ∈ `manual` | `roundMode` | `scene` | `masterLeft` | `end` | `restart`.
- `combat_summary {sceneId, countedRounds, countedSeconds, activeRoundCounts, minimumStoryAt, hiatuses:[{before:{roundNumber,turnId,actorId,actionKind}, after:{…}}]}` — só para quem pediu.
- `match_ended {matchId, storyAt, gameEndAt}` — mesa inteira; depois fecha com 4002. `storyAt` é
  tempo da história (ingênuo, sem offset); `gameEndAt` é instante real em RFC 3339 com offset, como
  `gameStartAt`.
- `end_match` também aceita o mínimo do round em andamento: o servidor fecha o round (+1 s se
  conta) antes de validar `storyAt`.
- Erros novos: `clock_not_set`, `clock_below_limit`, `clock_long_skip_unconfirmed`,
  `clock_play_in_race`, `clock_already_set`, `match_in_battle`, `not_battle_scene`.

**Histórico (`GET /history`):** eventos `clockStarted {at}`, `clockAdjusted {from,to,deltaSeconds}`,
`clockRealTime {from,to,elapsedSeconds,reason}` (gravado ao pausar); cenas com
`storyStartedAt/storyFinishedAt` e `briefFinalDescription`; rounds com `counted`/`drained`; turnos
com `storyAt`.

**REST:** criar/editar partida aceitam `storyStartTime` (`"HH:MM"` ou `"HH:MM:SS"`, opcional;
string vazia limpa na edição). `GET` da partida devolve `storyStartTime`, `gameEndAt` e, encerrada,
`storyEndTime`.

## 8. Front

### 8.1 O relógio na mesa (mestre e jogador)
- **Onde (minha, com a sugestão do dono):** pílula centralizada no topo do palco, sobre o mapa,
  logo abaixo da barra do topo — o lugar clássico do relógio do mundo em HUD de jogo. A barra
  geral (`GeneralBar`) e os avisos do palco (`StageNotices`) descem para baixo dela. No celular, a
  pílula mostra só a hora; a data aparece no painel.
- **Fechada:** ícone de **cronômetro** + `08:30:00` (+ data `01/01/2000` a partir do tablet).
  Sem hora definida: "Hora a definir".
- **Ao vivo (tempo real ligado):** ponto "ao vivo" pulsando e os segundos correndo, para todos.
  **Só para o mestre**, a atenção escala para ele **não esquecer** (dono): pulso leve a cada 10 s,
  pulso mais forte a cada 1 min (com o tempo decorrido, "▶ 3 min"), e a partir de 5 min tom de
  alerta fixo + um aviso curto a cada 5 min ("Tempo real ligado há 10 min"). Com
  `prefers-reduced-motion`, sem movimento: só cor/opacidade em degraus. Só tokens existentes
  (`src/styles/CLAUDE.md`: cor nova é decisão do designer).
- **Jogador:** a pílula não abre painel. Cada `clock_changed` feito pelo mestre (`adjust`, `set`,
  `play`, `pause`, `scene`, `start`, `end`) mostra um aviso no palco por ~6 s: *"O mestre avançou o
  tempo 2 h 10 min — agora 01/01/2000 10:40:00"* / *"voltou o tempo 15 min"* / *"ligou o tempo
  real"* / *"pausou o tempo real (+3 min 12 s)"*. Avisos seguidos se **somam no mesmo aviso**
  enquanto ele está na tela (minha). `round` não avisa (dono), só pisca de leve. O Histórico ganha
  a linha de cada evento de relógio.

### 8.2 O painel do mestre (dono, níveis de profundidade)
1. **Clique na pílula** → abre o painel ancorado abaixo dela:
   - a hora grande (clicável → nível 3) e a data; o limite em texto pequeno ("Não volta antes de
     08:12:40 — último turno");
   - **▶ Tempo real / ❚❚ Pausar** (desabilitado no disputado, com o porquê em `Tooltip`);
   - **+1 h** e **+1 min** — mudam o relógio **na hora**, sem "Aplicar" (dono);
   - **Passar dias** → abre o nível 3 já com +1 dia;
   - **⋯ Mais** → revela o nível 2.
2. **Mais** (mais escondido, dono): **+1 s** e os de voltar: **−1 h**, **−1 min**, **−1 s**. Um
   botão que cruzaria o limite fica desabilitado, com o limite no `Tooltip`.
3. **Edição direta** (clique na hora): data com **[−] 01/01/2000 [+]** (passos de 1 dia — **nada de
   calendário**, dono) e hora em três campos `HH : MM : SS` (digitar ou setas); prévia da diferença
   ("+1 dia 2 h"); **Aplicar** e **Cancelar**. Abaixo do limite, o Aplicar desabilita e diz por quê.
   Mais de 2 dias para frente → confirmação. Este é o único lugar com "Aplicar" além de passar dias
   (dono).
- Fecha com Esc, clique fora ou clique na pílula.

### 8.3 O editor de relógio reaproveitado
O nível 3 é um componente (`ClockEditor`) usado também no **diálogo de hora inicial**, no
**Encerrar combate** e no **Encerrar partida**, cada um com seu mínimo e seu valor inicial (dono:
"pode ser o mesmo componente, em outro lugar"). Feature-level em `src/features/match/clock/`.

### 8.4 Diálogos
- **Hora inicial (obrigatória):** sem Cancelar e sem fechar por fora; data fixa da partida; campos
  `HH : MM` (segundos opcionais). O jogador vê "Hora a definir" até lá.
- **Nova cena / Encerrar combate** (seção 4) — evolução do `SceneChangeDialog`.
- **Encerrar partida** (seção 5): botão perigoso ao lado de "Nova cena" nos controles do mestre,
  desabilitado com `Tooltip` em combate / turno aberto / sem hora; `ConfirmDialog` ("Não dá para
  desfazer") → diálogo com `ClockEditor` + resumo final opcional → aguarda `match_ended`.

### 8.5 Formulário da partida
Criar/editar ganham **"Hora na história (opcional)"** ao lado da data: dois campos numéricos
`HH : MM`, sem seletor que abre por cima (dono: nada de calendário na cara). Ajuda: "Se ficar vazio,
você informa a hora quando a partida começar."

## 9. Fora do escopo
- Histórico completo na página da partida encerrada (próxima sessão, dono).
- Habilidades de voltar no tempo (dono: talvez outra feature).
- Os status em si (sessão paralela) — só o helper de conversão.
- O enum `GameEventCategory.DateChange` (sem uso) fica como está.

## 10. Entrega e verificação
- **Back PR** (`feat/story-clock`): migrations, domínio do relógio, comandos WS, resumo de combate,
  `end_match`, guardas `IsFinished`, REST da hora de início, contratos (`match-clock.md` novo;
  `match-combat-ws.md`, `match.md`, `match-history.md` atualizados; `documentation-map.yaml`), regra
  de jogo em `docs/game/` (1 round = 1 s, hiatos). Testes por camada + integração; `go vet` por
  tarefa; `golangci-lint` sem teto antes do PR; smoke com `curl` e script WS.
- **Front PR** (`feat/story-clock`): relógio, painel, `ClockEditor`, diálogos, avisos, histórico,
  redirecionamentos, formulário. Testes Vitest; **verificação no navegador** com mestre e jogador
  (`go run ./cmd/devseed`, contas seed — nunca as contas do dono), larguras de celular, tablet e
  desktop.
- Handoff pelo `dev-checkout.sh` se algo ficar sem verificar.
