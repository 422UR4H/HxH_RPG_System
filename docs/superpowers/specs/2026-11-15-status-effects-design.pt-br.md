# Efeitos de status — design de produto

> Spec de produto, saído do brainstorm de 2026-11-07 a 2026-11-15 (dono do produto + Claude como PO).
> Fonte completa das decisões, com quem decidiu cada uma: `2026-11-07-status-brainstorm-notas.md`
> (mesma pasta). Em conflito, **este documento vale**; as notas guardam o histórico e os exemplos.
> Marcação: **[dono]** = decidido pelo dono; **[Claude]** = decisão de Claude, delegada, a revisar.
> Versão EN: escrita depois da aprovação desta (as duas entram no mesmo commit).

## 1. Intenção

O mestre de uma mesa de System X System não pode se sobrecarregar, nem de cabeça nem de cliques.
Hoje o motor de batalha calcula acerto, defesa e dano, mas **nada acontece depois do dano**: ninguém
fica atordoado, caído, sangrando ou inconsciente, e o HP para em zero. Este projeto cria os
**efeitos de status**: o que o personagem sofre, o que isso trava, como aparece e como acaba.

É também o terreno da tarefa seguinte (testes de perícia/atributo pedidos pelo mestre): rolar um
teste e nada acontecer é incompleto, e um status sem teste visível também. [dono]

### Linhas mestras (mesmo nível de hierarquia) [dono]

1. **Realidade.** O sistema representa a realidade. Regra que faz sentido físico, químico e
   biológico fica. Mas é uma modelagem de RPG: onde a fidelidade escalar a complexidade, vale o
   meio-termo.
2. **Cânon.** O sistema é canônico com Hunter x Hunter (tradução oficial do volume físico). Em
   conflito entre cânon e realidade pesam-se, caso a caso: a coerência com as mecânicas canônicas
   existentes, a realidade nas Aptidões Física/Mental, o cânon na Espiritual/Mental, o meio-termo e
   outros fatores.

As duas linhas devem ser escritas nas regras do jogo (seção de princípios em `docs/game/`).

### Princípios de desenho

- **Poucos status genéricos e combináveis**, como as actions: a narração dá o detalhe. O mestre que
  lê as regras não pode encontrar uma lista infinita. [dono]
- **Um status cabe numa frase** que o mestre lembra. Camada extra (nível, número, faces) só onde a
  obra ou a realidade exigem. [dono]
- **O sistema propõe, o mestre dispõe.** Tudo o que o sistema aplica, o mestre põe, troca ou tira; e
  ele pode pôr viés e modificador em qualquer teste. Ele pode quase tudo, desde que não quebre a
  coesão das regras. [dono]
- **O status diz o estado, não a quantidade.** Existe um Sangrando, nunca dois: fontes do mesmo
  status somam no número dele; status diferentes coexistem. [dono]
- **Não promover semelhança a lei.** Onde dois status se parecem, este documento diz o que cada um
  faz; não estender regra por simetria.

## 2. Objetivos, não-objetivos e antiobjetivos

**Objetivos**
- Catálogo de efeitos de status com mecânica calculada pelo sistema (§5).
- Testes de resistência que o sistema rola sozinho, visíveis no turno (§6).
- HP negativo, morte e a regra de crítico/erro crítico (§7).
- O mestre aplicando, editando e tirando status na mão, com apelido de cena (§8).
- Visibilidade por destinatário e histórico coerente com ela (§9).
- Status que duram fora da batalha e entre sessões (§10).
- Docs de jogo de combate e ajustes nas docs da ficha (§12).

**Não-objetivos (ficam fora, de propósito)**
- **Fadiga** — em standby; o dono volta com a ideia (§13). A Stamina continua sem efeito.
- **Testes que o mestre pede** (em lote, individuais, corrente de testes) — brainstorm e spec
  seguintes. Este desenho precisa servir a eles.
- **Saídas e entradas que dependem de uma ação com teste** [dono concordou, 2026-11-15]: soltar-se
  do Preso, primeiros socorros, reanimar, apagar o fogo, levantar-se, e o teste de resistência
  quando a fonte é o mestre (ele teria de dar a dificuldade). Vêm com o spec dos testes. **Até lá o
  mestre tira esses status na mão** e digita o número de um Paralisado ou Atônito que aplicar. As
  regras ficam descritas aqui (§5.3) para aquele spec implementar. Os testes **automáticos** (§6)
  entram agora.
- **Movimento** (salto, rolar, deslizar, distância) — o dono vai redesenhar. O que depende dele
  está marcado como provisório.
- **Habilidades Nen e itens** — só o básico e TODOs para a migração do Nen (§13).
- **Visual** — sessão no Claude Design; aqui só o que precisa aparecer e onde (§11).
- Mapear anatomia (qual braço, qual mão); recuperação por treino/descanso (última feature da v1).

**Antiobjetivos** (seria fracasso mesmo funcionando)
- O mestre precisar confirmar cada status com um clique, ou preencher formulário longo.
- Um jogador descobrir por um teste, pelo histórico ou por um ícone um status que não podia ver.
- Uma tabela de "tantos metros = tanto dano" ou qualquer regra fechada em tabela onde o sistema usa
  parâmetro e teste. [dono]
- A lista de status crescer por apelido: apelido é texto, não status.

## 3. Abordagens

| | Como é | Troca |
|---|---|---|
| 1. Catálogo fixo | cada status escrito à mão no código, com os seus campos | simples; todo status novo (Nen, itens) exige código |
| **2. Peças de efeito (escolhida)** | conjunto pequeno de efeitos básicos; cada status é uma combinação declarada num lugar só | mais trabalho inicial; status novo = combinação nova |
| 3. Etiquetas livres | texto do mestre, sem cálculo | descartada: não automatiza nada |

**Escolha: 2**, decisão de Claude aprovada pelo dono ("pode ir"). É o desenho das actions (poucos
elementos que se combinam), a escada Paralisado ⊂ Atordoado ⊂ Inconsciente sai das próprias peças, e
habilidades e itens futuros só combinam o que existe. **Limite:** a primeira entrega tem só as peças
que os status deste documento usam; não criar peça para o futuro.

## 4. Vocabulário [Claude, após dúvida do dono]

- **Barras de status** (*status bars*): HP, SP, AP. Nada muda.
- **Efeitos de status** (*status effects*): este documento. Na mesa, "status". No código,
  `StatusEffect` (não colide com o pacote `status` das barras).
- **Condição**: continua sendo o ajuste do mestre numa rolagem (viés, modificador).
- **Apelido**: nome de cena de um status aplicado ("Enjoado", "Braço direito quebrado").
- **Domínios de ação**: ataque, defesa, movimento, esquiva (os das actions).
- **Só rola quando pode falhar** [dono]: se até um erro crítico passaria, o sistema não rola.
- Nomes das perícias em PT conforme `docs/game/glossario.md`; ver §12 para as de Força. No código:
  Precisão = `Accuracy`, Deslizar = `Sneak`, Tato = `Tact`, Respiração = `Breath`, Cura = `Heal`,
  Tenacidade = `Tenacity`, Vitalidade = `Vitality`, Defesa = `Defense`, Acrobacia = `Acrobatics`.
- **Resiliência é atributo mental, sem perícia:** o teste é de atributo e rola com o **poder do
  atributo**. [dono: atributos mentais não têm perícias]
- **As três faces do Tato rolam com o mesmo valor** (a perícia Tato); a face só decide se o status
  pesa naquele teste. [Claude]

### Como as penalidades se somam

- **Vantagens e desvantagens ACUMULAM** [dono, 2026-11-15 — corrige a proposta de Claude]. Cada
  fonte conta: Debilitado + Visão anulada = duas desvantagens.
- **Vantagem e desvantagem se cancelam uma a uma** [Claude]: duas desvantagens e uma vantagem = uma
  desvantagem.
