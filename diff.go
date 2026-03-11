package main

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
)

type DiffLineKind int

const (
	DiffContext DiffLineKind = iota
	DiffAdded
	DiffRemoved
)

type DiffLine struct {
	Kind    DiffLineKind
	Content string
	OldNum  int
	NewNum  int
}

type DiffHunk struct {
	Header string
	Lines  []DiffLine
}

type FileDiff struct {
	OldPath string
	NewPath string
	Hunks   []DiffHunk
}

type ParsedDiff struct {
	FileDiffs []FileDiff
}

type diffFetchedMsg struct {
	diff *ParsedDiff
	err  error
}

func fetchDiffCmd(number int, repo string) tea.Cmd {
	return func() tea.Msg {
		out, err := exec.Command(
			"gh", "pr", "diff",
			fmt.Sprintf("%d", number),
			"--repo", repo,
		).Output()
		if err != nil {
			return diffFetchedMsg{err: fmt.Errorf("gh pr diff: %w", err)}
		}
		return diffFetchedMsg{diff: parseDiff(string(out))}
	}
}

func parseHunkNums(s string) (int, int) {
	// s is like "-1,5"
	s = strings.TrimLeft(s, "+-")
	parts := strings.SplitN(s, ",", 2)
	start, _ := strconv.Atoi(parts[0])
	count := 1
	if len(parts) == 2 {
		count, _ = strconv.Atoi(parts[1])
	}
	_ = count
	return start, 0
}

func parseDiff(raw string) *ParsedDiff {
	result := &ParsedDiff{}
	lines := strings.Split(raw, "\n")

	var currentFile *FileDiff
	var currentHunk *DiffHunk
	oldNum, newNum := 0, 0

	for _, line := range lines {
		if strings.HasPrefix(line, "diff --git ") {
			if currentFile != nil {
				if currentHunk != nil {
					currentFile.Hunks = append(currentFile.Hunks, *currentHunk)
					currentHunk = nil
				}
				result.FileDiffs = append(result.FileDiffs, *currentFile)
			}
			currentFile = &FileDiff{}
			continue
		}

		if currentFile == nil {
			continue
		}

		if strings.HasPrefix(line, "--- ") {
			path := strings.TrimPrefix(line, "--- ")
			path = strings.TrimPrefix(path, "a/")
			if path != "/dev/null" {
				currentFile.OldPath = path
			}
			continue
		}

		if strings.HasPrefix(line, "+++ ") {
			path := strings.TrimPrefix(line, "+++ ")
			path = strings.TrimPrefix(path, "b/")
			if path != "/dev/null" {
				currentFile.NewPath = path
			}
			continue
		}

		if strings.HasPrefix(line, "@@ ") {
			if currentHunk != nil {
				currentFile.Hunks = append(currentFile.Hunks, *currentHunk)
			}
			currentHunk = &DiffHunk{Header: line}
			// Parse @@ -old,count +new,count @@
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				oldNum, _ = parseHunkNums(parts[1])
				newNum, _ = parseHunkNums(parts[2])
			}
			continue
		}

		if currentHunk == nil {
			continue
		}

		if strings.HasPrefix(line, "+") {
			currentHunk.Lines = append(currentHunk.Lines, DiffLine{
				Kind:    DiffAdded,
				Content: line[1:],
				NewNum:  newNum,
			})
			newNum++
		} else if strings.HasPrefix(line, "-") {
			currentHunk.Lines = append(currentHunk.Lines, DiffLine{
				Kind:    DiffRemoved,
				Content: line[1:],
				OldNum:  oldNum,
			})
			oldNum++
		} else if strings.HasPrefix(line, " ") {
			currentHunk.Lines = append(currentHunk.Lines, DiffLine{
				Kind:    DiffContext,
				Content: line[1:],
				OldNum:  oldNum,
				NewNum:  newNum,
			})
			oldNum++
			newNum++
		}
	}

	if currentFile != nil {
		if currentHunk != nil {
			currentFile.Hunks = append(currentFile.Hunks, *currentHunk)
		}
		result.FileDiffs = append(result.FileDiffs, *currentFile)
	}

	return result
}

// diffTotalLines counts the total number of rendered rows for a diff.
func diffTotalLines(diff *ParsedDiff, termWidth int) int {
	count := 0
	for _, fd := range diff.FileDiffs {
		count++ // file header
		for _, hunk := range fd.Hunks {
			count++ // hunk header
			if termWidth >= 160 {
				// side-by-side: pair consecutive removed/added
				i := 0
				for i < len(hunk.Lines) {
					dl := hunk.Lines[i]
					if dl.Kind == DiffRemoved && i+1 < len(hunk.Lines) && hunk.Lines[i+1].Kind == DiffAdded {
						i += 2
					} else {
						i++
					}
					count++
				}
			} else {
				count += len(hunk.Lines)
			}
		}
	}
	return count
}

