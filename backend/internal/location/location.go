// Package location manages a user's training locations and the switchable
// current context (E9). Locations live in the versioned `locations` Coach Memory
// section (E3) as a JSON document — no separate table — so they are portable and
// account-scoped like the rest of the user model.
package location

import "time"

// Location is one training place with its own equipment profile.
type Location struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Equipment []string `json:"equipment"`
}

// CurrentContext points at the active location and an optional free-text note
// (e.g. "traveling this week, hotel gym only"). ChangedAt lets the engine treat
// a context change as a re-planning trigger (E5-S5).
type CurrentContext struct {
	LocationID string    `json:"location_id"`
	Note       string    `json:"note,omitempty"`
	ChangedAt  time.Time `json:"changed_at"`
}

// Doc is the JSON shape stored in the `locations` memory section.
type Doc struct {
	Locations []Location      `json:"locations"`
	Current   *CurrentContext `json:"current_context,omitempty"`
}

// find returns the index of the location with the given id, or -1.
func (d Doc) find(id string) int {
	for i, l := range d.Locations {
		if l.ID == id {
			return i
		}
	}
	return -1
}
