package firelfyiii

import (
	"fmt"
	"io"
	"math"
	"time"

	"github.com/sudores/firefly-iii-bank-sync/bank/dto"
	"github.com/rmg/iso4217"
)

const bpfsTag = "bpfs"

// transaction represents fireflyiii transaction
type transaction struct {
	ErrorIfDuplicateHash bool                    `json:"error_if_duplicate_hash,omitempty"`
	ApplyRules           bool                    `json:"apply_rules"`
	FireWebhooks         bool                    `json:"fire_webhooks"`
	GroupTitle           string                  `json:"group_title,omitempty"`
	Transactions         []transactionSplitStore `json:"transactions"`

	jsonDataReader io.Reader
}

type transactionSplitStore struct {
	// Type - set to withdrawal by default
	Type        string    `json:"type"`
	Description string    `json:"description"`
	Date        time.Time `json:"date"`
	Amount      string    `json:"amount"`
	Notes       string    `json:"notes"`

	// Tags - tag bpfs added by default
	Tags              []string `json:"tags"`
	SourceName        string   `json:"source_name"`
	CategoryName      string   `json:"category_name"`
	InternalReference string   `json:"internal_reference"`

	// ExternalID - set with bank api transaction id
	ExternalID   string     `json:"external_id"`
	ExternalURL  string     `json:"external_url"`
	InterestDate *time.Time `json:"interest_date"`
	BookDate     *time.Time `json:"book_date"`
	ProcessDate  *time.Time `json:"process_date"`
	DueDate      *time.Time `json:"due_date"`
	PaymentDate  *time.Time `json:"payment_date"`
	InvoiceDate  *time.Time `json:"invoice_date"`

	// All non necessary fields for request
	Order               int    `json:"order,omitempty"`
	CurrencyID          string `json:"currency_id,omitempty"`
	CurrencyCode        string `json:"currency_code,omitempty"`
	ForeignAmount       string `json:"foreign_amount,omitempty"`
	ForeignCurrencyID   string `json:"foreign_currency_id,omitempty"`
	ForeignCurrencyCode string `json:"foreign_currency_code,omitempty"`
	BudgetID            string `json:"budget_id,omitempty"`
	CategoryID          string `json:"category_id,omitempty"`
	SourceID            string `json:"source_id,omitempty"`
	DestinationID       string `json:"destination_id,omitempty"`
	DestinationName     string `json:"destination_name"`
	Reconciled          bool   `json:"reconciled,omitempty"`
	PiggyBankID         int    `json:"piggy_bank_id,omitempty"`
	PiggyBankName       string `json:"piggy_bank_name,omitempty"`
	BillID              string `json:"bill_id,omitempty"`
	BillName            string `json:"bill_name,omitempty"`
	BunqPaymentID       string `json:"bunq_payment_id,omitempty"`
	SepaCc              string `json:"sepa_cc,omitempty"`
	SepaCtOp            string `json:"sepa_ct_op,omitempty"`
	SepaCtID            string `json:"sepa_ct_id,omitempty"`
	SepaDb              string `json:"sepa_db,omitempty"`
	SepaCountry         string `json:"sepa_country,omitempty"`
	SepaEp              string `json:"sepa_ep,omitempty"`
	SepaCi              string `json:"sepa_ci,omitempty"`
	SepaBatchID         string `json:"sepa_batch_id,omitempty"`
}

func transactionDTOToTransaction(trans *dto.TransactionDTO) *transaction {
	tr := newTransaction()
	if trans.Transaction.Amount < 0 {
		tr.Transactions[0].Type = "withdrawal"
	} else if trans.Transaction.Amount > 0 {
		tr.Transactions[0].Type = "deposit"
	}
	currencyCode, _ := iso4217.ByCode(int(trans.Transaction.CurrencyCode))
	tr.Transactions[0].CurrencyCode = currencyCode
	tr.Transactions[0].Date = trans.Transaction.Time
	tr.Transactions[0].Amount = fmt.Sprint(math.Abs(float64(trans.Transaction.Amount)) / 100)
	tr.Transactions[0].Description = trans.Transaction.Description
	tr.Transactions[0].ExternalID = "AccountId: " + trans.AccountID
	tr.Transactions[0].Tags = append(tr.Transactions[0].Tags, bpfsTag)

	tr.Transactions[0].Notes = fmt.Sprintln(tr.Transactions[0].Notes+"MCC:", trans.Transaction.MCC)
	tr.Transactions[0].Notes = fmt.Sprintln(tr.Transactions[0].Notes+"Comment:", trans.Transaction.Comment)
	tr.Transactions[0].Notes = fmt.Sprintln(tr.Transactions[0].Notes+"Description:", trans.Transaction.Description)
	tr.Transactions[0].Notes = fmt.Sprintln(tr.Transactions[0].Notes+"Counter IBAN:", trans.Transaction.CounterIban)
	tr.Transactions[0].Notes = fmt.Sprintln(tr.Transactions[0].Notes+"Counter name:", trans.Transaction.CounterName)
	tr.Transactions[0].Notes = fmt.Sprintln(tr.Transactions[0].Notes+"Currency code:", currencyCode)
	return tr
}

func newTransaction() *transaction {
	return &transaction{
		ErrorIfDuplicateHash: false,
		ApplyRules:           true,
		FireWebhooks:         true,
		Transactions:         []transactionSplitStore{{}},
	}
}

// Getting account unmarshaling struct
type accounts struct {
	Data []account `json:"data"`
}
type account struct {
	Attributes accountAttrs `json:"attributes"`
}

type accountAttrs struct {
	Name  string `json:"name"`
	ID    string `json:"id"`
	Notes string `json:"notes"`
}
