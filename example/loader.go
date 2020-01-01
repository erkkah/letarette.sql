package main

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"

	"github.com/docopt/docopt-go"
)

var config struct {
	Source   string `docopt:"<json>"`
	DBFile   string `docopt:"<dbfile>"`
	Verbose  bool   `docopt:"-v"`
	Limit    string `docopt:"-l"`
	NumLimit int
}

type entry struct {
	Title string    `json:"title"`
	Text  string    `json:"text"`
	Date  time.Time `json:"date"`
}

func main() {
	usage := `JSON document to sqlite db loader.

Usage:
    loader [-l <limit>] [-v] <dbfile> <json>

Options:
    -l <limit>  Max number of documents to load [default: unlimited]
    -v          Verbose
`

	args, err := docopt.ParseDoc(usage)
	if err != nil {
		fmt.Printf("Failed to parse args: %v", err)
		os.Exit(1)
	}

	err = args.Bind(&config)
	if err != nil {
		fmt.Printf("Failed to bind args: %v", err)
		os.Exit(1)
	}

	if config.Limit != "unlimited" {
		config.NumLimit, _ = strconv.Atoi(config.Limit)
	}

	loader, err := newLoader()
	if err != nil {
		fmt.Printf("Failed to create loader: %v", err)
		os.Exit(1)
	}
	defer func() {
		err := loader.close()
		if err != nil {
			fmt.Printf("Failed to close db: %v", err)
		}
	}()

	err = loader.loadDocuments(config.Source)
	if err != nil {
		fmt.Printf("Failed to load documents: %v", err)
		os.Exit(1)
	}
}

type loader struct {
	db       *sqlx.DB
	tx       *sqlx.Tx
	inserter *sqlx.Stmt
}

func newLoader() (*loader, error) {
	sqliteURL, err := getDatabaseURL(config.DBFile)
	if err != nil {
		return nil, fmt.Errorf("Failed to get database URL: %w", err)
	}

	db, err := sqlx.Connect("sqlite3", sqliteURL)
	if err != nil {
		return nil, fmt.Errorf("Failed to open db: %v", err)
	}

	err = initDatabase(db)
	if err != nil {
		return nil, err
	}

	tx, err := db.Beginx()
	if err != nil {
		return nil, err
	}

	inserter, err := tx.Preparex("insert into docs (title, txt, updated) values (?, ?, ?)")
	if err != nil {
		return nil, err
	}

	return &loader{
		db:       db,
		tx:       tx,
		inserter: inserter,
	}, nil
}

func (l *loader) loadDocuments(objFile string) error {
	var fileReader io.Reader

	file, err := os.Open(objFile)
	if err != nil {
		return err
	}
	defer file.Close()
	fileReader = file

	if strings.HasSuffix(objFile, ".gz") {
		gzipReader, err := gzip.NewReader(fileReader)
		if err != nil {
			return err
		}
		defer gzipReader.Close()
		fileReader = gzipReader
	}

	decoder := json.NewDecoder(fileReader)
	rawSize := 0
	count := 0

	report := func() {
		log.Printf("%v docs, %v MB text loaded\n", count, rawSize/1024/1024)
	}

	for config.NumLimit == 0 || count < config.NumLimit {

		var e entry
		readErr := decoder.Decode(&e)
		if readErr == nil {
			rawSize += len([]byte(e.Text))
			rawSize += len([]byte(e.Title))

			if e.Date.IsZero() {
				e.Date = time.Now()
			}

			err = l.storeEntry(e)
			if err != nil {
				return err
			}

			count++

			if config.Verbose && count%1000 == 0 {
				report()
			}
		} else {
			if readErr != io.EOF {
				return readErr
			}
			break
		}
	}

	if config.Verbose {
		report()
	}

	return nil
}

func (l *loader) storeEntry(e entry) error {
	res, err := l.inserter.Exec(e.Title, e.Text, e.Date.UnixNano())
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows != 1 {
		return fmt.Errorf("Unexpected insert result")
	}
	return nil
}

func (l *loader) close() error {
	fmt.Println("Committing...")
	err := l.tx.Commit()
	if err != nil {
		return err
	}
	return l.db.Close()
}

func getDatabaseURL(dbPath string) (string, error) {
	abspath, err := filepath.Abs(dbPath)
	if err != nil {
		return "", fmt.Errorf("Failed to get absolute path to DB: %w", err)
	}
	escapedPath := strings.Replace(abspath, " ", "%20", -1)

	return fmt.Sprintf("file:%s?_journal=WAL&_foreign_keys=true&_timeout=500&cache=private&_sync=1&_rt=true", escapedPath), nil
}

func initDatabase(db *sqlx.DB) error {
	_, err := db.Exec(`
	create table docs (
		id integer primary key,
		title text,
		txt text,
		updated integer
	);
	create index docs_updated on docs(updated);
	`)
	return err
}