func truncateLine(s string, maxLen int) string {
	if maxLen <= 3 {
		return s
	}
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-1]) + "…"
}

// renderUnifiedDiff renders a unified diff view.
func renderUnifiedDiff(diff *ParsedDiff, scrollY, height, width int) string {
	var allLines []string

	for _, fd := range diff.FileDiffs {
		name := fd.NewPath
		if name == "" {
			name = fd.OldPath
		}
		allLines = append(allLines, diffHunkHeaderStyle.Render("─── "+name))

		for _, hunk := range fd.Hunks {
			allLines = append(allLines, diffHunkHeaderStyle.Render(hunk.Header))
			for _, dl := range hunk.Lines {
				content := truncateLine(dl.Content, width-4)
				switch dl.Kind {
				case DiffAdded:
					allLines = append(allLines, diffAddedStyle.Render("+ "+content))
				case DiffRemoved:
					allLines = append(allLines, diffRemovedStyle.Render("- "+content))
				case DiffContext:
					allLines = append(allLines, dimStyle.Render("  "+content))
				}
			}
		}
	}

	total := len(allLines)
	if total == 0 {
		return ""
	}
	if scrollY >= total {
		scrollY = total - 1
	}
	end := scrollY + height
	if end > total {
		end = total
	}
	return strings.Join(allLines[scrollY:end], "\n")
}

// renderSideBySideDiff renders a side-by-side diff view for wide terminals.
func renderSideBySideDiff(diff *ParsedDiff, scrollY, height, width int) string {
	paneWidth := (width - 3) / 2
	contentWidth := paneWidth - 5 // room for "NNN │ "

	var allLines []string

	for _, fd := range diff.FileDiffs {
		name := fd.NewPath
		if name == "" {
			name = fd.OldPath
		}
		allLines = append(allLines, diffHunkHeaderStyle.Render("─── "+name))

		for _, hunk := range fd.Hunks {
			allLines = append(allLines, diffHunkHeaderStyle.Render(hunk.Header))

			i := 0
			lines := hunk.Lines
			for i < len(lines) {
				dl := lines[i]

				if dl.Kind == DiffRemoved && i+1 < len(lines) && lines[i+1].Kind == DiffAdded {
					removed := dl
					added := lines[i+1]
					i += 2

					leftNum := fmt.Sprintf("%3d", removed.OldNum)
					rightNum := fmt.Sprintf("%3d", added.NewNum)
					leftContent := fmt.Sprintf("%-*s", contentWidth, truncateLine(removed.Content, contentWidth))
					rightContent := fmt.Sprintf("%-*s", contentWidth, truncateLine(added.Content, contentWidth))

					left := diffRemovedStyle.Render(fmt.Sprintf("%s │ %s", leftNum, leftContent))
					right := diffAddedStyle.Render(fmt.Sprintf("%s │ %s", rightNum, rightContent))
					allLines = append(allLines, left+" │ "+right)

				} else if dl.Kind == DiffRemoved {
					leftNum := fmt.Sprintf("%3d", dl.OldNum)
					leftContent := fmt.Sprintf("%-*s", contentWidth, truncateLine(dl.Content, contentWidth))
					left := diffRemovedStyle.Render(fmt.Sprintf("%s │ %s", leftNum, leftContent))
					right := strings.Repeat(" ", paneWidth)
					allLines = append(allLines, left+" │ "+right)
					i++

				} else if dl.Kind == DiffAdded {
					rightNum := fmt.Sprintf("%3d", dl.NewNum)
					rightContent := fmt.Sprintf("%-*s", contentWidth, truncateLine(dl.Content, contentWidth))
					left := strings.Repeat(" ", paneWidth)
					right := diffAddedStyle.Render(fmt.Sprintf("%s │ %s", rightNum, rightContent))
					allLines = append(allLines, left+" │ "+right)
					i++

				} else {
					// Context
					leftNum := fmt.Sprintf("%3d", dl.OldNum)
					rightNum := fmt.Sprintf("%3d", dl.NewNum)
					content := fmt.Sprintf("%-*s", contentWidth, truncateLine(dl.Content, contentWidth))
					left := dimStyle.Render(fmt.Sprintf("%s │ %s", leftNum, content))
					right := dimStyle.Render(fmt.Sprintf("%s │ %s", rightNum, content))
					allLines = append(allLines, left+" │ "+right)
					i++
				}
			}
		}
	}

	total := len(allLines)
	if total == 0 {
		return ""
	}
	if scrollY >= total {
		scrollY = total - 1
	}
	end := scrollY + height
	if end > total {
		end = total
	}
	return strings.Join(allLines[scrollY:end], "\n")
}
