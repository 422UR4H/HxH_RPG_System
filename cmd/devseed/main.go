// Command devseed builds, on a LOCAL dev stack, a match ready to play with known logins: a
// master, two players, their sheets accepted and enrolled, two NPCs, a map with every wall type
// and the match already started. It exists so a session (human or agent) can test the table as
// the master AND as a player without anyone's real password — see docs/dev/seed.md.
//
// It drives the public REST API and the game WebSocket only, exactly as the front does: no SQL,
// no repository wiring. A flow the API refuses fails the seed the same way it would fail the
// app, so the seed doubles as a smoke test of the onboarding path.
//
// Idempotent by the campaign name: if the master already has it with the match started, the seed
// prints what exists and stops — a campaign with a started match cannot be deleted, so a fresh
// table is -suffix. A campaign left half-built (no started match) is deleted and rebuilt, and a
// run that fails undoes what it created.
//
//	go run ./cmd/devseed            # API http://localhost:5000, WS ws://localhost:8081
//	go run ./cmd/devseed -suffix 2  # a second, independent table (other users and campaign)
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Password is shared by every seeded account. Dev only: these users exist on a local database.
const Password = "seed-dev-123"

type account struct {
	Nick, Email string
	token       string
	uuid        string
}

type seed struct {
	api, ws string
	client  *http.Client
	// undo runs in reverse when the seed fails, so a broken run leaves nothing half-built.
	undo []func()
}

func main() {
	apiURL := flag.String("api", envOr("DEVSEED_API", "http://localhost:5000"), "REST base URL")
	wsURL := flag.String("ws", envOr("DEVSEED_WS", "ws://localhost:8081"), "game WebSocket base URL")
	suffix := flag.String("suffix", "", "appended to every nick/e-mail/name — a second independent table")
	front := flag.String("front", envOr("DEVSEED_FRONT", "http://localhost:5173"), "front base URL, for the printed links")
	flag.Parse()

	s := &seed{api: strings.TrimRight(*apiURL, "/"), ws: strings.TrimRight(*wsURL, "/"), client: &http.Client{Timeout: 15 * time.Second}}
	sfx := *suffix

	master := &account{Nick: "SeedMestre" + sfx, Email: "mestre" + sfx + "@seed.dev"}
	players := []*account{
		{Nick: "SeedJog1" + sfx, Email: "jogador1" + sfx + "@seed.dev"},
		{Nick: "SeedJog2" + sfx, Email: "jogador2" + sfx + "@seed.dev"},
	}
	for _, a := range append([]*account{master}, players...) {
		s.must(s.ensureAccount(a), "account "+a.Email)
	}

	campaignName := "Seed mesa de teste" + sfx
	if id, ok := s.findCampaign(master, campaignName); ok {
		if matchID, started := s.startedMatch(master, id); started {
			fmt.Printf("Já existe: campanha %q com a partida iniciada. Nada foi criado.\n", campaignName)
			fmt.Printf("Para outra mesa do zero, rode com -suffix.\n\n")
			fmt.Printf("Mesa:  %s/campaigns/%s/matches/%s/game\n\n", strings.TrimRight(*front, "/"), id, matchID)
			printLogins(master, players)
			return
		}
		// Half-built by an earlier run: nothing started, so it can go.
		s.call(http.MethodDelete, master, "/campaigns/"+id, nil, nil)
	}

	campaignID := s.createCampaign(master, campaignName)
	s.undo = append(s.undo, func() { _, _, _ = s.do(http.MethodDelete, "/campaigns/"+campaignID, master.token, nil) })
	matchID := s.createMatch(master, campaignID)

	// Players: a free sheet each, submitted to the campaign, accepted, enrolled, accepted.
	sheetIDs := make([]string, len(players))
	for i, p := range players {
		sheetIDs[i] = s.createSheet(p, "", fmt.Sprintf("Jogador%d%s", i+1, sfx), fmt.Sprintf("Jogador Seed %d", i+1), playerClasses[i])
		id, tok := sheetIDs[i], p.token
		s.undo = append(s.undo, func() { _, _, _ = s.do(http.MethodDelete, "/charactersheets/"+id, tok, nil) })
		s.post(p, "/submissions/charactersheets/submit", map[string]any{"sheetUuid": sheetIDs[i], "campaignUuid": campaignID}, nil)
		s.post(master, "/submissions/"+sheetIDs[i]+"/accept", nil, nil)
		s.post(p, "/enrollments/charactersheets/enroll", map[string]any{"sheetUuid": sheetIDs[i], "matchUuid": matchID}, nil)
	}
	var enrollments struct {
		Enrollments []struct {
			UUID   string `json:"uuid"`
			Status string `json:"status"`
		} `json:"enrollments"`
	}
	s.get(master, "/matches/"+matchID+"/enrollments", &enrollments)
	for _, e := range enrollments.Enrollments {
		if e.Status != "accepted" {
			s.post(master, "/enrollments/"+e.UUID+"/accept", nil, nil)
		}
	}

	// NPCs: the master's sheets inside the campaign, put in the match.
	npcIDs := []string{
		s.createSheet(master, campaignID, "Capanga"+sfx, "Capanga da Seed", "Swordsman"),
		s.createSheet(master, campaignID, "Guarda"+sfx, "Guarda da Seed", "Samurai"),
	}
	for _, id := range npcIDs {
		s.post(master, "/matches/"+matchID+"/npcs", map[string]any{"characterSheetUuid": id}, nil)
	}

	mapID := s.createMap(master, campaignID, append(sheetIDs, npcIDs...))
	s.post(master, "/matches/"+matchID+"/map", map[string]any{"mapUuid": mapID}, nil)
	s.must(s.startMatch(master, matchID), "start_match")

	fmt.Printf("Pronto. Partida iniciada.\n\n")
	fmt.Printf("Mesa:  %s/campaigns/%s/matches/%s/game\n\n", strings.TrimRight(*front, "/"), campaignID, matchID)
	printLogins(master, players)
}

