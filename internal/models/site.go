package models

type SiteInfo struct {
	Name         string   `json:"name"`
	Tagline      string   `json:"tagline"`
	Description  string   `json:"description"`
	Logo         string   `json:"logo"`
	Address      string   `json:"address"`
	Phone        string   `json:"phone"`
	Email        string   `json:"email"`
	Whatsapp     string   `json:"whatsapp"`
	OpeningHours string   `json:"opening_hours"`
	WorkingDays  []string `json:"working_days"`
	MapLink      string   `json:"map_link"`
	Socials      Socials  `json:"socials"`
}

type Socials struct {
	Facebook  string `json:"facebook"`
	Instagram string `json:"instagram"`
	Twitter   string `json:"twitter"`
	Linkedin  string `json:"linkedin"`
}
