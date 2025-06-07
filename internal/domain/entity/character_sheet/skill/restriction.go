package skill

type Restriction struct {
	description string
	value       float64
}

func NewRestriction(description string, value float64) *Restriction {
	return &Restriction{description: description, value: value}
}

func (r *Restriction) GetDescription() string {
	return r.description
}

func (r *Restriction) GetValue() float64 {
	return r.value
}
