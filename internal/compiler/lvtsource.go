package compiler

import (
	"fmt"
	"strings"

	"github.com/livetemplate/livemdtools/internal/config"
)

// GenerateLvtSourceCode generates Go code for an lvt-source block.
// The generated code creates a State struct that fetches data from the configured source.
// When metadata["lvt-element"] is "table", it generates datatable-aware code.
// currentFile is the absolute path to the current markdown file (for same-file markdown sources).
func GenerateLvtSourceCode(sourceName string, sourceCfg config.SourceConfig, siteDir string, currentFile string, metadata map[string]string) (string, error) {
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
	case "markdown":
		return generateMarkdownSourceCode(sourceName, sourceCfg.File, sourceCfg.Anchor, siteDir, currentFile, sourceCfg.IsReadonly())
	default:
		return "", fmt.Errorf("unsupported source type: %s", sourceCfg.Type)
	}
}

// generateExecSourceCode generates code for an exec source with argument form support
func generateExecSourceCode(sourceName, cmd, siteDir string, manual bool) (string, error) {
	if cmd == "" {
		return "", fmt.Errorf("exec source %q: cmd is required", sourceName)
	}

	// Parse the command to extract executable and arguments
	executable, args, err := ParseExecCommand(cmd)
	if err != nil {
		return "", fmt.Errorf("exec source %q: failed to parse command: %w", sourceName, err)
	}

	// Run --help introspection to get flag descriptions (fails silently)
	descriptions := IntrospectScript(executable, siteDir)
	for i := range args {
		if desc, ok := descriptions[args[i].Name]; ok {
			args[i].Description = desc
		}
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

	escapedDir := strings.ReplaceAll(siteDir, `"`, `\"`)
	escapedExec := strings.ReplaceAll(executable, `"`, `\"`)

	// ExecArg struct for template rendering
	code.WriteString("// ExecArg represents a command-line argument\n")
	code.WriteString("type ExecArg struct {\n")
	code.WriteString("\tName        string `json:\"name\"`\n")
	code.WriteString("\tLabel       string `json:\"label\"`\n")
	code.WriteString("\tType        string `json:\"type\"`\n")
	code.WriteString("\tValue       string `json:\"value\"`\n")
	code.WriteString("\tPosition    int    `json:\"position\"`\n")
	code.WriteString("\tDescription string `json:\"description\"`\n")
	code.WriteString("}\n\n")

	// State struct with Args field
	code.WriteString("// State holds data fetched from the source and execution metadata\n")
	code.WriteString("type State struct {\n")
	code.WriteString("\tData       []map[string]interface{} `json:\"data\"`\n")
	code.WriteString("\tError      string `json:\"error,omitempty\"`\n")
	code.WriteString("\tOutput     string `json:\"output,omitempty\"`\n")
	code.WriteString("\tStderr     string `json:\"stderr,omitempty\"`\n")
	code.WriteString("\tDuration   int64  `json:\"duration,omitempty\"`\n")
	code.WriteString("\tStatus     string `json:\"status\"`\n")
	code.WriteString("\tCommand    string `json:\"command\"`\n")
	code.WriteString("\tArgs       []ExecArg `json:\"args\"`\n")
	code.WriteString("\tExecutable string `json:\"executable\"`\n")
	code.WriteString("\tWorkDir    string `json:\"-\"`\n")
	code.WriteString("}\n\n")

	// NewState constructor - initialize args
	code.WriteString("// NewState creates a new State and optionally fetches initial data\n")
	code.WriteString("func NewState() (*State, error) {\n")
	code.WriteString("\ts := &State{\n")
	code.WriteString(fmt.Sprintf("\t\tExecutable: %q,\n", escapedExec))
	code.WriteString(fmt.Sprintf("\t\tWorkDir: %q,\n", escapedDir))
	code.WriteString("\t\tArgs: []ExecArg{\n")

	// Generate arg initializers
	for _, arg := range args {
		escapedName := strings.ReplaceAll(arg.Name, `"`, `\"`)
		escapedLabel := strings.ReplaceAll(arg.Label, `"`, `\"`)
		escapedDefault := strings.ReplaceAll(arg.Default, `"`, `\"`)
		escapedDesc := strings.ReplaceAll(arg.Description, `"`, `\"`)
		code.WriteString(fmt.Sprintf("\t\t\t{Name: %q, Label: %q, Type: %q, Value: %q, Position: %d, Description: %q},\n",
			escapedName, escapedLabel, arg.Type, escapedDefault, arg.Position, escapedDesc))
	}

	code.WriteString("\t\t},\n")
	code.WriteString("\t}\n")
	code.WriteString("\ts.Command = s.buildCommandString()\n")

	if manual {
		code.WriteString("\ts.Status = \"idle\"\n")
	} else {
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

	// Run action - accepts form data and executes the command
	code.WriteString("// Run executes the command with current arg values from form\n")
	code.WriteString("func (s *State) Run(ctx *livetemplate.Context) error {\n")
	code.WriteString("\t// Update arg values from form submission\n")
	code.WriteString("\tfor i := range s.Args {\n")
	code.WriteString("\t\targ := &s.Args[i]\n")
	code.WriteString("\t\tif val := ctx.GetString(arg.Name); val != \"\" {\n")
	code.WriteString("\t\t\t// HTML checkboxes send 'on' when checked - convert to 'true'\n")
	code.WriteString("\t\t\tif arg.Type == \"bool\" && val == \"on\" {\n")
	code.WriteString("\t\t\t\targ.Value = \"true\"\n")
	code.WriteString("\t\t\t} else {\n")
	code.WriteString("\t\t\t\targ.Value = val\n")
	code.WriteString("\t\t\t}\n")
	code.WriteString("\t\t} else if arg.Type == \"bool\" {\n")
	code.WriteString("\t\t\t// Unchecked checkbox won't be in form data\n")
	code.WriteString("\t\t\targ.Value = \"false\"\n")
	code.WriteString("\t\t}\n")
	code.WriteString("\t}\n")
	code.WriteString("\ts.Command = s.buildCommandString()\n")
	code.WriteString("\n")
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

	// Refresh action - alias to Run
	code.WriteString("// Refresh re-fetches data from the source (alias to Run)\n")
	code.WriteString("func (s *State) Refresh(ctx *livetemplate.Context) error {\n")
	code.WriteString("\treturn s.Run(ctx)\n")
	code.WriteString("}\n\n")

	// buildCommandString - builds display command from current args
	code.WriteString("// buildCommandString builds the command string for display\n")
	code.WriteString("func (s *State) buildCommandString() string {\n")
	code.WriteString("\tparts := []string{s.Executable}\n")
	code.WriteString("\tfor _, arg := range s.Args {\n")
	code.WriteString("\t\tif arg.Position >= 0 {\n")
	code.WriteString("\t\t\tparts = append(parts, arg.Value)\n")
	code.WriteString("\t\t} else {\n")
	code.WriteString("\t\t\tparts = append(parts, \"--\"+arg.Name, arg.Value)\n")
	code.WriteString("\t\t}\n")
	code.WriteString("\t}\n")
	code.WriteString("\treturn strings.Join(parts, \" \")\n")
	code.WriteString("}\n\n")

	// buildCommandParts - builds command parts for execution
	code.WriteString("// buildCommandParts builds the command parts for execution\n")
	code.WriteString("func (s *State) buildCommandParts() []string {\n")
	code.WriteString("\tparts := []string{s.Executable}\n")
	code.WriteString("\t// Add positional args first (in order)\n")
	code.WriteString("\tfor _, arg := range s.Args {\n")
	code.WriteString("\t\tif arg.Position >= 0 {\n")
	code.WriteString("\t\t\tparts = append(parts, arg.Value)\n")
	code.WriteString("\t\t}\n")
	code.WriteString("\t}\n")
	code.WriteString("\t// Add named args\n")
	code.WriteString("\tfor _, arg := range s.Args {\n")
	code.WriteString("\t\tif arg.Position < 0 {\n")
	code.WriteString("\t\t\tparts = append(parts, \"--\"+arg.Name, arg.Value)\n")
	code.WriteString("\t\t}\n")
	code.WriteString("\t}\n")
	code.WriteString("\treturn parts\n")
	code.WriteString("}\n\n")

	// fetchData method - uses dynamic command building
	code.WriteString("// fetchData executes the source command and parses JSON output\n")
	code.WriteString("func (s *State) fetchData() error {\n")
	code.WriteString("\tparts := s.buildCommandParts()\n")
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
	code.WriteString("\texecCmd.Dir = s.WorkDir\n")
	code.WriteString("\n")
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

	// toTitleCase helper function - simple first-letter cap
	code.WriteString("// toTitleCase converts first letter to uppercase: \"name\" -> \"Name\"\n")
	code.WriteString("func toTitleCase(s string) string {\n")
	code.WriteString("\tif len(s) == 0 {\n")
	code.WriteString("\t\treturn s\n")
	code.WriteString("\t}\n")
	code.WriteString("\treturn strings.ToUpper(s[:1]) + s[1:]\n")
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

// generateMarkdownSourceCode generates code for a markdown section source
// currentFile is the absolute path to the current markdown file (for same-file sources)
func generateMarkdownSourceCode(sourceName, file, anchor, siteDir, currentFile string, readonly bool) (string, error) {
	if anchor == "" {
		return "", fmt.Errorf("markdown source %q: anchor is required", sourceName)
	}

	// For same-file sources, use currentFile as the file path
	if file == "" {
		file = currentFile
	}

	var code strings.Builder

	// Package declaration (required by generatePluginCode parser)
	code.WriteString("package state\n\n")

	// Imports (only what's actually used in the generated code)
	code.WriteString("import (\n")
	code.WriteString("\t\"crypto/rand\"\n")
	code.WriteString("\t\"encoding/hex\"\n")
	code.WriteString("\t\"fmt\"\n")
	code.WriteString("\t\"hash/fnv\"\n")
	code.WriteString("\t\"os\"\n")
	code.WriteString("\t\"path/filepath\"\n")
	code.WriteString("\t\"regexp\"\n")
	code.WriteString("\t\"strings\"\n")
	code.WriteString("\n")
	code.WriteString("\t\"github.com/livetemplate/livetemplate\"\n")
	code.WriteString(")\n\n")

	// State struct
	code.WriteString("// State holds data from the markdown section\n")
	code.WriteString("type State struct {\n")
	code.WriteString("\tData []map[string]interface{} `json:\"data\"`\n")
	code.WriteString("\tError string `json:\"error,omitempty\"`\n")
	code.WriteString("\tfilePath string\n")
	code.WriteString("\tanchor string\n")
	code.WriteString("\treadonly bool\n")
	code.WriteString("}\n\n")

	// Escape strings for embedding
	escapedFile := strings.ReplaceAll(file, `"`, `\"`)
	escapedAnchor := strings.ReplaceAll(anchor, `"`, `\"`)
	escapedDir := strings.ReplaceAll(siteDir, `"`, `\"`)

	// NewState constructor
	code.WriteString("// NewState creates a new State and loads data\n")
	code.WriteString("func NewState() (*State, error) {\n")
	code.WriteString("\ts := &State{\n")
	code.WriteString(fmt.Sprintf("\t\tfilePath: %q,\n", escapedFile))
	code.WriteString(fmt.Sprintf("\t\tanchor: %q,\n", escapedAnchor))
	code.WriteString(fmt.Sprintf("\t\treadonly: %t,\n", readonly))
	code.WriteString("\t}\n")
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
	code.WriteString("// Refresh reloads data from the markdown section\n")
	code.WriteString("func (s *State) Refresh(ctx *livetemplate.Context) error {\n")
	code.WriteString("\ts.Error = \"\"\n")
	code.WriteString("\tif err := s.fetchData(); err != nil {\n")
	code.WriteString("\t\ts.Error = err.Error()\n")
	code.WriteString("\t}\n")
	code.WriteString("\treturn nil\n")
	code.WriteString("}\n\n")

	// Write action handlers (only if not readonly)
	if !readonly {
		// Add action
		code.WriteString("// Add adds a new item to the markdown section\n")
		code.WriteString("func (s *State) Add(ctx *livetemplate.Context) error {\n")
		code.WriteString("\tif s.readonly {\n")
		code.WriteString("\t\treturn fmt.Errorf(\"source is read-only\")\n")
		code.WriteString("\t}\n")
		code.WriteString("\tdata := extractActionData(ctx)\n")
		code.WriteString("\tif err := s.writeItem(\"add\", data); err != nil {\n")
		code.WriteString("\t\ts.Error = err.Error()\n")
		code.WriteString("\t\treturn nil\n")
		code.WriteString("\t}\n")
		code.WriteString("\treturn s.Refresh(ctx)\n")
		code.WriteString("}\n\n")

		// Toggle action
		code.WriteString("// Toggle toggles the done state of a task item\n")
		code.WriteString("func (s *State) Toggle(ctx *livetemplate.Context) error {\n")
		code.WriteString("\tif s.readonly {\n")
		code.WriteString("\t\treturn fmt.Errorf(\"source is read-only\")\n")
		code.WriteString("\t}\n")
		code.WriteString("\tdata := extractActionData(ctx)\n")
		code.WriteString("\tif err := s.writeItem(\"toggle\", data); err != nil {\n")
		code.WriteString("\t\ts.Error = err.Error()\n")
		code.WriteString("\t\treturn nil\n")
		code.WriteString("\t}\n")
		code.WriteString("\treturn s.Refresh(ctx)\n")
		code.WriteString("}\n\n")

		// Delete action
		code.WriteString("// Delete removes an item from the markdown section\n")
		code.WriteString("func (s *State) Delete(ctx *livetemplate.Context) error {\n")
		code.WriteString("\tif s.readonly {\n")
		code.WriteString("\t\treturn fmt.Errorf(\"source is read-only\")\n")
		code.WriteString("\t}\n")
		code.WriteString("\tdata := extractActionData(ctx)\n")
		code.WriteString("\tif err := s.writeItem(\"delete\", data); err != nil {\n")
		code.WriteString("\t\ts.Error = err.Error()\n")
		code.WriteString("\t\treturn nil\n")
		code.WriteString("\t}\n")
		code.WriteString("\treturn s.Refresh(ctx)\n")
		code.WriteString("}\n\n")

		// Update action
		code.WriteString("// Update updates an existing item in the markdown section\n")
		code.WriteString("func (s *State) Update(ctx *livetemplate.Context) error {\n")
		code.WriteString("\tif s.readonly {\n")
		code.WriteString("\t\treturn fmt.Errorf(\"source is read-only\")\n")
		code.WriteString("\t}\n")
		code.WriteString("\tdata := extractActionData(ctx)\n")
		code.WriteString("\tif err := s.writeItem(\"update\", data); err != nil {\n")
		code.WriteString("\t\ts.Error = err.Error()\n")
		code.WriteString("\t\treturn nil\n")
		code.WriteString("\t}\n")
		code.WriteString("\treturn s.Refresh(ctx)\n")
		code.WriteString("}\n\n")

		// extractActionData helper - builds map from context
		code.WriteString("// extractActionData builds a data map from the context\n")
		code.WriteString("func extractActionData(ctx *livetemplate.Context) map[string]interface{} {\n")
		code.WriteString("\tdata := make(map[string]interface{})\n")
		code.WriteString("\t// Extract common fields used in markdown CRUD operations\n")
		code.WriteString("\tif id := ctx.GetString(\"id\"); id != \"\" {\n")
		code.WriteString("\t\tdata[\"id\"] = id\n")
		code.WriteString("\t}\n")
		code.WriteString("\tif text := ctx.GetString(\"text\"); text != \"\" {\n")
		code.WriteString("\t\tdata[\"text\"] = text\n")
		code.WriteString("\t}\n")
		code.WriteString("\tif ctx.Has(\"done\") {\n")
		code.WriteString("\t\tdata[\"done\"] = ctx.GetBool(\"done\")\n")
		code.WriteString("\t}\n")
		code.WriteString("\treturn data\n")
		code.WriteString("}\n\n")

		// writeItem method - core write logic
		code.WriteString("// writeItem performs write operations on the markdown file\n")
		code.WriteString("func (s *State) writeItem(action string, data map[string]interface{}) error {\n")
		code.WriteString("\tpath := s.resolvePath()\n")
		code.WriteString("\tif path == \"\" {\n")
		code.WriteString("\t\treturn fmt.Errorf(\"file path not set\")\n")
		code.WriteString("\t}\n")
		code.WriteString("\n")
		code.WriteString("\t// Read current file content\n")
		code.WriteString("\tcontent, err := os.ReadFile(path)\n")
		code.WriteString("\tif err != nil {\n")
		code.WriteString("\t\treturn fmt.Errorf(\"failed to read file: %w\", err)\n")
		code.WriteString("\t}\n")
		code.WriteString("\n")
		code.WriteString("\t// Find section boundaries\n")
		code.WriteString("\tanchorName := strings.TrimPrefix(s.anchor, \"#\")\n")
		code.WriteString("\tcontentStr := string(content)\n")
		code.WriteString("\n")
		code.WriteString("\tmatches := s.findSectionHeader(contentStr, anchorName)\n")
		code.WriteString("\tif matches == nil {\n")
		code.WriteString("\t\treturn fmt.Errorf(\"section with anchor %q not found\", s.anchor)\n")
		code.WriteString("\t}\n")
		code.WriteString("\n")
		code.WriteString("\tsectionStart := matches[1]\n")
		code.WriteString("\theaderLevel := len(contentStr[matches[2]:matches[3]])\n")
		code.WriteString("\n")
		code.WriteString("\t// Find the end of the section\n")
		code.WriteString("\tsectionEnd := len(contentStr)\n")
		code.WriteString("\tnextHeaderPattern := regexp.MustCompile(`(?m)^#{1,` + fmt.Sprintf(\"%d\", headerLevel) + `}\\s+`)\n")
		code.WriteString("\tif loc := nextHeaderPattern.FindStringIndex(contentStr[sectionStart:]); loc != nil {\n")
		code.WriteString("\t\tsectionEnd = sectionStart + loc[0]\n")
		code.WriteString("\t}\n")
		code.WriteString("\n")
		code.WriteString("\tsectionContent := contentStr[sectionStart:sectionEnd]\n")
		code.WriteString("\n")
		code.WriteString("\t// Detect format\n")
		code.WriteString("\tformat := detectFormat(sectionContent)\n")
		code.WriteString("\n")
		code.WriteString("\t// Perform the action\n")
		code.WriteString("\tvar newSection string\n")
		code.WriteString("\tswitch action {\n")
		code.WriteString("\tcase \"add\":\n")
		code.WriteString("\t\tnewSection, err = addItem(sectionContent, format, data)\n")
		code.WriteString("\tcase \"toggle\":\n")
		code.WriteString("\t\tnewSection, err = toggleItem(sectionContent, format, data)\n")
		code.WriteString("\tcase \"delete\":\n")
		code.WriteString("\t\tnewSection, err = deleteItem(sectionContent, format, data)\n")
		code.WriteString("\tcase \"update\":\n")
		code.WriteString("\t\tnewSection, err = updateItem(sectionContent, format, data)\n")
		code.WriteString("\tdefault:\n")
		code.WriteString("\t\treturn fmt.Errorf(\"unknown action: %s\", action)\n")
		code.WriteString("\t}\n")
		code.WriteString("\n")
		code.WriteString("\tif err != nil {\n")
		code.WriteString("\t\treturn err\n")
		code.WriteString("\t}\n")
		code.WriteString("\n")
		code.WriteString("\t// Reconstruct the file\n")
		code.WriteString("\tnewContent := contentStr[:sectionStart] + newSection + contentStr[sectionEnd:]\n")
		code.WriteString("\n")
		code.WriteString("\t// Write back to file\n")
		code.WriteString("\tif err := os.WriteFile(path, []byte(newContent), 0644); err != nil {\n")
		code.WriteString("\t\treturn fmt.Errorf(\"failed to write file: %w\", err)\n")
		code.WriteString("\t}\n")
		code.WriteString("\n")
		code.WriteString("\treturn nil\n")
		code.WriteString("}\n\n")

		// detectFormat helper
		code.WriteString("// detectFormat determines the list format in the section\n")
		code.WriteString("func detectFormat(content string) string {\n")
		code.WriteString("\ttaskPattern := regexp.MustCompile(`(?m)^\\s*-\\s+\\[([ xX])\\]\\s+`)\n")
		code.WriteString("\ttablePattern := regexp.MustCompile(`(?m)^\\s*\\|.+\\|`)\n")
		code.WriteString("\tbulletPattern := regexp.MustCompile(`(?m)^\\s*-\\s+[^\\[]`)\n")
		code.WriteString("\n")
		code.WriteString("\tif taskPattern.MatchString(content) {\n")
		code.WriteString("\t\treturn \"task\"\n")
		code.WriteString("\t}\n")
		code.WriteString("\tif tablePattern.MatchString(content) {\n")
		code.WriteString("\t\treturn \"table\"\n")
		code.WriteString("\t}\n")
		code.WriteString("\tif bulletPattern.MatchString(content) {\n")
		code.WriteString("\t\treturn \"bullet\"\n")
		code.WriteString("\t}\n")
		code.WriteString("\treturn \"task\" // default\n")
		code.WriteString("}\n\n")

		// generateContentID helper
		code.WriteString("// generateContentID generates a deterministic 8-char hex ID from content using FNV-1a\n")
		code.WriteString("func generateContentID(text string) string {\n")
		code.WriteString("\th := fnv.New32a()\n")
		code.WriteString("\th.Write([]byte(text))\n")
		code.WriteString("\treturn fmt.Sprintf(\"%08x\", h.Sum32())\n")
		code.WriteString("}\n\n")

		// extractItemID helper
		code.WriteString("// extractItemID extracts the ID from a line, using content-based ID if no explicit ID\n")
		code.WriteString("func extractItemID(line, format string) string {\n")
		code.WriteString("\tswitch format {\n")
		code.WriteString("\tcase \"task\":\n")
		code.WriteString("\t\ttaskPattern := regexp.MustCompile(`^\\s*-\\s+\\[([ xX])\\]\\s+(.+?)(?:\\s*<!--\\s*id:(\\w+)\\s*-->)?$`)\n")
		code.WriteString("\t\tif matches := taskPattern.FindStringSubmatch(line); matches != nil {\n")
		code.WriteString("\t\t\tif matches[3] != \"\" {\n")
		code.WriteString("\t\t\t\treturn matches[3] // Explicit ID\n")
		code.WriteString("\t\t\t}\n")
		code.WriteString("\t\t\treturn generateContentID(strings.TrimSpace(matches[2])) // Content-based ID\n")
		code.WriteString("\t\t}\n")
		code.WriteString("\tcase \"bullet\":\n")
		code.WriteString("\t\tbulletPattern := regexp.MustCompile(`^\\s*-\\s+(.+?)(?:\\s*<!--\\s*id:(\\w+)\\s*-->)?$`)\n")
		code.WriteString("\t\tif matches := bulletPattern.FindStringSubmatch(line); matches != nil {\n")
		code.WriteString("\t\t\ttext := strings.TrimSpace(matches[1])\n")
		code.WriteString("\t\t\t// Skip task list items\n")
		code.WriteString("\t\t\tif strings.HasPrefix(text, \"[ ]\") || strings.HasPrefix(text, \"[x]\") || strings.HasPrefix(text, \"[X]\") {\n")
		code.WriteString("\t\t\t\treturn \"\"\n")
		code.WriteString("\t\t\t}\n")
		code.WriteString("\t\t\tif matches[2] != \"\" {\n")
		code.WriteString("\t\t\t\treturn matches[2] // Explicit ID\n")
		code.WriteString("\t\t\t}\n")
		code.WriteString("\t\t\treturn generateContentID(text) // Content-based ID\n")
		code.WriteString("\t\t}\n")
		code.WriteString("\tcase \"table\":\n")
		code.WriteString("\t\ttableRowPattern := regexp.MustCompile(`^\\s*\\|(.+)\\|(?:\\s*<!--\\s*id:(\\w+)\\s*-->)?`)\n")
		code.WriteString("\t\tif matches := tableRowPattern.FindStringSubmatch(line); matches != nil {\n")
		code.WriteString("\t\t\tif matches[2] != \"\" {\n")
		code.WriteString("\t\t\t\treturn matches[2] // Explicit ID\n")
		code.WriteString("\t\t\t}\n")
		code.WriteString("\t\t\tcells := parseTableCells(matches[1])\n")
		code.WriteString("\t\t\treturn generateContentID(strings.Join(cells, \"|\")) // Content-based ID\n")
		code.WriteString("\t\t}\n")
		code.WriteString("\t}\n")
		code.WriteString("\treturn \"\"\n")
		code.WriteString("}\n\n")

		// addItem helper
		code.WriteString("// addItem adds a new item to the section\n")
		code.WriteString("func addItem(sectionContent, format string, data map[string]interface{}) (string, error) {\n")
		code.WriteString("\ttext, _ := data[\"text\"].(string)\n")
		code.WriteString("\tif text == \"\" {\n")
		code.WriteString("\t\treturn \"\", fmt.Errorf(\"text is required for add\")\n")
		code.WriteString("\t}\n")
		code.WriteString("\n")
		code.WriteString("\tnewID := generateID()\n")
		code.WriteString("\tvar newLine string\n")
		code.WriteString("\n")
		code.WriteString("\tswitch format {\n")
		code.WriteString("\tcase \"task\":\n")
		code.WriteString("\t\tnewLine = fmt.Sprintf(\"- [ ] %s <!-- id:%s -->\", text, newID)\n")
		code.WriteString("\tcase \"bullet\":\n")
		code.WriteString("\t\tnewLine = fmt.Sprintf(\"- %s <!-- id:%s -->\", text, newID)\n")
		code.WriteString("\tcase \"table\":\n")
		code.WriteString("\t\t// For tables, we need to build a row from the data\n")
		code.WriteString("\t\theaders := extractTableHeaders(sectionContent)\n")
		code.WriteString("\t\tvar cells []string\n")
		code.WriteString("\t\tfor _, h := range headers {\n")
		code.WriteString("\t\t\tif val, ok := data[h]; ok {\n")
		code.WriteString("\t\t\t\tcells = append(cells, fmt.Sprintf(\"%v\", val))\n")
		code.WriteString("\t\t\t} else {\n")
		code.WriteString("\t\t\t\tcells = append(cells, \"\")\n")
		code.WriteString("\t\t\t}\n")
		code.WriteString("\t\t}\n")
		code.WriteString("\t\tnewLine = fmt.Sprintf(\"| %s | <!-- id:%s -->\", strings.Join(cells, \" | \"), newID)\n")
		code.WriteString("\tdefault:\n")
		code.WriteString("\t\treturn \"\", fmt.Errorf(\"unsupported format: %s\", format)\n")
		code.WriteString("\t}\n")
		code.WriteString("\n")
		code.WriteString("\t// Append to section, ensuring a newline separator\n")
		code.WriteString("\tif !strings.HasSuffix(sectionContent, \"\\n\") {\n")
		code.WriteString("\t\tsectionContent += \"\\n\"\n")
		code.WriteString("\t}\n")
		code.WriteString("\treturn sectionContent + newLine + \"\\n\", nil\n")
		code.WriteString("}\n\n")

		// toggleItem helper
		code.WriteString("// toggleItem toggles the done state of a task\n")
		code.WriteString("func toggleItem(sectionContent, format string, data map[string]interface{}) (string, error) {\n")
		code.WriteString("\tif format != \"task\" {\n")
		code.WriteString("\t\treturn \"\", fmt.Errorf(\"toggle is only supported for task lists\")\n")
		code.WriteString("\t}\n")
		code.WriteString("\n")
		code.WriteString("\tid, _ := data[\"id\"].(string)\n")
		code.WriteString("\tif id == \"\" {\n")
		code.WriteString("\t\treturn \"\", fmt.Errorf(\"id is required for toggle\")\n")
		code.WriteString("\t}\n")
		code.WriteString("\n")
		code.WriteString("\t// First try to find by explicit ID comment\n")
		code.WriteString("\texplicitPattern := regexp.MustCompile(`(?m)^(\\s*-\\s+\\[)([ xX])(\\]\\s+.+?)\\s*<!--\\s*id:` + regexp.QuoteMeta(id) + `\\s*-->`)\n")
		code.WriteString("\tif explicitPattern.MatchString(sectionContent) {\n")
		code.WriteString("\t\tnewContent := explicitPattern.ReplaceAllStringFunc(sectionContent, func(match string) string {\n")
		code.WriteString("\t\t\tsubmatches := explicitPattern.FindStringSubmatch(match)\n")
		code.WriteString("\t\t\tcheckbox := submatches[2]\n")
		code.WriteString("\t\t\tvar newCheckbox string\n")
		code.WriteString("\t\t\tif checkbox == \" \" {\n")
		code.WriteString("\t\t\t\tnewCheckbox = \"x\"\n")
		code.WriteString("\t\t\t} else {\n")
		code.WriteString("\t\t\t\tnewCheckbox = \" \"\n")
		code.WriteString("\t\t\t}\n")
		code.WriteString("\t\t\treturn submatches[1] + newCheckbox + submatches[3] + \" <!-- id:\" + id + \" -->\"\n")
		code.WriteString("\t\t})\n")
		code.WriteString("\t\treturn newContent, nil\n")
		code.WriteString("\t}\n")
		code.WriteString("\n")
		code.WriteString("\t// Fall back to content-based ID matching\n")
		code.WriteString("\tlines := strings.Split(sectionContent, \"\\n\")\n")
		code.WriteString("\tfound := false\n")
		code.WriteString("\tfor i, line := range lines {\n")
		code.WriteString("\t\tlineID := extractItemID(line, format)\n")
		code.WriteString("\t\tif lineID == id {\n")
		code.WriteString("\t\t\tfound = true\n")
		code.WriteString("\t\t\ttaskPattern := regexp.MustCompile(`^(\\s*-\\s+\\[)([ xX])(\\]\\s+.+)$`)\n")
		code.WriteString("\t\t\tif submatches := taskPattern.FindStringSubmatch(line); submatches != nil {\n")
		code.WriteString("\t\t\t\tcheckbox := submatches[2]\n")
		code.WriteString("\t\t\t\tvar newCheckbox string\n")
		code.WriteString("\t\t\t\tif checkbox == \" \" {\n")
		code.WriteString("\t\t\t\t\tnewCheckbox = \"x\"\n")
		code.WriteString("\t\t\t\t} else {\n")
		code.WriteString("\t\t\t\t\tnewCheckbox = \" \"\n")
		code.WriteString("\t\t\t\t}\n")
		code.WriteString("\t\t\t\tlines[i] = submatches[1] + newCheckbox + submatches[3]\n")
		code.WriteString("\t\t\t}\n")
		code.WriteString("\t\t\tbreak\n")
		code.WriteString("\t\t}\n")
		code.WriteString("\t}\n")
		code.WriteString("\n")
		code.WriteString("\tif !found {\n")
		code.WriteString("\t\treturn \"\", fmt.Errorf(\"item with id %q not found\", id)\n")
		code.WriteString("\t}\n")
		code.WriteString("\n")
		code.WriteString("\treturn strings.Join(lines, \"\\n\"), nil\n")
		code.WriteString("}\n\n")

		// deleteItem helper
		code.WriteString("// deleteItem removes an item from the section\n")
		code.WriteString("func deleteItem(sectionContent, format string, data map[string]interface{}) (string, error) {\n")
		code.WriteString("\tid, _ := data[\"id\"].(string)\n")
		code.WriteString("\tif id == \"\" {\n")
		code.WriteString("\t\treturn \"\", fmt.Errorf(\"id is required for delete\")\n")
		code.WriteString("\t}\n")
		code.WriteString("\n")
		code.WriteString("\t// First try to find by explicit ID comment\n")
		code.WriteString("\tlinePattern := regexp.MustCompile(`(?m)^.*<!--\\s*id:` + regexp.QuoteMeta(id) + `\\s*-->.*\\n?`)\n")
		code.WriteString("\tif linePattern.MatchString(sectionContent) {\n")
		code.WriteString("\t\treturn linePattern.ReplaceAllString(sectionContent, \"\"), nil\n")
		code.WriteString("\t}\n")
		code.WriteString("\n")
		code.WriteString("\t// Fall back to content-based ID matching\n")
		code.WriteString("\tlines := strings.Split(sectionContent, \"\\n\")\n")
		code.WriteString("\tfound := false\n")
		code.WriteString("\tvar result []string\n")
		code.WriteString("\tfor _, line := range lines {\n")
		code.WriteString("\t\tlineID := extractItemID(line, format)\n")
		code.WriteString("\t\tif lineID == id {\n")
		code.WriteString("\t\t\tfound = true\n")
		code.WriteString("\t\t\tcontinue // Skip this line (delete it)\n")
		code.WriteString("\t\t}\n")
		code.WriteString("\t\tresult = append(result, line)\n")
		code.WriteString("\t}\n")
		code.WriteString("\n")
		code.WriteString("\tif !found {\n")
		code.WriteString("\t\treturn \"\", fmt.Errorf(\"item with id %q not found\", id)\n")
		code.WriteString("\t}\n")
		code.WriteString("\n")
		code.WriteString("\treturn strings.Join(result, \"\\n\"), nil\n")
		code.WriteString("}\n\n")

		// updateItem helper
		code.WriteString("// updateItem updates an existing item's text\n")
		code.WriteString("func updateItem(sectionContent, format string, data map[string]interface{}) (string, error) {\n")
		code.WriteString("\tid, _ := data[\"id\"].(string)\n")
		code.WriteString("\ttext, _ := data[\"text\"].(string)\n")
		code.WriteString("\tif id == \"\" {\n")
		code.WriteString("\t\treturn \"\", fmt.Errorf(\"id is required for update\")\n")
		code.WriteString("\t}\n")
		code.WriteString("\tif text == \"\" {\n")
		code.WriteString("\t\treturn \"\", fmt.Errorf(\"text is required for update\")\n")
		code.WriteString("\t}\n")
		code.WriteString("\n")
		code.WriteString("\t// First try to find by explicit ID comment\n")
		code.WriteString("\tswitch format {\n")
		code.WriteString("\tcase \"task\":\n")
		code.WriteString("\t\ttaskPattern := regexp.MustCompile(`(?m)^(\\s*-\\s+\\[[ xX]\\]\\s+).+?(\\s*<!--\\s*id:` + regexp.QuoteMeta(id) + `\\s*-->)`)\n")
		code.WriteString("\t\tif taskPattern.MatchString(sectionContent) {\n")
		code.WriteString("\t\t\treturn taskPattern.ReplaceAllString(sectionContent, \"${1}\"+text+\"${2}\"), nil\n")
		code.WriteString("\t\t}\n")
		code.WriteString("\tcase \"bullet\":\n")
		code.WriteString("\t\tbulletPattern := regexp.MustCompile(`(?m)^(\\s*-\\s+).+?(\\s*<!--\\s*id:` + regexp.QuoteMeta(id) + `\\s*-->)`)\n")
		code.WriteString("\t\tif bulletPattern.MatchString(sectionContent) {\n")
		code.WriteString("\t\t\treturn bulletPattern.ReplaceAllString(sectionContent, \"${1}\"+text+\"${2}\"), nil\n")
		code.WriteString("\t\t}\n")
		code.WriteString("\tdefault:\n")
		code.WriteString("\t\treturn \"\", fmt.Errorf(\"update not supported for format: %s\", format)\n")
		code.WriteString("\t}\n")
		code.WriteString("\n")
		code.WriteString("\t// Fall back to content-based ID matching\n")
		code.WriteString("\tlines := strings.Split(sectionContent, \"\\n\")\n")
		code.WriteString("\tfound := false\n")
		code.WriteString("\tfor i, line := range lines {\n")
		code.WriteString("\t\tlineID := extractItemID(line, format)\n")
		code.WriteString("\t\tif lineID == id {\n")
		code.WriteString("\t\t\tfound = true\n")
		code.WriteString("\t\t\tswitch format {\n")
		code.WriteString("\t\t\tcase \"task\":\n")
		code.WriteString("\t\t\t\ttaskPattern := regexp.MustCompile(`^(\\s*-\\s+\\[[ xX]\\]\\s+).+$`)\n")
		code.WriteString("\t\t\t\tif submatches := taskPattern.FindStringSubmatch(line); submatches != nil {\n")
		code.WriteString("\t\t\t\t\tlines[i] = submatches[1] + text\n")
		code.WriteString("\t\t\t\t}\n")
		code.WriteString("\t\t\tcase \"bullet\":\n")
		code.WriteString("\t\t\t\tbulletPattern := regexp.MustCompile(`^(\\s*-\\s+).+$`)\n")
		code.WriteString("\t\t\t\tif submatches := bulletPattern.FindStringSubmatch(line); submatches != nil {\n")
		code.WriteString("\t\t\t\t\tlines[i] = submatches[1] + text\n")
		code.WriteString("\t\t\t\t}\n")
		code.WriteString("\t\t\t}\n")
		code.WriteString("\t\t\tbreak\n")
		code.WriteString("\t\t}\n")
		code.WriteString("\t}\n")
		code.WriteString("\n")
		code.WriteString("\tif !found {\n")
		code.WriteString("\t\treturn \"\", fmt.Errorf(\"item with id %q not found\", id)\n")
		code.WriteString("\t}\n")
		code.WriteString("\n")
		code.WriteString("\treturn strings.Join(lines, \"\\n\"), nil\n")
		code.WriteString("}\n\n")

		// extractTableHeaders helper
		code.WriteString("// extractTableHeaders extracts column headers from a markdown table\n")
		code.WriteString("func extractTableHeaders(content string) []string {\n")
		code.WriteString("\tlines := strings.Split(content, \"\\n\")\n")
		code.WriteString("\ttablePattern := regexp.MustCompile(`^\\s*\\|(.+)\\|`)\n")
		code.WriteString("\n")
		code.WriteString("\tfor _, line := range lines {\n")
		code.WriteString("\t\tif matches := tablePattern.FindStringSubmatch(line); matches != nil {\n")
		code.WriteString("\t\t\treturn parseTableCells(matches[1])\n")
		code.WriteString("\t\t}\n")
		code.WriteString("\t}\n")
		code.WriteString("\treturn nil\n")
		code.WriteString("}\n\n")
	}

	// resolvePath helper
	code.WriteString("// resolvePath determines which file to read\n")
	code.WriteString("func (s *State) resolvePath() string {\n")
	code.WriteString(fmt.Sprintf("\tsiteDir := %q\n", escapedDir))
	code.WriteString("\tif s.filePath == \"\" {\n")
	code.WriteString("\t\t// Same file - this should be set by caller\n")
	code.WriteString("\t\treturn \"\"\n")
	code.WriteString("\t}\n")
	code.WriteString("\tif filepath.IsAbs(s.filePath) {\n")
	code.WriteString("\t\treturn s.filePath\n")
	code.WriteString("\t}\n")
	code.WriteString("\treturn filepath.Join(siteDir, s.filePath)\n")
	code.WriteString("}\n\n")

	// fetchData method
	code.WriteString("// fetchData reads and parses the markdown section\n")
	code.WriteString("func (s *State) fetchData() error {\n")
	code.WriteString("\tpath := s.resolvePath()\n")
	code.WriteString("\tif path == \"\" {\n")
	code.WriteString("\t\treturn fmt.Errorf(\"file path not set\")\n")
	code.WriteString("\t}\n")
	code.WriteString("\n")
	code.WriteString("\tcontent, err := os.ReadFile(path)\n")
	code.WriteString("\tif err != nil {\n")
	code.WriteString("\t\treturn fmt.Errorf(\"failed to read file: %w\", err)\n")
	code.WriteString("\t}\n")
	code.WriteString("\n")
	code.WriteString("\treturn s.parseSection(string(content))\n")
	code.WriteString("}\n\n")

	// parseSection method
	code.WriteString("// parseSection finds and parses the data section by anchor\n")
	code.WriteString("func (s *State) parseSection(content string) error {\n")
	code.WriteString("\tanchorName := strings.TrimPrefix(s.anchor, \"#\")\n")
	code.WriteString("\n")
	code.WriteString("\t// Find section header - try explicit {#anchor} first, then text-based\n")
	code.WriteString("\tmatches := s.findSectionHeader(content, anchorName)\n")
	code.WriteString("\tif matches == nil {\n")
	code.WriteString("\t\ts.Data = []map[string]interface{}{}\n")
	code.WriteString("\t\treturn nil\n")
	code.WriteString("\t}\n")
	code.WriteString("\n")
	code.WriteString("\tsectionStart := matches[1]\n")
	code.WriteString("\theaderLevel := len(content[matches[2]:matches[3]])\n")
	code.WriteString("\n")
	code.WriteString("\tsectionEnd := len(content)\n")
	code.WriteString("\tnextHeaderPattern := regexp.MustCompile(`(?m)^#{1,` + fmt.Sprintf(\"%d\", headerLevel) + `}\\s+`)\n")
	code.WriteString("\tif loc := nextHeaderPattern.FindStringIndex(content[sectionStart:]); loc != nil {\n")
	code.WriteString("\t\tsectionEnd = sectionStart + loc[0]\n")
	code.WriteString("\t}\n")
	code.WriteString("\n")
	code.WriteString("\tsectionContent := content[sectionStart:sectionEnd]\n")
	code.WriteString("\treturn s.detectAndParse(sectionContent)\n")
	code.WriteString("}\n\n")

	// detectAndParse method
	code.WriteString("// detectAndParse auto-detects the format and parses accordingly\n")
	code.WriteString("func (s *State) detectAndParse(content string) error {\n")
	code.WriteString("\tlines := strings.Split(content, \"\\n\")\n")
	code.WriteString("\n")
	code.WriteString("\ttaskListPattern := regexp.MustCompile(`^\\s*-\\s+\\[([ xX])\\]\\s+`)\n")
	code.WriteString("\tbulletListPattern := regexp.MustCompile(`^\\s*-\\s+[^\\[]`)\n")
	code.WriteString("\ttablePattern := regexp.MustCompile(`^\\s*\\|.+\\|`)\n")
	code.WriteString("\n")
	code.WriteString("\tfor _, line := range lines {\n")
	code.WriteString("\t\tline = strings.TrimSpace(line)\n")
	code.WriteString("\t\tif line == \"\" {\n")
	code.WriteString("\t\t\tcontinue\n")
	code.WriteString("\t\t}\n")
	code.WriteString("\n")
	code.WriteString("\t\tif taskListPattern.MatchString(line) {\n")
	code.WriteString("\t\t\treturn s.parseTaskList(lines)\n")
	code.WriteString("\t\t}\n")
	code.WriteString("\t\tif tablePattern.MatchString(line) {\n")
	code.WriteString("\t\t\treturn s.parseTable(lines)\n")
	code.WriteString("\t\t}\n")
	code.WriteString("\t\tif bulletListPattern.MatchString(line) {\n")
	code.WriteString("\t\t\treturn s.parseBulletList(lines)\n")
	code.WriteString("\t\t}\n")
	code.WriteString("\t}\n")
	code.WriteString("\n")
	code.WriteString("\ts.Data = []map[string]interface{}{}\n")
	code.WriteString("\treturn nil\n")
	code.WriteString("}\n\n")

	// parseTaskList method
	code.WriteString("// parseTaskList parses - [ ] item <!-- id:xxx --> format\n")
	code.WriteString("func (s *State) parseTaskList(lines []string) error {\n")
	code.WriteString("\tvar results []map[string]interface{}\n")
	code.WriteString("\ttaskPattern := regexp.MustCompile(`^\\s*-\\s+\\[([ xX])\\]\\s+(.+?)(?:\\s*<!--\\s*id:(\\w+)\\s*-->)?$`)\n")
	code.WriteString("\n")
	code.WriteString("\tfor _, line := range lines {\n")
	code.WriteString("\t\tmatches := taskPattern.FindStringSubmatch(line)\n")
	code.WriteString("\t\tif matches == nil {\n")
	code.WriteString("\t\t\tcontinue\n")
	code.WriteString("\t\t}\n")
	code.WriteString("\n")
	code.WriteString("\t\tdone := matches[1] == \"x\" || matches[1] == \"X\"\n")
	code.WriteString("\t\ttext := strings.TrimSpace(matches[2])\n")
	code.WriteString("\t\tid := matches[3]\n")
	code.WriteString("\t\tif id == \"\" {\n")
	code.WriteString("\t\t\tid = generateContentID(text)\n")
	code.WriteString("\t\t}\n")
	code.WriteString("\n")
	code.WriteString("\t\tresults = append(results, map[string]interface{}{\n")
	code.WriteString("\t\t\t\"id\":   id,\n")
	code.WriteString("\t\t\t\"text\": text,\n")
	code.WriteString("\t\t\t\"done\": done,\n")
	code.WriteString("\t\t})\n")
	code.WriteString("\t}\n")
	code.WriteString("\n")
	code.WriteString("\ts.Data = results\n")
	code.WriteString("\treturn nil\n")
	code.WriteString("}\n\n")

	// parseBulletList method
	code.WriteString("// parseBulletList parses - item <!-- id:xxx --> format\n")
	code.WriteString("func (s *State) parseBulletList(lines []string) error {\n")
	code.WriteString("\tvar results []map[string]interface{}\n")
	code.WriteString("\tbulletPattern := regexp.MustCompile(`^\\s*-\\s+(.+?)(?:\\s*<!--\\s*id:(\\w+)\\s*-->)?$`)\n")
	code.WriteString("\n")
	code.WriteString("\tfor _, line := range lines {\n")
	code.WriteString("\t\tmatches := bulletPattern.FindStringSubmatch(line)\n")
	code.WriteString("\t\tif matches == nil {\n")
	code.WriteString("\t\t\tcontinue\n")
	code.WriteString("\t\t}\n")
	code.WriteString("\n")
	code.WriteString("\t\ttext := strings.TrimSpace(matches[1])\n")
	code.WriteString("\t\tid := matches[2]\n")
	code.WriteString("\n")
	code.WriteString("\t\t// Skip task list items\n")
	code.WriteString("\t\tif strings.HasPrefix(text, \"[ ]\") || strings.HasPrefix(text, \"[x]\") || strings.HasPrefix(text, \"[X]\") {\n")
	code.WriteString("\t\t\tcontinue\n")
	code.WriteString("\t\t}\n")
	code.WriteString("\n")
	code.WriteString("\t\tif id == \"\" {\n")
	code.WriteString("\t\t\tid = generateContentID(text)\n")
	code.WriteString("\t\t}\n")
	code.WriteString("\n")
	code.WriteString("\t\tresults = append(results, map[string]interface{}{\n")
	code.WriteString("\t\t\t\"id\":   id,\n")
	code.WriteString("\t\t\t\"text\": text,\n")
	code.WriteString("\t\t})\n")
	code.WriteString("\t}\n")
	code.WriteString("\n")
	code.WriteString("\ts.Data = results\n")
	code.WriteString("\treturn nil\n")
	code.WriteString("}\n\n")

	// parseTable method
	code.WriteString("// parseTable parses | col1 | col2 | <!-- id:xxx --> format\n")
	code.WriteString("func (s *State) parseTable(lines []string) error {\n")
	code.WriteString("\tvar results []map[string]interface{}\n")
	code.WriteString("\tvar headers []string\n")
	code.WriteString("\theaderParsed := false\n")
	code.WriteString("\tseparatorSeen := false\n")
	code.WriteString("\n")
	code.WriteString("\ttableRowPattern := regexp.MustCompile(`^\\s*\\|(.+)\\|(?:\\s*<!--\\s*id:(\\w+)\\s*-->)?`)\n")
	code.WriteString("\tseparatorPattern := regexp.MustCompile(`^\\s*\\|[\\s\\-:|]+\\|`)\n")
	code.WriteString("\n")
	code.WriteString("\tfor _, line := range lines {\n")
	code.WriteString("\t\tline = strings.TrimSpace(line)\n")
	code.WriteString("\t\tif line == \"\" {\n")
	code.WriteString("\t\t\tcontinue\n")
	code.WriteString("\t\t}\n")
	code.WriteString("\n")
	code.WriteString("\t\tif separatorPattern.MatchString(line) {\n")
	code.WriteString("\t\t\tseparatorSeen = true\n")
	code.WriteString("\t\t\tcontinue\n")
	code.WriteString("\t\t}\n")
	code.WriteString("\n")
	code.WriteString("\t\tmatches := tableRowPattern.FindStringSubmatch(line)\n")
	code.WriteString("\t\tif matches == nil {\n")
	code.WriteString("\t\t\tcontinue\n")
	code.WriteString("\t\t}\n")
	code.WriteString("\n")
	code.WriteString("\t\tcells := parseTableCells(matches[1])\n")
	code.WriteString("\t\tid := matches[2]\n")
	code.WriteString("\n")
	code.WriteString("\t\tif !headerParsed {\n")
	code.WriteString("\t\t\theaders = cells\n")
	code.WriteString("\t\t\theaderParsed = true\n")
	code.WriteString("\t\t\tcontinue\n")
	code.WriteString("\t\t}\n")
	code.WriteString("\n")
	code.WriteString("\t\tif !separatorSeen {\n")
	code.WriteString("\t\t\tcontinue\n")
	code.WriteString("\t\t}\n")
	code.WriteString("\n")
	code.WriteString("\t\tif id == \"\" {\n")
	code.WriteString("\t\t\tid = generateContentID(strings.Join(cells, \"|\"))\n")
	code.WriteString("\t\t}\n")
	code.WriteString("\n")
	code.WriteString("\t\trow := map[string]interface{}{\"id\": id}\n")
	code.WriteString("\t\tfor i, cell := range cells {\n")
	code.WriteString("\t\t\tif i < len(headers) {\n")
	code.WriteString("\t\t\t\trow[headers[i]] = cell\n")
	code.WriteString("\t\t\t}\n")
	code.WriteString("\t\t}\n")
	code.WriteString("\t\tresults = append(results, row)\n")
	code.WriteString("\t}\n")
	code.WriteString("\n")
	code.WriteString("\ts.Data = results\n")
	code.WriteString("\treturn nil\n")
	code.WriteString("}\n\n")

	// parseTableCells helper
	code.WriteString("// parseTableCells splits a table row into cells\n")
	code.WriteString("func parseTableCells(row string) []string {\n")
	code.WriteString("\tparts := strings.Split(row, \"|\")\n")
	code.WriteString("\tvar cells []string\n")
	code.WriteString("\tfor _, part := range parts {\n")
	code.WriteString("\t\tcell := strings.TrimSpace(part)\n")
	code.WriteString("\t\tif cell != \"\" {\n")
	code.WriteString("\t\t\tcells = append(cells, cell)\n")
	code.WriteString("\t\t}\n")
	code.WriteString("\t}\n")
	code.WriteString("\treturn cells\n")
	code.WriteString("}\n\n")

	// generateID helper
	code.WriteString("// generateID creates a random 8-character ID\n")
	code.WriteString("func generateID() string {\n")
	code.WriteString("\tbytes := make([]byte, 4)\n")
	code.WriteString("\trand.Read(bytes)\n")
	code.WriteString("\treturn hex.EncodeToString(bytes)\n")
	code.WriteString("}\n\n")

	// slugify helper
	code.WriteString("// slugify converts heading text to an anchor-compatible slug\n")
	code.WriteString("func slugify(text string) string {\n")
	code.WriteString("\ttext = strings.ToLower(text)\n")
	code.WriteString("\ttext = strings.ReplaceAll(text, \" \", \"-\")\n")
	code.WriteString("\tre := regexp.MustCompile(`[^a-z0-9-]`)\n")
	code.WriteString("\treturn re.ReplaceAllString(text, \"\")\n")
	code.WriteString("}\n\n")

	// findSectionHeader helper
	code.WriteString("// findSectionHeader finds a section header by anchor name\n")
	code.WriteString("func (s *State) findSectionHeader(content, anchorName string) []int {\n")
	code.WriteString("\t// Pattern 1: Explicit {#anchor} syntax - takes precedence\n")
	code.WriteString("\texplicitPattern := regexp.MustCompile(`(?m)^(#{1,6})\\s+(.+?)\\s*\\{#` + regexp.QuoteMeta(anchorName) + `\\}\\s*$`)\n")
	code.WriteString("\tif matches := explicitPattern.FindStringSubmatchIndex(content); matches != nil {\n")
	code.WriteString("\t\treturn matches\n")
	code.WriteString("\t}\n")
	code.WriteString("\n")
	code.WriteString("\t// Pattern 2: Match heading text (slugified) - fallback\n")
	code.WriteString("\theadingPattern := regexp.MustCompile(`(?m)^(#{1,6})\\s+(.+?)\\s*$`)\n")
	code.WriteString("\texplicitAnchorPattern := regexp.MustCompile(`\\{#[^}]+\\}\\s*$`)\n")
	code.WriteString("\tallMatches := headingPattern.FindAllStringSubmatchIndex(content, -1)\n")
	code.WriteString("\tfor _, match := range allMatches {\n")
	code.WriteString("\t\theadingText := content[match[4]:match[5]]\n")
	code.WriteString("\t\t// Skip headings with explicit anchors\n")
	code.WriteString("\t\tif explicitAnchorPattern.MatchString(headingText) {\n")
	code.WriteString("\t\t\tcontinue\n")
	code.WriteString("\t\t}\n")
	code.WriteString("\t\tif slugify(headingText) == anchorName {\n")
	code.WriteString("\t\t\treturn match\n")
	code.WriteString("\t\t}\n")
	code.WriteString("\t}\n")
	code.WriteString("\treturn nil\n")
	code.WriteString("}\n")

	return code.String(), nil
}
