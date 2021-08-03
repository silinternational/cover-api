package auth

// User holds common attributes expected from auth providers
type User struct {
	FirstName string
	LastName  string
	Email     string
	UserID    string
	Nickname  string
	PhotoURL  string
}

// Response holds fields for login and logout responses. not all fields will have values
type Response struct {
	RedirectURL string
	AuthUser    *User
	Error       error
}
