package patch

import (
	"os"
	"path/filepath"
	"testing"
)

const simplePatch = `diff --git a/hello.txt b/hello.txt
index 1111111..2222222 100644
--- a/hello.txt
+++ b/hello.txt
@@ -1,2 +1,2 @@
-hello
+hello world
 line2
`

const badPatch = `diff --git a/hello.txt b/hello.txt
index 1111111..2222222 100644
--- a/hello.txt
+++ b/hello.txt
@@ -1,2 +1,2 @@
-HELLO
+hello world
 line2
`

func TestApplyAndRevert(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "hello.txt")
	if err := os.WriteFile(file, []byte("hello\nline2\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	bk := NewBookkeeper(dir)
	files, err := bk.Apply(simplePatch)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(files) != 1 || files[0] != "hello.txt" {
		t.Fatalf("unexpected files: %v", files)
	}

	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(data) != "hello world\nline2\n" {
		t.Fatalf("unexpected content: %q", string(data))
	}

	if err := bk.Revert(); err != nil {
		t.Fatalf("revert: %v", err)
	}
	data, err = os.ReadFile(file)
	if err != nil {
		t.Fatalf("read after revert: %v", err)
	}
	if string(data) != "hello\nline2\n" {
		t.Fatalf("unexpected reverted content: %q", string(data))
	}
}

func TestApplyAtomicOnFailure(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "hello.txt")
	orig := "hello\nline2\n"
	if err := os.WriteFile(file, []byte(orig), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	bk := NewBookkeeper(dir)
	if _, err := bk.Apply(badPatch); err == nil {
		t.Fatalf("expected apply failure")
	}
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(data) != orig {
		t.Fatalf("expected original content on failure")
	}
}

func TestDiffReturnsUnified(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "hello.txt")
	if err := os.WriteFile(file, []byte("hello\nline2\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	bk := NewBookkeeper(dir)
	if _, err := bk.Apply(simplePatch); err != nil {
		t.Fatalf("apply: %v", err)
	}
	patch, files, err := bk.Diff()
	if err != nil {
		t.Fatalf("diff: %v", err)
	}
	if len(files) != 1 || files[0] != "hello.txt" {
		t.Fatalf("unexpected files: %v", files)
	}
	if patch == "" {
		t.Fatalf("expected diff output")
	}
}
