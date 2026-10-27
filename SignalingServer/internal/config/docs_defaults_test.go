package config

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// The deployment manuals are the only place an operator learns what a variable
// defaults to, and a wrong default there is worse than a missing one: the
// operator sets nothing and believes a safety property that does not hold. This
// test reads the manuals the way a reader does - as tables of
// `NAME | default | description` - and compares the default column against the
// values Load() actually applies.

var (
	setDefaultRe = regexp.MustCompile(`(?m)viper\.SetDefault\("([A-Z0-9_]+)",\s*([^\n]+?)\)`)
	docRowRe     = regexp.MustCompile("(?m)^\\|\\s*`([A-Z0-9_]+)`\\s*\\|\\s*([^|]*)\\|\\s*([^|]*)\\|")
	envExampleRe = regexp.MustCompile(`(?m)^([A-Z0-9_]+)=(.*)$`)
)

// repoRoot is the checkout root, not the Go module root: this package lives in
// SignalingServer/ (its own module) while the manuals live in docs/ at the top
// of the repository. Walking to .git is what makes that difference explicit
// instead of an accident of how many levels the package happens to be nested.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not locate the repository root above %s", dir)
			return ""
		}
		dir = parent
	}
}

// codeDefaults parses config.go for every viper.SetDefault key. Parsing the
// source rather than calling Load() keeps this test from needing a database,
// flags or a working directory.
func codeDefaults(t *testing.T) map[string]string {
	t.Helper()
	src, err := os.ReadFile(filepath.Join(repoRoot(t), "SignalingServer", "internal", "config", "config.go"))
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]string{}
	for _, m := range setDefaultRe.FindAllStringSubmatch(string(src), -1) {
		out[m[1]] = unquoteDefault(m[2])
	}
	if len(out) == 0 {
		t.Fatal("no viper.SetDefault calls parsed; the regex or the file moved")
	}
	return out
}

func unquoteDefault(raw string) string {
	v := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(raw), ","))
	if s, err := strconv.Unquote(v); err == nil {
		return s
	}
	return v
}

// normaliseDefault makes the two spellings of "nothing configured" comparable:
// the manuals write `(empty)` or `""`, the code writes `""`.
func normaliseDefault(s string) string {
	s = strings.TrimSpace(s)
	// Both manuals are checked, so the Chinese "(空)" counts as the same
	// "nothing configured" as the English "(empty)".
	switch strings.ToLower(s) {
	case "", "(empty)", "empty", "''", `""`, "(空)", "空":
		return ""
	}
	if len(s) >= 2 && strings.HasPrefix(s, `"`) && strings.HasSuffix(s, `"`) {
		s = s[1 : len(s)-1]
	}
	return strings.TrimSpace(s)
}

