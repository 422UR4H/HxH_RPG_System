package sheet_test

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"sort"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/sheet"
	charactersheet "github.com/422UR4H/HxH_RPG_System/internal/application/character_sheet"
	cc "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_class"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/experience"
	sheetEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/google/uuid"
)

// Paridade com o motor de prévia do front (repo HxH_RPG_Environment_React,
// src/features/sheet/engine). Este teste grava, para cada classe e alguns
// cenários de escolha do jogador, a entrada que o front envia ao criar a ficha
// e a resposta que o back monta a partir dela. O front copia o arquivo e roda o
// próprio motor contra cada caso: se as duas implementações divergirem, o teste
// de lá falha.
//
// Sem -update o teste compara e FALHA se o arquivo estiver desatualizado — é o
// que obriga a regenerar quando uma fórmula da ficha muda. Depois de regenerar,
// sincronize o front (`npm run sync:sheet-fixtures` lá).
//
//	go test ./internal/app/api/sheet/ -run TestSheetParityFixtures -update
var updateParity = flag.Bool("update", false, "rewrite testdata/sheet_parity.json instead of comparing against it")

const parityPath = "testdata/sheet_parity.json"

type parityInput struct {
	SkillsExps        map[string]int `json:"skillsExps"`
	ProficienciesExps map[string]int `json:"proficienciesExps"`
	AttributePoints   map[string]int `json:"attributePoints"`
}

type parityCase struct {
	Name     string         `json:"name"`
	Class    string         `json:"class"`
	Input    parityInput    `json:"input"`
	Expected map[string]any `json:"expected"`
}

type parityFile struct {
	Note      string                                  `json:"note"`
	Constants map[string]float64                      `json:"constants"`
	ExpTables map[string][]int                        `json:"expTables"`
	Classes   map[string]sheet.CharacterClassResponse `json:"classes"`
	Cases     []parityCase                            `json:"cases"`
}

var (
	physPrimaries   = []enum.AttributeName{enum.Resistance, enum.Agility, enum.Flexibility, enum.Sense}
	mentalPrimaries = []enum.AttributeName{enum.Resilience, enum.Adaptability, enum.Judgment, enum.Creativity}
	parityOwner     = uuid.MustParse("00000000-0000-0000-0000-000000000001")
)

func parityConstants() map[string]float64 {
	return map[string]float64{
		"CHARACTER_COEFF":           sheetEntity.CHARACTER_COEFF,
		"TALENT_COEFF":              sheetEntity.TALENT_COEFF,
		"PHYSICAL_COEFF":            sheetEntity.PHYSICAL_COEFF,
		"MENTAL_COEFF":              sheetEntity.MENTAL_COEFF,
		"SPIRITUAL_COEFF":           sheetEntity.SPIRITUAL_COEFF,
		"SKILLS_COEFF":              sheetEntity.SKILLS_COEFF,
		"PHYSICAL_ATTRIBUTE_COEFF":  sheetEntity.PHYSICAL_ATTRIBUTE_COEFF,
		"MENTAL_ATTRIBUTE_COEFF":    sheetEntity.MENTAL_ATTRIBUTE_COEFF,
		"SPIRITUAL_ATTRIBUTE_COEFF": sheetEntity.SPIRITUAL_ATTRIBUTE_COEFF,
		"PHYSICAL_SKILLS_COEFF":     sheetEntity.PHYSICAL_SKILLS_COEFF,
		"MENTAL_SKILLS_COEFF":       sheetEntity.MENTAL_SKILLS_COEFF,
		"SPIRITUAL_SKILLS_COEFF":    sheetEntity.SPIRITUAL_SKILLS_COEFF,
		"SPIRITUAL_PRINCIPLE_COEFF": sheetEntity.SPIRITUAL_PRINCIPLE_COEFF,
	}
}

