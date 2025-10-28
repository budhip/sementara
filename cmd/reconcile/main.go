package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/farhaan/amartha-reconcile-system/internal/domain"
	"github.com/farhaan/amartha-reconcile-system/internal/domain/transaction"
	"github.com/farhaan/amartha-reconcile-system/internal/infrastructure/csv"
	"github.com/farhaan/amartha-reconcile-system/pkg/matcher"
)

func main() {
	// CLI flags
	systemFiles := flag.String("system", "", "Comma-separated paths to system transactions CSV file (required)")
	bankFiles := flag.String("banks", "", "Comma-separated paths to bank statement CSV files (required)")
	startDate := flag.String("start", "", "Start date for reconciliation (YYYY-MM-DD, required)")
	endDate := flag.String("end", "", "End date for reconciliation (YYYY-MM-DD, required)")
	flag.Parse()

	// Validate required flags
	if *systemFiles == "" || *bankFiles == "" || *startDate == "" || *endDate == "" {
		fmt.Println("Error: Missing required flags")
		flag.Usage()
		os.Exit(1)
	}

	// Parse dates
	start, err := time.Parse("2006-01-02", *startDate)
	if err != nil {
		fmt.Printf("Error: Invalid start date format: %v\n", err)
		os.Exit(1)
	}

	end, err := time.Parse("2006-01-02", *endDate)
	if err != nil {
		fmt.Printf("Error: Invalid end date format: %v\n", err)
		os.Exit(1)
	}

	// Parse bank files
	bankFilePaths := strings.Split(*bankFiles, ",")
	for i, path := range bankFilePaths {
		bankFilePaths[i] = strings.TrimSpace(path)
	}
	validBankFilePaths, invalidBankFilePaths := testPathValidity(bankFilePaths)
	fmt.Printf("Invalid bank file paths: %+v\n", invalidBankFilePaths)

	systemFilePaths := strings.Split(*systemFiles, ",")
	for i, path := range systemFilePaths {
		systemFilePaths[i] = strings.TrimSpace(path)
	}
	validSystemFilePaths, invalidSystemFilePaths := testPathValidity(systemFilePaths)
	fmt.Printf("Invalid system file paths: %+v\n", invalidSystemFilePaths)

	if len(validBankFilePaths) == 0 {
		fmt.Println("Error: No valid bank statement files provided")
		panic("No valid bank statement files provided")
	}
	if len(validSystemFilePaths) == 0 {
		fmt.Println("Error: No valid system transaction files provided")
		panic("No valid system transaction files provided")
	}

	fmt.Println("---------------------------------------------------------")
	fmt.Println("Amartha Transaction Reconciliation System")

	// Read system transactions
	systemTxns := make([]*transaction.Transaction, 0)
	systemCounts := make(map[string]int)

	for _, systemFile := range validSystemFilePaths {
		txns, err := readSystemTransactions(systemFile, start, end)
		if err != nil {
			fmt.Printf("Error reading %s: %v\n", systemFile, err)
			continue
		}

		// Count by system file (for reporting)
		if len(txns) > 0 {
			systemCounts[systemFile] = len(txns)
			fmt.Printf("%s: %d transactions\n", systemFile, len(txns))
		}
		systemTxns = append(systemTxns, txns...)
	}

	fmt.Printf("Loaded %d system transactions\n\n", len(systemTxns))

	// Read bank statements
	fmt.Println("Reading bank statements...")
	bankTxns := make([]*transaction.Transaction, 0)
	bankCounts := make(map[string]int)

	for _, bankFile := range validBankFilePaths {
		txns, err := readBankStatements(bankFile, start, end)
		if err != nil {
			fmt.Printf("Error reading %s: %v\n", bankFile, err)
			continue
		}

		// Count by bank source
		if len(txns) > 0 {
			source := txns[0].Source
			bankCounts[source] = len(txns)
			fmt.Printf("%s: %d transactions\n", source, len(txns))
		}

		bankTxns = append(bankTxns, txns...)
	}
	fmt.Printf("Total bank transactions: %d\n\n", len(bankTxns))

	m := matcher.NewExactMatcher(matcher.DefaultConfig())

	// Perform reconciliation
	fmt.Println("Reconciling transactions...")
	result, err := m.Match(systemTxns, bankTxns)
	if err != nil {
		fmt.Printf("Error during reconciliation: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Reconciliation complete")
	fmt.Println()

	// Print report
	printReconciliationReport(result, bankCounts, start, end)
}

func testPathValidity(paths []string) (validPaths []string, invalidPaths []string) {

	for _, path := range paths {
		if !isValidPath(path) {
			invalidPaths = append(invalidPaths, path)
			continue
		}

		validPaths = append(validPaths, path)
	}
	return validPaths, invalidPaths
}

func isValidPath(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func readSystemTransactions(filePath string, start, end time.Time) ([]*transaction.Transaction, error) {
	reader, err := csv.NewReader(filePath)
	if err != nil {
		return nil, err
	}

	txns := make([]*transaction.Transaction, 0)
	errorCount := 0

	err = reader.ReadSystemTransactions(func(row *csv.SystemTransactionRow, rowErr error) error {
		if rowErr != nil {
			errorCount++
			return nil // Continue processing
		}

		txn, err := csv.ParseSystemTransaction(row, "cli-job", "system-file")
		if err != nil {
			errorCount++
			return nil // Continue processing
		}

		// Filter by date range
		if txn.TransactionDate.Before(start) || txn.TransactionDate.After(end) {
			return nil // Skip
		}

		txns = append(txns, txn)
		return nil
	})

	if errorCount > 0 {
		fmt.Printf("Skipped %d invalid rows\n", errorCount)
	}

	return txns, err
}

func readBankStatements(filePath string, start, end time.Time) ([]*transaction.Transaction, error) {
	// Extract bank source from filename
	bankSource, err := csv.ExtractBankSourceFromFilename(filePath)
	if err != nil {
		return nil, fmt.Errorf("could not extract bank source from filename: %w", err)
	}

	reader, err := csv.NewReader(filePath)
	if err != nil {
		return nil, err
	}

	txns := make([]*transaction.Transaction, 0)
	errorCount := 0

	err = reader.ReadBankStatements(func(row *csv.BankStatementRow, rowErr error) error {
		if rowErr != nil {
			errorCount++
			return nil // Continue processing
		}

		txn, err := csv.ParseBankTransaction(row, "cli-job", "bank-file", bankSource)
		if err != nil {
			errorCount++
			return nil // Continue processing
		}

		// Filter by date range
		if txn.TransactionDate.Before(start) || txn.TransactionDate.After(end) {
			return nil // Skip
		}

		txns = append(txns, txn)
		return nil
	})

	if errorCount > 0 {
		fmt.Printf("%s: Skipped %d invalid rows\n", bankSource, errorCount)
	}

	return txns, err
}

func printReconciliationReport(result *matcher.MatchResult, bankCounts map[string]int, start, end time.Time) {
	fmt.Println("RECONCILIATION REPORT")

	// Period
	fmt.Printf("Reconciliation Period: %s to %s\n", start.Format("2006-01-02"), end.Format("2006-01-02"))
	fmt.Println()
	// Summary
	fmt.Println("SUMMARY")
	fmt.Println("---------------------------------------------------------")
	fmt.Printf("Total Transactions Processed:   %d\n", result.TotalSystemTxns+result.TotalBankTxns)
	fmt.Printf("System transactions:            %d\n", result.TotalSystemTxns)
	fmt.Printf("Bank transactions:              %d\n", result.TotalBankTxns)
	fmt.Printf("Matched Transactions:           %d (%.1f%%)\n", result.TotalMatched, result.MatchRate)
	fmt.Printf("Unmatched Transactions:         %d\n", len(result.UnmatchedSystem)+len(result.UnmatchedBank))
	fmt.Printf("Unmatched system:               %d\n", len(result.UnmatchedSystem))
	fmt.Printf("Unmatched bank:                 %d\n", len(result.UnmatchedBank))
	fmt.Printf("Total Discrepancy Amount:       %.2f\n", result.TotalDiscrepancy)

	// Matched transactions with discrepancies
	if len(result.Matched) > 0 {
		hasDiscrepancies := false
		for _, match := range result.Matched {
			if match.AmountDiscrepancy > 0.001 {
				hasDiscrepancies = true
				break
			}
		}

		if hasDiscrepancies {
			fmt.Println("MATCHED TRANSACTIONS WITH DISCREPANCIES")
			fmt.Println("---------------------------------------------------------")
			for _, match := range result.Matched {
				if match.AmountDiscrepancy > 0.001 {
					fmt.Printf("System: %s (%.2f) ↔ Bank: %s (%.2f) | Discrepancy: %.2f\n",
						match.SystemTransaction.ID,
						match.SystemTransaction.AbsAmount(),
						match.BankTransaction.ID,
						match.BankTransaction.AbsAmount(),
						match.AmountDiscrepancy)
				}
			}
			fmt.Println()
		}
	}

	fmt.Println()

	// Unmatched system transactions
	if len(result.UnmatchedSystem) > 0 {
		fmt.Println("UNMATCHED SYSTEM TRANSACTIONS")
		fmt.Println("---------------------------------------------------------")
		fmt.Println("Transactions in system but missing in bank statement(s):")
		fmt.Println()

		for _, txn := range result.UnmatchedSystem {
			typeStr := "CREDIT"
			if txn.Type == domain.TransactionTypeDebit {
				typeStr = "DEBIT"
			}
			fmt.Printf("ID: %-15s | Source: %-10s | Type: %-6s | Amount: %10.2f | Date: %s\n",
				txn.ID, txn.Source, typeStr, txn.AbsAmount(), txn.TransactionDate.Format("2006-01-02"))
		}
		fmt.Println()
	}

	// Unmatched bank transactions (grouped by bank)
	if len(result.UnmatchedBank) > 0 {
		fmt.Println("UNMATCHED BANK TRANSACTIONS")
		fmt.Println("---------------------------------------------------------")
		fmt.Println("Transactions in bank statement(s) but missing in system:")
		fmt.Println()

		// Group by bank source
		byBank := make(map[string][]*transaction.Transaction)
		for _, txn := range result.UnmatchedBank {
			byBank[txn.Source] = append(byBank[txn.Source], txn)
		}

		for bank, txns := range byBank {
			fmt.Printf("%s (%d transactions):\n", bank, len(txns))
			for _, txn := range txns {
				typeStr := "CREDIT"
				if txn.Type == domain.TransactionTypeDebit {
					typeStr = "DEBIT"
				}
				fmt.Printf("ID: %-15s | Type: %-6s | Amount: %10.2f | Date: %s\n",
					txn.ID, typeStr, txn.AbsAmount(), txn.TransactionDate.Format("2006-01-02"))
			}
			fmt.Println()
		}
	}
}
