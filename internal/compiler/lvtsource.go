package compiler

import (
	"fmt"
	"strings"

	"github.com/livetemplate/livepage/internal/config"
)

// GenerateLvtSourceCode generates Go code for an lvt-source block.
// The generated code creates a State struct that fetches data from the configured source.
// When metadata["lvt-element"] is "table", it generates datatable-aware code.
func GenerateLvtSourceCode(sourceName string, sourceCfg config.SourceConfig, siteDir string, metadata map[string]string) (string, error) {
	// Check if we should generate datatable code
	elementType := ""
	columns := ""
	if metadata != nil {
		elementType = metadata["lvt-element"]
		columns = metadata["lvt-columns"]
	}

	if elementType == "table" {
		// Generate datatable-aware code
		return generateDatatableSourceCode(sourceName, sourceCfg, siteDir, columns)
	}

	// Default: generate simple data code
	switch sourceCfg.Type {
	case "exec":
		return generateExecSourceCode(sourceName, sourceCfg.Cmd, siteDir, sourceCfg.Manual)
	case "pg":
		return generatePostgresSourceCode(sourceName, sourceCfg.Query, sourceCfg.Options)
	case "rest":
		return generateRestSourceCode(sourceName, sourceCfg.URL, sourceCfg.Options)
	case "json":
		return generateJSONFileSourceCode(sourceName, sourceCfg.File, siteDir)
	case "csv":
		return generateCSVFileSourceCode(sourceName, sourceCfg.File, siteDir, sourceCfg.Options)
	default:
		return "", fmt.Errorf("unsupported source type: %s", sourceCfg.Type)
	}
}

// generateExecSourceCode generates code for an exec source
func generateExecSourceCode(sourceName, cmd, siteDir string, manual bool) (string, error) {
	if cmd == "" {
		return "", fmt.Errorf("exec source %q: cmd is required", sourceName)
	}

	var code strings.Builder

	// Imports
	code.WriteString("import (\n")
	code.WriteString("\t\"bytes\"\n")
	code.WriteString("\t\"context\"\n")
	code.WriteString("\t\"encoding/json\"\n")
	code.WriteString("\t\"fmt\"\n")
	code.WriteString("\t\"os/exec\"\n")
	code.WriteString("\t\"strings\"\n")
	code.WriteString("\t\"time\"\n")
	code.WriteString(")\n\n")

	// Escape the command for embedding in Go code
	escapedCmd := strings.ReplaceAll(cmd, `"`, `\"`)
	escapedDir := strings.ReplaceAll(siteDir, `"`, `\"`)

	// State struct with Data field and execution metadata
	code.WriteString("// State holds data fetched from the source and execution metadata\n")
	code.WriteString("type State struct {\n")
	code.WriteString("\tData     []map[string]interface{} `json:\"data\"`\n")
	code.WriteString("\tError    string `json:\"error,omitempty\"`\n")
	code.WriteString("\tOutput   string `json:\"output,omitempty\"`\n")   // stdout
	code.WriteString("\tStderr   string `json:\"stderr,omitempty\"`\n")   // stderr
	code.WriteString("\tDuration int64  `json:\"duration,omitempty\"`\n") // execution time in ms
	code.WriteString("\tStatus   string `json:\"status\"`\n")             // idle, running, success, error
	code.WriteString("\tCommand  string `json:\"command\"`\n")            // command for display
	code.WriteString("}\n\n")

	// NewState constructor
	code.WriteString("// NewState creates a new State and optionally fetches initial data\n")
	code.WriteString("func NewState() (*State, error) {\n")
	code.WriteString(fmt.Sprintf("\ts := &State{Command: %q}\n", escapedCmd))
	if manual {
		// Manual mode: don't auto-execute
		code.WriteString("\ts.Status = \"idle\"\n")
	} else {
		// Auto-execute mode (default)
		code.WriteString("\ts.Status = \"running\"\n")
		code.WriteString("\tif err := s.fetchData(); err != nil {\n")
		code.WriteString("\t\ts.Error = err.Error()\n")
		code.WriteString("\t\ts.Status = \"error\"\n")
		code.WriteString("\t} else {\n")
		code.WriteString("\t\ts.Status = \"success\"\n")
		code.WriteString("\t}\n")
	}
	code.WriteString("\treturn s, nil\n")
	code.WriteString("}\n\n")

	// Close method
	code.WriteString("// Close is a no-op for source-only state\n")
	code.WriteString("func (s *State) Close() error {\n")
	code.WriteString("\treturn nil\n")
	code.WriteString("}\n\n")

	// Run action - executes the command (for manual mode or re-run)
	code.WriteString("// Run executes the command\n")
	code.WriteString("func (s *State) Run(ctx *livetemplate.Context) error {\n")
	code.WriteString("\ts.Status = \"running\"\n")
	code.WriteString("\ts.Error = \"\"\n")
	code.WriteString("\ts.Output = \"\"\n")
	code.WriteString("\ts.Stderr = \"\"\n")
	code.WriteString("\tif err := s.fetchData(); err != nil {\n")
	code.WriteString("\t\ts.Error = err.Error()\n")
	code.WriteString("\t\ts.Status = \"error\"\n")
	code.WriteString("\t} else {\n")
	code.WriteString("\t\ts.Status = \"success\"\n")
	code.WriteString("\t}\n")
	code.WriteString("\treturn nil\n")
	code.WriteString("}\n\n")

	// Refresh action - alias to Run for backwards compatibility
	code.WriteString("// Refresh re-fetches data from the source (alias to Run)\n")
	code.WriteString("func (s *State) Refresh(ctx *livetemplate.Context) error {\n")
	code.WriteString("\treturn s.Run(ctx)\n")
	code.WriteString("}\n\n")

	// fetchData method - executes the command and parses JSON
	code.WriteString("// fetchData executes the source command and parses JSON output\n")
	code.WriteString("func (s *State) fetchData() error {\n")
	code.WriteString(fmt.Sprintf("\tcmd := \"%s\"\n", escapedCmd))
	code.WriteString(fmt.Sprintf("\tworkDir := \"%s\"\n", escapedDir))
	code.WriteString("\n")

	// Parse and execute command
	code.WriteString("\t// Parse command\n")
	code.WriteString("\tparts := strings.Fields(cmd)\n")
	code.WriteString("\tif len(parts) == 0 {\n")
	code.WriteString("\t\treturn fmt.Errorf(\"empty command\")\n")
	code.WriteString("\t}\n")
	code.WriteString("\n")

	code.WriteString("\t// Execute with timeout\n")
	code.WriteString("\tstart := time.Now()\n")
	code.WriteString("\tctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)\n")
	code.WriteString("\tdefer cancel()\n")
	code.WriteString("\n")
	code.WriteString("\texecCmd := exec.CommandContext(ctx, parts[0], parts[1:]...)\n")
	code.WriteString("\texecCmd.Dir = workDir\n")
	code.WriteString("\n")

	// Use separate buffers for stdout/stderr
	code.WriteString("\t// Capture stdout and stderr separately\n")
	code.WriteString("\tvar stdoutBuf, stderrBuf bytes.Buffer\n")
	code.WriteString("\texecCmd.Stdout = &stdoutBuf\n")
	code.WriteString("\texecCmd.Stderr = &stderrBuf\n")
	code.WriteString("\n")
	code.WriteString("\terr := execCmd.Run()\n")
	code.WriteString("\ts.Duration = time.Since(start).Milliseconds()\n")
	code.WriteString("\ts.Output = stdoutBuf.String()\n")
	code.WriteString("\ts.Stderr = stderrBuf.String()\n")
	code.WriteString("\n")
	code.WriteString("\tif err != nil {\n")
	code.WriteString("\t\treturn fmt.Errorf(\"command failed: %w\", err)\n")
	code.WriteString("\t}\n")
	code.WriteString("\n")

	// Parse JSON
	code.WriteString("\t// Parse JSON output\n")
	code.WriteString("\treturn s.parseJSON([]byte(s.Output))\n")
	code.WriteString("}\n\n")

	// parseJSON method
	code.WriteString("// parseJSON handles array, object, and NDJSON formats\n")
	code.WriteString("func (s *State) parseJSON(data []byte) error {\n")
	code.WriteString("\tdata = []byte(strings.TrimSpace(string(data)))\n")
	code.WriteString("\tif len(data) == 0 {\n")
	code.WriteString("\t\ts.Data = []map[string]interface{}{}\n")
	code.WriteString("\t\treturn nil\n")
	code.WriteString("\t}\n")
	code.WriteString("\n")
	code.WriteString("\t// Try array first\n")
	code.WriteString("\tvar arr []map[string]interface{}\n")
	code.WriteString("\tif err := json.Unmarshal(data, &arr); err == nil {\n")
	code.WriteString("\t\ts.Data = arr\n")
	code.WriteString("\t\treturn nil\n")
	code.WriteString("\t}\n")
	code.WriteString("\n")
	code.WriteString("\t// Try single object\n")
	code.WriteString("\tvar obj map[string]interface{}\n")
	code.WriteString("\tif err := json.Unmarshal(data, &obj); err == nil {\n")
	code.WriteString("\t\ts.Data = []map[string]interface{}{obj}\n")
	code.WriteString("\t\treturn nil\n")
	code.WriteString("\t}\n")
	code.WriteString("\n")
	code.WriteString("\t// Try NDJSON (newline-delimited)\n")
	code.WriteString("\tlines := strings.Split(string(data), \"\\n\")\n")
	code.WriteString("\tvar results []map[string]interface{}\n")
	code.WriteString("\tfor _, line := range lines {\n")
	code.WriteString("\t\tline = strings.TrimSpace(line)\n")
	code.WriteString("\t\tif line == \"\" {\n")
	code.WriteString("\t\t\tcontinue\n")
	code.WriteString("\t\t}\n")
	code.WriteString("\t\tvar item map[string]interface{}\n")
	code.WriteString("\t\tif err := json.Unmarshal([]byte(line), &item); err != nil {\n")
	code.WriteString("\t\t\treturn fmt.Errorf(\"invalid JSON: %w\", err)\n")
	code.WriteString("\t\t}\n")
	code.WriteString("\t\tresults = append(results, item)\n")
	code.WriteString("\t}\n")
	code.WriteString("\n")
	code.WriteString("\tif len(results) > 0 {\n")
	code.WriteString("\t\ts.Data = results\n")
	code.WriteString("\t\treturn nil\n")
	code.WriteString("\t}\n")
	code.WriteString("\n")
	code.WriteString("\treturn fmt.Errorf(\"could not parse output as JSON\")\n")
	code.WriteString("}\n")

	return code.String(), nil
}

