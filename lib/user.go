package zbz

// User represents a user in the system with authentication data
type User struct {
	Model
	Name  string `db:"name" json:"name" validate:"required" scope:"read:users:profile,write:users:profile" desc:"The name of the user" ex:"John Doe"`
	Email string `db:"email" json:"email" validate:"required,email" scope:"read:users:profile,write:users:profile" desc:"The email of the user" ex:"john.doe@example.com"`

	// Auth0 integration fields
	Auth0ID string `db:"auth0_id" json:"auth0ID" validate:"required" scope:"read:admin:users,write:admin:users" desc:"Auth0 user identifier"`
}
