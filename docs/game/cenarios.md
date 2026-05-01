# Cenários

Um **Cenário** é o contexto narrativo e geográfico onde campanhas acontecem. É o "mundo" ou "universo" que o mestre cria para ambientar suas histórias.

## Estrutura

| Campo | Descrição |
|-------|-----------|
| Nome | Nome do cenário (5–32 caracteres) |
| Descrição Breve | Resumo curto para listagem (máx. 64 caracteres) |
| Descrição | Texto completo descrevendo o cenário |
| Campanhas | Lista de campanhas que ocorrem neste cenário |

## Regras

- O nome deve ter entre 5 e 32 caracteres
- A descrição breve não pode exceder 64 caracteres
- Nomes de cenário são únicos globalmente (não pode haver dois cenários com o mesmo nome)
- Cada cenário pertence a um único mestre (o criador)
- Um cenário pode conter múltiplas campanhas

## Relação com Campanhas

Atualmente, campanhas podem existir sem cenário (campo opcional). Futuramente, a vinculação será obrigatória, e cenários servirão como o nível organizacional mais alto da hierarquia:

```
Cenário → Campanhas → Partidas → Cenas → Turnos → Rounds
```

## Permissões

- Apenas o mestre que criou o cenário pode visualizá-lo
- A listagem retorna apenas cenários do próprio mestre

---

> **🔧 Para Desenvolvedores**
>
> Implementação técnica: [`docs/dev/campaigns-scenarios.md`](../dev/campaigns-scenarios.md)
> Código-fonte: `internal/domain/entity/scenario/`
