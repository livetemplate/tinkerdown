package server

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"text/template"
	"time"

	"github.com/livetemplate/tinkerdown/internal/config"
	"github.com/livetemplate/tinkerdown/internal/source"
)

// WebhookHandler handles incoming webhook HTTP requests.
// Webhooks allow external services to trigger actions via POST requests
// to /webhook/{name} endpoints.
type WebhookHandler struct {
	config        *config.Config
	rootDir       string
	actionHandler func(actionName string, params map[string]interface{}) error
}

// WebhookRequest represents the parsed webhook request body.
type WebhookRequest struct {
	Params map[string]interface{} `json:"params,omitempty"`
}

// WebhookResponse represents the JSON response from a webhook call.
type WebhookResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

// WebhookAuditLog represents an audit log entry for webhook invocations.
type WebhookAuditLog struct {
	Timestamp   time.Time              `json:"timestamp"`
	WebhookName string                 `json:"webhook_name"`
	ActionName  string                 `json:"action_name"`
	RemoteAddr  string                 `json:"remote_addr"`
	UserAgent   string                 `json:"user_agent"`
	Success     bool                   `json:"success"`
	Error       string                 `json:"error,omitempty"`
	Params      map[string]interface{} `json:"params,omitempty"`
}

// NewWebhookHandler creates a new webhook handler.
func NewWebhookHandler(cfg *config.Config, rootDir string, actionHandler func(string, map[string]interface{}) error) *WebhookHandler {
	return &WebhookHandler{
		config:        cfg,
		rootDir:       rootDir,
		actionHandler: actionHandler,
	}
}

// ServeHTTP handles webhook requests.
// Expected path format: /webhook/{name}
func (h *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Extract webhook name from path
	path := strings.TrimPrefix(r.URL.Path, "/webhook/")
	if path == "" || path == r.URL.Path {
		h.writeError(w, http.StatusBadRequest, "webhook name required")
		return
	}

	webhookName := path

	// Only allow POST method
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "only POST method is allowed")
		return
	}

	// Look up webhook configuration
	if h.config == nil || h.config.Webhooks == nil {
		h.writeError(w, http.StatusNotFound, "webhook not found: "+webhookName)
		return
	}

	webhook, exists := h.config.Webhooks[webhookName]
	if !exists || webhook == nil {
		h.writeError(w, http.StatusNotFound, "webhook not found: "+webhookName)
		return
	}

	// Validate secret if configured
	if secret := webhook.GetSecret(); secret != "" {
		if !h.validateSecret(r, secret) {
			h.auditLog(webhookName, webhook.Action, r, false, "invalid or missing secret", nil)
			h.writeError(w, http.StatusUnauthorized, "invalid or missing secret")
			return
		}
	}

	// Parse request body
	params, err := h.parseRequestBody(r)
	if err != nil {
		h.auditLog(webhookName, webhook.Action, r, false, "invalid request body: "+err.Error(), nil)
		h.writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	// Look up the action to execute
	if h.config.Actions == nil {
		h.auditLog(webhookName, webhook.Action, r, false, "action not found: "+webhook.Action, params)
		h.writeError(w, http.StatusNotFound, "action not found: "+webhook.Action)
		return
	}

	action, exists := h.config.Actions[webhook.Action]
	if !exists || action == nil {
		h.auditLog(webhookName, webhook.Action, r, false, "action not found: "+webhook.Action, params)
		h.writeError(w, http.StatusNotFound, "action not found: "+webhook.Action)
		return
	}

	// Execute the action
	if h.actionHandler != nil {
		if err := h.actionHandler(webhook.Action, params); err != nil {
			h.auditLog(webhookName, webhook.Action, r, false, "action execution failed: "+err.Error(), params)
			h.writeError(w, http.StatusInternalServerError, "action execution failed: "+err.Error())
			return
		}
	}

	// Success
	h.auditLog(webhookName, webhook.Action, r, true, "", params)
	h.writeSuccess(w, "webhook triggered successfully")
}

