package openai

import "testing"

func TestDefaultModels_ContainsCodexAuthModels(t *testing.T) {
	ids := make(map[string]bool, len(DefaultModels))
	for _, model := range DefaultModels {
		ids[model.ID] = true
	}

	for _, id := range []string{"gpt-5.3-codex", "gpt-5.3-codex-spark", "gpt-5.4", "gpt-5.4-mini", "gpt-5.5", "gpt-image-2"} {
		if !ids[id] {
			t.Fatalf("expected %q in OpenAI default model list", id)
		}
	}
}

func TestDefaultModels_OnlyExposeProductionAllowlist(t *testing.T) {
	want := []string{"gpt-5.3-codex", "gpt-5.3-codex-spark", "gpt-5.4", "gpt-5.4-mini", "gpt-5.5", "gpt-image-2"}
	got := DefaultModelIDs()
	if len(got) != len(want) {
		t.Fatalf("unexpected OpenAI default model count: got %d want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected OpenAI default model at %d: got %q want %q", i, got[i], want[i])
		}
	}
}

func TestDefaultTestModel(t *testing.T) {
	if DefaultTestModel != "gpt-5.5" {
		t.Fatalf("unexpected default test model: got %q want %q", DefaultTestModel, "gpt-5.5")
	}
}
