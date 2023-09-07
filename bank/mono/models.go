package mono

import (
	"time"

	"github.com/sudores/firefly-iii-bank-sync/bank/dto"
)

// WebhookStatementItem according to https://api.monobank.ua/docs/#tag/Kliyentski-personalni-dani/paths/~1personal~1webhook/post
type WebhookStatementItem struct {
	TransactionType string `json:"type" validate:"required"`
	Data            struct {
		Account       string        `json:"account" validate:"required"`
		StatementItem StatementItem `json:"statementItem" validate:"required"`
	} `json:"data" validate:"required"`
}

func (w *WebhookStatementItem) ToTransactionDTO() *dto.TransactionDTO {
	trans := &dto.TransactionDTO{
		AccountID: w.Data.Account,
		Transaction: dto.TransactionDTOTransaction{
			ID:           w.Data.StatementItem.ID,
			Amount:       w.Data.StatementItem.Amount,
			Comment:      w.Data.StatementItem.Comment,
			MCC:          w.Data.StatementItem.MCC,
			Description:  w.Data.StatementItem.Description,
			CurrencyCode: w.Data.StatementItem.CurrencyCode,
			CounterIban:  w.Data.StatementItem.CounterIban,
			CounterName:  w.Data.StatementItem.CounterName,
		}}
	trans.Transaction.Time = time.Unix(w.Data.StatementItem.Time, 0)
	return trans
}

// StatementItem according to https://api.monobank.ua/docs/#tag/Kliyentski-personalni-dani/paths/~1personal~1statement~1{account}~1{from}~1{to}/get
type StatementItem struct {
	ID              string `json:"id"`
	Time            int64  `json:"time"`
	Description     string `json:"description"`
	MCC             int32  `json:"mcc"`
	OriginalMCC     int32  `json:"originalMcc"`
	Hold            bool   `json:"hold"`
	Amount          int64  `json:"amount"`
	OperationAmount int64  `json:"operationAmount"`
	CurrencyCode    int32  `json:"currencyCode"`
	CommissionRate  int    `json:"commissionRate"`
	CashbackAmount  int    `json:"cashbackAmount"`
	Balance         int    `json:"balance"`
	Comment         string `json:"comment"`
	ReceiptID       string `json:"receiptId"`
	InvoiceID       string `json:"invoiceId"`
	CounterEdrpou   string `json:"counterEdrpou"`
	CounterIban     string `json:"counterIban"`
	CounterName     string `json:"counterName"`
}
