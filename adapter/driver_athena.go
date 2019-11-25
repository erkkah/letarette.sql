//+build athena

package adapter

import _ "github.com/segmentio/go-athena"

// "db=default&output_location=s3://results"

func init() {
	addDriver("athena")
}
