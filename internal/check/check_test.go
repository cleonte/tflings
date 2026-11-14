package check

import "testing"

func TestQuizOK(t *testing.T) {
	c, err := Load([]byte("answer: [B]\nexplanation: why\n"))
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		given []string
		want  bool
	}{
		{[]string{"b"}, true},
		{[]string{" B "}, true},
		{[]string{"a"}, false},
		{[]string{"a", "b"}, false},
		{nil, false},
	}
	for _, tc := range cases {
		if got := c.QuizOK(tc.given); got != tc.want {
			t.Errorf("QuizOK(%v) = %v, want %v", tc.given, got, tc.want)
		}
	}

	multi, _ := Load([]byte("answer: [a, c]\n"))
	if !multi.QuizOK([]string{"c", "a"}) || multi.QuizOK([]string{"a"}) {
		t.Error("a multiple-answer quiz needs exactly the right letters, in any order")
	}
}

func TestLoadRejectsUnknownFields(t *testing.T) {
	if _, err := Load([]byte("anwser: [b]\n")); err == nil {
		t.Error("a typo in check.yaml must be an error")
	}
}
