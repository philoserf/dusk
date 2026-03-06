// Package dusk provides astronomical calculations: twilight times,
// sunrise/sunset, moonrise/moonset, lunar phase, and celestial
// coordinate conversions.
//
// All angles are in degrees. Time parameters use [time.Time].
// Functions that produce local times accept an [Observer] with a [*time.Location] field.
// Zero-value [time.Time] signals "event did not occur" (e.g., the Moon
// does not rise on a given day); check with [time.Time.IsZero].
//
// Two sentinel errors distinguish polar edge cases:
// [ErrCircumpolar] (object always above the horizon) and
// [ErrNeverRises] (object never rises).
//
// # References
//
//   - Meeus, Jean. Astronomical Algorithms. 2nd ed. Willmann-Bell, 1998.
package dusk
