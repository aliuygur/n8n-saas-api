package auth

import "encore.dev/beta/auth"

type User struct {
	ID    string
	Email string
}

var testUser = &User{
	ID:    "5008d810-dc31-4874-89b2-4c890a3825fe",
	Email: "alioygur@gmail.com",
}

func GetUser() (*User, bool) {
	// return testUser, true
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
	// return testUser.ID, true
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
