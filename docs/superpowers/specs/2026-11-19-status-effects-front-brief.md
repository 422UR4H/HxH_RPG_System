# Efeitos de status no front — brief de produto para a sessão de PD

> Escrito pela sessão de PO (2026-11-19) para uma sessão de **Product Design**. O dono do produto
> repassa. Não é spec técnico: diz **o que** precisa existir na tela, **para quem** e **por quê**.
> O **como** (layout, componentes, interação) é seu, decidido com o dono **direto no navegador**.
> **Sem Claude Design**: o designer humano do projeto está fazendo os protótipos finais e vai
> levar dias; esta versão é a que o jogo usa até lá. Faça bem, mas sem pretensão de ser a final.

## Por que agora

O back da fase 1 dos efeitos de status está pronto (PR #111, `feat/status-effects-base`, e a
continuação `feat/status-effects-history`). O mestre já consegue, pelo WebSocket, aplicar, editar
e tirar status; o servidor já bloqueia ações de quem está atordoado, esconde quem está Escondido e
avisa o mestre. **Nada disso aparece na tela ainda.** Sem front, a fase 1 não existe para a mesa.

## Leia antes (nesta ordem)

1. O spec de produto, §1 (intenção e princípios), §4 (vocabulário), §5.3 (o catálogo — só as
   frases), §8 (o mestre aplica na mão), §9 (quem vê), §11 (onde aparece):
   `System_X_System/docs/superpowers/specs/2026-11-15-status-effects-design.pt-br.md`
2. O contrato que o front vai consumir — é a fonte da verdade do que existe no fio:
   `System_X_System/.worktrees/status-effects/docs/dev/api/match-status-effects.md`
   (se a worktree tiver sumido, o mesmo arquivo está na branch `feat/status-effects-history`).
   Leia em especial §3 (os 20 tipos, as variantes, os nomes em PT e o padrão de quem vê), §4
   (campos por tipo), §6 (o que bloqueia), §7 (visibilidade e Escondido) e **§10 (o que ainda não
   existe — não desenhe para isso como se existisse)**.
3. Os CLAUDE.md do front: `System_X_System_React/CLAUDE.md`, `src/components/CLAUDE.md` (Atomic
   Design, zona pixel-tuned, promover em vez de duplicar), `src/styles/CLAUDE.md` (tokens em duas
   camadas e o fluxo para cor nova).

## Quem usa e o que cada um precisa

**O mestre** está sobrecarregado: conduz a cena, abre ações, narra. Ele precisa:
- **ver num relance** quem está com o quê, no tabuleiro;
- **aplicar um status em um clique** no caso simples (escolhe o personagem, escolhe o status numa
  lista curta, aplica — o resto nasce com o padrão);
- ver **só os campos que aquele status tem** (pontos do Paralisado, domínios do Lesionado, faces
  do Tato, conduta do Compelido…), nunca um formulário único com tudo;
- aplicar o mesmo status a **vários personagens de uma vez** (o terremoto);
- **editar e tirar** pelo próprio status;
- dar um **apelido** (nome de cena) — campo livre com as sugestões do contrato §3;
- controlar **quem vê** cada status (público / só o dono do personagem e o mestre / só o mestre;
  o modelo é uma lista, e no futuro haverá "só estes jogadores");
