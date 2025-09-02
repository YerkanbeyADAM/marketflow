package model

import "time"

type MarketData struct {
	Symbol    string    `json:"symbol"`
	Price     float64   `json:"price"`
	Timestamp time.Time `json:"timestamp"`
	Exchange  string    `json:"exchange"`
}

type AggregatedData struct {
	Symbol    string    `json:"symbol"`
	Exchange  string    `json:"exchange"`
	Timestamp time.Time `json:"timestamp"`
	AvgPrice  float64   `json:"average_price"`
	MinPrice  float64   `json:"min_price"`
	MaxPrice  float64   `json:"max_price"`
}
