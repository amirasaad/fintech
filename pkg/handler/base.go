package handler

import (
	"context"
)

// BaseHandler provides common functionality for all handlers
type BaseHandler struct {
	next OperationHandler
}

// SetNext sets the next handler in the chain
func (h *BaseHandler) SetNext(handler OperationHandler) {
	h.next = handler
}

// Handle passes the request to the next handler in the chain
func (h *BaseHandler) Handle(ctx context.Context, req *OperationRequest) (*OperationResponse, error) {
	if h.next != nil {
		return h.next.Handle(ctx, req)
	}
	return &OperationResponse{}, nil
}
