package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/livetemplate/tinkerdown"
	"github.com/livetemplate/tinkerdown/internal/config"
)

// TestGetEffectiveSourcePrecedence covers the three-tier lookup order, and in
// particular the tier that exists for security rather than convenience.
//
// A page's frontmatter can declare its own sources. Without pinning, a generated page
// could reference an approved source name while defining that name as something the
// operator never approved — defeating any check that reasons about names alone, since
// the name really is in the approved set. Pinning makes an approved name mean one
// thing no matter what the page says.
//
// The two lower tiers (frontmatter beats site config) are long-standing behavior and
// must not change for projects that declare no generation block.
func TestGetEffectiveSourcePrecedence(t *testing.T) {
	const name = "requests"

	approved := config.SourceConfig{Type: "json", File: "./approved.json"}
	pageDefined := tinkerdown.SourceConfig{Type: "json", File: "./page.json"}

	pageWith := func(srcs map[string]tinkerdown.SourceConfig) *tinkerdown.Page {
		return &tinkerdown.Page{Config: tinkerdown.PageConfig{Sources: srcs}}
	}

	tests := []struct {
		name     string
		config   *config.Config
		page     *tinkerdown.Page
		wantFile string
		wantOK   bool
	}{
		{
			name: "approved source is pinned against a shadowing page",
			config: &config.Config{
				Sources:    map[string]config.SourceConfig{name: approved},
				Generation: &config.GenerationConfig{Sources: []string{name}},
			},
			page:     pageWith(map[string]tinkerdown.SourceConfig{name: pageDefined}),
			wantFile: "./approved.json",
			wantOK:   true,
		},
		{
			name: "without a generation block frontmatter still wins",
			config: &config.Config{
				Sources: map[string]config.SourceConfig{name: approved},
			},
			page:     pageWith(map[string]tinkerdown.SourceConfig{name: pageDefined}),
			wantFile: "./page.json",
			wantOK:   true,
		},
		{
			name: "a manifest does not pin names outside its approved set",
			config: &config.Config{
				Sources:    map[string]config.SourceConfig{name: approved},
				Generation: &config.GenerationConfig{Sources: []string{"something-else"}},
			},
			page:     pageWith(map[string]tinkerdown.SourceConfig{name: pageDefined}),
			wantFile: "./page.json",
			wantOK:   true,
		},
		{
			name: "approved name resolves from site config when the page declares nothing",
			config: &config.Config{
				Sources:    map[string]config.SourceConfig{name: approved},
				Generation: &config.GenerationConfig{Sources: []string{name}},
			},
			page:     pageWith(nil),
			wantFile: "./approved.json",
			wantOK:   true,
		},
		{
			name: "site config still serves as the fallback tier",
			config: &config.Config{
				Sources: map[string]config.SourceConfig{name: approved},
			},
			page:     pageWith(nil),
			wantFile: "./approved.json",
			wantOK:   true,
		},
		{
			name: "approving a name the project never defined does not invent a source",
			config: &config.Config{
				Generation: &config.GenerationConfig{Sources: []string{name}},
			},
			page:   pageWith(nil),
			wantOK: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := &WebSocketHandler{config: tc.config, page: tc.page}
			got, ok := h.getEffectiveSource(name)
			if ok != tc.wantOK {
				t.Fatalf("getEffectiveSource(%q) ok = %v, want %v", name, ok, tc.wantOK)
			}
			if ok && got.File != tc.wantFile {
				t.Errorf("resolved to %q, want %q", got.File, tc.wantFile)
			}
		})
	}
}

