//+build postgres

package adapter

import _ "github.com/lib/pq"

// postgres://pqgotest:password@localhost/pqgotest?sslmode=verify-full

func init() {
	addDriver("postgres")
}
