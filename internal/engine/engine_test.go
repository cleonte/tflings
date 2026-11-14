package engine

import "testing"

func TestParseVersion(t *testing.T) {
	for in, want := range map[string]Version{
		"1.12.2":       {Major: 1, Minor: 12, Patch: 2},
		"v1.17.0":      {Major: 1, Minor: 17},
		"1.13.0-beta1": {Major: 1, Minor: 13},
	} {
		got, err := ParseVersion(in)
		if err != nil || got != want {
			t.Errorf("ParseVersion(%q) = %v, %v", in, got, err)
		}
	}
	for _, bad := range []string{"one.two", "1.2", "1.x.3", ""} {
		if _, err := ParseVersion(bad); err == nil {
			t.Errorf("ParseVersion(%q) should fail", bad)
		}
	}
}

func TestMinTerraform(t *testing.T) {
	if !(Version{Major: 1, Minor: 12}).AtLeast(MinTerraform) || (Version{Major: 1, Minor: 11, Patch: 9}).AtLeast(MinTerraform) {
		t.Error("MinTerraform is 1.12")
	}
	if !(Version{Major: 2}).AtLeast(MinTerraform) {
		t.Error("2.0.0 is newer than 1.12")
	}
}