// parityExpTables guarda a tabela agregada de cada coeficiente usado: se o
// ponto flutuante do JS divergir do Go em algum nível, o front aponta onde.
func parityExpTables(constants map[string]float64) map[string][]int {
	tables := make(map[string][]int)
	for _, coeff := range constants {
		key := json.Number(formatCoeff(coeff)).String()
		if _, ok := tables[key]; ok {
			continue
		}
		table := experience.NewExpTable(coeff)
		agg := make([]int, experience.MAX_LVL)
		for lvl := range agg {
			agg[lvl] = table.GetAggregateExpByLvl(lvl)
		}
		tables[key] = agg
	}
	return tables
}

func formatCoeff(c float64) string {
	b, _ := json.Marshal(c)
	return string(b)
}

func buildParityFile(t *testing.T) parityFile {
	t.Helper()
	factory := sheetEntity.NewCharacterSheetFactory()
	classes := cc.NewCharacterClassFactory().Build()

	names := make([]string, 0, len(classes))
	for name := range classes {
		names = append(names, name.String())
	}
	sort.Strings(names)

	constants := parityConstants()
	file := parityFile{
		Note: "Gerado por internal/app/api/sheet/sheet_parity_test.go (repo HxH_RPG_System). " +
			"Não edite à mão: regenere com -update e sincronize o front.",
		Constants: constants,
		ExpTables: parityExpTables(constants),
		Classes:   make(map[string]sheet.CharacterClassResponse),
	}

	for _, name := range names {
		class := classes[enum.CharacterClassName(name)]
		if len(class.JointSkills) > 0 {
			t.Fatalf("classe %s tem JointSkills: o motor do front ainda não as implementa — "+
				"implemente lá e exponha em ClassDefinitionResponse antes", name)
		}

		classSheet, err := factory.BuildClassSheet(sheetEntity.CharacterProfile{
			NickName:         name,
			Alignment:        class.Profile.Alignment,
			Description:      class.Profile.Description,
			BriefDescription: class.Profile.BriefDescription,
		}, &class)
		if err != nil {
			t.Fatalf("BuildClassSheet(%s): %v", name, err)
		}
		file.Classes[name] = sheet.NewCharacterClassResponse(classSheet, class)

		for _, sc := range parityScenarios(class) {
			file.Cases = append(file.Cases, buildParityCase(t, factory, class, sc))
		}
	}
	return file
}

type parityScenario struct {
	name   string
	profs  map[enum.WeaponName]int
	spread bool // distribui os pontos em rodízio; senão concentra no primeiro primário
	points bool
}

// parityScenarios cobre: só a classe; proficiências escolhidas; pontos
// concentrados (atributo intermediário com média ímpar) e espalhados; e uma
// segunda escolha de armas (outras armas permitidas, na ordem inversa).
func parityScenarios(class cc.CharacterClass) []parityScenario {
	scenarios := []parityScenario{
		{name: "class-only"},
		{name: "points-spread", points: true, spread: true},
	}
	d := class.Distribution
	if d == nil || len(d.ProficiencyPoints) == 0 {
		return append(scenarios, parityScenario{name: "points-concentrated", points: true})
	}

	pick := func(reverse bool) map[enum.WeaponName]int {
		allowed := append([]enum.WeaponName(nil), d.ProficienciesAllowed...)
		if reverse {
			for i, j := 0, len(allowed)-1; i < j; i, j = i+1, j-1 {
				allowed[i], allowed[j] = allowed[j], allowed[i]
			}
		}
		profs := make(map[enum.WeaponName]int)
		for i, exp := range d.ProficiencyPoints {
			if i < len(allowed) {
				profs[allowed[i]] = exp
			}
		}
		return profs
	}
	return append(scenarios,
		parityScenario{name: "profs", profs: pick(false)},
		parityScenario{name: "profs-points-concentrated", profs: pick(false), points: true},
		parityScenario{name: "profs-alt-points-spread", profs: pick(true), points: true, spread: true},
	)
}

