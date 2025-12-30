package source

import (
	"context"
	"database/sql"
	"os"
	"time"

	"github.com/livetemplate/tinkerdown/internal/config"

	_ "github.com/lib/pq" // PostgreSQL driver
)

// PostgresSource executes queries against a PostgreSQL database
type PostgresSource struct {
	name           string
	query          string
	dsn            string
	db             *sql.DB
	timeout        time.Duration
	retryConfig    RetryConfig
	circuitBreaker *CircuitBreaker
}

// NewPostgresSource creates a new PostgreSQL source
func NewPostgresSource(name, query string, options map[string]string) (*PostgresSource, error) {
	return NewPostgresSourceWithConfig(name, query, options, config.SourceConfig{})
}

// NewPostgresSourceWithConfig creates a new PostgreSQL source with full configuration
func NewPostgresSourceWithConfig(name, query string, options map[string]string, cfg config.SourceConfig) (*PostgresSource, error) {
	if query == "" {
		return nil, &ValidationError{Source: name, Field: "query", Reason: "query is required"}
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
		return nil, &ValidationError{Source: name, Field: "dsn", Reason: "database connection required (set dsn in options or DATABASE_URL env)"}
	}

	// Open connection
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, &ConnectionError{Source: name, Address: "postgres", Err: err}
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
		return nil, &ConnectionError{Source: name, Address: "postgres", Err: err}
	}

	// Get timeout from config
	timeout := cfg.GetTimeout()

	// Build retry config
	retryConfig := RetryConfig{
		MaxRetries: cfg.GetRetryMaxRetries(),
		BaseDelay:  cfg.GetRetryBaseDelay(),
		MaxDelay:   cfg.GetRetryMaxDelay(),
		Multiplier: 2.0,
		EnableLog:  true,
	}

	// Create circuit breaker
	cbConfig := DefaultCircuitBreakerConfig()
	circuitBreaker := NewCircuitBreaker(name, cbConfig)

	return &PostgresSource{
		name:           name,
		query:          query,
		dsn:            dsn,
		db:             db,
		timeout:        timeout,
		retryConfig:    retryConfig,
		circuitBreaker: circuitBreaker,
	}, nil
}

// Name returns the source identifier
func (s *PostgresSource) Name() string {
	return s.name
}

// Fetch executes the query and returns results with retry and circuit breaker
func (s *PostgresSource) Fetch(ctx context.Context) ([]map[string]interface{}, error) {
	return s.circuitBreaker.Execute(ctx, func(ctx context.Context) ([]map[string]interface{}, error) {
		return WithRetry(ctx, s.name, s.retryConfig, func(ctx context.Context) ([]map[string]interface{}, error) {
			return s.doFetch(ctx)
		})
	})
}

// doFetch performs the actual query
func (s *PostgresSource) doFetch(ctx context.Context) ([]map[string]interface{}, error) {
	// Execute query with timeout
	queryCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	rows, err := s.db.QueryContext(queryCtx, s.query)
	if err != nil {
		return nil, NewSourceError(s.name, "query", err)
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return nil, &SourceError{Source: s.name, Operation: "get columns", Err: err, Retryable: false}
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
			return nil, &SourceError{Source: s.name, Operation: "scan row", Err: err, Retryable: false}
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
		return nil, NewSourceError(s.name, "row iteration", err)
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
