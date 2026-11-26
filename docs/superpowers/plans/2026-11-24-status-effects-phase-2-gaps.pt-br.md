# Efeitos de status — fase 2 (back): lacunas antes do plano

> Sessão de planejamento da fase 2, 2026-11-24. Spec lido: `2026-11-15-status-effects-design.pt-br.md`
> (branch `docs/status-effects-spec`, PR #110, seções 16 e 17 valendo sobre as anteriores).
> O plano só será escrito depois das respostas das lacunas A–F. Cada uma traz a recomendação
> de Claude.

## 1. O que o spec §14 mandou conferir no código

1. **A tela do mestre já mostra a redução pela defesa.** O cálculo do turno manda, por alvo,
   `rawDamage`, `defenseApplied` e `projectedDamage`, e o painel do mestre mostra
   `dano 10 → 8 (defesa −2)` (`ResolutionDetails.tsx`). Falta só o que a fase 2 cria: o dobro do
   crítico e a morte. Hoje esse "dano final" já é o número que sai do HP.
2. **Onde o HP é travado em zero:** em um lugar só, na barra (`status.Bar.DecreaseAt` não desce
   do mínimo, e o mínimo da barra de HP é sempre 0 — o `TODO: Implement Min for hit_points` em
   `health_points_bar.go`). O dano em si (`EffectiveDamage`) não fica negativo, o que está certo.
   O banco já tem a coluna `health_min_pts`.
3. **Como a condição do mestre entra numa rolagem:** cada teste guarda a condição do mestre
   (`bias` = vantagem/desvantagem, `modifier` = número). Hoje a vantagem rola um segundo
   conjunto inteiro e fica com o melhor; **o tamanho do viés não conta** (+1 e +3 dão o mesmo).
   A regra nova muda isso, e a forma dos dados guardados muda junto (ver decisão 1 abaixo).

**Fórmula do HP negativo conferida** contra os exemplos do spec: HP 20 → 14 ou 15, HP 30 → 17,
HP 51 → 21, HP 75 → 25 (as fichas de paridade batem).

**Achado:** as perícias avulsas de uma ação (`skills[]`, por exemplo Visão ou Furtividade) nunca
tiveram o total calculado pelo servidor; ele guarda só os dados e o valor. São os únicos testes
de sentido que existem hoje. Para os efeitos de sentido e o Debilitado pesarem neles, a fase 2
passa a calcular esse total (ver decisão 4).

## 2. Lacunas que mudam regra de jogo — para a sessão de PO

**A. O crítico dobra também os modificadores?** O spec diz "(dados + valor de teste) × 2". Os
números que se somam ao teste (o número da condição do mestre, a reserva da esquiva fechada, o
−3 de um Debilitado, o −2 do Lento) ficam dentro ou fora da conta?
Ex.: dados 10+10, perícia 8, Debilitado −3 → fora: (20+8)×2−3 = **53**; dentro: (20+8−3)×2 = **50**.
Erro crítico, dados 1+1: fora: (2+8)÷2−3 = **2**; dentro: (2+8−3)÷2 = **3**.
*Recomendação:* **fora** — é a leitura literal, e o crítico mede a sorte e a perícia, não o
ajuste da cena.

**B. Onde fica o HP de quem morre num golpe grande?** Ex.: HP 10, máximo negativo 16, golpe de
50. O spec diz "o HP fica onde ele morreu". Fica em **−40** (o valor exato do golpe) ou para em
**−17** (o primeiro ponto além do máximo)?
*Recomendação:* **−40**, o valor exato: é "onde morreu", e o mestre que desfaz a morte vê o tamanho
real do golpe. A barra continua mostrando o mínimo de **−16**; só o morto pode ficar abaixo dele.

**C. Ao desfazer a morte, o HP muda?** Hoje não existe nenhum jeito de o mestre mexer no HP
(nenhum comando, nenhuma cura). Se desfazer a morte deixa o HP em −40, o personagem continua além
do limite e **qualquer dano seguinte o mata de novo**, sem o mestre poder evitar.
*Recomendação:* desfazer a morte **sobe o HP para o máximo negativo** (−16 no exemplo): vivo, no
fundo do poço. Um comando de editar o HP fica para quando a cura existir.

**D. A morte na partida aparece fora dela?** A campanha já mostra o NPC morto (tag "Morto", card
escurecido) por uma marca da ficha (`deadAt`), que hoje só é preenchida à mão no banco. O spec
manda a morte aparecer "como o NPC morto já aparece na campanha".
*Recomendação:* **sim** — a morte pela regra preenche a mesma marca no fechamento do turno, e
desfazer a limpa. Uma verdade só: o status Morto na partida, a marca fora dela.

**E. Caído e o escape.** A tabela da guarda dá "esquiva por reflexo com desvantagem" ao Caído, mas
na coluna Escapar diz "só rolando ou deslizando, **sem penalidade extra**". O escape usa o mesmo
teste de Reflexo da esquiva. A desvantagem do Caído pesa no Reflexo do escape?
*Recomendação:* **não pesa**: "sem penalidade extra" quer dizer nenhuma. A desvantagem vale para a
reação padrão, a esquiva e a esquiva fechada. (Restringir o escape a rolar/deslizar continua
esperando o movimento.)

**F. "Escapa com desvantagem" (Guarda aberta): em qual teste?** O escape tem dois testes, e os dois
precisam passar: a esquiva (Reflexo) e o movimento.
*Recomendação:* **nos dois**, um nível cada: a desvantagem é do escape inteiro, como o próprio
escape é "esquiva E movimento".

## 2b. Respostas do dono (2026-11-24, ainda não levadas ao PO)

- **A:** os números somados ao teste **dobram junto**: crítico = (dados + valor de teste +
  modificadores) × 2; erro crítico = o mesmo ÷ 2, para baixo.
- **B:** o HP fica no valor exato (−40). O front desenhará o valor abaixo do mínimo na própria
  barra; o máximo negativo sai da ficha.
- **C:** em aberto — o mestre poderá mexer no HP direto no futuro; ao remover a morte o HP muda,
  "depende do caso". Claude perguntou qual caso (ver resposta no chat).
- **D:** sim, preencher a marca de morto da ficha; o status Morto fica explícito.
- **E:** por enquanto **a desvantagem pesa** no escape, com TODO: o movimento (rolar, deslizar)
  vai decidir por tipo de escape. Correção do dono a conferir: existe escape com Shift (o escape
  fechado); ver pergunta no chat.
- **F:** nos dois testes do escape.
- **G:** entra no PR do front (o dono repassa ao PO).

## 3. Decisões que Claude vai tomar (leituras do spec; vetar se discordar)

1. **Dados guardados.** Cada teste guarda o conjunto base e os dados extras que um viés pediu, e
   nunca rola de novo: se o mestre dá +2 depois, caem só os 2 dados que faltam, uma vez. No fio,
   `attempts.secondary` some e entra `attempts.extra`. O front só lê `primary` hoje.
2. **Domínio → teste.** Ataque = o teste de acerto (e a finta); defesa = a defesa padrão e o
   repelir; esquiva = o Reflexo e a Evasão (também a esquiva do escape); movimento = a velocidade
   do movimento (deslocamento e escapes). A velocidade da ação (Legerity) e o dano **não** são
   testes de domínio. Isso vale para Lesionado machucado, Visão anulada e Tato/equilíbrio.
3. **Visão anulada** pesa em ataque e defesa como domínios: **não** na esquiva (o spec lista a
   esquiva à parte quando a quer, como no Tato).
4. **Debilitado pesa em todo teste das perícias da Aptidão**, inclusive a velocidade da ação, a do
   movimento e as perícias avulsas, que passam a ter o total calculado (só o mestre o vê até o
   fechamento, como o resto do cálculo).
5. **"Falha sozinho"**: o teste não rola, sai com total 0 e a marca `autoFailed`, sem crítico.
   Atordoado e Inconsciente já não podem agir; a falha dos sentidos deles fica pronta para os
   testes que vierem.
6. **Crítico em qualquer teste, inclusive as velocidades** (a da ação muda a ordem da fila).
   Teste passivo não tem crítico, salvo quando um viés o faz rolar e os dados contam (como hoje).
   D20: crítico = o dado que fica é 20; erro = é 1.
7. **Dano do crítico**: dobra o dano final de cada alvo da cadeia (o que passa adiante não dobra) e
   o dano final na parede.
8. **Morto**: entra pelo sistema no fechamento do turno; dentro do turno, o cálculo do mestre diz
   quem morreria. Público **seguindo a névoa**, como todo status público (§16.1 item 4 vale sobre
   o "todos veem" do §7). Contém o Inconsciente (não age, não reage, sentidos falham). Não traz o
   Caído junto. Só o mestre o tira; ele não aplica Morto à mão nesta fase.
9. **HP negativo sem teste automático**: até os testes da fase seguinte, quem fica abaixo de 0 não
   desmaia nem sangra sozinho; o mestre aplica o Inconsciente à mão.
10. **Viés do mestre sem teto de regra**, com um limite técnico de 10 níveis por teste (para não
    rolar dados sem fim).

## 4. Para o dono (escopo, não regra)

**G. O que entra no PR do front.** Obrigatório: o mínimo da barra de HP no motor da ficha
(paridade). Recomendo também o mínimo para validar a regra na mesa: no cálculo do mestre, as
marcas "crítico ×2" no dano e "morre". O resto do visual fica com a sessão de PD.