- **Os números somam** (−2 e −3 = −5).
- Vale a regra que já existe: viés num teste passivo o torna rolado.
- **Como cada nível rola — um D10 a mais por nível** [dono, 2026-11-15: "é a cara desse sistema";
  "por enquanto"]: o teste rola 2 + N dados; na vantagem ficam os **2 melhores**, na desvantagem os
  **2 piores**. Crítico = os dois dados que ficam são 10; erro crítico = os dois são 1. Mantém a
  estabilidade que fez o sistema preferir 2 D10 ao D20, e com vantagem dupla a chance de crítico
  (5,2%) já alcança a do D20, sem os 5% de erro crítico — o caso do Gon roubando a plaqueta do
  Hisoka sai daqui, sem rolagem especial.
- Recusado: segunda rolagem inteira de 2 D10 com 1 D20 no nível 2 (linhas B e D20 da tabela, que
  ficam como registro da comparação).

  | Esquema | Média | Desvio | Crítico | Erro crítico | ≥ 8 | ≥ 11 | ≥ 14 | ≥ 17 |
  |---|---|---|---|---|---|---|---|---|
  | 2 D10 normal | 11,0 | 4,1 | 1,0% | 1,0% | 79% | 55% | 28% | 10% |
  | A — vantagem 1 (3 D10) | 13,5 | 3,7 | 2,8% | 0,1% | 93% | 79% | 53% | 23% |
  | A — vantagem 2 (4 D10) | 15,0 | 3,3 | 5,2% | 0,01% | 98% | 90% | 70% | 37% |
  | A — vantagem 3 (5 D10) | 16,0 | 2,9 | 8,2% | ~0% | 99% | 95% | 81% | 49% |
  | A — desvantagem 1 | 8,5 | 3,7 | 0,1% | 2,8% | 57% | 29% | 11% | 2% |
  | A — desvantagem 2 | 7,0 | 3,3 | 0,01% | 5,2% | 40% | 15% | 4% | 0,5% |
  | A — desvantagem 3 | 6,0 | 2,9 | ~0% | 8,2% | 28% | 8% | 1,5% | 0,1% |
  | B — vantagem 1 (melhor de duas) | 13,3 | 3,3 | 2,0% | 0,01% | 96% | 80% | 48% | 19% |
  | B — desvantagem 1 (pior de duas) | 8,7 | 3,3 | 0,01% | 2,0% | 62% | 30% | 8% | 1% |
  | 1 D20 | 10,5 | 5,8 | 5,0% | 5,0% | 65% | 50% | 35% | 20% |

  (Crítico = os dois dados que ficam são 10; erro crítico = os dois são 1.)

## 5. O catálogo

### 5.1 As peças de efeito [Claude]

| Peça | O que faz |
|---|---|
| **Não age** | bloqueia o envio de ações; as que estavam na fila saem |
| **Não reage** | nenhuma reação, nem as passivas (esquiva por reflexo, defesa padrão) |
| **Domínio travado** [domínios] | aquele domínio de ação não pode ser usado |
| **Aviso de domínio** [domínios] | o sistema avisa o mestre quando uma action usa o domínio; ele decide |
| **Reação restrita** [reação → proibida / com desvantagem / só rolada] | muda o que a guarda permite (tabela 5.4) |
| **Penalidade em testes** [escopo; desvantagem e/ou número] | escopo = Aptidão, perícia, face do Tato, domínio de ação ou velocidade de movimento |
| **Teste falha sozinho** [escopo] | o teste nem é rolado |
| **Não se desloca** | fica na casa (ou só na trajetória) |
| **Perde HP** [N por round] | dano contínuo; fora de batalha, contado pelo relógio da história (1 round = 1 s) |
| **Recuperação** [perícia; pontos] | o sistema rola a cada round e desconta; zerou, o status sai |
| **Teste por gatilho** [gatilho; perícia; falha → status] | ex.: equilíbrio ao levar golpe |
| **Conduta** [texto; resistível ou não] | lembra o mestre quando o personagem age |
| **Controle** [quem] | outro declara as ações |
| **Contador** | número que sobe com o tempo (afogamento) |

### 5.2 Diagrama de conjuntos [pedido do dono; desenho de Claude]

Um status dentro de outro contém todas as peças dele. O que não se encaixa é um domínio à parte no
conjunto universo.

```
┌──────────────────────────── EFEITOS DE STATUS ────────────────────────────┐
│                                                                           │
│ CONSCIÊNCIA E CORPO                         MENTE (vontade)               │
│ ┌─ Inconsciente ──────────────────────┐     ┌─ Controlado ─────────────┐  │
│ │ sem consciência                     │     │ outro declara as ações   │  │
│ │ ┌─ Atordoado ─────────────────────┐ │     │ ┌─ Compelido ─────────┐  │  │
│ │ │ ┌─ Paralisado ──┐┌─ Atônito ──┐ │ │     │ │ conduta imposta     │  │  │
│ │ │ │ corpo travado ││ mente      │ │ │     │ └─────────────────────┘  │  │
│ │ │ │ (domínios)    ││ travada    │ │ │     └──────────────────────────┘  │
│ │ │ │ ┌─ Preso ───┐ ││            │ │ │                                   │
│ │ │ │ │ não se    │ ││            │ │ │     TESTES                        │
│ │ │ │ │ desloca   │ ││            │ │ │     ┌─ Debilitado ─────────────┐  │
│ │ │ │ └───────────┘ ││            │ │ │     │ Física│Mental│Espiritual │  │
│ │ │ └───────────────┘└────────────┘ │ │     └──────────────────────────┘  │
│ │ │ + todos os sentidos anulados    │ │                                   │
│ │ └─────────────────────────────────┘ │     VIDA (só HP)                  │
│ └─────────────────────────────────────┘     ┌──────────────────────────┐  │
│                                             │ Sangrando  │ Queimando   │  │
│ PARTE DO CORPO                              │ Congelando │ Envenenado  │  │
│ ┌─ Lesionado: inutilizado ────────────┐     └──────────────────────────┘  │
│ │ ┌─ Lesionado: machucado ─────────┐  │                                   │
│ │ │ ┌─ Lento ───────────────────┐  │  │     AR                            │
│ │ │ │ (só velocidade do mov.)   │  │  │     ┌─ Asfixiando ─────────────┐  │
│ │ │ └───────────────────────────┘  │  │     │ fôlego → afogamento      │  │
│ │ └────────────────────────────────┘  │     └──────────────────────────┘  │
│ └─────────────────────────────────────┘                                   │
│                                             GUARDA (posição)              │
│ SENTIDOS (cada um dos cinco)                ┌────────┐┌───────┐┌────────┐ │
│ ┌─ anulado ───────────────────────────┐     │ Caído  ││ No ar ││ Guarda │ │
│ │ ┌─ prejudicado ──────────────────┐  │     │        ││       ││ aberta │ │
│ │ └────────────────────────────────┘  │     └────────┘└───────┘└────────┘ │
│ └─────────────────────────────────────┘                                   │
│                                                                           │
│ PRESENÇA: Escondido (quem não o percebeu não o vê em lugar nenhum)        │
│ EM ESPERA: Fadiga                                                         │
└───────────────────────────────────────────────────────────────────────────┘
```

Leituras do diagrama, para o dono confirmar na revisão:
- **Atordoado = Paralisado (corpo inteiro) + Atônito + sentidos anulados.** O Atônito é a metade
  "mente travada" do atordoamento; o Paralisado, a metade "corpo travado". Diferem na origem e na
  perícia. [Claude — nasceu do desenho]
- **Preso** é a peça "não se desloca" que o Paralisado de movimento também tem; o Preso mantém
  todas as reações.
- **Lento** é a parte de movimento do Lesionado machucado.
- **Controlado contém Compelido**: quem controla tudo impõe qualquer conduta.
- Os três da guarda não se contêm: são posições independentes. Caído **não** implica Guarda aberta.
  [dono]

### 5.3 Os status

Colunas: o que faz (a frase do mestre) · como entra · como acaba.

**Consciência e corpo**

