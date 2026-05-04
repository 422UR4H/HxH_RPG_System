# Layer Isolation: eliminar dependências domain/entity → pg/model

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remover todas as violações de importação de `pg/model` fora do `gateway/`, movendo tipos de projeção para `entity/character_sheet/` e invertendo as assinaturas do `IRepository` para usar entidades de domínio ricas.

**Architecture:** Tipos de leitura (`Summary`, `RelationshipUUIDs`, `StatusBar` DTO) movem para `entity/character_sheet/`. `IRepository.CreateCharacterSheet` e `GetCharacterSheetByUUID` passam a usar `*sheet.CharacterSheet`; a lógica de mapeamento (`CharacterSheetToModel`, `Wrap`, `ModelToProfile`) migra do domain para o gateway. Erros gateway-específicos são substituídos por erros de domínio nos dois métodos afetados.

**Tech Stack:** Go 1.23, pgx v5, goose migrations (sem migração necessária aqui).

---

### Task 1: Criar tipos de entidade em `entity/character_sheet/`

**Files:**
- Create: `internal/domain/entity/character_sheet/summary.go`
- Create: `internal/domain/entity/character_sheet/relationship_uuids.go`

- [ ] **Step 1: Criar `summary.go`**

```go
// internal/domain/entity/character_sheet/summary.go
package character_sheet

import (
	"time"

	"github.com/google/uuid"
)

type StatusBar struct {
	Min  int
	Curr int
	Max  int
}

type Summary struct {
	ID             int
	UUID           uuid.UUID
	PlayerUUID     *uuid.UUID
	MasterUUID     *uuid.UUID
	CampaignUUID   *uuid.UUID
	NickName       string
	FullName       string
	Alignment      string
	CharacterClass string
	Birthday       time.Time
	CategoryName   string
	CurrHexValue   *int
	Level          int
	Points         int
	TalentLvl      int
	PhysicalsLvl   int
	MentalsLvl     int
	SpiritualsLvl  int
	SkillsLvl      int
	Stamina        StatusBar
	Health         StatusBar
	Aura           StatusBar
	StoryStartAt   *time.Time
	StoryCurrentAt *time.Time
	DeadAt         *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
```

- [ ] **Step 2: Criar `relationship_uuids.go`**

```go
// internal/domain/entity/character_sheet/relationship_uuids.go
package character_sheet

import "github.com/google/uuid"

type RelationshipUUIDs struct {
	CampaignUUID *uuid.UUID
	PlayerUUID   *uuid.UUID
	MasterUUID   *uuid.UUID
}
```

- [ ] **Step 3: Verificar compilação**

```bash
go build ./internal/domain/entity/character_sheet/...
```

Expected: PASS (novos pacotes sem dependências problemáticas).

- [ ] **Step 4: Commit**

```bash
git add internal/domain/entity/character_sheet/summary.go \
        internal/domain/entity/character_sheet/relationship_uuids.go
git commit -m "feat(entity): add Summary, StatusBar and RelationshipUUIDs to character_sheet entity

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 2: Atualizar `IRepository` e mock

**Files:**
- Modify: `internal/domain/character_sheet/i_repository.go`
- Modify: `internal/domain/testutil/mock_character_sheet_repo.go`

- [ ] **Step 1: Substituir `i_repository.go`**

```go
// internal/domain/character_sheet/i_repository.go
package charactersheet

import (
	"context"

	csEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/status"
	"github.com/google/uuid"
)

type IRepository interface {
	CreateCharacterSheet(ctx context.Context, sheet *sheet.CharacterSheet) error
	ExistsCharacterWithNick(ctx context.Context, nick string) (bool, error)
	CountCharactersByPlayerUUID(ctx context.Context, playerUUID uuid.UUID) (int, error)
	GetCharacterSheetPlayerUUID(ctx context.Context, uuid uuid.UUID) (uuid.UUID, error)
	GetCharacterSheetByUUID(ctx context.Context, uuid string) (*sheet.CharacterSheet, error)
	ListCharacterSheetsByPlayerUUID(ctx context.Context, playerUUID string) ([]csEntity.Summary, error)
	UpdateNenHexagonValue(ctx context.Context, uuid string, val int) error
	GetCharacterSheetRelationshipUUIDs(ctx context.Context, uuid uuid.UUID) (csEntity.RelationshipUUIDs, error)
	ExistsSheetInCampaign(ctx context.Context, playerUUID uuid.UUID, campaignUUID uuid.UUID) (bool, error)
	UpdateStatusBars(ctx context.Context, sheetUUID string, health, stamina, aura status.IStatusBar) error
}
```

- [ ] **Step 2: Substituir `mock_character_sheet_repo.go`**

```go
// internal/domain/testutil/mock_character_sheet_repo.go
package testutil

import (
	"context"

	csEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/status"
	"github.com/google/uuid"
)

type MockCharacterSheetRepo struct {
	CreateCharacterSheetFn               func(ctx context.Context, sheet *sheet.CharacterSheet) error
	ExistsCharacterWithNickFn            func(ctx context.Context, nick string) (bool, error)
	CountCharactersByPlayerUUIDFn        func(ctx context.Context, playerUUID uuid.UUID) (int, error)
	GetCharacterSheetPlayerUUIDFn        func(ctx context.Context, uuid uuid.UUID) (uuid.UUID, error)
	GetCharacterSheetByUUIDFn            func(ctx context.Context, uuid string) (*sheet.CharacterSheet, error)
	ListCharacterSheetsByPlayerUUIDFn    func(ctx context.Context, playerUUID string) ([]csEntity.Summary, error)
	UpdateNenHexagonValueFn              func(ctx context.Context, uuid string, val int) error
	GetCharacterSheetRelationshipUUIDsFn func(ctx context.Context, uuid uuid.UUID) (csEntity.RelationshipUUIDs, error)
	ExistsSheetInCampaignFn              func(ctx context.Context, playerUUID uuid.UUID, campaignUUID uuid.UUID) (bool, error)
	UpdateStatusBarsFn                   func(ctx context.Context, uuid string, health, stamina, aura status.IStatusBar) error
}

func (m *MockCharacterSheetRepo) CreateCharacterSheet(ctx context.Context, s *sheet.CharacterSheet) error {
	if m.CreateCharacterSheetFn != nil {
		return m.CreateCharacterSheetFn(ctx, s)
	}
	return nil
}

func (m *MockCharacterSheetRepo) ExistsCharacterWithNick(ctx context.Context, nick string) (bool, error) {
	if m.ExistsCharacterWithNickFn != nil {
		return m.ExistsCharacterWithNickFn(ctx, nick)
	}
	return false, nil
}

func (m *MockCharacterSheetRepo) CountCharactersByPlayerUUID(ctx context.Context, playerUUID uuid.UUID) (int, error) {
	if m.CountCharactersByPlayerUUIDFn != nil {
		return m.CountCharactersByPlayerUUIDFn(ctx, playerUUID)
	}
	return 0, nil
}

func (m *MockCharacterSheetRepo) GetCharacterSheetPlayerUUID(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
	if m.GetCharacterSheetPlayerUUIDFn != nil {
		return m.GetCharacterSheetPlayerUUIDFn(ctx, id)
	}
	return uuid.Nil, nil
}

func (m *MockCharacterSheetRepo) GetCharacterSheetByUUID(ctx context.Context, id string) (*sheet.CharacterSheet, error) {
	if m.GetCharacterSheetByUUIDFn != nil {
		return m.GetCharacterSheetByUUIDFn(ctx, id)
	}
	return nil, nil
}

func (m *MockCharacterSheetRepo) ListCharacterSheetsByPlayerUUID(ctx context.Context, playerUUID string) ([]csEntity.Summary, error) {
	if m.ListCharacterSheetsByPlayerUUIDFn != nil {
		return m.ListCharacterSheetsByPlayerUUIDFn(ctx, playerUUID)
	}
	return nil, nil
}

func (m *MockCharacterSheetRepo) UpdateNenHexagonValue(ctx context.Context, id string, val int) error {
	if m.UpdateNenHexagonValueFn != nil {
		return m.UpdateNenHexagonValueFn(ctx, id, val)
	}
	return nil
}

func (m *MockCharacterSheetRepo) GetCharacterSheetRelationshipUUIDs(ctx context.Context, id uuid.UUID) (csEntity.RelationshipUUIDs, error) {
	if m.GetCharacterSheetRelationshipUUIDsFn != nil {
		return m.GetCharacterSheetRelationshipUUIDsFn(ctx, id)
	}
	return csEntity.RelationshipUUIDs{}, nil
}

func (m *MockCharacterSheetRepo) ExistsSheetInCampaign(ctx context.Context, playerUUID uuid.UUID, campaignUUID uuid.UUID) (bool, error) {
	if m.ExistsSheetInCampaignFn != nil {
		return m.ExistsSheetInCampaignFn(ctx, playerUUID, campaignUUID)
	}
	return false, nil
}

func (m *MockCharacterSheetRepo) UpdateStatusBars(ctx context.Context, id string, health, stamina, aura status.IStatusBar) error {
	if m.UpdateStatusBarsFn != nil {
		return m.UpdateStatusBarsFn(ctx, id, health, stamina, aura)
	}
	return nil
}
```

- [ ] **Step 3: Verificar que o projeto compila com erros esperados nas implementações**

```bash
go build ./internal/domain/... 2>&1 | head -40
```

Expected: erros de compilação em `gateway/pg/sheet/`, `domain/character_sheet/`, arquivos de teste — isso é esperado. Continuar.

- [ ] **Step 4: Commit**

```bash
git add internal/domain/character_sheet/i_repository.go \
        internal/domain/testutil/mock_character_sheet_repo.go
git commit -m "refactor(domain): update IRepository and mock to use entity types

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 3: Gateway — `update_status_bars.go`

**Files:**
- Modify: `internal/gateway/pg/sheet/update_status_bars.go`

