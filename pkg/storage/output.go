package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	classTitleRegex  = regexp.MustCompile(`^[A-Z]+-\d+-(.+?)(?:-\d+.*?(?:[|\s-]+))?(\d{2})-(\d{2})-(\d{4})\s*$`)
	invalidCharRegex = regexp.MustCompile(`[\\/:*?"<>|\x00-\x1f]+`)
	multiSpaceRegex  = regexp.MustCompile(`\s+`)
	dateFileRegex    = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})\.md$`)
)

// DetermineItemDir calcula el directorio específico para este video/clase:
// - Para clases de ORT: output/{materia}/{yyyy-mm-dd}/
// - Para otros videos:  output/{titulo}/
func DetermineItemDir(baseOutputDir, title string) string {
	trimmed := strings.TrimSpace(title)
	match := classTitleRegex.FindStringSubmatch(trimmed)
	if len(match) == 5 {
		subject := SanitizeFilename(match[1])
		day := match[2]
		month := match[3]
		year := match[4]
		dateFolder := fmt.Sprintf("%s-%s-%s", year, month, day)
		return filepath.Join(baseOutputDir, subject, dateFolder)
	}

	sanitized := SanitizeFilename(trimmed)
	if sanitized == "" {
		sanitized = "video-sin-titulo"
	}
	return filepath.Join(baseOutputDir, sanitized)
}

func SanitizeFilename(name string) string {
	clean := invalidCharRegex.ReplaceAllString(name, "-")
	clean = multiSpaceRegex.ReplaceAllString(clean, " ")
	clean = strings.Trim(clean, ". ")
	if len(clean) > 120 {
		clean = clean[:120]
	}
	return clean
}

func GetTranscriptPath(baseOutputDir, title string) string {
	return filepath.Join(DetermineItemDir(baseOutputDir, title), "transcript.md")
}

func GetVideoPath(baseOutputDir, title string) string {
	return filepath.Join(DetermineItemDir(baseOutputDir, title), "video.mp4")
}

func TranscriptExists(baseOutputDir, title string) bool {
	p := GetTranscriptPath(baseOutputDir, title)
	_, err := os.Stat(p)
	return err == nil
}

func VideoExists(baseOutputDir, title string) bool {
	p := GetVideoPath(baseOutputDir, title)
	_, err := os.Stat(p)
	return err == nil
}

func SaveTranscript(baseOutputDir, title, markdownContent string, overwrite bool) (string, error) {
	itemDir := DetermineItemDir(baseOutputDir, title)
	if err := os.MkdirAll(itemDir, 0755); err != nil {
		return "", fmt.Errorf("error al crear directorio %s: %w", itemDir, err)
	}

	targetPath := filepath.Join(itemDir, "transcript.md")
	if !overwrite && FileExists(targetPath) {
		return targetPath, nil
	}

	if err := os.WriteFile(targetPath, []byte(markdownContent), 0644); err != nil {
		return "", fmt.Errorf("error al escribir archivo %s: %w", targetPath, err)
	}

	return targetPath, nil
}

func FileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// MigrateOldFormat migra automáticamente archivos del formato previo
// (output/<materia>/<fecha>.md) al nuevo formato (output/<materia>/<fecha>/transcript.md)
func MigrateOldFormat(baseOutputDir string) {
	entries, err := os.ReadDir(baseOutputDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		subjectDir := filepath.Join(baseOutputDir, entry.Name())
		subEntries, err := os.ReadDir(subjectDir)
		if err != nil {
			continue
		}

		for _, sub := range subEntries {
			if sub.IsDir() {
				continue
			}
			m := dateFileRegex.FindStringSubmatch(sub.Name())
			if len(m) == 2 {
				dateFolder := m[1]
				oldPath := filepath.Join(subjectDir, sub.Name())
				newDir := filepath.Join(subjectDir, dateFolder)
				newPath := filepath.Join(newDir, "transcript.md")

				_ = os.MkdirAll(newDir, 0755)
				_ = os.Rename(oldPath, newPath)
			}
		}
	}
}
