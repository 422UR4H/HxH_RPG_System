# Dados (Dice)

> Sistema de rolagem de dados do HxH RPG.

## Tipos de Dados

O sistema utiliza 7 tipos de dados poliédricos:

| Dado | Lados | Uso Comum |
|------|-------|-----------|
| D4   | 4     | Dano menor, efeitos leves |
| D6   | 6     | Dano básico, ataques desarmados |
| D8   | 8     | Dano médio, armas leves |
| D10  | 10    | Dano forte, armas intermediárias |
| D12  | 12    | Dano pesado, armas de grande impacto |
| D20  | 20    | Testes gerais, checks de habilidade |
| D100 | 100   | Probabilidades especiais, tabelas de evento |

## Mecânica de Rolagem

### Geração de Resultado

O sistema utiliza aleatoriedade criptograficamente segura (`crypto/rand`) para gerar resultados. Em caso de falha (extremamente raro), realiza fallback para `math/rand/v2`.

- **Intervalo:** sempre `[1, N]` onde N é o número de lados
- **Estado:** o dado armazena o último resultado rolado
- **Resultado zero:** indica que o dado ainda não foi rolado

### Combinação de Dados

Armas e ações utilizam múltiplos dados combinados. O resultado final é a **soma de todos os dados** na rolagem.

**Exemplo:** Uma Espada (Sword) tem dados `[D10, D4]`. Ao atacar, rola-se ambos e soma-se os resultados.

## Dados por Arma

Cada arma possui um conjunto específico de dados que representa seu potencial de dano variável. Armas mais pesadas e complexas tendem a usar mais dados e/ou dados maiores.

Exemplos:
- **Adaga (Dagger):** 1×D8 — ataque rápido, dano consistente
- **Espada Longa (Longsword):** D12 + D10 + D4 — alto potencial, variável
- **Martelo de Guerra (Warhammer):** D12 + D12 + D6 — devastador

Consulte o documento de [Armas](armas.md) para a lista completa.
