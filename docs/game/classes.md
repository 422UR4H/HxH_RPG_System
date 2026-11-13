# Classes de Personagem

> Classes do System X System.

## Visão Geral

A classe é aquilo a que o personagem dedicou anos da própria vida para se tornar, até o momento em que a ficha é criada e a aventura começa. Ela define a **especialização inicial** e determina:
- Experiência (XP) inicial em perícias específicas
- Experiência inicial em proficiências de armas (em algumas classes, você escolhe as armas)
- Experiência inicial em atributos
- [Proficiências Conjuntas](./ficha-de-personagem/proficiencias.md#proficiencia-conjunta) exclusivas
- Categorias Nen indicadas (em breve)

Hoje há 12 classes, e novas estão a caminho.

## Escolha de Armas

Em algumas classes, parte da experiência inicial é de armas. A classe traz uma lista de armas e um ou mais lotes de experiência, e você escolhe qual arma da lista fica com cada lote.

**Exemplo:** uma classe com dois lotes, de 210 e 127 XP, pede que você escolha duas armas da lista dela. Uma fica com 210 XP e a outra, com 127.

Essa experiência já entra na ficha assim que você escolhe a classe. Por isso, a arma escolhida não muda os seus pontos para distribuir nem o resto da ficha: ela só decide em qual arma a experiência aparece. Só não deixe de escolher, porque a ficha só é criada quando cada lote tem a sua arma.

## Como a Classe Aplica Experiência Inicial

Você não precisa fazer conta nenhuma. Assim que você escolhe a classe, a ficha já mostra toda a experiência inicial que ela dá: nas perícias, nos atributos e nas armas, contando os lotes de armas.

Escolher as armas não soma nada de novo, só diz para onde vai a experiência de armas que a classe já deu. Trocar de classe, por outro lado, troca toda a experiência inicial, e por isso pode mudar os pontos para distribuir.

---

> **🔧 Para Desenvolvedores**
>
> Implementação técnica: [`docs/dev/character-sheet/factory.md`](../dev/character-sheet/factory.md)
> Código-fonte: `internal/domain/entity/character_class/`
>
> Lotes de armas: o preview de criação aplica a cascata de todos os lotes assim que a classe é
> escolhida, porque o caminho (`Proficiency.CascadeUpgradeTrigger` → `Physicals`) não depende da
> arma. A ficha criada recebe as armas escolhidas e fica igual ao preview.
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
