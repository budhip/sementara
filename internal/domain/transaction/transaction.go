package transaction

import (
	"time"

	"github.com/farhaan/amartha-reconcile-system/internal/domain"
)

// Transaction represents a financial transaction entity
type Transaction struct {
	ID              string
	JobID           string
	FileID          string
	SourceType      domain.SourceType
	TransactionDate time.Time
	Amount          float64
	Type            domain.TransactionType
	Source          string // Bank source (e.g., "BCA", "MANDIRI")
	RawData         map[string]any
	NormalizedData  map[string]any
	Matched         bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// NewTransaction creates a new transaction
func NewTransaction(
	jobID, fileID string,
	sourceType domain.SourceType,
	transactionDate time.Time,
	amount float64,
	txnType domain.TransactionType,
	source string,
) *Transaction {
	now := time.Now()
	return &Transaction{
		JobID:           jobID,
		FileID:          fileID,
		SourceType:      sourceType,
		TransactionDate: transactionDate,
		Amount:          amount,
		Type:            txnType,
		Source:          source,
		RawData:         make(map[string]any),
		NormalizedData:  make(map[string]any),
		Matched:         false,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

// IsDebit returns true if transaction is a debit
func (t *Transaction) IsDebit() bool {
	return t.Type == domain.TransactionTypeDebit || t.Amount < 0
}

// AbsAmount returns the absolute value of the amount
func (t *Transaction) AbsAmount() float64 {
	if t.Amount < 0 {
		return -t.Amount
	}
	return t.Amount
}

// NormalizeAmount normalizes the amount based on transaction type
// DEBIT transactions should be negative, CREDIT should be positive
func (t *Transaction) NormalizeAmount() {
	if t.Type == domain.TransactionTypeDebit && t.Amount > 0 {
		t.Amount = -t.Amount
	} else if t.Type == domain.TransactionTypeCredit && t.Amount < 0 {
		t.Amount = -t.Amount
	}
	t.UpdatedAt = time.Now()
}
