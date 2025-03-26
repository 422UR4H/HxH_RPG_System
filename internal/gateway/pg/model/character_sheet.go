package model

import (
	"time"

	"github.com/google/uuid"
)

type CharacterSheet struct {
	ID   int
	UUID uuid.UUID

	Profile            CharacterProfile
	Proficiencies      []Proficiency
	JointProficiencies []JointProficiency

	CurrHexValue *int
	TalentExp    int

	// Physical Attributes
	ResistancePts   int
	StrengthPts     int
	AgilityPts      int
	ActionSpeedPts  int
	FlexibilityPts  int
	DexterityPts    int
	SensePts        int
	ConstitutionPts int

	// Mental Attributes
	ResiliencePts   int
	AdaptabilityPts int
	WeightingPts    int
	CreativityPts   int
	ResilienceExp   int
	AdaptabilityExp int
	WeightingExp    int
	CreativityExp   int

	// Physical Skills
	// Resistance
	VitalityExp int
	EnergyExp   int
	DefenseExp  int
	// Strength
	PushExp          int
	GrabExp          int
	CarryCapacityExp int
	// Agility
	VelocityExp   int
	AccelerateExp int
	BrakeExp      int
	// Action Speed => TODO: change to other name
	AttackSpeedExp int // TODO: change to ActionSpeed
	RepelExp       int
	FeintExp       int
	// Flexibility
	AcrobaticsExp int
	EvasionExp    int
	SneakExp      int
	// Dexterity?
	ReflexExp   int
	AccuracyExp int
	StealthExp  int
	// Sense
	VisionExp  int
	HearingExp int
	SmellExp   int
	TactExp    int
	TasteExp   int
	// Constitution
	HealExp     int
	BreathExp   int
	TenacityExp int
	// SPIRITUALS
	// Spirit
	NenExp       int
	FocusExp     int
	WillPowerExp int

	// jointSkills

	// Nen Principles
	TenExp   int
	ZetsuExp int
	RenExp   int
	GyoExp   int
	KouExp   int
	KenExp   int
	RyuExp   int
	InExp    int
	EnExp    int
	// TODO: create SpiritualAttribute or similar for:
	AuraControlExp int // CoA
	AopExp         int

	// Nen Categories
	ReinforcementExp   int
	TransmutationExp   int
	MaterializationExp int
	SpecializationExp  int
	ManipulationExp    int
	EmissionExp        int

	StaminaCurrPts int
	HealthCurrPts  int
	// AuraCurrPts    int

	CreatedAt time.Time
	UpdatedAt time.Time
}
