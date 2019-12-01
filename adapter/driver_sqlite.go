//+build sqlite3

package adapter

import _ "github.com/mattn/go-sqlite3"

// Use ? for accessing bound parameters

func init() {
	addDriver("sqlite3")
}
