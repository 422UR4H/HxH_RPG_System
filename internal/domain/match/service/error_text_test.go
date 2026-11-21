package service_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
)

func TestSentinelErrors_ReadInPortuguese(t *testing.T) {
	if got, want := service.ErrQueueEmpty.Error(), "A fila de ações está vazia."; got != want {
		t.Errorf("ErrQueueEmpty = %q, want %q", got, want)
	}
}
