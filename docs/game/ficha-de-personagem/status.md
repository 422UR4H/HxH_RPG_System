# Status (Barras de Status)

## Visão Geral

O sistema de status representa os recursos vitais do personagem: vida (HP), estamina (SP) e aura (AP). Cada barra é calculada dinamicamente a partir das habilidades, atributos e perícias do personagem.

## Pontos de Vida — HP (Health Points)

**Fórmula:**
```
HP_máx = 20 + (nível_vitalidade + valor_resistência) × bônus_físicos
```

- **Base:** 20 pontos fixos
- **Coeficiente:** nível da perícia Vitalidade + valor do atributo Resistência
- **Bônus:** bônus da habilidade Físicos

Quando o personagem está com vida cheia, um upgrade mantém a vida no máximo novo.

## Pontos de Estamina — SP (Stamina Points)

**Fórmula:**
```
SP_máx = 10 × (nível_energia + valor_resistência) × bônus_físicos
```

- **Coeficiente:** 10 (multiplicador)
- **Fatores:** nível da perícia Energia + valor do atributo Resistência
- **Bônus:** bônus da habilidade Físicos

## Pontos de Aura — AP (Aura Points)

**Fórmula:**
```
AP_máx = 10 × (nível_mop + nível_consciência) × bônus_espirituais
```

- **Coeficiente:** 10 (multiplicador)
- **Fatores:** nível da perícia MOP + nível do atributo Consciência
- **Bônus:** bônus da habilidade Espirituais (convertido para inteiro)

> **Nota:** AP só existe para personagens com habilidade Espiritual ativa. Se `spirituals` for nil, a barra de aura não é criada.

## Mecânica das Barras

Cada barra possui três valores:
- **Mínimo (min):** limite inferior (geralmente 0)
- **Atual (curr):** valor corrente
- **Máximo (max):** limite superior calculado

### Operações
- **IncreaseAt(valor):** Aumenta o atual, limitado pelo máximo
- **DecreaseAt(valor):** Diminui o atual, limitado pelo mínimo
- **SetCurrent(valor):** Define o atual (erro se fora dos limites)
- **Upgrade():** Recalcula o máximo; se estava cheio, o atual acompanha
