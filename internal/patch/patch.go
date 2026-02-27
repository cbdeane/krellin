package patch

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pmezard/go-difflib/difflib"
)

type Bookkeeper struct {
	root       string
	originals  map[string][]byte
	lastFiles  []string
	lastPatch  string
}

func NewBookkeeper(root string) *Bookkeeper {
	return &Bookkeeper{root: root, originals: map[string][]byte{}}
}

func (b *Bookkeeper) Apply(diff string) ([]string, error) {
	files, hunks, err := parseUnifiedDiff(diff)
	if err != nil {
		return nil, err
	}
	b.lastFiles = nil
	b.lastPatch = ""
	originals := map[string][]byte{}
	newContents := map[string]string{}
	for _, file := range files {
		path := filepath.Join(b.root, file)
		orig, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		originals[file] = orig
		if _, ok := b.originals[file]; !ok {
			b.originals[file] = orig
		}
		newContent, err := applyHunks(string(orig), hunks[file])
		if err != nil {
			return nil, err
		}
		newContents[file] = newContent
		b.lastFiles = append(b.lastFiles, file)
	}
	for _, file := range files {
		path := filepath.Join(b.root, file)
		if err := os.WriteFile(path, []byte(newContents[file]), 0o644); err != nil {
			return nil, err
		}
	}
	b.lastPatch = diff
	return b.lastFiles, nil
}

func (b *Bookkeeper) Revert() error {
	for file, data := range b.originals {
		path := filepath.Join(b.root, file)
		if err := os.WriteFile(path, data, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func (b *Bookkeeper) LastFiles() []string {
	return append([]string{}, b.lastFiles...)
}

func (b *Bookkeeper) LastPatch() string {
	return b.lastPatch
}

func (b *Bookkeeper) Diff() (string, []string, error) {
	if len(b.lastFiles) == 0 {
		return "", nil, nil
	}
	var out strings.Builder
	for _, file := range b.lastFiles {
		orig, ok := b.originals[file]
		if !ok {
			continue
		}
		path := filepath.Join(b.root, file)
		curr, err := os.ReadFile(path)
		if err != nil {
			return "", nil, err
		}
		patch, err := unifiedDiff(file, string(orig), string(curr))
		if err != nil {
			return "", nil, err
		}
		out.WriteString(patch)
	}
	return out.String(), append([]string{}, b.lastFiles...), nil
}

type hunk struct {
	oldStart int
	oldLines int
	newStart int
	newLines int
	lines    []string
}

func parseUnifiedDiff(diff string) ([]string, map[string][]hunk, error) {
	lines := strings.Split(diff, "\n")
	files := []string{}
	hunks := map[string][]hunk{}
	var currentFile string
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		switch {
		case strings.HasPrefix(line, "+++ b/"):
			currentFile = strings.TrimPrefix(line, "+++ b/")
			files = append(files, currentFile)
		case strings.HasPrefix(line, "@@"):
			if currentFile == "" {
				return nil, nil, errors.New("hunk without file")
			}
			h, err := parseHunkHeader(line)
			if err != nil {
				return nil, nil, err
			}
			// collect hunk lines until next header or EOF
			j := i + 1
			for j < len(lines) && !strings.HasPrefix(lines[j], "@@") && !strings.HasPrefix(lines[j], "+++ b/") {
				h.lines = append(h.lines, lines[j])
				j++
			}
			hunks[currentFile] = append(hunks[currentFile], h)
			i = j - 1
		}
	}
	if len(files) == 0 {
		return nil, nil, errors.New("no files in patch")
	}
	return files, hunks, nil
}

func parseHunkHeader(line string) (hunk, error) {
	// format: @@ -oldStart,oldLines +newStart,newLines @@
	parts := strings.Split(line, " ")
	if len(parts) < 3 {
		return hunk{}, fmt.Errorf("invalid hunk header")
	}
	oldRange := strings.TrimPrefix(parts[1], "-")
	newRange := strings.TrimPrefix(parts[2], "+")
	oldStart, oldLines, err := parseRange(oldRange)
	if err != nil {
		return hunk{}, err
	}
	newStart, newLines, err := parseRange(newRange)
	if err != nil {
		return hunk{}, err
	}
	return hunk{oldStart: oldStart, oldLines: oldLines, newStart: newStart, newLines: newLines}, nil
}

func parseRange(r string) (int, int, error) {
	parts := strings.Split(r, ",")
	start, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, err
	}
	lines := 1
	if len(parts) > 1 {
		lines, err = strconv.Atoi(parts[1])
		if err != nil {
			return 0, 0, err
		}
	}
	return start, lines, nil
}

func applyHunks(content string, hunks []hunk) (string, error) {
	lines := strings.Split(content, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	offset := 0
	for _, h := range hunks {
		idx := h.oldStart - 1 + offset
		if idx < 0 || idx > len(lines) {
			return "", fmt.Errorf("hunk out of range")
		}
		newLines := []string{}
		removed := 0
		added := 0
		cursor := idx
		for _, l := range h.lines {
			if l == "" {
				continue
			}
			prefix := l[:1]
			text := l[1:]
			switch prefix {
			case " ":
				if cursor >= len(lines) || lines[cursor] != text {
					return "", fmt.Errorf("context mismatch")
				}
				newLines = append(newLines, lines[cursor])
				cursor++
			case "-":
				if cursor >= len(lines) || lines[cursor] != text {
					return "", fmt.Errorf("delete mismatch")
				}
				cursor++
				removed++
			case "+":
				newLines = append(newLines, text)
				added++
			default:
				return "", fmt.Errorf("invalid hunk line")
			}
		}
		// replace original range with newLines
		lines = append(lines[:idx], append(newLines, lines[cursor:]...)...)
		offset += added - removed
	}
	return strings.Join(lines, "\n") + "\n", nil
}

func unifiedDiff(path string, a string, b string) (string, error) {
	from := "a/" + path
	to := "b/" + path
	aLines := strings.Split(a, "\n")
	bLines := strings.Split(b, "\n")
	diff := difflib.UnifiedDiff{
		A:        aLines,
		B:        bLines,
		FromFile: from,
		ToFile:   to,
		Context:  3,
	}
	text, err := difflib.GetUnifiedDiffString(diff)
	if err != nil {
		return "", err
	}
	return text, nil
}
