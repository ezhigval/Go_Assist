package events

import "time"

// Segment — жизненный контекст (scope) LEGO-модели: «где/для кого», а не Go-context.
// Модули не ветвятся по scope; они принимают метаданные и валидируют права снаружи.
type Segment string

const (
	SegmentPersonal Segment = "personal"
	SegmentFamily   Segment = "family"
	SegmentWork     Segment = "work"
	SegmentBusiness Segment = "business"
	SegmentHealth   Segment = "health"
	SegmentTravel   Segment = "travel"
	SegmentPets     Segment = "pets"
	SegmentAssets   Segment = "assets"
)

var allSegments = []Segment{
	SegmentPersonal,
	SegmentFamily,
	SegmentWork,
	SegmentBusiness,
	SegmentHealth,
	SegmentTravel,
	SegmentPets,
	SegmentAssets,
}

// AllSegments копия канонического списка scope (политики, UI, валидация).
func AllSegments() []Segment {
	out := make([]Segment, len(allSegments))
	copy(out, allSegments)
	return out
}

// IsValidSegment true для известного жизненного контекста.
func IsValidSegment(s Segment) bool {
	switch s {
	case SegmentPersonal, SegmentFamily, SegmentWork, SegmentBusiness,
		SegmentHealth, SegmentTravel, SegmentPets, SegmentAssets:
		return true
	default:
		return false
	}
}

// SegmentFromString парсит строку в Segment.
func SegmentFromString(s string) (Segment, bool) {
	seg := Segment(s)
	if IsValidSegment(seg) {
		return seg, true
	}
	return "", false
}

// ParseSegmentFromAny извлекает scope из any (строка или Segment).
func ParseSegmentFromAny(v any) Segment {
	switch x := v.(type) {
	case string:
		if s, ok := SegmentFromString(x); ok {
			return s
		}
	case Segment:
		if IsValidSegment(x) {
			return x
		}
	}
	return ""
}

// DefaultSegment если scope не задан.
func DefaultSegment() Segment {
	return SegmentPersonal
}

// IsCareerScope work или business — карьера и предпринимательство (см. матрицу).
func IsCareerScope(s Segment) bool {
	return s == SegmentWork || s == SegmentBusiness
}

// EntityBase общие поля сущностей: scope + теги + время.
type EntityBase struct {
	ID        string    `json:"id"`
	Context   Segment   `json:"context"` // JSON-имя context = историческое; смысл = life scope
	Tags      []string  `json:"tags"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