| Status | A frase | Entra | Acaba |
|---|---|---|---|
| **Paralisado** | Os domínios marcados não obedecem; a mente está inteira. | fonte elemental (choque, veneno) ou mestre; marca domínios (padrão: os quatro) | Recuperação com **Tenacidade** a cada round |
| **Atônito** | A mente trava: não age, não reage, não usa habilidades. | fonte mental (a insanidade do Hisoka sobre o Kastro, pavor) | Recuperação com **Resiliência** a cada round |
| **Atordoado** | Fora de ação: não age, não reage, não usa habilidades nem os sentidos; consciência abalada. | dano recebido: **Tenacidade** contra o **dano final** (§6.2) | Recuperação com **Tenacidade** a cada round |
| **Inconsciente** | Além de estar Atordoado, está inconsciente. | HP abaixo de 0: **Vitalidade** (§7); afogamento | recobra quando a causa cessa — HP de volta a ≥ 0, reanimação, ou o mestre [Claude] |
| **Preso** | Não se desloca; age e reage normalmente (desvantagem a critério do mestre). | agarrão, rede, fios, escombros, gelo | ação declarada (ver abaixo) |

- **Recuperação (Atordoado, Paralisado, Atônito)** [dono]: o que faltou no teste de entrada
  (dificuldade − resultado) vira os pontos do status. **No começo de cada round** o sistema rola a
  perícia, sempre e sozinho, e desconta o resultado. Zerou, sai e já age naquele round. O que
  **sobrou** do teste que zerou entra como bônus na barra da **primeira ação** seguinte; em ação
  combinada (as duas barras), **metade para cada, arredondada para baixo** (provisório: ação e
  movimento serão redesenhados). Ex.: tinha 5, rolou 18 → sai com +13.
- Novo atordoamento em quem já está atordoado **soma** nos pontos. [dono]
- Enquanto Atordoado ou Inconsciente, **todo teste de sentido falha sozinho**. [Claude]
- O **Paralisado** pode usar habilidade que não dependa do corpo travado; o Atordoado não. TODO Nen.
- Fora da batalha nenhum deles é removido pela troca de cena: continuam valendo e se resolvem pelo
  relógio da história (ver §10, "Tempo fora do round").
- **Sair do Preso** [dono]: por alguém → disputa contra o **Agarrar** de quem segura; por algo → a
  dificuldade da coisa. É **ação declarada pelo jogador**; a interface limpa oferece **Forçar** e
  **Deslizar** e a opção de montar outra action livremente (pular para jogar o oponente no chão,
  Agarrar para esmagar ou quebrar). Quem escolhe a perícia é o jogador, na mecânica; a narração tem
  de refletir a escolha — se não refletir, o mestre penaliza, anula ou ajusta a perícia antes da
  colisão dos testes. **Isso vale para toda action.** Quem segura pode soltar; aliado pode libertar.

**Parte do corpo**

| Status | A frase | Entra | Acaba |
|---|---|---|---|
| **Lesionado** | Os domínios marcados estão machucados (desvantagem) ou inutilizados (o sistema avisa o mestre). | mestre ou fonte; a narração diz a parte | só com cura (ver abaixo) |
| **Lento** | Penalidade na velocidade do movimento. | mestre ou fonte | pela causa |

- Lesionado [dono]: um só por personagem; lesões novas somam domínios. Referência: braço ≈ ataque +
  defesa; perna ≈ movimento + esquiva; mão ≈ ataque. O sistema não mapeia anatomia.
- **Atravessa o fim da batalha.** Socorro na hora **não** devolve o uso. Sai com habilidade ou item
  de cura (regra na configuração deles, quando existirem), cura após a partida ou a futura
  recuperação; **até lá, o mestre decide quando sai**. O que nunca cura (o braço do Kite) é mudança
  na ficha, não status.
- Lento: só velocidade (anda depois, escapa pior). **Distância em aberto**, até o movimento. Proibir
  dash ou salto é do Lesionado (movimento inutilizado).

**Sentidos**

| Status | A frase | Entra | Acaba |
|---|---|---|---|
| **Sentido afetado** (Visão, Audição, Olfato, Tato, Paladar) | Prejudicado: desvantagem e/ou número nos testes do sentido. Anulado: o teste falha sozinho. | mestre ou fonte; resiste com a perícia da origem | pela causa: três saídas |

- **Três saídas** [dono]: (1) enquanto a causa durar (escuridão, fumaça, venda) — sem teste;
  (2) até o corpo se recuperar (clarão, estouro, areia) — Recuperação a cada round com a perícia da
  origem; (3) até ser tratado (olho perfurado) — é ferimento.
- **Clarão forte = Visão anulada, não atordoamento**; sai vencendo a **Tenacidade**. [dono]
- **Visão anulada → desvantagem no ataque, na defesa e na esquiva** (§16.4, item 35). Futuro: treino/habilidade tira (En). [dono]
- Um status por sentido; o nível sobe de prejudicado para anulado, não duplica.
- **Tato tem três faces** [dono]: toque, equilíbrio, preempção. O mestre marca livremente 1, 2 ou
  as 3. Só o **equilíbrio** afetado pesa sozinho em ataque, defesa, esquiva e movimento, e dispara
  um **teste de equilíbrio ao levar um golpe e em qualquer movimento**: falhou → Caído (anulado →
  falha sozinho). Deslizar e rolar não derrubam: regra por tipo de movimento, a fechar com o
  movimento. Os outros sentidos não têm faces.

**Testes**

| Status | A frase | Entra | Acaba |
|---|---|---|---|
| **Debilitado** — na mesa: **Corpo debilitado**, **Mente debilitada**, **Espírito debilitado** [dono] | Desvantagem nos testes da Aptidão marcada (Física, Mental ou Espiritual). | mestre ou fonte | pela causa (as três saídas dos sentidos) [Claude] |

- Um nível só; o número é opcional (o mestre troca a desvantagem por −N, ou soma). Não existe
  "Debilitado anulado". [Claude, após "complexo demais" do dono]
- **Pesa em todo teste da Aptidão**, inclusive resistências (Tenacidade, Cura, Vitalidade, Defesa) e
  Sentido. [dono]
- Um Debilitado por Aptidão; coexistem.
- Espiritual: Nen da Chama e da Consciência (Foco, Força de Vontade, Autoconhecimento, CoA, MOP,
  AOP) **e os demais testes de Nen** (princípios, técnicas, categorias, habilidades) — TODO onde o
  Nen ainda não existe. [dono]

**Vida (só HP)** — "dano" é palavra de HP; SP e AP não são vida. [dono]

| Status | A frase | Entra | Acaba |
|---|---|---|---|
| **Sangrando** | Perde N de HP por round até ser socorrido. | HP abaixo de 0 (**Cura**, §7); dano ou habilidade que sangra | só com primeiros socorros |
| **Queimando** | Perde N de HP por round até apagar. | **Tenacidade** contra a fonte, uma vez | alguém apaga (rolar no chão, água, abafar) |
| **Congelando** | Perde N de HP por round enquanto no frio. | idem | sai do frio ou se aquece |
| **Envenenado** | Perde N de HP por round. | idem | antídoto, tratamento ou o tempo do veneno |

- Fogo, frio e veneno: a Tenacidade entra **só na entrada**; depois, sem rolagem por round. [dono]
- **Veneno é fonte, não um status**: um veneno pode pôr Envenenado, Paralisado, Debilitado, Visão
  anulada, Inconsciente. "Envenenado" é só o dano por round. [dono]
- Sangrando [dono]: 1 por round por ter caído abaixo de 0; por habilidade, o que ela definir.
  Fontes somam. Estancar (o inconsciente rolando **Cura** a cada round, ou primeiros socorros):
  **cada dezena do resultado tira 1** do sangramento por round. Primeiros socorros, perícia
  escolhida pelo jogador: **Legerity** (torniquete; padrão em batalha), **Precisão** (costurar,
  curativo), **Forçar** (pressionar). Kit médico: bônus grande.

**Ar**

| Status | A frase | Entra | Acaba |
|---|---|---|---|
| **Asfixiando** | Segura o fôlego com Respiração; quando falha, começa a se afogar. | sem ar (água, estrangulamento, gás) | volta a respirar |

