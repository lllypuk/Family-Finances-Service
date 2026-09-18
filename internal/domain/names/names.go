// Package names holds the uniqueness key of user-entered names.
package names

import "strings"

// Key — ключ уникальности имени в семье. Алгоритм не менять после релиза: ключи уже записаны в базу.
func Key(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}
