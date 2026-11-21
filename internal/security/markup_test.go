package security

import "testing"

func TestUnsafeTemplateActionsOutsideValues(t *testing.T) {
	refused := []string{
		`<{{"meta"}} http-equiv="refresh" content="0;url=https://evil.example">`,
		`<img src="x" {{"onerror"}}="alert(1)">`,
		`<{{.tag}}>`,
		`<img x"{{"onerror"}}="alert(1)">`,
		`< {{- "meta"}} http-equiv="refresh">`,
		`<p>{{.a -}} x</p>`,
		`<!-- {{.x}} -->`,
		`<textarea>{{.x}}</textarea><{{"meta"}}>`,
		`<style>{{.x}}</style>`,
		`<title>{{.x}}</title>`,
		`<!{{.x}}>`,
		`<?{{.x}}>`,
		`<!-- a --!> <{{"meta"}} http-equiv="refresh">`,
		`<script>x</script>`,
		"<a href=\" java\tscript:alert(1)\">x</a>",
		`<img src=x onerror="a()">`,
		`<img {{.y}}=x>`,
	}
	for _, src := range refused {
		if found := UnsafeTemplate(src); len(found) == 0 {
			t.Errorf("%s: want a refusal", src)
		}
	}
	allowed := []string{
		`<li data-key="{{.seq}}" class="lvl-{{.level}} src-{{.source}}">{{.msg}}</li>`,
		`{{if .Paused}}<button name="Resume">Resume</button>{{else}}<button name="Pause">Pause</button>{{end}}`,
		`<span class='x {{.y}}'>{{len .Data}} lines</span>`,
		"a < b and {{.x}} > c",
		"<a href={{.url}}>x</a>", // an unquoted value: html/template escapes it
		`<input value="it's {{.x}}" placeholder='say "{{.y}}"'>`,
		`<!-- a note --><p>{{.x}}</p>`,
	}
	for _, src := range allowed {
		if found := UnsafeTemplate(src); len(found) != 0 {
			t.Errorf("%s: refused for %v", src, found)
		}
	}
}

func TestUnsafeHTML(t *testing.T) {
	if found := UnsafeHTML(`<p>Logs</p><style>.log{color:red}</style><a href="https://example.com">x</a>`); len(found) != 0 {
		t.Errorf("plain page refused for %v", found)
	}
	for _, doc := range []string{`<script>x</script>`, `<meta http-equiv="refresh">`, `<p onclick="x()">`, `<iframe srcdoc="x">`, `<a href="JAVASCRIPT:x">`} {
		if len(UnsafeHTML(doc)) == 0 {
			t.Errorf("%s: want a refusal", doc)
		}
	}
}
