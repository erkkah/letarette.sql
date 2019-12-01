//+build postgres

package adapter

import _ "github.com/lib/pq"

// postgres://pqgotest:password@localhost/pqgotest?sslmode=verify-full
// Use $1, $1, et.c. for accessing bound parameters.

func init() {
	addDriver("postgres")
}
