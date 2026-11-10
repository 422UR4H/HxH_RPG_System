# O Relógio da História

Toda partida tem um **relógio da história**: o horário que é *dentro do mundo do jogo*, com data e
hora até o segundo. Ele nunca é o horário do mundo real. É o mestre quem o controla; os jogadores
só o veem — e **toda mudança que o mestre faz nele aparece para a mesa**, nunca passa batida.

## Como a partida começa

- Ao criar a partida, o mestre pode informar a **hora de início** na história, ao lado da data.
- Quando a partida começa, o relógio nasce nessa data e hora, **parado**.
- Se o mestre não informou a hora, ele precisa dizê-la assim que entra na mesa (a data é a da
  partida). Até lá os jogadores veem "Hora a definir" e o mestre não consegue abrir ações, trocar
  de cena ou de regime, nem encerrar a partida. Os jogadores continuam podendo declarar ações.

## Nunca voltar antes do que já aconteceu

O relógio **não pode voltar** para antes de:

- a última ação que de fato aconteceu (o limite sobe quando um turno **abre** e quando fecha);
- a última troca de cena;
- a última troca de regime (livre ou disputado);
- o início da partida e cada rodada disputada.

O mestre vê, no painel, qual é o limite e por quê. Tempo real correndo não sobe o limite: o mestre
pode voltar o tempo que deixou correr, até o limite.

## Mexendo no relógio

O mestre pode **avançar ou recuar** o relógio por um valor ("+1 hora", "−15 minutos") ou **ir direto
a um horário**. Funciona com o relógio parado ou correndo.

Avançar **mais de 2 dias** de uma vez pede confirmação, porque os personagens perdem os dias de
treino desse período.

## Tempo real

Em cenas de conversa e exploração, o mestre pode **ligar o tempo real**: o relógio passa a correr
junto com o relógio do mundo, um segundo da história por segundo real.

- Só existe no regime **livre**. No regime **disputado** (corrida de ações) não dá para ligar.
- O relógio **sempre começa parado**; o mestre liga quando quiser.
- Ele **para sozinho** quando: o regime muda, a cena muda, o mestre sai da mesa ou a partida
  termina. Se o mestre sai e volta, precisa ligar de novo.
- Se o servidor cair com o tempo real ligado, o relógio volta parado no último ponto registrado
  (no máximo 1 minuto antes): o tempo em que nada estava rodando nunca conta.

## Uma rodada, um segundo

Cada **rodada disputada** em que ao menos uma ação foi aberta **vale 1 segundo** do relógio, somado
quando a rodada termina. Isso acontece ao vivo, junto com o combate, e o mestre não pode desfazer
esse mínimo. Rodada em que nenhuma ação foi aberta não conta, e os jogadores não recebem aviso por
rodada — apenas veem o horário andar.

## Hiatos

Às vezes uma rodada disputada termina porque **todos já agiram e ninguém declarou a próxima
ação**, e a história segue com uma nova rodada. Esse intervalo é um **hiato**: ninguém disputou
ações ali — os personagens podem ter conversado, se reposicionado ou procurado uma brecha.

O jogo não sabe quanto tempo o hiato durou; **o mestre se lembra e soma** esse tempo ao relógio. Ao
encerrar o combate, o mestre vê a lista de hiatos (a última ação antes e a primeira depois de cada
um) para acertar o horário. Uma rodada que termina com ações ainda esperando na fila não é hiato, e
a rodada que termina quando o mestre troca de cena logo em seguida também não: ali o combate acabou.

## Encerrando um combate

Um combate termina quando o mestre **troca de cena**. Nesse momento ele vê quantas rodadas foram
contadas e quantos segundos isso deu (já somados), os hiatos, e informa o **horário exato do fim
do combate** — que não pode ser antes do mínimo (o limite, mais o segundo da rodada em andamento).
Pode ainda escrever uma **descrição final** da cena. Trocar de cena de conversa também permite a
descrição final, mas sem resumo de combate.

## Encerrando a partida

- Só o mestre encerra, e **não dá para desfazer**.
- Não dá para encerrar com um combate em andamento (troque de cena antes), com uma ação aberta, nem
  sem o relógio ter hora.
- O mestre informa o **horário exato do fim** da partida (obrigatório) e, se quiser, um resumo final.
- Ao encerrar, a data do fim passa a ser a "data atual da história" da **campanha** e de **todas as
  fichas que participaram** (jogadores e NPCs), sem nunca voltar para uma data anterior à que já
  tinham — uma partida jogada fora de ordem não puxa a campanha para trás.
- Todos saem da mesa para a página da partida, que passa a constar como encerrada.

## Limites desta versão

- Não existe voltar no tempo por habilidade de personagem.
- Efeitos "por rodada" (como sangramento) usarão a conversão 1 rodada = 1 segundo; eles ainda não
  existem.

---

> **🔧 Para Desenvolvedores**
>
> Contrato do protocolo: [`docs/dev/api/match-clock.md`](../dev/api/match-clock.md) ·
> Desenho e decisões: [`docs/dev/match/clock.md`](../dev/match/clock.md)
> Código-fonte: `internal/domain/match/entity/clock/`
