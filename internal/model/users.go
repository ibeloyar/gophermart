package model

import "time"

type TokenInfo struct {
	ID    int64  `json:"id"`
	Login string `json:"login"`
}

type User struct {
	ID        int64     `json:"id"`
	Login     string    `json:"login"`
	Password  string    `json:"password"`
	CreatedAt time.Time `json:"created_at"`
}

type LoginDTO struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type RegisterDTO struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}
