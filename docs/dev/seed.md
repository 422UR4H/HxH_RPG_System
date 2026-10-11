# Seed de desenvolvimento — mesa de teste com logins conhecidos

`cmd/devseed` monta, num ambiente **local**, uma partida pronta para jogar como **mestre e como
jogador**, sem depender da senha de ninguém. Existe para testar a mesa ponta a ponta (a visão do
jogador, a névoa, reações, o histórico ao vivo) — inclusive numa sessão de agente.

```bash
# com back, game e front de pé (./dev-checkout.sh <branch>)
cd System_X_System
go run ./cmd/devseed
```

Ele imprime o link da mesa e os logins. Todos usam a senha `seed-dev-123`:

| Papel | E-mail | Personagem |
|---|---|---|
| mestre | `mestre@seed.dev` | NPCs Capanga (Swordsman) e Guarda (Samurai) |
| jogador | `jogador1@seed.dev` | Jogador1 (Ninja, Katana) |
| jogador | `jogador2@seed.dev` | Jogador2 (Swordsman) |

## O que ele cria

Tudo pela **API REST e pelo WebSocket**, os mesmos caminhos do app (sem SQL). Uma recusa da API
derruba a seed do mesmo jeito que derrubaria o app, então ela também serve de smoke test do
fluxo de entrada:

1. as três contas (`/auth/register`; se já existem, só `/auth/login`);
2. a campanha "Seed mesa de teste" e a partida "Seed: combate";
3. a ficha de cada jogador → submissão → aceite do mestre → inscrição na partida → aceite;
4. as duas fichas de NPC do mestre, postas na partida (`/matches/{uuid}/npcs`);
5. o mapa "Seed: sala" (16×12, quadrado): uma sala com parede, porta, janela, porta secreta não
   revelada e uma porta de ferro **trancada** no meio, e uma peça para cada personagem;
6. o mapa ligado à partida e a partida **iniciada** (`start_match` pelo socket do mestre).

## Rodar de novo

- **Idempotente pelo nome da campanha.** Se ela já existe com a partida iniciada, a seed só
  imprime o link e os logins. Uma campanha com partida iniciada **não pode ser apagada**
  (`DELETE /campaigns` recusa com 422), então uma mesa do zero é outra mesa:
  `go run ./cmd/devseed -suffix 2` cria contas, campanha e partida novas (`mestre2@seed.dev`…).
- Uma rodada que falha no meio **desfaz** o que criou (campanha e fichas). Uma campanha deixada
  pela metade por uma rodada antiga (sem partida iniciada) é apagada e refeita.
- Flags: `-api` (padrão `http://localhost:5000`), `-ws` (`ws://localhost:8081`), `-front`
  (`http://localhost:5173`, só para o link impresso). Também por env: `DEVSEED_API`,
  `DEVSEED_WS`, `DEVSEED_FRONT`.

## Mestre e jogador ao mesmo tempo, num navegador só

O login vive no `localStorage`, que é **por origem**. `http://localhost:5173` e
`http://127.0.0.1:5173` são origens diferentes, e a API já aceita as duas (`ALLOWED_ORIGINS`).
O Vite do `dev-checkout.sh`, porém, só escuta em `localhost` (`[::1]`); suba um segundo para o
IPv4, na mesma porta:

```bash
cd System_X_System_React
npx vite --host 127.0.0.1 --port 5173 --strictPort
```

Abra o mestre numa origem e o jogador na outra. O `kill $(lsof -ti:5173)` do dev-checkout
derruba os dois.

**Cuidado com o login que já está no navegador:** logar como jogador da seed em `localhost`
substitui o login que estava ali (não há logout por aba). Se a aba é de outra pessoa, use a
origem `127.0.0.1` para a seed.

## Limites

- Só para banco **local**: as senhas são públicas, neste arquivo.
- As fichas saem no mínimo que a criação aceita (sem EXP distribuída). As classes com
  distribuição de proficiência exigem exatamente os pontos dela (`character_class_factory.go`:
  Ninja = 127 numa de Dagger/Katana/Katar); as demais, nenhuma.
- Não há turno, ação nem reação na fila: a partida começa limpa, em "Cena: roleplay".
