package transcript

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

const (
	MinChars     = 120
	MaxChars     = 450
	PauseSeconds = 2.0
)

var (
	htmlTagRegex       = regexp.MustCompile(`<[^>]+>`)
	whitespaceRegex    = regexp.MustCompile(`\s+`)
	sentenceEndRegex   = regexp.MustCompile(`[.?!…]["»)]?$`)
)

type Cue struct {
	Start   float64
	End     float64
	Content string
}

type Paragraph struct {
	Start float64
	Text  string
}

func ParseVTT(text string) []Cue {
	text = strings.ReplaceAll(text, "\r", "")
	blocks := regexp.MustCompile(`\n{2,}`).Split(text, -1)
	var cues []Cue

	for _, block := range blocks {
		lines := strings.Split(strings.TrimSpace(block), "\n")
		timingIndex := -1
		for i, line := range lines {
			if strings.Contains(line, "-->") {
				timingIndex = i
				break
			}
		}
		if timingIndex == -1 {
			continue
		}

		parts := strings.Split(lines[timingIndex], "-->")
		if len(parts) < 2 {
			continue
		}

		startPart := strings.Fields(parts[0])[0]
		endPart := strings.Fields(parts[1])[0]

		start := parseVTTTime(startPart)
		end := parseVTTTime(endPart)

		rawContent := strings.Join(lines[timingIndex+1:], " ")
		cleanContent := htmlTagRegex.ReplaceAllString(rawContent, "")
		cleanContent = whitespaceRegex.ReplaceAllString(cleanContent, " ")
		cleanContent = strings.TrimSpace(cleanContent)

		if cleanContent != "" {
			cues = append(cues, Cue{
				Start:   start,
				End:     end,
				Content: cleanContent,
			})
		}
	}

	return cues
}

func parseVTTTime(val string) float64 {
	// formato puede ser mm:ss.sss o hh:mm:ss.sss
	parts := strings.Split(val, ":")
	var total float64
	for _, p := range parts {
		num, err := strconv.ParseFloat(p, 64)
		if err != nil {
			return 0
		}
		total = total*60 + num
	}
	return total
}

func GroupIntoParagraphs(cues []Cue) []Paragraph {
	var paragraphs []Paragraph
	var current *Paragraph

	for i := 0; i < len(cues); i++ {
		cue := cues[i]
		if current == nil {
			current = &Paragraph{
				Start: cue.Start,
				Text:  cue.Content,
			}
		} else {
			current.Text = current.Text + " " + cue.Content
		}

		length := len(current.Text)
		endsSentence := sentenceEndRegex.MatchString(cue.Content)

		isLast := i+1 >= len(cues)
		var nextCue *Cue
		if !isLast {
			nextCue = &cues[i+1]
		}

		hasPause := nextCue != nil && (nextCue.Start-cue.End >= PauseSeconds)

		if isLast || length >= MaxChars || (endsSentence && length >= MinChars) || hasPause {
			paragraphs = append(paragraphs, *current)
			current = nil
		}
	}

	return paragraphs
}

func FormatTimestamp(seconds float64) string {
	total := int(math.Floor(seconds))
	h := total / 3600
	m := (total % 3600) / 60
	s := total % 60

	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%02d:%02d", m, s)
}

func BuildMarkdown(title, videoID string, paragraphs []Paragraph) string {
	header := fmt.Sprintf("# %s\n\nFuente: https://player.vimeo.com/video/%s\n", title, videoID)
	var bodyParts []string
	for _, p := range paragraphs {
		bodyParts = append(bodyParts, fmt.Sprintf("[%s] %s", FormatTimestamp(p.Start), p.Text))
	}
	return header + "\n" + strings.Join(bodyParts, "\n\n") + "\n"
}
