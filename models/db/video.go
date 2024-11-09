package dbmodels

type Video struct {
	ID           string `db:"id" json:"id"`
	SectionStart int    `db:"section_start" json:"sectionStart"`
	SectionEnd   int    `db:"section_end" json:"sectionEnd"`
}
