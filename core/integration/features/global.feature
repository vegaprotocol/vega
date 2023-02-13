# 0005-COLL-001 - Collateral engine emits an event on each transfer with source account, destination account and amount:
#                   checked automatically after each feature test step (account balance differencens reconciled with emitted transfer events),
#                   see core/integration/main_test.go:s.StepContext().After... for details
# 0005-COLL-002 - In absence of deposits or withdrawals via a bridge the total amount of any asset across all the accounts for the asset remains constant:
#                   checked automatically after each scenario
#                   see core/integration/main_test.go:s.After... for details