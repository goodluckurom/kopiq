package stack_test

import (
	"testing"

	"kopiq/internal/domain"
	"kopiq/internal/stack"
)

func TestStackNoWrap(t *testing.T) {
	var s stack.Session
	steps, err := stack.StepsFromItems([]domain.ClipboardItem{
		{ID: 1, Kind: "text", TextContent: "a"},
		{ID: 2, Kind: "text", TextContent: "b"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Arm(steps); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		text, done, err := s.NextText()
		if err != nil || done {
			t.Fatalf("i=%d text=%q done=%v err=%v", i, text, done, err)
		}
		st := s.Advance()
		if i == 0 && (!st.Armed || st.Index != 1) {
			t.Fatalf("%#v", st)
		}
	}
	if s.Armed() {
		t.Fatal("should disarm after last")
	}
	_, done, _ := s.NextText()
	if !done {
		t.Fatal("expected done")
	}
}

func TestMissingStepBlocks(t *testing.T) {
	var s stack.Session
	if err := s.Arm([]stack.Step{{Text: "ok"}, {Text: ""}}); err == nil {
		t.Fatal("expected arm to reject empty step")
	}
}
