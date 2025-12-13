package main

import (
	"fmt"
	"time"
)

type event struct{
	ID string `json:"id"`
	Name string `json:"name"`
	Date int64 `json:"date"`
}

//helper for date formating
func (e event) Time() string {
	return time.Unix(int64(e.Date), 0).Format("2006-01-02 15:04:05")
}

type user struct{
	ID string `json:"id"`
	Name string `json:"name"`
}


type reservation struct {
	ID 	string `json:"id`
	Event_ID string `json:"event_id"`
	User_ID string `json:"user_id"`
	User_Name string `json:"user_name"`
}

var events = []event{
		{ID: "1", Name: "Conference", Date: 1735689600},
		{ID: "2", Name: "Workshop", Date: 1735776000},
		{ID: "3", Name: "Webinar", Date: 1735862400},
		{ID: "4", Name: "Meetup", Date: 1735948800},
		{ID: "5", Name: "Hackathon", Date: 1736035200},
	}

func 



func main() {
	for _, e := range events {
		fmt.Printf("ID: %s | Name: %s | Date: %s\n",
			e.ID, e.Name, e.Time())
	}
}