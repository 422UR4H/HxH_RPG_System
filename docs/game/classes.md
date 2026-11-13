# Classes de Personagem

> Classes do System X System.

## Visão Geral

A classe é aquilo a que o personagem dedicou anos da própria vida para se tornar, até o momento em que a ficha é criada e a aventura começa. Ela define a **especialização inicial** e determina:
- Experiência (XP) inicial em perícias específicas
- Experiência inicial em proficiências de armas
- Experiência inicial em atributos
- Proficiências Conjuntas exclusivas
- Armas que você pode escolher para receber experiência extra
- Categorias Nen indicadas (em breve)

Hoje há 12 classes, e novas estão a caminho.

## Escolha de Armas

Algumas classes deixam você escolher armas para receber experiência extra na criação. A classe oferece uma lista de armas e um ou mais lotes de experiência; você escolhe qual arma recebe cada lote.

**Exemplo:** uma classe com dois lotes, de 210 e 127 XP, pede que você escolha duas armas da lista dela. Uma recebe 210 XP e a outra, 127.

### O que pode dar errado

- **Faltou escolher arma.** A ficha só é criada quando todos os lotes da classe têm uma arma.
- **Os pontos para distribuir mudaram.** A experiência de uma arma também conta para a Aptidão Física (veja [Aptidões](./ficha-de-personagem/aptidoes.md)). Por isso, trocar a arma ou a classe pode mudar quantos pontos físicos você tem para distribuir. Confira o contador antes de criar a ficha.

## Como a Classe Aplica Experiência Inicial

É automático, e a ficha mostra o resultado em tempo real enquanto você escolhe a classe e as armas. A experiência inicial de cada perícia, proficiência e atributo é a soma de duas partes:

1. **A base da classe**: a experiência que a classe já dá, sem depender das suas escolhas.
2. **As armas que você escolheu**: os lotes de experiência extra, quando a classe oferece.

---

> **🔧 Para Desenvolvedores**
>
> Implementação técnica: [`docs/dev/character-sheet/factory.md`](../dev/character-sheet/factory.md)
> Código-fonte: `internal/domain/entity/character_class/`
>
> TODO(dono): voltar para a parte do jogador quando existir classe que distribui perícias.
> Hoje nenhuma classe define `SkillPoints`; só Ninja, Mercenário e Hunter definem
> `ProficiencyPoints`. A regra (correta, só adiantada em relação ao produto):
>
> **Perícias** — a classe define quantos lotes de XP distribuir e os valores disponíveis; o
> jogador escolhe **quais perícias** recebem esses lotes, só entre as permitidas pela classe.
> Exemplo: uma classe que oferece três lotes — 80, 40 e 40 — permite escolher 3 perícias da lista
> permitida, atribuindo 80 XP a uma, 40 a outra e 40 à terceira. Proficiências seguem a mesma
> mecânica com a lista de armas permitidas.
>
> Validação no back (`CharacterClass.ValidateSkills`/`ValidateProficiencies`): quantidade de
> escolhas = número de lotes; só opções da lista da classe; cada valor corresponde a um lote
> disponível, sem repetição; classe sem distribuição rejeita escolhas.
>
> Perícias Conjuntas de classe: `NewJointSkill` não é chamado fora de testes — nenhuma classe tem.
