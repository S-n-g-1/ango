package script

import "testing"

func TestNamespaceAssignedToLabels(t *testing.T) {
	prog := mustParse(t, "namespace a\ndefault x = 1\nlabel l\njump a.b\n")
	if prog.Namespace != "a" {
		t.Errorf("program namespace = %q, want a", prog.Namespace)
	}
	if len(prog.Labels) != 1 || prog.Labels[0].Namespace != "a" {
		t.Fatalf("label namespace not set: %+v", prog.Labels)
	}
	j, ok := prog.Labels[0].Body[0].(*JumpNode)
	if !ok || j.Label != "a.b" {
		t.Errorf("jump = %#v, want label a.b", prog.Labels[0].Body[0])
	}
}

func TestNoNamespaceStaysGlobal(t *testing.T) {
	prog := mustParse(t, "label l\nend\n")
	if prog.Namespace != "" || prog.Labels[0].Namespace != "" {
		t.Errorf("expected empty namespace, got %q / %q", prog.Namespace, prog.Labels[0].Namespace)
	}
}

func TestNamespaceSyntaxErrors(t *testing.T) {
	cases := map[string]string{
		"duplicate":        "namespace a\nnamespace b\nlabel l\nend\n",
		"after default":    "default x = 1\nnamespace a\nlabel l\nend\n",
		"after label":      "label l\nend\nnamespace a\n",
		"missing name":     "namespace\nlabel l\nend\n",
		"dotted name":      "namespace a.b\nlabel l\nend\n",
		"dot without name": "label l\njump a.\n",
		"leading dot":      "label l\njump .a\n",
	}
	for name, src := range cases {
		if _, err := Parse("t.ango", src); err == nil {
			t.Errorf("%s: expected a syntax error", name)
		}
	}
}

