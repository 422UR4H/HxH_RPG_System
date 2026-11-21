package match_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	"github.com/422UR4H/HxH_RPG_System/internal/app/api/match"
	matchUC "github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/statuseffect"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"
)

type mockRevealStatusEffect struct {
	fn func(ctx context.Context, matchUUID, effectUUID, userUUID uuid.UUID) error
}

func (m *mockRevealStatusEffect) Reveal(ctx context.Context, matchUUID, effectUUID, userUUID uuid.UUID) error {
	return m.fn(ctx, matchUUID, effectUUID, userUUID)
}

func TestRevealStatusEffectHandler(t *testing.T) {
	userUUID, matchUUID, effectUUID := uuid.New(), uuid.New(), uuid.New()

	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantDetail string
	}{
		{"revealed", nil, http.StatusNoContent, ""},
		{"not the master", matchUC.ErrNotMatchMaster, http.StatusForbidden, "Só o mestre da partida pode revelar."},
		{"match not found", matchUC.ErrMatchNotFound, http.StatusNotFound, "Partida não encontrada."},
		{"effect not found", statuseffect.ErrNotFound, http.StatusNotFound, "Esse efeito não está mais no personagem."},
		{"unexpected error", errors.New("db error"), http.StatusInternalServerError, "Não foi possível revelar agora."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, api := humatest.New(t)
			uc := &mockRevealStatusEffect{fn: func(_ context.Context, m, e, u uuid.UUID) error {
				if m != matchUUID || e != effectUUID || u != userUUID {
					t.Errorf("forwarded ids = %v/%v/%v", m, e, u)
				}
				return tt.err
			}}
			huma.Register(api, huma.Operation{
				Method:        http.MethodPost,
				Path:          "/matches/{matchUuid}/status-effects/{effectUuid}/reveal",
				Errors:        []int{http.StatusForbidden, http.StatusNotFound},
				DefaultStatus: http.StatusNoContent,
			}, match.RevealStatusEffectHandler(uc))

			ctx := context.WithValue(context.Background(), auth.UserIDKey, userUUID)
			resp := api.PostCtx(ctx, "/matches/"+matchUUID.String()+"/status-effects/"+effectUUID.String()+"/reveal")
			if resp.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d. Body: %s", resp.Code, tt.wantStatus, resp.Body.String())
			}
			if tt.wantDetail != "" && !strings.Contains(resp.Body.String(), tt.wantDetail) {
				t.Errorf("body %s does not carry %q", resp.Body.String(), tt.wantDetail)
			}
		})
	}
}
