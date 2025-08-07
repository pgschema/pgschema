# Apply Command

## Running Tests

```bash
# All apply tests
go test -v ./cmd/apply/

# Specific apply tests
go test -v ./cmd/apply/ -run "TestApplyCommand_TransactionRollback"
```
