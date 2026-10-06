package compiler

import (
	"strings"
	"testing"

	"ango/engine/script"
)

func compileSources(t *testing.T, srcs ...string) (*Program, error) {
	t.Helper()
	var asts []*script.ProgramNode
	for i, src := range srcs {
		ast, err := script.Parse(string(rune('a'+i))+".ango", src)
		if err != nil {
			t.Fatalf("parse %d: %v", i, err)
		}
		asts = append(asts, ast)
	}
	return CompileFiles(asts...)
}

const nsMain = "label start\njump chapter1.intro\n"

func TestNamespacedLabelsAreQualified(t *testing.T) {
	prog, err := compileSources(t, nsMain,
		"namespace chapter1\nlabel intro\njump finale\nlabel finale\nend\n")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"start", "chapter1.intro", "chapter1.finale"} {
		if _, ok := prog.Labels[name]; !ok {
			t.Errorf("missing label %q", name)
		}
	}
	for _, bare := range []string{"intro", "finale"} {
		if _, ok := prog.Labels[bare]; ok {
			t.Errorf("bare name %q leaked into global labels", bare)
		}
	}
}

func TestLocalLabelWinsOverGlobal(t *testing.T) {
	prog, err := compileSources(t,
		"label start\njump chapter1.intro\nlabel finale\nend\n",
		"namespace chapter1\nlabel intro\njump finale\nlabel finale\nend\n")
	if err != nil {
		t.Fatal(err)
	}
	at := prog.Labels["chapter1.intro"]
	if got, want := prog.Code[at].A, prog.Labels["chapter1.finale"]; got != want {
		t.Errorf("jump target = %d, want chapter1.finale (%d)", got, want)
	}
}

func TestBareNameOfNamespacedLabelIsUndefined(t *testing.T) {
	_, err := compileSources(t,
		"label start\njump intro\n",
		"namespace chapter1\nlabel intro\nend\n")
	if err == nil || !strings.Contains(err.Error(), "undefined label") {
		t.Errorf("expected undefined label error, got %v", err)
	}
}

func TestDuplicateQualifiedLabel(t *testing.T) {
	_, err := compileSources(t, nsMain,
		"namespace chapter1\nlabel intro\nend\n",
		"namespace chapter1\nlabel intro\nend\n")
	if err == nil || !strings.Contains(err.Error(), "chapter1.intro") ||
		!strings.Contains(err.Error(), "already defined") {
		t.Errorf("expected duplicate error naming chapter1.intro, got %v", err)
	}
}

func TestSameBareNameInDifferentNamespaces(t *testing.T) {
	_, err := compileSources(t,
		"label start\njump a.intro\n",
		"namespace a\nlabel intro\njump b.intro\n",
		"namespace b\nlabel intro\nend\n")
	if err != nil {
		t.Errorf("same bare name in different namespaces should compile: %v", err)
	}
}

func TestStartMustStayGlobal(t *testing.T) {
	_, err := compileSources(t, "namespace a\nlabel start\nend\n")
	if err == nil || !strings.Contains(err.Error(), "missing entry label") {
		t.Errorf("expected missing entry label, got %v", err)
	}
}
