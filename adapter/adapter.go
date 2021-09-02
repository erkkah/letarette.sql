package adapter

import (
	"context"
	"fmt"
	"io/ioutil"
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
	documentRequestSQL string
	space              string
}

func (a *adapter) Close() {
	a.manager.Close()
}

func stripComments(commented string) string {
	// Strip comments to avoid name binding getting caught on the url
	// in the license header and other colon characters.

	lines := strings.Split(string(commented), "\n")
	uncommented := []string{}
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "--") {
			continue
		}
		uncommented = append(uncommented, trimmed)
	}
	result := strings.Join(uncommented, "\n")

	return result
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
	self.indexRequestSQL = stripComments(string(bytes))

	bytes, err = ioutil.ReadFile(cfg.SQL.DocumentSQLFile)
	if err != nil {
		return nil, fmt.Errorf("Failed to load document update query file: %w", err)
	}
	self.documentRequestSQL = stripComments(string(bytes))

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

func (a *adapter) handleIndexRequest(ctx context.Context, req protocol.IndexUpdateRequest) (protocol.IndexUpdate, error) {
	if req.Space != a.space {
		return protocol.IndexUpdate{}, nil
	}

	logger.Debug.Printf("Index req: [%v]@%v\n", req, req.FromTime.UnixNano())

	params := struct {
		AfterDocument string `db:"afterDocument"`
		FromTimeNanos int64  `db:"fromTimeNanos"`
		DocumentLimit uint16 `db:"documentLimit"`
	}{
		string(req.AfterDocument), req.FromTime.UnixNano(), req.Limit,
	}

	start := time.Now()

	// select id, updated
	rows, err := a.db.NamedQueryContext(ctx,
		a.indexRequestSQL,
		params,
	)
	if err != nil {
		return protocol.IndexUpdate{}, fmt.Errorf("Failed to execute index query: %w", err)
	}

	duration := time.Since(start)

	logger.Debug.Printf("Index query took %v\n", duration)

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

	if len(result.Updates) > 0 {
		first := result.Updates[0].Updated
		last := result.Updates[len(result.Updates)-1].Updated
		logger.Debug.Printf("Returning %d updates (%v - %v) to indexer", len(result.Updates), first, last)
	}
	return result, nil
}

func (a *adapter) handleDocumentRequest(ctx context.Context, req protocol.DocumentRequest) (protocol.DocumentUpdate, error) {
	if req.Space != a.space {
		return protocol.DocumentUpdate{}, nil
	}

	arg := struct {
		WantedIDs []protocol.DocumentID `db:"wantedIDs"`
	}{
		req.Wanted,
	}

	// select id, updated, title, text, alive
	expanded, args, err := sqlx.Named(a.documentRequestSQL, arg)
	if err != nil {
		return protocol.DocumentUpdate{}, fmt.Errorf(`Failed to expand named parameters: %w`, err)
	}

	expanded, args, err = sqlx.In(expanded, args...)
	if err != nil {
		return protocol.DocumentUpdate{}, fmt.Errorf(`Failed to expand "in" statement: %w`, err)
	}

	expanded = a.db.Rebind(expanded)
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
