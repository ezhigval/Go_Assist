package events

import "testing"

func TestSegmentParsingAndDefaults(t *testing.T) {
	if seg, ok := SegmentFromString("family"); !ok || seg != SegmentFamily {
		t.Fatalf("SegmentFromString failed: seg=%q ok=%v", seg, ok)
	}
	if got := ParseSegmentFromAny("business"); got != SegmentBusiness {
		t.Fatalf("ParseSegmentFromAny string = %q, want %q", got, SegmentBusiness)
	}
	if got := ParseSegmentFromAny(SegmentTravel); got != SegmentTravel {
		t.Fatalf("ParseSegmentFromAny segment = %q, want %q", got, SegmentTravel)
	}
	if got := ParseSegmentFromAny("unknown"); got != "" {
		t.Fatalf("ParseSegmentFromAny invalid = %q, want empty", got)
	}
	if got := DefaultSegment(); got != SegmentPersonal {
		t.Fatalf("DefaultSegment = %q, want %q", got, SegmentPersonal)
	}
	if !IsCareerScope(SegmentWork) || !IsCareerScope(SegmentBusiness) || IsCareerScope(SegmentPets) {
		t.Fatalf("IsCareerScope returned inconsistent values")
	}
}

func TestAllSegmentsReturnsCopy(t *testing.T) {
	segments := AllSegments()
	segments[0] = SegmentAssets

	if got := AllSegments()[0]; got != SegmentPersonal {
		t.Fatalf("AllSegments leaked internal slice mutation: got %q", got)
	}
}
