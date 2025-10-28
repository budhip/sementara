package matcher

import (
	"github.com/farhaan/amartha-reconcile-system/internal/domain/transaction"
)

// MatchResult contains the results of a matching operation
type MatchResult struct {
	Matched          []MatchPair
	UnmatchedSystem  []*transaction.Transaction
	UnmatchedBank    []*transaction.Transaction
	AlgorithmUsed    string
	MatchRate        float64
	TotalSystemTxns  int
	TotalBankTxns    int
	TotalMatched     int
	TotalDiscrepancy float64
}

// MatchPair represents a matched pair of transactions
type MatchPair struct {
	SystemTransaction *transaction.Transaction
	BankTransaction   *transaction.Transaction
	ConfidenceScore   float64 // 0-100, 100 = exact match
	AmountDiscrepancy float64
}

// MatcherConfig configures the matching behavior
type MatcherConfig struct {
	// AmountTolerancePct is the percentage tolerance for amount matching (for fuzzy matchers)
	AmountTolerancePct float64
}

// DefaultConfig returns the default matcher configuration
func DefaultConfig() MatcherConfig {
	return MatcherConfig{
		AmountTolerancePct: 0.0, // Exact match
	}
}

// TransactionMatcher is the interface that all matching strategies must implement
// This enables pluggable matching algorithms (Strategy Pattern)
type TransactionMatcher interface {
	// Match takes system and bank transactions and returns matched pairs and unmatched transactions
	Match(systemTxns, bankTxns []*transaction.Transaction) (*MatchResult, error)

	// Name returns the name of the matching algorithm
	Name() string

	// SetConfig updates the matcher configuration
	SetConfig(config MatcherConfig)
}

// CalculateMatchRate computes the match rate as a percentage
func CalculateMatchRate(totalMatched, totalSystem, totalBank int) float64 {
	if totalSystem == 0 && totalBank == 0 {
		return 100.0
	}
	total := totalSystem + totalBank
	matched := totalMatched * 2 // Each match accounts for one system and one bank txn
	return (float64(matched) / float64(total)) * 100.0
}

// NewMatchResult creates a new match result
func NewMatchResult(algorithmName string) *MatchResult {
	return &MatchResult{
		Matched:         make([]MatchPair, 0),
		UnmatchedSystem: make([]*transaction.Transaction, 0),
		UnmatchedBank:   make([]*transaction.Transaction, 0),
		AlgorithmUsed:   algorithmName,
	}
}

// Finalize finalizes the match result by calculating statistics
func (mr *MatchResult) Finalize() {
	mr.TotalSystemTxns = len(mr.Matched) + len(mr.UnmatchedSystem)
	mr.TotalBankTxns = len(mr.Matched) + len(mr.UnmatchedBank)
	mr.TotalMatched = len(mr.Matched)
	mr.MatchRate = CalculateMatchRate(mr.TotalMatched, mr.TotalSystemTxns, mr.TotalBankTxns)

	// Calculate total discrepancy: matched pair differences + all unmatched amounts
	mr.TotalDiscrepancy = 0

	// Add amount differences from matched pairs
	for _, pair := range mr.Matched {
		mr.TotalDiscrepancy += pair.AmountDiscrepancy
	}

	// Add all unmatched system transaction amounts
	for _, txn := range mr.UnmatchedSystem {
		mr.TotalDiscrepancy += txn.AbsAmount()
	}

	// Add all unmatched bank transaction amounts
	for _, txn := range mr.UnmatchedBank {
		mr.TotalDiscrepancy += txn.AbsAmount()
	}
}