- [ ] **Step 1: Atualizar para receber `status.IStatusBar`**

```go
// internal/gateway/pg/sheet/update_status_bars.go
package sheet

import (
	"context"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/status"
)

func (r *Repository) UpdateStatusBars(
	ctx context.Context,
	sheetUUID string,
	health, stamina, aura status.IStatusBar,
) error {
	const query = `
		UPDATE character_sheets
		SET
			health_min_pts   = $1,
			health_curr_pts  = $2,
			health_max_pts   = $3,
			stamina_min_pts  = $4,
			stamina_curr_pts = $5,
			stamina_max_pts  = $6,
			aura_min_pts     = $7,
			aura_curr_pts    = $8,
			aura_max_pts     = $9,
			updated_at       = $10
		WHERE uuid = $11
	`
	_, err := r.q.Exec(ctx, query,
		health.GetMin(), health.GetCurrent(), health.GetMax(),
		stamina.GetMin(), stamina.GetCurrent(), stamina.GetMax(),
		aura.GetMin(), aura.GetCurrent(), aura.GetMax(),
		time.Now(),
		sheetUUID,
	)
	return err
}
```

- [ ] **Step 2: Verificar compilação do arquivo**

```bash
go build ./internal/gateway/pg/sheet/... 2>&1 | grep update_status
```

Expected: sem erros para este arquivo (outros arquivos do pacote ainda podem ter erros — ignorar por agora).

- [ ] **Step 3: Commit**

```bash
git add internal/gateway/pg/sheet/update_status_bars.go
git commit -m "refactor(gateway): UpdateStatusBars receives status.IStatusBar instead of model.StatusBar

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 4: Gateway — `read_character_sheet.go`

Este é o arquivo mais complexo. Absorve `Wrap` e `ModelToProfile` do domain, atualiza `ListCharacterSheetsByPlayerUUID` e `GetCharacterSheetRelationshipUUIDs`, e muda o retorno de `GetCharacterSheetByUUID` para `*sheet.CharacterSheet`.

**Files:**
- Modify: `internal/gateway/pg/sheet/read_character_sheet.go`

- [ ] **Step 1: Substituir o arquivo completo**

```go
// internal/gateway/pg/sheet/read_character_sheet.go
package sheet

import (
	"context"
	"errors"
	"fmt"
	"math"

	charactersheet "github.com/422UR4H/HxH_RPG_System/internal/domain/character_sheet"
	csEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet"
	domainSheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/experience"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/proficiency"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (r *Repository) GetCharacterSheetByUUID(
	ctx context.Context, uuid string,
) (*domainSheet.CharacterSheet, error) {
	tx, err := r.q.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			// TODO: maybe throws other error
			panic(p)
		}
		_ = tx.Rollback(ctx)
	}()

	const query = `
		SELECT
			cs.id, cs.uuid, cs.player_uuid, cs.master_uuid, cs.campaign_uuid,
			cs.category_name, cs.curr_hex_value, cs.talent_exp,
			cs.health_min_pts, cs.health_curr_pts, cs.health_max_pts,
			cs.stamina_min_pts, cs.stamina_curr_pts, cs.stamina_max_pts,
			cs.aura_min_pts, cs.aura_curr_pts, cs.aura_max_pts,
			cs.resistance_pts, cs.strength_pts, cs.agility_pts, cs.celerity_pts, cs.flexibility_pts, cs.dexterity_pts, cs.sense_pts, cs.constitution_pts,
			cs.resilience_pts, cs.adaptability_pts, cs.weighting_pts, cs.creativity_pts, cs.resilience_exp, cs.adaptability_exp, cs.weighting_exp, cs.creativity_exp,
			cs.vitality_exp, cs.energy_exp, cs.defense_exp, cs.push_exp, cs.grab_exp, cs.carry_exp, cs.velocity_exp, cs.accelerate_exp, cs.brake_exp,
			cs.legerity_exp, cs.repel_exp, cs.feint_exp, cs.acrobatics_exp, cs.evasion_exp, cs.sneak_exp, cs.reflex_exp, cs.accuracy_exp, cs.stealth_exp,
			cs.vision_exp, cs.hearing_exp, cs.smell_exp, cs.tact_exp, cs.taste_exp, cs.heal_exp, cs.breath_exp, cs.tenacity_exp,
			cs.nen_exp, cs.focus_exp, cs.will_power_exp,
			cs.ten_exp, cs.zetsu_exp, cs.ren_exp, cs.gyo_exp, cs.shu_exp, cs.kou_exp, cs.ken_exp, cs.ryu_exp, cs.in_exp, cs.en_exp, cs.aura_control_exp, cs.aop_exp,
			cs.reinforcement_exp, cs.transmutation_exp, cs.materialization_exp, cs.specialization_exp, cs.manipulation_exp, cs.emission_exp,
			cs.created_at, cs.updated_at,
			cp.uuid, cp.nickname, cp.fullname, cp.alignment, cp.character_class, cp.long_description, cp.brief_description, cp.birthday,
			cp.created_at, cp.updated_at
		FROM character_sheets cs
		JOIN character_profiles cp ON cs.uuid = cp.character_sheet_uuid
		WHERE cs.uuid = $1
	`
	row := tx.QueryRow(ctx, query, uuid)

	var m model.CharacterSheet
	var profile model.CharacterProfile

	err = row.Scan(
		&m.ID, &m.UUID, &m.PlayerUUID, &m.MasterUUID, &m.CampaignUUID,
		&m.CategoryName, &m.CurrHexValue, &m.TalentExp,
		&m.Health.Min, &m.Health.Curr, &m.Health.Max,
		&m.Stamina.Min, &m.Stamina.Curr, &m.Stamina.Max,
		&m.Aura.Min, &m.Aura.Curr, &m.Aura.Max,
		&m.ResistancePts, &m.StrengthPts, &m.AgilityPts, &m.CelerityPts, &m.FlexibilityPts, &m.DexterityPts, &m.SensePts, &m.ConstitutionPts,
		&m.ResiliencePts, &m.AdaptabilityPts, &m.WeightingPts, &m.CreativityPts, &m.ResilienceExp, &m.AdaptabilityExp, &m.WeightingExp, &m.CreativityExp,
		&m.VitalityExp, &m.EnergyExp, &m.DefenseExp, &m.PushExp, &m.GrabExp, &m.CarryExp, &m.VelocityExp, &m.AccelerateExp, &m.BrakeExp,
		&m.LegerityExp, &m.RepelExp, &m.FeintExp, &m.AcrobaticsExp, &m.EvasionExp, &m.SneakExp, &m.ReflexExp, &m.AccuracyExp, &m.StealthExp,
		&m.VisionExp, &m.HearingExp, &m.SmellExp, &m.TactExp, &m.TasteExp, &m.HealExp, &m.BreathExp, &m.TenacityExp,
		&m.NenExp, &m.FocusExp, &m.WillPowerExp,
		&m.TenExp, &m.ZetsuExp, &m.RenExp, &m.GyoExp, &m.ShuExp, &m.KouExp, &m.KenExp, &m.RyuExp, &m.InExp, &m.EnExp, &m.AuraControlExp, &m.AopExp,
		&m.ReinforcementExp, &m.TransmutationExp, &m.MaterializationExp, &m.SpecializationExp, &m.ManipulationExp, &m.EmissionExp,
		&m.CreatedAt, &m.UpdatedAt,
		&profile.UUID, &profile.NickName, &profile.FullName, &profile.Alignment, &profile.CharacterClass, &profile.Description, &profile.BriefDescription, &profile.Birthday,
		&profile.CreatedAt, &profile.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, charactersheet.ErrCharacterSheetNotFound
		}
		return nil, fmt.Errorf("failed to fetch character sheet: %w", err)
	}
	m.Profile = profile

	const proficienciesQuery = `
		SELECT weapon, exp
		FROM proficiencies
		WHERE character_sheet_uuid = $1
	`
	rows, err := tx.Query(ctx, proficienciesQuery, uuid)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch proficiencies: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var prof model.Proficiency
		if err := rows.Scan(&prof.Weapon, &prof.Exp); err != nil {
			return nil, fmt.Errorf("failed to scan proficiency: %w", err)
		}
		m.Proficiencies = append(m.Proficiencies, prof)
	}

	const jointProficienciesQuery = `
		SELECT name, weapons, exp
		FROM joint_proficiencies
		WHERE character_sheet_uuid = $1
	`
	rows, err = tx.Query(ctx, jointProficienciesQuery, uuid)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch joint proficiencies: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var jointProf model.JointProficiency
		if err := rows.Scan(&jointProf.Name, &jointProf.Weapons, &jointProf.Exp); err != nil {
			return nil, fmt.Errorf("failed to scan joint proficiency: %w", err)
		}
		m.JointProficiencies = append(m.JointProficiencies, jointProf)
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	domainProfile := modelToProfile(&m.Profile)
	categoryName := (*enum.CategoryName)(&m.CategoryName)
	factory := domainSheet.NewCharacterSheetFactory()
	charSheet, err := factory.Build(
		m.PlayerUUID,
		m.MasterUUID,
		m.CampaignUUID,
		*domainProfile,
		m.CurrHexValue,
		categoryName,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to build character sheet entity: %w", err)
	}

	charClass, err := enum.CharacterClassNameFrom(m.Profile.CharacterClass)
	if err != nil {
		return nil, err
	}
	if err := charSheet.AddDryCharacterClass(&charClass); err != nil {
		return nil, err
	}

	if _, err := wrap(charSheet, &m); err != nil {
		return nil, err
	}
	return charSheet, nil
}

func (r *Repository) GetCharacterSheetPlayerUUID(
	ctx context.Context, sheet_uuid uuid.UUID,
) (uuid.UUID, error) {
	const query = `
		SELECT player_uuid
		FROM character_sheets
		WHERE uuid = $1
	`
	var playerUUID uuid.UUID
	err := r.q.QueryRow(ctx, query, sheet_uuid).Scan(&playerUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, ErrCharacterSheetNotFound
		}
		return uuid.Nil, fmt.Errorf("failed to fetch character sheet player UUID: %w", err)
	}
	return playerUUID, nil
}

