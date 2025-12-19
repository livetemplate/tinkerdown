package compiler

import (
	"fmt"
	"regexp"
	"strings"
)

// sqlIdentifierRegex validates SQL identifiers (table/column names)
// Must start with letter, contain only alphanumeric and underscore
var sqlIdentifierRegex = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`)

// isValidSQLIdentifier checks if a name is a valid SQL identifier
func isValidSQLIdentifier(name string) bool {
	return sqlIdentifierRegex.MatchString(name)
}

// FormField represents a form field extracted from an LVT template
type FormField struct {
	Name     string // Field name from the name attribute
	Type     string // Go type (string, bool, int)
	HTMLType string // Original HTML input type (text, email, checkbox, etc.)
	Required bool   // Whether the field has the required attribute
}

// PersistConfig holds configuration for auto-persist forms
type PersistConfig struct {
	TableName string       // SQLite table name from lvt-persist attribute
	Action    string       // Action name from lvt-submit attribute
	Fields    []FormField  // Form fields extracted from template
}

// ParseFormFields extracts form fields from an LVT template content.
// It looks for <form lvt-persist="tablename"> and extracts all input fields.
func ParseFormFields(content string) (*PersistConfig, error) {
	// Find form with lvt-persist attribute
	formRegex := regexp.MustCompile(`<form[^>]*lvt-persist="([^"]+)"[^>]*>`)
	formMatch := formRegex.FindStringSubmatch(content)
	if formMatch == nil {
		return nil, nil // No auto-persist form found
	}

	tableName := formMatch[1]

	// Extract lvt-submit action name
	submitRegex := regexp.MustCompile(`<form[^>]*lvt-submit="([^"]+)"[^>]*>`)
	submitMatch := submitRegex.FindStringSubmatch(content)
	action := "save" // Default action name
	if submitMatch != nil {
		action = submitMatch[1]
	}

	// Extract form fields
	fields := extractFields(content)
	if len(fields) == 0 {
		return nil, fmt.Errorf("no form fields found in template with lvt-persist")
	}

	// Validate SQL identifiers to prevent injection
	if !isValidSQLIdentifier(tableName) {
		return nil, fmt.Errorf("invalid table name %q: must start with letter and contain only alphanumeric characters and underscores", tableName)
	}

	for _, field := range fields {
		if !isValidSQLIdentifier(field.Name) {
			return nil, fmt.Errorf("invalid field name %q: must start with letter and contain only alphanumeric characters and underscores", field.Name)
		}
	}

	return &PersistConfig{
		TableName: tableName,
		Action:    action,
		Fields:    fields,
	}, nil
}

// extractFields parses HTML content and extracts form fields
func extractFields(content string) []FormField {
	var fields []FormField

	// Parse input elements
	inputRegex := regexp.MustCompile(`<input[^>]*>`)
	inputMatches := inputRegex.FindAllString(content, -1)

	for _, input := range inputMatches {
		field := parseInputField(input)
		if field != nil {
			fields = append(fields, *field)
		}
	}

	// Parse textarea elements
	textareaRegex := regexp.MustCompile(`<textarea[^>]*name="([^"]+)"[^>]*>`)
	textareaMatches := textareaRegex.FindAllStringSubmatch(content, -1)

	for _, match := range textareaMatches {
		if len(match) >= 2 {
			name := match[1]
			required := strings.Contains(match[0], "required")
			fields = append(fields, FormField{
				Name:     name,
				Type:     "string",
				HTMLType: "textarea",
				Required: required,
			})
		}
	}

	// Parse select elements
	selectRegex := regexp.MustCompile(`<select[^>]*name="([^"]+)"[^>]*>`)
	selectMatches := selectRegex.FindAllStringSubmatch(content, -1)

	for _, match := range selectMatches {
		if len(match) >= 2 {
			name := match[1]
			required := strings.Contains(match[0], "required")
			fields = append(fields, FormField{
				Name:     name,
				Type:     "string",
				HTMLType: "select",
				Required: required,
			})
		}
	}

	return fields
}

// parseInputField extracts field info from an input element
func parseInputField(input string) *FormField {
	// Extract name attribute
	nameRegex := regexp.MustCompile(`name="([^"]+)"`)
	nameMatch := nameRegex.FindStringSubmatch(input)
	if nameMatch == nil {
		return nil // Skip inputs without name
	}
	name := nameMatch[1]

	// Extract type attribute (default to "text")
	typeRegex := regexp.MustCompile(`type="([^"]+)"`)
	typeMatch := typeRegex.FindStringSubmatch(input)
	htmlType := "text"
	if typeMatch != nil {
		htmlType = typeMatch[1]
	}

	// Skip submit and button types
	if htmlType == "submit" || htmlType == "button" || htmlType == "reset" {
		return nil
	}

	// Check for required attribute
	required := strings.Contains(input, "required")

	// Map HTML type to Go type
	goType := htmlTypeToGoType(htmlType)

	return &FormField{
		Name:     name,
		Type:     goType,
		HTMLType: htmlType,
		Required: required,
	}
}

