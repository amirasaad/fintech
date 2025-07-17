package transaction

import (
	"context"
	"testing"
)

//nolint:errcheck
func TestRepository_Create(t *testing.T) {
	ctx := context.Background()
	_ = ctx
	t.Skip("TDD: implement test for Create with dto.TransactionCreate")
}

//nolint:errcheck
func TestRepository_Update(t *testing.T) {
	ctx := context.Background()
	_ = ctx
	t.Skip("TDD: implement test for Update with dto.TransactionUpdate")
}

//nolint:errcheck
func TestRepository_PartialUpdate(t *testing.T) {
	ctx := context.Background()
	_ = ctx
	t.Skip("TDD: implement test for PartialUpdate with dto.TransactionUpdate")
}

//nolint:errcheck
func TestRepository_UpsertByPaymentID(t *testing.T) {
	ctx := context.Background()
	_ = ctx
	t.Skip("TDD: implement test for UpsertByPaymentID with dto.TransactionCreate")
}

//nolint:errcheck
func TestRepository_Get(t *testing.T) {
	ctx := context.Background()
	_ = ctx
	t.Skip("TDD: implement test for Get returning dto.TransactionRead")
}

//nolint:errcheck
func TestRepository_GetByPaymentID(t *testing.T) {
	ctx := context.Background()
	_ = ctx
	t.Skip("TDD: implement test for GetByPaymentID returning dto.TransactionRead")
}

//nolint:errcheck
func TestRepository_ListByUser(t *testing.T) {
	ctx := context.Background()
	_ = ctx
	t.Skip("TDD: implement test for ListByUser returning []*dto.TransactionRead")
}

//nolint:errcheck
func TestRepository_ListByAccount(t *testing.T) {
	ctx := context.Background()
	_ = ctx
	t.Skip("TDD: implement test for ListByAccount returning []*dto.TransactionRead")
}