func (r *Repository) ListCharacterSheetsByPlayerUUID(
	ctx context.Context, playerUUID string,
) ([]csEntity.Summary, error) {
	const query = `
		SELECT
			cs.id, cs.uuid, cs.player_uuid, cs.master_uuid, cs.campaign_uuid,
			cs.category_name, cs.curr_hex_value,
			COALESCE(cs.level, 0), COALESCE(cs.points, 0),
			COALESCE(cs.talent_lvl, 0), COALESCE(cs.skills_lvl, 0),
			COALESCE(cs.physicals_lvl, 0), COALESCE(cs.mentals_lvl, 0), COALESCE(cs.spirituals_lvl, 0),
			COALESCE(cs.health_min_pts, 0), COALESCE(cs.health_curr_pts, 0), COALESCE(cs.health_max_pts, 0),
			COALESCE(cs.stamina_min_pts, 0), COALESCE(cs.stamina_curr_pts, 0), COALESCE(cs.stamina_max_pts, 0),
			COALESCE(cs.aura_min_pts, 0), COALESCE(cs.aura_curr_pts, 0), COALESCE(cs.aura_max_pts, 0),
			cs.story_start_at, cs.story_current_at, cs.dead_at,
			cs.created_at, cs.updated_at,
			cp.nickname, cp.fullname, cp.alignment, cp.character_class, cp.birthday
		FROM character_sheets cs
		JOIN character_profiles cp ON cs.uuid = cp.character_sheet_uuid
		WHERE cs.player_uuid = $1
		ORDER BY cp.nickname ASC
	`
	rows, err := r.q.Query(ctx, query, playerUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to list character sheets: %w", err)
	}
	defer rows.Close()

	var out []csEntity.Summary
	for rows.Next() {
		var s csEntity.Summary
		err := rows.Scan(
			&s.ID, &s.UUID, &s.PlayerUUID, &s.MasterUUID, &s.CampaignUUID,
			&s.CategoryName, &s.CurrHexValue,
			&s.Level, &s.Points, &s.TalentLvl, &s.SkillsLvl,
			&s.PhysicalsLvl, &s.MentalsLvl, &s.SpiritualsLvl,
			&s.Health.Min, &s.Health.Curr, &s.Health.Max,
			&s.Stamina.Min, &s.Stamina.Curr, &s.Stamina.Max,
			&s.Aura.Min, &s.Aura.Curr, &s.Aura.Max,
			&s.StoryStartAt, &s.StoryCurrentAt, &s.DeadAt,
			&s.CreatedAt, &s.UpdatedAt,
			&s.NickName, &s.FullName, &s.Alignment, &s.CharacterClass, &s.Birthday,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan character sheet summary: %w", err)
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}
	return out, nil
}

func (r *Repository) GetCharacterSheetRelationshipUUIDs(
	ctx context.Context, sheet_uuid uuid.UUID,
) (csEntity.RelationshipUUIDs, error) {
	const query = `
		SELECT player_uuid, master_uuid, campaign_uuid
		FROM character_sheets
		WHERE uuid = $1
	`
	var rel csEntity.RelationshipUUIDs
	err := r.q.QueryRow(ctx, query, sheet_uuid).Scan(
		&rel.PlayerUUID,
		&rel.MasterUUID,
		&rel.CampaignUUID,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return csEntity.RelationshipUUIDs{}, charactersheet.ErrCharacterSheetNotFound
		}
		return csEntity.RelationshipUUIDs{},
			fmt.Errorf("failed to fetch character sheet relationship UUIDs: %w", err)
	}
	return rel, nil
}

// modelToProfile converts a pg/model CharacterProfile to the domain entity profile.
func modelToProfile(profile *model.CharacterProfile) *domainSheet.CharacterProfile {
	return &domainSheet.CharacterProfile{
		NickName:         profile.NickName,
		FullName:         profile.FullName,
		Alignment:        profile.Alignment,
		Description:      profile.Description,
		BriefDescription: profile.BriefDescription,
		Birthday:         profile.Birthday,
	}
}

// wrap populates charSheet with experience and status values from the DB model.
func wrap(charSheet *domainSheet.CharacterSheet, m *model.CharacterSheet) (wasCorrected bool, err error) {
	charSheet.UUID = m.UUID

	physicalAttrs := map[enum.AttributeName]int{
		enum.Resistance:   m.ResistancePts,
		enum.Strength:     m.StrengthPts,
		enum.Agility:      m.AgilityPts,
		enum.Celerity:     m.CelerityPts,
		enum.Flexibility:  m.FlexibilityPts,
		enum.Dexterity:    m.DexterityPts,
		enum.Sense:        m.SensePts,
		enum.Constitution: m.ConstitutionPts,
	}
	for name, points := range physicalAttrs {
		if points == 0 {
			continue
		}
		if _, _, err := charSheet.IncreasePtsForPhysPrimaryAttr(name, points); err != nil {
			return false, fmt.Errorf("%w %s: %v", domainSheet.ErrFailedToIncreasePhysAttrPts, name, err)
		}
	}

	// TODO: add mental attributes points or remove from modelSheet
	mentalAttrs := map[enum.AttributeName]int{
		enum.Resilience:   m.ResilienceExp,
		enum.Adaptability: m.AdaptabilityExp,
		enum.Weighting:    m.WeightingExp,
		enum.Creativity:   m.CreativityExp,
	}
	for name, exp := range mentalAttrs {
		if exp == 0 {
			continue
		}
		if err := charSheet.IncreaseExpForMentals(experience.NewUpgradeCascade(exp), name); err != nil {
			return false, fmt.Errorf("%w %s: %v", domainSheet.ErrFailedToIncreaseMentalExp, name, err)
		}
	}

	physicalSkills := map[enum.SkillName]int{
		enum.Vitality:   m.VitalityExp,
		enum.Energy:     m.EnergyExp,
		enum.Defense:    m.DefenseExp,
		enum.Push:       m.PushExp,
		enum.Grab:       m.GrabExp,
		enum.Carry:      m.CarryExp,
		enum.Velocity:   m.VelocityExp,
		enum.Accelerate: m.AccelerateExp,
		enum.Brake:      m.BrakeExp,
		enum.Legerity:   m.LegerityExp,
		enum.Repel:      m.RepelExp,
		enum.Feint:      m.FeintExp,
		enum.Acrobatics: m.AcrobaticsExp,
		enum.Evasion:    m.EvasionExp,
		enum.Sneak:      m.SneakExp,
		enum.Reflex:     m.ReflexExp,
		enum.Accuracy:   m.AccuracyExp,
		enum.Stealth:    m.StealthExp,
		enum.Vision:     m.VisionExp,
		enum.Hearing:    m.HearingExp,
		enum.Smell:      m.SmellExp,
		enum.Tact:       m.TactExp,
		enum.Taste:      m.TasteExp,
		enum.Heal:       m.HealExp,
		enum.Breath:     m.BreathExp,
		enum.Tenacity:   m.TenacityExp,
	}
	for name, exp := range physicalSkills {
		if exp == 0 {
			continue
		}
		if err := charSheet.IncreaseExpForSkill(experience.NewUpgradeCascade(exp), name); err != nil {
			return false, fmt.Errorf("%w %s: %v", domainSheet.ErrFailedToIncreaseSkillExp, name, err)
		}
	}

	spiritualPrinciples := map[enum.PrincipleName]int{
		enum.Ten:   m.TenExp,
		enum.Zetsu: m.ZetsuExp,
		enum.Ren:   m.RenExp,
		enum.Gyo:   m.GyoExp,
		enum.Shu:   m.ShuExp,
		enum.Kou:   m.KouExp,
		enum.Ken:   m.KenExp,
		enum.Ryu:   m.RyuExp,
		enum.In:    m.InExp,
		enum.En:    m.EnExp,
	}
	for name, exp := range spiritualPrinciples {
		if exp == 0 {
			continue
		}
		if err := charSheet.IncreaseExpForPrinciple(experience.NewUpgradeCascade(exp), name); err != nil {
			return false, fmt.Errorf("%w %s: %v", domainSheet.ErrFailedToIncreasePrincipleExp, name, err)
		}
	}

	spiritualCategories := map[enum.CategoryName]int{
		enum.Reinforcement:   m.ReinforcementExp,
		enum.Transmutation:   m.TransmutationExp,
		enum.Materialization: m.MaterializationExp,
		enum.Specialization:  m.SpecializationExp,
		enum.Manipulation:    m.ManipulationExp,
		enum.Emission:        m.EmissionExp,
	}
	for name, exp := range spiritualCategories {
		if exp == 0 {
			continue
		}
		if err := charSheet.IncreaseExpForCategory(experience.NewUpgradeCascade(exp), name); err != nil {
			return false, fmt.Errorf("%w %s: %v", domainSheet.ErrFailedToIncreaseCategoryExp, name, err)
		}
	}

	type statusEntry struct {
		name   enum.StatusName
		curr   int
		oldMax int
	}
	for _, e := range []statusEntry{
		{enum.Health, m.Health.Curr, m.Health.Max},
		{enum.Stamina, m.Stamina.Curr, m.Stamina.Max},
		{enum.Aura, m.Aura.Curr, m.Aura.Max},
	} {
		newMax, err := charSheet.GetMaxOfStatus(e.name)
		if err != nil {
			return false, fmt.Errorf("%w (%s): %v", domainSheet.ErrFailedToGetStatus, e.name, err)
		}
		minVal, err := charSheet.GetMinOfStatus(e.name)
		if err != nil {
			return false, fmt.Errorf("%w (%s): %v", domainSheet.ErrFailedToGetStatus, e.name, err)
		}
		corrected, correctionApplied := normalizeStatus(e.curr, e.oldMax, newMax, minVal)
		if correctionApplied {
			wasCorrected = true
		}
		if newMax == 0 {
			continue
		}
		if err := charSheet.SetCurrStatus(e.name, corrected); err != nil {
			return false, fmt.Errorf("%w (%s): %v", domainSheet.ErrFailedToSetStatus, e.name, err)
		}
	}

	physSkExp, err := charSheet.GetPhysSkillExpReference()
	if err != nil {
		return false, domainSheet.ErrFailedToGetPhysSkillExpRef
	}
	expTable := experience.NewExpTable(domainSheet.PHYSICAL_SKILLS_COEFF)
	newExp := experience.NewExperience(expTable)
	for _, prof := range m.Proficiencies {
		domainProf := proficiency.NewProficiency(
			enum.WeaponName(prof.Weapon), *newExp, physSkExp,
		)
		if err := charSheet.AddCommonProficiency(enum.WeaponName(prof.Weapon), domainProf); err != nil {
			return false, fmt.Errorf("%w: %v", domainSheet.ErrFailedToAddCommonProficiency, err)
		}
		if err := charSheet.IncreaseExpForProficiency(
			experience.NewUpgradeCascade(prof.Exp), enum.WeaponName(prof.Weapon),
		); err != nil {
			return false, fmt.Errorf("%w: %v", domainSheet.ErrFailedToIncreaseProficiencyExp, err)
		}
	}

	for _, jointProf := range m.JointProficiencies {
		weapons := []enum.WeaponName{}
		for _, weapon := range jointProf.Weapons {
			weapons = append(weapons, enum.WeaponName(weapon))
		}
		domainJointProf := proficiency.NewJointProficiency(
			*newExp, jointProf.Name, weapons,
		)
		if err := charSheet.AddJointProficiency(domainJointProf); err != nil {
			return false, fmt.Errorf("%w: %v", domainSheet.ErrFailedToAddJointProficiency, err)
		}
		// TODO: implement for create and add here too
	}

	charSheet.IncreaseExpForTalent(m.TalentExp)
	return wasCorrected, nil
}

