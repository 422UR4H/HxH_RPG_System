package sheet

import "time"

type CharacterProfile struct {
	NickName         string    `json:"nickname"`
	FullName         string    `json:"fullname"`
	Alignment        string    `json:"alignment"`
	Description      string    `json:"description"`
	BriefDescription string    `json:"brief_description"`
	Birthday         time.Time `json:"birthday"`
}

func (cp *CharacterProfile) Validate() error {
	if len(cp.NickName) < 3 || len(cp.NickName) > 16 {
		return NewInvalidNicknameLengthError(cp.NickName)
	}
	if len(cp.FullName) < 6 || len(cp.FullName) > 32 {
		return NewInvalidFullNameLengthError(cp.FullName)
	}
	if len(cp.Alignment) > 16 {
		return NewInvalidAlignmentLengthError(cp.Alignment)
	}
	if len(cp.BriefDescription) > 32 {
		return NewInvalidBriefDescriptionError(cp.BriefDescription)
	}
	if cp.Birthday.After(time.Now()) {
		return NewInvalidBirthdayError()
	}
	return nil
}

func (cp *CharacterProfile) ToString() string {
	profile := "Nick: " + cp.NickName + "\n"
	profile += "Name: " + cp.FullName + "\n"
	profile += "Alignment: " + cp.Alignment + " | "
	profile += "Birthday: " + cp.Birthday.String() + "\n"

	briefDesc := cp.BriefDescription
	if briefDesc != "" {
		profile += "Brief Description: " + briefDesc + "\n"
	}
	return profile
}
