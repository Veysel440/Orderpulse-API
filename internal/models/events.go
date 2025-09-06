package models

import "time"

type OrderEvent struct {
	ID      string    `json:"id"`
	OrderID string    `json:"orderId"`
	Type    string    `json:"type"`
	Status  string    `json:"status"`
	Amount  int       `json:"amount"`
	TS      time.Time `json:"ts"`
}
