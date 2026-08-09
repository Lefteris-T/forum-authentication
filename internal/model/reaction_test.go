package model

import "testing"

func TestReactionValues(t *testing.T) {
	if ReactionDislike != -1 {
		t.Errorf("ReactionDislike = %d, want -1", ReactionDislike)
	}

	if ReactionLike != 1 {
		t.Errorf("ReactionLike = %d, want 1", ReactionLike)
	}
}
