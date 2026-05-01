# Dados

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
| D20  | 20    | Testes gerais, testes de habilidade |
| D100 | 100   | Probabilidades especiais, tabelas de evento |

## Mecânica de Rolagem

### Geração de Resultado

Ao rolar um dado, o sistema gera um número aleatório entre 1 e o número de lados do dado. O resultado fica registrado até a próxima rolagem. Um resultado zero significa que o dado ainda não foi rolado.

### Combinação de Dados

Armas e ações utilizam múltiplos dados combinados. O resultado final é a **soma de todos os dados** na rolagem.

**Exemplo:** Uma Espada usa D10 + D4. Ao atacar, rola-se ambos e soma-se os resultados.

## Dados por Arma

Cada arma possui um conjunto específico de dados que representa seu potencial de dano variável. Armas mais pesadas e complexas tendem a usar mais dados e/ou dados maiores.

Exemplos:
- **Adaga:** 1×D8 — ataque rápido, dano consistente
- **Espada Longa:** D12 + D10 + D4 — alto potencial, variável
- **Martelo de Guerra:** D12 + D12 + D6 — devastador

Consulte o documento de [Armas](armas.md) para a lista completa.

## 🎲 Curiosidade: Aleatoriedade Verdadeira

O sistema de dados do HxH RPG utiliza um gerador de números aleatórios **criptograficamente seguro** — o mesmo tipo de tecnologia usada em sistemas bancários e de segurança digital. Isso significa que cada rolagem produz resultados **verdadeiramente aleatórios**: sem padrões, sem sequências previsíveis, sem possibilidade de manipulação.

Em cenários extremamente raros (praticamente impossíveis em condições normais), o sistema possui uma camada de segurança que recorre a um gerador pseudo-aleatório como alternativa. Na prática, isso quase nunca acontece.

**Por que isso importa para você, jogador?** Fairness total. Nenhum jogador é favorecido ou prejudicado por padrões ocultos nos dados. Cada rolagem é independente e genuinamente imprevisível — exatamente como dados reais bem balanceados, mas com a garantia matemática de um sistema digital.

---

> **🔧 Para Desenvolvedores**
>
> Implementação técnica: [`docs/dev/weapons-dice.md`](../dev/weapons-dice.md)
> Código-fonte: `internal/domain/entity/die/`