// htmlTypeToGoType maps HTML input types to Go types
func htmlTypeToGoType(htmlType string) string {
	switch htmlType {
	case "checkbox":
		return "bool"
	case "number", "range":
		return "int"
	case "email", "text", "password", "tel", "url", "search", "textarea":
		return "string"
	case "date", "datetime-local", "time":
		return "string" // Store as string, can be parsed later
	case "hidden":
		return "string"
	default:
		return "string"
	}
}

// GenerateAutoPersistCode generates Go code for auto-persist forms
func GenerateAutoPersistCode(config *PersistConfig, siteDBPath string) string {
	var code strings.Builder

	// Add imports block - these will be merged by the serverblock compiler
	code.WriteString("import (\n")
	code.WriteString("\t\"database/sql\"\n")
	code.WriteString("\t\"fmt\"\n")
	code.WriteString("\t\"strings\"\n")
	code.WriteString("\t\"time\"\n")
	code.WriteString("\t_ \"github.com/mattn/go-sqlite3\"\n")
	code.WriteString(")\n\n")

	// State struct with Errors field for template error binding
	code.WriteString(fmt.Sprintf("// Auto-generated State for %s table\n", config.TableName))
	code.WriteString("type State struct {\n")
	code.WriteString("\tdb *sql.DB\n")

	// Records slice to hold loaded data
	code.WriteString(fmt.Sprintf("\t%s []%sRecord `json:\"%s\"`\n",
		capitalize(config.TableName),
		capitalize(config.TableName),
		config.TableName))

	// Errors map for template error binding
	code.WriteString("\tErrors map[string]string `json:\"errors,omitempty\"`\n")

	code.WriteString("}\n\n")

	// Record struct for individual records
	code.WriteString(fmt.Sprintf("// %sRecord represents a single record in the %s table\n",
		capitalize(config.TableName), config.TableName))
	code.WriteString(fmt.Sprintf("type %sRecord struct {\n", capitalize(config.TableName)))
	code.WriteString("\tID        int       `json:\"id\"`\n")

	for _, field := range config.Fields {
		jsonTag := strings.ToLower(field.Name)
		code.WriteString(fmt.Sprintf("\t%s %s `json:\"%s\"`\n",
			capitalize(field.Name),
			field.Type,
			jsonTag))
	}

	code.WriteString("\tCreatedAt time.Time `json:\"created_at\"`\n")
	code.WriteString("}\n\n")

	// Constructor - returns error instead of panicking
	code.WriteString("// NewState creates a new State with database connection\n")
	code.WriteString("func NewState() (*State, error) {\n")
	code.WriteString(fmt.Sprintf("\tdb, err := sql.Open(\"sqlite3\", %q)\n", siteDBPath))
	code.WriteString("\tif err != nil {\n")
	code.WriteString("\t\treturn nil, fmt.Errorf(\"failed to open database: %w\", err)\n")
	code.WriteString("\t}\n\n")

	// Create table if not exists
	code.WriteString("\t// Create table if not exists\n")
	code.WriteString(fmt.Sprintf("\t_, err = db.Exec(`CREATE TABLE IF NOT EXISTS %s (\n", config.TableName))
	code.WriteString("\t\tid INTEGER PRIMARY KEY AUTOINCREMENT,\n")

	for _, field := range config.Fields {
		sqlType := goTypeToSQLType(field.Type)
		code.WriteString(fmt.Sprintf("\t\t%s %s,\n", field.Name, sqlType))
	}

	code.WriteString("\t\tcreated_at DATETIME DEFAULT CURRENT_TIMESTAMP\n")
	code.WriteString("\t)`)\n")
	code.WriteString("\tif err != nil {\n")
	code.WriteString("\t\tdb.Close()\n")
	code.WriteString("\t\treturn nil, fmt.Errorf(\"failed to create table: %w\", err)\n")
	code.WriteString("\t}\n\n")

	// Load existing records
	code.WriteString("\ts := &State{db: db}\n")
	code.WriteString("\ts.loadRecords()\n")
	code.WriteString("\treturn s, nil\n")
	code.WriteString("}\n\n")

	// Close method for cleanup
	code.WriteString("// Close closes the database connection\n")
	code.WriteString("func (s *State) Close() error {\n")
	code.WriteString("\tif s.db != nil {\n")
	code.WriteString("\t\treturn s.db.Close()\n")
	code.WriteString("\t}\n")
	code.WriteString("\treturn nil\n")
	code.WriteString("}\n\n")

	// loadRecords method
	code.WriteString("// loadRecords loads all records from the database\n")
	code.WriteString("func (s *State) loadRecords() {\n")

	// Build column list
	columns := []string{"id"}
	for _, field := range config.Fields {
		columns = append(columns, field.Name)
	}
	columns = append(columns, "created_at")

	code.WriteString(fmt.Sprintf("\trows, err := s.db.Query(`SELECT %s FROM %s ORDER BY created_at DESC`)\n",
		strings.Join(columns, ", "), config.TableName))
	code.WriteString("\tif err != nil {\n")
	code.WriteString("\t\treturn\n")
	code.WriteString("\t}\n")
	code.WriteString("\tdefer rows.Close()\n\n")

	code.WriteString(fmt.Sprintf("\ts.%s = nil\n", capitalize(config.TableName)))
	code.WriteString("\tfor rows.Next() {\n")
	code.WriteString(fmt.Sprintf("\t\tvar r %sRecord\n", capitalize(config.TableName)))

	// Build scan arguments
	scanArgs := []string{"&r.ID"}
	for _, field := range config.Fields {
		scanArgs = append(scanArgs, "&r."+capitalize(field.Name))
	}
	scanArgs = append(scanArgs, "&r.CreatedAt")

	code.WriteString(fmt.Sprintf("\t\terr := rows.Scan(%s)\n", strings.Join(scanArgs, ", ")))
	code.WriteString("\t\tif err != nil {\n")
	code.WriteString("\t\t\tcontinue\n")
	code.WriteString("\t\t}\n")
	code.WriteString(fmt.Sprintf("\t\ts.%s = append(s.%s, r)\n",
		capitalize(config.TableName), capitalize(config.TableName)))
	code.WriteString("\t}\n")
	code.WriteString("}\n\n")

	// Save action method with validation
	code.WriteString(fmt.Sprintf("// %s handles form submission and saves to database\n", capitalize(config.Action)))
	code.WriteString(fmt.Sprintf("func (s *State) %s(ctx *livetemplate.Context) error {\n", capitalize(config.Action)))

	// Clear previous errors
	code.WriteString("\ts.Errors = make(map[string]string)\n\n")

	// Extract field values from context using livetemplate.Context accessor methods
	for _, field := range config.Fields {
		switch field.Type {
		case "bool":
			code.WriteString(fmt.Sprintf("\t%s := ctx.GetBool(\"%s\")\n", field.Name, field.Name))
		case "int":
			code.WriteString(fmt.Sprintf("\t%s := ctx.GetInt(\"%s\")\n", field.Name, field.Name))
		default:
			code.WriteString(fmt.Sprintf("\t%s := ctx.GetString(\"%s\")\n", field.Name, field.Name))
		}
	}

	code.WriteString("\n")

	// Generate validation for required fields
	hasRequiredFields := false
	for _, field := range config.Fields {
		if field.Required {
			hasRequiredFields = true
			switch field.Type {
			case "string":
				code.WriteString(fmt.Sprintf("\tif strings.TrimSpace(%s) == \"\" {\n", field.Name))
				code.WriteString(fmt.Sprintf("\t\ts.Errors[\"%s\"] = \"%s is required\"\n", field.Name, capitalize(field.Name)))
				code.WriteString("\t}\n")
			case "int":
				// For int, we can't easily distinguish "not provided" from "0"
				// so we skip validation for int fields unless explicitly needed
			case "bool":
				// For bool, we don't validate - false is a valid value
			}
		}
	}

	// Return early if validation errors
	if hasRequiredFields {
		code.WriteString("\n\tif len(s.Errors) > 0 {\n")
		code.WriteString("\t\treturn nil // Validation errors are in s.Errors\n")
		code.WriteString("\t}\n\n")
	}

	// Build insert statement
	fieldNames := make([]string, len(config.Fields))
	placeholders := make([]string, len(config.Fields))
	valueVars := make([]string, len(config.Fields))

	for i, field := range config.Fields {
		fieldNames[i] = field.Name
		placeholders[i] = "?"
		valueVars[i] = field.Name
	}

	code.WriteString(fmt.Sprintf("\t_, err := s.db.Exec(`INSERT INTO %s (%s) VALUES (%s)`,\n",
		config.TableName,
		strings.Join(fieldNames, ", "),
		strings.Join(placeholders, ", ")))
	code.WriteString(fmt.Sprintf("\t\t%s)\n", strings.Join(valueVars, ", ")))
	code.WriteString("\tif err != nil {\n")
	code.WriteString("\t\treturn err\n")
	code.WriteString("\t}\n\n")

	// Reload records
	code.WriteString("\ts.loadRecords()\n")
	code.WriteString("\treturn nil\n")
	code.WriteString("}\n\n")

	// Generate Update action method
	code.WriteString(fmt.Sprintf("// Update handles updating an existing record in the %s table\n", config.TableName))
	code.WriteString("func (s *State) Update(ctx *livetemplate.Context) error {\n")
	code.WriteString("\ts.Errors = make(map[string]string)\n\n")
	code.WriteString("\tid := ctx.GetInt(\"id\")\n")
	code.WriteString("\tif id == 0 {\n")
	code.WriteString("\t\ts.Errors[\"id\"] = \"ID is required for update\"\n")
	code.WriteString("\t\treturn nil\n")
	code.WriteString("\t}\n\n")

	// Extract field values for update
	for _, field := range config.Fields {
		switch field.Type {
		case "bool":
			code.WriteString(fmt.Sprintf("\t%s := ctx.GetBool(\"%s\")\n", field.Name, field.Name))
		case "int":
			code.WriteString(fmt.Sprintf("\t%s := ctx.GetInt(\"%s\")\n", field.Name, field.Name))
		default:
			code.WriteString(fmt.Sprintf("\t%s := ctx.GetString(\"%s\")\n", field.Name, field.Name))
		}
	}
	code.WriteString("\n")

	// Validation for required fields in update
	for _, field := range config.Fields {
		if field.Required && field.Type == "string" {
			code.WriteString(fmt.Sprintf("\tif strings.TrimSpace(%s) == \"\" {\n", field.Name))
			code.WriteString(fmt.Sprintf("\t\ts.Errors[\"%s\"] = \"%s is required\"\n", field.Name, capitalize(field.Name)))
			code.WriteString("\t}\n")
		}
	}

	// Check for validation errors
	code.WriteString("\n\tif len(s.Errors) > 0 {\n")
	code.WriteString("\t\treturn nil\n")
	code.WriteString("\t}\n\n")

	// Build UPDATE statement
	setClause := make([]string, len(config.Fields))
	for i, field := range config.Fields {
		setClause[i] = fmt.Sprintf("%s=?", field.Name)
	}

	code.WriteString(fmt.Sprintf("\t_, err := s.db.Exec(`UPDATE %s SET %s WHERE id=?`,\n",
		config.TableName, strings.Join(setClause, ", ")))
	code.WriteString(fmt.Sprintf("\t\t%s, id)\n", strings.Join(valueVars, ", ")))
	code.WriteString("\tif err != nil {\n")
	code.WriteString("\t\treturn err\n")
	code.WriteString("\t}\n\n")
	code.WriteString("\ts.loadRecords()\n")
	code.WriteString("\treturn nil\n")
	code.WriteString("}\n\n")

	// Generate Delete action method
	code.WriteString(fmt.Sprintf("// Delete handles deleting a record from the %s table\n", config.TableName))
	code.WriteString("func (s *State) Delete(ctx *livetemplate.Context) error {\n")
	code.WriteString("\ts.Errors = make(map[string]string)\n\n")
	code.WriteString("\tid := ctx.GetInt(\"id\")\n")
	code.WriteString("\tif id == 0 {\n")
	code.WriteString("\t\ts.Errors[\"id\"] = \"ID is required for delete\"\n")
	code.WriteString("\t\treturn nil\n")
	code.WriteString("\t}\n\n")
	code.WriteString(fmt.Sprintf("\t_, err := s.db.Exec(`DELETE FROM %s WHERE id=?`, id)\n", config.TableName))
	code.WriteString("\tif err != nil {\n")
	code.WriteString("\t\treturn err\n")
	code.WriteString("\t}\n\n")
	code.WriteString("\ts.loadRecords()\n")
	code.WriteString("\treturn nil\n")
	code.WriteString("}\n")

	return code.String()
}

// goTypeToSQLType maps Go types to SQLite types
func goTypeToSQLType(goType string) string {
	switch goType {
	case "int":
		return "INTEGER"
	case "bool":
		return "INTEGER" // SQLite uses INTEGER for boolean
	case "float64":
		return "REAL"
	default:
		return "TEXT"
	}
}

// capitalize capitalizes the first letter of a string
func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
