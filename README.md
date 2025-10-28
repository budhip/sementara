# Transaction Reconciliation

Compares your transaction records with bank statements and tells you what matches and what doesn't.

## Usage

```bash
# Build it
go build -o bin/reconcile ./cmd/reconcile/

# Run it
./bin/reconcile \
  -system your_transactions.csv \
  -banks "bank1.csv,bank2.csv" \
  -start 2024-03-15 \
  -end 2024-03-22
```

That's it. It reads the CSVs, matches transactions, and prints a report.

## CSV Files

Transactions file:
```csv
trxID,amount,source,type,transactionTime
TRX001,150.50,BCA,DEBIT,2024-03-15T10:30:00Z
```

Bank statements (filename must be `{bank}_statement_{date}.csv`):
```csv
unique_identifier,amount,date
BCA_TX_001,-150.50,2024-03-15
```

Note: Bank amounts are negative for debits, positive for credits.

## How Matching Works

A transaction matches if the date, type, and amount are identical. That's it.

If the same transaction appears twice in the same bank (same date/type/amount), both get marked as unmatched. Better to flag it for manual review than guess wrong.

## What You Get

```
RECONCILIATION REPORT
Reconciliation Period: 2024-03-15 to 2024-03-22

SUMMARY
Total Transactions Processed:		24
System transactions:         		12
Bank transactions:           		12
Matched Transactions:          	8 (66.7%)
Unmatched Transactions:				8
Unmatched system:					   4
Unmatched bank:						4
Total Discrepancy Amount:			0.00

UNMATCHED SYSTEM TRANSACTIONS
Transactions in system but missing in bank statement(s):

ID: TRX001 | Source: BCA | Type: DEBIT | Amount: 150.50 | Date: 2024-03-15
...
```

## Testing

```bash
go test ./...
```

Sample CSV files are in `fixtures/` if you want to try it out first.

## Code Structure

```
cmd/reconcile/main.go              # Reads CSVs, runs matching, prints report
pkg/matcher/exact_matcher.go      # The matching logic
internal/infrastructure/csv/       # CSV parsing
internal/domain/transaction/       # Transaction data structure
```

The matching algorithm builds a hash map of bank transactions, then checks each system transaction against it. O(n) time complexity.

## Adding Different Matching Logic

Create a new file in `pkg/matcher/` and implement this:

```go
type TransactionMatcher interface {
    Match(systemTxns, bankTxns []*transaction.Transaction) (*MatchResult, error)
    Name() string
    SetConfig(config MatcherConfig)
}
```

Then update `cmd/reconcile/main.go` to use your new matcher instead of `ExactMatcher`.

