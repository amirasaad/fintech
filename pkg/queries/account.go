package queries

type GetAccountQuery struct {
	AccountID string
	UserID    string
}

type GetAccountResult struct {
	AccountID string
	UserID    string
	Balance   float64
	Currency  string
}
