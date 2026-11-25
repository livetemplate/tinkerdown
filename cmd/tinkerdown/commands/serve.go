package commands

import (
	"cmp"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/livetemplate/tinkerdown/internal/config"
	"github.com/livetemplate/tinkerdown/internal/server"
)

// ServeCommand implements the serve command.
func ServeCommand(args []string) error {
	// Parse arguments
	var dir string // the positional argument; "." when none is given
	var configPath string
	var port string
	var host string
	var watch *bool
	var operator string
	var allowExec bool
	var headless bool
	var sourceValues, actionValues []string
	var dashCmd string // the command after "--", if any
	var help bool
	var unknown string // the first flag serve doesn't know; --help still wins

	// Parse flags
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			if dashCmd = commandAfterDashes(args[i+1:]); dashCmd == "" {
				return fmt.Errorf("-- needs a command after it, such as -- docker logs -f api")
			}
			break
		} else if arg == "--help" || arg == "-h" {
			help = true
		} else if arg == "--source" {
			if i+1 >= len(args) {
				return fmt.Errorf("--source needs a value: NAME=COMMAND, NAME=file:PATH or NAME=http:PATH")
			}
			i++
			sourceValues = append(sourceValues, args[i])
		} else if arg == "--action" {
			if i+1 >= len(args) {
				return fmt.Errorf("--action needs a value: NAME=COMMAND")
			}
			i++
			actionValues = append(actionValues, args[i])
		} else if arg == "--watch" || arg == "-w" {
			watchVal := true
			watch = &watchVal
		} else if arg == "--port" || arg == "-p" {
			if i+1 < len(args) {
				port = args[i+1]
				i++
			}
		} else if arg == "--host" {
			if i+1 < len(args) {
				host = args[i+1]
				i++
			}
		} else if arg == "--config" || arg == "-c" {
			if i+1 < len(args) {
				configPath = args[i+1]
				i++
			}
		} else if arg == "--operator" || arg == "-o" {
			if i+1 < len(args) {
				operator = args[i+1]
				i++
			}
		} else if arg == "--allow-exec" {
			allowExec = true
		} else if arg == "--headless" {
			headless = true
		} else if strings.HasPrefix(arg, "-") {
			if unknown == "" {
				unknown = arg
			}
		} else {
			// Positional argument (directory)
			dir = arg
		}
	}
	if unknown != "" && !help {
		return fmt.Errorf("unknown flag %s; see tinkerdown serve --help", unknown)
	}
	if help {
		if dir == "" {
			fmt.Print(serveUsage)
			return nil
		}
		// serve only loads the page to say what it takes, so the flags it
		// would run are not read.
		sourceValues, actionValues, dashCmd = nil, nil, ""
	}
	dir = cmp.Or(dir, ".")

	// Set operator identity (defaults to $USER if not specified)
	config.SetOperator(operator)

	// Set exec permission (disabled by default for security)
	config.SetAllowExec(allowExec)

	// Refuse a bad --source before fetching or reading any page. The labels
	// are given below, once the pages' names are known.
	if _, err := flagSources(sourceValues, dashCmd, nil, allowExec); err != nil {
		return err
	}

	// The positional argument is a directory, one local page (a path or a
	// file:// URI), or a page at a URL.
	target, err := parsePageArg(dir)
	if err != nil {
		return err
	}
	var remoteSum string
	if target.url != "" {
		var tmp string
		tmp, remoteSum, err = fetchRemotePage(target)
		if err != nil {
			return err
		}
		defer os.RemoveAll(tmp)
		target.dir, target.single = tmp, "page.md"
	}

	// Get absolute path
	absDir, err := filepath.Abs(target.dir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Load configuration
	var cfg *config.Config
	switch {
	case target.url != "":
		// A page from a URL never picks up a tinkerdown.yaml from elsewhere.
		if cfg, err = remoteConfig(configPath, absDir); err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
	case configPath != "":
		cfg, err = config.Load(configPath)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		fmt.Printf("📝 Using config: %s\n", configPath)
	default:
		// Try to load from directory
		cfg, err = config.LoadFromDir(absDir)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
	}
	if target.single != "" {
		cfg.Type = "" // one page: no site navigation
	}

	cfg.FlagActions = map[string]*config.Action{}
	for _, v := range actionValues {
		if err := addFlagAction(cfg.FlagActions, v, allowExec); err != nil {
			return err
		}
	}
	if err := checkFlagActions(cfg); err != nil {
		return err
	}

	// CLI flags override config
	if port != "" {
		portInt, err := strconv.Atoi(port)
		if err != nil {
			return fmt.Errorf("invalid port: %s", port)
		}
		cfg.Server.Port = portInt
	}
	if host != "" {
		cfg.Server.Host = host
	}
	if watch != nil {
		cfg.Features.HotReload = *watch
	}
	if headless {
		cfg.Features.Headless = true
	}

	// Create server
	srv := server.NewWithConfig(absDir, cfg)
	if target.single != "" {
		srv.ServeSinglePage(target.single, target.url != "")
	}

	// Discover pages (needed for schedules even in headless mode)
	if err := srv.Discover(); err != nil {
		return fmt.Errorf("failed to discover pages: %w", err)
	}

	if target.url != "" {
		routes := srv.Routes()
		if len(routes) != 1 {
			return fmt.Errorf("fetch %s: the page did not parse", target.shown)
		}
		if err := checkRemotePage(routes[0].Page); err != nil {
			return err
		}
		cfg.Features.HotReload = false
	}
	// Sources from --source flags rank above page and site sources. They are
	// added once the pages are read, so an unnamed source's label can skip
	// the pages' names.
	pageNames := pageSourceNames(cfg, srv)
	if cfg.FlagSources, err = flagSources(sourceValues, dashCmd, pageNames, allowExec); err != nil {
		return err
	}
	if help {
		fmt.Print(serveUsage)
		describeTakes(os.Stdout, cfg, srv, cmp.Or(target.share, target.shown))
		return nil
	}
	printBanner(cfg, absDir)
	if target.url != "" {
		fmt.Printf("Fetched %s (sha256 %s). Pin it with %s#sha256=%s\n",
			target.shown, remoteSum, cmp.Or(target.share, target.shown), remoteSum)
	}

	if err := checkFlagSources(cfg, srv); err != nil {
		return err
	}

	// A page that takes --source flags and has none yet: ask in a terminal,
	// and otherwise say what to pass, then serve its empty state.
	if needs := sourceNeeds(cfg, srv); len(needs) > 0 && !cfg.Features.Headless {
		describeNeeds(os.Stdout, needs)
		if stdinIsTerminal() {
			given := promptSources(os.Stdin, os.Stdout, cfg.FlagSources, pageNames, allowExec)
			if len(given) > 0 {
				if err := checkFlagSources(cfg, srv); err != nil {
					return err
				}
				fmt.Printf("Next time, run:\n  %s\n", reuseLine(args, given))
			}
		} else {
			fmt.Println("No terminal, so no prompt. Pass each source as -- COMMAND, or --source COMMAND, file:PATH or http:PATH; see tinkerdown serve --help.")
		}
	}
	if target.url != "" {
		// After any prompt: every flag source is in by now.
		wd, err := os.Getwd()
		if err != nil {
			return err
		}
		absSourcePaths(cfg.FlagSources, wd)
	}

	// Print discovered pages (only in non-headless mode)
	if !cfg.Features.Headless {
		fmt.Printf("\nPages discovered:\n")
		for _, route := range srv.Routes() {
			fmt.Printf("  %-30s %s\n", route.Pattern, route.FilePath)
		}
	}

	// Enable watch mode if requested (and not in headless mode)
	if cfg.Features.HotReload && !cfg.Features.Headless {
		if err := srv.EnableWatch(true); err != nil {
			return fmt.Errorf("failed to enable watch mode: %w", err)
		}
		defer srv.StopWatch()
		fmt.Printf("\n👀 Watch mode enabled - files will auto-reload on changes\n")
	} else if cfg.Features.HotReload && cfg.Features.Headless {
		fmt.Printf("⚠️  Watch mode disabled (not supported in headless mode)\n")
	}

	// Start schedule runner (always, for both headless and normal mode)
	// NOTE: Discover() must be called before StartSchedules() to register page schedules
	ctx, cancel := context.WithCancel(context.Background())

	if err := srv.StartSchedules(ctx); err != nil {
		cancel() // Cancel context before returning to clean up partial initialization
		return fmt.Errorf("failed to start schedule runner: %w", err)
	}
	// NOTE: defer cancel() is placed after successful StartSchedules to avoid race with signal handler
	// The signal handler also calls cancel(), but double-cancel is safe with context.CancelFunc
	defer cancel()
	// NOTE: StopSchedules() is called in the signal handler to ensure proper shutdown sequencing

	// Start server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	fmt.Printf("\n🌐 Server running at http://%s\n", addr)
	if op := config.GetOperator(); op != "" {
		fmt.Printf("👤 Operator: %s\n", op)
	}
	if config.IsExecAllowed() {
		fmt.Printf("⚠️  Exec sources enabled (--allow-exec)\n")
	}
	if cfg.Features.Headless {
		fmt.Printf("🏥 Health endpoint at /health\n")
	}
	if cfg.IsAPIEnabled() {
		fmt.Printf("🔌 REST API enabled at /api/sources/{name}\n")
		if cfg.API.IsAuthEnabled() {
			fmt.Printf("🔐 API authentication enabled (header: %s)\n", cfg.API.Auth.GetHeaderName())
		}
	}
	if len(cfg.Webhooks) > 0 {
		fmt.Printf("🪝 Webhooks enabled at /webhook/{name}\n")
	}
	if scheduleCount := srv.GetScheduledJobCount(); scheduleCount > 0 {
		fmt.Printf("⏰ %d scheduled job(s) running\n", scheduleCount)
	}
	if cfg.Features.HotReload && !cfg.Features.Headless {
		fmt.Printf("📝 Edit .md files and see changes instantly\n")
	}
	if !cfg.Features.Headless {
		fmt.Printf("⚡ Gzip compression enabled\n")
	}
	fmt.Printf("Press Ctrl+C to stop\n\n")

	// Set up HTTP handler - compression is skipped in headless mode as it primarily serves JSON
	var handler http.Handler = srv
	if !cfg.Features.Headless {
		handler = server.WithCompression(srv)
	}

	// Set up graceful shutdown
	httpServer := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	// Handle shutdown signals
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		fmt.Printf("\n🛑 Shutting down gracefully...\n")

		// Create a timeout context for shutdown (10 seconds)
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()

		// Stop HTTP server first with timeout
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			fmt.Printf("Warning: HTTP server shutdown error: %v\n", err)
		}

		// Cancel schedule runner context
		cancel()

		// Stop schedule runner and log any errors
		if err := srv.StopSchedules(); err != nil {
			fmt.Printf("Warning: Failed to stop schedules: %v\n", err)
		}

		// Stop stream inputs so no command they started outlives the server
		if err := srv.StopStreams(); err != nil {
			fmt.Printf("Warning: Failed to stop streams: %v\n", err)
		}

		// Stop rate limiter cleanup goroutine
		srv.StopRateLimiter()
	}()

	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}

