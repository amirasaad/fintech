package transaction

import (
	"context"
	"testing"
)

func TestRepository_Create(t *testing.T) {
	ctx := context.Background()
	_ = ctx
	t.Skip("TDD: implement test for Create with dto.TransactionCreate")
}

func TestRepository_Update(t *testing.T) {
	ctx := context.Background()
	_ = ctx
	t.Skip("TDD: implement test for Update with dto.TransactionUpdate")
}

func TestRepository_PartialUpdate(t *testing.T) {
	ctx := context.Background()
	_ = ctx
	t.Skip("TDD: implement test for PartialUpdate with dto.TransactionUpdate")
}

func TestRepository_UpsertByPaymentID(t *testing.T) {
	ctx := context.Background()
	_ = ctx
	t.Skip("TDD: implement test for UpsertByPaymentID with dto.TransactionCreate")
}

func TestRepository_Get(t *testing.T) {
	ctx := context.Background()
	_ = ctx
	t.Skip("TDD: implement test for Get returning dto.TransactionRead")
}

func TestRepository_GetByPaymentID(t *testing.T) {
	ctx := context.Background()
	_ = ctx
	t.Skip("TDD: implement test for GetByPaymentID returning dto.TransactionRead")
}

func TestRepository_ListByUser(t *testing.T) {
	ctx := context.Background()
	_ = ctx
	t.Skip("TDD: implement test for ListByUser returning []*dto.TransactionRead")
}

func TestRepository_ListByAccount(t *testing.T) {
	ctx := context.Background()
	_ = ctx
	t.Skip("TDD: implement test for ListByAccount returning []*dto.TransactionRead")
}