// generatePostgresSourceCode generates code for a PostgreSQL source
func generatePostgresSourceCode(sourceName, query string, options map[string]string) (string, error) {
	if query == "" {
		return "", fmt.Errorf("pg source %q: query is required", sourceName)
	}

	var code strings.Builder

	// Imports
	code.WriteString("import (\n")
	code.WriteString("\t\"context\"\n")
	code.WriteString("\t\"database/sql\"\n")
	code.WriteString("\t\"fmt\"\n")
	code.WriteString("\t\"os\"\n")
	code.WriteString("\t\"time\"\n")
	code.WriteString("\n")
	code.WriteString("\t_ \"github.com/lib/pq\"\n")
	code.WriteString(")\n\n")

	// State struct with Data field
	code.WriteString("// State holds data fetched from the database\n")
	code.WriteString("type State struct {\n")
	code.WriteString("\tData []map[string]interface{} `json:\"data\"`\n")
	code.WriteString("\tError string `json:\"error,omitempty\"`\n")
	code.WriteString("\tdb *sql.DB\n")
	code.WriteString("}\n\n")

	// NewState constructor
	code.WriteString("// NewState creates a new State and fetches initial data\n")
	code.WriteString("func NewState() (*State, error) {\n")
	code.WriteString("\ts := &State{}\n")
	code.WriteString("\tif err := s.connect(); err != nil {\n")
	code.WriteString("\t\ts.Error = err.Error()\n")
	code.WriteString("\t\treturn s, nil\n")
	code.WriteString("\t}\n")
	code.WriteString("\tif err := s.fetchData(); err != nil {\n")
	code.WriteString("\t\ts.Error = err.Error()\n")
	code.WriteString("\t}\n")
	code.WriteString("\treturn s, nil\n")
	code.WriteString("}\n\n")

	// Close method
	code.WriteString("// Close releases database resources\n")
	code.WriteString("func (s *State) Close() error {\n")
	code.WriteString("\tif s.db != nil {\n")
	code.WriteString("\t\treturn s.db.Close()\n")
	code.WriteString("\t}\n")
	code.WriteString("\treturn nil\n")
	code.WriteString("}\n\n")

	// Refresh action
	code.WriteString("// Refresh re-fetches data from the database\n")
	code.WriteString("func (s *State) Refresh(ctx *livetemplate.Context) error {\n")
	code.WriteString("\ts.Error = \"\"\n")
	code.WriteString("\tif s.db == nil {\n")
	code.WriteString("\t\tif err := s.connect(); err != nil {\n")
	code.WriteString("\t\t\ts.Error = err.Error()\n")
	code.WriteString("\t\t\treturn nil\n")
	code.WriteString("\t\t}\n")
	code.WriteString("\t}\n")
	code.WriteString("\tif err := s.fetchData(); err != nil {\n")
	code.WriteString("\t\ts.Error = err.Error()\n")
	code.WriteString("\t}\n")
	code.WriteString("\treturn nil\n")
	code.WriteString("}\n\n")

	// connect method
	code.WriteString("// connect establishes database connection\n")
	code.WriteString("func (s *State) connect() error {\n")

	// Get DSN from options or environment
	dsn := ""
	if options != nil {
		dsn = options["dsn"]
	}
	if dsn != "" {
		// Escape the DSN for embedding in Go code
		escapedDSN := strings.ReplaceAll(dsn, `"`, `\"`)
		code.WriteString(fmt.Sprintf("\tdsn := \"%s\"\n", escapedDSN))
	} else {
		code.WriteString("\tdsn := os.Getenv(\"DATABASE_URL\")\n")
	}
	code.WriteString("\tif dsn == \"\" {\n")
	code.WriteString("\t\treturn fmt.Errorf(\"DATABASE_URL not set\")\n")
	code.WriteString("\t}\n")
	code.WriteString("\n")
	code.WriteString("\tdb, err := sql.Open(\"postgres\", dsn)\n")
	code.WriteString("\tif err != nil {\n")
	code.WriteString("\t\treturn fmt.Errorf(\"failed to open database: %w\", err)\n")
	code.WriteString("\t}\n")
	code.WriteString("\n")
	code.WriteString("\tdb.SetMaxOpenConns(5)\n")
	code.WriteString("\tdb.SetMaxIdleConns(2)\n")
	code.WriteString("\tdb.SetConnMaxLifetime(5 * time.Minute)\n")
	code.WriteString("\n")
	code.WriteString("\tctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)\n")
	code.WriteString("\tdefer cancel()\n")
	code.WriteString("\tif err := db.PingContext(ctx); err != nil {\n")
	code.WriteString("\t\tdb.Close()\n")
	code.WriteString("\t\treturn fmt.Errorf(\"failed to connect: %w\", err)\n")
	code.WriteString("\t}\n")
	code.WriteString("\n")
	code.WriteString("\ts.db = db\n")
	code.WriteString("\treturn nil\n")
	code.WriteString("}\n\n")

	// fetchData method
	code.WriteString("// fetchData executes the query and populates Data\n")
	code.WriteString("func (s *State) fetchData() error {\n")

	// Escape the query for embedding in Go code
	escapedQuery := strings.ReplaceAll(query, `"`, `\"`)
	escapedQuery = strings.ReplaceAll(escapedQuery, "\n", "\\n")
	code.WriteString(fmt.Sprintf("\tquery := \"%s\"\n", escapedQuery))
	code.WriteString("\n")

	code.WriteString("\tctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)\n")
	code.WriteString("\tdefer cancel()\n")
	code.WriteString("\n")
	code.WriteString("\trows, err := s.db.QueryContext(ctx, query)\n")
	code.WriteString("\tif err != nil {\n")
	code.WriteString("\t\treturn fmt.Errorf(\"query failed: %w\", err)\n")
	code.WriteString("\t}\n")
	code.WriteString("\tdefer rows.Close()\n")
	code.WriteString("\n")
	code.WriteString("\tcolumns, err := rows.Columns()\n")
	code.WriteString("\tif err != nil {\n")
	code.WriteString("\t\treturn fmt.Errorf(\"failed to get columns: %w\", err)\n")
	code.WriteString("\t}\n")
	code.WriteString("\n")
	code.WriteString("\tvar results []map[string]interface{}\n")
	code.WriteString("\tfor rows.Next() {\n")
	code.WriteString("\t\tvalues := make([]interface{}, len(columns))\n")
	code.WriteString("\t\tvaluePtrs := make([]interface{}, len(columns))\n")
	code.WriteString("\t\tfor i := range values {\n")
	code.WriteString("\t\t\tvaluePtrs[i] = &values[i]\n")
	code.WriteString("\t\t}\n")
	code.WriteString("\n")
	code.WriteString("\t\tif err := rows.Scan(valuePtrs...); err != nil {\n")
	code.WriteString("\t\t\treturn fmt.Errorf(\"failed to scan row: %w\", err)\n")
	code.WriteString("\t\t}\n")
	code.WriteString("\n")
	code.WriteString("\t\trow := make(map[string]interface{})\n")
	code.WriteString("\t\tfor i, col := range columns {\n")
	code.WriteString("\t\t\tval := values[i]\n")
	code.WriteString("\t\t\tif b, ok := val.([]byte); ok {\n")
	code.WriteString("\t\t\t\tval = string(b)\n")
	code.WriteString("\t\t\t}\n")
	code.WriteString("\t\t\trow[col] = val\n")
	code.WriteString("\t\t}\n")
	code.WriteString("\t\tresults = append(results, row)\n")
	code.WriteString("\t}\n")
	code.WriteString("\n")
	code.WriteString("\tif err := rows.Err(); err != nil {\n")
	code.WriteString("\t\treturn fmt.Errorf(\"row iteration error: %w\", err)\n")
	code.WriteString("\t}\n")
	code.WriteString("\n")
	code.WriteString("\ts.Data = results\n")
	code.WriteString("\treturn nil\n")
	code.WriteString("}\n")

	return code.String(), nil
}