// TestGetPageActionsPrecedence covers the action side, which needed two changes rather
// than one.
//
// Actions never had the site-config fallback sources have: a page could only invoke
// actions declared in its own frontmatter. That left `generation.actions` inert — it
// named a surface a generated page had no way to reach, forcing it to declare those
// actions itself, which is what approval exists to prevent. So approved actions must
// both become *reachable* and be *pinned* against redefinition.
func TestGetPageActionsPrecedence(t *testing.T) {
	const name = "approve-request"

	site := &config.Action{Kind: "sql", Statement: "SITE"}
	page := tinkerdown.Action{Kind: "sql", Statement: "PAGE"}

	handler := func(gen *config.GenerationConfig, pageActions map[string]tinkerdown.Action) *WebSocketHandler {
		return &WebSocketHandler{
			config: &config.Config{
				Actions:    map[string]*config.Action{name: site},
				Generation: gen,
			},
			page: &tinkerdown.Page{Config: tinkerdown.PageConfig{Actions: pageActions}},
		}
	}

	t.Run("approved site action is reachable without the page declaring it", func(t *testing.T) {
		h := handler(&config.GenerationConfig{Actions: []string{name}}, nil)
		got := h.getPageActions()
		if got[name] == nil {
			t.Fatalf("approved action %q unreachable; generation.actions would be inert", name)
		}
		if got[name].Statement != "SITE" {
			t.Errorf("resolved to %q, want SITE", got[name].Statement)
		}
	})

	t.Run("approved action is pinned against a shadowing page", func(t *testing.T) {
		h := handler(&config.GenerationConfig{Actions: []string{name}},
			map[string]tinkerdown.Action{name: page})
		if got := h.getPageActions()[name]; got == nil || got.Statement != "SITE" {
			t.Errorf("page redefinition won; approved actions must be pinned")
		}
	})

	t.Run("without a generation block site actions stay unreachable", func(t *testing.T) {
		// Long-standing behavior: actions written for schedules or webhooks are not
		// callable from arbitrary pages. Only approval opts an action in.
		h := handler(nil, nil)
		if got := h.getPageActions(); got[name] != nil {
			t.Errorf("site action leaked to a page with no generation block")
		}
	})

	t.Run("unapproved page actions still work", func(t *testing.T) {
		h := handler(&config.GenerationConfig{Actions: []string{name}},
			map[string]tinkerdown.Action{"page-only": page})
		got := h.getPageActions()
		if got["page-only"] == nil || got["page-only"].Statement != "PAGE" {
			t.Errorf("a page action outside the approved set should be unaffected")
		}
	})

	t.Run("approving an undefined action does not synthesise one", func(t *testing.T) {
		h := &WebSocketHandler{
			config: &config.Config{Generation: &config.GenerationConfig{Actions: []string{"typo"}}},
			page:   &tinkerdown.Page{},
		}
		if got := h.getPageActions(); got["typo"] != nil {
			t.Errorf("approving an undefined name invented an action")
		}
	})
}

// TestApprovedMergePinsItsInputs covers the hole pinning alone leaves for a
// merge: an approved merge is pinned, but if its inputs resolved with page
// precedence, a page could redefine a non-approved input name and change what
// the approved merge runs.
func TestApprovedMergePinsItsInputs(t *testing.T) {
	manifestWeb := config.SourceConfig{Type: "stream", Cmd: "docker logs -f web"}
	pageWeb := tinkerdown.SourceConfig{Type: "stream", Cmd: "cat /etc/shadow"}
	cfg := &config.Config{
		Sources: map[string]config.SourceConfig{
			"web": manifestWeb,
			"all": {Type: "merge", Inputs: []string{"web"}},
		},
		Generation: &config.GenerationConfig{Sources: []string{"all"}},
	}
	h := &WebSocketHandler{
		config: cfg,
		page:   &tinkerdown.Page{Config: tinkerdown.PageConfig{Sources: map[string]tinkerdown.SourceConfig{"web": pageWeb}}},
	}

	if got, _ := mergeInputLookup(h.config, h.page, "all")("web"); got.Cmd != manifestWeb.Cmd {
		t.Fatalf("approved merge resolved input web to %q, want the manifest's %q", got.Cmd, manifestWeb.Cmd)
	}
	// A merge outside the approved set keeps the usual precedence.
	if got, _ := mergeInputLookup(h.config, h.page, "other")("web"); got.Cmd != pageWeb.Cmd {
		t.Fatalf("unapproved merge resolved input web to %q, want the page's %q", got.Cmd, pageWeb.Cmd)
	}
}

// TestApprovedWildcardMergeTakesOnlyManifestSources: an approved merge with
// inputs ["*"] must not pick up page or flag sources, or a page could slip a
// command into a merge the operator approved.
func TestApprovedWildcardMergeTakesOnlyManifestSources(t *testing.T) {
	cfg := &config.Config{
		Sources: map[string]config.SourceConfig{
			"web": {Type: "stream", Cmd: "docker logs -f web"},
			"all": {Type: "merge", Inputs: []string{"*"}},
		},
		FlagSources: map[string]config.SourceConfig{"flagged": {Type: "stream", File: "/tmp/x.log"}},
		Generation:  &config.GenerationConfig{Sources: []string{"all"}},
	}
	page := &tinkerdown.Page{Config: tinkerdown.PageConfig{Sources: map[string]tinkerdown.SourceConfig{
		"sneaky": {Type: "stream", Cmd: "cat /etc/shadow"},
	}}}
	h := &WebSocketHandler{config: cfg, page: page}

	got, _ := ResolveSource(h.config, h.page, "all")
	if len(got.Inputs) != 1 || got.Inputs[0] != "web" {
		t.Fatalf("approved [\"*\"] merge expanded to %v, want only [web]", got.Inputs)
	}

	// The same merge outside the approved set takes every stream it can see.
	cfg.Generation = nil
	got, _ = ResolveSource(h.config, h.page, "all")
	if want := []string{"flagged", "sneaky", "web"}; !slices.Equal(got.Inputs, want) {
		t.Fatalf("unapproved [\"*\"] merge expanded to %v, want %v", got.Inputs, want)
	}
}