func printLogins(master *account, players []*account) {
	fmt.Printf("Logins (senha de todos: %s)\n", Password)
	fmt.Printf("  mestre:   %s\n", master.Email)
	for _, p := range players {
		fmt.Printf("  jogador:  %s\n", p.Email)
	}
	fmt.Println("\nDois papéis ao mesmo tempo no mesmo navegador: cada origem guarda o próprio login.")
	fmt.Println("O Vite do dev-checkout só escuta em localhost ([::1]); suba um segundo para a outra origem:")
	fmt.Println("  (no System_X_System_React) npx vite --host 127.0.0.1 --port 5173 --strictPort")
	fmt.Println("e abra um papel em http://localhost:5173 e o outro em http://127.0.0.1:5173.")
}

// ensureAccount registers the account, or logs in when the e-mail is already taken.
func (s *seed) ensureAccount(a *account) error {
	body := map[string]any{"nick": a.Nick, "email": a.Email, "password": Password, "confirmPass": Password}
	status, _, err := s.do(http.MethodPost, "/auth/register", "", body)
	if err != nil {
		return err
	}
	if status != http.StatusCreated && status != http.StatusConflict {
		return fmt.Errorf("register %s: status %d", a.Email, status)
	}
	var out struct {
		Token string `json:"token"`
		User  struct {
			UUID string `json:"uuid"`
		} `json:"user"`
	}
	status, raw, err := s.do(http.MethodPost, "/auth/login", "", map[string]any{"email": a.Email, "password": Password})
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("login %s: status %d: %s", a.Email, status, raw)
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return err
	}
	a.token, a.uuid = out.Token, out.User.UUID
	return nil
}

func (s *seed) findCampaign(master *account, name string) (string, bool) {
	var out struct {
		Campaigns []struct {
			UUID string `json:"uuid"`
			Name string `json:"name"`
		} `json:"campaigns"`
	}
	s.get(master, "/campaigns", &out)
	for _, c := range out.Campaigns {
		if c.Name == name {
			return c.UUID, true
		}
	}
	return "", false
}

// startedMatch reports the campaign's match that already started, if any.
func (s *seed) startedMatch(master *account, campaignID string) (string, bool) {
	var out struct {
		Campaign struct {
			Matches []struct {
				UUID        string  `json:"uuid"`
				GameStartAt *string `json:"gameStartAt"`
			} `json:"matches"`
		} `json:"campaign"`
	}
	s.get(master, "/campaigns/"+campaignID, &out)
	for _, m := range out.Campaign.Matches {
		if m.GameStartAt != nil {
			return m.UUID, true
		}
	}
	return "", false
}