// generateRestSourceCode generates code for a REST API source
func generateRestSourceCode(sourceName, url string, options map[string]string) (string, error) {
	if url == "" {
		return "", fmt.Errorf("rest source %q: url is required", sourceName)
	}

	var code strings.Builder

	// Imports
	code.WriteString("import (\n")
	code.WriteString("\t\"context\"\n")
	code.WriteString("\t\"encoding/json\"\n")
	code.WriteString("\t\"fmt\"\n")
	code.WriteString("\t\"io\"\n")
	code.WriteString("\t\"net/http\"\n")
	code.WriteString("\t\"os\"\n")
	code.WriteString("\t\"strings\"\n")
	code.WriteString("\t\"time\"\n")
	code.WriteString(")\n\n")

	// State struct with Data field
	code.WriteString("// State holds data fetched from the REST API\n")
	code.WriteString("type State struct {\n")
	code.WriteString("\tData []map[string]interface{} `json:\"data\"`\n")
	code.WriteString("\tError string `json:\"error,omitempty\"`\n")
	code.WriteString("\tclient *http.Client\n")
	code.WriteString("}\n\n")

	// NewState constructor
	code.WriteString("// NewState creates a new State and fetches initial data\n")
	code.WriteString("func NewState() (*State, error) {\n")
	code.WriteString("\ts := &State{\n")
	code.WriteString("\t\tclient: &http.Client{Timeout: 30 * time.Second},\n")
	code.WriteString("\t}\n")
	code.WriteString("\tif err := s.fetchData(); err != nil {\n")
	code.WriteString("\t\ts.Error = err.Error()\n")
	code.WriteString("\t}\n")
	code.WriteString("\treturn s, nil\n")
	code.WriteString("}\n\n")

	// Close method
	code.WriteString("// Close is a no-op for REST sources\n")
	code.WriteString("func (s *State) Close() error {\n")
	code.WriteString("\treturn nil\n")
	code.WriteString("}\n\n")

	// Refresh action
	code.WriteString("// Refresh re-fetches data from the REST API\n")
	code.WriteString("func (s *State) Refresh(ctx *livetemplate.Context) error {\n")
	code.WriteString("\ts.Error = \"\"\n")
	code.WriteString("\tif err := s.fetchData(); err != nil {\n")
	code.WriteString("\t\ts.Error = err.Error()\n")
	code.WriteString("\t}\n")
	code.WriteString("\treturn nil\n")
	code.WriteString("}\n\n")

	// fetchData method
	code.WriteString("// fetchData makes HTTP request and parses JSON response\n")
	code.WriteString("func (s *State) fetchData() error {\n")

	// Escape the URL for embedding in Go code
	escapedURL := strings.ReplaceAll(url, `"`, `\"`)
	code.WriteString(fmt.Sprintf("\turl := os.ExpandEnv(\"%s\")\n", escapedURL))

	// Method
	method := "GET"
	if options != nil && options["method"] != "" {
		method = strings.ToUpper(options["method"])
	}
	code.WriteString(fmt.Sprintf("\tmethod := \"%s\"\n", method))
	code.WriteString("\n")

	code.WriteString("\tctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)\n")
	code.WriteString("\tdefer cancel()\n")
	code.WriteString("\n")
	code.WriteString("\treq, err := http.NewRequestWithContext(ctx, method, url, nil)\n")
	code.WriteString("\tif err != nil {\n")
	code.WriteString("\t\treturn fmt.Errorf(\"failed to create request: %w\", err)\n")
	code.WriteString("\t}\n")
	code.WriteString("\n")

	// Set headers
	code.WriteString("\treq.Header.Set(\"Accept\", \"application/json\")\n")

	// Parse headers from options
	if options != nil && options["headers"] != "" {
		for _, h := range strings.Split(options["headers"], ",") {
			parts := strings.SplitN(h, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				escapedKey := strings.ReplaceAll(key, `"`, `\"`)
				escapedValue := strings.ReplaceAll(value, `"`, `\"`)
				code.WriteString(fmt.Sprintf("\treq.Header.Set(\"%s\", os.ExpandEnv(\"%s\"))\n", escapedKey, escapedValue))
			}
		}
	}

	// Auth header
	if options != nil && options["auth_header"] != "" {
		escapedAuth := strings.ReplaceAll(options["auth_header"], `"`, `\"`)
		code.WriteString(fmt.Sprintf("\treq.Header.Set(\"Authorization\", os.ExpandEnv(\"%s\"))\n", escapedAuth))
	}

	// API key
	if options != nil && options["api_key"] != "" {
		escapedKey := strings.ReplaceAll(options["api_key"], `"`, `\"`)
		code.WriteString(fmt.Sprintf("\treq.Header.Set(\"X-API-Key\", os.ExpandEnv(\"%s\"))\n", escapedKey))
	}

	code.WriteString("\n")
	code.WriteString("\tresp, err := s.client.Do(req)\n")
	code.WriteString("\tif err != nil {\n")
	code.WriteString("\t\treturn fmt.Errorf(\"request failed: %w\", err)\n")
	code.WriteString("\t}\n")
	code.WriteString("\tdefer resp.Body.Close()\n")
	code.WriteString("\n")
	code.WriteString("\tif resp.StatusCode < 200 || resp.StatusCode >= 300 {\n")
	code.WriteString("\t\tbody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))\n")
	code.WriteString("\t\treturn fmt.Errorf(\"HTTP %d: %s\", resp.StatusCode, string(body))\n")
	code.WriteString("\t}\n")
	code.WriteString("\n")
	code.WriteString("\tbody, err := io.ReadAll(resp.Body)\n")
	code.WriteString("\tif err != nil {\n")
	code.WriteString("\t\treturn fmt.Errorf(\"failed to read response: %w\", err)\n")
	code.WriteString("\t}\n")
	code.WriteString("\n")
	code.WriteString("\treturn s.parseJSON(body)\n")
	code.WriteString("}\n\n")

	// parseJSON method
	code.WriteString("// parseJSON handles array, object, and wrapped responses\n")
	code.WriteString("func (s *State) parseJSON(data []byte) error {\n")
	code.WriteString("\tdata = []byte(strings.TrimSpace(string(data)))\n")
	code.WriteString("\tif len(data) == 0 {\n")
	code.WriteString("\t\ts.Data = []map[string]interface{}{}\n")
	code.WriteString("\t\treturn nil\n")
	code.WriteString("\t}\n")
	code.WriteString("\n")
	code.WriteString("\t// Try array first\n")
	code.WriteString("\tvar arr []map[string]interface{}\n")
	code.WriteString("\tif err := json.Unmarshal(data, &arr); err == nil {\n")
	code.WriteString("\t\ts.Data = arr\n")
	code.WriteString("\t\treturn nil\n")
	code.WriteString("\t}\n")
	code.WriteString("\n")
	code.WriteString("\t// Try single object\n")
	code.WriteString("\tvar obj map[string]interface{}\n")
	code.WriteString("\tif err := json.Unmarshal(data, &obj); err == nil {\n")
	code.WriteString("\t\t// Check for \"data\" field (common API pattern)\n")
	code.WriteString("\t\tif dataField, ok := obj[\"data\"]; ok {\n")
	code.WriteString("\t\t\tif dataArr, ok := dataField.([]interface{}); ok {\n")
	code.WriteString("\t\t\t\tresults := make([]map[string]interface{}, 0, len(dataArr))\n")
	code.WriteString("\t\t\t\tfor _, item := range dataArr {\n")
	code.WriteString("\t\t\t\t\tif itemMap, ok := item.(map[string]interface{}); ok {\n")
	code.WriteString("\t\t\t\t\t\tresults = append(results, itemMap)\n")
	code.WriteString("\t\t\t\t\t}\n")
	code.WriteString("\t\t\t\t}\n")
	code.WriteString("\t\t\t\tif len(results) > 0 {\n")
	code.WriteString("\t\t\t\t\ts.Data = results\n")
	code.WriteString("\t\t\t\t\treturn nil\n")
	code.WriteString("\t\t\t\t}\n")
	code.WriteString("\t\t\t}\n")
	code.WriteString("\t\t}\n")
	code.WriteString("\t\t// Check for \"results\" field\n")
	code.WriteString("\t\tif resultsField, ok := obj[\"results\"]; ok {\n")
	code.WriteString("\t\t\tif resultsArr, ok := resultsField.([]interface{}); ok {\n")
	code.WriteString("\t\t\t\tresults := make([]map[string]interface{}, 0, len(resultsArr))\n")
	code.WriteString("\t\t\t\tfor _, item := range resultsArr {\n")
	code.WriteString("\t\t\t\t\tif itemMap, ok := item.(map[string]interface{}); ok {\n")
	code.WriteString("\t\t\t\t\t\tresults = append(results, itemMap)\n")
	code.WriteString("\t\t\t\t\t}\n")
	code.WriteString("\t\t\t\t}\n")
	code.WriteString("\t\t\t\tif len(results) > 0 {\n")
	code.WriteString("\t\t\t\t\ts.Data = results\n")
	code.WriteString("\t\t\t\t\treturn nil\n")
	code.WriteString("\t\t\t\t}\n")
	code.WriteString("\t\t\t}\n")
	code.WriteString("\t\t}\n")
	code.WriteString("\t\ts.Data = []map[string]interface{}{obj}\n")
	code.WriteString("\t\treturn nil\n")
	code.WriteString("\t}\n")
	code.WriteString("\n")
	code.WriteString("\treturn fmt.Errorf(\"could not parse response as JSON\")\n")
	code.WriteString("}\n")

	return code.String(), nil
}

