package models

import "time"

type FinanceFilter struct {
	DateRange
	GroupBy string `form:"group_by" json:"group_by"`
}

type FinancePoint struct {
	Period   time.Time `db:"period" json:"period"`
	Revenue  float64   `db:"revenue" json:"revenue"`
	Expenses float64   `db:"expenses" json:"expenses"`
	Profit   float64   `db:"profit" json:"profit"`
}

type FinanceSummary struct {
	Revenue   float64        `json:"revenue"`
	Expenses  float64        `json:"expenses"`
	NetProfit float64        `json:"net_profit"`
	Status    string         `json:"status"`
	Points    []FinancePoint `json:"points"`
}

type RevenueSummary struct {
	Total  float64        `json:"total"`
	Points []FinancePoint `json:"points"`
}

type ExpenseSummary struct {
	Total   float64            `json:"total"`
	Points  []FinancePoint     `json:"points"`
	Payouts []InstructorPayout `json:"payouts"`
}
