# Campanhas

Uma **Campanha** é uma série de partidas conectadas por uma narrativa contínua, conduzida por um mestre. Representa a "mesa de RPG" no sentido mais amplo.

## Estrutura

| Campo | Descrição |
|-------|-----------|
| Nome | Nome da campanha (5–32 caracteres) |
| Descrição Breve Inicial | Resumo para listagem (máx. 255 caracteres) |
| Descrição | Texto completo da proposta da campanha |
| É Pública | Se outros jogadores podem visualizar a campanha |
| Link de Chamada | Link para a sala de voz/vídeo (Discord, Meet, etc.) |
| Início da História | Data ficcional onde a narrativa começa |
| Data Atual da História | Onde a narrativa está atualmente (opcional) |
| Fim da História | Quando a narrativa encerra (opcional, preenchido ao finalizar) |

## Regras

- O nome deve ter entre 5 e 32 caracteres
- A descrição breve não pode exceder 255 caracteres
- A data de início da história não pode ser vazia
- Um mestre pode ter no máximo **10 campanhas** ativas
- Se vinculada a um cenário, o cenário deve existir

## Visibilidade

- **Pública**: qualquer jogador logado pode visualizar
- **Privada**: apenas o mestre pode visualizar

## Submissão de Fichas

Jogadores submetem fichas de personagem para participar de uma campanha. O mestre pode aceitar ou rejeitar cada submissão:

1. Jogador cria ficha de personagem
2. Jogador submete a ficha para a campanha desejada
3. Mestre aceita ou rejeita
4. Se aceita, a ficha fica vinculada à campanha

### Restrições de Submissão
- O jogador deve ser dono da ficha
- A ficha não pode estar já submetida em outra campanha
- O mestre não pode submeter sua própria ficha à sua campanha

## Hierarquia

```
Campanha
├── Fichas de Personagem (aceitas)
├── Fichas Pendentes (aguardando aprovação)
└── Partidas
```

---

> **🔧 Para Desenvolvedores**
>
> Implementação técnica: [`docs/dev/campaigns-scenarios.md`](../dev/campaigns-scenarios.md)
> Código-fonte: `internal/domain/entity/campaign/`
