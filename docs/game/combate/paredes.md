# Paredes, Portas e Obstáculos

As paredes do mapa tático são alvos válidos para ações de jogadores. Toda ação de interação com paredes entra na **fila de prioridade** e é aberta pelo mestre — respeitando a ordem de iniciativa.

## Tipos de Parede

| Tipo | Descrição |
|---|---|
| `wall` / `terrain` | Parede sólida ou terreno. Pode ser atacada (se tiver HP). |
| `door` | Porta. Pode ser aberta, fechada ou atacada. |
| `window` | Janela. Comporta-se como porta para fins de ação. |
| `secret_door` | Porta secreta. Invisível para jogadores até o mestre revelá-la; revelada, é uma porta como as outras (abrir, fechar, trancar, arrombar). |

## Ações Disponíveis por Tipo

| Tipo de parede | Ações visíveis para o jogador |
|---|---|
| `wall` / `terrain` | "Atacar" (se `maxHp > 0` e não destruída) |
| `door` (fechada, destrancada) | "Abrir", "Atacar" |
| `door` (trancada) | "Arrombar fechadura", "Atacar" |
| `window` | "Abrir", "Atacar" |

## Alcance

Para **abrir, fechar ou arrombar**, o personagem precisa estar **encostado** na parede: numa das
casas que tocam nela, de qualquer um dos lados. Se está longe, a própria ação pode incluir o
movimento até lá (ação combinada). Sem estar encostado, a ação é recusada.

**Atacar** uma parede não exige estar encostado — um ataque a distância continua valendo.

Uma habilidade que permita agir numa parede de longe passa pela mão do mestre.

## Tranca

- Um personagem **não abre** uma porta ou janela trancada: a tentativa não muda nada.
- **Fechar** funciona sempre, trancada ou não.
- **Arrombar fechadura** ainda não rola teste (a corrente de testes de perícia não existe): a
  tentativa é declarada e o mestre decide. Se deu certo, ele destranca.
- O **mestre** abre e fecha qualquer passagem, trancada ou não, e **tranca e destranca** portas
  e janelas.

## A mão do mestre

Sem NPC escolhido, o mestre toca em **qualquer** parede e vê a vida dela (ou que é indestrutível,
ou que está destruída). Ele abre, fecha, tranca, destranca, revela porta secreta e **destrói**
paredes na hora — uma ou várias de uma vez (um terremoto). Destruir não é ataque: a parede
indestrutível também cai. Com um NPC escolhido, o mestre age só através do NPC.

## Parede destruída

Vira entulho: não bloqueia movimento nem visão, e não aceita mais nenhuma ação (abrir, fechar,
arrombar, atacar).

## Quando o dano cai

O dano de um ataque numa parede cai no **fechamento** do turno, junto com o dano nos personagens.
Enquanto o turno está aberto, o mestre vê no cálculo quanto cada parede vai perder. Uma parede
indestrutível pode estar entre os alvos de um golpe em área — ela só não recebe dano.

## Resistência e Dano

Cada parede tem:
- **HP** — pontos de vida atuais
- **HP Máximo** — `0` significa indestrutível
- **Resistência** — subtrai do dano bruto do ataque

Fórmula: `dano efetivo = max(0, dano bruto − resistência)`

Se o dano bruto não ultrapassar a resistência, a parede não é danificada. O excesso retorna como **dano rebote** para o atacante (apenas ataques corpo-a-corpo — regra completa em implementação futura).

## Estados Visuais

| Estado | Condição | Visual |
|---|---|---|
| Intacta | `hp == maxHp` | Cor cheia, opacidade 1.0 |
| Danificada | `0 < hp < maxHp` | Tracejada, opacidade 0.8 |
| Destruída | `destroyed == true` | Pontilhada fina, opacidade 0.4, marcas × |
| Indestrutível | `maxHp == 0` | Cor cheia, opacidade 1.0 (sem barra de HP) |

> 🔧 Para Desenvolvedores: `docs/dev/match/turns-rounds.md` — fluxo de Action + TurnResolver.
