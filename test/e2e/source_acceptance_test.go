package e2e_test

import (
	"path/filepath"
	"testing"
)

func TestSourceListEndToEnd(t *testing.T) {
	bin := buildCubby(t)
	tmp := t.TempDir()
	host := filepath.Join(tmp, "host")
	src1 := filepath.Join(tmp, "src1")
	src2 := filepath.Join(tmp, "src2")
	mustWrite(t, filepath.Join(src1, "cubby.toml"), "profiles = [\"work\", \"personal\"]\n")
	mustWrite(t, filepath.Join(src2, "cubby.toml"), "profiles = [\"client\"]\n")
	mustWrite(t, filepath.Join(host, ".cubby.toml"), "[[source]]\nname = \"one\"\npath = \""+src1+"\"\n\n[[source]]\nname = \"two\"\npath = \""+src2+"\"\n")

	jsonResult := runCubby(t, bin, host, "source", "list", "--json")
	if jsonResult.code != 0 {
		t.Fatalf("source list --json code = %d, stdout = %s, stderr = %s", jsonResult.code, jsonResult.stdout, jsonResult.stderr)
	}
	wantJSON := `{"sources":[{"name":"one","path":"` + filepath.ToSlash(filepath.Clean(src1)) + `","profiles":["work","personal"]},{"name":"two","path":"` + filepath.ToSlash(filepath.Clean(src2)) + `","profiles":["client"]}]}` + "\n"
	if jsonResult.stdout != wantJSON {
		t.Fatalf("source list --json stdout = %q, want %q", jsonResult.stdout, wantJSON)
	}

	tableResult := runCubby(t, bin, host, "source", "list")
	if tableResult.code != 0 {
		t.Fatalf("source list code = %d, stdout = %s, stderr = %s", tableResult.code, tableResult.stdout, tableResult.stderr)
	}
	for _, want := range []string{"NAME", "PATH", "PROFILES", "one", filepath.Clean(src1), "work,personal", "two", filepath.Clean(src2), "client"} {
		assertContains(t, tableResult.stdout, want)
	}
}
