package source

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
)

// PostgresSource executes queries against a PostgreSQL database
type PostgresSource struct {
	name  string
	query string
	dsn   string
	db    *sql.DB
}

// NewPostgresSource creates a new PostgreSQL source
func NewPostgresSource(name, query string, options map[string]string) (*PostgresSource, error) {
	if query == "" {
		return nil, fmt.Errorf("pg source %q: query is required", name)
	}

	// Get DSN from options or environment variable
	dsn := ""
	if options != nil {
		dsn = options["dsn"]
	}
	if dsn == "" {
		dsn = os.Getenv("DATABASE_URL")
	}
	if dsn == "" {
		return nil, fmt.Errorf("pg source %q: database connection required (set dsn in options or DATABASE_URL env)", name)
	}

	// Open connection
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("pg source %q: failed to open database: %w", name, err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("pg source %q: failed to connect: %w", name, err)
	}

	return &PostgresSource{
		name:  name,
		query: query,
		dsn:   dsn,
		db:    db,
	}, nil
}

// Name returns the source identifier
func (s *PostgresSource) Name() string {
	return s.name
}

// Fetch executes the query and returns results
func (s *PostgresSource) Fetch(ctx context.Context) ([]map[string]interface{}, error) {
	// Execute query with timeout
	queryCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	rows, err := s.db.QueryContext(queryCtx, s.query)
	if err != nil {
		return nil, fmt.Errorf("pg source %q: query failed: %w", s.name, err)
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("pg source %q: failed to get columns: %w", s.name, err)
	}

	// Prepare result slice
	var results []map[string]interface{}

	// Iterate rows
	for rows.Next() {
		// Create a slice of interface{} to hold column values
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		// Scan row into values
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("pg source %q: failed to scan row: %w", s.name, err)
		}

		// Build map from column names to values
		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			// Handle []byte (common for text columns)
			if b, ok := val.([]byte); ok {
				val = string(b)
			}
			row[col] = val
		}
		results = append(results, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("pg source %q: row iteration error: %w", s.name, err)
	}

	return results, nil
}

// Close releases the database connection
func (s *PostgresSource) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}