Tudo [dono], perícias [Claude]:
1. **Fôlego:** resiste com **Respiração** contra uma dificuldade que sobe **1 a cada 10 segundos**.
   Não rola enquanto não pode falhar. Cada action que gasta SP sobe a dificuldade na hora e pede a
   rolagem — **lacuna:** o gasto de SP não existe ainda.
2. **Falhou:** surge a **barra de afogamento**, em 0, subindo 1 a cada 10 s. Só sobe; sem máximo.
   Aparece só em combate, não na ficha. **Não tira HP.**
3. **Debatendo-se:** consciente, resiste com **Vitalidade** contra o valor da barra. Falhou →
   Inconsciente; a barra continua.
4. **Salvo consciente:** a barra some; HP intacto. **Salvo inconsciente:** a barra segue subindo até
   alguém **reanimar** — teste de quem socorre (padrão **Forçar**; outras de primeiros socorros)
   contra o valor da barra.
5. **Morte:** 15 minutos de história inconsciente sem reanimação (≈10 min reais, com viés para
   cima por serem hunters). Parâmetro ajustável.
- Calibrar depois com o cânon: Morel (meia a uma hora submerso), Gon (15 min de fôlego).

**Guarda (posição)**

| Status | A frase | Entra | Acaba |
|---|---|---|---|
| **Caído** | No chão: só rola ou desliza; reflexo e repelir com desvantagem. | falha de equilíbrio, golpe, o jogador se joga, mestre | levanta-se (custo: com o movimento) |
| **No ar** | Não muda a trajetória; sem reflexo e sem escape. | durante o salto (futuro) | ao pousar |
| **Guarda aberta** | Descuidado: sem reflexo, sem repelir, defesa só rolada, escapa com desvantagem. | mestre (ou sistema, no futuro) | fim da batalha (provisório) |

- No ar [dono]: o sistema aplica enquanto durar o salto. Perdura quando o teste de salto falha e
  ele não chega à casa alvo no turno. **Não se aplica ao dash.** A frase de `reacoes.md` sobre o
  dash ("está no ar… fica exposto") está **errada** e deve ser corrigida: quem já deu dash e é
  atacado dificilmente se move de novo; pode evadir, repelir, aparar ou defender.
- Guarda aberta: TODO — no futuro o jogador mantém a guarda por um botão ("entrar em guarda",
  liga com `Stance`); aí o sistema deixa de remover no fim da batalha.
- Inconsciente sugere Caído por padrão; carregado ou sentado, sai o Caído. [dono]

**Mente (vontade)**

| Status | A frase | Entra | Acaba |
|---|---|---|---|
| **Compelido** | Carrega uma conduta imposta; o sistema lembra o mestre quando ele age. | medo, habilidade, mestre | pela causa |
| **Controlado** | Outro (jogador ou mestre) declara as ações dele. | habilidade de manipulação | pela habilidade |

- Compelido [dono/Claude]: o sistema **não bloqueia**. O mestre pede teste (**Resiliência ou Força
  de Vontade**, pela leitura dele), marca **irresistível** (o teste falha sozinho — habilidade de
  manipulação) ou deixa passar. Medo é Compelido. A agulha do Illumi no Killua é Compelido
  irresistível com condição ("fugir quando em perigo"), não controle direto.
- **Não existe mecânica de provocação**: provoca-se na roleplay. [dono]
- Controlado: **só o básico** — o status existe e diz quem controla. A declaração por outra pessoa
  chega com as habilidades.
- Confuso e Alucinando não são status: são combinações. [dono]

**Presença** [dono, 2026-11-15]

| Status | A frase | Entra | Acaba |
|---|---|---|---|
| **Escondido** | Quem não o percebeu não o vê em lugar nenhum da mesa. | **Furtividade** contra a percepção dos outros; ou o mestre | é percebido, revela-se ao agir às claras, ou o mestre |

- Vale para NPC (o inimigo na moita desenhada no fundo do mapa — o sistema não precisa saber que há
  uma moita) e para personagem de jogador.
- Para quem **não** o vê, ele some de tudo: **peça no mapa, barras de ação e de movimento do
  round/turno, lista de personagens, e as ações que fizer enquanto escondido** (histórico
  escondido, com a mesma regra de §9: aparece quando for revelado).
- **Quem vê** é uma lista, como nos outros status: **será granular** (um jogador percebe e os
  outros não). **Nesta entrega usa o padrão dos demais:** vê o mestre e o dono do personagem; ao
  ser revelado, todos veem. Modelar já como lista.
- O teste de Furtividade contra a percepção (Visão, Audição… de cada observador) é disputa entre
  personagens: vem com o spec dos testes. **Até lá o mestre esconde e revela na mão.**
- Diferente dos outros status, a visibilidade aqui é o próprio efeito: o que se esconde é o
  **personagem**, não só o status.

### 5.4 O que cada status deixa fazer na guarda

| | Agir | Mover | Esquiva por reflexo | Evasão | Defesa | Repelir | Escapar |
|---|---|---|---|---|---|---|---|
| **No ar** | ✓ | só a trajetória | ✗ | ✓ | ✓ | ✓ | ✗ |
| **Guarda aberta** | ✓ | ✓ | ✗ | ✓ | ✓ só rolada | ✗ (aparar vale como defesa armada) | ✓ com desvantagem |
| **Caído** | ✓ | só rolar / deslizar | ✓ com desvantagem | ✓ | ✓ | ✓ com desvantagem | ✓ com desvantagem na esquiva e no movimento — provisório até o movimento (§16.4, itens 33 e 37) |
| **Preso** | ✓ | ✗ | ✓ | ✓ | ✓ | ✓ | ✗ |
| **Paralisado** | conforme os domínios | conforme os domínios | conforme (esquiva) | conforme (esquiva) | conforme (defesa) | conforme (defesa) | conforme (movimento + esquiva) |
| **Atônito / Atordoado / Inconsciente** | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ |

Toda desvantagem automática desta tabela é o padrão; o mestre muda. Rolar e deslizar ainda não
existem como movimentos: a linha do Caído se completa com o movimento.

### 5.5 Combinações que o sistema faz sozinho [Claude]

- Inconsciente → aplica Caído junto (removível).
- Falha no teste de equilíbrio → Caído.
- Falha na Vitalidade do afogamento → Inconsciente.
- HP abaixo de 0 → testes de inconsciência e sangramento (§7).
- No ar + (Atordoado ou Inconsciente) → o sistema **avisa** o mestre (queda); não calcula sozinho.
Nada além disso é automático: Atordoado não derruba por conta própria.

## 6. Testes que o sistema rola sozinho

### 6.1 Como entram e quem vê

- Teste automático que falha → o status **entra direto**, sem confirmação. [dono]
- **Dentro de um turno:** aparece no cálculo do turno, junto do dano, que já é só do mestre até o
  fechamento. O mestre troca, tira ou põe condição antes de fechar; fechado, vale para a mesa.
- **Fora de turno** (recuperação por round, fôlego, veneno): entra na hora; o mestre desfaz.
- **Visibilidade do teste = visibilidade do status** (§9): mestre sempre; dono do personagem ao
  fechar o turno; os outros só se o status for público.
- **Card do teste** [dono]: `Tenacidade 12 contra 20 - Atordoado: 8` — perícia, resultado,
  dificuldade, status e o número dele. Sem a palavra "reserva". Fica acoplado à action que o
  gerou (hoje a action é texto; o card vem com o design).

### 6.2 A ordem de um golpe [dono]

1. O acerto pega.
2. A defesa reduz o dano.
3. O **dano final** sai do HP. **O mestre deve ver a redução** (bruto → defesa → final); quem
   implementar confere o que a tela já mostra.
4. **Teste de atordoamento: Tenacidade contra o dano final.** Falhou → Atordoado com a diferença.
   Tenacidade e não Defesa, para a Defesa não valer duas vezes no mesmo golpe.
