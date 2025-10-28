package matcher

import (
	"testing"
	"time"

	"github.com/farhaan/amartha-reconcile-system/internal/domain"
	"github.com/farhaan/amartha-reconcile-system/internal/domain/transaction"
)

func TestExactMatcher_Name(t *testing.T) {
	matcher := NewExactMatcher(MatcherConfig{})
	if matcher.Name() != "exact" {
		t.Errorf("Expected name 'exact', got %s", matcher.Name())
	}
}

func TestExactMatcher_Match_ExactMatches(t *testing.T) {
	matcher := NewExactMatcher(MatcherConfig{})
	date := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)

	systemTxns := []*transaction.Transaction{
		createSystemTransaction("SYS001", "BCA", 150.50, domain.TransactionTypeDebit, date),
		createSystemTransaction("SYS002", "BCA", 1000.00, domain.TransactionTypeCredit, date),
	}

	bankTxns := []*transaction.Transaction{
		createBankTransaction("BANK001", "BCA", -150.50, domain.TransactionTypeDebit, date),
		createBankTransaction("BANK002", "BCA", 1000.00, domain.TransactionTypeCredit, date),
	}

	result, err := matcher.Match(systemTxns, bankTxns)
	if err != nil {
		t.Fatalf("Match failed: %v", err)
	}

	if len(result.Matched) != 2 {
		t.Errorf("Expected 2 matches, got %d", len(result.Matched))
	}

	if len(result.UnmatchedSystem) != 0 {
		t.Errorf("Expected 0 unmatched system transactions, got %d", len(result.UnmatchedSystem))
	}

	if len(result.UnmatchedBank) != 0 {
		t.Errorf("Expected 0 unmatched bank transactions, got %d", len(result.UnmatchedBank))
	}

	// Verify confidence scores
	for _, match := range result.Matched {
		if match.ConfidenceScore != 100.0 {
			t.Errorf("Expected confidence score 100.0, got %f", match.ConfidenceScore)
		}
		if match.AmountDiscrepancy != 0 {
			t.Errorf("Expected no discrepancy for exact match, got %f", match.AmountDiscrepancy)
		}
	}
}

func TestExactMatcher_Match_DateMismatch(t *testing.T) {
	matcher := NewExactMatcher(MatcherConfig{})
	date1 := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)
	date2 := time.Date(2024, 3, 16, 0, 0, 0, 0, time.UTC)

	systemTxns := []*transaction.Transaction{
		createSystemTransaction("SYS001", "BCA", 150.50, domain.TransactionTypeDebit, date1),
	}

	// Bank transaction on different date
	bankTxns := []*transaction.Transaction{
		createBankTransaction("BANK001", "BCA", -150.50, domain.TransactionTypeDebit, date2),
	}

	result, err := matcher.Match(systemTxns, bankTxns)
	if err != nil {
		t.Fatalf("Match failed: %v", err)
	}

	if len(result.Matched) != 0 {
		t.Errorf("Expected 0 matches (date mismatch), got %d", len(result.Matched))
	}
}

func TestExactMatcher_Match_TypeMismatch(t *testing.T) {
	matcher := NewExactMatcher(MatcherConfig{})
	date := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)

	systemTxns := []*transaction.Transaction{
		createSystemTransaction("SYS001", "BCA", 150.50, domain.TransactionTypeDebit, date),
	}

	// Bank transaction with opposite type (credit instead of debit)
	bankTxns := []*transaction.Transaction{
		createBankTransaction("BANK001", "BCA", 150.50, domain.TransactionTypeCredit, date),
	}

	result, err := matcher.Match(systemTxns, bankTxns)
	if err != nil {
		t.Fatalf("Match failed: %v", err)
	}

	if len(result.Matched) != 0 {
		t.Errorf("Expected 0 matches (type mismatch), got %d", len(result.Matched))
	}
}

func TestExactMatcher_Match_AmountMismatch(t *testing.T) {
	matcher := NewExactMatcher(MatcherConfig{})
	date := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)

	systemTxns := []*transaction.Transaction{
		createSystemTransaction("SYS001", "BCA", 150.50, domain.TransactionTypeDebit, date),
	}

	// Bank transaction with different amount
	bankTxns := []*transaction.Transaction{
		createBankTransaction("BANK001", "BCA", -150.75, domain.TransactionTypeDebit, date),
	}

	result, err := matcher.Match(systemTxns, bankTxns)
	if err != nil {
		t.Fatalf("Match failed: %v", err)
	}

	if len(result.Matched) != 0 {
		t.Errorf("Expected 0 matches (amount mismatch), got %d", len(result.Matched))
	}
}

