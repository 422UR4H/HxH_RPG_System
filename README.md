Front-end:
https://github.com/422UR4H/HxH_RPG_Environment_React


# Hunter X Hunter RPG System

A plataforma do sistema canônico de RPG de HxH, ainda nomeado de forma literal e sem as regras compiladas em um livro, é a mais nova versão de [GxG Batalhas: Integrado](https://github.com/422UR4H/GxG-Batalhas-Integrado)!


## O que há de novo?

Primeiramente, essa nova versão do projeto foi escrita em Go! Ela se propõe a ser mais canônica, dinâmica e imersiva que a última, levando em consideração o sistema de atributos físicos e mentais propostos por Netero, cada princípio Nen, básico ou avançado e categorias Nen exatamente como é na obra, além de outras coisas, como o treino, evolução granular e dinâmica de personagens, stamina, criação de habilidades Nen e muito mais!


### O Novo Sistema de Batalhas

Renomeado como Sistema de Combate, é a "calculadora" que emulador batalhas Nen de alto nível com, por exemplo, a utilização de Ryu exatamente como proposto no anime!
A mecânica de batalha é dinâmica. Além dos turnos serem rápidos, evitando o tédio que leva à distração, a plataforma garante uma interface amigável e simples pros jogadores declararem sua própria ação de forma mecânicamente amarrada com o sistema, sem perder qualquer liberdade garantida em uma mesa de RPG! Isso possibilita automação de cálculos e mais imersão, com foco na narração do combate em si.

Essa dinamicidade também se dá, por meio da "quebra" da estrutura robusta e inflexível do turno, tornando-os fluidos. Dependendo da ação realizada pelo personagem, os parâmetros para a velocidade de execução da mesma, varia. É possível um personagem realizar 2 ou mais ações (ataques, movimentos, etc.), enquanto outro personagem realiza apenas 1 e o gerenciamento realizado é explícito e automático, liberando carga cognitiva para o que é mais importante no RPG.


## Mais sobre o projeto

Os Sistemas (tanto as regras, quanto este aqui hospedado) possui versões antigas, onde somente quem tinha contato com o software era o mestre. Estes sistemas apenas emulavam Hunter X Hunter da forma mais canônica que foi possível na época. [Vale a pena dar uma olhada!](https://github.com/422UR4H/GxG-Batalhas-Integrado)

Agora os sistemas estão em migração para essa nova proposta, visando melhorar a experiência de mesas de RPG online e presencial! Sim, a plataforma é híbrida e com esta nova versão os jogadores utilização fichas digitais (em celulares, tablets, notebooks e computadores), e ainda haverá versão para impressão.
As fichas digitais já estão prontas aqui e no [Front-end!](https://github.com/422UR4H/HxH_RPG_Environment_React)

A nova etapa do projeto consiste em realizar a definição das entidades de domínio, contratos, modelagem lógica da solução de todos os recursos a serem utilizados pelos usuários durante uma partida. Algumas entidades e contratos já estão prontos, assim como o fluxo de um jogo a nível de regra de jogo. Este é muito importante porque é a solução do problema de dinâmica de partidas de RPG, principalmente com mais de 5 jogadores na mesa. A nova proposta traz essa melhoria, além do contato direto dos jogadores com a plataforma e esses são os 2 problemas que estão sendo resolvidos técnicamente nesse momento!

Decidi logar as atualizações no board do Excalidraw aqui abaixo por enquanto, mas também deixar alguns prints (que estarão rapidamente desatualizados), apenas para registrar o trabalho que pode parecer parado, mas que está em constante desenvolvimento!!


**Links do desenvolvimento lógico no Excalidraw:**
* https://excalidraw.com/#json=o-b6B0_9vGhSqAtTbc_hO,ZOPhGHyULdy_hpXW3AVFGg


Diagrama atual de Atributos Físicos:
![alt text](image.png)

Diagrama de Actions:
![alt text](image-2.png)

Contrato de Actions:
![alt text](image-3.png)

Entidade de Actions:
![alt text](image-4.png)

Protótipo do Front-end do Mestre:
![alt text](image-7.png)

Fluxo de Alto Nível do Combate (regra de jogo):
![alt text](image-5.png)

Fluxo de Baixo Nível do Combate (nível técnico) - em desenvolvimento:
![alt text](image-6.png)