// normalizeStatus corrects curr when it exceeds the recalculated newMax.
func normalizeStatus(curr, oldMax, newMax, minVal int) (int, bool) {
	if newMax == 0 {
		fmt.Printf("TODO(logger): status normalized anomaly: newMax is 0, curr %d not corrected\n", curr)
		return curr, false
	}
	if curr <= newMax {
		return curr, false
	}
	if oldMax <= 0 {
		fmt.Printf("TODO(logger): status normalized (fallback): curr %d → new_max %d\n", curr, newMax)
		return newMax, true
	}
	corrected := int(math.Round(float64(newMax) * float64(curr) / float64(oldMax)))
	corrected = max(minVal, min(newMax, corrected))
	fmt.Printf("TODO(logger): status normalized: curr %d → %d (old_max: %d, new_max: %d)\n", curr, corrected, oldMax, newMax)
	return corrected, true
}
```

- [ ] **Step 2: Verificar compilação do pacote**

```bash
go build ./internal/gateway/pg/sheet/... 2>&1 | grep -v create_character_sheet
```

Expected: erros apenas em `create_character_sheet.go` (ainda usa assinatura antiga). Os outros arquivos devem compilar.

- [ ] **Step 3: Commit**

```bash
git add internal/gateway/pg/sheet/read_character_sheet.go
git commit -m "refactor(gateway): absorb Wrap/ModelToProfile into sheet repository

GetCharacterSheetByUUID now returns *sheet.CharacterSheet directly.
GetCharacterSheetRelationshipUUIDs returns csEntity.RelationshipUUIDs.
Both return charactersheet.ErrCharacterSheetNotFound on not-found.
ListCharacterSheetsByPlayerUUID scans into csEntity.Summary.

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 5: Gateway — `create_character_sheet.go`

Absorve `CharacterSheetToModel` do domain. Recebe `*sheet.CharacterSheet` e mapeia internamente.

**Files:**
- Modify: `internal/gateway/pg/sheet/create_character_sheet.go`

- [ ] **Step 1: Substituir o arquivo**

