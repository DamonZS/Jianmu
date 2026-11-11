package codexsummary

import (
	"reflect"
	"testing"
)

func TestMessage(t *testing.T) {
	msg, ours := Message(Encrypted("done: a, b\nnext: c"))
	want := UserMessage(Prefix + "\ndone: a, b\nnext: c")
	if !ours || !reflect.DeepEqual(msg, want) {
		t.Fatalf("jianmu's compaction read as %v, %v; want %v", msg, ours, want)
	}
	if msg, ours := Message("gAAAAABopaque=="); ours || msg != nil {
		t.Fatalf("OpenAI's own compaction read as jianmu's: %v", msg)
	}
	if msg, ours := Message(Mark + "%%%"); !ours || msg != nil {
		t.Fatalf("an unreadable summary of jianmu's: %v, %v; want nil, true", msg, ours)
	}
}
