package openai

import "testing"

func TestDefaultModels_ContainsCodexAuthModels(t *testing.T) {
	ids := make(map[string]bool, len(DefaultModels))
	for _, model := range DefaultModels {
		ids[model.ID] = true
	}

	for _, id := range []string{"gpt-5.5", "gpt-5.3-codex-spark", "gpt-image-2"} {
		if !ids[id] {
			t.Fatalf("expected %q in OpenAI default model list", id)
		}
	}
}
