package matcher

import (
	"math"
	"time"

	"github.com/farhaan/amartha-reconcile-system/internal/domain/transaction"
)

// ExactMatcher matches transactions by date, type, and amount
type ExactMatcher struct {
	config MatcherConfig
}

func NewExactMatcher(config MatcherConfig) TransactionMatcher {
	return &ExactMatcher{
		config: config,
	}
}

func (em *ExactMatcher) SetConfig(config MatcherConfig) {
	em.config = config
}

func (em *ExactMatcher) Name() string {
	return "exact"
}

// Match finds matching transactions between system and bank records.
// Builds a hash map of bank transactions by date_type_amount, then checks each
// system transaction against it. If there's exactly one match, we match them.
// If there are multiple matches (ambiguous), we mark all as unmatched rather than guess.
func (em *ExactMatcher) Match(systemTxns, bankTxns []*transaction.Transaction) (*MatchResult, error) {
	result := NewMatchResult(em.Name())

	bankTxnMap := make(map[string][]*transaction.Transaction)
	for _, bankTxn := range bankTxns {
		key := em.generateKey(bankTxn)
		bankTxnMap[key] = append(bankTxnMap[key], bankTxn)
	}

	matchedBankTxns := make(map[string]bool)

	for _, sysTxn := range systemTxns {
		key := em.generateKey(sysTxn)
		candidates, exists := bankTxnMap[key]

		if !exists || len(candidates) == 0 {
			result.UnmatchedSystem = append(result.UnmatchedSystem, sysTxn)
			continue
		}

		availableCandidates := make([]*transaction.Transaction, 0)
		for _, bankTxn := range candidates {
			if !matchedBankTxns[bankTxn.ID] {
				availableCandidates = append(availableCandidates, bankTxn)
			}
		}

		if len(availableCandidates) > 1 {
			result.UnmatchedSystem = append(result.UnmatchedSystem, sysTxn)
			continue
		}

		matched := false
		for _, bankTxn := range availableCandidates {
			if em.isExactMatch(sysTxn, bankTxn) {
				pair := MatchPair{
					SystemTransaction: sysTxn,
					BankTransaction:   bankTxn,
					ConfidenceScore:   100.0,
					AmountDiscrepancy: em.calculateDiscrepancy(sysTxn, bankTxn),
				}
				result.Matched = append(result.Matched, pair)
				matchedBankTxns[bankTxn.ID] = true
				matched = true
				break
			}
		}

		if !matched {
			result.UnmatchedSystem = append(result.UnmatchedSystem, sysTxn)
		}
	}

	for _, bankTxn := range bankTxns {
		if !matchedBankTxns[bankTxn.ID] {
			result.UnmatchedBank = append(result.UnmatchedBank, bankTxn)
		}
	}

	result.Finalize()
	return result, nil
}

// generateKey creates a key like "2024-03-15_debit_15050" for hashing.
// Uses absolute amount so debits and credits with same value get different keys.
func (em *ExactMatcher) generateKey(txn *transaction.Transaction) string {
	dateStr := txn.TransactionDate.Format("2006-01-02")
	typeStr := "credit"
	if txn.IsDebit() {
		typeStr = "debit"
	}
	amount := txn.AbsAmount()
	return dateStr + "_" + typeStr + "_" + formatAmount(amount)
}

// isExactMatch checks if two transactions are the same (date, type, amount).
func (em *ExactMatcher) isExactMatch(sysTxn, bankTxn *transaction.Transaction) bool {
	if !isSameDay(sysTxn.TransactionDate, bankTxn.TransactionDate) {
		return false
	}
	if sysTxn.IsDebit() != bankTxn.IsDebit() {
		return false
	}
	if !amountsEqual(sysTxn.AbsAmount(), bankTxn.AbsAmount()) {
		return false
	}
	return true
}

// calculateDiscrepancy returns the amount difference. Should always be 0 for exact matches.
func (em *ExactMatcher) calculateDiscrepancy(sysTxn, bankTxn *transaction.Transaction) float64 {
	return math.Abs(sysTxn.AbsAmount() - bankTxn.AbsAmount())
}

// isSameDay checks if two dates are on the same day (ignores time).
func isSameDay(t1, t2 time.Time) bool {
	y1, m1, d1 := t1.Date()
	y2, m2, d2 := t2.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}

// amountsEqual checks if two amounts are equal (within 0.001 for floating point errors).
func amountsEqual(a1, a2 float64) bool {
	const epsilon = 0.001
	return math.Abs(a1-a2) < epsilon
}

// formatAmount converts amount to string for use in keys.
func formatAmount(amount float64) string {
	return string(rune(int(amount * 100)))
}
