package match_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	appmatch "github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/google/uuid"
)

func TestEndMatchUCRejectsNonMaster(t *testing.T) {
	s := matchsession.NewMatchSession(uuid.New(), nil, nil)
	master := uuid.New()
	uc := appmatch.NewEndMatchUC()
	_, err := uc.Execute(context.Background(), s, master, uuid.New(), time.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC), nil, false)
	if !errors.Is(err, appmatch.ErrNotMatchMaster) {
		t.Fatalf("err = %v, want ErrNotMatchMaster", err)
	}
}

func TestEndMatchUCRejectsLongBrief(t *testing.T) {
	s := matchsession.NewMatchSession(uuid.New(), nil, nil)
	master := uuid.New()
	long := strings.Repeat("a", 256)
	_, err := appmatch.NewEndMatchUC().Execute(context.Background(), s, master, master, time.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC), &long, false)
	if !errors.Is(err, matchsession.ErrBriefFinalTooLong) {
		t.Fatalf("err = %v, want ErrBriefFinalTooLong", err)
	}
}

func TestEndMatchUCEnds(t *testing.T) {
	s := matchsession.NewMatchSession(uuid.New(), nil, nil)
	master := uuid.New()
	at := time.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC)
	end, err := appmatch.NewEndMatchUC().Execute(context.Background(), s, master, master, at, nil, false)
	if err != nil || !end.StoryAt.Equal(at) {
		t.Fatalf("end=%+v err=%v", end, err)
	}
}

func TestEndMatchUCCarriesBriefFinal(t *testing.T) {
	master := uuid.New()
	at := time.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC)
	uc := appmatch.NewEndMatchUC()
	run := func(in *string) (*matchsession.MatchEnd, error, *matchsession.MatchSession) {
		s := matchsession.NewMatchSession(uuid.New(), nil, nil)
		e, err := uc.Execute(context.Background(), s, master, master, at, in, false)
		return e, err, s
	}
	padded := " Fim da saga "
	if e, err, s := run(&padded); err != nil || e.BriefFinalDescription == nil || *e.BriefFinalDescription != "Fim da saga" {
		t.Fatalf("trim: %+v %v", e, err)
	} else if s.GetActiveScene().BriefFinalDescription != nil {
		t.Error("must not be set on the scene")
	}
	blank := "   "
	if e, err, _ := run(&blank); err != nil || e.BriefFinalDescription != nil {
		t.Fatalf("blank: %+v %v", e, err)
	}
	long := strings.Repeat("é", 256)
	if _, err, s := run(&long); !errors.Is(err, matchsession.ErrBriefFinalTooLong) || s.IsEnded() {
		t.Fatalf("long: err=%v", err)
	}
}
