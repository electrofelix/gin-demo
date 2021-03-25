package entity

import "time"

type User struct {
	Email     string        `json:"email" binding:"required"`
	Name      string        `json:"name" binding:"required"`
	Password  string        `json:"password,omitempty" binding:"required"`
	LastLogin time.Duration `json:"last_login"`
}
