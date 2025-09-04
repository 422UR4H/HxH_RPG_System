package sheet

import (
	"strconv"
	"strings"
	"time"
)

type CharacterProfile struct {
	NickName         string     `json:"nickname"`
	FullName         string     `json:"fullname"`
	Alignment        string     `json:"alignment"`
	Description      string     `json:"description"`
	BriefDescription string     `json:"brief_description"`
	Birthday         *time.Time `json:"birthday"`
	Age              int        `json:"age"`
}

func (cp *CharacterProfile) Validate() error {
	if len(cp.NickName) < 3 || len(cp.NickName) > 16 {
		return NewInvalidNicknameLengthError(cp.NickName)
	}
	if len(cp.FullName) < 6 || len(cp.FullName) > 32 {
		return NewInvalidFullNameLengthError(cp.FullName)
	}
	if len(cp.BriefDescription) > 255 {
		return NewInvalidBriefDescriptionError(cp.BriefDescription)
	}
	if cp.Age < 0 {
		return NewInvalidAgeError()
	}
	return cp.ValidateAlignment()
}

func (cp *CharacterProfile) ValidateAlignment() error {
	if cp.Alignment == "" {
		return nil
	}

	alignment := strings.Split(cp.Alignment, "-")
	if len(alignment) != 2 {
		return NewInvalidAlignmentError(cp.Alignment)
	}

	validFirst := map[string]struct{}{
		"Lawful":  {},
		"Neutral": {},
		"Chaotic": {},
	}
	validSecond := map[string]struct{}{
		"Good":    {},
		"Neutral": {},
		"Evil":    {},
	}
	first := alignment[0]
	second := alignment[1]

	if _, ok := validFirst[first]; !ok {
		return NewInvalidAlignmentError(cp.Alignment)
	}
	if _, ok := validSecond[second]; !ok {
		return NewInvalidAlignmentError(cp.Alignment)
	}
	return nil
}

func (cp *CharacterProfile) ToString() string {
	profile := "Nick: " + cp.NickName + "\n"
	profile += "Name: " + cp.FullName + "\n"
	profile += "Alignment: " + cp.Alignment + " | "
	profile += "Birthday: " + cp.Birthday.String() + "\n"
	profile += "Age: " + strconv.Itoa(cp.Age) + "\n"

	briefDesc := cp.BriefDescription
	if briefDesc != "" {
		profile += "Brief Description: " + briefDesc + "\n"
	}
	return profile
}
