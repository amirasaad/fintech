package user

// NewUser represents the request body for creating a new user.
type NewUser struct {
	Username string `json:"username" validate:"required,max=50,min=3"`
	Email    string `json:"email" validate:"required,email,max=50"`
	Password string `json:"password" validate:"required,min=6,max=72"`
}

// UpdateUserInput represents the request body for updating user information.
type UpdateUserInput struct {
	Names string `json:"names" validate:"max=100"`
}

// PasswordInput represents the request body for password confirmation operations.
type PasswordInput struct {
	Password string `json:"password" validate:"required"`
}
