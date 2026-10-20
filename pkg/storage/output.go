package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	// Soporta títulos de ORT como:
	// - "FI-7669-Fundamentos de Ingeniería  SF-78522-ID_ N4A0826 _FIS | 14-10-2026"
	// - "FI-7669-Fundamentos de Ingeniería SF-78522-13-10-2026"
	// - "FI-6411-Finanzas y val. de proy.-78428-AN_N7B0826 _FI | 15-10-2026"
	classTitleRegex  = regexp.MustCompile(`^[A-Z]+-\d+-(.+?)(?:-\d+.*?(?:[|\s-]+))?(\d{2})-(\d{2})-(\d{4})\s*$`)
	invalidCharRegex = regexp.MustCompile(`[\\/:*?"<>|\x00-\x1f]+`)
	multiSpaceRegex  = regexp.MustCompile(`\s+`)
)

type OutputPathInfo struct {
	Folder   string
	Filename string
}

func DetermineOutputPath(title string) OutputPathInfo {
	trimmed := strings.TrimSpace(title)
	match := classTitleRegex.FindStringSubmatch(trimmed)
	if len(match) == 5 {
		subject := SanitizeFilename(match[1])
		day := match[2]
		month := match[3]
		year := match[4]
		return OutputPathInfo{
			Folder:   subject,
			Filename: fmt.Sprintf("%s-%s-%s.md", year, month, day),
		}
	}

	sanitized := SanitizeFilename(trimmed)
	if sanitized == "" {
		sanitized = "transcripcion-vimeo"
	}
	return OutputPathInfo{
		Folder:   "",
		Filename: sanitized + ".md",
	}
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

func TranscriptExists(baseOutputDir, title string) bool {
	pathInfo := DetermineOutputPath(title)
	targetDir := baseOutputDir
	if pathInfo.Folder != "" {
		targetDir = filepath.Join(baseOutputDir, pathInfo.Folder)
	}
	targetFile := filepath.Join(targetDir, pathInfo.Filename)
	_, err := os.Stat(targetFile)
	return err == nil
}

func SaveTranscript(baseOutputDir, title, markdownContent string, overwrite bool) (string, error) {
	pathInfo := DetermineOutputPath(title)
	targetDir := baseOutputDir
	if pathInfo.Folder != "" {
		targetDir = filepath.Join(baseOutputDir, pathInfo.Folder)
	}

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return "", fmt.Errorf("error al crear directorio %s: %w", targetDir, err)
	}

	targetPath := filepath.Join(targetDir, pathInfo.Filename)
	if !overwrite {
		targetPath = uniquePath(targetPath)
	}

	if err := os.WriteFile(targetPath, []byte(markdownContent), 0644); err != nil {
		return "", fmt.Errorf("error al escribir archivo %s: %w", targetPath, err)
	}

	return targetPath, nil
}

func uniquePath(filePath string) string {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return filePath
	}

	dir := filepath.Dir(filePath)
	ext := filepath.Ext(filePath)
	name := strings.TrimSuffix(filepath.Base(filePath), ext)

	for i := 1; ; i++ {
		candidate := filepath.Join(dir, fmt.Sprintf("%s (%d)%s", name, i, ext))
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
}
