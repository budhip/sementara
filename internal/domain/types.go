package domain

// TransactionType represents the type of transaction
type TransactionType string

const (
	TransactionTypeDebit  TransactionType = "DEBIT"
	TransactionTypeCredit TransactionType = "CREDIT"
)

// SourceType indicates whether transaction is from system or bank
type SourceType string

const (
	SourceTypeSystem SourceType = "SYSTEM"
	SourceTypeBank   SourceType = "BANK"
)
