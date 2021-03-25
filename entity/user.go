package entity

import "time"

type User struct {
	Id        string        `json:"id"`
	Name      string        `json:"name"`
	Email     string        `json:"email"`
	Password  string        `json:"password,omitempty"`
	LastLogin time.Duration `json:"last_login"`
}
