package sheet

import "time"

type CharacterProfile struct {
	NickName         string
	FullName         string
	Alignment        string
	Description      string
	BriefDescription string
	Birthday         time.Time
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
