package statuseffect_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/statuseffect"
	"github.com/google/uuid"
)

func ip(n int) *int   { return &n }
func bp(b bool) *bool { return &b }

func TestNormalize(t *testing.T) {
	all := statuseffect.AllDomains()
	ctrl := uuid.New()
	long255 := strings.Repeat("ã", 255)
	long256 := strings.Repeat("ã", 256)

	tests := []struct {
		name     string
		kind     statuseffect.Kind
		sense    statuseffect.Sense
		aptitude statuseffect.Aptitude
		in       statuseffect.ParamsInput
		want     statuseffect.Params
		wantErr  bool
	}{
		{name: "paralyzed defaults to the four domains", kind: statuseffect.KindParalyzed,
			in: statuseffect.ParamsInput{Points: ip(3)}, want: statuseffect.Params{Domains: all, Points: 3}},
		{name: "paralyzed keeps chosen domains in declaration order", kind: statuseffect.KindParalyzed,
			in:   statuseffect.ParamsInput{Points: ip(1), Domains: []statuseffect.Domain{statuseffect.DomainDodge, statuseffect.DomainAttack}},
			want: statuseffect.Params{Points: 1, Domains: []statuseffect.Domain{statuseffect.DomainAttack, statuseffect.DomainDodge}}},
		{name: "paralyzed requires points", kind: statuseffect.KindParalyzed, wantErr: true},
		{name: "stupefied requires points", kind: statuseffect.KindStupefied, wantErr: true},
		{name: "stunned requires points", kind: statuseffect.KindStunned, wantErr: true},
		{name: "stupefied with points", kind: statuseffect.KindStupefied,
			in: statuseffect.ParamsInput{Points: ip(2)}, want: statuseffect.Params{Points: 2}},
		{name: "stunned with points", kind: statuseffect.KindStunned,
			in: statuseffect.ParamsInput{Points: ip(2)}, want: statuseffect.Params{Points: 2}},
		{name: "unconscious has no params", kind: statuseffect.KindUnconscious, want: statuseffect.Params{}},
		{name: "restrained has no params", kind: statuseffect.KindRestrained, want: statuseffect.Params{}},
		{name: "asphyxiating has no params", kind: statuseffect.KindAsphyxiating, want: statuseffect.Params{}},
		{name: "prone has no params", kind: statuseffect.KindProne, want: statuseffect.Params{}},
		{name: "airborne has no params", kind: statuseffect.KindAirborne, want: statuseffect.Params{}},
		{name: "open guard has no params", kind: statuseffect.KindOpenGuard, want: statuseffect.Params{}},
		{name: "hidden has no params", kind: statuseffect.KindHidden, want: statuseffect.Params{}},
		{name: "bleeding damage per round defaults to one", kind: statuseffect.KindBleeding, want: statuseffect.Params{HPPerRound: 1}},
		{name: "burning damage per round defaults to one", kind: statuseffect.KindBurning, want: statuseffect.Params{HPPerRound: 1}},
		{name: "freezing damage per round defaults to one", kind: statuseffect.KindFreezing, want: statuseffect.Params{HPPerRound: 1}},
		{name: "poisoned damage per round defaults to one", kind: statuseffect.KindPoisoned, want: statuseffect.Params{HPPerRound: 1}},
		{name: "damage per round keeps the given value", kind: statuseffect.KindPoisoned,
			in: statuseffect.ParamsInput{HPPerRound: ip(4)}, want: statuseffect.Params{HPPerRound: 4}},
		{name: "sense requires a level", kind: statuseffect.KindSense, sense: statuseffect.SenseVision, wantErr: true},
		{name: "impaired sense defaults to disadvantage", kind: statuseffect.KindSense, sense: statuseffect.SenseVision,
			in: statuseffect.ParamsInput{Level: statuseffect.SenseImpaired}, want: statuseffect.Params{Level: statuseffect.SenseImpaired, Disadvantage: true}},
		{name: "impaired sense with only a penalty", kind: statuseffect.KindSense, sense: statuseffect.SenseHearing,
			in:   statuseffect.ParamsInput{Level: statuseffect.SenseImpaired, Disadvantage: bp(false), Penalty: ip(2)},
			want: statuseffect.Params{Level: statuseffect.SenseImpaired, Penalty: 2}},
		{name: "impaired sense needs disadvantage or penalty", kind: statuseffect.KindSense, sense: statuseffect.SenseSmell,
			in: statuseffect.ParamsInput{Level: statuseffect.SenseImpaired, Disadvantage: bp(false)}, wantErr: true},
		{name: "nullified sense refuses disadvantage and penalty", kind: statuseffect.KindSense, sense: statuseffect.SenseTaste,
			in: statuseffect.ParamsInput{Level: statuseffect.SenseNullified, Disadvantage: bp(true)}, wantErr: true},
		{name: "nullified sense refuses penalty", kind: statuseffect.KindSense, sense: statuseffect.SenseTaste,
			in: statuseffect.ParamsInput{Level: statuseffect.SenseNullified, Penalty: ip(1)}, wantErr: true},
		{name: "nullified sense has no extras", kind: statuseffect.KindSense, sense: statuseffect.SenseTaste,
			in: statuseffect.ParamsInput{Level: statuseffect.SenseNullified}, want: statuseffect.Params{Level: statuseffect.SenseNullified}},
		{name: "tact requires at least one face", kind: statuseffect.KindSense, sense: statuseffect.SenseTact,
			in: statuseffect.ParamsInput{Level: statuseffect.SenseNullified}, wantErr: true},
		{name: "tact keeps faces in declaration order", kind: statuseffect.KindSense, sense: statuseffect.SenseTact,
			in: statuseffect.ParamsInput{Level: statuseffect.SenseNullified,
				Faces: []statuseffect.TouchFace{statuseffect.FacePreemption, statuseffect.FaceTouch}},
			want: statuseffect.Params{Level: statuseffect.SenseNullified,
				Faces: []statuseffect.TouchFace{statuseffect.FaceTouch, statuseffect.FacePreemption}}},
		{name: "faces only on tact", kind: statuseffect.KindSense, sense: statuseffect.SenseVision,
			in: statuseffect.ParamsInput{Level: statuseffect.SenseNullified, Faces: []statuseffect.TouchFace{statuseffect.FaceTouch}}, wantErr: true},
		{name: "duplicate faces refused", kind: statuseffect.KindSense, sense: statuseffect.SenseTact,
			in: statuseffect.ParamsInput{Level: statuseffect.SenseNullified,
				Faces: []statuseffect.TouchFace{statuseffect.FaceTouch, statuseffect.FaceTouch}}, wantErr: true},
		{name: "weakened defaults to disadvantage", kind: statuseffect.KindWeakened, aptitude: statuseffect.AptitudePhysical,
			want: statuseffect.Params{Disadvantage: true}},
		{name: "weakened with only a penalty", kind: statuseffect.KindWeakened, aptitude: statuseffect.AptitudeMental,
			in: statuseffect.ParamsInput{Disadvantage: bp(false), Penalty: ip(3)}, want: statuseffect.Params{Penalty: 3}},
		{name: "weakened needs disadvantage or penalty", kind: statuseffect.KindWeakened, aptitude: statuseffect.AptitudeSpiritual,
			in: statuseffect.ParamsInput{Disadvantage: bp(false)}, wantErr: true},
		{name: "weakened requires an aptitude", kind: statuseffect.KindWeakened, wantErr: true},
		{name: "aptitude only on weakened", kind: statuseffect.KindSlowed, aptitude: statuseffect.AptitudePhysical,
			in: statuseffect.ParamsInput{Penalty: ip(1)}, wantErr: true},
		{name: "sense only on sense kind", kind: statuseffect.KindProne, sense: statuseffect.SenseVision, wantErr: true},
		{name: "injured requires at least one domain", kind: statuseffect.KindInjured, wantErr: true},
		{name: "injured keeps per-domain levels", kind: statuseffect.KindInjured,
			in: statuseffect.ParamsInput{Injuries: map[statuseffect.Domain]statuseffect.InjuryLevel{
				statuseffect.DomainAttack: statuseffect.InjuryHurt, statuseffect.DomainMovement: statuseffect.InjuryDisabled}},
			want: statuseffect.Params{Injuries: map[statuseffect.Domain]statuseffect.InjuryLevel{
				statuseffect.DomainAttack: statuseffect.InjuryHurt, statuseffect.DomainMovement: statuseffect.InjuryDisabled}}},
		{name: "injured refuses an unknown level", kind: statuseffect.KindInjured,
			in: statuseffect.ParamsInput{Injuries: map[statuseffect.Domain]statuseffect.InjuryLevel{statuseffect.DomainAttack: "bad"}}, wantErr: true},
		{name: "slowed requires a penalty", kind: statuseffect.KindSlowed, wantErr: true},
		{name: "slowed with a penalty", kind: statuseffect.KindSlowed,
			in: statuseffect.ParamsInput{Penalty: ip(2)}, want: statuseffect.Params{Penalty: 2}},
		{name: "compelled requires a conduct", kind: statuseffect.KindCompelled,
			in: statuseffect.ParamsInput{Conduct: "   "}, wantErr: true},
		{name: "compelled keeps the conduct and irresistible", kind: statuseffect.KindCompelled,
			in:   statuseffect.ParamsInput{Conduct: "obey", Irresistible: bp(true)},
			want: statuseffect.Params{Conduct: "obey", Irresistible: true}},
		{name: "conduct is trimmed and capped at 255 runes (255 fits)", kind: statuseffect.KindCompelled,
			in: statuseffect.ParamsInput{Conduct: "  " + long255 + "  "}, want: statuseffect.Params{Conduct: long255}},
		{name: "conduct is trimmed and capped at 255 runes (256 refused)", kind: statuseffect.KindCompelled,
			in: statuseffect.ParamsInput{Conduct: long256}, wantErr: true},
		{name: "controlled without controller", kind: statuseffect.KindControlled, want: statuseffect.Params{}},
		{name: "controlled with controller", kind: statuseffect.KindControlled,
			in: statuseffect.ParamsInput{ControllerID: &ctrl}, want: statuseffect.Params{ControllerID: &ctrl}},
		{name: "param foreign to the kind is refused (points on prone)", kind: statuseffect.KindProne,
			in: statuseffect.ParamsInput{Points: ip(1)}, wantErr: true},
		{name: "param foreign to the kind is refused (conduct on stunned)", kind: statuseffect.KindStunned,
			in: statuseffect.ParamsInput{Points: ip(1), Conduct: "x"}, wantErr: true},
		{name: "param foreign to the kind is refused (domains on stunned)", kind: statuseffect.KindStunned,
			in: statuseffect.ParamsInput{Points: ip(1), Domains: []statuseffect.Domain{statuseffect.DomainAttack}}, wantErr: true},
		{name: "param foreign to the kind is refused (controller on hidden)", kind: statuseffect.KindHidden,
			in: statuseffect.ParamsInput{ControllerID: &ctrl}, wantErr: true},
		{name: "zero or negative numbers refused (points zero)", kind: statuseffect.KindStunned,
			in: statuseffect.ParamsInput{Points: ip(0)}, wantErr: true},
		{name: "zero or negative numbers refused (penalty zero)", kind: statuseffect.KindSlowed,
			in: statuseffect.ParamsInput{Penalty: ip(0)}, wantErr: true},
		{name: "zero or negative numbers refused (damage zero)", kind: statuseffect.KindBleeding,
			in: statuseffect.ParamsInput{HPPerRound: ip(0)}, wantErr: true},
		{name: "zero or negative numbers refused (negative weakened penalty)", kind: statuseffect.KindWeakened, aptitude: statuseffect.AptitudePhysical,
			in: statuseffect.ParamsInput{Penalty: ip(-1)}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := statuseffect.Normalize(tt.kind, tt.sense, tt.aptitude, tt.in)
			if tt.wantErr {
				if !errors.Is(err, statuseffect.ErrInvalidParams) {
					t.Fatalf("want ErrInvalidParams, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}

	t.Run("unknown kind", func(t *testing.T) {
		_, err := statuseffect.Normalize("nope", "", "", statuseffect.ParamsInput{})
		if !errors.Is(err, statuseffect.ErrUnknownKind) {
			t.Fatalf("want ErrUnknownKind, got %v", err)
		}
	})
}

func TestParse(t *testing.T) {
	if k, err := statuseffect.ParseKind("openGuard"); err != nil || k != statuseffect.KindOpenGuard {
		t.Errorf("ParseKind: %v %v", k, err)
	}
	if _, err := statuseffect.ParseKind("x"); !errors.Is(err, statuseffect.ErrUnknownKind) {
		t.Errorf("ParseKind unknown: %v", err)
	}
	if _, err := statuseffect.ParseSense("x"); !errors.Is(err, statuseffect.ErrInvalidParams) {
		t.Errorf("ParseSense unknown: %v", err)
	}
	if _, err := statuseffect.ParseAptitude("x"); !errors.Is(err, statuseffect.ErrInvalidParams) {
		t.Errorf("ParseAptitude unknown: %v", err)
	}
	if _, err := statuseffect.ParseDomain("x"); !errors.Is(err, statuseffect.ErrInvalidParams) {
		t.Errorf("ParseDomain unknown: %v", err)
	}
	if s, err := statuseffect.ParseSenseLevel("impaired"); err != nil || s != statuseffect.SenseImpaired {
		t.Errorf("ParseSenseLevel: %v %v", s, err)
	}
	if _, err := statuseffect.ParseInjuryLevel("x"); !errors.Is(err, statuseffect.ErrInvalidParams) {
		t.Errorf("ParseInjuryLevel unknown: %v", err)
	}
	if _, err := statuseffect.ParseTouchFace("x"); !errors.Is(err, statuseffect.ErrInvalidParams) {
		t.Errorf("ParseTouchFace unknown: %v", err)
	}
}