```go
// internal/gateway/pg/sheet/create_character_sheet.go
package sheet

import (
	"context"
	"fmt"
	"time"

	domainSheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/model"
	"github.com/google/uuid"
)

func (r *Repository) CreateCharacterSheet(
	ctx context.Context, sheet *domainSheet.CharacterSheet,
) error {
	m := charSheetToModel(sheet)

	tx, err := r.q.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			// TODO: maybe throws other error
			panic(p)
		}
		_ = tx.Rollback(ctx)
	}()

	const sheetQuery = `
		INSERT INTO character_sheets (
			uuid, player_uuid, category_name, curr_hex_value, talent_exp,
			level, points, talent_lvl, physicals_lvl, mentals_lvl, spirituals_lvl, skills_lvl,
			health_min_pts, health_curr_pts, health_max_pts,
			stamina_min_pts, stamina_curr_pts, stamina_max_pts,
			aura_min_pts, aura_curr_pts, aura_max_pts,
			resistance_pts, strength_pts, agility_pts, celerity_pts, flexibility_pts, dexterity_pts, sense_pts, constitution_pts,
			resilience_pts, adaptability_pts, weighting_pts, creativity_pts, resilience_exp, adaptability_exp, weighting_exp, creativity_exp,
			vitality_exp, energy_exp, defense_exp, push_exp, grab_exp, carry_exp, velocity_exp, accelerate_exp, brake_exp,
			legerity_exp, repel_exp, feint_exp, acrobatics_exp, evasion_exp, sneak_exp, reflex_exp, accuracy_exp, stealth_exp,
			vision_exp, hearing_exp, smell_exp, tact_exp, taste_exp, heal_exp, breath_exp, tenacity_exp,
			nen_exp, focus_exp, will_power_exp,
			ten_exp, zetsu_exp, ren_exp, gyo_exp, shu_exp, kou_exp, ken_exp, ryu_exp, in_exp, en_exp, aura_control_exp, aop_exp,
			reinforcement_exp, transmutation_exp, materialization_exp, specialization_exp, manipulation_exp, emission_exp,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10, $11, $12,
			$13, $14, $15,
			$16, $17, $18,
			$19, $20, $21,
			$22, $23, $24, $25, $26, $27, $28, $29,
			$30, $31, $32, $33, $34, $35, $36, $37,
			$38, $39, $40, $41, $42, $43, $44, $45, $46,
			$47, $48, $49, $50, $51, $52, $53, $54, $55,
			$56, $57, $58, $59, $60, $61, $62, $63,
			$64, $65, $66,
			$67, $68, $69, $70, $71, $72, $73, $74, $75, $76, $77,
			$78, $79, $80, $81, $82, $83,
			$84, $85, $86
		) RETURNING id
	`
	var sheetID int
	err = tx.QueryRow(ctx, sheetQuery,
		m.UUID, m.PlayerUUID, m.CategoryName, m.CurrHexValue, m.TalentExp,
		m.Level, m.Points, m.TalentLvl, m.PhysicalsLvl, m.MentalsLvl, m.SpiritualsLvl, m.SkillsLvl,
		m.Health.Min, m.Health.Curr, m.Health.Max,
		m.Stamina.Min, m.Stamina.Curr, m.Stamina.Max,
		m.Aura.Min, m.Aura.Curr, m.Aura.Max,
		m.ResistancePts, m.StrengthPts, m.AgilityPts, m.CelerityPts, m.FlexibilityPts, m.DexterityPts, m.SensePts, m.ConstitutionPts,
		m.ResiliencePts, m.AdaptabilityPts, m.WeightingPts, m.CreativityPts, m.ResilienceExp, m.AdaptabilityExp, m.WeightingExp, m.CreativityExp,
		m.VitalityExp, m.EnergyExp, m.DefenseExp, m.PushExp, m.GrabExp, m.CarryExp, m.VelocityExp, m.AccelerateExp, m.BrakeExp,
		m.LegerityExp, m.RepelExp, m.FeintExp, m.AcrobaticsExp, m.EvasionExp, m.SneakExp, m.ReflexExp, m.AccuracyExp, m.StealthExp,
		m.VisionExp, m.HearingExp, m.SmellExp, m.TactExp, m.TasteExp, m.HealExp, m.BreathExp, m.TenacityExp,
		m.NenExp, m.FocusExp, m.WillPowerExp,
		m.TenExp, m.ZetsuExp, m.RenExp, m.GyoExp, m.ShuExp, m.KouExp, m.KenExp, m.RyuExp, m.InExp, m.EnExp, m.AuraControlExp, m.AopExp,
		m.ReinforcementExp, m.TransmutationExp, m.MaterializationExp, m.SpecializationExp, m.ManipulationExp, m.EmissionExp,
		m.CreatedAt, m.UpdatedAt,
	).Scan(&sheetID)
	if err != nil {
		return fmt.Errorf("failed to save character sheet: %w", err)
	}

	const profileQuery = `
		INSERT INTO character_profiles (
			uuid, character_sheet_uuid, nickname, fullname, alignment, character_class, long_description, brief_description, birthday, age, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
		)
	`
	_, err = tx.Exec(ctx, profileQuery,
		m.Profile.UUID, m.UUID, m.Profile.NickName, m.Profile.FullName, m.Profile.Alignment,
		m.Profile.CharacterClass, m.Profile.Description, m.Profile.BriefDescription, m.Profile.Birthday, m.Profile.Age,
		m.Profile.CreatedAt, m.Profile.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to save character profile: %w", err)
	}

	const proficienciesQuery = `
		INSERT INTO proficiencies (
			character_sheet_uuid, weapon, exp
		) VALUES (
			$1, $2, $3
		)
	`
	for _, proficiency := range m.Proficiencies {
		_, err = tx.Exec(ctx, proficienciesQuery,
			m.UUID, proficiency.Weapon, proficiency.Exp,
		)
		if err != nil {
			return fmt.Errorf("failed to save proficiency %s: %w", proficiency.Weapon, err)
		}
	}

	const jointProficienciesQuery = `
		INSERT INTO joint_proficiencies (
			character_sheet_uuid, name, weapons, exp
		) VALUES (
			$1, $2, $3, $4
		)
	`
	for _, jointProficiency := range m.JointProficiencies {
		_, err = tx.Exec(ctx, jointProficienciesQuery,
			m.UUID, jointProficiency.Name, jointProficiency.Weapons, jointProficiency.Exp,
		)
		if err != nil {
			return fmt.Errorf("failed to save joint proficiency %s: %w", jointProficiency.Name, err)
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

// charSheetToModel converts a domain CharacterSheet entity to the pg/model used for persistence.
// TODO: refactor these maps (inherited from domain use case)
func charSheetToModel(sheet *domainSheet.CharacterSheet) *model.CharacterSheet {
	now := time.Now()
	profile := sheet.GetProfile()
	physAttrs := sheet.GetPhysicalAttributes()
	mentalAttrs := sheet.GetMentalAttributes()
	physSkills := sheet.GetPhysicalSkills()
	principles := sheet.GetPrinciples()
	categories := sheet.GetCategories()
	statusBars := sheet.GetAllStatusBar()
	profs := sheet.GetCommonProficiencies()
	jointProfs := sheet.GetJointProficiencies()

	categoryName, err := sheet.GetCategoryName()
	categoryString := ""
	if err != nil {
		categoryString = categoryName.String()
	}

	physicalsLvl, _ := sheet.GetLevelOfAbility(enum.Physicals)
	mentalsLvl, _ := sheet.GetLevelOfAbility(enum.Mentals)
	spiritualsLvl, _ := sheet.GetLevelOfAbility(enum.Spirituals)
	skillsLvl, _ := sheet.GetLevelOfAbility(enum.Skills)

	modelProfs := []model.Proficiency{}
	for weapon, prof := range profs {
		modelProfs = append(modelProfs, model.Proficiency{
			UUID:      uuid.New(),
			Weapon:    weapon.String(),
			Exp:       prof.GetExpPoints(),
			CreatedAt: now,
			UpdatedAt: now,
		})
	}

	modelJointProfs := []model.JointProficiency{}
	for name, prof := range jointProfs {
		weapons := []string{}
		for _, weapon := range prof.GetWeapons() {
			weapons = append(weapons, weapon.String())
		}
		modelJointProfs = append(modelJointProfs, model.JointProficiency{
			UUID:      uuid.New(),
			Name:      name,
			Exp:       prof.GetExpPoints(),
			Weapons:   weapons,
			CreatedAt: now,
			UpdatedAt: now,
		})
	}

	playerUUID := sheet.GetPlayerUUID()
	return &model.CharacterSheet{
		UUID:       sheet.UUID,
		PlayerUUID: playerUUID,

		Profile: model.CharacterProfile{
			UUID:             uuid.New(),
			CharacterClass:   sheet.GetCharacterClass().String(),
			NickName:         profile.NickName,
			FullName:         profile.FullName,
			Alignment:        profile.Alignment,
			Description:      profile.Description,
			BriefDescription: profile.BriefDescription,
			Birthday:         profile.Birthday,
			Age:              profile.Age,
			CreatedAt:        now,
			UpdatedAt:        now,
		},
		CategoryName: categoryString,
		CurrHexValue: sheet.GetCurrHexValue(),
		TalentExp:    sheet.GetTalentExpPoints(),

		Level:         sheet.GetLevel(),
		Points:        sheet.GetCharacterPoints(),
		TalentLvl:     sheet.GetTalentLevel(),
		PhysicalsLvl:  physicalsLvl,
		MentalsLvl:    mentalsLvl,
		SpiritualsLvl: spiritualsLvl,
		SkillsLvl:     skillsLvl,

		Health: model.StatusBar{
			Min:  statusBars[enum.Health].GetMin(),
			Curr: statusBars[enum.Health].GetCurrent(),
			Max:  statusBars[enum.Health].GetMax(),
		},
		Stamina: model.StatusBar{
			Min:  statusBars[enum.Stamina].GetMin(),
			Curr: statusBars[enum.Stamina].GetCurrent(),
			Max:  statusBars[enum.Stamina].GetMax(),
		},
		// Aura: model.StatusBar{...},

		ResistancePts:   physAttrs[enum.Resistance].GetPoints(),
		StrengthPts:     physAttrs[enum.Strength].GetPoints(),
		AgilityPts:      physAttrs[enum.Agility].GetPoints(),
		CelerityPts:     physAttrs[enum.Celerity].GetPoints(),
		FlexibilityPts:  physAttrs[enum.Flexibility].GetPoints(),
		DexterityPts:    physAttrs[enum.Dexterity].GetPoints(),
		SensePts:        physAttrs[enum.Sense].GetPoints(),
		ConstitutionPts: physAttrs[enum.Constitution].GetPoints(),

		ResiliencePts:   mentalAttrs[enum.Resilience].GetPoints(),
		AdaptabilityPts: mentalAttrs[enum.Adaptability].GetPoints(),
		WeightingPts:    mentalAttrs[enum.Weighting].GetPoints(),
		CreativityPts:   mentalAttrs[enum.Creativity].GetPoints(),
		ResilienceExp:   mentalAttrs[enum.Resilience].GetExpPoints(),
		AdaptabilityExp: mentalAttrs[enum.Adaptability].GetExpPoints(),
		WeightingExp:    mentalAttrs[enum.Weighting].GetExpPoints(),
		CreativityExp:   mentalAttrs[enum.Creativity].GetExpPoints(),

		VitalityExp:   physSkills[enum.Vitality].GetExpPoints(),
		EnergyExp:     physSkills[enum.Energy].GetExpPoints(),
		DefenseExp:    physSkills[enum.Defense].GetExpPoints(),
		PushExp:       physSkills[enum.Push].GetExpPoints(),
		GrabExp:       physSkills[enum.Grab].GetExpPoints(),
		CarryExp:      physSkills[enum.Carry].GetExpPoints(),
		VelocityExp:   physSkills[enum.Velocity].GetExpPoints(),
		AccelerateExp: physSkills[enum.Accelerate].GetExpPoints(),
		BrakeExp:      physSkills[enum.Brake].GetExpPoints(),
		LegerityExp:   physSkills[enum.Legerity].GetExpPoints(),
		RepelExp:      physSkills[enum.Repel].GetExpPoints(),
		FeintExp:      physSkills[enum.Feint].GetExpPoints(),
		AcrobaticsExp: physSkills[enum.Acrobatics].GetExpPoints(),
		EvasionExp:    physSkills[enum.Evasion].GetExpPoints(),
		SneakExp:      physSkills[enum.Sneak].GetExpPoints(),
		ReflexExp:     physSkills[enum.Reflex].GetExpPoints(),
		AccuracyExp:   physSkills[enum.Accuracy].GetExpPoints(),
		StealthExp:    physSkills[enum.Stealth].GetExpPoints(),
		VisionExp:     physSkills[enum.Vision].GetExpPoints(),
		HearingExp:    physSkills[enum.Hearing].GetExpPoints(),
		SmellExp:      physSkills[enum.Smell].GetExpPoints(),
		TactExp:       physSkills[enum.Tact].GetExpPoints(),
		TasteExp:      physSkills[enum.Taste].GetExpPoints(),
		HealExp:       physSkills[enum.Heal].GetExpPoints(),
		BreathExp:     physSkills[enum.Breath].GetExpPoints(),
		TenacityExp:   physSkills[enum.Tenacity].GetExpPoints(),

		TenExp:   principles[enum.Ten].GetExpPoints(),
		ZetsuExp: principles[enum.Zetsu].GetExpPoints(),
		RenExp:   principles[enum.Ren].GetExpPoints(),
		GyoExp:   principles[enum.Gyo].GetExpPoints(),
		ShuExp:   principles[enum.Shu].GetExpPoints(),
		KouExp:   principles[enum.Kou].GetExpPoints(),
		KenExp:   principles[enum.Ken].GetExpPoints(),
		RyuExp:   principles[enum.Ryu].GetExpPoints(),
		InExp:    principles[enum.In].GetExpPoints(),
		EnExp:    principles[enum.En].GetExpPoints(),

		ReinforcementExp:   categories[enum.Reinforcement].GetExpPoints(),
		TransmutationExp:   categories[enum.Transmutation].GetExpPoints(),
		MaterializationExp: categories[enum.Materialization].GetExpPoints(),
		SpecializationExp:  categories[enum.Specialization].GetExpPoints(),
		ManipulationExp:    categories[enum.Manipulation].GetExpPoints(),
		EmissionExp:        categories[enum.Emission].GetExpPoints(),

		Proficiencies:      modelProfs,
		JointProficiencies: modelJointProfs,

		CreatedAt: now,
		UpdatedAt: now,
	}
}
```

- [ ] **Step 2: Verificar compilação completa do gateway/pg/sheet**

```bash
go build ./internal/gateway/pg/sheet/...
```

Expected: PASS sem erros.

- [ ] **Step 3: Commit**

```bash
git add internal/gateway/pg/sheet/create_character_sheet.go
git commit -m "refactor(gateway): absorb CharacterSheetToModel into sheet repository

CreateCharacterSheet now receives *sheet.CharacterSheet and maps internally.

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 6: Simplificar domain use cases

**Files:**
- Modify: `internal/domain/character_sheet/create_character_sheet.go`
- Modify: `internal/domain/character_sheet/get_character_sheet.go`
- Modify: `internal/domain/character_sheet/list_character_sheets.go`
- Modify: `internal/domain/enrollment/enroll_character_sheet.go`

- [ ] **Step 1: Substituir `create_character_sheet.go`**

Remove `CharacterSheetToModel`, remove import `pg/model`, chama repo com entidade diretamente.

```go
// internal/domain/character_sheet/create_character_sheet.go
package charactersheet

