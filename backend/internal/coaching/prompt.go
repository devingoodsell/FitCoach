package coaching

import (
	"encoding/json"

	"pro.d11l.fitcoach/backend/internal/injury"
	"pro.d11l.fitcoach/backend/internal/memory"
	"pro.d11l.fitcoach/backend/internal/onboarding"
)

// systemPrompt is the coach's role and the hard output contract. The deterministic
// safety layer (E7/E13) is the real enforcement of contraindications and bounds —
// these instructions reduce the chance the model proposes something that has to be
// corrected, but are never the safety guarantee.
const systemPrompt = `You are FitCoach, an injury-aware, age-aware strength and healthspan coach.
You design ONE workout session from the user's Coach Memory and current state.

Output a SINGLE JSON object matching the provided schema. No prose, no markdown.

Rules:
- Always include an aging_block targeting the user's aging_emphases (bone_balance,
  joint_tendon, vo2max, cardio_base). Set aging_block.emphases to the emphases you
  trained. This block is required in every session.
- Respect every contraindication in derived.contraindications: never prescribe a
  movement in an avoid list, and avoid loading an injured region.
- Make programming age-appropriate: adjust volume, intensity, recovery, and exercise
  selection to the user's age and aging_emphases. When age is known, include at least
  one reasoning note with tag "age_aware".
- Only prescribe exercises the user's available_equipment supports.
- Every set must carry an explicit rest_sec. Use load_kg in kilograms (omit it for
  bodyweight). Set rpe_target where it helps the user gauge effort.
- reasoning: 2-5 brief, plain-language notes explaining key choices. NO medical
  claims, diagnoses, or treatment advice.

Fill only: warmup, main_work, accessory, aging_block, reasoning. The server adds the
id, timestamp, model, disclaimer, and safety results.`

// promptInput is the user-turn payload: the deterministic Coach Memory bytes plus
// the derived planning context the engine resolved (so the model needn't compute
// contraindications, age, or emphases itself).
type promptInput struct {
	CoachMemory json.RawMessage `json:"coach_memory"`
	Derived     derivedContext  `json:"derived"`
}

type derivedContext struct {
	Age                *int                      `json:"age,omitempty"`
	AgingEmphases      *onboarding.AgingEmphases `json:"aging_emphases,omitempty"`
	Contraindications  []injury.Contraindication `json:"contraindications,omitempty"`
	AvailableEquipment []string                  `json:"available_equipment,omitempty"`
}

// buildPrompt assembles the user-turn JSON from the memory payload and derived
// context. Deterministic given its inputs.
func buildPrompt(
	payload memory.PromptPayload,
	age int,
	hasAge bool,
	emphases *onboarding.AgingEmphases,
	contra []injury.Contraindication,
	equipment []string,
) ([]byte, error) {
	mem, err := payload.Marshal()
	if err != nil {
		return nil, err
	}
	d := derivedContext{
		AgingEmphases:      emphases,
		Contraindications:  contra,
		AvailableEquipment: equipment,
	}
	if hasAge {
		a := age
		d.Age = &a
	}
	return json.Marshal(promptInput{CoachMemory: mem, Derived: d})
}

// generationSchema is the structured-output JSON schema for the MODEL-authored
// slice of a session (warmup/main_work/accessory/aging_block/reasoning). Server
// fields (id, timestamps, disclaimer, safety) are added afterward and so are not
// in this schema. It stays within the structured-output subset: no numeric
// min/max, additionalProperties:false on every object.
func generationSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []any{"warmup", "main_work", "accessory", "aging_block", "reasoning"},
		"properties": map[string]any{
			"warmup":    exerciseArraySchema(),
			"main_work": exerciseArraySchema(),
			"accessory": exerciseArraySchema(),
			"aging_block": map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"required":             []any{"emphases", "items"},
				"properties": map[string]any{
					"emphases": map[string]any{
						"type":  "array",
						"items": map[string]any{"type": "string", "enum": []any{"bone_balance", "joint_tendon", "vo2max", "cardio_base"}},
					},
					"items": exerciseArraySchema(),
				},
			},
			"reasoning": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type":                 "object",
					"additionalProperties": false,
					"required":             []any{"text"},
					"properties": map[string]any{
						"text": map[string]any{"type": "string"},
						"tag":  map[string]any{"type": "string", "enum": []any{"intensity", "exercise_choice", "injury_accommodation", "age_aware", "recovery"}},
					},
				},
			},
		},
	}
}

func exerciseArraySchema() map[string]any {
	return map[string]any{
		"type":  "array",
		"items": exerciseSchema(),
	}
}

func exerciseSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []any{"name", "movement", "sets"},
		"properties": map[string]any{
			"name":     map[string]any{"type": "string"},
			"movement": map[string]any{"type": "string"},
			"region":   map[string]any{"type": "string"},
			"notes":    map[string]any{"type": "string"},
			"sets": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type":                 "object",
					"additionalProperties": false,
					"required":             []any{"type"},
					"properties": map[string]any{
						"type":         map[string]any{"type": "string", "enum": []any{"reps", "time", "distance"}},
						"reps":         map[string]any{"type": "integer"},
						"load_kg":      map[string]any{"type": "number"},
						"rpe_target":   map[string]any{"type": "number"},
						"duration_sec": map[string]any{"type": "integer"},
						"rest_sec":     map[string]any{"type": "integer"},
					},
				},
			},
		},
	}
}
