package dbmodels

type Channel struct {
	ID       int    `db:"id" json:"id"`
	Name     string `db:"name" json:"name"`
	Position int    `db:"position" json:"position"`
}
