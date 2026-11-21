package security

import "testing"

func TestUnsafeMarkupTemplateActionsInTags(t *testing.T) {
	refused := []string{
		`<{{"meta"}} http-equiv="refresh" content="0;url=https://evil.example">`,
		`<img src="x" {{"onerror"}}="alert(1)">`,
		`<a href={{.url}}>x</a>`,
		`<{{.tag}}>`,
		`<img x"{{"onerror"}}="alert(1)">`,
		`< {{- "meta"}} http-equiv="refresh">`,
		`<p>{{.a -}} x</p>`,
		`<!-- {{.x}} -->`,
		`<img alt=x{{.y}}>`,
		`<img alt= {{.y}}>`,
	}
	for _, src := range refused {
		if found := UnsafeMarkup(src); len(found) == 0 {
			t.Errorf("%s: want a refusal", src)
		}
	}
	allowed := []string{
		`<li data-key="{{.seq}}" class="lvl-{{.level}} src-{{.source}}">{{.msg}}</li>`,
		`{{if .Paused}}<button name="Resume">Resume</button>{{else}}<button name="Pause">Pause</button>{{end}}`,
		`<span class='x {{.y}}'>{{len .Data}} lines</span>`,
		"a < b and {{.x}} > c",
		`<input value="it's {{.x}}" placeholder='say "{{.y}}"'>`,
		`<!-- a note --><p>{{.x}}</p>`,
	}
	for _, src := range allowed {
		if found := UnsafeMarkup(src); len(found) != 0 {
			t.Errorf("%s: refused for %v", src, found)
		}
	}
}