5. Se o HP ficou abaixo de 0: §7.
6. Se o alvo tem o equilíbrio afetado: teste de equilíbrio (Tato) → Caído.

### 6.3 Tema → perícia (vai para as docs de jogo)

| Tema | Perícia |
|---|---|
| Aparar o golpe (reduzir dano; a queda) | Defesa |
| Aguentar o que passou: atordoamento, choque, fogo, frio, veneno, clarão | Tenacidade |
| Sangue: não começar a sangrar, estancar | Cura |
| Ficar consciente (HP abaixo de 0, afogamento) | Vitalidade — o corpo; a Força de Vontade fica com a exaustão (Fadiga) e o medo. O mestre pode trocar a perícia do teste. [dono concordou] |
| Fôlego | Respiração |
| A mente: pavor, nojo, choque | Resiliência |
| Medo e conduta imposta | Resiliência ou Força de Vontade (o mestre lê a cena) |
| Equilíbrio | Tato (face equilíbrio) |

## 7. HP negativo, morte e crítico

- **HP negativo máx = 10 + nível da Vitalidade + valor da Resistência + bônus da Aptidão Física**,
  arredondado para baixo. [dono; arredondamento Claude] Ex. (fixture de paridade): HP 20 → 14 ou
  15; HP 30 → 17; HP 51 → 21; HP 75 → 25.
- O dano que passa de zero continua descendo. **Passou 1 além do máximo negativo → morre.** Um
  golpe maior que HP atual + HP negativo máx mata na hora.
- **Inconsciência:** sempre que o HP desce mais abaixo de 0 (golpe ou sangramento), **Vitalidade
  contra o valor negativo exato** (em −4, dificuldade 4).
- **Sangramento ao passar de 0:** **Cura** contra **último dano recebido + |HP negativo|** (levou 8
  e ficou em −5 → 13). Falhou → Sangrando 1 por round. Quem já sangrava faz um teste **mais fácil**
  (quanto: do builder, parâmetro); falhou, soma 1.
- Fora do round, a perda de HP e os testes do inconsciente correm pelo relógio da história.
- **Crítico e erro crítico** [dono]:
  - **O que gera crítico é o teste**, nunca o dado de dano: os dois dados do teste em 10 = crítico;
    os dois em 1 = erro crítico. Uma arma 2 D12 que tira 12 e 12 dá 24 de dano e pronto.
  - **Em qualquer teste** (acerto, esquiva, defesa, resistências): crítico → (dados + valor de
    teste + todos os números somados ao teste) × 2; erro crítico → ÷ 2, para baixo (§16.4, item 28).
  - **No acerto**, além disso: crítico **dobra o dano FINAL** — o que sai do HP, depois de todas as
    reduções. Dano final 0 continua 0. Vale também para paredes e (futuros) objetos. No acerto crítico valem as duas coisas juntas: o total do teste
    dobra e o dano final dobra. [dono confirmou]
  - Hoje o código só marca as flags. **A regra mora em `MatchRules`** (camada de domínio da
    partida), **nunca** na entidade `die` nem nas rolagens de dado: essas entidades não conhecem
    regras das camadas acima.
- **Morte** [dono]: é **permanente**; o **mestre pode desfazer**; a **peça fica no mapa como
  corpo**, pode ser carregada, e pode ser reanimada ou controlada por habilidade (ex.: manipulação
  de cadáveres) — TODO Nen. **Todos veem** que ele morreu, seguindo a névoa; na campanha, só ao encerrar a partida (§16.4, itens 31–32). Aparece como o NPC morto já aparece
  na campanha: **tag de morto e card escurecido**. (Hoje um NPC "forçado como morto" fica com o HP
  no máximo, o que não faz sentido: o morto pela regra fica com o HP onde morreu.)
- **O mestre sempre vê a morte antes da mesa** [dono]: dentro do turno ela aparece no cálculo, que
  é só dele até fechar, e ele pode intervir. Fora do turno, ver §10. "Morto" é um efeito de status público
  que contém o Inconsciente e não tem saída própria. [a forma como status é de Claude]
- Isto muda fórmulas da ficha (mínimo da barra de HP): regenerar a paridade back/front.

## 8. O mestre aplica na mão [dono]

1. **Um clique no caso simples:** personagem → status numa lista curta → aplicar. O resto nasce com
   o padrão, e **todo padrão é editável**.
2. **Só o campo que o status tem** (domínios, faces, nível, número, conduta). Sem formulário único.
3. **Vários personagens de uma vez** (atalho; o caminho comum será o teste em lote).
4. **Editar e tirar** pelo próprio status.
5. **Apelido:** campo livre mostrado à mesa junto do status, com **sugestões por status**. Peso
   narrativo, não mecânico: o sistema não o interpreta. Documentar em `o-mestre.md`.
6. O mestre libera manualmente quem está bloqueado (Atordoado etc.), se quiser.

**Apelidos sugeridos** [Claude, enxuto]: Visão — Cego, Ofuscado · Audição — Surdo, Zumbido · Tato
(equilíbrio) — Tonto, Vertigem · Tato (toque) — Dormente · Corpo debilitado — Enjoado, Febril,
Fraco · Mente debilitada — Abalado, Confuso · Espírito debilitado — Aura abalada, Desanimado · Paralisado — Em choque,
Travado · Atônito — Em pânico, Horrorizado · Preso — Agarrado, Enredado, Soterrado, Pendurado ·
Compelido — Amedrontado, Hipnotizado, Em fúria (sem "Enfeitiçado": não há feitiço no mundo
hunter) · Escondido — Furtivo, Emboscado · Guarda aberta — Desprevenido, Surpreso ·
Queimando — Em chamas · Asfixiando — Afogando, Estrangulado · Lesionado — livre ("braço quebrado").

## 9. Quem vê [dono]

- **Padrão pelo que se enxerga de fora**, para NPC e para personagem de jogador (há PK):
  - **públicos:** Caído, No ar, Preso, Queimando, Inconsciente, Atordoado, Asfixiando e Guarda
    aberta [os dois últimos: Claude];
  - **escondidos:** Envenenado, Congelando, Debilitado, sentido afetado, Lesionado, Lento,
    Paralisado, Atônito, Compelido, Controlado, Sangrando, Escondido [Congelando, Lento, Paralisado, Atônito:
    Claude — o dono revisa].
- **Escondido** = só o mestre, no NPC; **o dono do personagem e o mestre**, no PJ. O mestre pode pôr
  "só o mestre" (o efeito age sem o jogador saber — a agulha do Illumi).
- O mestre torna público quando quiser. **Configuração da mesa:** status de PJ públicos por padrão.
- **Futuro, a prever no modelo:** revelar a **jogadores específicos**; a mesma regra servirá a
  portas abertas/trancadas. Modelar como lista de quem vê; **sem registro = padrão**, para não
  gastar banco no caso comum. A forma final é de quem implementar.
- **Histórico:** entrada, mudança e saída de status e os testes ficam no histórico do turno e na
  página da partida. O que é privado **não aparece para quem não vê**, até virar público — mesmo
  partidas depois; ao ser revelado, o passado aparece. [dono; "o passado aparece" é Claude]
- O status privado pesa nos testes do jogador mesmo sem ele saber por quê.

- **A névoa vem por cima de tudo isto** [dono, 2026-11-17]: o jogador só recebe os status públicos
  de um personagem enquanto o vê. Detalhe em §16.1, item 4.

## 10. Persistência e continuidade [dono]

- **Qualquer status, barra ou efeito vale para a ficha em qualquer campanha e partida** [dono]: o
  Lesionado ganho numa campanha aparece se a ficha jogar em outra. A tabela é da ficha, não da
  partida. O HP negativo é o próprio HP da ficha.
