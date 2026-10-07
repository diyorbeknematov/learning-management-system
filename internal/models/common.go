package models

import "time"

type ListResponse[T any] struct {
	Items []T `json:"items"`
	Total int `json:"total"`
	Page  int `json:"page"`
	Limit int `json:"limit"`
}

type DateRange struct {
	From *time.Time `form:"from" json:"from"`
	To   *time.Time `form:"to" json:"to"`
}