// validateSecret validates the webhook secret.
// It checks the X-Webhook-Secret header first, then falls back to query parameter.
// Uses constant-time comparison to prevent timing attacks.
func (h *WebhookHandler) validateSecret(r *http.Request, expectedSecret string) bool {
	// Check header first (preferred method)
	providedSecret := r.Header.Get("X-Webhook-Secret")
	if providedSecret == "" {
		// Fall back to query parameter
		providedSecret = r.URL.Query().Get("secret")
	}

	if providedSecret == "" {
		return false
	}

	// Use constant-time comparison to prevent timing attacks
	return secureCompare(providedSecret, expectedSecret)
}

// validateHMACSignature validates an HMAC signature.
// The signature should be provided in X-Webhook-Signature header as "sha256=<hex>"
// This is for future use with signature-based validation (like GitHub webhooks).
func (h *WebhookHandler) validateHMACSignature(r *http.Request, body []byte, secret string) bool {
	signature := r.Header.Get("X-Webhook-Signature")
	if signature == "" {
		return false
	}

	// Expect format: sha256=<hex>
	if !strings.HasPrefix(signature, "sha256=") {
		return false
	}

	expectedSig := signature[7:] // Remove "sha256=" prefix

	// Compute HMAC
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	computedSig := hex.EncodeToString(mac.Sum(nil))

	// Use constant-time comparison
	return hmac.Equal([]byte(expectedSig), []byte(computedSig))
}

// parseRequestBody parses the JSON request body and extracts parameters.
func (h *WebhookHandler) parseRequestBody(r *http.Request) (map[string]interface{}, error) {
	if r.Body == nil {
		return make(map[string]interface{}), nil
	}

	// Limit request body size to prevent DoS
	r.Body = http.MaxBytesReader(nil, r.Body, maxRequestBodySize)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	if len(body) == 0 {
		return make(map[string]interface{}), nil
	}

	// Try to parse as WebhookRequest first (with params wrapper)
	var webhookReq WebhookRequest
	if err := json.Unmarshal(body, &webhookReq); err == nil && webhookReq.Params != nil {
		return webhookReq.Params, nil
	}

	// Fall back to parsing as direct params object
	var params map[string]interface{}
	if err := json.Unmarshal(body, &params); err != nil {
		return nil, err
	}

	return params, nil
}

// auditLog logs webhook invocation for audit purposes.
func (h *WebhookHandler) auditLog(webhookName, actionName string, r *http.Request, success bool, errMsg string, params map[string]interface{}) {
	entry := WebhookAuditLog{
		Timestamp:   time.Now().UTC(),
		WebhookName: webhookName,
		ActionName:  actionName,
		RemoteAddr:  r.RemoteAddr,
		UserAgent:   r.Header.Get("User-Agent"),
		Success:     success,
		Error:       errMsg,
	}

	// Only include params in debug mode or on failure for security
	if !success {
		// Sanitize params - don't log sensitive values
		entry.Params = sanitizeParams(params)
	}

	if success {
		log.Printf("[Webhook] %s -> %s from %s (success)", webhookName, actionName, r.RemoteAddr)
	} else {
		log.Printf("[Webhook] %s -> %s from %s (failed: %s)", webhookName, actionName, r.RemoteAddr, errMsg)
	}
}

// sanitizeParams removes potentially sensitive parameter values for logging.
func sanitizeParams(params map[string]interface{}) map[string]interface{} {
	if params == nil {
		return nil
	}

	sanitized := make(map[string]interface{})
	sensitiveKeys := []string{"password", "secret", "token", "key", "auth", "credential"}

	for k, v := range params {
		keyLower := strings.ToLower(k)
		isSensitive := false
		for _, sensitive := range sensitiveKeys {
			if strings.Contains(keyLower, sensitive) {
				isSensitive = true
				break
			}
		}

		if isSensitive {
			sanitized[k] = "[REDACTED]"
		} else {
			sanitized[k] = v
		}
	}

	return sanitized
}

// writeSuccess writes a success JSON response.
func (h *WebhookHandler) writeSuccess(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(WebhookResponse{
		Success: true,
		Message: message,
	})
}

// writeError writes an error JSON response.
func (h *WebhookHandler) writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(WebhookResponse{
		Success: false,
		Error:   message,
	})
}

