Help me add or modify a severity level in the DEWCON system.

Ask which sub-system (JuiceCon or CCF), the level number, dewpoint threshold, descriptor name, and description text. Then update the appropriate calculator in `internal/juicecon/calculator.go` or `internal/ccf/calculator.go`, adjusting the switch statement thresholds. If this changes the overall range, also update `internal/index/system.go`. Run `go build ./...` and `go vet ./...` to verify.
