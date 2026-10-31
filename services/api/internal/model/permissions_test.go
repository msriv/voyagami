package model

import "testing"

func TestCan(t *testing.T) {
	want := map[Action][3]bool{ // owner, editor, viewer
		ActionEdit:       {true, true, false},
		ActionJournal:    {true, true, false},
		ActionManageTrip: {true, false, false},
		ActionManageCrew: {true, false, false},
		ActionModerate:   {true, false, false},
	}
	for a, w := range want {
		for i, r := range []Role{RoleOwner, RoleEditor, RoleViewer} {
			if got := Can(r, a); got != w[i] {
				t.Errorf("Can(%s, %d) = %v, want %v", r, a, got, w[i])
			}
		}
	}
	if Can("", ActionEdit) || Can("stranger", ActionEdit) || Can(RoleOwner, 0) {
		t.Error("unknown role or action must be denied")
	}
}
