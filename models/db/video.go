package dbmodels

type Video struct {
	ID           string `db:"id" json:"id"`
	ChannelID    int    `db:"channel_id" json:"channelId,omitempty"`
	SectionStart int    `db:"section_start" json:"sectionStart"`
	SectionEnd   int    `db:"section_end" json:"sectionEnd"`
	Position     int    `db:"position" json:"position"`
}
