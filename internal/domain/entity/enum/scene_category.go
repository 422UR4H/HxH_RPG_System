package enum

type SceneCategory string

const (
	Roleplay SceneCategory = "roleplay"
	Battle   SceneCategory = "battle"
)

func (sc SceneCategory) String() string {
	return string(sc)
}

func AllSceneCategories() []SceneCategory {
	return []SceneCategory{Roleplay, Battle}
}

// SceneCategoryFrom validates a wire string against the enum's exact values, same pattern as
// SkillNameFrom/WeaponNameFrom: an exact match against the catalogue, nothing fuzzy. Before
// this existed, change_scene did `enum.SceneCategory(payload.Category)` and stored whatever
// string arrived — a scene could end up with a category that matched neither "battle" nor
// "roleplay", and it would come back out that way in scene_changed and match_full_state.
func SceneCategoryFrom(s string) (SceneCategory, error) {
	for _, c := range AllSceneCategories() {
		if s == c.String() {
			return c, nil
		}
	}
	return "", newInvalidNameOfError("scene category", s)
}
