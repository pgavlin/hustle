package jq

import "testing"

func testShape() Shape {
	return ObjectShape{Fields: map[string]Shape{
		"time":  StringShape{},
		"level": EnumShape{Inner: StringShape{}, Values: []any{"DEBUG", "INFO", "WARN", "ERROR"}},
		"msg":   StringShape{},
		"port":  NumberShape{},
		"host":  StringShape{},
		"headers": ObjectShape{Fields: map[string]Shape{
			"content-type": StringShape{},
			"accept":       StringShape{},
		}},
	}}
}

func containsSuggestion(suggestions []Suggestion, want string) bool {
	for _, s := range suggestions {
		if s.Text == want {
			return true
		}
	}
	return false
}

func TestComplete_DotPrefix(t *testing.T) {
	suggestions := Complete(".h", 2, testShape())
	if !containsSuggestion(suggestions, ".headers") {
		t.Errorf("expected .headers, got %v", suggestions)
	}
	if !containsSuggestion(suggestions, ".host") {
		t.Errorf("expected .host, got %v", suggestions)
	}
	if containsSuggestion(suggestions, ".level") {
		t.Error("should not suggest .level for .h")
	}
}

func TestComplete_DotAlone(t *testing.T) {
	suggestions := Complete(".", 1, testShape())
	if len(suggestions) < 6 {
		t.Errorf("expected at least 6 field suggestions, got %d", len(suggestions))
	}
}

func TestComplete_AfterPipe(t *testing.T) {
	expr := `select(.level == "ERROR") | .h`
	suggestions := Complete(expr, len(expr), testShape())
	if !containsSuggestion(suggestions, `select(.level == "ERROR") | .host`) {
		t.Errorf("expected .host after pipe, got %v", suggestions)
	}
	if !containsSuggestion(suggestions, `select(.level == "ERROR") | .headers`) {
		t.Errorf("expected .headers after pipe, got %v", suggestions)
	}
}

func TestComplete_InsideSelect(t *testing.T) {
	expr := "select(."
	suggestions := Complete(expr, len(expr), testShape())
	if !containsSuggestion(suggestions, "select(.level") {
		t.Errorf("expected .level inside select, got %v", suggestions)
	}
}

func TestComplete_ToEntriesPipe(t *testing.T) {
	expr := "to_entries | .[0] | .k"
	suggestions := Complete(expr, len(expr), testShape())
	if !containsSuggestion(suggestions, "to_entries | .[0] | .key") {
		t.Errorf("expected .key, got %v", suggestions)
	}
}

func TestComplete_BuiltinPrefix(t *testing.T) {
	suggestions := Complete("sel", 3, testShape())
	if !containsSuggestion(suggestions, "select(") {
		t.Errorf("expected select(, got %v", suggestions)
	}
}

func TestComplete_BuiltinHasName(t *testing.T) {
	suggestions := Complete("sel", 3, testShape())
	for _, s := range suggestions {
		if s.Text == "select(" {
			if s.Builtin != "select" {
				t.Errorf("expected Builtin=select, got %q", s.Builtin)
			}
			return
		}
	}
	t.Error("select( not found in suggestions")
}

func TestComplete_FieldHasNoBuiltin(t *testing.T) {
	suggestions := Complete(".h", 2, testShape())
	for _, s := range suggestions {
		if s.Builtin != "" {
			t.Errorf("field suggestion %q should not have Builtin set, got %q", s.Text, s.Builtin)
		}
	}
}

func TestComplete_NestedField(t *testing.T) {
	expr := ".headers.c"
	suggestions := Complete(expr, len(expr), testShape())
	if !containsSuggestion(suggestions, ".headers.content-type") {
		t.Errorf("expected .headers.content-type, got %v", suggestions)
	}
}

func TestComplete_ValueSuggestion(t *testing.T) {
	// .level == <cursor> should suggest enum values
	expr := `.level == `
	suggestions := Complete(expr, len(expr), testShape())
	if !containsSuggestion(suggestions, `.level == "ERROR"`) {
		t.Errorf("expected \"ERROR\" value suggestion, got %v", suggestions)
	}
	if !containsSuggestion(suggestions, `.level == "INFO"`) {
		t.Errorf("expected \"INFO\" value suggestion, got %v", suggestions)
	}
}

func TestComplete_ValueSuggestionPartial(t *testing.T) {
	// .level == "E should suggest "ERROR"
	expr := `.level == "E`
	suggestions := Complete(expr, len(expr), testShape())
	if !containsSuggestion(suggestions, `.level == "ERROR"`) {
		t.Errorf("expected \"ERROR\" for partial, got %v", suggestions)
	}
	if containsSuggestion(suggestions, `.level == "INFO"`) {
		t.Error("should not suggest INFO for \"E prefix")
	}
}

func TestComplete_NoValueForNonEnum(t *testing.T) {
	// .msg == <cursor> should not suggest values (msg has no enum)
	expr := `.msg == `
	suggestions := Complete(expr, len(expr), testShape())
	// Should fall back to regular suggestions (fields/builtins), not enum values
	if containsSuggestion(suggestions, `.msg == "ERROR"`) {
		t.Error("should not suggest enum values for non-enum field")
	}
}

func TestComplete_NoMatch(t *testing.T) {
	suggestions := Complete(".zzz", 4, testShape())
	if len(suggestions) != 0 {
		t.Errorf("expected no suggestions, got %v", suggestions)
	}
}

func TestComplete_FullExpressionFormat(t *testing.T) {
	suggestions := Complete(".h", 2, testShape())
	for _, s := range suggestions {
		if len(s.Text) < 2 || s.Text[:2] != ".h" {
			t.Errorf("suggestion %q doesn't start with .h", s.Text)
		}
	}
}
