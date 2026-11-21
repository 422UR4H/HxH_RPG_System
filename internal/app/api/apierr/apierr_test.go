package apierr_test

import (
	"bytes"
	"errors"
	"log"
	"strings"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/apierr"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/errtext"
)

func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(log.Writer()) })
	return &buf
}

func TestInternal_HidesTheRealErrorAndLogsIt(t *testing.T) {
	buf := captureLog(t)

	err := apierr.Internal(errors.New("pq: connection refused"))

	se := err
	if se.GetStatus() != 500 {
		t.Fatalf("want a 500 status error, got %v", err)
	}
	if se.Error() != "Algo deu errado. Tente de novo." {
		t.Errorf("detail = %q, want the fixed sentence", se.Error())
	}
	if strings.Contains(se.Error(), "pq") {
		t.Error("the internal error must not reach the response")
	}
	if !strings.Contains(buf.String(), "pq: connection refused") {
		t.Errorf("the original error must be logged, log = %q", buf.String())
	}
}

func TestNoUser_IsAFixedSentence(t *testing.T) {
	captureLog(t)

	se := apierr.NoUser()

	if se.GetStatus() != 500 || se.Error() != "Não foi possível identificar sua conta. Entre de novo." {
		t.Errorf("got %d %q", se.GetStatus(), se.Error())
	}
}

func TestText_SentenceIsShownAndEnglishFallsBack(t *testing.T) {
	buf := captureLog(t)

	if got := apierr.Text(errtext.Sentence("Partida não encontrada.")); got != "Partida não encontrada." {
		t.Errorf("sentence: got %q", got)
	}
	if got := apierr.Text(errors.New("character sheet not found")); got != "Não foi possível fazer isso." {
		t.Errorf("english error: got %q", got)
	}
	if !strings.Contains(buf.String(), "character sheet not found") {
		t.Errorf("the original error must be logged, log = %q", buf.String())
	}
}