func buildParityCase(
	t *testing.T,
	factory *sheetEntity.CharacterSheetFactory,
	class cc.CharacterClass,
	sc parityScenario,
) parityCase {
	t.Helper()
	if len(sc.profs) == 0 {
		// Estado intermediário da prévia: classe escolhida, slots de
		// proficiência ainda vazios. O back recusa esse envio (exige todos os
		// slots), mas o front mostra essa ficha enquanto o jogador escolhe; sem
		// a distribuição, a validação aceita e a montagem é a mesma.
		class.Distribution = nil
	}
	owner := charactersheet.SheetOwner{PlayerUUID: &parityOwner}
	input := &charactersheet.CreateCharacterSheetInput{
		CharacterClass:    class.GetName(),
		SkillsExps:        map[enum.SkillName]int{},
		ProficienciesExps: sc.profs,
		AttributePoints:   map[enum.AttributeName]int{},
	}
	if input.ProficienciesExps == nil {
		input.ProficienciesExps = map[enum.WeaponName]int{}
	}

	if sc.points {
		// o número de pontos é o nível de Físicos/Mentais DEPOIS das escolhas
		probe, err := charactersheet.AssembleCharacterSheet(factory, class, owner, input)
		if err != nil {
			t.Fatalf("%s/%s (sondagem): %v", class.GetNameString(), sc.name, err)
		}
		physLvl, _ := probe.GetLevelOfAbility(enum.Physicals)
		mentLvl, _ := probe.GetLevelOfAbility(enum.Mentals)
		distributePoints(input.AttributePoints, physPrimaries, physLvl, sc.spread)
		distributePoints(input.AttributePoints, mentalPrimaries, mentLvl, sc.spread)
	}

	charSheet, err := charactersheet.AssembleCharacterSheet(factory, class, owner, input)
	if err != nil {
		t.Fatalf("%s/%s: %v", class.GetNameString(), sc.name, err)
	}

	raw, err := json.Marshal(sheet.NewCharacterSheetResponse(charSheet))
	if err != nil {
		t.Fatal(err)
	}
	var expected map[string]any
	if err := json.Unmarshal(raw, &expected); err != nil {
		t.Fatal(err)
	}
	// dono e perfil não são calculados: ficam fora do contrato
	for _, k := range []string{"uuid", "playerUuid", "masterUuid", "campaignUuid", "profile", "submission"} {
		delete(expected, k)
	}

	return parityCase{
		Name:  class.GetNameString() + "/" + sc.name,
		Class: class.GetNameString(),
		Input: parityInput{
			SkillsExps:        stringKeys(input.SkillsExps),
			ProficienciesExps: stringKeys(input.ProficienciesExps),
			AttributePoints:   stringKeys(input.AttributePoints),
		},
		Expected: expected,
	}
}

func distributePoints(dst map[enum.AttributeName]int, primaries []enum.AttributeName, total int, spread bool) {
	for i := 0; i < total; i++ {
		name := primaries[0]
		if spread {
			name = primaries[i%len(primaries)]
		}
		dst[name]++
	}
}

func stringKeys[K ~string](m map[K]int) map[string]int {
	out := make(map[string]int, len(m))
	for k, v := range m {
		out[string(k)] = v
	}
	return out
}

func marshalParity(t *testing.T, file parityFile) []byte {
	t.Helper()
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", " ")
	if err := enc.Encode(file); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestSheetParityFixtures(t *testing.T) {
	got := marshalParity(t, buildParityFile(t))

	// Os mapas do Go não têm ordem, e a montagem percorre vários deles. Gerar de
	// novo e exigir o mesmo resultado prova que a ordem da cascata não muda a
	// ficha — premissa de que o motor do front depende.
	for i := 0; i < 5; i++ {
		if again := marshalParity(t, buildParityFile(t)); !bytes.Equal(got, again) {
			t.Fatal("a montagem da ficha não é determinística: a ordem dos mapas mudou o resultado")
		}
	}

	if *updateParity {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(parityPath, got, 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("reescrito %s — sincronize o front: npm run sync:sheet-fixtures", parityPath)
		return
	}

	want, err := os.ReadFile(parityPath)
	if err != nil {
		t.Fatalf("lendo %s: %v (gere com -update)", parityPath, err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("%s está desatualizado: a montagem da ficha mudou. Regenere com\n"+
			"  go test ./internal/app/api/sheet/ -run TestSheetParityFixtures -update\n"+
			"e sincronize o front (npm run sync:sheet-fixtures), ajustando o motor TS se o teste de lá falhar.",
			parityPath)
	}
}
