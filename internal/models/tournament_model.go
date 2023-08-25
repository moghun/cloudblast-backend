package models

import "time"

type Tournament struct {
	TournamentID 			string    `json:"tournament_id"`
	StartTime    			time.Time `json:"start_time"`
	EndTime      			time.Time `json:"end_time"`
	NumRegisteredUsers    	int       `json:"num_registered_users"`
	Finished				bool	  `json:"finished"`
	LatestGroupID			int 	  `json:"latest_group_id"`
}