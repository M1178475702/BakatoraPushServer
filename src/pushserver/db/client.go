package db

import "github.com/jmoiron/sqlx"

type ClientDb interface {
	FindByClientId(clientId string) (client *Client, err error)
	FindByClientName(clientName string) (client *Client, err error)
}

type ClientDbImpl struct {
	db *sqlx.DB
}

func (c *ClientDbImpl) FindByClientId(clientId string) (client *Client, err error){
	row := c.db.QueryRow("select * from Client where client_id = ?", clientId)
	client = new(Client)
	err = row.Scan(client)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func (c *ClientDbImpl) FindByClientName(clientName string) (client *Client, err error){
	row := c.db.QueryRow("select * from Client where client_name = ?", clientName)
	client = new(Client)
	err = row.Scan(&client.Id, &client.ClientId, &client.ClientName, &client.Password)
	if err != nil {
		return nil, err
	}
	return client, nil
}
