package currency

import (
	"time"

	"github.com/amirasaad/fintech/pkg/money"
	"github.com/amirasaad/fintech/pkg/registry"
	currencysvc "github.com/amirasaad/fintech/pkg/service/currency"
)

// RegisterRequest represents the request body for registering a currency.
type RegisterRequest struct {
	Code     string            `json:"code" validate:"required,len=3,uppercase"`
	Name     string            `json:"name" validate:"required"`
	Symbol   string            `json:"symbol" validate:"required"`
	Decimals int               `json:"decimals" validate:"required,min=0,max=8"`
	Country  string            `json:"country,omitempty"`
	Region   string            `json:"region,omitempty"`
	Active   bool              `json:"active"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// CurrencyResponse represents the response structure for currency data
type CurrencyResponse struct {
	Code      string            `json:"code"`
	Name      string            `json:"name"`
	Symbol    string            `json:"symbol"`
	Decimals  int               `json:"decimals"`
	Country   string            `json:"country,omitempty"`
	Region    string            `json:"region,omitempty"`
	Active    bool              `json:"active"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt *time.Time        `json:"updated_at,omitempty"`
}

// ToResponse converts a currency entity to a response DTO
func ToResponse(entity *currencysvc.Entity) *CurrencyResponse {
	if entity == nil {
		return nil
	}

	updatedAt := entity.UpdatedAt()

	return &CurrencyResponse{
		Code:      entity.Code.String(),
		Name:      entity.Name,
		Symbol:    entity.Symbol,
		Decimals:  entity.Decimals,
		Country:   entity.Country,
		Region:    entity.Region,
		Active:    entity.Active,
		Metadata:  entity.Metadata(),
		CreatedAt: entity.CreatedAt(),
		UpdatedAt: &updatedAt,
	}
}

// ToServiceEntity converts a RegisterRequest to a service layer entity
func (r *RegisterRequest) ToServiceEntity() *currencysvc.Entity {
	return &currencysvc.Entity{
		Entity:   registry.NewBaseEntity(r.Code, r.Name),
		Code:     money.Code(r.Code),
		Name:     r.Name,
		Symbol:   r.Symbol,
		Decimals: r.Decimals,
		Country:  r.Country,
		Region:   r.Region,
		Active:   r.Active,
	}
}
