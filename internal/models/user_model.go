package models

type User struct {
	Username string `json:"username"`
	Level   int    `json:"level"`
	Coins   int    `json:"coins"`
	Latest_Tournament_ID int `json:"latest_tournament_id"`
}
