package utils

import (
	"testing"
)

func TestBuildAnnotationPatch(t *testing.T) {
	patch := BuildAnnotationPatch("inrun.dev/managed-by", "inrun")
	meta, ok := patch["metadata"].(map[string]interface{})
	if !ok {
		t.Fatal("expected metadata map")
	}
	ann, ok := meta["annotations"].(map[string]interface{})
	if !ok {
		t.Fatal("expected annotations map")
	}
	if ann["inrun.dev/managed-by"] != "inrun" {
		t.Errorf("unexpected annotation value: %v", ann["inrun.dev/managed-by"])
	}
}

func TestBuildAnnotationPatch_DifferentKey(t *testing.T) {
	patch := BuildAnnotationPatch("foo", "bar")
	meta := patch["metadata"].(map[string]interface{})
	ann := meta["annotations"].(map[string]interface{})
	if ann["foo"] != "bar" {
		t.Errorf("expected bar, got %v", ann["foo"])
	}
}

func TestMerge_EmptyTarget(t *testing.T) {
	s := ""
	Merge(&s, "hello", ",")
	if s != "hello" {
		t.Errorf("expected hello, got %q", s)
	}
}

func TestMerge_NonEmptyTarget(t *testing.T) {
	s := "hello"
	Merge(&s, "world", ",")
	if s != "hello,world" {
		t.Errorf("expected hello,world, got %q", s)
	}
}

func TestMerge_SpaceSeparator(t *testing.T) {
	s := "a"
	Merge(&s, "b", " ")
	if s != "a b" {
		t.Errorf("expected 'a b', got %q", s)
	}
}

func TestRequireStrParams_AllPresent(t *testing.T) {
	err := RequireStrParams(map[string]string{"a": "1", "b": "2"})
	if err != nil {
		t.Errorf("all present must not error: %v", err)
	}
}

func TestRequireStrParams_MissingValue(t *testing.T) {
	err := RequireStrParams(map[string]string{"key": ""})
	if err == nil {
		t.Error("empty value must return error")
	}
}
