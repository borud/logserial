// Package model contains the model types.
package model

// Message represents a log message.
type Message struct {
	TS     uint64 `json:"ts" db:"ts"`
	Device string `json:"device" db:"device"`
	Msg    string `json:"msg" db:"msg"`
}