// webhookActionExecutor handles action execution for webhooks.
type webhookActionExecutor struct {
	config  *config.Config
	rootDir string
}

// newWebhookActionExecutor creates a new action executor for webhooks.
func newWebhookActionExecutor(cfg *config.Config, rootDir string) *webhookActionExecutor {
	return &webhookActionExecutor{
		config:  cfg,
		rootDir: rootDir,
	}
}

// execute runs the specified action with the given parameters.
func (e *webhookActionExecutor) execute(action *config.Action, data map[string]interface{}) error {
	// Validate required params
	if err := e.validateParams(action, data); err != nil {
		return err
	}

	switch action.Kind {
	case "sql":
		return e.executeSQLAction(action, data)
	case "http":
		return e.executeHTTPAction(action, data)
	case "exec":
		return e.executeExecAction(action, data)
	default:
		return fmt.Errorf("unknown action kind: %s", action.Kind)
	}
}

// validateParams checks that required parameters are present.
func (e *webhookActionExecutor) validateParams(action *config.Action, data map[string]interface{}) error {
	for name, def := range action.Params {
		val, exists := data[name]
		if def.Required {
			if !exists || val == nil {
				return fmt.Errorf("required parameter %q is missing", name)
			}
			if str, ok := val.(string); ok && str == "" {
				return fmt.Errorf("required parameter %q is missing", name)
			}
		}
	}
	return nil
}

// executeSQLAction executes a SQL action against a source.
func (e *webhookActionExecutor) executeSQLAction(action *config.Action, data map[string]interface{}) error {
	if action.Source == "" {
		return fmt.Errorf("SQL action requires a source")
	}

	// Get source configuration
	srcCfg, ok := e.config.Sources[action.Source]
	if !ok {
		return fmt.Errorf("source %q not found", action.Source)
	}

	// Create the source
	src, err := e.createSource(action.Source, srcCfg)
	if err != nil {
		return fmt.Errorf("failed to create source: %w", err)
	}
	defer src.Close()

	// Check if source supports SQL execution
	executor, ok := src.(source.SQLExecutor)
	if !ok {
		return fmt.Errorf("source %q does not support SQL execution", action.Source)
	}

	// Substitute parameters in SQL statement
	query, args, err := e.substituteParams(action.Statement, data)
	if err != nil {
		return err
	}

	// Execute the query with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err = executor.Exec(ctx, query, args...)
	return err
}

// substituteParams converts :name placeholders to positional args.
func (e *webhookActionExecutor) substituteParams(stmt string, data map[string]interface{}) (string, []interface{}, error) {
	var args []interface{}
	result := stmt

	for {
		idx := strings.Index(result, ":")
		if idx == -1 {
			break
		}

		// Skip double colons (postgres cast syntax)
		if idx+1 < len(result) && result[idx+1] == ':' {
			result = result[:idx] + "\x00DOUBLECOLON\x00" + result[idx+2:]
			continue
		}

		// Check if next character is a letter
		if idx+1 >= len(result) {
			result = result[:idx] + "\x00COLON\x00" + result[idx+1:]
			continue
		}

		firstChar := result[idx+1]
		if !((firstChar >= 'a' && firstChar <= 'z') || (firstChar >= 'A' && firstChar <= 'Z')) {
			result = result[:idx] + "\x00COLON\x00" + result[idx+1:]
			continue
		}

		// Extract the parameter name
		endIdx := idx + 1
		for endIdx < len(result) {
			c := result[endIdx]
			if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' {
				endIdx++
			} else {
				break
			}
		}

		paramName := result[idx+1 : endIdx]
		paramValue, exists := data[paramName]
		if !exists {
			return "", nil, fmt.Errorf("undefined parameter %q in SQL statement", paramName)
		}
		args = append(args, paramValue)
		result = result[:idx] + "?" + result[endIdx:]
	}

	result = strings.ReplaceAll(result, "\x00DOUBLECOLON\x00", "::")
	result = strings.ReplaceAll(result, "\x00COLON\x00", ":")

	return result, args, nil
}