func docTables(t *testing.T, rel string) []string {
	t.Helper()
	body, err := os.ReadFile(filepath.Join(repoRoot(t), filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	var rows []string
	for _, m := range docRowRe.FindAllStringSubmatch(string(body), -1) {
		rows = append(rows, m[1]+"\t"+strings.TrimSpace(m[2]))
	}
	return rows
}

// TestDeploymentDocsListEveryConfiguredVariable fails when the code grows a
// setting the manuals never mention, which is how a security-relevant knob ends
// up undocumented and therefore left at a default nobody evaluated.
func TestDeploymentDocsListEveryConfiguredVariable(t *testing.T) {
	defaults := codeDefaults(t)
	for _, doc := range []string{
		filepath.Join("docs", "signaling-server-deployment.md"),
		filepath.Join("docs", "信令服务器部署.md"),
	} {
		rows := docTables(t, doc)
		documented := map[string]bool{}
		for _, row := range rows {
			documented[strings.SplitN(row, "\t", 2)[0]] = true
		}
		for key := range defaults {
			if !documented[key] {
				t.Errorf("%s: %s has a viper.SetDefault but is not documented", doc, key)
			}
		}
	}
}

// TestDeploymentDocsDefaultsMatchCode fails when the manual states a default the
// code does not apply. The one documented exception is DB_DRIVER, whose row
// enumerates the two accepted values instead of restating the default; the
// reader still gets the value, so it is allowed through.
func TestDeploymentDocsDefaultsMatchCode(t *testing.T) {
	defaults := codeDefaults(t)
	allowedEnum := map[string]bool{"DB_DRIVER": true}
	for _, doc := range []string{
		filepath.Join("docs", "signaling-server-deployment.md"),
		filepath.Join("docs", "信令服务器部署.md"),
	} {
		for _, row := range docTables(t, doc) {
			parts := strings.SplitN(row, "\t", 2)
			key, stated := parts[0], normaliseDefault(parts[1])
			want, ok := defaults[key]
			if !ok || allowedEnum[key] {
				continue
			}
			if stated != normaliseDefault(want) {
				t.Errorf("%s: %s documents default %q but config.go applies %q",
					doc, key, parts[1], want)
			}
		}
	}
}

// TestDeploymentDocsDescribeUndocumentedBehaviour guards the claims that are
// about what a setting *does*, not what it defaults to. Each of these is a
// sentence an operator would act on, and each was once wrong in the manual.
func TestDeploymentDocsDescribeUndocumentedBehaviour(t *testing.T) {
	body, err := os.ReadFile(filepath.Join(repoRoot(t), "docs", "signaling-server-deployment.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)

	claims := []struct {
		what    string
		mustSay string
	}{
		{
			what:    "METRICS_TOKEN empty disables the endpoint rather than exposing it unauthenticated",
			mustSay: "disables the endpoint entirely",
		},
		{
			what:    "REVOKE_COOLDOWN_SECONDS zero means immediate reconnects are allowed",
			mustSay: "0 disables it",
		},
		{
			what:    "RECORDING_URL_SECRET falls back to API_KEY",
			mustSay: "Falls back to `API_KEY`",
		},
		{
			what:    "TRUSTED_PROXIES empty is not 'trust nobody'",
			mustSay: "Empty = loopback and private ranges",
		},
	}
	for _, c := range claims {
		if !strings.Contains(text, c.mustSay) {
			t.Errorf("the manual no longer states that %s (expected the phrase %q)",
				c.what, c.mustSay)
		}
	}
}

// TestEnvExampleDefaultsMatchCode covers .env.example, the file a new
// deployment copies verbatim. A value there that the code does not default to
// is the most expensive kind of drift: start.sh creates the database it reads
// from DB_NAME, so a template naming a database nobody creates fails on first
// boot with a connection error that points at the server rather than the file
// the operator was told to copy.
//
// Only the infrastructure keys are compared. A template is allowed - expected,
// even - to hand an operator working values for optional features: TURN_URLS
// points at a real relay, RUN_AUTO_MIGRATE is on so a fresh database gets its
// schema. Those are choices, not drift. The database and server coordinates
// are not choices: they have to be the same values start.sh and the server
// agree on, or the first boot fails.
func TestEnvExampleDefaultsMatchCode(t *testing.T) {
	body, err := os.ReadFile(filepath.Join(repoRoot(t), "SignalingServer", ".env.example"))
	if err != nil {
		t.Fatal(err)
	}
	infrastructure := []string{
		"SERVER_HOST", "SERVER_PORT",
		"DB_DRIVER", "DB_HOST", "DB_PORT", "DB_USER", "DB_NAME",
		"REDIS_HOST", "REDIS_PORT",
	}
	defaults := codeDefaults(t)
	stated := map[string]string{}
	for _, m := range envExampleRe.FindAllStringSubmatch(string(body), -1) {
		stated[m[1]] = strings.TrimSpace(m[2])
	}
	for _, key := range infrastructure {
		want, ok := defaults[key]
		if !ok {
			t.Errorf("%s has no viper.SetDefault; it is not something a template can rely on", key)
			continue
		}
		got, ok := stated[key]
		if !ok {
			t.Errorf(".env.example does not set %s, so a copied deployment falls back to the code default %q", key, want)
			continue
		}
		if got != want {
			t.Errorf(".env.example sets %s=%q but config.go defaults to %q and start.sh creates %s from that default; "+
				"a deployment that copies the file will name a database nobody creates",
				key, got, want, want)
		}
	}
}