func (s *seed) createCampaign(master *account, name string) string {
	var out struct {
		UUID string `json:"uuid"`
	}
	s.post(master, "/campaigns", map[string]any{
		"name": name, "briefInitialDescription": "Criada pelo cmd/devseed", "description": "Mesa de teste local (cmd/devseed).",
		"isPublic": false, "callLink": "",
		"storyStartAt": "2000-01-01", "storyCurrentAt": "2000-01-01T12:00:00Z",
	}, &out)
	return out.UUID
}

func (s *seed) createMatch(master *account, campaignID string) string {
	var out struct {
		Match struct {
			UUID string `json:"uuid"`
		} `json:"match"`
	}
	s.post(master, "/matches", map[string]any{
		"campaignUuid": campaignID, "title": "Seed: combate", "briefInitialDescription": "Mesa de teste",
		"description": "Partida de teste local (cmd/devseed).",
		"isPublic":    false, "gameScheduledAt": time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339),
		"storyStartAt": "2000-01-01",
	}, &out)
	return out.Match.UUID
}

// createSheet: campaignID empty = a player's free sheet; set = an NPC of that campaign's master.
// The classes the seed builds and the proficiency EXP each one must be created with: a class
// with a distribution takes exactly its points among the weapons it allows (Ninja: one of
// Dagger/Katana/Katar at 127); one without takes none (Swordsman and Samurai come armed).
var playerClasses = []string{"Ninja", "Swordsman"}

var classProficiencies = map[string]map[string]int{
	"Ninja": {"Katana": 127},
}

func (s *seed) createSheet(owner *account, campaignID, nick, fullname, class string) string {
	body := map[string]any{
		"profile": map[string]any{
			"nickname": nick, "fullname": fullname, "briefDescription": "Ficha da seed",
			"description": "Ficha criada pelo cmd/devseed.", "alignment": "Neutral-Neutral",
			"birthday": "0000-05-15T00:00:00Z", "age": 20,
		},
		"characterClass":    class,
		"skillsExps":        map[string]int{},
		"proficienciesExps": orEmpty(classProficiencies[class]),
	}
	if campaignID != "" {
		body["campaignUuid"] = campaignID
	}
	var out struct {
		CharacterSheet struct {
			UUID string `json:"uuid"`
		} `json:"characterSheet"`
	}
	s.post(owner, "/charactersheets", body, &out)
	return out.CharacterSheet.UUID
}

// createMap: a 16×12 square map with one of each wall kind the table handles — a plain wall,
// a locked door, an open door, a window and an unrevealed secret door — and a piece for each
// character. Walls are in world coordinates (cellSize 64), the same space the editor saves.
func (s *seed) createMap(master *account, campaignID string, characterIDs []string) string {
	const c = 64.0
	grid := map[string]any{
		"kind": "square", "cols": 16, "rows": 12, "cellSize": 64, "skewRatio": 1.0, "rotation": 0,
		"color": "#ffffff", "opacity": 0.5, "lineStyle": "solid",
	}
	var created struct {
		Map struct {
			ID string `json:"id"`
		} `json:"map"`
	}
	s.post(master, "/campaigns/"+campaignID+"/maps", map[string]any{"name": "Seed: sala", "grid": grid, "pieces": []any{}}, &created)

	// A room on the right half: x from 8 to 13 cells, y from 2 to 9 cells.
	walls := []map[string]any{
		wall("wall", "stone", [2]float64{8 * c, 2 * c}, [2]float64{13 * c, 2 * c}, 100, 5),
		wall("window", "wood", [2]float64{13 * c, 2 * c}, [2]float64{13 * c, 4 * c}, 20, 2),
		wall("wall", "stone", [2]float64{13 * c, 4 * c}, [2]float64{13 * c, 9 * c}, 100, 5),
		wall("wall", "stone", [2]float64{13 * c, 9 * c}, [2]float64{8 * c, 9 * c}, 100, 5),
		wall("wall", "stone", [2]float64{8 * c, 9 * c}, [2]float64{8 * c, 6 * c}, 100, 5),
		wall("door", "wood", [2]float64{8 * c, 6 * c}, [2]float64{8 * c, 5 * c}, 40, 3),
		wall("wall", "stone", [2]float64{8 * c, 5 * c}, [2]float64{8 * c, 3 * c}, 100, 5),
		wall("secret_door", "stone", [2]float64{8 * c, 3 * c}, [2]float64{8 * c, 2 * c}, 60, 5),
		// A locked door splitting the room in two.
		wall("door", "iron", [2]float64{10 * c, 9 * c}, [2]float64{10 * c, 8 * c}, 80, 8),
		wall("wall", "stone", [2]float64{10 * c, 8 * c}, [2]float64{10 * c, 5 * c}, 100, 5),
	}
	walls[8]["locked"] = true

	slots := [][2]int{{3, 4}, {3, 7}, {11, 4}, {11, 7}}
	pieces := make([]map[string]any, 0, len(characterIDs))
	for i, id := range characterIDs {
		pieces = append(pieces, map[string]any{
			"id": uuid.NewString(), "characterId": id, "visible": true,
			"coord": map[string]any{"slot": map[string]any{"kind": "square", "col": slots[i][0], "row": slots[i][1]}, "z": 0},
		})
	}
	s.put(master, "/maps/"+created.Map.ID, map[string]any{"name": "Seed: sala", "grid": grid, "pieces": pieces, "walls": walls})
	return created.Map.ID
}

