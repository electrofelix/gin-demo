package entity

import "time"

type User struct {
	Id        string    `json:"id"`
	Email     string    `json:"email" binding:"required"`
	Name      string    `json:"name" binding:"required"`
	Password  string    `json:"password,omitempty" binding:"required"`
	LastLogin time.Time `json:"last_login"`
}

type UserLogin struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}