import (
	"context"
	"sync"

	domainCampaign "github.com/422UR4H/HxH_RPG_System/internal/domain/campaign"
	cc "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_class"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	pgCampaign "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/campaign"
	"github.com/google/uuid"
)

type ICreateCharacterSheet interface {
	CreateCharacterSheet(
		ctx context.Context, input *CreateCharacterSheetInput,
	) (*sheet.CharacterSheet, error)
}

type CreateCharacterSheetUC struct {
	characterClasses *sync.Map
	characterSheets  *sync.Map
	factory          *sheet.CharacterSheetFactory
	repo             IRepository
	campaignRepo     domainCampaign.IRepository
}

func NewCreateCharacterSheetUC(
	charClasses *sync.Map,
	charSheets *sync.Map,
	factory *sheet.CharacterSheetFactory,
	repo IRepository,
	campaignRepo domainCampaign.IRepository,
) *CreateCharacterSheetUC {
	return &CreateCharacterSheetUC{
		characterClasses: charClasses,
		characterSheets:  charSheets,
		factory:          factory,
		repo:             repo,
		campaignRepo:     campaignRepo,
	}
}

type DistributionInput struct {
}

type CreateCharacterSheetInput struct {
	PlayerUUID        *uuid.UUID
	MasterUUID        *uuid.UUID
	CampaignUUID      *uuid.UUID
	Profile           sheet.CharacterProfile
	CharacterClass    enum.CharacterClassName
	CategorySet       sheet.TalentByCategorySet
	SkillsExps        map[enum.SkillName]int
	ProficienciesExps map[enum.WeaponName]int
}

func (uc *CreateCharacterSheetUC) CreateCharacterSheet(
	ctx context.Context, input *CreateCharacterSheetInput,
) (*sheet.CharacterSheet, error) {

	class, exists := uc.characterClasses.Load(input.CharacterClass)
	if !exists {
		return nil, NewCharacterClassNotFoundError(input.CharacterClass.String())
	}
	charClass := class.(cc.CharacterClass)

	skillsExps := input.SkillsExps
	if err := charClass.ValidateSkills(skillsExps); err != nil {
		return nil, err
	}
	profExps := input.ProficienciesExps
	if err := charClass.ValidateProficiencies(profExps); err != nil {
		return nil, err
	}
	charClass.ApplySkills(skillsExps)
	charClass.ApplyProficiencies(profExps)

	if err := uc.validateNickName(input.Profile.NickName); err != nil {
		return nil, err
	}

	if input.PlayerUUID != nil {
		characterSheetsCount, err := uc.repo.CountCharactersByPlayerUUID(
			ctx, *input.PlayerUUID,
		)
		if err != nil {
			return nil, err
		}
		if characterSheetsCount >= 20 {
			return nil, ErrMaxCharacterSheetsLimit
		}
	}
	if input.CampaignUUID != nil {
		masterUUID, err := uc.campaignRepo.GetCampaignMasterUUID(ctx, *input.CampaignUUID)
		if err == pgCampaign.ErrCampaignNotFound {
			return nil, domainCampaign.ErrCampaignNotFound
		}
		if err != nil {
			return nil, err
		}
		if *input.MasterUUID != masterUUID {
			return nil, domainCampaign.ErrNotCampaignOwner
		}
	}

	set := input.CategorySet
	characterSheet, err := uc.factory.Build(
		input.PlayerUUID,
		input.MasterUUID,
		input.CampaignUUID,
		input.Profile,
		set.GetInitialHexValue(),
		nil,
		&charClass,
	)
	if err != nil {
		return nil, err
	}
	talentLvl := set.GetTalentLvl()
	characterSheet.InitTalentWithLvl(talentLvl)

	characterSheet.UUID = uuid.New()
	uc.characterSheets.Store(characterSheet.UUID, characterSheet)

	if err = uc.repo.CreateCharacterSheet(ctx, characterSheet); err != nil {
		return nil, err
	}
	return characterSheet, nil
}

func (uc *CreateCharacterSheetUC) validateNickName(nick string) error {
	var allowedNickName = true
	uc.characterClasses.Range(func(_, value any) bool {
		charClass := value.(cc.CharacterClass)
		if charClass.GetNameString() == nick {
			allowedNickName = false
			return false
		}
		return true
	})
	if !allowedNickName {
		return NewNicknameNotAllowedError(nick)
	}
	return nil
}
```

- [ ] **Step 2: Substituir `get_character_sheet.go`**

Remove `Wrap`, `ModelToProfile`, `hydrateCharacterSheet`, `normalizeStatus`. Simplifica `persistNormalizedStatus` para passar `status.IStatusBar` diretamente.

```go
// internal/domain/character_sheet/get_character_sheet.go
package charactersheet

import (
	"context"
	"fmt"
	"sync"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/auth"
	domainCampaign "github.com/422UR4H/HxH_RPG_System/internal/domain/campaign"
	domainSheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	pgCampaign "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/campaign"
	"github.com/google/uuid"
)

type IGetCharacterSheet interface {
	GetCharacterSheet(
		ctx context.Context, charSheetId uuid.UUID, playerId uuid.UUID,
	) (*domainSheet.CharacterSheet, error)
}

type GetCharacterSheetUC struct {
	characterSheets *sync.Map
	factory         *domainSheet.CharacterSheetFactory
	repo            IRepository
	campaignRepo    domainCampaign.IRepository
}

func NewGetCharacterSheetUC(
	charSheets *sync.Map,
	factory *domainSheet.CharacterSheetFactory,
	repo IRepository,
	campaignRepo domainCampaign.IRepository,
) *GetCharacterSheetUC {
	return &GetCharacterSheetUC{
		characterSheets: charSheets,
		factory:         factory,
		repo:            repo,
		campaignRepo:    campaignRepo,
	}
}

func (uc *GetCharacterSheetUC) GetCharacterSheet(
	ctx context.Context, sheetUUID uuid.UUID, userUUID uuid.UUID,
) (*domainSheet.CharacterSheet, error) {

	// TODO: fix, move after auth validations or remove
	// if charSheet, ok := uc.characterSheets.Load(sheetUUID); ok {
	// 	return charSheet.(*sheet.CharacterSheet), nil
	// }

	charSheet, err := uc.repo.GetCharacterSheetByUUID(ctx, sheetUUID.String())
	if err != nil {
		return nil, err
	}
	masterUUID := charSheet.GetMasterUUID()
	playerUUID := charSheet.GetPlayerUUID()

	if masterUUID != nil && *masterUUID == userUUID {
		return uc.checkAndNormalize(ctx, sheetUUID.String(), charSheet)
	}
	if playerUUID != nil && *playerUUID == userUUID {
		return uc.checkAndNormalize(ctx, sheetUUID.String(), charSheet)
	}

	campaignUUID := charSheet.GetCampaignUUID()
	if campaignUUID == nil {
		return nil, auth.ErrInsufficientPermissions
	}

	campaignMasterUUID, err := uc.campaignRepo.GetCampaignMasterUUID(ctx, *campaignUUID)
	if err == pgCampaign.ErrCampaignNotFound {
		return nil, domainCampaign.ErrCampaignNotFound
	}
	if err != nil {
		return nil, err
	}
	if campaignMasterUUID == userUUID {
		return uc.checkAndNormalize(ctx, sheetUUID.String(), charSheet)
	}
	return nil, auth.ErrInsufficientPermissions
}

func (uc *GetCharacterSheetUC) checkAndNormalize(
	ctx context.Context,
	sheetUUID string,
	charSheet *domainSheet.CharacterSheet,
) (*domainSheet.CharacterSheet, error) {
	// Status normalization is handled inside the gateway's wrap function.
	// If correction was applied, persist asynchronously.
	// TODO: expose wasCorrected from gateway or move normalization trigger here
	return charSheet, nil
}

func (uc *GetCharacterSheetUC) persistNormalizedStatus(
	ctx context.Context,
	sheetUUID string,
	charSheet *domainSheet.CharacterSheet,
) {
	allBars := charSheet.GetAllStatusBar()
	if err := uc.repo.UpdateStatusBars(ctx, sheetUUID,
		allBars[enum.Health],
		allBars[enum.Stamina],
		allBars[enum.Aura],
	); err != nil {
		fmt.Printf("TODO(logger): failed to persist normalized status for sheet %s: %v\n", sheetUUID, err)
	}
}
```

> **Nota:** `wasCorrected` era retornado por `Wrap` e trigava `persistNormalizedStatus` de forma assíncrona. Com `Wrap` dentro do gateway, o use case perde essa informação. A tarefa de reexpor esse sinal fica para um refactor futuro (o comportamento de normalização continua ativo dentro do gateway — só o persist assíncrono no use case ficou sem trigger por enquanto). Adicione um TODO se necessário.

- [ ] **Step 3: Substituir `list_character_sheets.go`**

```go
// internal/domain/character_sheet/list_character_sheets.go
package charactersheet

import (
	"context"

	csEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet"
	"github.com/google/uuid"
)

type IListCharacterSheets interface {
	ListCharacterSheets(
		ctx context.Context, playerId uuid.UUID,
	) ([]csEntity.Summary, error)
}

type ListCharacterSheetsUC struct {
	repo IRepository
}

func NewListCharacterSheetsUC(repo IRepository) *ListCharacterSheetsUC {
	return &ListCharacterSheetsUC{repo: repo}
}

func (uc *ListCharacterSheetsUC) ListCharacterSheets(
	ctx context.Context, playerId uuid.UUID,
) ([]csEntity.Summary, error) {
	return uc.repo.ListCharacterSheetsByPlayerUUID(ctx, playerId.String())
}
```

- [ ] **Step 4: Substituir `domain/enrollment/enroll_character_sheet.go`**

Remove import de `pg/sheet`. O gateway agora retorna `charactersheet.ErrCharacterSheetNotFound` diretamente.

```go
// internal/domain/enrollment/enroll_character_sheet.go
package enrollment

