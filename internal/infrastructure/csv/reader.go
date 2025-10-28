package csv

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/farhaan/amartha-reconcile-system/internal/domain"
	"github.com/farhaan/amartha-reconcile-system/internal/domain/transaction"
)

// SystemTransactionRow represents a row from the system_transactions.csv
type SystemTransactionRow struct {
	TrxID           string
	Amount          string
	Source          string
	Type            string
	TransactionTime string
	RowNumber       int64
}

// BankStatementRow represents a row from a bank statement CSV
type BankStatementRow struct {
	UniqueIdentifier string
	Amount           string
	Date             string
	RowNumber        int64
}

// Reader provides streaming CSV reading capabilities
type Reader struct {
	filePath string
	file     *os.File
	reader   *csv.Reader
	headers  []string
	rowCount int64
}

// NewReader creates a new CSV reader
func NewReader(filePath string) (*Reader, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", filePath, err)
	}

	csvReader := csv.NewReader(file)
	csvReader.TrimLeadingSpace = true

	// Read headers
	headers, err := csvReader.Read()
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to read headers from %s: %w", filePath, err)
	}

	return &Reader{
		filePath: filePath,
		file:     file,
		reader:   csvReader,
		headers:  headers,
		rowCount: 0,
	}, nil
}

// ReadSystemTransactions reads system transactions in streaming fashion.
// Validates headers, parses each row, and invokes callback for processing.
// Errors are passed to callback allowing graceful handling and continuation.
func (r *Reader) ReadSystemTransactions(callback func(*SystemTransactionRow, error) error) error {
	defer r.Close()

	expectedHeaders := []string{"trxID", "amount", "source", "type", "transactionTime"}
	if !r.validateHeaders(expectedHeaders) {
		return fmt.Errorf("invalid headers in system transaction file. Expected: %v, Got: %v",
			expectedHeaders, r.headers)
	}

	for {
		record, err := r.reader.Read()
		if err == io.EOF {
			break
		}

		r.rowCount++

		if err != nil {
			if cbErr := callback(nil, fmt.Errorf("row %d: failed to read: %w", r.rowCount, err)); cbErr != nil {
				return cbErr
			}
			continue
		}

		if len(record) != len(expectedHeaders) {
			if cbErr := callback(nil, fmt.Errorf("row %d: expected %d columns, got %d",
				r.rowCount, len(expectedHeaders), len(record))); cbErr != nil {
				return cbErr
			}
			continue
		}

		row := &SystemTransactionRow{
			TrxID:           strings.TrimSpace(record[0]),
			Amount:          strings.TrimSpace(record[1]),
			Source:          strings.TrimSpace(record[2]),
			Type:            strings.TrimSpace(record[3]),
			TransactionTime: strings.TrimSpace(record[4]),
			RowNumber:       r.rowCount,
		}

		if err := callback(row, nil); err != nil {
			return err
		}
	}

	return nil
}

// ReadBankStatements reads bank statement transactions in streaming fashion.
// Validates headers, parses each row, and invokes callback for processing.
// Errors are passed to callback allowing graceful handling and continuation.
func (r *Reader) ReadBankStatements(callback func(*BankStatementRow, error) error) error {
	defer r.Close()

	expectedHeaders := []string{"unique_identifier", "amount", "date"}
	if !r.validateHeaders(expectedHeaders) {
		return fmt.Errorf("invalid headers in bank statement file. Expected: %v, Got: %v",
			expectedHeaders, r.headers)
	}

	for {
		record, err := r.reader.Read()
		if err == io.EOF {
			break
		}

		r.rowCount++

		if err != nil {
			if cbErr := callback(nil, fmt.Errorf("row %d: failed to read: %w", r.rowCount, err)); cbErr != nil {
				return cbErr
			}
			continue
		}

		if len(record) != len(expectedHeaders) {
			if cbErr := callback(nil, fmt.Errorf("row %d: expected %d columns, got %d",
				r.rowCount, len(expectedHeaders), len(record))); cbErr != nil {
				return cbErr
			}
			continue
		}

		row := &BankStatementRow{
			UniqueIdentifier: strings.TrimSpace(record[0]),
			Amount:           strings.TrimSpace(record[1]),
			Date:             strings.TrimSpace(record[2]),
			RowNumber:        r.rowCount,
		}

		if err := callback(row, nil); err != nil {
			return err
		}
	}

	return nil
}

