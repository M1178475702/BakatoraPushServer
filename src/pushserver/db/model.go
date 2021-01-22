package db

type (
	Client struct {
		Id         int    `db:"id"`
		ClientId   string `db:"client_id"`
		ClientName string `db:"client_name"`
		Password   string `db:"password"`
	}
)
