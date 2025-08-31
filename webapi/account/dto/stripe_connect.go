package dto

// InitiateOnboardingResponse represents the response from initiating Stripe Connect onboarding
type InitiateOnboardingResponse struct {
	// OnboardingURL is the URL to redirect the user to for Stripe Connect onboarding
	OnboardingURL string `json:"onboarding_url"`
}

// OnboardingStatusResponse represents the response for checking onboarding status
type OnboardingStatusResponse struct {
	// IsComplete indicates whether the user has completed Stripe Connect onboarding
	IsComplete bool `json:"is_complete"`
}

// ErrorResponse represents a generic error response
type ErrorResponse struct {
	// Error is the error message
	Error string `json:"error"`
}