// TestFlagSourceOutranksPageAndSite: `serve --source` is the operator's choice
// for this run, so it replaces a page or site source of the same name.
func TestFlagSourceOutranksPageAndSite(t *testing.T) {
	cfg := &config.Config{
		Sources:     map[string]config.SourceConfig{"api": {Type: "stream", File: "site.log"}},
		FlagSources: map[string]config.SourceConfig{"api": {Type: "stream", File: "flag.log"}},
	}
	page := &tinkerdown.Page{Config: tinkerdown.PageConfig{Sources: map[string]tinkerdown.SourceConfig{
		"api": {Type: "stream", File: "page.log"},
	}}}
	h := &WebSocketHandler{config: cfg, page: page}
	if got, _ := h.getEffectiveSource("api"); got.File != "flag.log" {
		t.Fatalf("api resolved to %q, want the flag's flag.log", got.File)
	}
}

func TestWildcardMergeHonoursAcceptsKinds(t *testing.T) {
	cfg := &config.Config{FlagSources: map[string]config.SourceConfig{
		"cmd": {Type: "stream", Cmd: "echo hi"},
		"log": {Type: "stream", File: "a.log"},
	}}
	page := &tinkerdown.Page{Config: tinkerdown.PageConfig{Sources: map[string]tinkerdown.SourceConfig{
		"all": {Type: "merge", Inputs: []string{"*"}, Accepts: &tinkerdown.SourceAccepts{Kinds: []string{"file"}}},
	}}}
	h := &WebSocketHandler{config: cfg, page: page}
	got, _ := ResolveSource(h.config, h.page, "all")
	if !slices.Equal(got.Inputs, []string{"log"}) || got.Accepts == nil {
		t.Fatalf("file-only merge expanded to %v (accepts %v), want [log]", got.Inputs, got.Accepts)
	}
}

func TestHostAllowed(t *testing.T) {
	srv := &Server{config: &config.Config{Server: config.ServerConfig{Host: "logs.internal"}}}
	request := func(local, host string) *http.Request {
		r := httptest.NewRequest("GET", "/", nil)
		r.Host = host
		if local != "" {
			addr, _ := net.ResolveTCPAddr("tcp", local)
			r = r.WithContext(context.WithValue(r.Context(), http.LocalAddrContextKey, addr))
		}
		return r
	}
	cases := []struct {
		local, host string
		want        bool
	}{
		{"127.0.0.1:8080", "localhost:8080", true},
		{"127.0.0.1:8080", "LocalHost.:8080", true},
		{"127.0.0.1:8080", "127.0.0.1:8080", true},
		{"[::1]:8080", "[::1]:8080", true},
		{"127.0.0.1:8080", "host.docker.internal:8080", true},
		{"127.0.0.1:8080", "logs.internal:8080", true}, // the configured host
		{"127.0.0.1:8080", "evil.example:8080", false}, // DNS rebinding
		{"[::1]:8080", "evil.example", false},
		{"192.168.1.5:8080", "anything.example", true}, // exposed on purpose with --host
		{"", "example.com", true},                      // no network connection
	}
	for _, c := range cases {
		if got := srv.hostAllowed(request(c.local, c.host)); got != c.want {
			t.Errorf("local %s, Host %s: allowed = %v, want %v", c.local, c.host, got, c.want)
		}
	}
}

func TestBlockErrorShowsOnlyValidationMessages(t *testing.T) {
	err := config.ValidateStreamSource("api", config.SourceConfig{Type: "stream", Cmd: "x", File: "y"}, nil)
	if got := blockErrorFor("api", err).Error; !strings.Contains(got, "exactly one of cmd, file, http or sql") {
		t.Errorf("validation error shown as %q", got)
	}
	secret := fmt.Errorf("start %q: exit 1", "curl -H 'Authorization: token SECRET' api")
	if got := blockErrorFor("api", secret).Error; strings.Contains(got, "SECRET") || !strings.Contains(got, "server log") {
		t.Errorf("other error shown as %q; it must not carry the command", got)
	}
}