func TestExactMatcher_Match_MultipleTransactionsSameSource(t *testing.T) {
	matcher := NewExactMatcher(MatcherConfig{})
	date := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)

	systemTxns := []*transaction.Transaction{
		createSystemTransaction("SYS001", "BCA", 150.50, domain.TransactionTypeDebit, date),
		createSystemTransaction("SYS002", "BCA", 1000.00, domain.TransactionTypeCredit, date),
		createSystemTransaction("SYS003", "MANDIRI", 500.00, domain.TransactionTypeDebit, date),
	}

	bankTxns := []*transaction.Transaction{
		createBankTransaction("BANK001", "BCA", -150.50, domain.TransactionTypeDebit, date),
		createBankTransaction("BANK002", "BCA", 1000.00, domain.TransactionTypeCredit, date),
		createBankTransaction("BANK003", "MANDIRI", -500.00, domain.TransactionTypeDebit, date),
	}

	result, err := matcher.Match(systemTxns, bankTxns)
	if err != nil {
		t.Fatalf("Match failed: %v", err)
	}

	if len(result.Matched) != 3 {
		t.Errorf("Expected 3 matches, got %d", len(result.Matched))
	}

	if len(result.UnmatchedSystem) != 0 {
		t.Errorf("Expected 0 unmatched system transactions, got %d", len(result.UnmatchedSystem))
	}

	if len(result.UnmatchedBank) != 0 {
		t.Errorf("Expected 0 unmatched bank transactions, got %d", len(result.UnmatchedBank))
	}
}

func TestExactMatcher_Match_PartialMatches(t *testing.T) {
	matcher := NewExactMatcher(MatcherConfig{})
	date := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)

	systemTxns := []*transaction.Transaction{
		createSystemTransaction("SYS001", "BCA", 150.50, domain.TransactionTypeDebit, date),
		createSystemTransaction("SYS002", "BCA", 1000.00, domain.TransactionTypeCredit, date),
		createSystemTransaction("SYS003", "MANDIRI", 500.00, domain.TransactionTypeDebit, date),
	}

	// Only one matching bank transaction
	bankTxns := []*transaction.Transaction{
		createBankTransaction("BANK001", "BCA", -150.50, domain.TransactionTypeDebit, date),
	}

	result, err := matcher.Match(systemTxns, bankTxns)
	if err != nil {
		t.Fatalf("Match failed: %v", err)
	}

	if len(result.Matched) != 1 {
		t.Errorf("Expected 1 match, got %d", len(result.Matched))
	}

	if len(result.UnmatchedSystem) != 2 {
		t.Errorf("Expected 2 unmatched system transactions, got %d", len(result.UnmatchedSystem))
	}

	if len(result.UnmatchedBank) != 0 {
		t.Errorf("Expected 0 unmatched bank transactions, got %d", len(result.UnmatchedBank))
	}
}

func TestExactMatcher_Match_EmptyInputs(t *testing.T) {
	matcher := NewExactMatcher(MatcherConfig{})

	result, err := matcher.Match([]*transaction.Transaction{}, []*transaction.Transaction{})
	if err != nil {
		t.Fatalf("Match failed: %v", err)
	}

	if len(result.Matched) != 0 {
		t.Errorf("Expected 0 matches, got %d", len(result.Matched))
	}

	if result.MatchRate != 100.0 {
		t.Errorf("Expected 100%% match rate for empty inputs, got %f", result.MatchRate)
	}
}

func TestExactMatcher_Match_Statistics(t *testing.T) {
	matcher := NewExactMatcher(MatcherConfig{})
	date := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)

	systemTxns := []*transaction.Transaction{
		createSystemTransaction("SYS001", "BCA", 150.50, domain.TransactionTypeDebit, date),
		createSystemTransaction("SYS002", "BCA", 1000.00, domain.TransactionTypeCredit, date),
	}

	bankTxns := []*transaction.Transaction{
		createBankTransaction("BANK001", "BCA", -150.50, domain.TransactionTypeDebit, date),
	}

	result, err := matcher.Match(systemTxns, bankTxns)
	if err != nil {
		t.Fatalf("Match failed: %v", err)
	}

	if result.TotalSystemTxns != 2 {
		t.Errorf("Expected 2 total system transactions, got %d", result.TotalSystemTxns)
	}

	if result.TotalBankTxns != 1 {
		t.Errorf("Expected 1 total bank transaction, got %d", result.TotalBankTxns)
	}

	if result.TotalMatched != 1 {
		t.Errorf("Expected 1 total matched, got %d", result.TotalMatched)
	}

	if result.AlgorithmUsed != "exact" {
		t.Errorf("Expected algorithm 'exact', got %s", result.AlgorithmUsed)
	}

}

// Tests for Strict Mode (no source field matching)