// generateJSONFileSourceCode generates code for a JSON file source
func generateJSONFileSourceCode(sourceName, file, siteDir string) (string, error) {
	if file == "" {
		return "", fmt.Errorf("json source %q: file is required", sourceName)
	}

	var code strings.Builder

	// Imports
	code.WriteString("import (\n")
	code.WriteString("\t\"encoding/json\"\n")
	code.WriteString("\t\"fmt\"\n")
	code.WriteString("\t\"os\"\n")
	code.WriteString("\t\"path/filepath\"\n")
	code.WriteString("\t\"strings\"\n")
	code.WriteString(")\n\n")

	// State struct
	code.WriteString("// State holds data from the JSON file\n")
	code.WriteString("type State struct {\n")
	code.WriteString("\tData []map[string]interface{} `json:\"data\"`\n")
	code.WriteString("\tError string `json:\"error,omitempty\"`\n")
	code.WriteString("}\n\n")

	// NewState constructor
	code.WriteString("// NewState creates a new State and loads data\n")
	code.WriteString("func NewState() (*State, error) {\n")
	code.WriteString("\ts := &State{}\n")
	code.WriteString("\tif err := s.fetchData(); err != nil {\n")
	code.WriteString("\t\ts.Error = err.Error()\n")
	code.WriteString("\t}\n")
	code.WriteString("\treturn s, nil\n")
	code.WriteString("}\n\n")

	// Close method
	code.WriteString("// Close is a no-op for file sources\n")
	code.WriteString("func (s *State) Close() error {\n")
	code.WriteString("\treturn nil\n")
	code.WriteString("}\n\n")

	// Refresh action
	code.WriteString("// Refresh reloads data from the file\n")
	code.WriteString("func (s *State) Refresh(ctx *livetemplate.Context) error {\n")
	code.WriteString("\ts.Error = \"\"\n")
	code.WriteString("\tif err := s.fetchData(); err != nil {\n")
	code.WriteString("\t\ts.Error = err.Error()\n")
	code.WriteString("\t}\n")
	code.WriteString("\treturn nil\n")
	code.WriteString("}\n\n")

	// fetchData method
	code.WriteString("// fetchData reads and parses the JSON file\n")
	code.WriteString("func (s *State) fetchData() error {\n")

	// Escape file path
	escapedFile := strings.ReplaceAll(file, `"`, `\"`)
	escapedDir := strings.ReplaceAll(siteDir, `"`, `\"`)
	code.WriteString(fmt.Sprintf("\tfilePath := \"%s\"\n", escapedFile))
	code.WriteString(fmt.Sprintf("\tsiteDir := \"%s\"\n", escapedDir))
	code.WriteString("\n")
	code.WriteString("\t// Resolve path\n")
	code.WriteString("\tif !filepath.IsAbs(filePath) {\n")
	code.WriteString("\t\tfilePath = filepath.Join(siteDir, filePath)\n")
	code.WriteString("\t}\n")
	code.WriteString("\n")
	code.WriteString("\tdata, err := os.ReadFile(filePath)\n")
	code.WriteString("\tif err != nil {\n")
	code.WriteString("\t\treturn fmt.Errorf(\"failed to read file: %w\", err)\n")
	code.WriteString("\t}\n")
	code.WriteString("\n")
	code.WriteString("\treturn s.parseJSON(data)\n")
	code.WriteString("}\n\n")

	// parseJSON method
	code.WriteString("// parseJSON handles array, object, and NDJSON\n")
	code.WriteString("func (s *State) parseJSON(data []byte) error {\n")
	code.WriteString("\tdata = []byte(strings.TrimSpace(string(data)))\n")
	code.WriteString("\tif len(data) == 0 {\n")
	code.WriteString("\t\ts.Data = []map[string]interface{}{}\n")
	code.WriteString("\t\treturn nil\n")
	code.WriteString("\t}\n")
	code.WriteString("\n")
	code.WriteString("\t// Try array first\n")
	code.WriteString("\tvar arr []map[string]interface{}\n")
	code.WriteString("\tif err := json.Unmarshal(data, &arr); err == nil {\n")
	code.WriteString("\t\ts.Data = arr\n")
	code.WriteString("\t\treturn nil\n")
	code.WriteString("\t}\n")
	code.WriteString("\n")
	code.WriteString("\t// Try single object\n")
	code.WriteString("\tvar obj map[string]interface{}\n")
	code.WriteString("\tif err := json.Unmarshal(data, &obj); err == nil {\n")
	code.WriteString("\t\ts.Data = []map[string]interface{}{obj}\n")
	code.WriteString("\t\treturn nil\n")
	code.WriteString("\t}\n")
	code.WriteString("\n")
	code.WriteString("\t// Try NDJSON\n")
	code.WriteString("\tlines := strings.Split(string(data), \"\\n\")\n")
	code.WriteString("\tvar results []map[string]interface{}\n")
	code.WriteString("\tfor _, line := range lines {\n")
	code.WriteString("\t\tline = strings.TrimSpace(line)\n")
	code.WriteString("\t\tif line == \"\" {\n")
	code.WriteString("\t\t\tcontinue\n")
	code.WriteString("\t\t}\n")
	code.WriteString("\t\tvar item map[string]interface{}\n")
	code.WriteString("\t\tif err := json.Unmarshal([]byte(line), &item); err != nil {\n")
	code.WriteString("\t\t\treturn fmt.Errorf(\"invalid JSON: %w\", err)\n")
	code.WriteString("\t\t}\n")
	code.WriteString("\t\tresults = append(results, item)\n")
	code.WriteString("\t}\n")
	code.WriteString("\tif len(results) > 0 {\n")
	code.WriteString("\t\ts.Data = results\n")
	code.WriteString("\t\treturn nil\n")
	code.WriteString("\t}\n")
	code.WriteString("\n")
	code.WriteString("\treturn fmt.Errorf(\"could not parse as JSON\")\n")
	code.WriteString("}\n")

	return code.String(), nil
}