import (
	"context"

	charactersheet "github.com/422UR4H/HxH_RPG_System/internal/domain/character_sheet"
	matchDomain "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	matchPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/match"
	"github.com/google/uuid"
)

type IEnrollCharacterInMatch interface {
	Enroll(
		ctx context.Context,
		matchUUID uuid.UUID,
		characterSheetUUID uuid.UUID,
		playerUUID uuid.UUID,
	) error
}

type EnrollCharacterInMatchUC struct {
	repo      IRepository
	matchRepo matchDomain.IRepository
	sheetRepo charactersheet.IRepository
}

func NewEnrollCharacterInMatchUC(
	repo IRepository,
	matchRepo matchDomain.IRepository,
	sheetRepo charactersheet.IRepository,
) *EnrollCharacterInMatchUC {
	return &EnrollCharacterInMatchUC{
		repo:      repo,
		matchRepo: matchRepo,
		sheetRepo: sheetRepo,
	}
}

func (uc *EnrollCharacterInMatchUC) Enroll(
	ctx context.Context,
	matchUUID uuid.UUID,
	sheetUUID uuid.UUID,
	playerUUID uuid.UUID,
) error {
	sheetRelationship, err := uc.sheetRepo.GetCharacterSheetRelationshipUUIDs(
		ctx, sheetUUID,
	)
	if err != nil {
		return err
	}
	// TODO: treat if the request was made by a master too
	if sheetRelationship.PlayerUUID == nil ||
		*sheetRelationship.PlayerUUID != playerUUID {
		return charactersheet.ErrNotCharacterSheetOwner
	}

	alreadyEnrolled, err := uc.repo.ExistsEnrolledCharacterSheet(
		ctx, sheetUUID, matchUUID,
	)
	if err != nil {
		return err
	}
	if alreadyEnrolled {
		return ErrCharacterAlreadyEnrolled
	}

	match, err := uc.matchRepo.GetMatch(ctx, matchUUID)
	if err == matchPg.ErrMatchNotFound {
		return matchDomain.ErrMatchNotFound
	}
	if err != nil {
		return err
	}
	if match.GameStartAt != nil {
		return ErrMatchAlreadyStarted
	}
	if match.StoryEndAt != nil {
		return ErrMatchAlreadyFinished
	}
	if sheetRelationship.CampaignUUID == nil ||
		*sheetRelationship.CampaignUUID != match.CampaignUUID {
		return ErrCharacterNotInCampaign
	}
	return uc.repo.EnrollCharacterSheet(ctx, matchUUID, sheetUUID)
}
```

- [ ] **Step 5: Verificar compilação do domain**

```bash
go build ./internal/domain/...
```

Expected: PASS. Se houver erros residuais em arquivos de teste, são esperados — resolver na Task 9.

- [ ] **Step 6: Commit**

```bash
git add internal/domain/character_sheet/create_character_sheet.go \
        internal/domain/character_sheet/get_character_sheet.go \
        internal/domain/character_sheet/list_character_sheets.go \
        internal/domain/enrollment/enroll_character_sheet.go
git commit -m "refactor(domain): simplify use cases — remove pg/model imports

CharacterSheetToModel, Wrap, ModelToProfile moved to gateway.
Use cases now pass/receive domain entity types through IRepository.

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 7: Atualizar entidades `enrollment` e `campaign`

**Files:**
- Modify: `internal/domain/entity/enrollment/enrollment.go`
- Modify: `internal/domain/entity/campaign/campaign.go`

- [ ] **Step 1: Atualizar `enrollment.go`**

```go
// internal/domain/entity/enrollment/enrollment.go
package enrollment

import (
	"time"

	csEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet"
	"github.com/google/uuid"
)

type PlayerRef struct {
	UUID uuid.UUID
	Nick string
}

type Enrollment struct {
	UUID           uuid.UUID
	Status         string
	CreatedAt      time.Time
	CharacterSheet csEntity.Summary
	Player         PlayerRef
}
```

- [ ] **Step 2: Atualizar `campaign.go`**

```go
// internal/domain/entity/campaign/campaign.go
package campaign

import (
	"time"

	csEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/match"
	"github.com/google/uuid"
)

type Campaign struct {
	UUID                    uuid.UUID
	MasterUUID              uuid.UUID
	ScenarioUUID            *uuid.UUID
	Name                    string
	BriefInitialDescription string
	BriefFinalDescription   *string
	Description             string
	IsPublic                bool
	CallLink                string
	StoryStartAt            time.Time
	StoryCurrentAt          *time.Time
	StoryEndAt              *time.Time
	CharacterSheets         []csEntity.Summary
	PendingSheets           []csEntity.Summary
	Matches                 []match.Summary
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

func NewCampaign(
	masterUUID uuid.UUID,
	scenarioUUID *uuid.UUID,
	name string,
	briefInitialDescription string,
	description string,
	isPublic bool,
	callLink string,
	storyStartAt time.Time,
	storyCurrentAt *time.Time,
) (*Campaign, error) {
	now := time.Now()
	return &Campaign{
		UUID:                    uuid.New(),
		MasterUUID:              masterUUID,
		ScenarioUUID:            scenarioUUID,
		Name:                    name,
		BriefInitialDescription: briefInitialDescription,
		Description:             description,
		IsPublic:                isPublic,
		CallLink:                callLink,
		StoryStartAt:            storyStartAt,
		StoryCurrentAt:          storyCurrentAt,
		CreatedAt:               now,
		UpdatedAt:               now,
	}, nil
}
```

- [ ] **Step 3: Compilar**

```bash
go build ./internal/domain/entity/...
```

Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/domain/entity/enrollment/enrollment.go \
        internal/domain/entity/campaign/campaign.go
git commit -m "refactor(entity): replace pg/model imports with csEntity.Summary

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 8: Atualizar gateway `campaign` e `enrollment`

**Files:**
- Modify: `internal/gateway/pg/campaign/read_campaign.go`
- Modify: `internal/gateway/pg/enrollment/list_by_match_uuid.go`

- [ ] **Step 1: Atualizar `read_campaign.go` — remover import de `pg/model`**

Substituir as variáveis de scan de `model.CharacterSheetSummary` por `csEntity.Summary`. Apenas a declaração das variáveis e o import mudam — nenhuma lógica de SQL ou scan.

No arquivo `read_campaign.go`, localizar:
```go
import (
    ...
    "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/model"
    ...
)
```
e substituir por:
```go
import (
    ...
    csEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet"
    ...
)
```

Depois, substituir as duas ocorrências de:
```go
var pendingSheets []model.CharacterSheetSummary
for rows.Next() {
    var sheet model.CharacterSheetSummary
```
por:
```go
var pendingSheets []csEntity.Summary
for rows.Next() {
    var sheet csEntity.Summary
```

Fazer o mesmo para `characterSheets`:
```go
var characterSheets []model.CharacterSheetSummary
for rows.Next() {
    var sheet model.CharacterSheetSummary
```
→
```go
var characterSheets []csEntity.Summary
for rows.Next() {
    var sheet csEntity.Summary
```

- [ ] **Step 2: Atualizar `list_by_match_uuid.go` — remover import de `pg/model` implícito**

O arquivo usa `e.CharacterSheet` que agora é `csEntity.Summary` (mudou em Task 7). O scan funciona identicamente — nenhum campo mudou. Verificar apenas que o arquivo compila sem o import de `pg/model` (ele nunca importou diretamente — só importava `entity/enrollment`, que agora usa `csEntity.Summary`).

```bash
go build ./internal/gateway/pg/enrollment/...
```

Expected: PASS sem alteração.

- [ ] **Step 3: Compilar gateway completo**

```bash
go build ./internal/gateway/...
```

Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/gateway/pg/campaign/read_campaign.go
git commit -m "refactor(gateway): campaign repository uses csEntity.Summary for sheet scans

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 9: Atualizar app layer

**Files:**
- Modify: `internal/app/api/sheet/character_sheet_sumary_response.go`

- [ ] **Step 1: Trocar import de `pg/model` por `csEntity`**

Substituir no início do arquivo:
```go
import (
    "time"

    "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/model"
    "github.com/google/uuid"
)
```
por:
```go
import (
    "time"

    csEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet"
    "github.com/google/uuid"
)
```

Substituir todas as ocorrências de `*model.CharacterSheetSummary` por `*csEntity.Summary` nas assinaturas das funções `ToPrivateOnlyResponse`, `ToPrivateSummaryResponse`, `ToPublicSummaryResponse`, `ToBaseSummaryResponse`.

- [ ] **Step 2: Verificar compilação do app**

```bash
go build ./internal/app/...
```

Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add internal/app/api/sheet/character_sheet_sumary_response.go
git commit -m "refactor(app): sheet summary response uses csEntity.Summary

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 10: Atualizar testes unitários

**Files:**
- Modify: `internal/domain/character_sheet/list_character_sheets_test.go`
- Modify: `internal/domain/character_sheet/create_character_sheet_test.go`
- Modify: `internal/domain/character_sheet/get_character_sheet_test.go`
- Modify: `internal/domain/character_sheet/testutil_test.go`
- Modify: `internal/app/api/sheet/mocks_test.go`
- Modify: `internal/app/api/sheet/list_character_sheets_test.go`
- Modify: `internal/app/api/match/list_match_enrollments_test.go`
- Modify: `internal/domain/enrollment/enrollment_test.go`

- [ ] **Step 1: Atualizar `list_character_sheets_test.go`**

Substituir `model.CharacterSheetSummary` por `csEntity.Summary` e trocar o import:

