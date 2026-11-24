package commands

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/livetemplate/tinkerdown"
	"github.com/livetemplate/tinkerdown/internal/config"
	"github.com/livetemplate/tinkerdown/internal/security"
)

// Limits for a page fetched from a URL.
const (
	maxRemotePage      = 1 << 20 // bytes
	remoteFetchTimeout = 15 * time.Second
	maxRemoteRedirects = 3
	maxRemoteBuffer    = 10000
	minRemoteFlush     = 50 * time.Millisecond
)

// pageArg is what serve's positional argument names: a directory, one local
// page, or a page at a URL, which a short name expands to.
type pageArg struct {
	dir    string // site root
	single string // with dir: the one page to serve, relative to dir
	url    string // a remote page's URL, without the #sha256= pin
	shown  string // url without credentials, for every message
	share  string // a short name as typed, which the pin line offers instead of shown
	pin    string // the expected sha256, hex
}

// parsePageArg sorts serve's positional argument into a directory, a local
// page (a path or a file:// URI), a remote URL, or a short name such as
// github.com/OWNER/REPO/PATH[@REF], which names a page on GitHub. An existing
// local path wins over a short name, so a GOPATH-style checkout still serves.
func parsePageArg(arg string) (pageArg, error) {
	if strings.HasPrefix(arg, "https://") || strings.HasPrefix(arg, "http://") {
		return parseURLArg(arg)
	}
	if strings.HasPrefix(arg, "github.com/") {
		if _, err := os.Stat(arg); os.IsNotExist(err) {
			return parseShortName(arg)
		}
	}
	if strings.HasPrefix(arg, "file://") {
		u, err := url.Parse(arg)
		if err != nil {
			return pageArg{}, fmt.Errorf("page URI %q: %w", arg, err)
		}
		if u.Host != "" && u.Host != "localhost" {
			return pageArg{}, fmt.Errorf("page URI %q: a file:// URI must name this machine (no host, or localhost)", arg)
		}
		arg = u.Path
	}
	info, err := os.Stat(arg)
	switch {
	case os.IsNotExist(err) && filepath.Ext(arg) == ".md":
		return pageArg{}, fmt.Errorf("page does not exist: %s", arg)
	case os.IsNotExist(err):
		return pageArg{}, fmt.Errorf("directory does not exist: %s", arg)
	case err != nil:
		return pageArg{}, fmt.Errorf("%s: %w", arg, err)
	}
	if info.IsDir() {
		return pageArg{dir: arg}, nil
	}
	if filepath.Ext(arg) != ".md" {
		return pageArg{}, fmt.Errorf("%s: serve takes a directory or a .md file", arg)
	}
	return pageArg{dir: filepath.Dir(arg), single: filepath.Base(arg)}, nil
}

func parseURLArg(arg string) (pageArg, error) {
	u, err := url.Parse(arg)
	if err != nil {
		return pageArg{}, fmt.Errorf("page URL: %w", err)
	}
	if u.Scheme == "http" && !isLocalHost(u.Hostname()) {
		return pageArg{}, fmt.Errorf("page URL %s: use https; plain http is allowed only to localhost", u.Redacted())
	}
	pin, _ := strings.CutPrefix(u.Fragment, "sha256=")
	if u.Fragment != "" && pin == u.Fragment {
		return pageArg{}, fmt.Errorf("page URL %s: the only fragment allowed is #sha256=HEX", u.Redacted())
	}
	if pin != "" && !sha256Hex.MatchString(pin) {
		return pageArg{}, fmt.Errorf("page URL %s: #sha256= needs 64 hex characters", u.Redacted())
	}
	u.Fragment = ""
	shown := *u
	shown.User = nil
	return pageArg{url: u.String(), shown: shown.String(), pin: strings.ToLower(pin)}, nil
}

