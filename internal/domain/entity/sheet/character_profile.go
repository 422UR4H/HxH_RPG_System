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
