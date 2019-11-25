//+build sqlite3

package adapter

import _ "github.com/mattn/go-sqlite3"

func init() {
	addDriver("sqlite3")
}
