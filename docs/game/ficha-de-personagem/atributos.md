# Atributos

## Visão Geral

Atributos representam as características fundamentais de um personagem no HxH RPG System. Eles são divididos em três categorias — **Físicos**, **Mentais** e **Espirituais** — e determinam diretamente o poder, valor e capacidades do personagem em diversas situações de jogo.

Existem três tipos de atributo:

- **Primários** — a base da ficha. Recebem os pontos que o jogador distribui.
- **Médios** — a média de dois primários. Não recebem pontos: crescem junto com os primários que os formam.
- **Espirituais** — o poder Nen. Não recebem pontos.

## Atributos Físicos (Velocidade de Progressão: 5.0)

Os atributos físicos formam um losango. Os quatro **primários** ficam nos vértices e os quatro **médios** no meio das arestas, cada um entre os dois primários que o formam:

```
            Resistência
   Força                Constituição
Agilidade                     Sentido
   Celeridade             Destreza
           Flexibilidade
```

| Atributo | Tipo | Formado por | O que representa |
|---|---|---|---|
| Resistência | Primário | — | Defesa, vitalidade e energia física |
| Agilidade | Primário | — | Deslocamento: velocidade, aceleração e frenagem |
| Flexibilidade | Primário | — | Amplitude de movimentos: acrobacia, evasão, mover-se sem ser ouvido |
| Sentido | Primário | — | Percepção pelos cinco sentidos |
| Força | Médio | Resistência + Agilidade | Potência física: empurrar, agarrar, carregar |
| Celeridade | Médio | Agilidade + Flexibilidade | Rapidez dos movimentos: ligeireza, repelir, fintar |
| Destreza | Médio | Flexibilidade + Sentido | Precisão e controle fino: reflexo, pontaria, furtividade |
| Constituição | Médio | Sentido + Resistência | Fôlego e recuperação: cura, respiração, tenacidade |

Por isso, um ponto em um primário fortalece também os dois médios vizinhos. Um ponto em Agilidade, por exemplo, sobe Força e Celeridade.

## Atributos Mentais (Velocidade de Progressão: 1.0)

Os quatro atributos mentais são todos **primários**: não há médios entre eles.

| Atributo | Descrição |
|---|---|
| Resiliência | Resistência mental e foco |
| Adaptabilidade | Capacidade de se ajustar a situações |
| Julgamento (ou Ponderação) | Análise e tomada de decisão |
| Criatividade | Inovação e pensamento lateral |

## Atributos Espirituais

Atributos espirituais representam o poder Nen do personagem. Diferem dos demais por **não possuírem pontos distribuíveis** — seu poder vem exclusivamente do nível, bônus de habilidade e buffs.

| Atributo | Velocidade de Progressão | Descrição |
|---|---|---|
| Chama | 1.0 | Intensidade e força da aura |
| Consciência | 1.0 | Controle e percepção da aura |

> **Nota:** Atributos espirituais só existem para personagens que despertaram o Nen.

## Fórmulas

Cada atributo primário possui:
- **Pontos distribuídos** — adicionados pelo jogador
- **Experiência própria** — recebida via progressão em cascata
- **Nível** — determinado pela tabela de progressão
- **Bônus de habilidade** — influência da habilidade governante (Físicos ou Mentais)

> *valor = pontos + nível*
>
> *poder = valor + bônus de habilidade (arredondado) + buff*

O **poder** é o número usado nos testes.

### Atributos médios

> *pontos = média arredondada dos pontos dos dois primários que o formam*
>
> *valor = pontos + nível*
>
> *poder = valor + bônus de habilidade (arredondado) + buff*

O arredondamento utilizado é o padrão: metade arredonda para longe do zero. Exemplo: a média de 3 e 4 é 3,5, que arredonda para 4.

O bônus de habilidade de um atributo médio é a média dos bônus dos seus atributos primários componentes.

### Atributos espirituais

> *poder = nível + bônus de habilidade (arredondado) + buff*

## Distribuição de Pontos

Na criação do personagem, o jogador distribui pontos nos **atributos primários físicos e mentais**:

- Os pontos físicos são iguais ao **nível da habilidade Físicos**, e os mentais ao **nível da habilidade Mentais**. Esses níveis vêm da experiência inicial que a classe dá.
- Cada grupo é distribuído entre os seus próprios primários: pontos físicos só vão para Resistência, Agilidade, Flexibilidade e Sentido; pontos mentais, para Resiliência, Adaptabilidade, Julgamento e Criatividade.
- Todos os pontos de cada grupo precisam ser usados — nem mais, nem menos.
- Atributos médios e espirituais não recebem pontos. Os médios acompanham os primários automaticamente; os espirituais dependem só de experiência e buffs.

Depois da criação, o progresso dos atributos vem da experiência obtida por treinamento (progressão em cascata) e de buffs temporários.

## Progressão em Cascata

Quando experiência é inserida em uma perícia, ela se propaga pela cadeia:

1. A **perícia** recebe o XP
2. O **atributo** recebe o XP (para atributos médios, o XP é dividido entre os dois primários)
3. A **habilidade** recebe o XP
4. A **experiência do personagem** recebe o XP

Para atributos médios, a divisão de XP entre os primários é feita de forma precisa, levando em conta o resto de divisões anteriores. Isso garante que nenhum ponto de experiência é perdido ao longo do tempo.

## Buffs Temporários

Buffs são bônus (ou penalidades) temporários que alteram o poder de um atributo. Eles representam efeitos passageiros no jogo, como:
- **Itens equipados** — uma armadura pode aumentar a Resistência
- **Habilidades Nen** — técnicas que ampliam temporariamente um atributo
- **Condições de combate** — efeitos de terreno, poções, estados alterados

Os buffs são aplicados e removidos automaticamente pelo sistema conforme as circunstâncias do jogo. Quando um buff está ativo, seu valor é somado diretamente ao poder do atributo afetado, refletindo imediatamente em todas as mecânicas que dependem daquele atributo.

---

> **🔧 Para Desenvolvedores**
>
> Implementação técnica: [`docs/dev/character-sheet/abilities-attributes.md`](../../dev/character-sheet/abilities-attributes.md)
> Código-fonte: `internal/domain/entity/character_sheet/attribute/`