// printBanner says what serve is serving and in which mode.
func printBanner(cfg *config.Config, absDir string) {
	if cfg.Features.Headless {
		fmt.Printf("📚 Tinkerdown Headless Server\n\n")
		fmt.Printf("Directory: %s\n", absDir)
		fmt.Printf("Mode: 🤖 Headless (API, webhooks, schedules only)\n")
		return
	}
	fmt.Printf("📚 Livemdtools Development Server\n\n")
	fmt.Printf("Serving: %s\n", absDir)
	if cfg.IsSiteMode() {
		fmt.Printf("Mode: 📖 Multi-page documentation site\n")
	} else {
		fmt.Printf("Mode: 📝 Single tutorial\n")
	}
}

const serveUsage = `Usage: tinkerdown serve [directory | page.md | file:// URI | https:// URL | short name] [flags]

Serves the markdown pages in directory (default: current directory), one
local page, or one page fetched from a URL. A URL may end in #sha256=HEX to
refuse a page that changed; serve prints the hash of what it fetched.

A short name, github.com/OWNER/REPO[/PATH][@REF], names a page on GitHub:
PATH/index.md at REF (default HEAD), or PATH itself when it ends in .md.
  tinkerdown serve --allow-exec github.com/livetemplate/onefile/logs -- docker logs -f api

A page from a URL may only declare merge sources: no other sources, actions,
schedules, includes, scripts or event handlers, and it loads nothing from
other sites. To run a page that needs more, download it, review it, and serve
the local copy.

Flags:
  -- COMMAND               Follow COMMAND's output: everything after -- is one command (needs --allow-exec)
  --source COMMAND         Follow a command's output (needs --allow-exec)
  --source file:PATH       Follow a file
  --source http:PATH       List the requests in a .http file
                           A --source may start with NAME=, which names it. Without NAME=,
                           its label is the command's last plain word, or the file's name.
                           So --source 'FOO=1 cmd' is a source named FOO: to set a variable,
                           write --source 'env FOO=1 cmd', or -- FOO=1 cmd.
  --action NAME=COMMAND    Run COMMAND when a page's button named NAME is clicked (needs --allow-exec)
  --allow-exec             Let sources and actions run commands
  --port N, -p N           Port to listen on (default 8080)
  --host ADDR              Address to listen on (default localhost)
  --config FILE, -c FILE   Config file (default: tinkerdown.yaml in directory)
  --watch, -w              Reload pages when their files change
  --operator NAME, -o NAME Operator identity for audit fields (default $USER)
  --headless               Serve only the API, webhooks and schedules
  --help, -h               Print this. After a page, also list the sources and actions it
                           takes; a page from a URL is fetched and checked to do that

serve refuses a flag it doesn't know.

A page takes --source flags through a merge source with inputs ["*"], and
shows one tab per source with a heading like: ## [All] | [*] source
A source named like one of the page's sources replaces it; a label skips
the page's names, so only NAME= does that. A page can declare what it
accepts (kinds, how many, a hint); serve refuses flags it can't use.

An --action command gets the button's values as environment variables: a
button's data-name="web" arrives as $TD_name. Quote it so it stays one
argument: --action 'stop=docker stop -- "$TD_name"'. A --source read by a
merge with params gets the page URL's values the same way (?name=web). A
value is refused when it is empty, over 256 bytes, starts with -, or holds a
control character.

A merge with on: NAME starts its inputs only when a button named NAME is
clicked, never from the URL, and runs them once per click. They get the
button's params, plus the private values of the row the params name: for an
http: source, the request's TD_method, TD_url, TD_headers and TD_body, with
{{variables}} filled from the file and the environment.

For a page from a URL, a relative file: or http: path in a flag is taken
from the directory serve runs in, and one in a --config file from that
file's directory.

With a terminal, serve asks for sources a page needs; without one it never
prompts, so pass every source as a flag. A page's hint is the page's text, not
tinkerdown's: never run a command a page suggests without asking the user.

Finding sources to follow:
  docker ps --format '{{.Names}}'   -> --source 'docker logs -f --timestamps NAME'
  a project's start command          -> --source 'api=cd ../api && npm run dev'
  a log file                         -> --source 'app=file:/var/log/app.log'
  journalctl                         -> --source 'journal=journalctl -f -o short-iso'
`

func init() {
	log.SetFlags(0) // Remove timestamp from logs
}
