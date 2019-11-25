//+build mysql

package adapter

import _ "github.com/go-sql-driver/mysql"

// user:password@/dbname

func init() {
	addDriver("mysql")
}
