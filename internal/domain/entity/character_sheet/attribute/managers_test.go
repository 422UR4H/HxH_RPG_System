package attribute_test

import (
	"errors"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/attribute"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

// --- Manager (physical/mental attributes) tests ---

func newTestManager() (*attribute.Manager, map[enum.AttributeName]*int) {
	resBuff := 0
	aglBuff := 0
	constBuff := 0

	buffs := map[enum.AttributeName]*int{
		enum.Resistance:   &resBuff,
		enum.Agility:      &aglBuff,
		enum.Constitution: &constBuff,
	}

	resAttr := newTestPrimaryAttribute(enum.Resistance, buffs[enum.Resistance])
	aglAttr := newTestPrimaryAttribute(enum.Agility, buffs[enum.Agility])

	primAttrs := map[enum.AttributeName]*attribute.PrimaryAttribute{
		enum.Resistance: resAttr,
		enum.Agility:    aglAttr,
	}

	midAttr := newTestMiddleAttribute(enum.Constitution, buffs[enum.Constitution], resAttr, aglAttr)
	midAttrs := map[enum.AttributeName]*attribute.MiddleAttribute{
		enum.Constitution: midAttr,
	}

	mgr := attribute.NewAttributeManager(primAttrs, midAttrs, buffs)
	return mgr, buffs
}

func TestAttributeManager_Get_Primary(t *testing.T) {
	mgr, _ := newTestManager()

	attr, err := mgr.Get(enum.Resistance)
	if err != nil {
		t.Fatalf("Get(Resistance) error: %v", err)
	}
	if attr.GetPoints() != 0 {
		t.Errorf("initial points: got %d, want 0", attr.GetPoints())
	}
}

func TestAttributeManager_Get_Middle(t *testing.T) {
	mgr, _ := newTestManager()

	attr, err := mgr.Get(enum.Constitution)
	if err != nil {
		t.Fatalf("Get(Constitution) error: %v", err)
	}
	if attr.GetPoints() != 0 {
		t.Errorf("initial points: got %d, want 0", attr.GetPoints())
	}
}

func TestAttributeManager_Get_NotFound(t *testing.T) {
	mgr, _ := newTestManager()

	_, err := mgr.Get(enum.Flame)
	if err == nil {
		t.Fatal("expected error for non-existent attribute")
	}
	if !errors.Is(err, attribute.ErrAttributeNotFound) {
		t.Errorf("expected ErrAttributeNotFound, got %v", err)
	}
}

func TestAttributeManager_GetPrimary(t *testing.T) {
	mgr, _ := newTestManager()

	pa, err := mgr.GetPrimary(enum.Resistance)
	if err != nil {
		t.Fatalf("GetPrimary error: %v", err)
	}
	if pa.GetName() != enum.Resistance {
		t.Errorf("name: got %v, want Resistance", pa.GetName())
	}
}

func TestAttributeManager_GetPrimary_NotFound(t *testing.T) {
	mgr, _ := newTestManager()

	_, err := mgr.GetPrimary(enum.Constitution)
	if err == nil {
		t.Fatal("expected error for non-primary attribute")
	}
	if !errors.Is(err, attribute.ErrPrimaryAttributeNotFound) {
		t.Errorf("expected ErrPrimaryAttributeNotFound, got %v", err)
	}
}

func TestAttributeManager_IncreasePointsForPrimary(t *testing.T) {
	mgr, _ := newTestManager()

	result, err := mgr.IncreasePointsForPrimary(enum.Resistance, 3)
	if err != nil {
		t.Fatalf("IncreasePointsForPrimary error: %v", err)
	}
	if result[enum.Resistance] != 3 {
		t.Errorf("Resistance points: got %d, want 3", result[enum.Resistance])
	}
}

func TestAttributeManager_IncreasePointsForPrimary_NotFound(t *testing.T) {
	mgr, _ := newTestManager()

	_, err := mgr.IncreasePointsForPrimary(enum.Flame, 3)
	if err == nil {
		t.Fatal("expected error for non-existent primary attribute")
	}
}

func TestAttributeManager_SetBuff(t *testing.T) {
	mgr, _ := newTestManager()

	buffs, err := mgr.SetBuff(enum.Resistance, 5)
	if err != nil {
		t.Fatalf("SetBuff error: %v", err)
	}
	if *buffs[enum.Resistance] != 5 {
		t.Errorf("buff value: got %d, want 5", *buffs[enum.Resistance])
	}
}

func TestAttributeManager_SetBuff_NotFound(t *testing.T) {
	mgr, _ := newTestManager()

	_, err := mgr.SetBuff(enum.Flame, 5)
	if err == nil {
		t.Fatal("expected error for non-existent attribute")
	}
}

func TestAttributeManager_RemoveBuff(t *testing.T) {
	mgr, _ := newTestManager()

	if _, err := mgr.SetBuff(enum.Resistance, 5); err != nil {
		t.Fatal(err)
	}
	buffs, err := mgr.RemoveBuff(enum.Resistance)
	if err != nil {
		t.Fatalf("RemoveBuff error: %v", err)
	}
	if *buffs[enum.Resistance] != 0 {
		t.Errorf("buff after remove: got %d, want 0", *buffs[enum.Resistance])
	}
}

func TestAttributeManager_GetAllAttributes(t *testing.T) {
	mgr, _ := newTestManager()

	all := mgr.GetAllAttributes()
	// 2 primary + 1 middle = 3
	if len(all) != 3 {
		t.Errorf("GetAllAttributes count: got %d, want 3", len(all))
	}
	for _, name := range []enum.AttributeName{enum.Resistance, enum.Agility, enum.Constitution} {
		if _, ok := all[name]; !ok {
			t.Errorf("missing %v in GetAllAttributes", name)
		}
	}
}

func TestAttributeManager_GetAttributesLevel(t *testing.T) {
	mgr, _ := newTestManager()

	levels := mgr.GetAttributesLevel()
	for name, lvl := range levels {
		if lvl != 0 {
			t.Errorf("initial level of %v: got %d, want 0", name, lvl)
		}
	}
}

func TestAttributeManager_GetAttributesPoints(t *testing.T) {
	mgr, _ := newTestManager()

	points := mgr.GetAttributesPoints()
	for name, pts := range points {
		if pts != 0 {
			t.Errorf("initial points of %v: got %d, want 0", name, pts)
		}
	}
}

// --- SpiritualManager tests ---

func newTestSpiritualManager() (*attribute.SpiritualManager, map[enum.AttributeName]*int) {
	flameBuff := 0
	conscBuff := 0

	buffs := map[enum.AttributeName]*int{
		enum.Flame:      &flameBuff,
		enum.Conscience: &conscBuff,
	}

	flameAttr := newTestSpiritualAttribute(enum.Flame, buffs[enum.Flame])
	conscAttr := newTestSpiritualAttribute(enum.Conscience, buffs[enum.Conscience])

	attrs := map[enum.AttributeName]*attribute.SpiritualAttribute{
		enum.Flame:      flameAttr,
		enum.Conscience: conscAttr,
	}

	mgr := attribute.NewSpiritualAttributeManager(attrs, buffs)
	return mgr, buffs
}

func TestSpiritualManager_Get(t *testing.T) {
	mgr, _ := newTestSpiritualManager()

	attr, err := mgr.Get(enum.Flame)
	if err != nil {
		t.Fatalf("Get(Flame) error: %v", err)
	}
	if attr.GetExpPoints() != 0 {
		t.Errorf("initial exp: got %d, want 0", attr.GetExpPoints())
	}
}

func TestSpiritualManager_Get_NotFound(t *testing.T) {
	mgr, _ := newTestSpiritualManager()

	_, err := mgr.Get(enum.Resistance)
	if err == nil {
		t.Fatal("expected error for non-existent spiritual attribute")
	}
	if !errors.Is(err, attribute.ErrAttributeNotFound) {
		t.Errorf("expected ErrAttributeNotFound, got %v", err)
	}
}

func TestSpiritualManager_SetBuff(t *testing.T) {
	mgr, _ := newTestSpiritualManager()

	buffs, err := mgr.SetBuff(enum.Flame, 7)
	if err != nil {
		t.Fatalf("SetBuff error: %v", err)
	}
	if *buffs[enum.Flame] != 7 {
		t.Errorf("buff value: got %d, want 7", *buffs[enum.Flame])
	}
}

func TestSpiritualManager_RemoveBuff(t *testing.T) {
	mgr, _ := newTestSpiritualManager()

	if _, err := mgr.SetBuff(enum.Flame, 7); err != nil {
		t.Fatal(err)
	}
	buffs, err := mgr.RemoveBuff(enum.Flame)
	if err != nil {
		t.Fatalf("RemoveBuff error: %v", err)
	}
	if *buffs[enum.Flame] != 0 {
		t.Errorf("buff after remove: got %d, want 0", *buffs[enum.Flame])
	}
}

func TestSpiritualManager_GetAllAttributes(t *testing.T) {
	mgr, _ := newTestSpiritualManager()

	all := mgr.GetAllAttributes()
	if len(all) != 2 {
		t.Errorf("GetAllAttributes count: got %d, want 2", len(all))
	}
	for _, name := range []enum.AttributeName{enum.Flame, enum.Conscience} {
		if _, ok := all[name]; !ok {
			t.Errorf("missing %v in GetAllAttributes", name)
		}
	}
}

func TestSpiritualManager_GetAttributesLevel(t *testing.T) {
	mgr, _ := newTestSpiritualManager()

	levels := mgr.GetAttributesLevel()
	for name, lvl := range levels {
		if lvl != 0 {
			t.Errorf("initial level of %v: got %d, want 0", name, lvl)
		}
	}
}