- receber os **avisos** do servidor (`statusNotes`: "ele usou o braço inutilizado", "isso fere a
  conduta imposta") no lugar em que já decide sobre a ação.

**O jogador** precisa:
- ver **com destaque** quando o próprio personagem ganha um status, e principalmente **entender
  por que não consegue agir** quando o envio é bloqueado (`blocked_by_status`) — hoje ele só
  receberia um erro;
- ver os status do **alvo** e do **próprio personagem** onde ele monta a ação;
- nunca ver o que não pode ver (status escondido, personagem Escondido). Isso é regra de produto,
  não detalhe: um ícone, uma contagem, um espaço vazio que entregue um segredo é um bug.

## Onde precisa aparecer (decidido pelo dono)

1. **Na peça do mapa:** os status como **ícones sem nome ao redor da peça**; ao selecionar/clicar,
   os nomes expandem. O mestre vê os status ao selecionar a peça.
2. **No painel de enviar actions** (bottom sheet no celular; no desktop é o mesmo elemento sem ser
   bottom sheet — o dono pretende redesenhá-lo): ele foi pensado justamente para mostrar o status
   do alvo e a action sobre ele, e também o do próprio personagem de forma explícita, junto das
   barras de status (HP, SP, AP), da armadura se conhecida e afins. O mestre também vê ali.
3. **Na lista de personagens da mesa**, com o número e o apelido; para o mestre, editar e tirar.
4. **No próprio jogador, com destaque**, quando ganha um status ou não pode agir por causa dele.

Fora de escopo agora: a página da ficha fora da partida; efeitos de tela (escurecer etc.); a
barra de afogamento e a mensagem "está morrendo" (as mecânicas não existem ainda — mas deixe
lugar para uma barra extra temporária por personagem e para um aviso urgente ao mestre).

## PR 2 do back — histórico e Escondido (PROVISÓRIO)

> O PR 2 (`feat/status-effects-history`) ainda está em andamento. O contrato do histórico sai em
> `docs/dev/api/match-history.md` quando ele abrir; **até lá, trate estes itens como provisórios**
> e desenhe o lugar deles sem fechar os detalhes. Decisões do dono de 2026-11-19, no spec §16.2.

- **Eventos de status no histórico**, cada um com a sua **causa** (a ação ou a reação do turno que
  o causou; sem causa, o evento fica no turno, na ordem do horário; fora de turno, nos eventos do
  round).
- **Causa opcional ao aplicar pelo card da reação:** com o turno aberto, o mestre pode aplicar,
  editar ou tirar um status a partir do card de uma reação ou ação, e isso registra a causa.
- **"Revelar?" ao tirar o Escondido:** o front pergunta ao mestre se revela o que ficou oculto
  (ações, alvos e status daquele personagem no período).
- **Botão de revelar no card do evento:** o histórico do mestre marca o que ainda está escondido, e
  ele revela depois, inclusive com a partida encerrada. Revelar vale para todos.
- **Alvo escondido cortado:** para quem não percebe um alvo Escondido, ele some da lista de alvos e
  do resultado, ao vivo e no histórico, até ser revelado.

## Restrições de produto

- **Seguir princípios de UI/UX** (dono). Leitura rápida para o mestre vale mais que completude.
- **Seguir o padrão existente do site:** procure o equivalente antes de criar (card = o
  `cardSurface` do card de campanha; erro = caixa escura + borda 1px + red600; red500 é do
  combate). Só cores da paleta; cor nova segue o fluxo de `src/styles/CLAUDE.md`.
- **Botões só de ícone** (expandir, recolher, fechar): grandes, podem ficar por cima do vizinho;
  miúdo e "encaixadinho" já foi recusado pelo dono. Sem exagero.
- **"Não regredir" inclui o visual do estado:** ao mexer na peça, no painel ou na lista, não apague
  pistas visuais que já existem (ex.: a peça selecionada, o encaixe do painel).
- **Nomes em português na tela, como no contrato §3** ("Visão anulada", "Corpo debilitado",
  "Guarda aberta"). O apelido aparece junto do status, sem substituí-lo para o mestre.
- **Ícones:** 20 tipos é muito para memorizar. Pense em agrupar pela família (consciência e corpo,
  parte do corpo, sentidos, testes, vida, ar, guarda, mente, presença — ver o diagrama do spec
  §5.2) e em mostrar o número quando o status tem um.
- Código em inglês, UI e comentários em português.

## O que eu (PO) espero de volta

1. Um **design doc curto** no front (`System_X_System_React/docs/superpowers/specs/`), com as
   decisões tomadas com o dono, uma por linha, e prints ou mockups de cada lugar da lista acima.
2. As **perguntas de regra de jogo** que surgirem (ex.: "o jogador vê os pontos do Paralisado de
   outro jogador?") **listadas, não decididas** — o dono traz para a sessão de PO.
3. Se o contrato do back não der o que a tela precisa, **liste a lacuna** (campo, mensagem) em vez
   de contornar no front: o back é a fonte da verdade e corrige o contrato.

Depois disso a implementação do front segue o fluxo normal (plano → subagentes → PR), em outra
sessão ou na mesma, a critério do dono.

## Como verificar no navegador

`go run ./cmd/devseed` em `System_X_System/` monta uma mesa com mestre e jogadores de login
conhecido (`docs/dev/seed.md`, inclui como ter os dois papéis no mesmo navegador). **Nunca derrube
a aba já logada do dono**: use as contas do seed. Front sempre na porta 5173; se estiver ocupada,
confira o PID antes de achar que é o seu servidor (há histórico de Vite órfão).