```go
// internal/domain/character_sheet/list_character_sheets_test.go
package charactersheet_test

import (
	"context"
	"errors"
	"testing"

	charactersheet "github.com/422UR4H/HxH_RPG_System/internal/domain/character_sheet"
	csEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/testutil"
	"github.com/google/uuid"
)

func TestListCharacterSheets(t *testing.T) {
	ctx := context.Background()

	t.Run("happy path - returns list", func(t *testing.T) {
		expected := []csEntity.Summary{
			{UUID: uuid.New(), NickName: "Gon"},
			{UUID: uuid.New(), NickName: "Killua"},
		}
		mockRepo := &testutil.MockCharacterSheetRepo{
			ListCharacterSheetsByPlayerUUIDFn: func(ctx context.Context, playerUUID string) ([]csEntity.Summary, error) {
				return expected, nil
			},
		}

		uc := charactersheet.NewListCharacterSheetsUC(mockRepo)
		playerUUID := uuid.New()

		result, err := uc.ListCharacterSheets(ctx, playerUUID)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if len(result) != 2 {
			t.Errorf("expected 2 results, got %d", len(result))
		}
		if result[0].NickName != "Gon" {
			t.Errorf("expected first NickName 'Gon', got %q", result[0].NickName)
		}
	})

	t.Run("happy path - empty list", func(t *testing.T) {
		mockRepo := &testutil.MockCharacterSheetRepo{
			ListCharacterSheetsByPlayerUUIDFn: func(ctx context.Context, playerUUID string) ([]csEntity.Summary, error) {
				return []csEntity.Summary{}, nil
			},
		}

		uc := charactersheet.NewListCharacterSheetsUC(mockRepo)
		result, err := uc.ListCharacterSheets(ctx, uuid.New())
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if len(result) != 0 {
			t.Errorf("expected empty list, got %d items", len(result))
		}
	})

	t.Run("error - repo error", func(t *testing.T) {
		repoErr := errors.New("database error")
		mockRepo := &testutil.MockCharacterSheetRepo{
			ListCharacterSheetsByPlayerUUIDFn: func(ctx context.Context, playerUUID string) ([]csEntity.Summary, error) {
				return nil, repoErr
			},
		}

		uc := charactersheet.NewListCharacterSheetsUC(mockRepo)
		_, err := uc.ListCharacterSheets(ctx, uuid.New())
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, repoErr) {
			t.Errorf("expected repo error, got: %v", err)
		}
	})
}
```

- [ ] **Step 2: Atualizar demais testes unitários — remover imports `pg/model`**

Para cada arquivo listado abaixo, substituir `"github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/model"` por `csEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet"` (quando o import for somente para `CharacterSheetSummary`) e atualizar os tipos nos mocks e asserções conforme necessário.

Executar depois de cada arquivo:
```bash
go test ./internal/domain/character_sheet/... 2>&1 | head -20
go test ./internal/app/api/sheet/... 2>&1 | head -20
go test ./internal/app/api/match/... 2>&1 | head -20
go test ./internal/domain/enrollment/... 2>&1 | head -20
```

- [ ] **Step 3: Rodar todos os testes unitários**

```bash
go test ./internal/...
```

Expected: PASS em todos os testes unitários.

- [ ] **Step 4: Commit**

```bash
git add internal/domain/character_sheet/list_character_sheets_test.go \
        internal/domain/character_sheet/create_character_sheet_test.go \
        internal/domain/character_sheet/get_character_sheet_test.go \
        internal/domain/character_sheet/testutil_test.go \
        internal/app/api/sheet/mocks_test.go \
        internal/app/api/sheet/list_character_sheets_test.go \
        internal/app/api/match/list_match_enrollments_test.go \
        internal/domain/enrollment/enrollment_test.go
git commit -m "test: update unit tests to use csEntity types after layer isolation

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 11: Atualizar integration tests do gateway

**Files:**
- Modify: `internal/gateway/pg/sheet/sheet_integration_test.go`

- [ ] **Step 1: Substituir `buildTestSheet` para usar entidade de domínio**

O helper agora retorna `*domainSheet.CharacterSheet` construída via factory.

```go
func buildTestSheet(playerUUID *uuid.UUID) *domainsheet.CharacterSheet {
	factory := domainsheet.NewCharacterSheetFactory()
	profile := domainsheet.CharacterProfile{
		NickName:         "TestChar",
		FullName:         "Test Character",
		Alignment:        "Neutral",
		Description:      "A test character",
		BriefDescription: "Test",
	}
	birthday := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	profile.Birthday = &birthday

	s, err := factory.Build(playerUUID, nil, nil, profile, nil, nil, nil)
	if err != nil {
		panic(fmt.Sprintf("buildTestSheet: %v", err))
	}
	s.UUID = uuid.New()
	return s
}
```

- [ ] **Step 2: Atualizar `TestCreateCharacterSheet`**

```go
func TestCreateCharacterSheet(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := sheet.NewRepository(pool)
	ctx := context.Background()

	playerStr := pgtest.InsertTestUser(t, pool, "player", "player@test.com", "pass123")
	playerUUID := uuid.MustParse(playerStr)

	t.Run("happy path player-owned with proficiencies", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		s := buildTestSheet(&playerUUID)
		if err := repo.CreateCharacterSheet(ctx, s); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		got, err := repo.GetCharacterSheetByUUID(ctx, s.UUID.String())
		if err != nil {
			t.Fatalf("expected sheet to be readable after create, got: %v", err)
		}
		if got.UUID != s.UUID {
			t.Fatalf("expected UUID %s, got %s", s.UUID, got.UUID)
		}
	})

	t.Run("master-owned requires player_uuid nil — not supported by repo", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		// NOTE: CreateCharacterSheet only inserts player_uuid, not master_uuid.
		// The XOR constraint (chk_exclusive_owner) requires exactly one of player_uuid/master_uuid.
		// Passing a master-owned sheet (playerUUID nil) hits the constraint.
		masterUUID := playerUUID
		s := buildTestSheet(nil)
		_ = masterUUID // sheet has playerUUID=nil, master not inserted into SQL → constraint violation
		err := repo.CreateCharacterSheet(ctx, s)
		if err == nil {
			t.Fatal("expected error for master-owned sheet, got nil")
		}
	})
}
```

- [ ] **Step 3: Atualizar `TestGetCharacterSheetByUUID`**

```go
func TestGetCharacterSheetByUUID(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := sheet.NewRepository(pool)
	ctx := context.Background()

	playerStr := pgtest.InsertTestUser(t, pool, "player", "player@test.com", "pass123")
	playerUUID := uuid.MustParse(playerStr)

	created := buildTestSheet(&playerUUID)
	if err := repo.CreateCharacterSheet(ctx, created); err != nil {
		t.Fatalf("setup: failed to create sheet: %v", err)
	}

	t.Run("found", func(t *testing.T) {
		got, err := repo.GetCharacterSheetByUUID(ctx, created.UUID.String())
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if got.UUID != created.UUID {
			t.Fatalf("expected UUID %s, got %s", created.UUID, got.UUID)
		}
		if got.GetProfile().NickName != "TestChar" {
			t.Fatalf("expected nickname %q, got %q", "TestChar", got.GetProfile().NickName)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := repo.GetCharacterSheetByUUID(ctx, uuid.New().String())
		if !errors.Is(err, charactersheet.ErrCharacterSheetNotFound) {
			t.Fatalf("expected ErrCharacterSheetNotFound, got %v", err)
		}
	})
}
```

- [ ] **Step 4: Atualizar `TestGetCharacterSheetRelationshipUUIDs` — trocar erro esperado**

Localizar a asserção de "not found":
```go
if !errors.Is(err, sheet.ErrCharacterSheetNotFound) {
```
e substituir por:
```go
if !errors.Is(err, charactersheet.ErrCharacterSheetNotFound) {
```

Adicionar import de `charactersheet` e remover import de `pg/model` do arquivo.

- [ ] **Step 5: Rodar integration tests**

```bash
go test -tags=integration ./internal/gateway/pg/sheet/...
```

Expected: PASS em todos os casos.

- [ ] **Step 6: Commit**

```bash
git add internal/gateway/pg/sheet/sheet_integration_test.go
git commit -m "test(gateway): update sheet integration tests for entity-based IRepository

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 12: Cleanup — remover tipos obsoletos de `pg/model`

**Files:**
- Delete: `internal/gateway/pg/model/character_sheet_summary.go`
- Delete: `internal/gateway/pg/model/status_bar.go`
- Delete: `internal/gateway/pg/model/character_sheet_relationship_uuids.go`

- [ ] **Step 1: Verificar que nenhum arquivo fora do gateway importa `pg/model`**

```bash
grep -rl "gateway/pg/model" internal/ --include="*.go" | grep -v "^internal/gateway/"
```

Expected: nenhuma saída.

- [ ] **Step 2: Verificar que `pg/model` não referencia os tipos removidos internamente**

```bash
grep -rn "CharacterSheetSummary\|character_sheet_relationship_uuids\|status_bar" \
  internal/gateway/pg/model/ --include="*.go"
```

Expected: nenhuma saída (os arquivos a deletar não são referenciados por outros arquivos do pacote).

- [ ] **Step 3: Deletar os arquivos**

```bash
rm internal/gateway/pg/model/character_sheet_summary.go \
   internal/gateway/pg/model/status_bar.go \
   internal/gateway/pg/model/character_sheet_relationship_uuids.go
```

- [ ] **Step 4: Compilação e testes finais**

```bash
go build ./...
go test ./internal/...
go vet -tags=integration ./internal/gateway/pg/...
```

Expected: PASS em tudo.

- [ ] **Step 5: Commit final**

```bash
git add -u internal/gateway/pg/model/
git commit -m "refactor(gateway): remove CharacterSheetSummary, StatusBar and RelationshipUUIDs from pg/model

These types now live in entity/character_sheet/. No code outside
gateway/ imports pg/model anymore — layer isolation restored.

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```
