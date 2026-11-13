# Classes de personagem

Quando uma classe entra ou sai de `Build()` em `character_class_factory.go`, atualize também:

- a contagem e a lista em `docs/game/classes.md`;
- as marcas "(em breve)" em `docs/game/glossario.md`.

Por quê: esses docs são mostrados aos jogadores no painel de regras da ficha. Se a lista ficar
diferente do que `GET /classes` devolve, o jogador lê uma regra que não bate com o que ele escolhe.
