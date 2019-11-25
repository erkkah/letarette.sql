// +build !athena,!mssql,!mysql,!postgres,!sqlite3

package adapter

var warn int = "Refusing to build with no valid driver tag!"
