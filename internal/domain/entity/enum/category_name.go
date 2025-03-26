package enum

import "fmt"

type CategoryName string

const (
	Reinforcement   CategoryName = "Reinforcement"
	Transmutation   CategoryName = "Transmutation"
	Materialization CategoryName = "Materialization"
	Specialization  CategoryName = "Specialization"
	Manipulation    CategoryName = "Manipulation"
	Emission        CategoryName = "Emission"
)

func (cn CategoryName) String() string {
	return string(cn)
}

func AllNenCategoryNames() []CategoryName {
	return []CategoryName{
		Reinforcement,
		Emission,
		Transmutation,
		Manipulation,
		Materialization,
		Specialization,
	}
}

func CategoryNameFrom(s string) (CategoryName, error) {
	for _, name := range AllNenCategoryNames() {
		if s == name.String() {
			return name, nil
		}
	}
	return "", fmt.Errorf("%w%s: %s", ErrInvalidNameOf, "category", s)
}
