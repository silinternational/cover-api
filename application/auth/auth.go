package auth

// User holds common attributes expected from auth providers
type User struct {
	FirstName            string
	LastName             string
	Email                string
	StaffID              string
	AccessToken          string `json:"AccessToken"`
	AccessTokenExpiresAt int64  `json:"AccessTokenExpiresAt"`
	IsNew                bool
}

// Response holds fields for login and logout responses. not all fields will have values
type Response struct {
	RedirectURL string
	AuthUser    *User
	Error       error
}
