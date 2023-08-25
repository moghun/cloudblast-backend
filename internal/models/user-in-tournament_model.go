package models

// UserInTournament represents a user's participation in a tournament
type UserInTournament struct {
	Username     string    `json:"username"`
	TournamentID string    `json:"tournament_id"`
	GroupID      int       `json:"group_id"`
	Score        int       `json:"score"`
	Rank         int       `json:"rank"`
	Claimed	  	 bool      `json:"claimed"`
}
