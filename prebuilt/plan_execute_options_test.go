package prebuilt

import "testing"

func TestApplyManusOptions(t *testing.T) {
	t.Run("options fill unset fields", func(t *testing.T) {
		got := applyManusOptions(ManusConfig{},
			WithWorkDir("/tmp/w"),
			WithPlanPath("/tmp/plan.md"),
			WithNotesPath("/tmp/notes.md"),
			WithOutputPath("/tmp/out.md"),
			WithAutoSave(true),
			WithVerbose(true),
		)
		if got.WorkDir != "/tmp/w" || got.PlanPath != "/tmp/plan.md" ||
			got.NotesPath != "/tmp/notes.md" || got.OutputPath != "/tmp/out.md" ||
			!got.AutoSave || !got.Verbose {
			t.Fatalf("options not folded into config: %+v", got)
		}
	})

	t.Run("config takes precedence over options", func(t *testing.T) {
		cfg := ManusConfig{WorkDir: "/keep", AutoSave: true, Verbose: true}
		got := applyManusOptions(cfg, WithWorkDir("/override"), WithAutoSave(false))
		if got.WorkDir != "/keep" {
			t.Errorf("WorkDir = %q, want /keep", got.WorkDir)
		}
		if !got.AutoSave {
			t.Error("AutoSave should stay true when config set it")
		}
	})

	t.Run("no options is a no-op", func(t *testing.T) {
		cfg := ManusConfig{WorkDir: "/x"}
		if got := applyManusOptions(cfg); got != cfg {
			t.Errorf("no-op changed config: %+v", got)
		}
	})
}

func TestApplyPEVOptions(t *testing.T) {
	t.Run("options fill unset fields", func(t *testing.T) {
		got := applyPEVOptions(PEVAgentConfig{},
			WithMaxRetries(5),
			WithSystemMessage("sys"),
			WithVerificationPrompt("verify"),
			WithVerbose(true),
		)
		if got.MaxRetries != 5 || got.SystemMessage != "sys" ||
			got.VerificationPrompt != "verify" || !got.Verbose {
			t.Fatalf("options not folded into config: %+v", got)
		}
	})

	t.Run("config takes precedence over options", func(t *testing.T) {
		cfg := PEVAgentConfig{MaxRetries: 2, SystemMessage: "keep"}
		got := applyPEVOptions(cfg, WithMaxRetries(9), WithSystemMessage("override"))
		if got.MaxRetries != 2 || got.SystemMessage != "keep" {
			t.Errorf("config not preserved: %+v", got)
		}
	})
}
