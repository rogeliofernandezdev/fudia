package domain

import (
	"regexp"
	"strings"
)

type ExpenseCategoryView struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Active bool   `json:"active"`
}
type ExpenseView struct {
	ID            string  `json:"id"`
	CategoryID    string  `json:"categoryId"`
	CategoryName  string  `json:"categoryName"`
	Description   string  `json:"description"`
	Amount        string  `json:"amount"`
	PaymentMethod string  `json:"paymentMethod"`
	BusinessDate  string  `json:"businessDate"`
	Reference     string  `json:"reference"`
	Notes         string  `json:"notes"`
	Status        string  `json:"status"`
	CreatedByName string  `json:"createdByName"`
	CreatedAt     string  `json:"createdAt"`
	VoidedByName  string  `json:"voidedByName"`
	VoidedAt      *string `json:"voidedAt"`
	VoidReason    string  `json:"voidReason"`
}
type ExpenseInput struct {
	CategoryID    string `json:"categoryId"`
	Description   string `json:"description"`
	Amount        string `json:"amount"`
	PaymentMethod string `json:"paymentMethod"`
	BusinessDate  string `json:"businessDate"`
	Reference     string `json:"reference"`
	Notes         string `json:"notes"`
}

var ExpenseAmountPattern = regexp.MustCompile("^[0-9]{1,12}([.][0-9]{1,2})?$")

func ValidExpenseAmount(value string) bool {
	if !ExpenseAmountPattern.MatchString(value) {
		return false
	}
	plain := strings.ReplaceAll(value, ".", "")
	return strings.Trim(plain, "0") != ""
}

type ExpenseCategoryInput struct {
	Name string `json:"name"`
}
