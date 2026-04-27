
---------- usecases
campaign
character_sheet
enrollment
match
scenario
submission
---------- others
auth
session
error.go


.entity
├── campaign
├── character_class
├── character_sheet <- test
│  ├── ability
│  ├── attribute
│  ├── experience
│  ├── proficiency
│  ├── sheet
│  ├── skill
│  ├── spiritual
│  └── status <- test
├── die   <- test
├── enum
├── item  <- test
├── match
│  ├── action
│  ├── battle
│  └── turn
├── scenario
└── user




testes de skill abstraem match, pois são utilizados em treinos, etc.

a ficha fornece métodos: GetValueForTestOf
basta que algum obj some o valor do dado e o ValueForTest
  - além de outros modificadores, como por exemplo o de uma arma/item, ou o de um status
    * não é de responsabilidade da ficha saber se a arma/item deve ser somada nesse valor
    * mas talvez o status sim
      * esse tipo de coisa pode afetar até mesmo treinos
      * já item/arma pode ter o efeito inverso
        * (arma pesada ajudar no treino e arma com status bônus atrapalhar, por exemplo)
      - definir melhor os status


de fato, é melhor ter uma fachada ou um mediator -> preciso do projeto desacoplado
  -> reaprender sobre esses 2 padrões
    -> entender as diferenças entre eles
  -> entender em qual nível/camada ele precisa ficar
