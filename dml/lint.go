package dml

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

type LintIssue struct {
	Level   string
	Code    string
	Message string
	Line    int
}

func Lint(path string) ([]LintIssue, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	content := strings.Join(lines, "\n")
	var issues []LintIssue

	mapOpenRe := regexp.MustCompile(`^\s*([A-Za-z_][A-Za-z0-9_]*)\s*=\s*\{\s*$`)
	typedEntryRe := regexp.MustCompile(`^\s*[A-Za-z_][A-Za-z0-9_]*\s+[A-Za-z_][A-Za-z0-9_]*\s*=`)
	rootAssignRe := regexp.MustCompile(`^\s*([A-Za-z_][A-Za-z0-9_]*)\s*=\s*(.*)$`)
	defaultRe := regexp.MustCompile(`^\s*default\s+([A-Za-z_][A-Za-z0-9_]*)\s*=`)

	n := len(lines)
	i := 0
	rootVars := 0
	mapCount := 0
	defaultNames := make(map[string]int)

	for i < n {
		line := lines[i]

		if m := defaultRe.FindStringSubmatch(line); m != nil {
			name := m[1]
			defaultNames[name] = i + 1
		}

		if mo := mapOpenRe.FindStringSubmatch(line); mo != nil {
			mapCount++
			mapName := mo[1]
			openLine := i + 1
			depth := 1
			j := i + 1
			for j < n && depth > 0 {
				l := lines[j]
				depth += strings.Count(l, "{")
				depth -= strings.Count(l, "}")
				j++
			}
			if depth != 0 {
				issues = append(issues, LintIssue{
					Level:   "error",
					Code:    "MAP_UNCLOSED",
					Message: fmt.Sprintf("Mapa %q niezamknięta", mapName),
					Line:    openLine,
				})
				i++
				continue
			}
			closeIdx := j - 1
			empty := true
			firstEntryIdx := -1
			for k := i + 1; k < closeIdx; k++ {
				s := strings.TrimSpace(lines[k])
				if s == "" || strings.HasPrefix(s, "#") || strings.HasPrefix(s, "//") {
					continue
				}
				empty = false
				if firstEntryIdx == -1 {
					firstEntryIdx = k
				}
				if typedEntryRe.MatchString(lines[k]) {
					issues = append(issues, LintIssue{
						Level:   "error",
						Code:    "TYPED_MAP_ENTRY",
						Message: "Typed entries w mapie (np. 'string port = ...') - unikaj typowanych wpisów w mapach",
						Line:    k + 1,
					})
				}
			}
			if empty {
				issues = append(issues, LintIssue{
					Level:   "warning",
					Code:    "EMPTY_MAP",
					Message: fmt.Sprintf("Pusta mapa %q", mapName),
					Line:    openLine,
				})
			} else {
				lastIdx := -1
				for k := closeIdx - 1; k > i; k-- {
					s := strings.TrimSpace(lines[k])
					if s == "" || strings.HasPrefix(s, "#") || strings.HasPrefix(s, "//") {
						continue
					}
					lastIdx = k
					break
				}
				if lastIdx != -1 {
					if strings.HasSuffix(strings.TrimRight(lines[lastIdx], " \t"), ",") {
						issues = append(issues, LintIssue{
							Level:   "error",
							Code:    "MAP_TRAILING_COMMA",
							Message: fmt.Sprintf("Przecinek po ostatnim elemencie mapy %q", mapName),
							Line:    lastIdx + 1,
						})
					}
				}
			}
			i = closeIdx + 1
			continue
		}

		if m := rootAssignRe.FindStringSubmatch(line); m != nil {
			rhs := strings.TrimSpace(m[2])
			if rhs == "" || !strings.HasPrefix(rhs, "{") {
				rootVars++
			}
		}

		i++
	}

	if rootVars > 0 && mapCount > 0 {
		issues = append(issues, LintIssue{
			Level:   "warning",
			Code:    "MIXED_MAP_STYLE",
			Message: "Mieszany styl: użyto map i zmiennych root (map + root vars) — rozważ ujednolicenie",
			Line:    1,
		})
	}

	for name, lineNum := range defaultNames {
		occRe := regexp.MustCompile(`\b` + regexp.QuoteMeta(name) + `\b\s*=`)
		matches := occRe.FindAllStringIndex(content, -1)
		if len(matches) <= 1 {
			issues = append(issues, LintIssue{
				Level:   "warning",
				Code:    "UNUSED_DEFAULT",
				Message: fmt.Sprintf("Nieużyty default %q", name),
				Line:    lineNum,
			})
		}
	}

	return issues, nil
}
