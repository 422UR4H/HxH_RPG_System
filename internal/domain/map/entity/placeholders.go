package entity

type Decoration struct {
	ID       string  `json:"id"`
	URL      string  `json:"url"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	Width    float64 `json:"width"`
	Height   float64 `json:"height"`
	Rotation float64 `json:"rotation"`
	ZOrder   int     `json:"z_order"`
	Opacity  float64 `json:"opacity"`
}

type MapItem struct {
	ID        string `json:"id"`
	ItemDefID string `json:"item_def_id"`
}

type BgImage struct {
	URL      string  `json:"url"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	Width    float64 `json:"width"`
	Height   float64 `json:"height"`
	Rotation float64 `json:"rotation"`
	Opacity  float64 `json:"opacity"`
}
