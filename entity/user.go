package entity

import "time"

type User struct {
	Email     string        `json:"email"`
	Name      string        `json:"name"`
	Password  string        `json:"password,omitempty"`
	LastLogin time.Duration `json:"last_login"`
}
