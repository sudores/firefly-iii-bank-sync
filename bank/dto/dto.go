package dto

import "time"

type TransactionDTO struct {
	AccountID   string                    `json:"account_id"`
	Transaction TransactionDTOTransaction `json:"transaction"`
}

type TransactionDTOTransaction struct {
	ID           string    `json:"id"`
	Amount       int64     `json:"amount"`
	Comment      string    `json:"comment"`
	Time         time.Time `json:"time"`
	MCC          int32     `json:"mcc"`
	Description  string    `json:"description"`
	CurrencyCode int32     `json:"currency_code"`
	CounterIban  string    `json:"counter_iban"`
	CounterName  string    `json:"counter_name"`
}

type ToTransactionDTOer interface {
	ToTransactionDTO() TransactionDTO
}