// parseShortName expands a short name into its raw GitHub URL and parses that
// as any other page URL, so the fetch, pin and page checks are the same.
// Messages name both forms, and the pin line offers the short one.
func parseShortName(arg string) (pageArg, error) {
	name, frag, _ := strings.Cut(arg, "#")
	raw, err := expandShortName(name)
	var p pageArg
	if err == nil {
		p, err = parseURLArg(raw + "#" + frag) // an empty fragment is no pin
	}
	if err != nil {
		return pageArg{}, fmt.Errorf("short name %s: %w", name, err)
	}
	p.shown = name + " (" + p.shown + ")"
	p.share = name
	return p, nil
}

const shortNameForm = "github.com/OWNER/REPO[/PATH][@REF]"

var (
	githubOwner = regexp.MustCompile(`^[A-Za-z0-9]([A-Za-z0-9-]{0,37}[A-Za-z0-9])?$`)
	githubPart  = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
	githubRef   = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9._-]*$`)
)

// expandShortName turns github.com/OWNER/REPO[/PATH][@REF] into
// https://raw.githubusercontent.com/OWNER/REPO/REF/PATH/index.md. REF
// defaults to HEAD, the repo's default branch, and splits at the last "@".
// A PATH ending in .md names the page itself. Every part is checked against
// a character set, so nothing in the name can change the fetched path.
func expandShortName(name string) (string, error) {
	rest := strings.TrimPrefix(name, "github.com/")
	ref := "HEAD"
	if i := strings.LastIndex(rest, "@"); i >= 0 {
		rest, ref = rest[:i], rest[i+1:]
		if !githubRef.MatchString(ref) || strings.Contains(ref, "..") {
			return "", errors.New("@REF must be a branch, tag or commit of letters, digits, \".\", \"_\" and \"-\", not starting with \".\" or \"-\"")
		}
	}
	parts := strings.Split(rest, "/")
	if len(parts) < 2 {
		return "", fmt.Errorf("a short name is %s", shortNameForm)
	}
	if !githubOwner.MatchString(parts[0]) {
		return "", errors.New("OWNER must be 1-39 letters, digits or \"-\", not starting or ending with \"-\"")
	}
	for _, part := range parts[1:] {
		if !githubPart.MatchString(part) || part == "." || part == ".." {
			return "", fmt.Errorf("each part after OWNER must be letters, digits, \".\", \"_\" or \"-\", and not \".\" or \"..\" (%s)", shortNameForm)
		}
	}
	page := strings.Join(parts[2:], "/")
	switch {
	case page == "":
		page = "index.md"
	case !strings.HasSuffix(page, ".md"):
		page += "/index.md"
	}
	return "https://raw.githubusercontent.com/" + parts[0] + "/" + parts[1] + "/" + ref + "/" + page, nil
}

var sha256Hex = regexp.MustCompile(`^[0-9a-fA-F]{64}$`)

func isLocalHost(h string) bool {
	return h == "localhost" || h == "127.0.0.1" || h == "::1"
}

// fetchRemotePage downloads a page into a new private directory as page.md
// and returns the directory and the content's sha256. The name never comes
// from the URL, so a path in it can't escape the directory.
//
// The URL the operator typed is fetched as given, private addresses
// included, since they chose it. Each redirect must be https and pass
// security.ValidateHTTPURL, so a page host can't bounce the fetch into the
// operator's network.
func fetchRemotePage(p pageArg) (dir, sum string, err error) {
	client := &http.Client{
		Timeout: remoteFetchTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxRemoteRedirects {
				return fmt.Errorf("more than %d redirects", maxRemoteRedirects)
			}
			if req.URL.Scheme != "https" {
				return fmt.Errorf("redirect to %s: only https is allowed", req.URL.Redacted())
			}
			if err := security.ValidateHTTPURL(req.URL.String()); err != nil {
				return fmt.Errorf("redirect to %s: %w", req.URL.Redacted(), err)
			}
			return nil
		},
	}
	resp, err := client.Get(p.url)
	if err != nil {
		var ue *url.Error
		if errors.As(err, &ue) {
			err = ue.Err // *url.Error repeats the URL, credentials included
		}
		return "", "", fmt.Errorf("fetch %s: %w", p.shown, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("fetch %s: %s", p.shown, resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxRemotePage+1))
	if err != nil {
		return "", "", fmt.Errorf("fetch %s: %w", p.shown, err)
	}
	if len(body) > maxRemotePage {
		return "", "", fmt.Errorf("fetch %s: the page is over %d bytes", p.shown, maxRemotePage)
	}
	h := sha256.Sum256(body)
	sum = hex.EncodeToString(h[:])
	if p.pin != "" && p.pin != sum {
		return "", "", fmt.Errorf("fetch %s: content sha256 is %s, not the pinned %s", p.shown, sum, p.pin)
	}
	dir, err = os.MkdirTemp("", "tinkerdown-page-") // mode 0700
	if err != nil {
		return "", "", err
	}
	if err := os.WriteFile(filepath.Join(dir, "page.md"), body, 0o600); err != nil {
		os.RemoveAll(dir)
		return "", "", err
	}
	return dir, sum, nil
}

// checkRemotePage refuses anything a page from a URL may not do. Its
// frontmatter may hold only title, merge sources (with accepts), styling,
// charts and cosmetic block and feature settings; its blocks only lvt and
// markdown. Its markdown output and each lvt template are read with the HTML
// tokenizer for unsafe markup.
func checkRemotePage(page *tinkerdown.Page) error {
	var problems []string
	add := func(format string, args ...any) {
		problems = append(problems, security.CleanPageText(fmt.Sprintf(format, args...)))
	}
	for name, src := range page.Config.Sources {
		if src.Type != "merge" {
			add("source %s (%s): only merge sources are allowed", name, src.Type)
			continue
		}
		if src.Buffer > maxRemoteBuffer {
			add("source %s: buffer %d is over %d", name, src.Buffer, maxRemoteBuffer)
		}
		if d, err := time.ParseDuration(src.Flush); err == nil && d < minRemoteFlush {
			add("source %s: flush %s is under %s", name, src.Flush, minRemoteFlush)
		}
	}
	for key, set := range map[string]bool{
		"description":   page.Description != "",
		"image":         page.Image != "",
		"source_repo":   page.SourceRepo != "",
		"source_path":   page.SourcePath != "",
		"source_ref":    page.SourceRef != "",
		"source_commit": page.SourceCommit != "",
	} {
		if set {
			add("frontmatter %s: only title, sources, styling, charts, blocks and features are allowed", key)
		}
	}
	for name := range page.Config.Actions {
		add("action %s: actions are not allowed", name)
	}
	if len(page.Schedules) > 0 || len(page.Imperatives) > 0 {
		add("schedules are not allowed")
	}
	if len(page.WasmBlocks) > 0 {
		add("wasm blocks are not allowed")
	}
	if len(page.EmbedRoutes) > 0 {
		add("embed-lvt blocks are not allowed")
	}
	if len(page.IncludedFiles) > 0 {
		add("includes are not allowed")
	}
	for id, b := range page.ServerBlocks {
		if b.Metadata["lvt-source"] == "" {
			add("server block %s: only lvt blocks are allowed", id)
		}
	}
	for _, u := range security.UnsafeHTML(page.StaticHTML) {
		add("%s", u)
	}
	for _, id := range slices.Sorted(maps.Keys(page.InteractiveBlocks)) {
		for _, u := range security.UnsafeTemplate(page.InteractiveBlocks[id].Content) {
			add("block %s: %s", id, u)
		}
	}
	if len(problems) == 0 {
		return nil
	}
	return errors.New("a page from a URL can't contain:\n  " + strings.Join(problems, "\n  ") +
		"\nPass sources as --source flags instead, or download the page, review it, and serve the local copy:\n  tinkerdown serve page.md")
}

// remoteConfig is the config for a page from a URL: only what --config names,
// never a tinkerdown.yaml that happens to sit in the current directory.
//
// A relative file or http path in that config is taken from the config
// file's directory, since the page's own directory is a temporary one.
func remoteConfig(configPath, dir string) (*config.Config, error) {
	if configPath == "" {
		return config.LoadFromDir(dir) // dir is the fresh temp directory: defaults
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, err
	}
	base, err := filepath.Abs(filepath.Dir(configPath))
	if err != nil {
		return nil, err
	}
	absSourcePaths(cfg.Sources, base)
	return cfg, nil
}