// Close closes the underlying file
func (r *Reader) Close() error {
	if r.file != nil {
		return r.file.Close()
	}
	return nil
}

// validateHeaders checks if the actual headers match expected (case-insensitive)
func (r *Reader) validateHeaders(expected []string) bool {
	if len(r.headers) != len(expected) {
		return false
	}

	for i, header := range r.headers {
		if !strings.EqualFold(header, expected[i]) {
			return false
		}
	}

	return true
}

// ParseSystemTransaction converts a SystemTransactionRow to a Transaction entity.
// Parses and validates amount, type (DEBIT/CREDIT), and timestamp (RFC3339 format).
// Stores raw data for audit and normalizes amount based on transaction type.
func ParseSystemTransaction(row *SystemTransactionRow, jobID, fileID string) (*transaction.Transaction, error) {
	amount, err := strconv.ParseFloat(row.Amount, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid amount %q: %w", row.Amount, err)
	}

	txnType := domain.TransactionTypeCredit
	if strings.ToUpper(row.Type) == "DEBIT" {
		txnType = domain.TransactionTypeDebit
	} else if strings.ToUpper(row.Type) != "CREDIT" {
		return nil, fmt.Errorf("invalid transaction type %q", row.Type)
	}

	txnTime, err := time.Parse(time.RFC3339, row.TransactionTime)
	if err != nil {
		return nil, fmt.Errorf("invalid transaction time %q: %w", row.TransactionTime, err)
	}

	txn := transaction.NewTransaction(
		jobID,
		fileID,
		domain.SourceTypeSystem,
		txnTime,
		amount,
		txnType,
		strings.ToUpper(row.Source),
	)
	txn.ID = row.TrxID

	txn.RawData = map[string]any{
		"trxID":           row.TrxID,
		"amount":          row.Amount,
		"source":          row.Source,
		"type":            row.Type,
		"transactionTime": row.TransactionTime,
		"rowNumber":       row.RowNumber,
	}

	txn.NormalizeAmount()
	return txn, nil
}

// ParseBankTransaction converts a BankStatementRow to a Transaction entity.
// Parses amount and date, determines transaction type from amount sign (negative=debit).
// Stores raw data for audit and normalizes amount.
func ParseBankTransaction(row *BankStatementRow, jobID, fileID, bankSource string) (*transaction.Transaction, error) {
	amount, err := strconv.ParseFloat(row.Amount, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid amount %q: %w", row.Amount, err)
	}

	txnType := domain.TransactionTypeCredit
	if amount < 0 {
		txnType = domain.TransactionTypeDebit
	}

	txnDate, err := parseDate(row.Date)
	if err != nil {
		return nil, fmt.Errorf("invalid date %q: %w", row.Date, err)
	}

	txn := transaction.NewTransaction(
		jobID,
		fileID,
		domain.SourceTypeBank,
		txnDate,
		amount,
		txnType,
		strings.ToUpper(bankSource),
	)
	txn.ID = row.UniqueIdentifier

	txn.RawData = map[string]any{
		"unique_identifier": row.UniqueIdentifier,
		"amount":            row.Amount,
		"date":              row.Date,
		"bankSource":        bankSource,
		"rowNumber":         row.RowNumber,
	}

	txn.NormalizeAmount()
	return txn, nil
}

// parseDate parses various date formats
func parseDate(dateStr string) (time.Time, error) {
	formats := []string{
		"2006-01-02",
		"2006/01/02",
		"02-01-2006",
		"02/01/2006",
		"2006-01-02T15:04:05Z",
		time.RFC3339,
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %s", dateStr)
}

// ExtractBankSourceFromFilename extracts the bank source from a filename.
// Expected format: {bank_name}_statement_{date}.csv
// Example: mandiri_statement_2024-03-15.csv returns "MANDIRI"
func ExtractBankSourceFromFilename(filePath string) (string, error) {
	filename := filepath.Base(filePath)
	parts := strings.Split(filename, "_")

	if len(parts) < 2 {
		return "", fmt.Errorf("invalid bank statement filename format: %s", filename)
	}

	bankName := strings.ToUpper(parts[0])
	if bankName == "" {
		return "", fmt.Errorf("could not extract bank name from filename: %s", filename)
	}

	return bankName, nil
}
