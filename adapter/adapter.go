package adapter

import (
	"context"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/erkkah/letarette/pkg/client"
	"github.com/erkkah/letarette/pkg/protocol"
	"github.com/jmoiron/sqlx"
)

// Adapter is a bridge between Letarette and a SQL database,
// implementing a Letarette Document Manager for one Letarette space
// by adding two SQL queries.
type Adapter interface {
	Close()
}

type adapter struct {
	manager            client.DocumentManager
	ctx                context.Context
	cancel             context.CancelFunc
	db                 *sqlx.DB
	indexRequestSQL    string
	documentRequestSQL string
	space              string
}

func (a *adapter) Close() {
	a.cancel()
	a.manager.Close()
}

// New creates an Adapter instance, connected to both NATS
// and the database, ready to start handling index requests.
func New(cfg Config, errorHandler func(error)) (Adapter, error) {
	mgr, err := client.StartDocumentManager(
		cfg.Nats.URL,
		client.WithTopic(cfg.Nats.Topic),
		client.WithErrorHandler(errorHandler),
	)
	if err != nil {
		return nil, fmt.Errorf("Failed to start document manager: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	db, err := sqlx.ConnectContext(ctx, cfg.Db.Driver, cfg.Db.Connection)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("DB connection failed: %w", err)
	}

	self := &adapter{
		manager: mgr,
		ctx:     ctx,
		cancel:  cancel,
		db:      db,
		space:   cfg.Index.Space,
	}

	bytes, err := ioutil.ReadFile(cfg.SQL.IndexSQLFile)
	if err != nil {
		return nil, fmt.Errorf("Failed to load index update query file: %w", err)
	}
	self.indexRequestSQL = string(bytes)

	bytes, err = ioutil.ReadFile(cfg.SQL.DocumentSQLFile)
	if err != nil {
		return nil, fmt.Errorf("Failed to load document update query file: %w", err)
	}
	self.documentRequestSQL = string(bytes)

	indexHandler := func(req protocol.IndexUpdateRequest) (protocol.IndexUpdate, error) {
		return self.handleIndexRequest(req)
	}

	documentHandler := func(req protocol.DocumentRequest) (protocol.DocumentUpdate, error) {
		return self.handleDocumentRequest(req)
	}

	mgr.StartIndexRequestHandler(indexHandler)
	mgr.StartDocumentRequestHandler(documentHandler)

	return self, nil
}

func (a *adapter) handleIndexRequest(req protocol.IndexUpdateRequest) (protocol.IndexUpdate, error) {
	if req.Space != a.space {
		return protocol.IndexUpdate{}, nil
	}
	// select id, updated
	rows, err := a.db.QueryContext(a.ctx, a.indexRequestSQL, req.AfterDocument, req.FromTime.UnixNano(), req.Limit)
	if err != nil {
		return protocol.IndexUpdate{}, fmt.Errorf("Failed to query DB: %w", err)
	}
	result := protocol.IndexUpdate{
		Space: a.space,
	}
	for rows.Next() {
		rowData := struct {
			ID      string
			Updated int64
		}{}
		err = rows.Scan(&rowData)
		if err != nil {
			return protocol.IndexUpdate{}, fmt.Errorf("Failed to scan row: %w", err)
		}
		result.Updates = append(result.Updates, protocol.DocumentReference{
			ID:      protocol.DocumentID(rowData.ID),
			Updated: time.Unix(0, rowData.Updated),
		})
	}
	return result, nil
}

func (a *adapter) handleDocumentRequest(req protocol.DocumentRequest) (protocol.DocumentUpdate, error) {
	if req.Space != a.space {
		return protocol.DocumentUpdate{}, nil
	}
	// select id, updated, title, text, alive
	rows, err := a.db.QueryContext(a.ctx, a.documentRequestSQL, req.Wanted)
	if err != nil {
		return protocol.DocumentUpdate{}, fmt.Errorf("Failed to query DB: %w", err)
	}

	result := protocol.DocumentUpdate{
		Space: a.space,
	}
	for rows.Next() {
		rowData := struct {
			ID      string
			Updated int64
			Title   string
			Text    string
			Alive   bool
		}{}
		err = rows.Scan(&rowData)
		if err != nil {
			return protocol.DocumentUpdate{}, fmt.Errorf("Failed to scan row: %w", err)
		}
		result.Documents = append(result.Documents, protocol.Document{
			ID:      protocol.DocumentID(rowData.ID),
			Updated: time.Unix(0, rowData.Updated),
			Title:   rowData.Title,
			Text:    rowData.Text,
			Alive:   rowData.Alive,
		})
	}
	return result, nil
}
