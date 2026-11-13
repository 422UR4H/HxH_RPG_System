# Status (Barras de Status)

## Visão Geral

As barras de status são os recursos vitais do personagem: Vida (HP), Energia/Stamina (SP) e Aura (AP). O máximo de cada barra vem das aptidões, dos atributos e das perícias do personagem.

## Vida — HP (Health Points)

**Fórmula:**

> **HP máximo** = 20 + (nível da Vitalidade + valor da Resistência) × bônus da Aptidão Física

- **Base:** 20 pontos fixos
- **Coeficiente:** nível da perícia Vitalidade + valor do atributo Resistência
- **Bônus:** bônus da [Aptidão Física](./aptidoes.md#bonus-de-aptidao)

Quando o personagem está com vida cheia, uma evolução mantém a vida no máximo novo.

## Stamina — SP (Stamina Points)

**Fórmula:**

> **SP máximo** = 10 × [(nível da perícia Energia + valor da Resistência) × bônus da Aptidão Física]

- **Coeficiente:** 10 (multiplicador)
- **Fatores:** nível da perícia Energia + valor do atributo Resistência
- **Bônus:** bônus da [Aptidão Física](./aptidoes.md#bonus-de-aptidao)

A conta entre colchetes é arredondada para baixo antes de multiplicar por 10. Por isso, o SP máximo é sempre um múltiplo de 10.

## Aura — AP (Aura Points)

**Fórmula:**

> **AP máximo** = 10 × (nível do MOP + nível do Nen da Consciência) × bônus da Aptidão Espiritual

- **Coeficiente:** 10 (multiplicador)
- **Fatores:** nível da perícia MOP + nível do atributo Nen da Consciência
- **Bônus:** bônus da [Aptidão Espiritual](./aptidoes.md#bonus-de-aptidao), arredondado para baixo antes de entrar na conta

> **Nota:** AP só existe para personagens que despertaram o Nen. Se o personagem não tiver despertado o Nen, a barra de aura não é criada.

## Mecânica das Barras

Cada barra possui três valores:
- **Mínimo:** limite inferior (geralmente 0)
- **Atual:** valor corrente
- **Máximo:** limite superior calculado

### Como as Barras Funcionam

- **Recuperar:** o personagem pode recuperar pontos até o valor máximo da barra
- **Perder:** o personagem perde pontos ao receber dano, gastar Stamina ou usar aura, até o valor mínimo da barra
- **Definir valor:** em situações especiais, o valor atual pode ser definido diretamente, desde que esteja dentro dos limites da barra
- **Evolução:** quando o personagem evolui, o máximo da barra é recalculado; se a barra estava cheia antes da evolução, o valor atual acompanha o novo máximo

---

> **🔧 Para Desenvolvedores**
>
> Implementação técnica: [`docs/dev/character-sheet/status.md`](../../dev/character-sheet/status.md)
> Código-fonte: `internal/domain/entity/character_sheet/status/`
