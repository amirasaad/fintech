package auth

// LoginInput represents the request body for user authentication.
type LoginInput struct {
	Identity string `json:"identity" validate:"required"`
	Password string `json:"password" validate:"required"`
}
