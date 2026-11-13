# Atributos

## Visão Geral

Atributos são as características fundamentais do personagem. Eles se dividem em **Físicos**, **Mentais** e **Espirituais**, e o poder de cada um é o número que você usa nos testes.

Existem três tipos de atributo:

- **Primários** — os canônicos. Foram ditos pelo próprio Netero, assim como o [Talento](./aptidoes.md#talento). São a base da ficha e recebem os pontos que você distribui.
- **Secundários** — os mecânicos. Foram criados para o jogo, porque várias mecânicas de RPG dependiam deles, como Força e Destreza. Cada um é a média de dois primários e não recebe pontos.
- **Espirituais** — o poder Nen. Não recebem pontos.

Todos os atributos evoluem com treino, como no mangá e no anime: os físicos treinam pelas suas perícias, os mentais treinam diretamente e os espirituais também podem ser treinados. Só os primários físicos e mentais recebem pontos para distribuir.

## Atributos Físicos

Os atributos físicos formam um losango. Os quatro **primários** ficam nos vértices e os quatro **secundários** no meio das arestas, cada um entre os dois primários que o formam:

```diagram physical paint=all
            Resistência
   Força                Constituição
Agilidade                     Sentido
   Celeridade             Destreza
           Flexibilidade
```

| Atributo | Na ficha | Tipo | O que representa |
|---|---|---|---|
| Resistência | Resistance (RES) | Primário | Defesa, vitalidade e energia física |
| Agilidade | Agility (AGI) | Primário | Deslocamento: rapidez, aceleração e frenagem |
| Flexibilidade | Flexibility (FLX) | Primário | Amplitude de movimentos: acrobacia, evasão, deslizar |
| Sentido | Sense (SEN) | Primário | Percepção pelos cinco sentidos |
| Força | Strength (Str) | Secundário | Média de Resistência e Agilidade. Potência física: empurrar, agarrar, carregar |
| Celeridade | Celerity (Cel) | Secundário | Média de Agilidade e Flexibilidade. Golpes e movimentos rápidos: ligeireza, repelir, finta |
| Destreza | Dexterity (Dex) | Secundário | Média de Flexibilidade e Sentido. Precisão e controle fino: reflexo, pontaria, furtividade |
| Constituição | Constitution (Con) | Secundário | Média de Sentido e Resistência. Fôlego e recuperação: cura, respiração, tenacidade |

Por isso, um ponto em um primário fortalece também os dois secundários vizinhos. Um ponto em Agilidade, por exemplo, sobe Força e Celeridade.

```diagram physical paint=AGI,Str,Cel,AGI-Str,Cel-AGI
            Resistência
   Força                Constituição
Agilidade                     Sentido
   Celeridade             Destreza
           Flexibilidade
```

## Atributos Mentais

Os quatro atributos mentais são primários. Ainda não existem secundários entre eles.

São atributos canônicos, citados pelo Netero, e usados para avaliar os candidatos no Exame Hunter.

```diagram mental paint=all
          Resiliência
Criatividade     Adaptabilidade
         Julgamento
```

| Atributo | Na ficha | Descrição |
|---|---|---|
| Resiliência | Resilience (RSL) | Capacidade de aguentar a pressão, se recuperar dos golpes da vida e não se quebrar. Exemplo: no fim da luta contra o Hisoka, Kastro falhou num teste de Resiliência, o que o impediu de materializar seu clone e o levou à morte |
| Adaptabilidade | Adaptability (ADP) | Capacidade de se ajustar a situações |
| Julgamento (ou Ponderação) | Judgment (JUD) | Análise e tomada de decisão. É o atributo usado, por exemplo, ao ponderar sobre o nível e as capacidades de um oponente |
| Criatividade | Creativity (CRE) | Inovação e pensamento lateral. Muito importante para materializadores: impacta diretamente as habilidades de materialização |

## Atributos Espirituais

Os atributos espirituais representam o poder Nen do personagem e só existem para quem despertou o Nen. Eles não recebem pontos para distribuir: o poder deles vem do nível e do bônus de aptidão.

| Atributo | Na ficha | Descrição |
|---|---|---|
| Nen da Chama | Flame | A chama que se acende no espírito: a força de vontade, a decisão. Determina a maturidade do espírito |
| Nen da Consciência | Conscience | O verdadeiro Nen: o poder de controlar livremente a energia vital (aura). Desperta naturalmente quando o espírito atinge certa maturidade (Nen da Chama) |

> **Nota:** As mecânicas canônicas de Nen estão sendo migradas do sistema antigo para este novo (System X System).

## Como ler um atributo

Na ficha, cada atributo mostra três números sob a sigla: o **nível**, os **pontos** e o **poder**.

```diagram attribute-node
RES  nível · pontos · poder
```

## Fórmulas

Cada atributo primário possui:
- **Pontos distribuídos** — os que você coloca nele
- **Experiência própria** — recebida em cascata pelo treino e pela progressão em combate, a partir da evolução das perícias, ou direto nos atributos mentais (veja [Progressão em Cascata](./atributos.md#progressao-em-cascata))
- **Nível** — vem da experiência acumulada
- **Bônus de aptidão** — o da aptidão a que ele pertence: Física ou Mental (veja [Aptidões](./aptidoes.md#bonus-de-aptidao))

> *valor = pontos + nível*
>
> *poder = valor + bônus de aptidão (arredondado para baixo)*

O **poder** é o número usado nos testes.

### Atributos secundários

> *pontos = média dos pontos dos dois primários que o formam (arredondada para baixo)*
>
> *valor = pontos + nível*
>
> *poder = valor + bônus de aptidão (arredondado para baixo)*

```diagram physical paint=RES,AGI,Str,AGI-Str,Str-RES
            Resistência
   Força                Constituição
Agilidade                     Sentido
   Celeridade             Destreza
           Flexibilidade
```

Exemplo: com 3 pontos em Resistência e 4 em Agilidade, a média é 3,5, e a Força fica com **3** pontos.

O bônus de aptidão de um atributo secundário é a média dos bônus dos seus dois primários.

### Atributos espirituais

> *poder = nível + bônus de aptidão (arredondado para baixo)*

## Distribuição de Pontos

Na criação do personagem, você distribui pontos nos **atributos primários físicos e mentais**:

- Você tem tantos pontos físicos quanto o nível da **Aptidão Física**, e tantos pontos mentais quanto o nível da **Aptidão Mental**. Esses níveis vêm da experiência inicial que a classe dá (veja [Aptidões](./aptidoes.md#pontos-para-distribuir)).
- Pontos físicos só vão para Resistência, Agilidade, Flexibilidade e Sentido; pontos mentais, para Resiliência, Adaptabilidade, Julgamento e Criatividade.
- Todos os pontos de cada grupo precisam ser usados — nem mais, nem menos.
- Atributos secundários e espirituais não recebem pontos. Os secundários acompanham os primários sozinhos.

Depois da criação, os atributos evoluem pela experiência que chega em cascata (veja abaixo).

## Progressão em Cascata

A experiência que o personagem ganha treinando sobe em cascata pela ficha:

1. A **perícia** treinada recebe a experiência.
2. O **atributo** da perícia recebe a mesma experiência. Se for um atributo secundário, ele a reparte entre os seus dois primários.
3. A **aptidão** do atributo recebe a experiência, e a Aptidão de Perícias também.
4. Por fim, a experiência chega ao **nível do personagem**.

Os atributos mentais treinam direto: a experiência entra no atributo e segue dele para a Aptidão Mental e para o nível do personagem.

**Exemplo:** imagine que seu personagem treina *Acrobacia*. Esse treino não melhora apenas a Acrobacia: ele também fortalece a *Flexibilidade*, que faz crescer a *Aptidão Física*, e por fim soma ao nível do personagem. É como treinar um movimento específico que acaba fortalecendo o corpo inteiro.

Hoje, na ficha, você não precisa fazer nada com isso. A cascata vai fazer diferença de verdade quando o treino chegar ao SxS, e aí ela ganha mais explicações.

## Buffs Temporários

Buffs são bônus (ou penalidades) temporários que alteram o poder de um atributo. Eles representam efeitos passageiros no jogo, como:
- **Itens equipados** — uma armadura pode aumentar a Resistência
- **Habilidades Nen** — técnicas que ampliam temporariamente um atributo
- **Condições de combate** — efeitos de terreno, poções, estados alterados

O buff soma direto no poder do atributo, e tudo o que depende daquele atributo sente a diferença na hora:

> *poder = valor + bônus de aptidão + buff*

## Velocidade de Progressão

Cada parte da ficha tem uma **velocidade de progressão**: um número que diz quanta experiência ela pede para subir de nível. Quanto maior o número, mais XP cada nível pede. Os atributos físicos têm velocidade 5; os mentais e os espirituais, 1. Veja todas na [Tabela de Progressão](./experiencia.md#tabela-de-progressao).

---

> **🔧 Para Desenvolvedores**
>
> Implementação técnica: [`docs/dev/character-sheet/abilities-attributes.md`](../../dev/character-sheet/abilities-attributes.md)
> Código-fonte: `internal/domain/entity/character_sheet/attribute/`
>
> - Arredondamento: `int()` sobre o bônus (`GetPower`) e divisão inteira nos pontos do
>   secundário (`MiddleAttribute.GetPoints`); o front espelha com `Math.trunc`/`Math.floor`.
> - Repartição do secundário: `MiddleAttribute.CascadeUpgrade` soma o resto da divisão anterior
>   (`exp % len(primaryAttrs)`) antes de dividir, para nenhum ponto de XP se perder com o tempo.
> - Buffs: mapa por atributo, somado em `GetPower` de primários, secundários e espirituais.
