# Partidas

Uma **Partida** (Match) é uma sessão de jogo individual dentro de uma campanha. Representa um encontro específico onde os jogadores se reúnem para jogar.

## Estrutura

| Campo | Descrição |
|-------|-----------|
| Título | Nome da partida (5–32 caracteres) |
| Descrição Breve Inicial | Resumo do que acontecerá (máx. 255 caracteres) |
| Descrição | Contexto narrativo completo para os jogadores |
| É Pública | Se a partida aparece nas listagens públicas |
| Início do Jogo (real) | Data e hora real de quando a sessão acontecerá |
| Início da História | Data ficcional onde esta sessão se passa |

## Regras de Criação

- O título deve ter entre 5 e 32 caracteres
- A descrição breve não pode exceder 255 caracteres
- O **início do jogo** (data real) não pode ser no passado
- O **início do jogo** não pode ser mais que 1 ano no futuro
- Apenas o mestre da campanha pode criar partidas nela
- O **início da história** deve ser após o início da história da campanha
- Se a campanha tem data de fim, o início da história deve ser antes dela

## Inscrição de Personagens (Enrollment)

Após a ficha ser aceita na campanha, o jogador pode inscrevê-la em partidas específicas:

1. A ficha deve pertencer ao jogador que solicita
2. A ficha não pode estar já inscrita nesta partida
3. A partida deve pertencer à mesma campanha onde a ficha foi aceita

### Restrições
- A ficha deve estar aceita na campanha da partida
- Não é possível se inscrever duas vezes na mesma partida

## Partidas Públicas Futuras

A plataforma oferece uma listagem de partidas públicas futuras (upcoming). Esta lista mostra partidas:
- Marcadas como públicas
- Com data de início do jogo posterior ao momento atual
- De qualquer campanha (para descoberta por novos jogadores)

## Hierarquia Interna (durante o jogo)

```
Partida
├── Cenas (Scenes) — roleplay ou battle
│   └── Turnos (Turns) — modo free ou race
│       └── Rounds — ação de um personagem
│           └── Ações e Reações
└── Eventos de Jogo (Game Events)
```

A categorização de cenas (roleplay vs battle) serve para classificação e
leitura histórica. O modo do turno (free vs race) é independente da
categoria da cena — embora o esperado seja roleplay/free e battle/race, o
mestre pode configurar diferente.

Ver `docs/game/cenas-e-turnos.md` para detalhes completos.

> **Nota:** A Turn/Round Engine está em refatoração semântica.

## Execução em Tempo Real

Quando uma partida é iniciada pelo mestre, todos os participantes (mestre e
jogadores inscritos) se conectam via WebSocket ao Game Server. O fluxo:

1. Jogadores entram no **lobby** da partida (WebSocket)
2. Mestre clica **"Iniciar Partida"** → todos recebem o evento
3. A partida roda em tempo real com troca de mensagens bidirecionais
4. Cenas, turnos, rounds e ações são transmitidos pelo WebSocket

Ver spec de design do WebSocket Game Server para detalhes técnicos.

## Visibilidade

- **Pública**: qualquer jogador pode visualizar
- **Privada**: apenas o mestre pode visualizar