func wall(kind, material string, p1, p2 [2]float64, hp, resistance int) map[string]any {
	sense := "full"
	if kind == "window" {
		sense = "none"
	}
	return map[string]any{
		"id": uuid.NewString(), "p1": p1, "p2": p2, "wallType": kind, "material": material,
		"move": true, "sense": sense, "direction": "both", "open": false, "locked": false,
		"hp": hp, "maxHp": hp, "resistance": resistance, "destroyed": false, "revealed": false,
	}
}

// startMatch opens the master's socket and sends start_match, then waits for match_started.
func (s *seed) startMatch(master *account, matchID string) error {
	q := url.Values{"match_uuid": {matchID}, "token": {master.token}}
	conn, _, err := websocket.DefaultDialer.Dial(s.ws+"/ws?"+q.Encode(), nil)
	if err != nil {
		return err
	}
	defer conn.Close() //nolint:errcheck
	if err := conn.WriteJSON(map[string]any{"type": "start_match", "payload": map[string]any{}}); err != nil {
		return err
	}
	deadline := time.Now().Add(10 * time.Second)
	_ = conn.SetReadDeadline(deadline)
	for time.Now().Before(deadline) {
		var msg struct {
			Type    string          `json:"type"`
			Payload json.RawMessage `json:"payload"`
		}
		if err := conn.ReadJSON(&msg); err != nil {
			return err
		}
		switch msg.Type {
		case "match_started":
			return nil
		case "error":
			return fmt.Errorf("start_match refused: %s", msg.Payload)
		}
	}
	return errors.New("no match_started within 10s")
}

func (s *seed) get(a *account, path string, out any) { s.call(http.MethodGet, a, path, nil, out) }
func (s *seed) post(a *account, path string, body, out any) {
	s.call(http.MethodPost, a, path, body, out)
}
func (s *seed) put(a *account, path string, body any) { s.call(http.MethodPut, a, path, body, nil) }

func (s *seed) call(method string, a *account, path string, body, out any) {
	status, raw, err := s.do(method, path, a.token, body)
	s.must(err, method+" "+path)
	if status >= 300 {
		s.fail(fmt.Errorf("%s %s: status %d: %s", method, path, status, raw))
	}
	if out != nil {
		s.must(json.Unmarshal(raw, out), "decode "+path)
	}
}

func (s *seed) do(method, path, token string, body any) (int, []byte, error) {
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return 0, nil, err
		}
		r = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, s.api+path, r)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := s.client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer res.Body.Close() //nolint:errcheck
	raw, err := io.ReadAll(res.Body)
	return res.StatusCode, raw, err
}

func (s *seed) must(err error, what string) {
	if err != nil {
		s.fail(fmt.Errorf("%s: %w", what, err))
	}
}

// fail undoes what this run created (campaign, then sheets — newest first) and exits.
func (s *seed) fail(err error) {
	undo := s.undo
	s.undo = nil
	for i := len(undo) - 1; i >= 0; i-- {
		undo[i]()
	}
	if len(undo) > 0 {
		log.Printf("desfeito o que esta rodada criou")
	}
	log.Fatal(err)
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func orEmpty(m map[string]int) map[string]int {
	if m == nil {
		return map[string]int{}
	}
	return m
}