- Efeitos de status existem **fora da batalha e entre sessões**. **Tabela própria referenciando a
  ficha**; não engordar a ficha. Substitui a nota antiga em `character_status.go` ("não será
  persistido").
- **Sair do combate para a roleplay não muda nada:** cada status segue até a própria saída. O fim
  da batalha só remove a Guarda aberta (provisório).
- Gravação segue a regra da casa: dentro de turno aberto nada é gravado antes do fechamento; o
  status entra na transação do fechamento.
- Valem para NPCs do mesmo jeito.

### Tempo fora do round [Claude — revisar]

- **"Fim da batalha"** = o mestre tirar a mesa do turno disputado (mudança do modo do round para o
  turno livre). É o único evento que remove a Guarda aberta.
- Fora do turno disputado o sistema **não rola segundo a segundo**. Quando o relógio da história
  avança, cada status é resolvido **em lote**, pelo tempo que passou:
  - **Recuperação** (Atordoado, Paralisado, Atônito, sentido se recuperando): usa o valor médio da
    rolagem por segundo; se o tempo bastou, o status sai. Sem rolagem.
  - **Perde HP e contador de afogamento:** antes de confirmar um avanço de tempo com alguém
    sangrando, envenenado, queimando ou se afogando, o sistema **mostra ao mestre** quanto cada um
    perderia e quem morreria ou cairia inconsciente, e ele confirma, ajusta ou trata antes. Nada de
    matar alguém em silêncio num salto de dez minutos.
  - **Mesmo com o relógio correndo segundo a segundo** [dono]: o relógio **pausa 1 ou 2 segundos
    (rounds) antes de o personagem morrer** e mostra ao mestre uma mensagem de que ele está
    morrendo. A mensagem tem um botão para **torná-la pública**, se o mestre quiser. Regra simples;
    a forma exata é de quem implementar.
- Dentro do turno disputado vale o round a round descrito em §5.3.

## 11. Onde aparece [dono]

- **Na peça do mapa:** ícones sem nome ao redor da peça; ao selecionar/clicar, os nomes expandem.
- **No painel de actions** (bottom sheet no celular): status do alvo e do próprio personagem, junto
  das barras, da armadura (se conhecida) e afins. Para mestre e jogador.
- **Na lista de personagens da mesa**, com número e apelido; para o mestre, editar e tirar. A barra
  de afogamento aparece aqui.
- **No próprio jogador, com destaque**, quando ganha um status ou não pode agir por causa dele
  (o envio fica bloqueado enquanto Atordoado/Atônito/Inconsciente e ele precisa entender por quê).
- **Pilha sobre a peça** [dono, 2026-11-19, com a sessão de PD]: de baixo para cima — a **peça**;
  logo acima, os **status** (se houver; organização da "opção C" da sessão de PD); acima deles, as
  **barras de Vida e de Stamina** (a de Aura substitui a de Stamina quando existir), se forem
  visíveis; no topo, o **balão de ação/reação**, se o personagem estiver agindo ou reagindo (o balão
  não fica colado ao personagem). As barras são simples: a cor padrão das barras para o valor atual
  e a cor mais escura do projeto para a parte vazia. **Cada jogador vê só as próprias barras, por
  padrão; o mestre vê todas e pode revelar as de outros jogadores a todos.** As reações aparecerão ao
  redor da peça (depois). ⚠️ **Lacuna para o back:** hoje Vida e Stamina só vão ao dono e ao mestre
  (`character_hp_changed`); revelar as barras de um personagem aos outros jogadores ainda não
  existe no contrato. Entra com a visibilidade por jogador (§16.3, item 24).
- Seguir princípios de UI/UX; o visual desta fase é decidido com a sessão de PD, sem Claude Design
  (o designer humano do projeto faz os protótipos finais depois). Efeitos de tela (escurecer etc.) e a página
  da ficha fora da partida: depois.

## 12. Docs e ajustes na ficha

- **Regras de combate** (`docs/game/combate/`, exibidas em outro lugar do site, não no painel "?"
  da ficha): efeitos de status com o diagrama de conjuntos (§5.2), a tabela tema → perícia, a tabela
  da guarda, os exemplos das notas (seção "Exemplos para as docs de jogo") e a correção do dash em
  `reacoes.md`. Sem tabela de alturas.
- **Princípios do sistema:** as duas linhas mestras (§1) e o gráfico da Biske (está nas notas, com
  o fim de D perto do meio da linha 4) — este último entra quando a Fadiga entrar.
- **Doc do mestre:** aplicar na mão, apelidos, visibilidade.
- **Ficha:** enriquecer a descrição de **Força de Vontade** com o sentido canônico (a vontade de
  lutar contra quem parece mais forte, sem assumir a própria pior condição — Biske ao Killua).
- **Perícias de Força** [dono]: **Forçar (Force) / Agarrar (Grab) / Carga (Charge)**. Forçar =
  empurrar, puxar, arrombar, pressionar. Carga = capacidade de carga **e** momentum: golpe
  carregado e acumular velocidade além da Aceleração (o bruto, como o Uvogin, que não freia num
  instante). Renomear `Push→Force` e `Carry→Charge` pede migração como a das armas (20261113) e
  paridade back/front. A mecânica de golpe carregado e investida é da tarefa de testes/actions.

## 13. Reservado, em espera e provisório

**Fadiga — em standby.** Não implementar. O que está de pé para a volta: status próprio, posto pelo
sistema a partir do SP (tudo o que piora a condição física tira SP); nada acima de 50%; Stamina
negativa como o HP; Força de Vontade para ficar acordado; o piso sai da própria rolagem; barras de
comprimento diferente por personagem; **sem valor fixo de modificador**. Fontes: gráfico da Biske e
fala do Morel ("de zero a 100… devo estar em 40… não, 35"), nas notas.

**Queda — regra definida, gatilho depois do movimento.** Dano = fator de altura + o que faltou no
pouso (ou − o que sobrou) − Defesa; ≤ 0 não machuca. Pouso: **Acrobacia**. Sem chance de reagir:
altura − Defesa. **Fator de altura é parâmetro** (valor inicial 3 por metro), nunca tabela; a
resistência receberá a parcela do Nen (saltos de 30 m sem dano).

**Para o spec dos testes e colisões — dano negativo volta para o atacante** [dono, 2026-11-15;
regra obrigatória, não implementar aqui]: se a defesa supera o dano (soco de 20 numa defesa de 50 →
−30), o excedente **volta para quem bateu**, que o reduz com um teste. Exemplos do dono, com teste
de 20 (valor 10 + dados 10): excedente 30 → recebe 10; excedente 40 → recebe 20. Depois, teste de
atordoamento sobre o que recebeu, e o membro que golpeou pode ganhar status (Lesionado, dormente).
**Erro crítico no acerto dobra o dano que volta** (10 → 20; 20 → 40); **acerto crítico o reduz à
metade** (10 → 5; 20 → 10). O dono sugeriu Tenacidade para reduzir, notando que algum teste se
repetiria. Nos exemplos, "excedente 40" é o soco de 20 numa defesa de 60 (dono confirmou a leitura).
**Direção preferida pelo dono (sugestão de Claude; fechar no spec dos testes):** tratar o que volta
como um golpe no atacante — **Defesa reduz, Tenacidade decide o atordoamento** —, a mesma ordem de
§6.2, sem repetir perícia.

**Para o spec dos testes — empilhar testes** [dono, 2026-11-15]: o mestre deve poder **empilhar**
testes na mesma pilha (corrente) de um teste, inclusive nos automáticos daqui. Não entra agora; o
dono tem um rascunho e prefere incrementar depois de status e testes fechados. O desenho dos
testes automáticos (§6) não deve impedir isso: cada teste é um item da pilha da action.

**TODO para a migração do Nen:** efeito do Atordoado/Inconsciente na aura (a aura ainda pode
protegê-lo); "fluxo quebrado" (técnicas ativas caem); o que continua sem o usuário é regra de cada
habilidade (materialização); Paralisado usando habilidade; Debilitado espiritual nos testes de Nen;
En tirando a dependência da Visão; declaração de ações do Controlado; habilidades e itens que
aplicam e curam status.

**Provisório até o movimento:** distância do Lento; rolar/deslizar/levantar do Caído; No ar no
salto; exceções do teste de equilíbrio; divisão do bônus pós-atordoamento.

**Outros futuros anotados:** biotipo (pesos no diagrama físico); mapa de local e volume do corpo
lesionado; ponderação para estimar a própria condição; configuração com dados reais (cada rolagem
automática custa caro nela — por isso poucas rolagens por round).

## 14. O que fica com quem constrói

- **Comentários e TODOs** [dono]: o que esta implementação tornar **contraditório** deve ser
  apagado ou corrigido (ex.: "não será persistido" em `character_status.go`). **Não** apagar
  comentário ou TODO fora deste escopo — a regra "NEVER remove TODO comments" existe porque sessões
  apagavam TODOs alheios. Logo no início, **reescrever essa regra no `AGENTS.md`** nesse sentido.

- A forma técnica das peças, da tabela e da lista de visibilidade; nomes em inglês no código
  (identificadores e títulos de teste em inglês; comentários e UI em português).
- Contratos WS/REST novos em `docs/dev/api/` e registro em `documentation-map.yaml`.
- O corte em fases. Sugestão: (1) base — tabela, peças, catálogo, aplicar/editar/tirar na mão com
  visibilidade; (2) crítico, HP negativo e morte; (3) testes automáticos no fechamento do turno e
  efeitos nas rolagens; (4) dano contínuo e recuperação fora de turno pelo relógio, asfixia;
  (5) front: exibição, histórico; (6) docs de jogo e renomeação das perícias de Força.
- **Configuração da mesa** (status de PJ públicos por padrão): o mecanismo de configurações de
  partida não existe; por ora é um parâmetro em `MatchRules` com o padrão deste documento, sem
  tela. [Claude]
- Parâmetros em `MatchRules`: fator de altura, ritmo da asfixia (1 por 10 s), 15 min de morte,
  desconto do teste de quem já sangra.
- Conferir no código, antes de planejar: o que a tela do mestre mostra hoje da redução por defesa;
  onde o HP é travado em zero; como a "condição" do mestre entra numa rolagem nova.

- **Processo entre implementação e spec** [dono, 2026-11-19]: cada PR de implementação traz, em
  português, uma seção **"Decisões tomadas"** (do dono e de quem implementou). A sessão de PO as
  incorpora a este spec (PT e EN no mesmo commit) quando o PR abre. A próxima implementação lê o
  spec, não o chat.

## 15. Para o dono revisar (decisões de Claude)

1. Atordoado = Paralisado + Atônito + sentidos anulados (§5.2); Atônito bloqueia agir e reagir.
2. Inconsciente acaba quando a causa cessa (HP ≥ 0, reanimação) ou pelo mestre.
3. Padrões de visibilidade que você não citou: Asfixiando e Guarda aberta públicos; Congelando,
   Lento, Paralisado e Atônito escondidos.
4. Ao revelar um status, o histórico passado dele aparece.
5. Combinações automáticas limitadas às de §5.5.
6. Lista de apelidos sugeridos (§8).
7. Reanimar e pressionar a ferida usam **Forçar** (o novo nome do Empurrão).
9. Penalidades (§4): vantagem e desvantagem se cancelam uma a uma.
11. Morto como efeito de status público que contém o Inconsciente (§7).
10. Tempo fora do round (§10): resolução em lote e aviso ao mestre antes de um salto de tempo com
   alguém perdendo HP.
8. Nomes: Inconsciente (não Desmaiado); status de sentido se chama "Visão anulada", e "Cego" é
   apelido; a família elemental inclui o tóxico, listado por extenso nas docs.

## 16. Decisões da implementação (fase 1)

> Tomadas nas sessões de implementação do back (PR #111 e o PR 2, de histórico). Onde houver
> conflito com as seções acima, **valem estas**. Fontes: o plano
> `docs/superpowers/plans/2026-11-17-status-effects-phase-1.md` (seções "Owner decisions",
> "Decisions this plan makes", "Deferred") e o contrato `docs/dev/api/match-status-effects.md`.

### 16.1 Do dono — 2026-11-17 (PR #111)

1. **Dano por round começa em 1** (Sangrando, Queimando, Congelando, Envenenado).
2. **Pontos de recuperação** (Atordoado, Paralisado, Atônito): na aplicação manual o mestre os
   digita; com o teste automático, saem da falha.
3. **`npc_added` vai só ao mestre.** O NPC no tabuleiro chega aos jogadores pela névoa, como
   qualquer peça.
4. **Status públicos dependem da névoa.** O jogador só recebe os status de um personagem enquanto
   o vê; o dono e o mestre sempre veem. Saiu da visão: o servidor não manda nada e o jogador fica
   com o último estado visto. Voltou: recebe a lista atual. Partida sem tabuleiro: todos veem. Com
   tabuleiro, personagem sem peça não é visto pelos outros.
5. **As barras do round** (ação e movimento) seguem a mesma visão.
6. **Escondido tira a peça do mapa** e esconde barras e status de quem não está na lista dos que o
   percebem.
7. **Inconsciente traz Caído**; o mestre pode tirar o Caído e deixar só o Inconsciente.
8. **Tato, face equilíbrio:** prejudicado dá a desvantagem/número do próprio status em ataque,
   defesa, esquiva e movimento; anulado dá uma desvantagem nesses quatro. Declarado agora; aplicado
   na fase das rolagens.
9. **Tudo o que não depende de outra implementação já é aplicado** (os bloqueios de ação e de
   reação).
10. Da regra nova de vantagem/desvantagem (§4), respondido para o plano: **teste passivo com N
    níveis** rola 2 + N dados e compara com o valor base, como hoje; **conjunto de D20**: um D20 a
    mais por nível.

### 16.2 Do dono — 2026-11-19 (PR 2: histórico e Escondido)

11. **Cada evento de status no histórico guarda a sua causa:** a ação ou a reação específica do
    turno. Por enquanto o mestre a informa, opcionalmente, ao aplicar, editar ou tirar com o turno
    aberto (por exemplo, pelo card da reação). Sem causa, o evento fica no turno, na ordem do
    horário; fora de turno, nos eventos do round. O teste, quando existir, preenche o mesmo campo,
    mais a identificação do próprio teste. **Nunca inferir pela "última reação aberta":** no
    fechamento todas estão abertas.
12. **Escondido oculta as ações do personagem**, e com elas os testes e os status ligados a essas
    ações, de quem não o percebe — ao vivo e no histórico.
13. **Nada fica oculto para sempre.** O mestre revela quando quiser, inclusive depois de a partida
    encerrar:
    - o período inteiro de um Escondido (ações, alvos e status daquele personagem no período) ou
      um status só;
    - a revelação vale para todos (por jogador é futuro) e cobre o que aconteceu até aquele instante;
    - fluxo: ao tirar o Escondido, o front pergunta ao mestre se revela o que ficou oculto; se não
      revelar, pode fazê-lo depois pelo card do evento.
14. **Ampliar quem percebe o personagem NÃO revela o passado.** Só o comando do mestre revela. No
    futuro, um teste de percepção poderá fazer um jogador perceber.
15. **Alvo Escondido:** para quem não o percebe, ele sai da lista de alvos e do resultado (ao vivo
    e no histórico) até ser revelado. Ele pode ter sido atingido sem querer.
16. **O resultado fechado do golpe vai a todos**, mesmo a quem não percebe o atacante, nesta fase.
17. Quem estava olhando quando o Escondido entrou **vê ele entrar** e depois não vê mais nada
    daquele personagem, mas mantém no histórico o que já tinha visto.
18. **Banco enxuto para o caso raro:** pouquíssimas ações e status ficarão de fato escondidos.

### 16.3 Do dono — 2026-11-20 (PR #112)

19. **Ação cancelada no histórico:** só o dono do personagem e o mestre veem a entrada; os outros
    jogadores não recebem nada.
20. **Alvo Escondido não deixa rastro** para quem não o percebe: nem identificação, nem números, nem
    reação, nem posição. Se o golpe o atingiu, o resultado traz só quantos alvos ocultos foram
    atingidos, e a tela mostra "Desconhecido 1", "Desconhecido 2"… Se não atingiu, não aparece
    nada. A margem do ataque também some para esse leitor. Depois que o mestre revela, aparece o
    resultado real.
21. **Desfazer no mesmo turno, com "Devolver?":** quando o mestre tira um status no mesmo turno em
    que o aplicou, é um **desfazer** ("como se nunca tivesse existido"), não uma revelação. Se o
    status tinha tirado ações da fila ou trocado uma reação, a tela pergunta ao mestre se devolve e
    lista o que volta. Devolvido, tudo volta igual: mesma ação, mesmo lugar na fila, mesmos dados e
    as edições do mestre. O que não puder voltar chega ao mestre com o motivo em português. Fora do
    turno, ou depois do fechamento, não há pergunta. Um futuro "desfazer" geral (ctrl+Z) usa este
    mesmo caminho.
22. **As recusas por status dizem o nome do status em português** (ex.: "Paralisado").
23. **Revelar depois que o NPC sai da partida:** fica como está por enquanto.
24. **Barras de Vida e Stamina para outros jogadores** [2026-11-21]: entram junto com a
    **visibilidade por jogador** (qual personagem viu quem estava Escondido etc.), na mesma
    sessão. Hoje a Vida só vai ao dono e ao mestre, a Stamina não chega ao vivo, e o mestre não tem
    como revelar barras: essas três lacunas fecham nela.

25. **Revelar barras de status** [2026-11-21; regra registrada, o refinamento da visibilidade por
    jogador ainda virá]: **por personagem**, com um **atalho para revelar as de todos os personagens
    de jogador a todos os jogadores**. **NPCs são revelados um por um** — o atalho não os inclui.
    Revelar todos os NPCs de uma vez, se existir, é outro botão e vai para a **configuração da
    partida**.
26. **A Stamina segue a mesma regra de quem vê da Vida** [2026-11-21].
27. **Golpe no vazio** [2026-11-21]: um ataque cujo único alvo é um Escondido que esquivou aparece,
    para quem não percebe o alvo, como ataque sem alvo visível. Já funciona assim.

### 16.4 Do dono — 2026-11-24 (fase 2: rolagens, HP negativo e morte)

28. **O crítico dobra todo o poder somado aos dados**, não só a perícia: crítico = (dados + valor de
    teste + todos os números somados ao teste) × 2; erro crítico = o mesmo ÷ 2, para baixo. Ex.:
    dados 10 + 10, perícia 8, Corpo debilitado −3 → (20 + 8 − 3) × 2 = **50**.
29. **Quem morre num golpe grande fica com o HP onde morreu.** HP 10, máximo negativo 16, golpe de
    50 → HP **−40**. A barra mostrará, quando desenhada, o mínimo do personagem (−16) e o valor além
    dele (o −17, o −40). O máximo negativo pode ser lido da ficha.
30. **Desfazer a morte = o mestre remover o status Morto**, depois do fechamento do turno. (Dentro do
    turno não há morte para desfazer: o cálculo do mestre só avisa "morre", e se ele mudar o golpe
    antes de fechar, o personagem não morre.) Ao remover, **o mestre pode informar o HP com que o
    personagem volta; sem informar, o HP sobe para o máximo negativo** (−16 no exemplo). Um comando
    para o mestre mexer no HP direto virá depois. Reanimação por habilidade Nen: TODO.
31. **Morte na campanha:** a morte pela regra preenche a marca de morto da ficha (a tag "Morto" e o
    card escurecido, que a campanha já usa) e o status do morto fica explícito. **A marca só entra na
    campanha quando a partida for encerrada**, não na hora. Desfazer a morte limpa a marca.
32. **A morte segue a névoa na mesa**, como os outros status públicos (§16.1, item 4): quem não vê a
    peça só recebe o Morto quando ela entrar na vista; sem mapa, todos veem na hora. O "todos veem"
    do §7 deve ser lido assim.
33. **Caído pesa nos três escapes** (o escape e o escape com guarda, que usam Dash, e o escape
    fechado, que usa Shift), com desvantagem. **Provisório:** o movimento ainda está sendo desenhado
    como um todo; "rolar sem penalidade" é só um ponto possível desse desenho, não uma regra. Deixar
    TODO no código, na peça do Caído e onde o escape lê a desvantagem.
34. **Guarda aberta: o escape tem desvantagem nos dois testes**, na esquiva e no movimento, um nível
    em cada (o escape só passa se os dois passarem).
35. **Visão anulada pesa no ataque, na defesa e na esquiva** (reflexo, evasão e a esquiva dos
    escapes), um nível de desvantagem por teste.
36. **O PR do front da fase 2** traz o mínimo da barra de HP no motor da ficha (paridade) e as
    marcas "crítico ×2" e "morre" no cálculo do mestre, para a regra ser validada na mesa. O resto do
    visual fica com a sessão de PD.

37. **Caído no escape: desvantagem nos dois testes**, um nível na esquiva e um no movimento, como
    na Guarda aberta (item 34). Vale para os três escapes; provisório até o movimento (mesmo TODO do
    item 33).
38. **Teste de Tato declarado numa ação conta como a face toque** [provisório; proposta de Claude
    aceita pelo dono]. Pesa se o toque estiver afetado (prejudicado: a desvantagem e/ou o número do
    status; anulado: o teste falha sozinho). O equilíbrio não pesa nesse teste, porque já pesa por
    ataque, defesa, esquiva e movimento (§16.1, item 8); a preempção não tem teste hoje. Ex.: Tato
    prejudicado só no equilíbrio, tateando uma fechadura → sem penalidade; com o toque prejudicado →
    com penalidade. Se a narração for de equilíbrio (a corda bamba), o mestre compensa com a
    condição até o spec dos testes, que deixa o teste dizer a face — no teste pedido pelo mestre isso
    já está decidido (`2026-11-15-testes-brainstorm-notas.md`, P1); na corrente de testes de uma
    action, será decidido lá.

### 16.5 De quem implementou (o dono foi informado)

- Reaplicar um status que o personagem já tem é recusado; o mestre edita o existente. Vários
  personagens de uma vez: tudo ou nada.
- A lista de quem vê é guardada por efeito (vazia = o padrão do tipo); o jogador nunca vê essa
  lista.
- Revelar é uma chamada HTTP só do mestre; funciona durante e depois da partida, e o front
  recarrega o histórico. Um aviso pelo WebSocket pode vir depois.
- No histórico, vê um evento de status quem enxergava o personagem logo antes ou logo depois da
  mudança, mais o dono.
- Contenção herda o que um status **faz**, não como ele **acaba** (o Inconsciente não tem a
  recuperação por Tenacidade).
- O resto (campos obrigatórios, mapeamento reação ↔ peça, efeito na fila e nas reações
  pendentes, limites de texto) está no plano, "Decisions this plan makes".

## 17. O que vem depois e molda o modelo

Quem implementar qualquer fase precisa saber disto de saída, porque decide a forma dos dados.

- **A maioria dos status virá de testes** [dono, 2026-11-19]. O brainstorm dos testes está em
  andamento (`2026-11-15-testes-brainstorm-notas.md`). Muitos sairão das **colisões entre a ação e
  as reações**, então um mesmo turno aplica status **em vários momentos e em vários alvos** (várias
  reações, vários alvos). Testes **fora de batalha** também aplicarão status. Os status manuais do
  mestre continuam existindo.
- **O sistema saberá a causa de cada status** quando os testes existirem. O campo de causa (§16.2,
  item 11) já nasce para isso.
- **Revelação:** vale também para os **testes ocultos** (o teste secreto do mestre, a percepção
  contra Escondido).
- **Quem vê será por jogador.** Um jogador que passa no teste de percepção percebe o Escondido e os
  outros não; a revelação por jogador é a mesma lista. O modelo já é lista; falta o comando.
- **Barras de Vida e Stamina sobre a peça**, com revelação pelo mestre (§11): entram com a
  visibilidade por jogador (§16.3, item 24).
- **Empilhar testes** (§13) só acrescenta testes à pilha de uma action; não muda a ordem.
