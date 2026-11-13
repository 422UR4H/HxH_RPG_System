# Sistema de Experiência

## Visão Geral

A experiência (XP) é a base de toda a progressão no System X System. Cada parte da ficha — perícias, proficiências, atributos, aptidões, princípios Nen — tem a própria experiência e o próprio nível.

## Tabela de Progressão

Cada parte da ficha tem uma **velocidade de progressão** própria. São até **100 níveis**, e a experiência pedida por nível cresce de forma não linear: os primeiros níveis são rápidos, os intermediários exigem mais dedicação e os últimos são os mais desafiadores.

### Por parte da ficha

| Componente | Velocidade de Progressão | Descrição |
|---|---|---|
| Personagem | 10.0 | Nível geral do personagem |
| Talento | 2.0 | Nível do talento |
| Aptidão Física | 20.0 | Aptidão física |
| Aptidão Mental | 20.0 | Aptidão mental |
| Aptidão Espiritual | 5.0 | Aptidão espiritual (Nen) |
| Aptidão de Perícias | 20.0 | Treino de todas as perícias juntas |
| Atributos Físicos | 5.0 | Cada atributo físico |
| Atributos Mentais | 1.0 | Cada atributo mental |
| Atributos Espirituais | 1.0 | Cada atributo espiritual |
| Perícias Físicas | 1.0 | Cada perícia física |
| Perícias Espirituais | 3.0 | Cada perícia espiritual |
| Proficiências | 1.0 | Cada proficiência de arma |
| Princípios Nen | 1.0 | Cada princípio e categoria |

Quanto maior o número, mais XP é preciso para subir de nível.

> **🔍 Curiosidade — A Fórmula por trás da Progressão**
>
> A experiência pedida em cada nível *n* vem de uma soma de três curvas em "S":
>
> ```math
> \begin{aligned} f(n) &= \frac{1700}{1 + e^{0{,}37\,(12 - n)}} \\ &+ \frac{1800}{1 + e^{0{,}37\,(38 - n)}} \\ &+ \frac{2000}{1 + e^{0{,}28\,(74 - n)}} \end{aligned}
> ```
>
> Depois, o resultado é multiplicado pela velocidade de progressão do componente. Na prática, a progressão começa suave, acelera nos níveis intermediários e desacelera nos mais altos. Você não precisa fazer essa conta: a ficha faz por você.

## Experiência do Personagem

A experiência do personagem define o **nível do personagem**. Toda experiência que o personagem ganha — numa perícia, numa arma, num princípio Nen — acaba chegando até ela.

### Pontos de Personagem

Os **pontos de personagem** são a soma dos níveis das quatro aptidões e medem a força geral do personagem. Não são pontos para distribuir. Veja [Aptidões](./aptidoes.md#pontos-de-personagem).

## Progressão em Cascata

A experiência ganha treinando sobe em cascata: da perícia para o atributo, dele para a aptidão e, por fim, para o nível do personagem. A explicação completa, com exemplo, está no fim de [Atributos](./atributos.md#progressao-em-cascata).

---

> **🔧 Para Desenvolvedores**
>
> Implementação técnica: [`docs/dev/character-sheet/experience.md`](../../dev/character-sheet/experience.md)
> Código-fonte: `internal/domain/entity/character_sheet/experience/`
>
> - Fórmula: `expTableFunction` em `exp_table.go` (`A_PARAM/10 = 0,37`, `B_PARAM/10 = 0,28`);
>   `NewExpTable` multiplica pelo coeficiente e trunca para `int`.
> - Coeficientes: constantes `*_COEFF` em `character_sheet_factory.go`. `MENTAL_SKILLS_COEFF`
>   (2.0) existe, mas não há perícias mentais — por isso saiu da tabela do jogador.
