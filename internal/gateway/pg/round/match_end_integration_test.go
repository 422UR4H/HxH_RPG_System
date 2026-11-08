//go:build integration

package round_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	appmatch "github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/clock"
	roundentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/round"
	sceneentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/scene"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/matchevent"
	pgclock "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/clock"
	"github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/pgtest"
	roundrepo "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/round"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// matchEndFixture é uma partida com o par ativo já em linha e os dados do fim prontos.
type matchEndFixture struct {
	resolutionFixture
	campaignUUID uuid.UUID
	data         appmatch.MatchEndData
}

func newMatchEndFixture(t *testing.T, pool *pgxpool.Pool, repo *roundrepo.Repository, withEvent bool) matchEndFixture {
	t.Helper()
	ctx := context.Background()
	fx := seedMatchAndSheets(t, pool)
	var campaignUUID uuid.UUID
	if err := pool.QueryRow(ctx, `SELECT campaign_uuid FROM matches WHERE uuid = $1`, fx.matchUUID).Scan(&campaignUUID); err != nil {
		t.Fatalf("campaign of match: %v", err)
	}
	base := time.Date(2026, 11, 8, 10, 0, 0, 0, time.UTC)
	sc := sceneentity.ReconstructScene(uuid.New(), enum.Roleplay, "Taverna", base)
	rd := roundentity.ReconstructRound(uuid.New(), enum.Free, base, false, false)
	if err := repo.EnsureSceneAndRound(ctx, fx.matchUUID, sc, rd); err != nil {
		t.Fatalf("EnsureSceneAndRound: %v", err)
	}
	real := base.Add(2 * time.Hour)
	storyAt := time.Date(2000, 1, 3, 21, 15, 30, 0, time.UTC)
	sc.Close(real)
	sc.SetStoryFinishedAt(storyAt)
	rd.SetDrained(true)
	rd.Close(real)
	final := "A caravana chegou"
	d := appmatch.MatchEndData{
		MatchUUID: fx.matchUUID, GameEndAt: real, StoryAt: storyAt, BriefFinalDescription: &final,
		End:   appmatch.RoundEnd{Scene: sc, Round: rd},
		Clock: clock.Snapshot{StoryAt: &storyAt, LimitAt: &storyAt, LimitReason: clock.LimitScene},
	}
	if withEvent {
		payload, _ := json.Marshal(map[string]any{"from": "2000-01-03T21:00:00", "to": "2000-01-03T21:15:30", "elapsedSeconds": 930, "reason": "end"})
		d.RealTimeEvent = &matchevent.Event{
			UUID: uuid.New(), MatchUUID: fx.matchUUID, SceneUUID: sc.GetID(), RoundUUID: rd.GetID(),
			Kind: matchevent.KindClockRealTime, Payload: payload, CreatedAt: real,
		}
	}
	return matchEndFixture{resolutionFixture: fx, campaignUUID: campaignUUID, data: d}
}

func addParticipant(t *testing.T, pool *pgxpool.Pool, matchUUID, sheetUUID uuid.UUID) {
	t.Helper()
	pgtest.InsertTestMatchParticipant(t, pool, matchUUID.String(), sheetUUID.String(), time.Now())
}

func sheetStoryDate(t *testing.T, pool *pgxpool.Pool, sheetUUID uuid.UUID) *time.Time {
	t.Helper()
	var d *time.Time
	if err := pool.QueryRow(context.Background(), `SELECT story_current_at FROM character_sheets WHERE uuid = $1`, sheetUUID).Scan(&d); err != nil {
		t.Fatalf("sheet story_current_at: %v", err)
	}
	return d
}

func campaignStoryAt(t *testing.T, pool *pgxpool.Pool, campaignUUID uuid.UUID) *time.Time {
	t.Helper()
	var at *time.Time
	if err := pool.QueryRow(context.Background(), `SELECT story_current_at FROM campaigns WHERE uuid = $1`, campaignUUID).Scan(&at); err != nil {
		t.Fatalf("campaign story_current_at: %v", err)
	}
	return at
}