// pageSourceConfig copies a page's frontmatter source field by field. A field
// added to tinkerdown.SourceConfig and not copied would silently be dropped.
func TestPageSourceConfigCopiesEveryField(t *testing.T) {
	pageType := reflect.TypeOf(tinkerdown.SourceConfig{})
	src := reflect.New(pageType).Elem()
	for i := 0; i < pageType.NumField(); i++ {
		f := src.Field(i)
		switch f.Kind() {
		case reflect.String:
			f.SetString("x")
		case reflect.Bool:
			f.SetBool(true)
		case reflect.Int:
			f.SetInt(1)
		case reflect.Slice:
			f.Set(reflect.MakeSlice(f.Type(), 1, 1))
		case reflect.Map:
			f.Set(reflect.MakeMap(f.Type()))
		case reflect.Pointer:
			f.Set(reflect.New(f.Type().Elem()))
		}
	}
	got := reflect.ValueOf(pageSourceConfig(src.Interface().(tinkerdown.SourceConfig)))
	for i := 0; i < pageType.NumField(); i++ {
		name := pageType.Field(i).Name
		out := got.FieldByName(name)
		if !out.IsValid() {
			t.Errorf("config.SourceConfig has no field %s", name)
		} else if out.IsZero() {
			t.Errorf("pageSourceConfig drops %s", name)
		}
	}
}

// The WebSocket URL carries the page URL's query, and the page path wins over
// a page= in the query.
func TestPageWSURL(t *testing.T) {
	if got := pageWSURL("ws", "h", "/a b"); got != "ws://h/ws?page=%2Fa+b" {
		t.Fatalf("pageWSURL = %s", got)
	}
}

// A ["*"] merge doesn't take the inputs of a merge with params: run without
// their TD_ values they would fail.
func TestResolveSourceReservesParamsInputs(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.FlagSources = map[string]config.SourceConfig{
		"docker": {Type: "stream", Cmd: "docker events"},
		"logs":   {Type: "stream", Cmd: `docker logs -f -- "$TD_name"`},
	}
	page := &tinkerdown.Page{Config: tinkerdown.PageConfig{Sources: map[string]tinkerdown.SourceConfig{
		"containers": {Type: "merge", Inputs: []string{"*"}},
		"logview":    {Type: "merge", Inputs: []string{"logs"}, Params: map[string]tinkerdown.SourceParam{"name": {In: "containers.fields.name"}}},
	}}}
	src, _ := ResolveSource(cfg, page, "containers")
	if !slices.Equal(src.Inputs, []string{"docker"}) {
		t.Fatalf("containers inputs = %v, want [docker]", src.Inputs)
	}
}

// Only a page with a block on a merge with params asks the client to change
// the query without a page load.
func TestLiveQueryMeta(t *testing.T) {
	cfg := config.DefaultConfig()
	page := &tinkerdown.Page{
		Config: tinkerdown.PageConfig{Sources: map[string]tinkerdown.SourceConfig{
			"containers": {Type: "merge", Inputs: []string{"*"}},
			"logview":    {Type: "merge", Inputs: []string{"logs"}, Params: map[string]tinkerdown.SourceParam{"name": {In: "containers.fields.name"}}},
		}},
		ServerBlocks: map[string]*tinkerdown.ServerBlock{"lvt-0": {Metadata: map[string]string{"lvt-source": "containers"}}},
	}
	if got := liveQueryMeta(cfg, page); got != "" {
		t.Errorf("no block shows logview, got %q", got)
	}
	page.ServerBlocks["lvt-1"] = &tinkerdown.ServerBlock{Metadata: map[string]string{"lvt-source": "logview"}}
	if got := liveQueryMeta(cfg, page); !strings.Contains(got, `name="tinkerdown-live-query"`) {
		t.Errorf("a block shows logview, got %q", got)
	}
}

// An --action that replaces a page's action keeps the page's values, whatever
// else the page set, so the checks never drop away: with a confirm, say, the
// page action isn't values-only, and its values must still apply. The flag
// action itself, shared by every page, is left without them.
func TestFlagActionKeepsThePagesValues(t *testing.T) {
	values := map[string]tinkerdown.SourceParam{"path": {In: "files.fields.path"}}
	flag := &config.Action{Kind: "exec", Cmd: `git add -- "$TD_path"`, FromFlag: true}
	for name, page := range map[string]tinkerdown.Action{
		"values only":        {Values: values},
		"values and confirm": {Values: values, Confirm: "Stage it?"},
	} {
		h := &WebSocketHandler{
			config: &config.Config{FlagActions: map[string]*config.Action{"stage": flag}},
			page:   &tinkerdown.Page{Config: tinkerdown.PageConfig{Actions: map[string]tinkerdown.Action{"stage": page}}},
		}
		got := h.getPageActions()["stage"]
		if got == nil || got.Cmd != flag.Cmd || got.Values["path"].In != "files.fields.path" {
			t.Errorf("%s: the --action lost the page's values: %+v", name, got)
		}
	}
	if flag.Values != nil {
		t.Error("the shared flag action took a page's values")
	}
}