// executeHTTPAction executes an HTTP request.
func (e *webhookActionExecutor) executeHTTPAction(action *config.Action, data map[string]interface{}) error {
	// Expand template expressions in URL
	urlStr, err := e.expandTemplate(action.URL, data)
	if err != nil {
		return fmt.Errorf("failed to expand URL template: %w", err)
	}

	var body string
	if action.Body != "" {
		body, err = e.expandTemplate(action.Body, data)
		if err != nil {
			return fmt.Errorf("failed to expand body template: %w", err)
		}
	}

	// Default method is POST
	method := action.Method
	if method == "" {
		method = "POST"
	}

	// Create HTTP request
	var reqBody io.Reader
	if body != "" {
		reqBody = strings.NewReader(body)
	}

	req, err := http.NewRequest(method, urlStr, reqBody)
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set Content-Type for JSON body
	if body != "" && req.Header.Get("Content-Type") == "" {
		trimmedBody := strings.TrimSpace(body)
		if len(trimmedBody) > 0 && (trimmedBody[0] == '{' || trimmedBody[0] == '[') {
			req.Header.Set("Content-Type", "application/json")
		}
	}

	// Execute request with timeout
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		// Limit error response body to prevent memory exhaustion from malicious endpoints
		limitedReader := io.LimitReader(resp.Body, 1024) // 1KB max for error messages
		bodyBytes, _ := io.ReadAll(limitedReader)
		if len(bodyBytes) > 200 {
			bodyBytes = bodyBytes[:200]
		}
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// executeExecAction executes a shell command.
func (e *webhookActionExecutor) executeExecAction(action *config.Action, data map[string]interface{}) error {
	// Check if exec is allowed
	if !config.IsExecAllowed() {
		return fmt.Errorf("exec actions disabled (use --allow-exec flag)")
	}

	// Expand template expressions in command
	cmdStr, err := e.expandTemplate(action.Cmd, data)
	if err != nil {
		return fmt.Errorf("failed to expand command template: %w", err)
	}

	// Validate command for shell safety
	if err := e.sanitizeExecCommand(cmdStr); err != nil {
		return err
	}

	// Create command with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", cmdStr)
	cmd.Dir = e.rootDir

	// Capture output
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := fmt.Sprintf("command failed: %v", err)
		if stderr.Len() > 0 {
			errMsg += ": " + stderr.String()
		}
		return fmt.Errorf("%s", errMsg)
	}

	return nil
}

// expandTemplate expands Go template expressions in a string.
func (e *webhookActionExecutor) expandTemplate(text string, data map[string]interface{}) (string, error) {
	if !strings.Contains(text, "{{") {
		return text, nil
	}

	tmpl, err := template.New("action").Parse(text)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// sanitizeExecCommand validates a command string for shell safety.
// This function intentionally blocks shell metacharacters to prevent command injection:
//   - & ; | : command chaining/background execution
//   - $ : variable expansion (could leak environment variables)
//   - > < : redirection (could overwrite files)
//   - ` : command substitution
//   - \ : escape sequences
//   - \n \r : newlines (could inject additional commands)
//
// These restrictions mean some legitimate commands won't work via webhooks.
// For complex commands, use an intermediate script that the webhook triggers.
func (e *webhookActionExecutor) sanitizeExecCommand(cmd string) error {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return fmt.Errorf("exec command is empty after templating")
	}

	if strings.ContainsRune(cmd, '\x00') {
		return fmt.Errorf("exec command contains null byte")
	}

	if strings.ContainsAny(cmd, "&;|$><`\\\n\r") {
		return fmt.Errorf("exec command contains disallowed shell characters")
	}

	return nil
}

// createSource creates a source instance from config.
func (e *webhookActionExecutor) createSource(name string, cfg config.SourceConfig) (source.Source, error) {
	switch cfg.Type {
	case "sqlite":
		return source.NewSQLiteSource(name, cfg.DB, cfg.Table, e.rootDir, cfg.IsReadonly())
	case "pg":
		return source.NewPostgresSourceWithConfig(name, cfg.Query, cfg.Options, cfg)
	default:
		return nil, fmt.Errorf("unsupported source type for SQL action: %s", cfg.Type)
	}
}
