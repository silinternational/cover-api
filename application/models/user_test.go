package models

import (
	"testing"
)

func (ms *ModelSuite) TestUser_Validate() {
	t := ms.T()
	tests := []struct {
		name     string
		user     User
		wantErr  bool
		errField string
	}{
		{
			name: "minimum",
			user: User{
				Email:   "user@example.com",
				AppRole: AppRoleUser,
			},
			wantErr: false,
		},
		{
			name: "missing email",
			user: User{
				AppRole: AppRoleUser,
			},
			wantErr:  true,
			errField: "User.Email",
		},
		{
			name: "missing approle",
			user: User{
				Email: "dummy@dusos.com",
			},
			wantErr:  true,
			errField: "User.AppRole",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vErr, _ := tt.user.Validate(DB)
			if tt.wantErr {
				if vErr.Count() == 0 {
					t.Errorf("Expected an error, but did not get one")
				} else if len(vErr.Get(tt.errField)) == 0 {
					t.Errorf("Expected an error on field %v, but got none (errors: %+v)", tt.errField, vErr.Errors)
				}
			} else if vErr.HasAny() {
				t.Errorf("Unexpected error: %+v", vErr)
			}
		})
	}
}

func (ms *ModelSuite) TestUser_CreateInitialPolicy() {
	t := ms.T()

	f := CreateUserFixtures(ms.DB, 2)
	user := f.Users[0]

	tests := []struct {
		name    string
		user    User
		wantErr bool
	}{
		{
			name:    "missing ID",
			user:    User{},
			wantErr: true,
		},
		{
			name:    "good",
			user:    user,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.user.CreateInitialPolicy(DB)
			if tt.wantErr {
				ms.Error(err)
				return
			}

			ms.NoError(err)

			policyUser := PolicyUser{}
			err = ms.DB.Where("user_id = ?", tt.user.ID).First(&policyUser)
			ms.NoError(err, "error trying to find resulting policyUser")
		})
	}
}
