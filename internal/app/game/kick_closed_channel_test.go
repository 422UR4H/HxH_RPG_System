package game

import (
	"context"
	"encoding/json"
	"sync"
	"testing"

	"github.com/google/uuid"
)

// noopKickUC aprova toda expulsão: estes testes olham só o que a sala faz com a conexão.
type noopKickUC struct{}

func (noopKickUC) Kick(_ context.Context, _, _, _ uuid.UUID) error { return nil }

// kickTestRoom monta uma sala com o mestre e um jogador já à mesa, sem socket: os testes leem
// direto da fila (send) e do sinal de fechamento (done) de cada cliente.
func kickTestRoom() (room *Room, masterUUID uuid.UUID, master, player *Client) {
	masterUUID = uuid.New()
	playerUUID := uuid.New()
	room = NewRoom(uuid.New(), masterUUID, RoomDeps{KickPlayerUC: noopKickUC{}})
	master = NewClient(masterUUID, nil, "gm")
	player = NewClient(playerUUID, nil, "p1")
	room.clients[masterUUID] = master
	room.clients[playerUUID] = player
	return room, masterUUID, master, player
}

// sendRecovering chama SendMessage e devolve o pânico, se houver, para o teste falhar com uma
// mensagem clara em vez de derrubar o binário inteiro (o servidor de verdade não tem esse luxo).
func sendRecovering(c *Client, msg Message) (panicked any) {
	defer func() { panicked = recover() }()
	c.SendMessage(msg)
	return nil
}

func TestKickPlayer_SendAfterKickDoesNotPanic(t *testing.T) {
	room, masterUUID, _, player := kickTestRoom()

	if err := room.KickPlayer(masterUUID, player.userUUID); err != nil {
		t.Fatalf("KickPlayer: %v", err)
	}

	// O intercalamento que derruba o servidor, fixado: outra goroutine (um broadcast por
	// jogador, a resposta à mensagem que o próprio expulso mandou) pegou o ponteiro do cliente
	// antes da expulsão e só agora envia.
	if p := sendRecovering(player, NewServerMessage(MsgTypeChatMessage, ChatPayload{Message: "late"})); p != nil {
		t.Fatalf("SendMessage to a kicked client panicked: %v", p)
	}
}

func TestKickPlayer_KickedClientGetsPlayerKickedThenClose(t *testing.T) {
	room, masterUUID, master, player := kickTestRoom()

	if err := room.KickPlayer(masterUUID, player.userUUID); err != nil {
		t.Fatalf("KickPlayer: %v", err)
	}

	// O expulso recebe player_kicked antes de a conexão fechar: a mensagem já está na fila
	// quando o fechamento é sinalizado, e a WritePump esvazia a fila antes do quadro de
	// fechamento.
	select {
	case data := <-player.send:
		var m Message
		if err := json.Unmarshal(data, &m); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if m.Type != MsgTypePlayerKicked {
			t.Fatalf("kicked player got %s first, want player_kicked", m.Type)
		}
	default:
		t.Fatal("kicked player's queue must hold player_kicked")
	}
	select {
	case <-player.done:
	default:
		t.Fatal("kicked player's connection must be signalled to close")
	}

	// Quem fica à mesa também é avisado, e continua conectado.
	select {
	case data := <-master.send:
		var m Message
		if err := json.Unmarshal(data, &m); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if m.Type != MsgTypePlayerKicked {
			t.Fatalf("master got %s, want player_kicked", m.Type)
		}
	default:
		t.Fatal("master must receive player_kicked")
	}
	select {
	case <-master.done:
		t.Fatal("master's connection must stay open")
	default:
	}

	room.mu.RLock()
	_, still := room.clients[player.userUUID]
	room.mu.RUnlock()
	if still {
		t.Fatal("kicked player must leave the room")
	}
}

// A mesma corrida sem intercalamento fixado: vários remetentes martelando o cliente enquanto a
// expulsão acontece. Com -race, também prova que o fechamento não disputa memória com o envio.
func TestKickPlayer_ConcurrentSendsDuringKickDoNotPanic(t *testing.T) {
	room, masterUUID, _, player := kickTestRoom()

	const senders = 8
	var (
		wg      sync.WaitGroup
		start   = make(chan struct{})
		mu      sync.Mutex
		panics  []any
		message = NewServerMessage(MsgTypeChatMessage, ChatPayload{Message: "spam"})
	)
	for range senders {
		wg.Go(func() {
			<-start
			for range 50 {
				if p := sendRecovering(player, message); p != nil {
					mu.Lock()
					panics = append(panics, p)
					mu.Unlock()
					return
				}
			}
		})
	}

	close(start)
	if err := room.KickPlayer(masterUUID, player.userUUID); err != nil {
		t.Fatalf("KickPlayer: %v", err)
	}
	wg.Wait()

	if len(panics) > 0 {
		t.Fatalf("%d sender(s) panicked sending to the kicked client; first: %v", len(panics), panics[0])
	}
}
