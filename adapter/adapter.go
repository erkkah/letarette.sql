package adapter

import (
	"context"
	"fmt"
	"io/ioutil"
	"regexp"
	"strings"
	"time"

	"github.com/erkkah/letarette/pkg/client"
	"github.com/erkkah/letarette/pkg/logger"
	"github.com/erkkah/letarette/pkg/protocol"
	"github.com/jmoiron/sqlx"
)

// Adapter is a bridge between Letarette and a SQL database,
// implementing a Letarette Document Manager for a Letarette space
// using two SQL queries.
type Adapter interface {
	Close()
}

type adapter struct {
	manager            client.DocumentManager
	db                 *sqlx.DB
	indexRequestSQL    string
	indexParamOrder    []int
	documentRequestSQL string
	space              string
}

func (a *adapter) Close() {
	a.manager.Close()
}

// New creates an Adapter instance, connected to both NATS
// and the database, ready to start handling index requests.
func New(cfg Config, errorHandler func(error)) (Adapter, error) {
	mgr, err := client.StartDocumentManager(
		cfg.Nats.URLS,
		client.WithTopic(cfg.Nats.Topic),
		client.WithErrorHandler(errorHandler),
		client.WithRootCAs(cfg.Nats.RootCAs...),
		client.WithSeedFile(cfg.Nats.SeedFile),
	)
	if err != nil {
		return nil, fmt.Errorf("Failed to start document manager: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	db, err := sqlx.ConnectContext(ctx, cfg.Db.Driver, cfg.Db.Connection)
	cancel()

	if err != nil {
		return nil, fmt.Errorf("DB connection failed: %w", err)
	}

	logger.Info.Printf("Connected to %q DB using %q", cfg.Db.Driver, cfg.Db.Connection)

	self := &adapter{
		manager: mgr,
		db:      db,
		space:   cfg.Index.Space,
	}

	bytes, err := ioutil.ReadFile(cfg.SQL.IndexSQLFile)
	if err != nil {
		return nil, fmt.Errorf("Failed to load index update query file: %w", err)
	}
	self.indexRequestSQL = string(bytes)

	bindingOrder, err := extractBindingOrder(self.indexRequestSQL)
	if err != nil {
		return nil, err
	}
	if len(bindingOrder) != 0 {
		self.indexParamOrder = bindingOrder
	}

	bytes, err = ioutil.ReadFile(cfg.SQL.DocumentSQLFile)
	if err != nil {
		return nil, fmt.Errorf("Failed to load document update query file: %w", err)
	}
	self.documentRequestSQL = string(bytes)

	indexHandler := func(ctx context.Context, req protocol.IndexUpdateRequest) (protocol.IndexUpdate, error) {
		return self.handleIndexRequest(ctx, req)
	}

	documentHandler := func(ctx context.Context, req protocol.DocumentRequest) (protocol.DocumentUpdate, error) {
		return self.handleDocumentRequest(ctx, req)
	}

	mgr.StartIndexRequestHandler(indexHandler)
	mgr.StartDocumentRequestHandler(documentHandler)

	return self, nil
}

var matchBinding = regexp.MustCompile(`@\[([\w,\s]+)\]`)

func extractBindingOrder(query string) ([]int, error) {
	lines := strings.Split(query, "\n")

	order := [3]int{}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "---") {
			match := matchBinding.FindStringSubmatch(line)
			if match != nil && len(match) == 1 {
				paramOrder := strings.Split(match[0], ",")
				if len(paramOrder) != 3 {
					return []int{}, fmt.Errorf("Invalid number of bind parameters: %v", line)
				}
				for index, param := range paramOrder {
					switch param {
					case "afterDocument":
						order[index] = 0
					case "fromTimeNanos":
						order[index] = 1
					case "documentLimit":
						order[index] = 2
					default:
						return []int{}, fmt.Errorf("Invalid bind statement: %v", line)
					}
				}
				return order[:], nil
			}
		}
	}

	return []int{0, 1, 2}, nil
}

func (a *adapter) handleIndexRequest(ctx context.Context, req protocol.IndexUpdateRequest) (protocol.IndexUpdate, error) {
	if req.Space != a.space {
		return protocol.IndexUpdate{}, nil
	}

	params := []interface{}{
		req.AfterDocument, req.FromTime.UnixNano(), req.Limit,
	}

	// select id, updated
	rows, err := a.db.QueryxContext(ctx,
		a.indexRequestSQL,
		params[a.indexParamOrder[0]],
		params[a.indexParamOrder[1]],
		params[a.indexParamOrder[2]],
	)
	if err != nil {
		return protocol.IndexUpdate{}, fmt.Errorf("Failed to query DB: %w", err)
	}

	result := protocol.IndexUpdate{
		Space: a.space,
	}
	for rows.Next() {
		rowData := struct {
			ID      string
			Updated int64 `db:"updatedNanos"`
		}{}
		err = rows.StructScan(&rowData)
		if err != nil {
			return protocol.IndexUpdate{}, fmt.Errorf("Failed to scan row: %w", err)
		}
		result.Updates = append(result.Updates, protocol.DocumentReference{
			ID:      protocol.DocumentID(rowData.ID),
			Updated: time.Unix(0, rowData.Updated),
		})
	}

	logger.Debug.Printf("Returning %d updates to indexer", len(result.Updates))
	return result, nil
}

func (a *adapter) handleDocumentRequest(ctx context.Context, req protocol.DocumentRequest) (protocol.DocumentUpdate, error) {
	if req.Space != a.space {
		return protocol.DocumentUpdate{}, nil
	}

	// select id, updated, title, text, alive
	expanded, args, err := sqlx.In(a.documentRequestSQL, req.Wanted)
	if err != nil {
		return protocol.DocumentUpdate{}, fmt.Errorf(`Failed to expand "in" statement: %w`, err)
	}

	rows, err := a.db.QueryxContext(ctx, expanded, args...)
	if err != nil {
		return protocol.DocumentUpdate{}, fmt.Errorf("Failed to query DB: %w", err)
	}

	result := protocol.DocumentUpdate{
		Space: a.space,
	}
	for rows.Next() {
		rowData := struct {
			ID      string
			Updated int64 `db:"updatedNanos"`
			Title   string
			Txt     string
			Alive   bool
		}{}
		err = rows.StructScan(&rowData)
		if err != nil {
			return protocol.DocumentUpdate{}, fmt.Errorf("Failed to scan row: %w", err)
		}
		result.Documents = append(result.Documents, protocol.Document{
			ID:      protocol.DocumentID(rowData.ID),
			Updated: time.Unix(0, rowData.Updated),
			Title:   rowData.Title,
			Text:    rowData.Txt,
			Alive:   rowData.Alive,
		})
	}

	logger.Debug.Printf("Sending %d documents to indexer", len(result.Documents))
	return result, nil
}