// generateCSVFileSourceCode generates code for a CSV file source
func generateCSVFileSourceCode(sourceName, file, siteDir string, options map[string]string) (string, error) {
	if file == "" {
		return "", fmt.Errorf("csv source %q: file is required", sourceName)
	}

	hasHeader := true
	if options != nil && options["header"] == "false" {
		hasHeader = false
	}

	var code strings.Builder

	// Imports
	code.WriteString("import (\n")
	code.WriteString("\t\"encoding/csv\"\n")
	code.WriteString("\t\"fmt\"\n")
	code.WriteString("\t\"os\"\n")
	code.WriteString("\t\"path/filepath\"\n")
	code.WriteString(")\n\n")

	// State struct
	code.WriteString("// State holds data from the CSV file\n")
	code.WriteString("type State struct {\n")
	code.WriteString("\tData []map[string]interface{} `json:\"data\"`\n")
	code.WriteString("\tError string `json:\"error,omitempty\"`\n")
	code.WriteString("}\n\n")

	// NewState constructor
	code.WriteString("// NewState creates a new State and loads data\n")
	code.WriteString("func NewState() (*State, error) {\n")
	code.WriteString("\ts := &State{}\n")
	code.WriteString("\tif err := s.fetchData(); err != nil {\n")
	code.WriteString("\t\ts.Error = err.Error()\n")
	code.WriteString("\t}\n")
	code.WriteString("\treturn s, nil\n")
	code.WriteString("}\n\n")

	// Close method
	code.WriteString("// Close is a no-op for file sources\n")
	code.WriteString("func (s *State) Close() error {\n")
	code.WriteString("\treturn nil\n")
	code.WriteString("}\n\n")

	// Refresh action
	code.WriteString("// Refresh reloads data from the file\n")
	code.WriteString("func (s *State) Refresh(ctx *livetemplate.Context) error {\n")
	code.WriteString("\ts.Error = \"\"\n")
	code.WriteString("\tif err := s.fetchData(); err != nil {\n")
	code.WriteString("\t\ts.Error = err.Error()\n")
	code.WriteString("\t}\n")
	code.WriteString("\treturn nil\n")
	code.WriteString("}\n\n")

	// fetchData method
	code.WriteString("// fetchData reads and parses the CSV file\n")
	code.WriteString("func (s *State) fetchData() error {\n")

	// Escape file path
	escapedFile := strings.ReplaceAll(file, `"`, `\"`)
	escapedDir := strings.ReplaceAll(siteDir, `"`, `\"`)
	code.WriteString(fmt.Sprintf("\tfilePath := \"%s\"\n", escapedFile))
	code.WriteString(fmt.Sprintf("\tsiteDir := \"%s\"\n", escapedDir))
	code.WriteString(fmt.Sprintf("\thasHeader := %t\n", hasHeader))
	code.WriteString("\n")
	code.WriteString("\t// Resolve path\n")
	code.WriteString("\tif !filepath.IsAbs(filePath) {\n")
	code.WriteString("\t\tfilePath = filepath.Join(siteDir, filePath)\n")
	code.WriteString("\t}\n")
	code.WriteString("\n")
	code.WriteString("\tfile, err := os.Open(filePath)\n")
	code.WriteString("\tif err != nil {\n")
	code.WriteString("\t\treturn fmt.Errorf(\"failed to open file: %w\", err)\n")
	code.WriteString("\t}\n")
	code.WriteString("\tdefer file.Close()\n")
	code.WriteString("\n")
	code.WriteString("\treader := csv.NewReader(file)\n")
	code.WriteString("\trecords, err := reader.ReadAll()\n")
	code.WriteString("\tif err != nil {\n")
	code.WriteString("\t\treturn fmt.Errorf(\"failed to read CSV: %w\", err)\n")
	code.WriteString("\t}\n")
	code.WriteString("\n")
	code.WriteString("\tif len(records) == 0 {\n")
	code.WriteString("\t\ts.Data = []map[string]interface{}{}\n")
	code.WriteString("\t\treturn nil\n")
	code.WriteString("\t}\n")
	code.WriteString("\n")
	code.WriteString("\tvar headers []string\n")
	code.WriteString("\tvar dataRows [][]string\n")
	code.WriteString("\n")
	code.WriteString("\tif hasHeader {\n")
	code.WriteString("\t\theaders = records[0]\n")
	code.WriteString("\t\tdataRows = records[1:]\n")
	code.WriteString("\t} else {\n")
	code.WriteString("\t\theaders = make([]string, len(records[0]))\n")
	code.WriteString("\t\tfor i := range headers {\n")
	code.WriteString("\t\t\theaders[i] = fmt.Sprintf(\"col%d\", i+1)\n")
	code.WriteString("\t\t}\n")
	code.WriteString("\t\tdataRows = records\n")
	code.WriteString("\t}\n")
	code.WriteString("\n")
	code.WriteString("\tresults := make([]map[string]interface{}, 0, len(dataRows))\n")
	code.WriteString("\tfor _, row := range dataRows {\n")
	code.WriteString("\t\trowMap := make(map[string]interface{})\n")
	code.WriteString("\t\tfor i, value := range row {\n")
	code.WriteString("\t\t\tif i < len(headers) {\n")
	code.WriteString("\t\t\t\trowMap[headers[i]] = value\n")
	code.WriteString("\t\t\t}\n")
	code.WriteString("\t\t}\n")
	code.WriteString("\t\tresults = append(results, rowMap)\n")
	code.WriteString("\t}\n")
	code.WriteString("\n")
	code.WriteString("\ts.Data = results\n")
	code.WriteString("\treturn nil\n")
	code.WriteString("}\n")

	return code.String(), nil
}