func TestPersistMatchEnd(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := roundrepo.NewRepository(pool)
	ctx := context.Background()

	t.Run("writes the end in one transaction", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		fx := newMatchEndFixture(t, pool, repo, true)
		if _, err := pool.Exec(ctx, `UPDATE campaigns SET story_current_at = NULL WHERE uuid = $1`, fx.campaignUUID); err != nil {
			t.Fatalf("clear campaign story: %v", err)
		}

		if err := repo.PersistMatchEnd(ctx, fx.data); err != nil {
			t.Fatalf("PersistMatchEnd: %v", err)
		}

		var gameEnd, storyEnd time.Time
		var final *string
		if err := pool.QueryRow(ctx, `SELECT game_end_at, story_end_at, brief_final_description FROM matches WHERE uuid = $1`,
			fx.matchUUID).Scan(&gameEnd, &storyEnd, &final); err != nil {
			t.Fatalf("read match: %v", err)
		}
		if !gameEnd.Equal(fx.data.GameEndAt) {
			t.Errorf("game_end_at = %v, want %v", gameEnd, fx.data.GameEndAt)
		}
		if storyEnd.Format("2006-01-02") != "2000-01-03" {
			t.Errorf("story_end_at = %v, want the date of storyAt (2000-01-03)", storyEnd)
		}
		if final == nil || *final != "A caravana chegou" {
			t.Errorf("brief_final_description = %v", final)
		}

		var sceneFinished, roundFinished *time.Time
		var storyFinished *time.Time
		pool.QueryRow(ctx, `SELECT finished_at, story_finished_at FROM scenes WHERE uuid = $1`, fx.data.End.Scene.GetID()).Scan(&sceneFinished, &storyFinished) //nolint:errcheck
		pool.QueryRow(ctx, `SELECT finished_at FROM rounds WHERE uuid = $1`, fx.data.End.Round.GetID()).Scan(&roundFinished)                                    //nolint:errcheck
		if sceneFinished == nil || roundFinished == nil {
			t.Errorf("scene finished %v / round finished %v, want both closed", sceneFinished, roundFinished)
		}
		if storyFinished == nil || !storyFinished.Equal(fx.data.StoryAt) {
			t.Errorf("story_finished_at = %v, want %v", storyFinished, fx.data.StoryAt)
		}

		snap, err := pgclock.NewRepository(pool).Load(ctx, fx.matchUUID)
		if err != nil || snap == nil || snap.StoryAt == nil || !snap.StoryAt.Equal(fx.data.StoryAt) || snap.Running {
			t.Errorf("clock = %+v (%v), want paused at %v", snap, err, fx.data.StoryAt)
		}

		var kind string
		var evScene uuid.UUID
		if err := pool.QueryRow(ctx, `SELECT kind, scene_uuid FROM match_events WHERE uuid = $1`, fx.data.RealTimeEvent.UUID).Scan(&kind, &evScene); err != nil {
			t.Fatalf("clockRealTime event not written: %v", err)
		}
		if kind != string(matchevent.KindClockRealTime) || evScene != fx.data.End.Scene.GetID() {
			t.Errorf("event kind %q scene %s, want clockRealTime in the closed scene", kind, evScene)
		}

		if at := campaignStoryAt(t, pool, fx.campaignUUID); at == nil || !at.Equal(fx.data.StoryAt) {
			t.Errorf("campaign story_current_at = %v, want %v", at, fx.data.StoryAt)
		}
	})

	t.Run("without real time running writes no event", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		fx := newMatchEndFixture(t, pool, repo, false)
		if err := repo.PersistMatchEnd(ctx, fx.data); err != nil {
			t.Fatalf("PersistMatchEnd: %v", err)
		}
		var n int
		pool.QueryRow(ctx, `SELECT count(*) FROM match_events WHERE match_uuid = $1`, fx.matchUUID).Scan(&n) //nolint:errcheck
		if n != 0 {
			t.Errorf("match_events = %d, want 0", n)
		}
	})

	t.Run("an unwritten round end lands closed with it", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		fx := newMatchEndFixture(t, pool, repo, false)
		at := time.Date(2026, 11, 8, 9, 0, 0, 0, time.UTC)
		oldSc := sceneentity.ReconstructScene(uuid.New(), enum.Battle, "", at)
		oldRd := roundentity.ReconstructRound(uuid.New(), enum.Race, at, true, true)
		oldSc.Close(at.Add(time.Minute))
		oldRd.Close(at.Add(time.Minute))
		fx.data.UnwrittenRoundEnds = []appmatch.RoundEnd{{Scene: oldSc, Round: oldRd}}
		if err := repo.PersistMatchEnd(ctx, fx.data); err != nil {
			t.Fatalf("PersistMatchEnd: %v", err)
		}
		if finished, exists, _ := roundRowsOf(t, pool, oldRd.GetID(), oldSc.GetID()); !exists || finished == nil {
			t.Errorf("unwritten round exists=%v finished=%v, want a closed row", exists, finished)
		}
	})

	t.Run("a finished match is refused and nothing moves", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		fx := newMatchEndFixture(t, pool, repo, false)
		if err := repo.PersistMatchEnd(ctx, fx.data); err != nil {
			t.Fatalf("first PersistMatchEnd: %v", err)
		}
		later := fx.data
		later.GameEndAt = fx.data.GameEndAt.Add(time.Hour)
		laterStory := fx.data.StoryAt.Add(48 * time.Hour)
		later.StoryAt = laterStory
		if err := repo.PersistMatchEnd(ctx, later); err == nil {
			t.Fatal("a second PersistMatchEnd was accepted")
		}
		if at := campaignStoryAt(t, pool, fx.campaignUUID); at == nil || !at.Equal(fx.data.StoryAt) {
			t.Errorf("campaign story_current_at = %v after a refused end, want %v", at, fx.data.StoryAt)
		}
	})
}