func TestExactMatcher_StrictMode_MatchAcrossBanks(t *testing.T) {
	matcher := NewExactMatcher(DefaultConfig())
	date := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)

	// System transaction without relying on source
	systemTxns := []*transaction.Transaction{
		createSystemTransaction("SYS001", "BCA", 150.50, domain.TransactionTypeDebit, date),
	}

	// Bank transaction from different source should still match in strict mode
	bankTxns := []*transaction.Transaction{
		createBankTransaction("BANK001", "MANDIRI", -150.50, domain.TransactionTypeDebit, date),
	}

	result, err := matcher.Match(systemTxns, bankTxns)
	if err != nil {
		t.Fatalf("Match failed: %v", err)
	}

	// In strict mode, source is ignored, so this should match
	if len(result.Matched) != 1 {
		t.Errorf("Expected 1 match in strict mode (ignoring source), got %d", len(result.Matched))
	}

	if len(result.UnmatchedSystem) != 0 {
		t.Errorf("Expected 0 unmatched system transactions, got %d", len(result.UnmatchedSystem))
	}
}

func TestExactMatcher_StrictMode_AmbiguousMatch(t *testing.T) {
	matcher := NewExactMatcher(DefaultConfig())
	date := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)

	// Single system transaction
	systemTxns := []*transaction.Transaction{
		createSystemTransaction("SYS001", "BCA", 150.50, domain.TransactionTypeDebit, date),
	}

	// TWO bank transactions with identical date/type/amount from different banks
	bankTxns := []*transaction.Transaction{
		createBankTransaction("BANK001", "BCA", -150.50, domain.TransactionTypeDebit, date),
		createBankTransaction("BANK002", "MANDIRI", -150.50, domain.TransactionTypeDebit, date),
	}

	result, err := matcher.Match(systemTxns, bankTxns)
	if err != nil {
		t.Fatalf("Match failed: %v", err)
	}

	// Ambiguous - should mark system transaction as unmatched (conservative)
	if len(result.Matched) != 0 {
		t.Errorf("Expected 0 matches (ambiguous), got %d", len(result.Matched))
	}

	if len(result.UnmatchedSystem) != 1 {
		t.Errorf("Expected 1 unmatched system transaction (ambiguous), got %d", len(result.UnmatchedSystem))
	}

	// Both bank transactions should remain unmatched
	if len(result.UnmatchedBank) != 2 {
		t.Errorf("Expected 2 unmatched bank transactions, got %d", len(result.UnmatchedBank))
	}
}

func TestExactMatcher_StrictMode_UnambiguousAfterMatching(t *testing.T) {
	matcher := NewExactMatcher(DefaultConfig())
	date := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)

	// Two system transactions with same amount
	systemTxns := []*transaction.Transaction{
		createSystemTransaction("SYS001", "BCA", 150.50, domain.TransactionTypeDebit, date),
		createSystemTransaction("SYS002", "MANDIRI", 150.50, domain.TransactionTypeDebit, date),
	}

	// Two bank transactions with identical amounts
	bankTxns := []*transaction.Transaction{
		createBankTransaction("BANK001", "BCA", -150.50, domain.TransactionTypeDebit, date),
		createBankTransaction("BANK002", "MANDIRI", -150.50, domain.TransactionTypeDebit, date),
	}

	result, err := matcher.Match(systemTxns, bankTxns)
	if err != nil {
		t.Fatalf("Match failed: %v", err)
	}

	// Both should be marked as unmatched due to ambiguity
	if len(result.Matched) != 0 {
		t.Errorf("Expected 0 matches (both ambiguous), got %d", len(result.Matched))
	}

	if len(result.UnmatchedSystem) != 2 {
		t.Errorf("Expected 2 unmatched system transactions, got %d", len(result.UnmatchedSystem))
	}

	if len(result.UnmatchedBank) != 2 {
		t.Errorf("Expected 2 unmatched bank transactions, got %d", len(result.UnmatchedBank))
	}
}

// Helper functions for creating test transactions

func createSystemTransaction(id, source string, amount float64, txnType domain.TransactionType, date time.Time) *transaction.Transaction {
	txn := transaction.NewTransaction(
		"test-job",
		"test-file",
		domain.SourceTypeSystem,
		date,
		amount,
		txnType,
		source,
	)
	txn.ID = id
	txn.NormalizeAmount()
	return txn
}

func createBankTransaction(id, source string, amount float64, txnType domain.TransactionType, date time.Time) *transaction.Transaction {
	txn := transaction.NewTransaction(
		"test-job",
		"test-file",
		domain.SourceTypeBank,
		date,
		amount,
		txnType,
		source,
	)
	txn.ID = id
	txn.NormalizeAmount()
	return txn
}
