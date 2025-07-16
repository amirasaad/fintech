package handler

import "context"

// Handle calls the payment provider and updates the request with the payment ID
func (h *PaymentProviderHandler) Handle(ctx context.Context, req *OperationRequest) (*OperationResponse, error) {
	if h.provider == nil {
		h.logger.Warn("PaymentProviderHandler: no provider configured, skipping payment initiation")
		return h.BaseHandler.Handle(ctx, req)
	}

	pid, err := h.provider.InitiatePayment(ctx, req.UserID, req.AccountID, req.ConvertedMoney.Amount(), string(req.ConvertedMoney.Currency()))
	if err != nil {
		h.logger.Error("PaymentProviderHandler: payment initiation failed", "error", err)
		return &OperationResponse{Error: err}, nil
	}
	req.PaymentID = pid
	h.logger.Info("PaymentProviderHandler: payment initiated", "payment_id", pid)
	return h.BaseHandler.Handle(ctx, req)
}
