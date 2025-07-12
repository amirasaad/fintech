package currency

// RegisterCurrencyRequest represents the request body for registering a currency.
type RegisterCurrencyRequest struct {
	Code     string            `json:"code" validate:"required,len=3,uppercase"`
	Name     string            `json:"name" validate:"required"`
	Symbol   string            `json:"symbol" validate:"required"`
	Decimals int               `json:"decimals" validate:"required,min=0,max=8"`
	Country  string            `json:"country,omitempty"`
	Region   string            `json:"region,omitempty"`
	Active   bool              `json:"active"`
	Metadata map[string]string `json:"metadata,omitempty"`
}
