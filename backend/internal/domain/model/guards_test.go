package model

import "testing"

func TestParsePipelineConfigYAML_GuardsDefaultOnFail(t *testing.T) {
	yaml := []byte(`groups:
  - id: dev
    name: Development
    guards:
      - kind: log_silence
        threshold: 120
      - kind: cost_batch
        max: 5
        on_fail: fail
    steps:
      - id: dev-agent
        name: dev-agent
        action_type: agent_run
        guards:
          - kind: wallclock
            max: 1800
`)

	cfg, err := ParsePipelineConfigYAML(yaml)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	grp := cfg.Groups[0]
	if len(grp.Guards) != 2 {
		t.Fatalf("expected 2 stage guards, got %d", len(grp.Guards))
	}
	// log_silence has no on_fail → defaults to halt-gate.
	if grp.Guards[0].Kind != GuardLogSilence || grp.Guards[0].OnFail != GuardOnFailHaltGate {
		t.Errorf("log_silence guard not normalized: %+v", grp.Guards[0])
	}
	if grp.Guards[0].Threshold != 120 {
		t.Errorf("expected threshold 120, got %d", grp.Guards[0].Threshold)
	}
	// cost_batch keeps its explicit on_fail.
	if grp.Guards[1].OnFail != GuardOnFailFail {
		t.Errorf("expected cost_batch on_fail=fail, got %q", grp.Guards[1].OnFail)
	}
	if grp.Guards[1].Max != 5 {
		t.Errorf("expected cost_batch max 5, got %v", grp.Guards[1].Max)
	}

	// Step-level guard normalized too.
	step := grp.Steps[0]
	if len(step.Guards) != 1 || step.Guards[0].OnFail != GuardOnFailHaltGate {
		t.Errorf("step guard not normalized: %+v", step.Guards)
	}
}

func TestGuardsForStep(t *testing.T) {
	cfg := &PipelineConfigYAML{
		Groups: []PipelineGroup{
			{
				ID:   "dev",
				Name: "Development",
				Guards: []Guard{
					{Kind: GuardWallclock, Max: 1800, OnFail: GuardOnFailHaltGate},
				},
				Steps: []PipelineStep{
					{
						ID:   "dev-agent",
						Name: "dev-agent",
						Guards: []Guard{
							{Kind: GuardLogSilence, Threshold: 120, OnFail: GuardOnFailHaltGate},
						},
					},
					{ID: "no-guards", Name: "no-guards"},
				},
			},
		},
	}

	// Step with own guards: step guards first, then stage guards.
	guards := cfg.GuardsForStep("dev-agent")
	if len(guards) != 2 {
		t.Fatalf("expected 2 effective guards, got %d", len(guards))
	}
	if guards[0].Kind != GuardLogSilence || guards[1].Kind != GuardWallclock {
		t.Errorf("unexpected guard order: %+v", guards)
	}

	// Step without own guards still inherits the stage guard.
	guards = cfg.GuardsForStep("no-guards")
	if len(guards) != 1 || guards[0].Kind != GuardWallclock {
		t.Errorf("expected inherited stage guard, got %+v", guards)
	}

	// Unknown step: no guards.
	if g := cfg.GuardsForStep("missing"); g != nil {
		t.Errorf("expected nil for unknown step, got %+v", g)
	}
}

func TestValidHITLResolutionAction(t *testing.T) {
	valid := []string{HITLActionResume, HITLActionOverride, HITLActionSendBack, HITLActionSkip, HITLActionAbort}
	for _, a := range valid {
		if !ValidHITLResolutionAction(a) {
			t.Errorf("expected %q to be valid", a)
		}
	}
	for _, a := range []string{"", "approve", "reject", "bogus"} {
		if ValidHITLResolutionAction(a) {
			t.Errorf("expected %q to be invalid", a)
		}
	}
}