func TestPersistMatchEndNeverRegressesCampaign(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := roundrepo.NewRepository(pool)
	ctx := context.Background()
	pgtest.TruncateAll(t, pool)
	fx := newMatchEndFixture(t, pool, repo, false)
	ahead := time.Date(2000, 6, 1, 8, 0, 0, 0, time.UTC)
	if _, err := pool.Exec(ctx, `UPDATE campaigns SET story_current_at = $1 WHERE uuid = $2`, ahead, fx.campaignUUID); err != nil {
		t.Fatalf("seed campaign story: %v", err)
	}

	if err := repo.PersistMatchEnd(ctx, fx.data); err != nil {
		t.Fatalf("PersistMatchEnd: %v", err)
	}

	if at := campaignStoryAt(t, pool, fx.campaignUUID); at == nil || !at.Equal(ahead) {
		t.Errorf("campaign story_current_at = %v, want %v unchanged — a match played out of order never pulls it back", at, ahead)
	}
}

func TestPersistMatchEndAdvancesParticipantSheets(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := roundrepo.NewRepository(pool)
	ctx := context.Background()
	pgtest.TruncateAll(t, pool)
	fx := newMatchEndFixture(t, pool, repo, false)
	masterUUID := fx.masterUUID.String()
	campaign := fx.campaignUUID.String()

	// attacker: ficha de jogador (player_uuid); victim: NPC (master_uuid). Os dois numa data anterior.
	player := pgtest.InsertTestUser(t, pool, "player-end", "player-end@test.com", "pass")
	playerSheet := uuid.MustParse(pgtest.InsertTestCharacterSheet(t, pool, &player, nil, &campaign, "jogador"))
	npcSheet := fx.victimSheet
	aheadSheet := uuid.MustParse(pgtest.InsertTestCharacterSheet(t, pool, nil, &masterUUID, &campaign, "adiante"))
	otherSheet := uuid.MustParse(pgtest.InsertTestCharacterSheet(t, pool, nil, &masterUUID, &campaign, "outra"))

	earlier := time.Date(1999, 12, 25, 0, 0, 0, 0, time.UTC)
	ahead := time.Date(2000, 2, 1, 0, 0, 0, 0, time.UTC)
	for _, s := range []uuid.UUID{playerSheet, npcSheet, otherSheet} {
		if _, err := pool.Exec(ctx, `UPDATE character_sheets SET story_current_at = $1 WHERE uuid = $2`, earlier, s); err != nil {
			t.Fatalf("seed sheet date: %v", err)
		}
	}
	if _, err := pool.Exec(ctx, `UPDATE character_sheets SET story_current_at = $1 WHERE uuid = $2`, ahead, aheadSheet); err != nil {
		t.Fatalf("seed sheet date: %v", err)
	}
	addParticipant(t, pool, fx.matchUUID, playerSheet)
	addParticipant(t, pool, fx.matchUUID, npcSheet)
	addParticipant(t, pool, fx.matchUUID, aheadSheet)
	// otherSheet participa de OUTRA partida.
	otherMatch := uuid.MustParse(pgtest.InsertTestMatch(t, pool, masterUUID, campaign, "Outra"))
	addParticipant(t, pool, otherMatch, otherSheet)

	if err := repo.PersistMatchEnd(ctx, fx.data); err != nil {
		t.Fatalf("PersistMatchEnd: %v", err)
	}

	endDate := "2000-01-03"
	for name, s := range map[string]uuid.UUID{"player sheet": playerSheet, "npc sheet": npcSheet} {
		if d := sheetStoryDate(t, pool, s); d == nil || d.Format("2006-01-02") != endDate {
			t.Errorf("%s story_current_at = %v, want %s", name, d, endDate)
		}
	}
	if d := sheetStoryDate(t, pool, aheadSheet); d == nil || !d.Equal(ahead) {
		t.Errorf("sheet already later = %v, want %v unchanged", d, ahead)
	}
	if d := sheetStoryDate(t, pool, otherSheet); d == nil || !d.Equal(earlier) {
		t.Errorf("sheet of another match = %v, want %v unchanged", d, earlier)
	}
}

func TestReadMatchEnd(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := roundrepo.NewRepository(pool)
	ctx := context.Background()
	pgtest.TruncateAll(t, pool)
	fx := newMatchEndFixture(t, pool, repo, false)

	if got, err := repo.ReadMatchEnd(ctx, fx.matchUUID); err != nil || got != nil {
		t.Fatalf("ReadMatchEnd before the end = %+v, %v; want nil, nil", got, err)
	}
	if err := repo.PersistMatchEnd(ctx, fx.data); err != nil {
		t.Fatalf("PersistMatchEnd: %v", err)
	}
	got, err := repo.ReadMatchEnd(ctx, fx.matchUUID)
	if err != nil || got == nil {
		t.Fatalf("ReadMatchEnd after the end = %+v, %v", got, err)
	}
	if !got.GameEndAt.Equal(fx.data.GameEndAt) || got.StoryAt == nil || !got.StoryAt.Equal(fx.data.StoryAt) {
		t.Errorf("ReadMatchEnd = %+v, want gameEndAt %v, storyAt %v", got, fx.data.GameEndAt, fx.data.StoryAt)
	}
}
