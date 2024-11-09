package repo

import (
	"database/sql"

	dbmodels "github.com/ozencb/couchtube/models/db"
)

type ChannelRepository interface {
	FetchAllChannels() ([]dbmodels.Channel, error)
	InsertChannel(tx *sql.Tx, channelName string) (int, error)
	DeleteAllChannels(tx *sql.Tx) error
}

type channelRepository struct {
	db *sql.DB
}

func NewChannelRepository(db *sql.DB) ChannelRepository {
	return &channelRepository{db: db}
}

func (r *channelRepository) BeginTx() (*sql.Tx, error) {
	return r.db.Begin()
}

func (r *channelRepository) FetchAllChannels() ([]dbmodels.Channel, error) {
	query := `
		SELECT c.id, c.name
		FROM channels c
		JOIN videos v ON c.id = v.channel_id
		GROUP BY c.id, c.name;
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var channels []dbmodels.Channel
	for rows.Next() {
		var channel dbmodels.Channel
		if err := rows.Scan(&channel.ID, &channel.Name); err != nil {
			return nil, err
		}
		channels = append(channels, channel)
	}

	return channels, rows.Err()
}

func (r *channelRepository) InsertChannel(tx *sql.Tx, channelName string) (int, error) {
	exec := r.db.Exec
	if tx != nil {
		exec = tx.Exec
	}

	result, err := exec("INSERT INTO channels (name) VALUES (?) RETURNING id", channelName)
	if err != nil {
		return 0, err
	}

	id, err := result.LastInsertId()
	return int(id), err
}

func (r *channelRepository) DeleteAllChannels(tx *sql.Tx) error {
	exec := r.db.Exec
	if tx != nil {
		exec = tx.Exec
	}

	_, err := exec("DELETE FROM channels")
	return err
}
