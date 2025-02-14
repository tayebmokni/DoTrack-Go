package model

import (
    "time"
)

type User struct {
    ID        string    `json:"id"`
    Email     string    `json:"email"`
    Password  string    `json:"-"` // Password is not included in JSON
    Name      string    `json:"name"`
    Admin     bool      `json:"admin"`
    CreatedAt time.Time `json:"createdAt"`
}

func NewUser(email, password, name string) *User {
    return &User{
        ID:        GenerateID(),
        Email:     email,
        Password:  password, // In production, this should be hashed
        Name:      name,
        Admin:     false,
        CreatedAt: time.Now(),
    }
}
