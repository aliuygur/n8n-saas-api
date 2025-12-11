package auth

import "encore.dev/beta/auth"

type User struct {
	ID    string
	Email string
}

func GetUser() (*User, bool) {
	user, ok := auth.Data().(*User)
	return user, ok
}

func MustGetUser() *User {
	user, ok := GetUser()
	if !ok {
		panic("user not authenticated")
	}
	return user
}

func GetUserID() (string, bool) {
	uid, ok := auth.UserID()
	if !ok {
		return "", false
	}
	return string(uid), true
}

func MustGetUserID() string {
	uid, ok := GetUserID()
	if !ok {
		panic("user not authenticated")
	}
	return uid
}