// generateDatatableSourceCode generates code that uses the datatable component
// This creates a State with a *datatable.DataTable field and action handlers for sort/pagination
func generateDatatableSourceCode(sourceName string, sourceCfg config.SourceConfig, siteDir, columns string) (string, error) {
	var code strings.Builder

	// Imports - include datatable package
	code.WriteString("import (\n")
	code.WriteString("\t\"context\"\n")
	code.WriteString("\t\"encoding/json\"\n")
	code.WriteString("\t\"fmt\"\n")
	code.WriteString("\t\"strings\"\n")

	// Add source-specific imports
	switch sourceCfg.Type {
	case "exec":
		code.WriteString("\t\"os/exec\"\n")
	case "json":
		code.WriteString("\t\"os\"\n")
	case "csv":
		code.WriteString("\t\"encoding/csv\"\n")
		code.WriteString("\t\"os\"\n")
	case "rest":
		code.WriteString("\t\"io\"\n")
		code.WriteString("\t\"net/http\"\n")
	case "pg":
		code.WriteString("\t\"database/sql\"\n")
		code.WriteString("\t_ \"github.com/lib/pq\"\n")
	}

	code.WriteString("\n")
	code.WriteString("\t\"github.com/livetemplate/components/datatable\"\n")
	code.WriteString(")\n\n")

	// Parse columns from the lvt-columns attribute
	type col struct {
		Field string
		Label string
	}
	var cols []col
	if columns != "" {
		for _, c := range strings.Split(columns, ",") {
			parts := strings.SplitN(strings.TrimSpace(c), ":", 2)
			if len(parts) == 2 {
				cols = append(cols, col{Field: parts[0], Label: parts[1]})
			} else if len(parts) == 1 {
				// Use field name as label with title case
				field := parts[0]
				label := strings.Title(strings.ReplaceAll(field, "_", " "))
				cols = append(cols, col{Field: field, Label: label})
			}
		}
	}

	// State struct with Table field
	code.WriteString("// State holds the datatable component\n")
	code.WriteString("type State struct {\n")
	code.WriteString("\tTable *datatable.DataTable `json:\"table\"`\n")
	code.WriteString("\tError string `json:\"error,omitempty\"`\n")
	code.WriteString("}\n\n")

	// NewState constructor
	datatableID := sourceName // Use source name as datatable ID
	code.WriteString("// NewState creates a new State and fetches initial data\n")
	code.WriteString("func NewState() (*State, error) {\n")
	code.WriteString("\ts := &State{}\n")

	// Build column definitions
	code.WriteString("\t// Initialize datatable with columns\n")
	code.WriteString("\tcolumns := []datatable.Column{\n")
	if len(cols) > 0 {
		for _, c := range cols {
			fieldTitle := strings.Title(strings.ReplaceAll(c.Field, "_", ""))
			code.WriteString(fmt.Sprintf("\t\t{ID: %q, Label: %q, Sortable: true},\n", fieldTitle, c.Label))
		}
	}
	code.WriteString("\t}\n\n")

	code.WriteString(fmt.Sprintf("\ts.Table = datatable.New(%q,\n", datatableID))
	code.WriteString("\t\tdatatable.WithColumns(columns),\n")
	code.WriteString("\t\tdatatable.WithPageSize(10),\n")
	code.WriteString("\t\tdatatable.WithStriped(true),\n")
	code.WriteString("\t\tdatatable.WithHoverable(true),\n")
	code.WriteString("\t)\n\n")

	code.WriteString("\tif err := s.fetchData(); err != nil {\n")
	code.WriteString("\t\ts.Error = err.Error()\n")
	code.WriteString("\t}\n")
	code.WriteString("\treturn s, nil\n")
	code.WriteString("}\n\n")

	// Close method
	code.WriteString("// Close is a no-op for source-only state\n")
	code.WriteString("func (s *State) Close() error {\n")
	code.WriteString("\treturn nil\n")
	code.WriteString("}\n\n")

	// Refresh action
	code.WriteString("// Refresh re-fetches data from the source\n")
	code.WriteString("func (s *State) Refresh(ctx *livetemplate.Context) error {\n")
	code.WriteString("\ts.Error = \"\"\n")
	code.WriteString("\tif err := s.fetchData(); err != nil {\n")
	code.WriteString("\t\ts.Error = err.Error()\n")
	code.WriteString("\t}\n")
	code.WriteString("\treturn nil\n")
	code.WriteString("}\n\n")

	// Sort action - handles sort_<id> actions
	methodName := "Sort" + toMethodName(datatableID)
	code.WriteString(fmt.Sprintf("// %s handles sorting the datatable\n", methodName))
	code.WriteString(fmt.Sprintf("func (s *State) %s(ctx *livetemplate.Context) error {\n", methodName))
	code.WriteString("\tcolumn := ctx.GetString(\"column\")\n")
	code.WriteString("\tif column != \"\" {\n")
	code.WriteString("\t\ts.Table.Sort(column)\n")
	code.WriteString("\t}\n")
	code.WriteString("\treturn nil\n")
	code.WriteString("}\n\n")

	// Next page action
	nextPageMethod := "NextPage" + toMethodName(datatableID)
	code.WriteString(fmt.Sprintf("// %s goes to next page\n", nextPageMethod))
	code.WriteString(fmt.Sprintf("func (s *State) %s(ctx *livetemplate.Context) error {\n", nextPageMethod))
	code.WriteString("\ts.Table.NextPage()\n")
	code.WriteString("\treturn nil\n")
	code.WriteString("}\n\n")

	// Previous page action
	prevPageMethod := "PrevPage" + toMethodName(datatableID)
	code.WriteString(fmt.Sprintf("// %s goes to previous page\n", prevPageMethod))
	code.WriteString(fmt.Sprintf("func (s *State) %s(ctx *livetemplate.Context) error {\n", prevPageMethod))
	code.WriteString("\ts.Table.PreviousPage()\n")
	code.WriteString("\treturn nil\n")
	code.WriteString("}\n\n")

	// fetchData method - generates source-specific code
	code.WriteString("// fetchData fetches data from the source and populates the datatable\n")
	code.WriteString("func (s *State) fetchData() error {\n")

	// Generate fetch logic based on source type
	switch sourceCfg.Type {
	case "exec":
		escapedCmd := strings.ReplaceAll(sourceCfg.Cmd, `"`, `\"`)
		escapedDir := strings.ReplaceAll(siteDir, `"`, `\"`)
		code.WriteString(fmt.Sprintf("\tcmd := \"%s\"\n", escapedCmd))
		code.WriteString(fmt.Sprintf("\tworkDir := \"%s\"\n", escapedDir))
		code.WriteString("\n")
		code.WriteString("\tparts := strings.Fields(cmd)\n")
		code.WriteString("\tif len(parts) == 0 {\n")
		code.WriteString("\t\treturn fmt.Errorf(\"empty command\")\n")
		code.WriteString("\t}\n")
		code.WriteString("\n")
		code.WriteString("\tctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)\n")
		code.WriteString("\tdefer cancel()\n")
		code.WriteString("\n")
		code.WriteString("\texecCmd := exec.CommandContext(ctx, parts[0], parts[1:]...)\n")
		code.WriteString("\texecCmd.Dir = workDir\n")
		code.WriteString("\n")
		code.WriteString("\toutput, err := execCmd.Output()\n")
		code.WriteString("\tif err != nil {\n")
		code.WriteString("\t\treturn err\n")
		code.WriteString("\t}\n")
		code.WriteString("\n")
		code.WriteString("\treturn s.parseJSONToRows(output)\n")

	case "json":
		escapedFile := strings.ReplaceAll(sourceCfg.File, `"`, `\"`)
		escapedDir := strings.ReplaceAll(siteDir, `"`, `\"`)
		code.WriteString(fmt.Sprintf("\tfilePath := \"%s/%s\"\n", escapedDir, escapedFile))
		code.WriteString("\tdata, err := os.ReadFile(filePath)\n")
		code.WriteString("\tif err != nil {\n")
		code.WriteString("\t\treturn err\n")
		code.WriteString("\t}\n")
		code.WriteString("\treturn s.parseJSONToRows(data)\n")

	case "rest":
		escapedURL := strings.ReplaceAll(sourceCfg.URL, `"`, `\"`)
		code.WriteString(fmt.Sprintf("\turl := \"%s\"\n", escapedURL))
		code.WriteString("\tresp, err := http.Get(url)\n")
		code.WriteString("\tif err != nil {\n")
		code.WriteString("\t\treturn err\n")
		code.WriteString("\t}\n")
		code.WriteString("\tdefer resp.Body.Close()\n")
		code.WriteString("\tdata, err := io.ReadAll(resp.Body)\n")
		code.WriteString("\tif err != nil {\n")
		code.WriteString("\t\treturn err\n")
		code.WriteString("\t}\n")
		code.WriteString("\treturn s.parseJSONToRows(data)\n")

	default:
		// Fallback - just return empty
		code.WriteString("\treturn nil\n")
	}

	code.WriteString("}\n\n")

	// parseJSONToRows helper
	code.WriteString("// parseJSONToRows parses JSON data and populates datatable rows\n")
	code.WriteString("func (s *State) parseJSONToRows(data []byte) error {\n")
	code.WriteString("\tdata = []byte(strings.TrimSpace(string(data)))\n")
	code.WriteString("\tif len(data) == 0 {\n")
	code.WriteString("\t\ts.Table.Rows = []datatable.Row{}\n")
	code.WriteString("\t\treturn nil\n")
	code.WriteString("\t}\n\n")

	code.WriteString("\t// Try to parse as array of objects\n")
	code.WriteString("\tvar items []map[string]interface{}\n")
	code.WriteString("\tif err := json.Unmarshal(data, &items); err != nil {\n")
	code.WriteString("\t\t// Try single object\n")
	code.WriteString("\t\tvar item map[string]interface{}\n")
	code.WriteString("\t\tif err := json.Unmarshal(data, &item); err != nil {\n")
	code.WriteString("\t\t\treturn err\n")
	code.WriteString("\t\t}\n")
	code.WriteString("\t\titems = []map[string]interface{}{item}\n")
	code.WriteString("\t}\n\n")

	code.WriteString("\t// Convert to datatable rows\n")
	code.WriteString("\trows := make([]datatable.Row, 0, len(items))\n")
	code.WriteString("\tfor i, item := range items {\n")
	code.WriteString("\t\t// Use 'id' field if present, otherwise use index\n")
	code.WriteString("\t\trowID := fmt.Sprintf(\"%d\", i)\n")
	code.WriteString("\t\tif id, ok := item[\"id\"]; ok {\n")
	code.WriteString("\t\t\trowID = fmt.Sprintf(\"%v\", id)\n")
	code.WriteString("\t\t}\n\n")
	code.WriteString("\t\t// Convert keys to title case for datatable column matching\n")
	code.WriteString("\t\tcellData := make(map[string]interface{})\n")
	code.WriteString("\t\tfor k, v := range item {\n")
	code.WriteString("\t\t\ttitleKey := toTitleCase(k)\n")
	code.WriteString("\t\t\tcellData[titleKey] = v\n")
	code.WriteString("\t\t}\n\n")
	code.WriteString("\t\trows = append(rows, datatable.Row{\n")
	code.WriteString("\t\t\tID:   rowID,\n")
	code.WriteString("\t\t\tData: cellData,\n")
	code.WriteString("\t\t})\n")
	code.WriteString("\t}\n\n")

	code.WriteString("\ts.Table.Rows = rows\n")

	// Auto-discover columns if not specified
	code.WriteString("\n\t// Auto-discover columns if none specified\n")
	code.WriteString("\tif len(s.Table.Columns) == 0 && len(items) > 0 {\n")
	code.WriteString("\t\tfor k := range items[0] {\n")
	code.WriteString("\t\t\ttitleKey := toTitleCase(k)\n")
	code.WriteString("\t\t\ts.Table.Columns = append(s.Table.Columns, datatable.Column{\n")
	code.WriteString("\t\t\t\tID:       titleKey,\n")
	code.WriteString("\t\t\t\tLabel:    titleKey,\n")
	code.WriteString("\t\t\t\tSortable: true,\n")
	code.WriteString("\t\t\t})\n")
	code.WriteString("\t\t}\n")
	code.WriteString("\t}\n")

	code.WriteString("\treturn nil\n")
	code.WriteString("}\n\n")

	// toTitleCase helper function
	code.WriteString("// toTitleCase converts a string to title case, handling underscores\n")
	code.WriteString("func toTitleCase(s string) string {\n")
	code.WriteString("\tparts := strings.Split(s, \"_\")\n")
	code.WriteString("\tfor i, p := range parts {\n")
	code.WriteString("\t\tif len(p) > 0 {\n")
	code.WriteString("\t\t\tparts[i] = strings.ToUpper(string(p[0])) + strings.ToLower(p[1:])\n")
	code.WriteString("\t\t}\n")
	code.WriteString("\t}\n")
	code.WriteString("\treturn strings.Join(parts, \"\")\n")
	code.WriteString("}\n")

	return code.String(), nil
}

// toMethodName converts a snake_case or dash-case string to PascalCase method name
func toMethodName(s string) string {
	// Replace dashes with underscores, then process
	s = strings.ReplaceAll(s, "-", "_")
	parts := strings.Split(s, "_")
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(string(p[0])) + strings.ToLower(p[1:])
		}
	}
	return strings.Join(parts, "")
}
