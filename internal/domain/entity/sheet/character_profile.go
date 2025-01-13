package charactersheet

import "time"

type CharacterProfile struct {
	NickName         string
	FullName         string
	Description      string
	BriefDescription string
	Birthday         time.Time
}
