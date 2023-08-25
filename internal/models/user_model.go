package models

type User struct {
	ID	   				  	string 	`json:"id"`
	Username 				string 	`json:"username"`
	Password 				string 	`json:"password"`
	Country 				string 	`json:"country"`
	Progress_Level   		int    	`json:"progress_level"`
	Coins   				int    	`json:"coins"`
	Latest_Tournament_ID 	string 	`json:"latest_tournament_id"`
}
