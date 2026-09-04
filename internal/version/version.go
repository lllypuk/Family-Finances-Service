// Package version хранит версию сборки.
package version

// Version подставляется линкером: -X family-budget-service/internal/version.Version=<ver>.
// Без -ldflags остаётся "dev".
var Version = "dev"

// String возвращает версию сборки.
func String() string {
	return Version
}
