package model

import (
	"time"

	"github.com/google/uuid"
)

type CharacterSheet struct {
	ID           int
	UUID         uuid.UUID
	PlayerUUID   *uuid.UUID
	MasterUUID   *uuid.UUID
	CampaignUUID *uuid.UUID

	Profile            CharacterProfile
	Proficiencies      []Proficiency
	JointProficiencies []JointProficiency

	CategoryName string
	CurrHexValue *int
	TalentExp    int

	// Levels
	Level         int
	Points        int
	TalentLvl     int
	PhysicalsLvl  int
	MentalsLvl    int
	SpiritualsLvl int
	SkillsLvl     int

	// Status
	Stamina StatusBar
	Health  StatusBar
	Aura    StatusBar

	// Physical Attributes
	ResistancePts   int
	StrengthPts     int
	AgilityPts      int
	CelerityPts     int
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
	PushExp  int
	GrabExp  int
	CarryExp int
	// Agility
	VelocityExp   int
	AccelerateExp int
	BrakeExp      int
	// Celerity
	LegerityExp int
	RepelExp    int
	FeintExp    int
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
	ShuExp   int
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

	CreatedAt time.Time
	UpdatedAt time.Time
}
