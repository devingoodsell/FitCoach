package coaching

import (
	"context"

	"pro.d11l.fitcoach/backend/internal/injury"
)

// InjuryGenerate adapts the session Generator to the injury package's Generate
// seam, so the natural-language parse and the identification assist (E7-PR2/
// E7-PR7) call Claude through the SAME server-side path as session generation.
// coaching is the only package that talks to Anthropic (CLAUDE.md §2); injury
// depends only on the func, never on coaching or the key. coaching already
// imports injury (contraindications), so returning injury.Generate adds no new
// dependency edge.
func InjuryGenerate(gen Generator) injury.Generate {
	return func(ctx context.Context, system string, prompt []byte, schema map[string]any) ([]byte, error) {
		res, err := gen.Generate(ctx, GenerationRequest{System: system, Prompt: prompt, Schema: schema})
		if err != nil {
			return nil, err
		}
		return res.SessionJSON, nil
	}
}
