package tinkerdown

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/mod/zip"
)

// TestModuleZipKeepsEmbeddedFiles fails if the Go module zip, which is what
// `go install …@version` downloads, would leave out a file a //go:embed
// pattern uses. A clean checkout still builds, so only this catches it before
// a release: the zip drops every directory named vendor, which broke
// v0.7.2 (#364).
func TestModuleZipKeepsEmbeddedFiles(t *testing.T) {
	root, err := filepath.Abs(".")
	if err != nil {
		t.Fatal(err)
	}
	embedded := embeddedFiles(t, root)
	if len(embedded) == 0 {
		t.Fatal("go list found no embedded files; the check would pass vacuously")
	}

	files, err := zip.CheckDir(root)
	if err != nil {
		t.Fatalf("the module's files don't make a valid module zip: %v", err)
	}
	for _, f := range files.Omitted {
		for _, e := range embedded {
			if e == f.Path || strings.HasPrefix(e, f.Path+string(filepath.Separator)) {
				rel, _ := filepath.Rel(root, e)
				t.Errorf("%s is embedded but left out of the module zip: %v", rel, f.Err)
			}
		}
	}
}

// embeddedFiles lists, as absolute paths, every file a //go:embed pattern in
// the module matches.
func embeddedFiles(t *testing.T, root string) []string {
	t.Helper()
	cmd := exec.Command("go", "list", "-json=Dir,EmbedFiles,TestEmbedFiles", "./...")
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "GOWORK=off")
	out, err := cmd.Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			t.Fatalf("go list: %v\n%s", err, ee.Stderr)
		}
		t.Fatalf("go list: %v", err)
	}
	var files []string
	dec := json.NewDecoder(strings.NewReader(string(out)))
	for {
		var pkg struct {
			Dir                        string
			EmbedFiles, TestEmbedFiles []string
		}
		if err := dec.Decode(&pkg); err == io.EOF {
			break
		} else if err != nil {
			t.Fatalf("decode go list output: %v", err)
		}
		for _, f := range append(pkg.EmbedFiles, pkg.TestEmbedFiles...) {
			files = append(files, filepath.Join(pkg.Dir, f))
		}
	}
	return files
}
