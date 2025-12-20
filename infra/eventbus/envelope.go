//go:build redis || kafka
// +build redis kafka

package eventbus

import "encoding/json"

type envelope struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}
