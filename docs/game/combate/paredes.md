# Paredes, Portas e Obstáculos

As paredes do mapa tático são alvos válidos para ações de jogadores. Toda ação de interação com paredes entra na **fila de prioridade** e é aberta pelo mestre — respeitando a ordem de iniciativa.

## Tipos de Parede

| Tipo | Descrição |
|---|---|
| `wall` / `terrain` | Parede sólida ou terreno. Pode ser atacada (se tiver HP). |
| `door` | Porta. Pode ser aberta, fechada ou atacada. |
| `window` | Janela. Comporta-se como porta para fins de ação. |
| `secret_door` | Porta secreta. Invisível para jogadores — não aparece no menu de ação. |

## Ações Disponíveis por Tipo

| Tipo de parede | Ações visíveis para o jogador |
|---|---|
| `wall` / `terrain` | "Atacar" (se `maxHp > 0` e não destruída) |
| `door` (fechada, destrancada) | "Abrir", "Atacar" |
| `door` (trancada) | "Arrombar fechadura", "Atacar" |
| `window` | "Abrir", "Atacar" |

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
