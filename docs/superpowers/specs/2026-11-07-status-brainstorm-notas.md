# Status dos personagens — notas do brainstorm (rascunho)

> Rascunho vivo da sessão de brainstorm de 2026-11-07. Vira o spec de produto no fim.
> Não commitar como está. "Palpite" = proposto por Claude; marcado quando o dono aceitou.
> Desenho atual: `.superpowers/brainstorm/<sessão>/content/status-camadas-v*.html` (raiz).

## ▶ RETOMADA (leia primeiro)

**Estado em 2026-11-08, madrugada.** Brainstorm em andamento na skill `superpowers:brainstorming`,
fase de perguntas. Nada foi implementado; o spec ainda não foi escrito. O dono pausou para dormir e
volta na mesma sessão.

**Retomado em 2026-11-14** (sessão nova, modelo Fable; contexto recarregado deste arquivo). O que
mudou na semana e toca este desenho:
- **Relógio da história já existe** (spec 2026-11-08, `docs/dev/match/clock.md`) — é o "contador de
  tempo" da pendência paralela; a "conta fora do round" do sangramento pode se apoiar nele.
- **Vocabulário oficial** (regras da ficha, #107/#109): sistema = **System X System**; os 4 grupos
  `Abilities` = **Aptidões**; atributos médios = **secundários**; Stamina (não "Estamina");
  arredondamento sempre para baixo. As docs de jogo (`docs/game/`) são servidas no painel de
  regras: doc errada vira regra errada para o jogador.
- **Vida negativa ainda não existe no código** (o dono ia resolver) — o spec daqui a define.
- **Fadiga em STANDBY** (três tentativas; a última recusada). O spec pode sair sem ela. Não insistir.
- Compelido/Controlado aceitos por enquanto. Caído e Guarda aberta: lacunas dependem do movimento.
- Dano contínuo e Caído (parcial) fechados.
- **BLOCO 1 ENCERRADO (2026-11-14)**, com a Fadiga em standby e a distância do Lento / detalhes do Caído dependentes do movimento.
- Bloco 2 (gatilhos) fechado.
- Bloco 3 (visibilidade) fechado.
- Bloco 6 (nome: Barras de status × Efeitos de status) fechado.
- Bloco 7 (Força: Forçar / Agarrar / Carga) fechado.
- Blocos 1–8 percorridos. **O dono quer "fechar bem fechado", sem pressa.** Segue a **rodada de pontas soltas** (uma por vez):
  1. ~~Preso: como sai~~ (fechado); 2. ~~Guarda aberta: escapar?~~ (fechado); 3. ~~No ar × dash~~ (fechado); 4. ~~Atordoado~~ (fechado); 5. ~~Debilitado mental e espiritual~~ (fechado); 6. ~~Lesionado~~ (fechado); 7. ~~mestre aplica na mão~~ (fechado); 8. ~~fim da batalha~~ (fechado); 9. ~~NPCs~~ (fechado). **Claude decide e lista para o dono revisar no spec:** 10 (combinações sugeridas), 12 (histórico), 13 (nomes pendentes), 15 (aura/Nen reservado, com TODOs). ~~14~~ fechado. ~~11~~ fechado (visual com o Claude Design). **PERGUNTAS ENCERRADAS em 2026-11-15.** Próximo: mensagem das abordagens → spec (EN + PT-BR) → builder check. **Dever de Claude no spec:** diagrama de conjuntos de todos os status. Depois: abordagens, spec, builder check.
- ⚠️ O dono achou o Debilitado "complexo demais pra um status só": ver "Regra de escrita" na Intenção.

**ESTADO EM 2026-11-15:** perguntas encerradas; abordagem escolhida = **peças de efeito** (dono:
"pode ir"). **Spec PT-BR escrito:** `2026-11-15-status-effects-design.pt-br.md` (mesma pasta, não
commitado; a branch atual do repo é de outra sessão — `feat/rules-weapons-catalogue` — **não
commitar nela**). Builder check despachado. Falta: trazer ao dono só as perguntas "dele" + a lista
da §15 do spec; aprovação; versão EN; `writing-plans`; depois o brainstorm dos **testes**.

**Builder check voltou (2026-11-15).** Resolvido por Claude e já no spec: nomes das perícias no
código, Resiliência como teste de atributo, faces do Tato com o mesmo valor, soma de penalidades
(§4), tempo fora do round em lote com aviso ao mestre (§10), configuração da mesa como parâmetro.
**Quatro perguntas enviadas ao dono (aguardando):** (1) as saídas por ação — soltar-se, primeiros
socorros, reanimar, apagar o fogo — entram agora ou com o spec dos testes? (2) o que é a morte no
sistema; (3) o status vale para a ficha em qualquer campanha/partida ou só naquela; (4) crítico ×2
e erro crítico ÷2 valem para toda rolagem existente?

**Respostas do dono em 2026-11-15 (já no spec, v2):** (1) saídas por ação ficam para o spec dos
testes; (2) morte permanente, mestre desfaz, peça vira corpo (carregável; reanimável/controlável
por habilidade), todos veem; (3) status/barras/efeitos valem para a ficha em **qualquer campanha**;
(4) crítico: gerado só por teste (nunca pelo dado de dano), vale em qualquer teste; no acerto
dobra o **dano final**; regra em `MatchRules`, nunca em `die`/rolagens. **Vantagens e desvantagens
ACUMULAM** — esquema de rolagem em aberto (A: +1 D10 por nível, ficam os 2 melhores/piores; B:
segunda rolagem de 2 D10 e, no nível 2, 1 D20); tabela de probabilidades no spec §4; **aguardando
a escolha do dono**. **Regra nova para o spec dos testes:** dano negativo volta para o atacante
(detalhes e exemplos no spec §13). Decisões de Claude do output do builder check: aprovadas.
**Git:** branch `docs/status-effects-spec` (a partir da main, criada sem mexer no checkout, via
`scratchpad/commit_spec.sh`), PR rascunho aberto; cada versão do spec = um commit para o dono ver o
diff. As notas **não** estão commitadas.

**2026-11-15, v3 do spec:** dono escolheu o **esquema A** (um D10 a mais por nível de
vantagem/desvantagem; ficam os 2 melhores/piores) "por enquanto"; o caso do Gon sai dele, **sem**
rolagem especial de D20. Confirmou as duas leituras (excedente 40 = defesa 60; acerto crítico dobra
o total do teste e o dano final). Para o spec dos testes: ele **prefere** tratar o dano que volta
como golpe no atacante (Defesa reduz, Tenacidade decide o atordoamento). Falta: o dono terminar de
ler e aprovar; versão EN; notas no PR; `writing-plans`.

**2026-11-15, v4 do spec (revisão do dono, §§1–11 e 15 lidas e aprovadas):** novo status
**Escondido** (Furtividade × percepção; some de peça, barras, lista e histórico para quem não o vê;
granular no futuro, padrão agora); morte = tag + card escurecido como o NPC morto de hoje, e o
mestre **sempre vê antes**; relógio pausa 1–2 s antes de alguém morrer, com mensagem ao mestre e
botão de tornar pública; nomes de mesa do Debilitado: Corpo / Mente / Aura debilitada; "Hipnotizado"
no lugar de "Enfeitiçado" (jujutsu é futuro distante); comentários contraditórios **podem** ser
apagados e a regra do `AGENTS.md` deve ser reescrita no início da implementação. Dono perguntou se
"ficar consciente" deveria ser Força de Vontade — Claude recomendou manter Vitalidade (aguardando).
**Plano de tokens do dono:** 86% da semana usados, reset na terça; ele refina regras no fim de
semana e quer contexto pronto antes de pagar o plano maior.

**2026-11-15, v5 do spec — DONO CONCORDA COM TUDO** (leu §§1–11 e 15; 12–14 resumidas no
terminal). Últimos ajustes: "Espírito debilitado" (não "Aura"); Vitalidade mantida para ficar
consciente; **empilhar testes** anotado para o spec dos testes (o mestre empilha testes na pilha
de um teste; dono tem rascunho; incrementar depois). Próximo combinado: **brainstorm dos testes**
nesta sessão; versão EN, notas no PR e `writing-plans` a partir de terça (reset de tokens).

**Como conduzir (combinado com o dono):**
- Claude no papel de **PO**, não de líder técnico; o documento final é de produto, para outra sessão
  implementar. Sempre em português. Discussão de igual para igual.
- **Uma pergunta por mensagem**, de preferência com um palpite que ele corrige. Quando ele diz
  "decida", Claude decide e registra como decisão de Claude.
- Registrar **toda** resposta aqui na hora (ele teme o autocompact). Guardar **todos** os exemplos:
  vão para as docs de jogo.
- Não promover semelhança de forma a lei (memória `feedback_dont_promote_patterns_to_laws`).

**Desenho visual:** `System_X_System_Project/.superpowers/brainstorm/413965-1794092598/content/status-camadas-v3.html`
(é o estado atual). O servidor do visual companion desliga sozinho após 4 h parado; para religar na
mesma porta: `bash ~/.claude/plugins/cache/claude-plugins-official/superpowers/7.0.0/skills/brainstorming/scripts/start-server.sh --project-dir /home/azzurah/Documentos/HxH_RPG_Environment_Project/System_X_System_Project`
e escrever um arquivo novo (v4) no `content/`.

**Próximo passo:** continuar o bloco 1 (o que cada status faz e como acaba). Já respondido: Visão
anulada → desvantagem. Falta, nesta ordem sugerida:
1. Sentido afetado: anulado = falha automática (respondido); falta confirmar a saída "TR encerra";
   o que o Tato prejudicado faz no equilíbrio; "Atordoado não usa os sentidos" nos testes.
2. Debilitado: tamanho da desvantagem, nomes dos membros, como sai.
3. Lento: quanto reduz, de onde vem (movimento pode ser redesenhado pelo dono — não aprofundar).
4. Fatigado: limite de SP, o que trava (talvez só Lento + Debilitado físico). Conferir se o SP já é
   gasto em algum lugar do código.
5. Compelido: alcance, como sai.
6. Caído (esquiva por reflexo? repelir? custo de levantar) e Guarda aberta (escapar?) — o dono
   disse que ainda pensa; perguntar só se ele tiver resposta.
7. Queimando / Congelando / Envenenado: intensidade padrão e saída.
8. Asfixia / afogamento: para onde vai, já que não drena SP.

**Blocos depois do 1:** (2) gatilhos — o que o sistema aplica sozinho; (3) quem vê cada status
(um veneno pode ser segredo do mestre? lembrar: pendentes e dados já são segredo do mestre);
(6) nome do conceito ("status" já são as barras HP/SP/AP; "condição" é o ajuste do mestre);
(7) perícias de Força (código: Push/Grab/Carry; o dono hesita entre Force/Grab/Charge e
Push/Pull/Charge); (8) fórmulas da queda (perícia da resistência: Defesa ou Vitalidade; fator de
altura). Blocos 4 (repetição) e 5 (como o mestre aplica = testes em lote, feature futura) já
fechados.

**Depois do brainstorm:** escrever o spec de produto (EN + PT-BR, em `docs/superpowers/specs/`),
rodar o builder check (subagente com `builder-check-prompt.md` da skill), trazer ao dono só as
perguntas "dele", e decidir como compilar as docs de jogo (tabela tema → perícia, exemplos). Depois,
`superpowers:writing-plans`. A tarefa seguinte, combinada para esta mesma sessão se o contexto
deixar: **testes de perícia/atributo** (corrente de testes, Criatividade/Adaptabilidade como teste
de atributo mental, testes em lote e individuais, regra do crítico em `MatchRules`).

**Pendência paralela registrada:** contador de tempo — `System_X_System_Project/docs/handoffs/2026-11-07-contador-de-tempo.md`.

## Intenção

- Automatizar para o mestre não se sobrecarregar: nem de cabeça, nem de cliques no site.
- Liberdade narrativa: poucos status genéricos e combináveis representam qualquer coisa, do
  mesmo jeito que as actions (domínios + interseções; a narração dá o detalhe).
- O sistema propõe, o mestre dispõe: tudo o que o sistema aplica, o mestre põe, troca ou tira.
- O mestre que lê as docs do jogo não pode encontrar uma lista infinita de status.
- Nomes: os melhores são os mais abstratos que abarcam os casos específicos e combinam com a
  mecânica (dono).

- **Linha mestra (dono, 2026-11-14 — "deveria ter dito no início"):** o sistema é uma
  **representação da realidade**. Se uma regra faz sentido **físico, químico e biológico**, ela
  fica. Mas é uma **modelagem**, um sistema de RPG: não precisa ser fiel a ponto de trazer
  complexidade grande, porque o sistema já é complexo. Onde a complexidade escalar muito, o
  caminho é o **meio-termo**.

- **Segunda linha mestra, no MESMO nível da primeira (dono, 2026-11-14):** o sistema é
  **canônico** com Hunter x Hunter. As informações da obra (tradução oficial do volume físico) são
  fonte de regra. **Quando o cânon conflita com a realidade** e isso atrapalha a modelagem, pesar
  caso a caso:
  - coerência com as **mecânicas canônicas existentes** (redação do dono);
  - **realidade** nas Aptidões Física e/ou Mental;
  - **cânon** na Aptidão Espiritual e/ou Mental (depende do caso);
  - um **meio-termo** entre os dois, quando cabe;
  - outros fatores.
  Não é tarefa fácil (dono). ⚠️ **Levar às regras:** o dono pediu que as duas linhas mestras fiquem
  registradas nas regras do jogo. Decisão de Claude: ficam aqui e o spec manda a implementação
  escrevê-las em `docs/game/` (uma seção de princípios do sistema), porque as docs de jogo são
  servidas ao jogador e só entram por PR.
- **Regra de escrita (esclarecida ao dono):** cada status tem de caber em uma frase que o mestre
  lembre ("Debilitado: desvantagem nos testes da Aptidão marcada — Física, Mental ou Espiritual"; o dono pediu deixar **explícito de que grupo** se fala). Camadas extras — níveis, número, faces —
  só entram onde o dono pedir ou onde a obra/realidade exigir. Não é proibição de profundidade: a
  Fadiga, por exemplo, ele quer com degraus.

## Forma (proposta, aceita como direção)

Três camadas, como nas actions:
1. **Domínio** — a parte do personagem que o status trava: Tempo (quando age), Guarda (como
   reage), Corpo (onde está / como se move), Sentidos (o que percebe), Testes (como rola), Vida
   (só HP), Mente (raciocina / decide), Aura (Nen). Organiza a doc. Diagrama de Venn não é
   obrigatório (dono); a tabela domínio × status serviu.
2. **Status base** — a peça que o sistema calcula; a lista curta que o mestre aprende. Cada um
   carrega **origem** (decide a perícia que resiste), **intensidade** e **saída** (como acaba).
3. **Apelido** — o nome da cena; rótulo para uma combinação de status base (como "investida" =
   movimento + ataque). Vários status ao mesmo tempo são normais.

## De onde vem um status

1. **Regra direta, automática** — dano recebido → teste de atordoamento; HP < 0 → testes de
   inconsciência e sangramento.
2. **Teste que o mestre pede** — evento (terremoto: todos rolam equilíbrio) ou terreno (ponte,
   gelo, corda bamba). Manual.
3. **Falha na corrente de testes** de uma ação — o golpe "com tudo" que erra feio.
4. **O jogador declara** — se jogar no chão. Sem teste.
5. **Habilidade** (futuro) — a habilidade define área e efeito; o sistema envia o teste ao alvo.

Por cima de tudo: o mestre põe, troca e tira qualquer status a qualquer momento.

### Gatilhos automáticos (bloco 2 — dono, 2026-11-14: "dedução perfeita")
- Teste automático que falha → o status **entra direto**, sem clique de confirmação. Dentro de um
  turno, aparece no cálculo do turno junto com o dano (que já é só do mestre até fechar): o mestre
  troca ou tira antes de fechar; fechado, vale para a mesa. Fora de turno (veneno agindo, barra de
  afogamento), entra na hora e o mestre desfaz se quiser.
- **Condição do dono: o teste tem de aparecer de fato** (rolagem, dificuldade, resultado) — "rolar
  teste e não acontecer nada é incompleto", e status sem teste visível também.
- **Fronteira com a próxima tarefa (decisão de Claude):** este spec cobre os **testes que o sistema
  rola sozinho** (resistência a status) e como aparecem. Os **testes que o mestre pede** (em lote e
  individuais, corrente de testes) são o brainstorm seguinte, logo depois deste, com spec próprio;
  o dono ofereceu embutir aqui — Claude preferiu separar para não inchar, mantendo a mesma sessão.

### Quem vê cada status (bloco 3 — dono aceitou, 2026-11-14)
- Cada status aplicado tem uma **visibilidade**: (1) **todos** — padrão, e é como nascem os que o
  sistema aplica; (2) **só o dono do personagem e o mestre**; (3) **só o mestre** — o efeito age nos
  testes sem aparecer para o jogador (veneno não percebido, a agulha do Illumi). O mestre troca
  quando quiser.
- **Futuro (dono) — pesa na modelagem do back desde já:** **jogadores específicos** poderão ver
  **status específicos**. A mesma regra de "quem vê" valerá para outras coisas, como **portas
  abertas ou trancadas**. Modelar a visibilidade como lista de quem vê (por destinatário), não como
  três valores fixos; as três opções de hoje são atalhos dessa lista.

- **Preocupação do dono com o banco (2026-11-14):** serão muitos status; enquanto "jogador
  específico vê status específico" não existir, o ideal é **não consumir nada do banco** com isso.
  Ele cogitou tabela de relacionamento, sem certeza. Resposta de Claude, para o spec técnico: a
  **ausência de registro = todos veem**; só a exceção (escondido, ou lista de quem vê) grava algo.
  Custo zero no caso comum. Decisão final é de quem implementar.

### Nome do conceito (bloco 6 — decisão de Claude após a dúvida do dono, 2026-11-14)
- Dono: em RPG as duas coisas se chamam status; HP/SP também indicam o "estado" do personagem.
  Sugeriu nome composto ("status bars" + algo como "status badges", que ele mesmo achou ruim).
- **Decisão:** nome composto, com os termos já consagrados em jogos: **Barras de status** (*status
  bars* — HP, SP, AP; nada muda nas docs nem no código) e **Efeitos de status** (*status effects* —
  Caído, Sangrando...). Na fala da mesa, "status" serve para os dois; o composto desfaz a dúvida
  quando precisa. No código, `StatusEffect` não colide com o `status` das barras. "Condição"
  segue sendo o ajuste do mestre.

### Perícias de Força (bloco 7 — fechado, 2026-11-14)
- **Trio: Forçar (Force) / Agarrar (Grab) / Carga (Charge).** Direção proposta por Claude
  (empurrar e puxar são a mesma capacidade → "Forçar"), aceita pelo dono; o "Carregar" é o
  **Charge** dele.
  - **Forçar:** empurrar, puxar, arrombar, pressionar a ferida, compressão no peito.
  - **Agarrar:** segurar e imobilizar (Preso), soltar-se, ficar pendurado.
  - **Carga (dono):** capacidade de carga **e** aumentar o *momentum* — **golpes carregados** (o
    terceiro golpe do combo de boxe: jab, direto e o carregado; também com armas longas e menores;
    golpe mais lento, que pede uma pequena abertura, em geral contra oponente de força parecida) e
    o impacto da **investida** (arrancada/disparada). Ele pensa nisso "faz um tempo".
- Avaliação de Claude pelos pilares: faz sentido físico — erguer muito e bater pesado vêm da mesma
  força máxima. **Ressalva dita ao dono:** "acumular velocidade" é da **Aceleração** (Agilidade); a
  Carga não deixa mais rápido, ela mede **quanto impacto** o movimento vira ao acertar.
- **Ressalva retirada (dono explicou, 2026-11-14):** a Aceleração diz o quão rápido ele acelera,
  mas **ninguém acelera infinitamente**. A Carga serve para **acumular momentum além da capacidade
  de aceleração**. Personagens ágeis têm Aceleração e Frenagem parecidas (vão do 0 ao máximo e do
  máximo ao 0). Os **brutos** (ex.: **Uvogin**) aceleram pouco mas têm mais força: acumulam
  velocidade **além** do que conseguem acelerar, e **não conseguem frear num instante** — Carga
  maior que Aceleração. Então a Carga **aumenta a velocidade** dentro dessa lógica, assim como
  aumenta o dano no golpe carregado. Claude concorda: é a física de massa ganhando momentum.
- **Futuro (dono): biotipo** — mecânica que atribuirá **pesos no diagrama físico**, como o hexágono
  Nen dá peso ao tipo de aura.
- ⚠️ Para o spec/implementação: hoje o código tem `Push`/`Grab`/`Carry` (enum, docs, front,
  experiência gravada por nome). Renomear `Push→Force` e `Carry→Charge` pede **migração** como a
  das armas (20261113) e atualizar a paridade back/front. Golpe carregado e investida são da
  tarefa de testes/actions, não deste spec.

## Origens (famílias) → perícia que resiste

| Família | Exemplos | Resiste com |
|---|---|---|
| Física | impacto (concussivo), corte, perfuração, ultraperfuração | **Defesa** (Resistência) |
| Elemental | eletricidade, fogo, frio, veneno e afins — listar explícito nas docs | **Tenacidade** (Constituição) |
| Mental | a insanidade do Hisoka sobre o Kastro, medo, nojo, confusão mental | **Resiliência** (atributo mental) |
| Sangramento | — | **Cura** (Constituição) |

## Decidido

### Status não se repete (dono)
- O status indica **o estado**, não a quantidade. Existe **um** Sangrando, nunca mil; a intensidade
  (1 a 1000+ de HP por round, sem teto) é um número dentro dele, e as fontes somam nesse número.
- Status só se combinam com **outros** status — como nas actions: só existe uma action de ataque e
  uma de movimento, independente de quantas casas o personagem tenta andar.
- Palpite de Claude: um novo atordoamento em quem já está atordoado soma na reserva.
- Correção do dono (2026-11-08): **tipos diferentes coexistem** — sangrar e queimar são dois status;
  o que não se acumula é o mesmo tipo (dois cortes = um Sangrando mais forte). Vale para sentidos etc.

### Famílias (decisão de Claude, delegada pelo dono)
- **Família** = a mecânica compartilhada (o que trava, como a intensidade funciona). Organiza a doc.
- **Status** = um membro da família. É o que aparece no personagem, com a sua intensidade. **Um de
  cada; as fontes do mesmo membro somam; membros diferentes coexistem.**
- **Critério para virar família:** só quando os membros **não são intercambiáveis** — acabam de
  jeitos diferentes, resistem com perícias diferentes ou travam coisas diferentes. Não criar família
  por simetria.
- Aplicação:
  | Família | Membros | Por que não é um status só |
  |---|---|---|
  | Atordoamento | **Atordoado** (físico, Defesa), **Paralisado** (elemental, Tenacidade), **Atônito** (mental, Resiliência) | resistem e re-rolam com perícias diferentes; cada um drena a sua reserva; fora de ação enquanto qualquer um durar |
  | Dano contínuo | **Sangrando** (Cura; sai com socorros), **Queimando** (sai apagando), **Congelando**, **Envenenado** (Tenacidade; antídoto) | saídas e perícias diferentes |
  | Sentido afetado | **Visão, Audição, Olfato, Tato, Paladar** — cada um *prejudicado* ou *anulado* | cada sentido trava testes diferentes; o nível sobe (prejudicado → anulado), não duplica |
  | Debilitado | por grupo: físicos, mentais, espirituais (nomes a definir) | travam testes diferentes |
- Os demais (Inconsciente, Caído, No ar, Preso, Guarda aberta, Lento, Lesionado, Fatigado, Compelido)
  são status únicos. Novos membros só quando uma habilidade pedir.
- **Membro inutilizado saiu** (dono: mapear membro por membro fica ruim; decisão de Claude):
  - perna machucada → **Lento** (ou **Caído**, se não consegue ficar de pé);
  - membro **perdido** de vez (o braço do Kite) é mudança **na ficha**, não status: status é estado
    passageiro.

### Sentido afetado
- **Visão anulada → desvantagem no ataque e na defesa** (eles dependem da visão — dono). O
  personagem pode **treinar** para não depender dela (futuro: habilidade/treino que tira a
  desvantagem; ex.: En).
- **Anulado → qualquer teste daquele sentido falha sozinho** (dono). Ele citou o "TR encerra" do
  D&D. ⚠️ No D&D, "TR encerra" (save ends) é uma **duração** — o efeito dura até a criatura passar
  num teste de resistência, rolado a cada turno — e a falha automática é a da condição Cego. A
  confirmar se ele quis dizer também essa saída (rolar resistência a cada round para encerrar).

- **Respondido em 2026-11-14 (dono):** no vocabulário dele, "TR encerra" quer dizer só **"o teste
  falha automaticamente"** — vale para sentido **anulado**. **Não** é a duração do D&D. Até quando o
  status se mantém **vai de caso a caso**: quem diz é a causa.
- **Três saídas, todas válidas** (palpite de Claude, aceito):
  1. **Enquanto a causa durar** — escuridão, fumaça, venda, efeito de debuff, arma ou habilidade:
     o sentido fica anulado (falha automática) até o efeito sumir. Sem teste.
  2. **Até o corpo se recuperar** — clarão, estouro, areia nos olhos, pancada que deixa tonto: o
     sistema rola a cada round com a perícia da origem; passou, o sentido volta. (Palpite de Claude
     ainda **sem resposta explícita**: reaproveitar a reserva do Atordoado — a falta vira reserva e
     cada round desconta. Não tratar como lei.)
  3. **Até ser tratado** — olho perfurado, tímpano rompido: é ferimento; segue o Lesionado.
- **Prejudicado** (não anulado): pode ser **desvantagem**, **modificador negativo** ou **os dois** —
  caso a caso, depende do sentido e da causa (dono). O status precisa carregar as duas coisas
  (a marca de desvantagem e um número), e a fonte ou o mestre preenche.

- **Clarão forte = cegueira, não atordoamento** (dono, 2026-11-14): o personagem ainda se move,
  escuta e reage a estímulos; só a Visão não funciona, **até vencer um teste de Tenacidade** (Claude
  concorda: luz e som em excesso são da família elemental). O dono abriu a resposta com "boa!" ao
  palpite da reserva — lido como aceite: a falta no primeiro teste vira reserva, cada round desconta,
  e o clarão forte cega por mais tempo. Confirmar se ele estranhar.

### Tato: três faces (dono, 2026-11-14 — fechado)
- Problema visto pelo dono: com um status só de Tato, **vertigem** e **mãos dormentes** viram a
  mesma coisa, e as três partes do Tato (toque, equilíbrio, preempção) caem juntas.
- Dono: **separar equilíbrio e preempção do toque, ainda dentro do Tato** (a perícia da ficha não
  muda).
- Dono: quem está com **vertigem tem dificuldade para atacar, contra-atacar, esquivar etc.** e
  **pode até cair** (ganhar Caído).
- Proposta de Claude (aguardando resposta): o status de Tato carrega a **face** marcada — toque,
  equilíbrio ou preempção, uma ou mais —, como o Lesionado carrega os domínios. Só a face
  **equilíbrio** mexe sozinha em ataque, defesa, esquiva e movimento. O teste de Tato pedido pelo
  mestre diz a face ("teste de equilíbrio"), e o status só pesa nos testes da face marcada. Custo:
  o pedido de teste (próxima tarefa) ganha esse campo quando a perícia é Tato. Não estender aos
  outros sentidos.

- **Fechado (dono, 2026-11-14):** a proposta das faces vale, e o mestre **escolhe livremente 1, 2 ou
  as 3 faces** ao mesmo tempo.
- **Queda de quem está com o equilíbrio afetado:** o sistema pede sozinho um teste de equilíbrio
  quando ele **leva um golpe** e em **qualquer movimento** (não só os bruscos — correção do dono).
  Falhou → Caído. Equilíbrio anulado → o teste falha sozinho. **Exceção por tipo de movimento:**
  deslizar e rolar não derrubam (ele já está no chão). Isso é específico de cada movimento e será
  tratado quando o movimento for desenhado — **por ora só guardar**.
- O mestre também pode pedir teste para ele não cair; o desenho já cobre.
- **Princípio (dono):** o mestre pode fazer quase qualquer coisa, desde que não quebre a coesão das
  regras; há um limite até onde o sistema consegue garantir isso, e está tudo bem.

### Lesionado (status único — dono aceitou; "muito importante pro Hunter x Hunter")
- A **narração** diz a parte ("braço direito quebrado"); o sistema não mapeia anatomia.
- O mestre marca os **domínios de ação** atingidos: ataque, defesa, movimento, esquiva (braço ≈
  ataque + defesa; perna ≈ movimento + esquiva; mão ≈ ataque).
- Dois níveis, como os sentidos: **machucado** (desvantagem nos testes desses domínios) e
  **inutilizado** (a parte não pode ser usada; como o sistema não sabe em que mão está a arma, ele
  **avisa o mestre** quando uma action usa um domínio marcado, e o mestre decide).
- Um Lesionado só: lesões novas somam domínios no mesmo status.
- ~~Dura até o fim da batalha ou até ser tratado.~~ Ver "Lesionado — como acaba" (2026-11-14). O permanente vai para a ficha (outra sessão).
- Não é Debilitado: Debilitado = "como está" (grupo de testes); Lesionado = "o que está quebrado"
  (domínio de ação).
- **Futuro (dono):** uma funcionalidade que mapeie **o local e o volume exatos do corpo** inutilizado
  ou perdido, mesmo sem nome de status.

### Testes em lote e individuais (dono)
- "Como o mestre aplica na mesa" (o terremoto para todos) é a **funcionalidade de testes em lote**,
  construída depois desta. Também virá o **teste individual** (talvez a mesma rota). Fora do escopo
  aqui, mas o desenho dos status deve servir a ela.

### Docs de jogo
- A tabela "tema → perícia" (Defesa = golpe; Tenacidade = elemental; Cura = sangue; Vitalidade =
  consciência; Resiliência = mente) **vai para as docs de jogo**, enriquecida. Claude decide no fim
  como compilar (dono).

### Independência
Status são independentes; o sistema sugere pares. Inconsciente → também Caído por padrão; se
alguém carrega ou senta o personagem, sai o Caído e fica o Inconsciente.

### Atordoado
- Ao receber dano: teste da perícia da família contra CD = dano recebido.
- A falta (CD − resultado) vira a **reserva de atordoamento** (pontos).
- A cada round o sistema rola de novo — **sempre, automático** — e o resultado desconta da
  reserva. Zerou, sai do atordoamento.
- Ao sair: a **sobra** do teste que zerou a reserva (resultado − reserva restante) entra como
  **bônus** na barra (ação ou movimento) da **primeira ação** que vier depois. Ele se recuperou
  cedo dentro do round e ainda tem tempo — o mesmo racional do pouso do salto. Ex.: reserva 5,
  rolou 18 → +13 na primeira ação.
- **Enquanto atordoado:** não age, não reage (nem esquiva por reflexo nem defesa passiva), **não
  usa os sentidos**, mas **ainda raciocina**. Afeta a aura (futuro: a aura ao redor do corpo
  ainda pode protegê-lo, mesmo abalada).
- Palpites sem objeção: rola de novo **no começo de cada round** (passou, já age naquele round
  com o bônus; falhou, fica o round inteiro); as ações que tinha na fila **saem da fila**.

- **"Não usa os sentidos" nos testes (decisão de Claude, 2026-11-14, avisada ao dono):** enquanto
  Atordoado ou Inconsciente, todo teste de sentido falha sozinho — a mesma regra do sentido anulado
  —, sem criar um status de sentido à parte. Acabou o atordoamento, os sentidos voltam.

### Debilitado (em discussão, 2026-11-14)
- Grupos pelas Aptidões do código: **Físicos, Mentais, Espirituais** (a 4ª, `Skills`, não é grupo
  de teste). Os Físicos têm 8 atributos: Resistência, Força, Agilidade, Celeridade, Flexibilidade,
  Destreza, Sentido, Constituição.
- Proposta de Claude: carrega **desvantagem e/ou modificador negativo**, como o sentido prejudicado;
  um Debilitado por grupo, cada um com o seu número.
- **Alcance (dono, 2026-11-14):** se faz sentido com a realidade, mantém → o Debilitado (físicos)
  **pesa em todo teste do grupo**, inclusive resistências (Defesa, Tenacidade, Cura, Vitalidade) e
  Sentido. Corpo enjoado ou intoxicado de fato resiste pior, estanca pior e percebe pior. O número
  fica com a fonte ou com o mestre, que controla o quanto afunda.
- **Fontes do Enjoado** (resposta de Claude à pergunta do dono), cada uma com a perícia que resiste:
  veneno, comida estragada, gás ou fumaça tóxica (Tenacidade); golpe no estômago ou no fígado
  (Defesa); nojo, cena de carnificina (Resiliência); balanço de barco, giro (equilíbrio do Tato).
  O status que fica é o mesmo: Debilitado (físicos).
- **Padrão ao aplicar sem número (decisão de Claude):** só desvantagem.
- **Saída (decisão de Claude, avisada):** as mesmas três dos sentidos, pela causa — enquanto a causa
  durar; até o corpo se recuperar (rola a cada round); até ser tratado.

- **Dono (2026-11-14): "complexo demais pra um status só"**, embora confie no desenho final.
  **Simplificação (decisão de Claude):** o Debilitado tem **um nível só** — desvantagem nos testes do
  grupo; o número é opcional, para o mestre que quiser. **Não existe "Debilitado anulado"**: um corpo
  em que todo teste físico falha sozinho já é Atordoado / Paralisado / Inconsciente. "Enjoado" é
  apelido de Debilitado (físicos).
- **Veneno é fonte, não um status só** (dono): há veneno que enjoa (Debilitado), que paralisa
  (Paralisado), que tira HP por round (Envenenado, o dano contínuo), que cega, que faz dormir etc.
  Um veneno pode pôr vários status. Nas docs: "Envenenado" = só o dano contínuo; o resto é o que
  aquele veneno faz.

### Lento (dono, 2026-11-14 — provisório)
- Pesa **só na velocidade do movimento** (desvantagem e/ou modificador negativo). Anda depois e
  escapa pior. Proibir dash/salto é do Lesionado (movimento inutilizado), não do Lento.
- **Distância: em aberto de propósito** — definir depois da modelagem do movimento (dono).

- **Fechado em 2026-11-14 (dono):**
  - **Atordoado de novo enquanto atordoado → a nova falta soma na reserva** (confirmado).
  - **Sobra do teste quando a primeira ação é combinada** (usa as duas barras): **metade para cada
    barra, arredondada para baixo** ("opção 3", do dono; Claude concorda como solução provisória).
    Em ação simples, a sobra vai inteira para a barra daquela ação. ⚠️ **Provisório:** o dono ainda
    vai mudar o sistema de ação e movimento (qual ocorre primeiro).

### Debilitado mental e espiritual (dono, 2026-11-14 — "palpites excelentes")
- **Mental** — pesa nos testes dos atributos mentais (Resiliência, Adaptabilidade, Julgamento,
  Criatividade e os 4 que entrarão). Cenas: abalado por susto ou perda, dor forte que não deixa
  pensar, sono atrasado, cabeça confusa depois de uma pancada.
- **Espiritual** — pesa no Nen da Chama e da Consciência (Foco, Força de Vontade, Autoconhecimento,
  CoA, MOP, AOP) **e também nos outros testes de Nen: princípios e técnicas, categorias e
  habilidades** (acréscimo do dono). Cenas: aura abalada diante de aura maligna muito maior (Gon e
  Killua travados pela presença do Hisoka / Illumi), dúvida que enfraquece a determinação, desânimo
  após derrota. Existe já agora, antes das habilidades Nen.
- Consequência aceita: Debilitado espiritual piora a Força de Vontade, que resiste ao medo
  (Compelido) — quem está com a aura abalada cede mais fácil; é o que a Biske descreve no Killua.

- Dono confirmou: o Debilitado espiritual **pode ser criado agora**. Onde houver lacuna de Nen,
  deixar **TODO** para quando ele migrar o Nen do sistema antigo para este.

### Lesionado — como acaba (dono, 2026-11-14; corrige "até o fim da batalha")
- **Atravessa o fim da batalha.** Só sai com cura.
- **Socorro na hora NÃO devolve o uso** (dono recusou o palpite da tala: braço quebrado não volta a
  ser usado; nem na obra). Inutilizado continua inutilizado.
- **Sai por:** habilidade de cura ou item capaz de resolver a lesão (a regra fica na configuração do
  item/habilidade, quando existirem); cura depois da partida; e, no futuro, a funcionalidade de
  **treino, descanso e recuperação** (a última que fecha a versão 1 do projeto). **Até lá, o mestre
  define quando o status sai.**
- O que nunca cura continua sendo mudança na ficha.

### Persistência (dono, 2026-11-14)
- **Efeitos de status existem fora da batalha** e entre sessões (Lesionado, veneno lento, futura
  Fadiga). Dono já tinha notado: **tabela própria referenciando a ficha** — não enfiar na ficha,
  que já é gigante e dividida em tabelas. Basta buscar essa tabela a mais e exibir no local
  adequado. Isso supera a nota antiga do código ("CharacterStatus não será persistido").
- Onde exibir fora da partida: talvez uma sessão no **Claude Design** quando for a hora.

### Para onde vão as regras (dono, 2026-11-14)
- A **grande maior parte** disto são **regras de combate** (`docs/game/combate/`), exibidas **em
  outro local do site** — não nas regras da ficha, que o jogador já vê hoje no painel "?".
  Correções e enriquecimentos da ficha (descrição da Força de Vontade, perícias de Força) podem ir
  para lá.

### O mestre aplica na mão (dono, 2026-11-14)
1. **Um clique no caso simples:** personagem → status numa lista curta → aplicar; o resto nasce
   com o padrão. **O mestre precisa poder alterar os valores padrão** (dono).
2. **Só o campo que aquele status tem** (faces, domínios, conduta, intensidade); sem formulário
   único.
3. **Vários personagens de uma vez.** Ressalva do dono: o terremoto *pode* derrubar todos, mas **em
   geral o status vem de um teste que o mestre pediu e o personagem falhou** — o caminho comum é o
   da próxima tarefa (testes em lote); a aplicação direta em vários é o atalho.
4. **Editar e tirar** pelo próprio status.
5. **Apelido (nome de cena):** campo livre que aparece para a mesa junto do status. Vem com
   **apelidos padrão sugeridos** por status (os já levantados aqui); o mestre escolhe um ou escreve
   o seu. **Influência narrativa, não mecânica:** o sistema não interpreta o apelido. Vai para a
   **doc do mestre** (`docs/game/combate/o-mestre.md`).

- Apelidos padrão: Claude **pode enriquecer** com outros além dos já vistos, só os que fizerem
  sentido — "não precisa poluir; a ideia é facilitar" (dono).

### Fim da batalha (dono, 2026-11-14)
- **Sair do combate para a roleplay mantém a continuidade**: nada muda por causa da troca; cada
  status segue até a própria saída. O mestre pode passar o tempo e remover status, "mas isso é
  outra história".
- **Saem sozinhos** os que duram instantes e só fazem sentido com a luta correndo: **Atordoado** e
  **No ar**.
- **Paralisado e Atônito dependem da situação** (dono) → **não saem sozinhos**. Decisão de Claude:
  seguem com a reserva, que continua sendo descontada pelo relógio da história (1 round = 1 s); o
  mestre tira quando quiser.
- **Guarda aberta:** no futuro depende de o **jogador clicar num botão no canto da tela** para
  manter a guarda fechada — aí não se remove sozinha. **Agora: o sistema remove no fim da batalha,
  com TODO** registrando essa regra para quando o botão existir.
- Os que tiram HP (Sangrando, veneno) seguem valendo, contados pelo relógio da história.
- Antes de encerrar, o mestre vê o que sai e o que fica e pode mexer (palpite de Claude, sem
  objeção).

- **Revisão (decisão de Claude, delegada pelo dono em 2026-11-14):** uma regra só para a família
  do atordoamento — **Atordoado também não sai sozinho**; segue com a reserva descontada pelo
  relógio (sai em segundos de história). O No ar sai ao pousar, que é a saída dele. Logo, **o fim
  da batalha não remove nada**, exceto a Guarda aberta (provisório, com TODO).

### Visibilidade padrão — NPCs e jogadores (dono, 2026-11-14: "palpite perfeito")
- **Dois grupos pelo que se enxerga de fora:**
  - **nascem visíveis para todos:** Caído, No ar, Preso, Queimando, Inconsciente, Atordoado;
  - **nascem escondidos:** Envenenado, Debilitado, sentido afetado, Lesionado, Compelido,
    Controlado, Sangrando.
- **Vale também para personagens de jogador** (correção do dono: os jogadores **podem ser PK**; pode
  não ser interessante revelar que um PJ está debilitado). "Escondido" = só o mestre no NPC; **o
  dono do personagem e o mestre** no PJ (decisão de Claude: o jogador sente o próprio corpo, salvo
  o mestre pôr "só o mestre").
- O mestre **torna público quando quiser**, e no futuro só para **jogadores específicos**.
- **Opção do mestre:** deixar os status dos personagens de jogador **visíveis por padrão**, se
  preferir (configuração da mesa).
- Isso **substitui** "status que o sistema aplica nascem como todos veem" do bloco 3.

### Como o teste automático aparece (dono, 2026-11-14: "quase perfeito")
- **Quem vê:** o mestre vê tudo no cálculo do turno e pode trocar/tirar antes de fechar; o dono do
  personagem vê ao fechar o turno; os outros só se o status for público (teste de status escondido
  fica escondido); fora de turno, na hora, pela mesma visibilidade.
- **O mestre pode pôr viés e modificador no teste** (a "condição" que já existe nas rolagens).
- **Texto do card (dono):** `Defesa 12 contra 20 - Atordoado: 8`. **A palavra "reserva" não
  aparece** para o usuário (é nome interno das notas). O card do teste fica **acoplado/empilhado na
  action** que o gerou (hoje a action é só texto, não card).
- **Ordem do golpe (dono explicou):** o acerto pega → a defesa reduz o dano (o soco aparado com o
  braço) → o **dano final** sai do HP → **esse dano final é a CD do teste de atordoamento**, feito
  nesse momento. Isso balanceia o jogo. **Enriquecimento pedido:** o mestre deve **ver a redução**
  que a defesa fez (dano bruto → defesa → dano final); a sessão que implementar confere o que a
  tela já mostra hoje.
- **Histórico:** status privado (e o teste dele) **não aparece no histórico de quem não o vê**, até
  virar público — mesmo que isso leve várias partidas. O histórico mostra os da partida, e ao fim
  do turno também na **página da partida** (fora de `/game`).
- **Perícia do atordoamento por golpe — reaberta pelo dono:** ele não queria Defesa duas vezes (uma
  para reduzir o dano, outra para não atordoar) e tinha sugerido **Tenacidade**; aceita Defesa se
  Claude achar melhor. Proposta de Claude (aguardando): **Tenacidade**, e com isso "Paralisado"
  deixa de ser membro e vira apelido de Atordoado (mesma perícia); a família fica **Atordoado
  (golpe ou elemental — Tenacidade)** e **Atônito (mental — Resiliência)**.

- **Tenacidade fechada** para o atordoamento por golpe (dono: "estou contigo"). Tema: **Defesa
  apara o golpe; Tenacidade aguenta o que passou** (e o elemental); Resiliência segura a mente.
  ⚠️ Atualizar a tabela "Origens → perícia": Física deixa de ser Defesa no teste de atordoamento.
- **Juntar Paralisado e Atordoado: RECUSADO (dono).** Era nisso que ele pensava ao falar de
  **diagrama de domínios / Venn**, como as actions:
  - **O atordoado está paralisado, mas o paralisado não está atordoado.**
  - **Atordoado:** não reage, não usa habilidades "ou coisas do tipo"; pode até estar consciente,
    mas com a **consciência abalada**.
  - **Paralisado:** **plena consciência**; e não necessariamente o corpo todo — **pode mover partes
    do corpo**, conforme o caso.
- Leitura de Claude (aguardando): uma **escada em que cada degrau contém o anterior** —
  Paralisado (corpo travado, mente inteira) ⊂ Atordoado (+ consciência abalada: não reage, não usa
  habilidades, sentidos falham) ⊂ Inconsciente (+ sem consciência). Paralisia parcial marcada pelos
  **domínios de ação** (ataque, defesa, movimento, esquiva), como no Lesionado; padrão = todos.

- **Escada e paralisia parcial ACEITAS (dono, 2026-11-14: "gostei demais").** Paralisado ⊂
  Atordoado ⊂ Inconsciente. Paralisado marca **domínios de ação** travados (ataque, defesa,
  movimento, esquiva; padrão = todos); sai rolando Tenacidade a cada round. Mente inteira → pode
  usar habilidade que não dependa de mover o corpo (TODO para a migração do Nen); o atordoado não.
- **Pedido do dono: desenhar TODOS os status nessa lógica de conjuntos** (um contendo o outro); o
  que não se encaixar fica como domínio separado dentro do conjunto universo — "então tudo cabe".
  Pode gerar insights. Vai no **documento de saída (spec)** e nas **docs relacionadas**; o
  subagente que implementar as docs adiciona lá. Diagrama em texto, como o do gráfico da Biske.

### Onde aparece na tela (dono, 2026-11-14/15)
- O visual fica para o **Claude Design** (ou o dono traz uma ideia). O spec segue sem isso.
- **Na peça do mapa:** o mestre vê os status ao **selecionar a peça**; o ideal é vê-los como
  **ícones sem nome ao redor da peça**, e ao clicar os nomes expandem.
- **Na bottom sheet de enviar actions** (jogador; no desktop é o mesmo elemento sem ser bottom
  sheet, e o dono deve redesenhar): ela foi pensada para mostrar **o status do alvo e a action
  sobre ele**, e também o do **próprio personagem** de forma explícita — barras de status, os
  efeitos de status, a armadura (se conhecida) e afins. O mestre também vê ali.
- **Na lista de personagens da mesa** e **no próprio jogador, com destaque** quando ganha um status
  ou não pode agir por causa dele (palpite de Claude; dono: "muito importante, não tinha pensado").
- **Seguir princípios de UI/UX design** (dono). Sempre respeitando a visibilidade.
- Fora da partida (página da ficha): depois, com o design.

### Inconsciente (era "Desmaiado" — nome em aberto, recomendação: Inconsciente)
- Tudo do Atordoado **e não raciocina**. Afeta a aura.
- HP < 0 → teste de **Vitalidade** para não ficar inconsciente (palpite aceito).

### Sangramento
- HP < 0 → teste de **Cura** para não sangrar (palpite aceito).
- Sangramento também vem de tipos específicos de dano (habilidade de um ninja etc.).
- **Quanto tira:** sangrando por ter desmaiado de ferimentos → **1 de HP por round**. Sangrando
  por habilidade (ou outra coisa) → o que a habilidade definir; padrão 1, pode ser 2 ou mais.
- **Empilha:** cada fonte soma. Dois golpes de uma habilidade que sangra x → 2x por round.
- **Desmaiou já sangrando:** o sangramento que havia continua; ele faz um teste de **Cura mais
  fácil** que o de quem desmaia sem sangrar — falhou, soma mais 1 por round (o do desmaio).
- **Só para com primeiros socorros.** Consciente, o próprio personagem pode se tratar; se não
  conseguir (ou estiver inconsciente), precisa de alguém. No sistema antigo era teste de
  Inteligência (não existe mais). Hoje cabem três perícias, pela narração (texto aprovado para a
  doc de jogo):
  - **Precisão** — costurar ou fazer o curativo; a descrição da Destreza é "controle motor fino".
  - **Empurrar** — quando a narração é pressionar a ferida.
  - **Legerity** — o torniquete rápido, no meio da luta. **Padrão em batalha** (dono).
- Kit médico → bônus grande. Habilidade também pode resolver.
- **Criatividade e Adaptabilidade viram teste de atributo mental dentro da corrente** (dono): o
  mestre define uma CD base (10, 15, ou 20 se for mais difícil ou se a ideia não foi tão boa) e
  ela entra na corrente; falhar prejudica o resultado final. Vale para vários testes. ⚠️ Detalhar
  na próxima tarefa (testes de perícia).

### HP negativo (segunda barra)
- Ao zerar o HP, o personagem passa a perder **HP negativo**, até o máximo dele. Passou 1 além do
  máximo negativo → **morre**.
- Fórmula do HP hoje (código): `HP máx = 20 + (nível da Vitalidade + valor da Resistência) ×
  bônus dos Físicos`. Ex.: ficha do fixture de paridade, Vitalidade 0, Resistência 1, bônus 5 →
  25; Resistência 2 → 30. TODO antigo no código: `Min = -generateStatus.GetLvl()`.
- **HP negativo máx = 10 + nível da Vitalidade + valor da Resistência + bônus dos Físicos**
  (dono: soma em vez de multiplicar, com os mesmos fatores do HP, para não crescer tanto quanto o
  positivo; base 10 "para não ficar tão pouco"). Arredondado para baixo, como o HP (decisão de
  Claude; o bônus pode ser fracionado). Ex. (fixture): Netrunner só classe (HP 20) → 14; Samurai
  (20) → 15; Hunter pontos espalhados (30) → 17; Weapons Master (36) → 18; Monk (51) → 21; Ninja
  pontos concentrados (75) → 25. Dono: "qualquer coisa eu balanceio depois".
- **Aceito:** o dano que passa do zero continua descendo pelo HP negativo; um golpe maior que HP
  atual + HP negativo máx mata na hora.
- Recusada: fórmula com Tenacidade + Constituição (cresceria demais).

### Testes abaixo de 0 (inconsciência e sangramento)
- **Inconsciência:** sempre que o HP desce mais abaixo de 0 (golpe ou sangramento), teste de
  **Vitalidade contra o valor negativo exato** (em −4, CD 4). Sem o "+10" do palpite: a base 10 do
  HP negativo já faz esse papel. Chegou a 0 e desce só sangrando → testa a cada round em que perde
  HP, CD = o HP negativo dele.
- **Só rola quando pode falhar** (dono confirmou): se até um erro crítico passaria (CD ≤ resultado
  de um erro crítico), o sistema não rola o teste.
- **Sangramento ao passar de 0:** teste de **Cura** com **CD = último dano recebido + |HP negativo
  em que ele ficou|**. ⚠️ Somar o **valor absoluto**: o HP negativo tem sinal −, não subtrair. Ex.:
  levou 8 e ficou em −5 → CD 8 + 5 = 13. (Histórico: Cura foi palpite aceito na 2ª rodada; o dono
  cogitou Tenacidade em 2026-11-07 — ver recomendação de Claude abaixo.)
- **Inconsciente, a cada round:** rola **Cura** para estancar. Não zera o sangramento de uma vez:
  **cada dezena do resultado tira 1 do sangramento por round** (confirmado: resultado 23 com
  sangramento 3/round → fica 1/round). Enquanto não zera, continua perdendo HP. **Os primeiros
  socorros seguem a mesma conta** (confirmado).
- **Conta fora do round:** a perda de HP e os testes de quem está inconsciente não precisam
  acontecer dentro do round; podem ser contabilizados por fora, pelo **contador de tempo** (ver
  `System_X_System_Project/docs/handoffs/2026-11-07-contador-de-tempo.md`).
- **Decidido (recomendação de Claude): Cura nos dois testes de sangue** (não começar a sangrar e estancar), não
  Tenacidade. Uma perícia por tema, fácil de lembrar: Defesa = golpe; Tenacidade = elemental (choque,
  fogo, frio, veneno); **Cura = sangue**; Vitalidade = ficar consciente; Resiliência = mente. Tenacidade
  já carrega o atordoamento elemental; Cura ficaria quase sem uso.

### Crítico e erro crítico (regra de rolagem — nova para o código)
- 2 D10 somados (o "D20" de hoje). **Crítico** (os dois dados 10): resultado = (dados + valor de
  teste) **× 2**. **Erro crítico** (os dois dados 1): resultado = (dados + valor de teste) **÷ 2,
  arredondado para baixo**. Valor de teste = o parâmetro final da perícia ou atributo (o poder, com
  bônus).
- Hoje o código só marca `IsCritical`/`IsCriticalFailure` (`roll_calculator.go`); o `Total` não
  dobra nem divide. A sessão que implementar pode atribuir.
- **Onde mora:** fora da entidade `die` (o dono a quer abstrata dessa regra), numa camada de
  domínio mais externa, porque configurações de partida/campanha poderão ter regras diferentes.
  **Decidido: `MatchRules`** (`internal/domain/match/match_rules.go`, "os números que uma
  mesa pode mudar").

### Queda
- Dano = falta no teste (CD − resultado) + fator de altura (fórmula a fechar) − teste de
  resistência do personagem. Resultado ≤ 0 = sem dano: não é qualquer queda que machuca.
- Palpite: queda sem teste (empurrado do prédio) → só a altura conta.
- Perícia da resistência à queda: Defesa ou Vitalidade — a fechar.

- **Fechamento do bloco 8 (2026-11-14):**
  - **Fator de altura é PARÂMETRO, não tabela** (dono): "não definimos regras dessa forma nesse
    sistema". Na obra há quem pule de **30 m ou mais** sem dano porque usa **Nen**, e o Nen entra no
    cálculo da resistência. Logo: nada de tabela "x metros = y" nas regras; o 3 por metro é só o
    valor inicial do parâmetro (`MatchRules`), e a resistência precisa aceitar a parcela do Nen
    quando ela existir.
  - **Parkour/rolamento reduz ou anula o dano** (dono: se faz sentido com a realidade — faz: o
    rolamento espalha o impacto). Perícia delegada a Claude → **Acrobacia** (o controle do corpo no
    ar e no rolamento; é o próprio teste de pouso). Precisão (Destreza) recusada: é controle motor
    fino/mira. A conta tem sinal: o que **faltou** no pouso soma no dano, o que **sobrou** desconta.
  - **Resistência: Defesa** (Claude; o chão é um golpe). Vitalidade só depois, se o HP cair de 0.
  - Sem chance de reagir (empurrado, inconsciente): sem teste de pouso; altura − Defesa.

### No ar (novo, trazido pelo dono)
- Só evita ataques com **Evasão**; não se desloca para outro lugar além de onde já vai cair (a
  menos que uma habilidade ou objeto permita).
- **Não** é Exposto: ainda pode evadir, defender ou repelir, conforme o ângulo. O sistema não
  trata o detalhe; o mestre decide e pode pedir teste.

### Na tela, agora
- Efeitos visuais (escurecer a tela etc.) ficam para outra fase.
- Agora: o status aparece para o próprio jogador e para os outros; o atordoado não consegue agir
  até sair. **Decidido:** o sistema **bloqueia o envio** de ações e reações enquanto
  Atordoado/Inconsciente; o mestre pode liberar.

### No ar × dash × salto (dono, 2026-11-14)
- **As regras de hoje estão erradas** (`docs/game/combate/reacoes.md`, linha do Escapar: "durante
  o dash você está 'no ar' e não pode esquivar: fica exposto"). ⚠️ **Corrigir a doc** na
  implementação. O certo:
  - **No dash o personagem não fica exposto nem No ar.** Quem já se moveu com dash e é atacado
    naquele período (ação gatilho e outros casos) **dificilmente — talvez nem consiga — se mover de
    novo**; resta **evadir, repelir, aparar ou defender**. Dono aprovou a leitura de Claude: "quem
    arranca está comprometido com a trajetória".
  - **"Exposto" é do salto**, no sentido de estar **No ar**, não de Guarda aberta.
  - O exemplo do Gon roubando a plaqueta do Hisoka é **Guarda aberta durante o próprio ataque**
    ("cirúrgico").
- **Não aplicar No ar no dash agora**; o movimento será visto em breve.
- **No ar no salto:** o sistema aplica sozinho **enquanto durar o salto** e tira ao pousar.
  Normalmente o salto acontece dentro do próprio turno, e ele só é pego por ação gatilho ou caso
  específico. O status **perdura** quando ele **falha no teste de salto** e não chega à casa alvo
  num único turno (lento demais): fica No ar até o próximo turno dele, quando pousa — ou mais,
  conforme o salto.

### O que cada status deixa fazer na guarda

| | Agir | Mover | Esquiva por reflexo | Evasão | Defesa | Repelir | Escapar |
|---|---|---|---|---|---|---|---|
| **No ar** | ✓ | só a trajetória | ✗ | ✓ | ✓ | ✓ | ✗ |
| **Guarda aberta** | ✓ (arriscado) | ✓ | ✗ | ✓ | ✓ **rolada** (sem passiva) | ✗ — mas aparar conta como defesa (armada) | ✓ com desvantagem |
| **Caído** | ✓ | só rolar / deslizar | ✓ com desvantagem | ✓ | ✓ | ✓ com desvantagem | ✓ só rolando ou deslizando |
| **Preso** | ✓ (desvantagem a critério do mestre) | ✗ | ✓ | ✓ | ✓ | ✓ | ✗ |
| **Atordoado / Inconsciente** | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ |

- **Preso** não se move (o "parcial" da proposta sai; Enredado vira Preso, e o mestre dá a
  desvantagem que a cena pedir; "se desloca menos" é Lento).
- **Caído** depende de movimentos que ainda não existem de verdade (rolar, deslizar). ⚠️ O dono
  talvez precise resolver o **movimento** antes — talvez até antes dos testes de perícia.
- **Evasão** é a última "esquiva": o jogador não escolhe quando usar. Ela aparece quando ele está
  no ar, exposto, ou prestes a receber um crítico — para mitigar o dano ou ao menos não levar o
  crítico. Exemplo: Killua × Binolt (Greed Island, com a Biske) — quase perde o olho, evade e leva
  só um corte na bochecha.
- **Exposto = fora de guarda.** Mecânica futura do dono: o mestre e o próprio sistema podem pôr o
  personagem fora de guarda; o jogador precisa ficar esperto e clicar em **"entrar em guarda"**.
  Na batalha, só em momentos específicos; o uso maior é **fora da batalha** (alguém caçando o
  personagem). Liga com as **posturas** já reservadas no motor (`Stance`: em guarda / ofensiva /
  defensiva / evasiva). **Nome: Guarda aberta** (decisão de Claude, delegada pelo dono): o termo já aparece em `acoes.md` ("ficar de guarda aberta") e no Excalidraw, casa com o botão "entrar em guarda" e libera o "exposto" que `reacoes.md` usa solto no dash.

### Caído (dono, 2026-11-14)
- Caído **não é** necessariamente Guarda aberta (são independentes).
- **Esquiva por reflexo: com desvantagem.**
- **Escapar: pode, rolando, sem penalidade extra** (fora a que o escape já tem).
- **Repelir no chão: desvantagem automática** do sistema; o mestre modifica como entender (dono).

### Fogo, frio e veneno (dono aceitou o palpite, 2026-11-14)
- **Entrada:** um teste de **Tenacidade** contra a fonte, na hora. Falhou → pega com a intensidade
  da fonte. A Tenacidade entra **só no começo**.
- **Enquanto dura:** tira X de HP por round, **sem rolar** a cada round.
- **Queimando** acaba quando alguém apaga (rolar no chão, água, abafar); **Congelando**, quando sai
  do frio ou se aquece; **Envenenado**, com antídoto/tratamento ou quando passa o tempo do veneno.

### Asfixiando / afogamento (dono, 2026-11-14)
- Palpite de Claude aceito ("muito legal"): sem ar, ele resiste com **Respiração** contra uma
  dificuldade que sobe; falhou → começa a se afogar. Acaba quando volta a respirar.
- **Ritmo (dono):** a dificuldade sobe **1 ponto a cada 10 segundos** em batalha (o relógio da
  história conta 1 round = 1 s), **por enquanto**. Ajustar depois para os casos excepcionais do
  cânon: **Morel** aguenta meia hora a uma hora debaixo d'água; **Gon** prende a respiração 15 min.
- **Gasto de SP acelera (dono):** cada action que consome SP sobe a dificuldade **na hora**, e ele
  rola Respiração naquele momento. Lutar ou correr sem ar tira o fôlego mais rápido. ⚠️ **Lacuna:**
  o consumo de SP ainda não existe — registrar para preencher depois.
- **Não rola nos primeiros rounds:** só rola quando pode falhar (regra já decidida) ou quando uma
  action com gasto de SP sobe a dificuldade.
- **Não perde HP** (dono, corrige o palpite): ele **se afoga** — um **estado paralelo** em que pode
  morrer, "como se a vida estivesse caindo", sem dano direto. **Salvo, volta com o HP onde estava.**
- **Barra de afogamento:** só aparece quando o personagem está se afogando; **só no combate, não
  na ficha**.

- **Barra de afogamento — forma (dono, 2026-11-14):** ela **só aumenta**; não se define ponto zero
  nem máximo, "pelo menos por enquanto". Existe a fase em que ele **se debate consciente** e resiste
  com uma perícia até **perder a consciência**; a barra **continua subindo** com ele inconsciente.
  **O quão afogado ele está define a dificuldade de reanimá-lo.** "Pode usar quantas perícias
  quiser, desde que façam real sentido."
- Realidade conferida por Claude: prender o fôlego → debater-se consciente (≈30–60 s) → apagar por
  falta de oxigênio → morte em minutos.
- **Proposta de Claude (aguardando):** falhou na Respiração → barra aparece em 0 e sobe 1 a cada
  10 s (e com gasto de SP); consciente, resiste com **Vitalidade** (a perícia de ficar consciente)
  contra o valor da barra, só rolando quando pode falhar; falhou → Inconsciente, barra segue
  subindo. Salvo consciente → barra some, sem teste. Salvo inconsciente → a barra segue subindo até
  alguém **reanimar**: teste de quem socorre (padrão **Empurrar**, a compressão no peito; as outras
  de primeiros socorros conforme a narração) contra o valor da barra. Morte: sem número — quando a
  dificuldade ficou fora do alcance de quem socorre, ou quando o mestre declarar.

- **Dono aceitou a proposta (2026-11-14):** "prefiro coerência; desde que faça sentido e esteja
  coerente, está perfeito". Perícias: Respiração (fôlego), Vitalidade (não apagar), Empurrar como
  padrão de quem reanima. Tenacidade fica de fora.
- **Morte por afogamento (dono):** pôr na regra o **tempo médio real** até a morte de um afogado
  não reanimado, **com viés para cima** (personagens são fora do comum, talento de hunter).
  Decisão de Claude: na vida real o dano cerebral começa em 4–6 min sem oxigênio e a morte vem por
  volta de 10 min → **15 minutos depois de perder a consciência** sem ser reanimado, contados pelo
  relógio da história. Número padrão, ajustável (`MatchRules`). Antes disso, vale a reanimação
  contra o valor da barra.

### Preso — como sai (dono, 2026-11-14)
- **Preso por alguém** (agarrão): disputa. Quem está preso gasta a ação tentando se soltar **à
  força (Forçar)** ou **escorregando (Deslizar / Sneak)** contra o **Agarrar** de quem segura. Quem
  segura pode soltar.
- **Preso por algo** (rede, fios, escombros, gelo): a coisa tem uma dificuldade (mestre ou
  habilidade); tenta com Forçar ou outra perícia; aliado pode ajudar ou libertar.
- Soltar-se é **ação declarada**; o sistema não rola sozinho a cada round.
- **Ajuste do dono — vale para toda action:** quem escolhe a perícia é **o próprio jogador, na
  mecânica**. A narração entra na conta, mas tem de **refletir a mecânica escolhida**. Se não
  refletir, o mestre decide: **dá penalidade, anula a tentativa ou ajusta a perícia** conforme a
  narração, antes de seguir para a colisão dos testes — como já acontece hoje com as actions.
  (Correção a Claude, que tinha escrito "conforme narrar" e Acrobacia em vez de Deslizar.)

- **Complemento do dono:** Forçar e Deslizar são o **padrão** e aparecem sozinhos na **interface
  limpa**, junto com uma opção para **escolher livremente** outra perícia ou habilidade (ou seja,
  montar a action). Exemplos dele: pular acelerando numa direção para jogar o oponente contra o
  chão; usar **Agarrar** para esmagar o oponente ou apertar/quebrar uma parte do corpo dele e assim
  se soltar.

### Guarda aberta — escapar (dono concordou, 2026-11-14)
- Pode **escapar, com desvantagem na esquiva do escape**. É descuido, não incapacidade.

### Dano contínuo é só HP
- "Dano" é palavra de HP (vida). SP e AP não são vida.
- Hipotermia / congelando causa dano no **HP** (já reduzido pela Tenacidade), não no SP.

### Fatigado (SP baixo)
- Sem fôlego / fatigado é **consequência** de SP muito baixo: o status é atribuído quando ele já
  está cansado. Não é "dano no SP".

- **Conferido no código (2026-11-14):** **nada gasta SP hoje.** `Weapon.GetStaminaCost()` calcula o
  custo (corpo a corpo = peso da arma; arma de fogo = 1,0) mas nenhum ataque o desconta. SP máx =
  10 × [(nível de Energia + Resistência) × bônus dos Físicos]. Logo, qualquer limite de SP só passa
  a valer quando o gasto existir.
- **Notas antigas do dono** em `internal/domain/match/character_status.go` (comentário, não apagar):
  "stamina: cansado, exausto"; "mental: confuso, abalado, cansado, exausto, insano"; "posições:
  abaixado, deitado, caído — talvez caído = deitado + atordoado"; sentidos "cego, surdo, olfato
  comprometido, talvez mudo"; "furtivo?". E: o `CharacterStatus` **não seria persistido**, seria
  reconstruído a partir das actions ("considerar statusEngine") — ponto técnico para o spec avisar:
  status que o mestre põe na mão e que duram entre sessões pedem persistência; decidir lá.
- ~~Proposta de Claude: Fatigado vira apelido de Debilitado (físicos).~~ **Recusada pelo dono
  (2026-11-14):** a fadiga tem especificidades e **não entra no Debilitado**. O sistema pô-la
  sozinho conforme o SP era mesmo a ideia. "Hoje nada gasta stamina, mas irá gastar sim."

### Fadiga — o gráfico da Biske (em discussão, 2026-11-14; "fritar" antes de definir)
- **Ideia original do dono:** penalidades conforme o **nível de fadiga**. O personagem **cai em
  barras** e tem um **mínimo possível** — não é uma porcentagem que chega a 0. Existe um **nível
  máximo de exaustão** que ele não consegue mais resistir, "de acordo com a sua própria perícia
  relacionada e o deslocamento do próprio dado"; ali pode ficar incapacitado e **desmaiar**. O
  desenho em barras é "lindo para o nosso RPG" e fácil de mecanizar com rolagem de dados.
- **Fonte canônica 1 — Biske explica ao Killua** (arco das Formigas Quimera; tradução oficial do
  volume físico, transcrita pelo dono): "O poder de combate varia dependendo do ambiente e da
  condição física do momento. Por exemplo, se o lutador C estiver em perfeitas condições, pode
  derrotar o lutador D quando ele estiver em um dia ruim." (...) "Ou seja, dependendo da situação,
  tanto A como B ou C podem derrotar D. Mas... No seu caso! Você estima o poder máximo do
  oponente!! E sempre luta assumindo a sua pior condição. Mesmo quando luta com alguém do mesmo
  nível... não para de pensar o quanto o oponente é forte ou o que ele pode fazer para te deixar em
  perigo. Se o oponente for mais forte, mesmo que seja um pouco, você já desiste. A sua vontade de
  lutar desaparece por completo." Ela diz que ele precisará desenvolver **força de vontade** para
  encarar oponentes mais poderosos.
- **O gráfico** (descrito pelo dono): 6 linhas × 5 colunas; colunas na ordem E, C, B, A, D. O topo
  da linha 1 é a melhor condição física (e talvez do ambiente — não explícito); o fim da linha 6, a
  pior. Cada barra vai do melhor estado (em cima) ao pior (embaixo). Em "linhas" a partir do topo:
  | Personagem | Começa | Termina | Comprimento |
  |---|---|---|---|
  | D | início da linha 1 (0,0) | perto do meio da linha 4 (≈3,4; correção do dono: fica **um pouco mais acima** do que Claude desenhou, não tão perto do fim de A) | ≈3,4 |
  | A | fim da linha 1 (1,0) | fim da linha 4 (4,0) | 3,0 |
  | C | exatamente na linha 3 (2,0) | iniciozinho da linha 6 (≈5,1) | ≈3,1 |
  | B | pouco abaixo do início da linha 4 (≈3,2) | fim da linha 6 (6,0) | ≈2,8 |
  | E | pouco abaixo do início da linha 5 (≈4,2) | passa a 6 e a 7, pedacinho da 8 (≈7,2) | ≈3,0 |
- **Leitura de Claude** (a confirmar com o dono):
  1. As barras têm quase o **mesmo comprimento** (≈3 linhas), do mais forte ao mais fraco → a
     condição tira uma **quantidade fixa**, não uma porcentagem. Por isso nunca chega a zero e por
     isso o mais fraco num dia bom vence o mais forte num dia ruim.
  2. A conta fecha com o texto: o pior de D (≈3,6) fica abaixo do melhor de A (1,0), C (2,0) e B
     (≈3,2) — os três podem vencê-lo; o melhor de E (≈4,2) fica abaixo do pior de D — E nunca vence.
     Ou seja, a condição vira **degraus próximos**, não qualquer diferença.
  3. É o mesmo desenho de um teste: os dados (2 a 20) são uma barra, deslocada pelo valor de teste.
     Fadiga = descer a barra em degraus.
- **Dono aprovou a leitura e o desenho** ("que gráfico lindo... é canônico"): o desenho em texto
  **pode ir para as regras** quando Claude julgar o momento — com o fim de D um pouco mais alto.
  Versão a levar (ajustar o D):
  ```
          E     C     B     A     D
       ┌─────┬─────┬─────┬─────┬─────┐
    1  │     │     │     │     │ ███ │ ← melhor condição
       │     │     │     │     │ ███ │
    2  │     │     │     │ ███ │ ███ │
       │     │     │     │ ███ │ ███ │
    3  │     │ ███ │     │ ███ │ ███ │
       │     │ ███ │     │ ███ │ ███ │
    4  │     │ ███ │ ▄▄▄ │ ███ │ ▀▀▀ │
       │     │ ███ │ ███ │ ███ │     │
    5  │ ▄▄▄ │ ███ │ ███ │     │     │
       │ ███ │ ███ │ ███ │     │     │
    6  │ ███ │ ▀▀▀ │ ███ │     │     │
       │ ███ │     │ ███ │     │     │
       └─────┴─────┴─────┴─────┴─────┘
   (7)   ███
         ███
   (8)   ▀▀▀
  ```
- **Porta única (dono confirmou, 2026-11-14 — "perfeito"):** tudo o que piora a condição física
  **tira SP** (lutar, marcha forçada, noite sem dormir, fome, frio); a Fadiga **só lê o SP**. Dia
  ruim = começar a cena com o SP baixo. Ferimentos ficam fora (têm os status deles).
- **Dono:** a fadiga depende da stamina em si; ao longo de vários dias sem descansar a **stamina não
  recupera**, então a fadiga aumenta. A stamina dos hunters deve ser bem grande.
- **Rascunho de modelo (Claude; nada definido):** Fadiga é **status próprio, em degraus**, posto
  pelo sistema conforme o SP; cada degrau tira um **número fixo** dos testes físicos, igual para
  todos (modificador, não desvantagem nem %); o último degrau é **Exausto**, o piso — ali rola para
  não desmaiar e, quando nem um crítico passaria, cai sem rolar (regra "só rola quando pode
  falhar"). Números só de ilustração: 3 degraus de −3 (as barras têm ≈3 linhas) → −9 no fundo,
  metade da amplitude dos dados.
- **Fonte canônica 2 — Morel** (citação achada pelo dono no volume, 2026-11-14), antes da invasão
  do palácio, depois de dias de combate em Gorteau do Leste: "Sinto o corpo pesado... forcei demais
  durante vários dias. De zero a 100... devo estar em 40... não, 35." Depois: "Queria poder lutar
  em sua melhor condição... mas, uma vez que não tenho energias para desperdiçar, é até melhor para
  o plano." O que ensina: a obra mede a condição **de 0 a 100**; a 35 ele sente o corpo pesado mas
  ainda luta; a fadiga **acumula em dias** sem descanso.
  - Para medir a própria condição ele teria usado **ponderação** (dono). Anotar, **sem regra
    agora**: por enquanto o jogador simplesmente vê a verdade da própria condição.
- ~~Escada de Claude: 3 degraus fixos (−3/−6/−9) por quartos do SP.~~ **Recusada pelo dono
  (2026-11-14):** "valores fixos nos modificadores é ruim".
- **O que o dono disse no lugar (2026-11-14):**
  - a penalidade só começa quando a stamina chega a **50%**; acima disso, tudo no máximo;
  - **outra ideia dele: a stamina funciona como o HP** — tem valor **negativo**, e o piso é o
    negativo mínimo (ou perto dele); ao chegar a **0 de SP**, o personagem passa a fazer testes de
    alguma perícia **para se manter acordado**;
  - na fonte **cada personagem tem uma faixa de deslocamento diferente**: uns têm abrangência maior
    (**mais resilientes à fadiga**), outros menor (barra mais curta) → **os degraus dependem de
    perícia do personagem** (corrige a leitura 1 de Claude, "mesmo comprimento");
  - ele vai **desenvolver a ideia e voltar antes de fecharmos** — a Fadiga fica **em espera**.
- **Esqueleto oferecido por Claude (2ª tentativa, juntando as duas ideias dele; aguardando):**
  1. acima de 50% do SP, nada; 2. de 50% para baixo a Fadiga começa e cresce a cada degrau; 3. no
  zero a barra continua — **Stamina negativa**, como o HP negativo — e cada descida pede um teste
  para se manter acordado contra o quanto está negativo; 4. **o piso sai da própria rolagem**:
  quando nem o melhor dado + a perícia alcançam a CD, ele não resiste e desmaia — quem tem perícia
  maior desce mais fundo (barra mais comprida). Fica com o dono: tamanho do degrau, quanto tira, e a
  perícia que segura (Energia? Força de Vontade?).
- **Dono gostou do esqueleto (2026-11-14):** "gostei da sua ideia... ficou legal esse desenho".
  **A perícia que segura o personagem acordado é a Força de Vontade.** Ele aceita que Claude traga
  ideias e soluções para fechar o resto; "qualquer coisa nós mudamos depois".
- **Achado de escala (Claude, conferido nas docs):** o SP cresce **multiplicando** (10 × [(Energia +
  Resistência) × bônus]) e o valor de teste cresce **somando** (nível da perícia + poder do
  atributo). Um degrau medido em **pontos de SP** explodiria nos personagens fortes (SP 550 →
  dezenas de degraus contra valores de teste de 10 a 20). Por isso o degrau tem de ser uma **fração
  da própria barra**. Mesmo motivo para não usar "CD = SP negativo ponto a ponto" como no HP.
- **Proposta de Claude para fechar (3ª tentativa; aguardando):**
  - **Degrau = 1/10 da própria barra de SP** — o tamanho do degrau é pessoal (vem da Energia): quem
    tem mais SP precisa de muito mais esforço para descer um.
  - Acima de 50%: nada. Abaixo de 50%: **−1 por degrau** nos testes da Aptidão Física (−1 abaixo de
    50%, −2 abaixo de 40%... −5 abaixo de 10%). Morel a 35 → −2.
  - Abaixo de zero (**Stamina negativa**): os degraus continuam (−6, −7...) e, **a cada degrau
    negativo alcançado**, teste de **Força de Vontade** contra CD 10 no primeiro, 20 no segundo, 30
    no terceiro. Falhou → Inconsciente. Quando nem o melhor dado alcança, cai sem rolar: é o piso.
  - Resultado: barras de comprimentos diferentes — todos têm os 5 degraus até o zero; quantos
    degraus negativos aguenta depende da Força de Vontade (≈1 a 3).
  - Igual para todos: o **valor** do degrau (−1). Pessoal: o **tamanho** do degrau e a
    **profundidade**. ⚠️ O dono disse que "valores fixos nos modificadores é ruim" — conferir se o
    −1 por degrau incomoda. Coeficientes (50%, 1/10, CD 10 por degrau) em `MatchRules`.
- **3ª tentativa recusada (dono, 2026-11-14):** entendeu a complexidade técnica, mas não gostou da
  solução; "essa vai ser um pouco difícil de resolver". **FADIGA EM STANDBY.** Pode ficar **para
  outro momento** e o spec fechar **sem esse status** — hoje a stamina não conta nada, então está
  ok. O que fica de pé para quando voltar: status próprio (não Debilitado); posto pelo sistema pelo
  SP (porta única); nada acima de 50%; stamina negativa como o HP; Força de Vontade para ficar
  acordado; piso pela própria rolagem; barras de comprimento diferente por personagem; **sem valor
  fixo de modificador**; achado de escala acima. Não insistir: só reabrir se o dono trouxer a ideia.
- **Força de Vontade** (perícia canônica do Nen da Chama): a fala da Biske é sobre ela. Hoje a
  descrição nas docs é "Perseverança e determinação para seguir em frente quando tudo pede para
  parar" e aparece na ficha. ⚠️ **Na implementação, enriquecer a descrição** com esse sentido
  canônico: a vontade de lutar contra quem é (ou parece) mais forte, sem assumir a própria pior
  condição. Também pesa no desenho de atributos/testes (próxima tarefa).
- **Perguntas abertas da Fadiga:** o que desce o personagem na barra (só SP? fome, sono, ambiente,
  ferimentos?); quantos degraus e em que faixas de SP; quanto cada degrau tira; em que testes pesa;
  a perícia que resiste no piso (Energia? Vitalidade?) e o papel da Força de Vontade; se a aura (AP)
  tem fadiga própria; como o SP volta (descanso, relógio da história).

### Atributos e perícias
- Atributos mentais **não terão perícias**. Entram mais 4 atributos mentais (total 8). Teste
  mental é teste de atributo.
- Equilíbrio e preempção pertencem ao **Tato** (Sentido).
- Existe a ideia de uma configuração futura com **dados reais** (o jogador rola dado físico e
  informa) — rolagens automáticas custam caro nela.

### Compelido e Controlado (dono, 2026-11-14)
- **"Depende da situação"** (dono). O que ele disse:
  - **Não há sistema de provocação** como no D&D: provocar se faz **na roleplay**.
  - **Amedrontado é importante.** O teste pode ser mais de um, pela situação e pela leitura do
    mestre: **Resiliência ou Força de Vontade**.
  - **Manipulação por habilidade Nen** → "TR encerra" (o teste falha sozinho: não resiste). O
    personagem é **literalmente controlado por outro jogador ou pelo mestre**; o mestre decide.
    Essas habilidades ainda não existem — **preparar só o básico** no status.
  - **A agulha do Illumi no Killua** é habilidade de manipulação, mas **não é controle direto**: o
    Illumi não decide o que o Killua faz. É um **comando em uma condição específica** — fugir quando
    está em perigo.
- **Fechamento (decisão de Claude, a partir disso):** são **dois status**.
  - **Compelido** — o personagem carrega uma **conduta imposta** ("fugir de X", "não se aproximar
    de X", "fugir quando estiver em perigo"). O sistema **não bloqueia nada**: mostra a conduta e
    **lembra o mestre** quando o personagem age; o mestre decide — pede teste (Resiliência ou Força
    de Vontade), marca como **irresistível** (habilidade de manipulação: falha sozinho) ou deixa
    passar. Medo é Compelido; a agulha é Compelido irresistível com condição.
  - **Controlado** — outro declara as ações por ele (jogador ou mestre). **Só o básico agora:** o
    status existe e diz **quem controla**; a declaração de ações por outra pessoa chega com as
    habilidades de manipulação.
- Exemplo que Claude sugeriu para as docs (a confirmar com o dono): a **Corrente do Julgamento** do
  Kurapika — conduta imposta com **consequência** se violada, sem tirar a autonomia.
- O dono não lembrou de outros casos; voltar aqui se surgirem. **Aceito pelo dono "por enquanto"** (2026-11-14).

### Confuso / Alucinando
Não têm status próprio: cabem numa combinação de vários status (dono).

## Futuro (habilidades / Nen)
- Materialização: objetos e seres materializados continuam com o usuário atordoado ou
  inconsciente; criatura por comando é autônoma e segue lutando; manipulada direto, talvez não.
  Palpite: é regra **de cada habilidade** ("continua sem o usuário"), não do status.
- Controlado (outro declara as ações por ele — Illumi, Shalnark); Fluxo quebrado (técnicas de Nen
  ativas caem).

## Exemplos para as docs de jogo (não perder — o dono quer o máximo)

- Terremoto: o mestre pede equilíbrio (Tato) de todo mundo; quem falha cai.
- Ponte balançando, corda bamba, gelo, superfície escorregadia: atravessar pede equilíbrio.
- Habilidade que dá à aura a propriedade do gelo / de algo escorregadio, ou materializa algo
  assim: a área da habilidade manda o teste de equilíbrio sozinha.
- O golpe "com tudo" que erra: o mestre põe Caído.
- O personagem se joga no chão: Caído sem teste.
- Muito dano: Caído e Sangrando juntos. HP < 0: Inconsciente e Sangrando.
- Inconsciente carregado por um aliado ou posto sentado: sai o Caído, fica o Inconsciente.
- Choque elétrico: Tenacidade contra o dano para não ficar atordoado (paralisado).
- Kastro × Hisoka: a insanidade do Hisoka abala o Kastro, que fica atordoado (mental), não
  consegue usar o clone para se proteger e morre por isso.
- Atordoado por 20 de dano tirando 15: reserva 5; no round seguinte tira 18 → sai, +13 na
  primeira ação.
- Salto mortal sobre o inimigo com golpe de espada (actions): salto + acerto + Acrobacia.
- Apelidos da página: Cego/Ofuscado (Visão), Surdo/Zumbido (Audição), Tonto/Desequilibrado/
  Vertigem (Tato), Paralisado por choque, Atordoado mental (nojo, medo, confusão), Sangrando, Em
  chamas, Envenenado, Congelando, Afogando/Asfixiando, Agarrado, Enredado, Soterrado, Pendurado
  na borda, Amedrontado, Enjoado, Provocado, Em pânico, Enfeitiçado, Em fúria, Desprevenido,
  Surpreso.

## Em aberto

- Atordoado "não usa os sentidos": efeito na tela fica para depois; nos testes (percepção falha
  sozinha?) ainda a fechar.
- Caído: esquiva por reflexo e repelir? Exposto: escapar?
- Asfixia / afogamento: com o SP fora, para onde vai (Fôlego por round → Inconsciente? dano no
  HP?).
- Fatigado: limite de SP e o que trava. Talvez seja só apelido de Lento + Debilitado (físicos).
- No ar: o sistema aplica sozinho no salto (e no dash)? `reacoes.md` diz que durante o dash o
  personagem "não pode esquivar: fica exposto" — conferir contra a regra nova.
- Veneno dentro de "elemental" (palpite: soa estranho; talvez "tóxico", mesma perícia).
- Nomes: Inconsciente × Desmaiado; adjetivo (Cego) × substantivo (Cegueira); se os sentidos nem
  precisam de apelido ("Visão anulada" já se lê bem); nome genérico para o atordoado mental.
- Caso de borda (importante): a primeira ação depois do atordoamento usa as duas barras (ação
  combinada, ou duas ações com a mesma velocidade). Dono: "uma sempre ocorrerá primeiro".
- O que cada status base faz em detalhe (Caído, Preso, Lento, Exposto, Debilitado, Compelido…).
- Nome do conceito: "status" já são as barras HP/SP/AP; "condição" já é o ajuste do mestre.
- Perícias de Força: o código tem Push/Grab/Carry; o Excalidraw tem três versões.
- Fechar fórmulas nesta sessão (forma da regra sim; coeficientes com padrão ajustável).
