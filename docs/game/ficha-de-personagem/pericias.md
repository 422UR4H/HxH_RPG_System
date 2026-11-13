# Perícias

## Visão Geral

Perícias são as capacidades específicas do personagem. Cada perícia pertence a um atributo e mede o quanto o personagem é bom numa ação concreta — de empurrar um oponente a sentir um cheiro ou controlar a própria aura.

## Fórmula: Valor de Teste

Quando o personagem faz um teste de perícia, o **valor de teste** é:

> **Valor de Teste** = nível da perícia + poder do atributo

Onde:
- **nível da perícia** — vem da experiência acumulada na perícia
- **poder do atributo** — o poder do atributo a que a perícia pertence (veja [Atributos](./atributos.md#formulas))

Os [buffs temporários](./atributos.md#buffs-temporarios) também somam no valor de teste enquanto duram.

## Lista Completa de Perícias (32)

Cada perícia física pertence a um atributo físico, primário ou secundário. As perícias espirituais pertencem aos atributos espirituais e só existem para quem despertou o Nen. Não há perícias mentais.

### Atributos Físicos

#### Resistência (primário)

| Perícia | Na ficha | Descrição |
|---|---|---|
| Vitalidade | Vitality | Capacidade de resistir a dano e manter-se consciente |
| Energia | Energy | Reserva energética para ações prolongadas |
| Defesa | Defense | Capacidade de bloquear ou absorver impactos |

#### Força (secundário)

| Perícia | Na ficha | Descrição |
|---|---|---|
| Empurrão | Push | Força para empurrar objetos ou oponentes |
| Agarrar | Grab | Capacidade de segurar e imobilizar |
| Carregar | Carry | Peso máximo que pode ser transportado |

#### Agilidade (primário)

| Perícia | Na ficha | Descrição |
|---|---|---|
| Rapidez | Quickness | Velocidade corporal, usada em ajustes de postura, por exemplo |
| Aceleração | Accelerate | Rapidez para atingir a velocidade máxima |
| Freio | Brake | Capacidade de parar ou reduzir a velocidade rapidamente |

#### Celeridade (secundário)

| Perícia | Na ficha | Descrição |
|---|---|---|
| Ligeireza | Legerity | Agilidade fina e controle de movimentos rápidos. Responde pela velocidade de ataque, por exemplo |
| Repelir | Repel | Repelir ou aparar ataques, armados ou desarmados |
| Finta | Feint | Enganar o oponente com movimentos falsos |

#### Flexibilidade (primário)

| Perícia | Na ficha | Descrição |
|---|---|---|
| Acrobacia | Acrobatics | Movimentos acrobáticos e piruetas em saltos |
| Evasão | Evasion | Esquiva sem grande deslocamento, inclusive a esquiva aérea. Também evita golpes críticos que podem matar, mutilar, cegar etc. |
| Deslizar | Sneak | Passar por frestas e buracos e se mover deslizando rente ao chão |

#### Destreza (secundário)

| Perícia | Na ficha | Descrição |
|---|---|---|
| Reflexo | Reflex | Tempo de reação a estímulos |
| Precisão | Accuracy | Pontaria e acerto em geral, incluindo acertos críticos |
| Furtividade | Stealth | Se esconder, passar despercebido e se mover silenciosamente |

#### Sentido (primário)

| Perícia | Na ficha | Descrição |
|---|---|---|
| Visão | Vision | Acuidade visual e capacidade de notar detalhes |
| Audição | Hearing | Percepção sonora e localização pelo som |
| Olfato | Smell | Detecção e identificação de odores |
| Tato | Tact | Sensibilidade ao toque, equilíbrio e preempção (sentir uma presença física) |
| Paladar | Taste | Identificação de substâncias pelo sabor |

#### Constituição (secundário)

| Perícia | Na ficha | Descrição |
|---|---|---|
| Curar | Heal | Recuperação natural de ferimentos |
| Respiração | Breath | Resistência respiratória e capacidade pulmonar |
| Tenacidade | Tenacity | Resistência à fadiga e perseverança física. Ajuda a resistir a danos elementais, como eletricidade, queimaduras e veneno |

### Atributos Espirituais

#### Nen da Chama

| Perícia | Na ficha | Descrição |
|---|---|---|
| Foco | Focus | Atenção sustentada em qualquer tarefa: numa luta, num estudo ou ao canalizar o Nen |
| Força de Vontade | WillPower | Perseverança e determinação para seguir em frente quando tudo pede para parar |
| Autoconhecimento | SelfKnowledge | Compreensão de si mesmo e do próprio Nen |

#### Nen da Consciência

As siglas AOP e MOP aparecem no anime, na luta de Gon contra Knuckle, no arco das Formigas Quimera.

| Perícia | Na ficha | Descrição |
|---|---|---|
| Controle de Aura (CoA) | Coa | Controle fino da aura, como a Destreza é para os movimentos |
| MOP | Mop | Quantidade máxima de aura armazenada dentro do corpo, como uma bateria |
| AOP | Aop | Volume de aura que o usuário consegue manter fora do corpo com o Ren |

## Progressão em Cascata

Quando uma perícia recebe experiência, ela sobe para o atributo da perícia, para a aptidão e para o nível do personagem. A explicação completa está no fim de [Atributos](./atributos.md#progressao-em-cascata).

---

> **🔧 Para Desenvolvedores**
>
> Implementação técnica: [`docs/dev/character-sheet/skills-proficiencies.md`](../../dev/character-sheet/skills-proficiencies.md)
> Código-fonte: `internal/domain/entity/character_sheet/skill/`
>
> - Cascata da perícia comum (`CommonSkill.CascadeUpgradeTrigger`): perícia → atributo →
>   aptidão do atributo → personagem, e também → Aptidão de Perícias (`Skills`).
> - Buffs por perícia: `skill.Manager` guarda um buff por `SkillName`, somado ao valor de teste
>   na leitura, sem alterar experiência nem nível.
> - Perícias Conjuntas (`JointSkill`) existem no domínio, mas nenhuma classe as concede
>   (`NewJointSkill` só é chamado em testes), então saíram do texto do jogador. Resumo: agrupam
>   duas ou mais perícias comuns; o `Manager` busca a conjunta antes da comum; têm buff próprio
>   somado ao valor de teste (`nível da conjunta + poder do atributo + buff da conjunta`); na
>   cascata, a experiência é multiplicada pelo número de perícias comuns da conjunta antes de
>   seguir para a aptidão.
