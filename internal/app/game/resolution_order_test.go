package game

import (
	"encoding/json"
	"testing"

	domainservice "github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/google/uuid"
)

// O resolution_updated é o retrato inteiro do turno, não um diff — e ele nasce em leitores
// diferentes: a abertura e as edições do mestre saem do read pump dele, cada reação sai do
// read pump de quem reagiu. Calculado sob r.mu e enviado fora dele, um retrato mais velho podia
// chegar ao mestre DEPOIS de um mais novo, e a tela dele ficava com o velho (o caso real: a
// abertura ainda com a reação padrão sentada, chegando depois da esquiva fechada que a
// substituiu — abrir aquela reação respondia "reaction not found").
//
// Estes testes fixam a promessa: o mestre nunca recebe um retrato de turno aberto mais velho
// que o último que já recebeu; o liquidado (o fechamento) sai sempre, para todos.

// resolutionTypes drains a recipient's buffer and lists what it got, keeping only the
// resolution_updated ones as "open" or "settled".
func resolutionTypes(t *testing.T, c *Client) []string {
	t.Helper()
	var out []string
	for {
		select {
		case data := <-c.send:
			var msg Message
			if err := json.Unmarshal(data, &msg); err != nil {
				t.Fatalf("unmarshal message: %v", err)
			}
			if msg.Type != MsgTypeResolutionUpdate {
				continue
			}
			var p ResolutionUpdatedPayload
			if err := json.Unmarshal(msg.Payload, &p); err != nil {
				t.Fatalf("unmarshal resolution_updated: %v", err)
			}
			if p.IsSettled {
				out = append(out, "settled")
			} else {
				out = append(out, "open")
			}
		default:
			return out
		}
	}
}

func TestPublishResolution_TheMasterNeverGetsAnOlderOpenCalculationAfterANewerOne(t *testing.T) {
	masterID := uuid.New()
	r := &Room{clients: map[uuid.UUID]*Client{}, masterUUID: masterID}
	master := barsRecipient(r, masterID)
	turnID := uuid.New()
	open := &domainservice.TurnResolution{}

	// A reação (calculada depois) sai primeiro; a abertura (calculada antes) chega atrasada.
	r.publishResolution(turnID, open, 2)
	r.publishResolution(turnID, open, 1)
	if got := resolutionTypes(t, master); len(got) != 1 {
		t.Fatalf("the master got %v, want exactly one open calculation — the older one must be dropped", got)
	}

	// Na ordem certa, os dois saem: o mais novo substitui o outro na tela.
	r.publishResolution(turnID, open, 3)
	r.publishResolution(turnID, open, 4)
	if got := resolutionTypes(t, master); len(got) != 2 {
		t.Fatalf("the master got %v, want both calculations sent in order", got)
	}

	// O mesmo comando pode produzir dois cálculos com a mesma idade (o fechamento e a abertura
	// de um open_next_action): o segundo não é mais velho que o primeiro, então sai.
	r.publishResolution(turnID, open, 4)
	if got := resolutionTypes(t, master); len(got) != 1 {
		t.Fatalf("the master got %v, want the same-age calculation sent", got)
	}
}

func TestPublishResolution_TheSettledResolutionAlwaysGoesOutAndRetiresOlderOpenOnes(t *testing.T) {
	masterID, playerID := uuid.New(), uuid.New()
	r := &Room{clients: map[uuid.UUID]*Client{}, masterUUID: masterID}
	master := barsRecipient(r, masterID)
	player := barsRecipient(r, playerID)
	closedTurn, nextTurn := uuid.New(), uuid.New()

	// Uma reação já do turno seguinte (idade 6) chegou ao mestre antes do fechamento (idade 5)
	// do turno anterior: o fechamento é outra notícia, não um retrato do mesmo turno, e sai.
	r.publishResolution(nextTurn, &domainservice.TurnResolution{}, 6)
	r.publishResolution(closedTurn, &domainservice.TurnResolution{IsSettled: true}, 5)
	if got := resolutionTypes(t, master); len(got) != 2 || got[1] != "settled" {
		t.Fatalf("the master got %v, want [open settled]", got)
	}
	if got := resolutionTypes(t, player); len(got) != 1 || got[0] != "settled" {
		t.Fatalf("the player got %v, want [settled]", got)
	}

	// Um cálculo aberto mais velho que o fechamento, chegando depois dele, é descartado.
	r.publishResolution(closedTurn, &domainservice.TurnResolution{IsSettled: true}, 8)
	_ = resolutionTypes(t, master)
	r.publishResolution(closedTurn, &domainservice.TurnResolution{}, 7)
	if got := resolutionTypes(t, master); len(got) != 0 {
		t.Fatalf("the master got %v after the close, want nothing — that calculation is older than the close", got)
	}
}
