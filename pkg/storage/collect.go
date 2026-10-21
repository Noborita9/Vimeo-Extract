package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type CollectOptions struct {
	BaseDir       string // Directorio base de output (default: "./output")
	GlobalDestDir string // Directorio para recopilar todos los transcripts (default: "./output/transcripts")
	SubjectFilter string // Filtrar por materia específica (opcional)
	PerSubject    bool   // Crear también output/<materia>/transcripts/ (default: true)
}

type CollectedResult struct {
	SourcePath        string
	GlobalPath        string
	PerSubjectPath    string
	Subject           string
	Filename          string
}

func CollectTranscripts(opts CollectOptions) ([]CollectedResult, error) {
	if opts.BaseDir == "" {
		opts.BaseDir = "./output"
	}
	if opts.GlobalDestDir == "" {
		opts.GlobalDestDir = filepath.Join(opts.BaseDir, "transcripts")
	}

	absBase, err := filepath.Abs(opts.BaseDir)
	if err != nil {
		return nil, fmt.Errorf("error al obtener ruta absoluta de base: %w", err)
	}

	absGlobalDest, err := filepath.Abs(opts.GlobalDestDir)
	if err != nil {
		return nil, fmt.Errorf("error al obtener ruta absoluta de destino: %w", err)
	}

	if err := os.MkdirAll(opts.GlobalDestDir, 0755); err != nil {
		return nil, fmt.Errorf("error al crear directorio destino %s: %w", opts.GlobalDestDir, err)
	}

	var results []CollectedResult

	err = filepath.Walk(absBase, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return nil
		}

		// Omitir el directorio destino global o cualquier subcarpeta 'transcripts'
		if info.IsDir() {
			if path == absGlobalDest || info.Name() == "transcripts" || info.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}

		if info.Name() != "transcript.md" {
			return nil
		}

		// Obtener ruta relativa respecto al baseDir
		relDir, err := filepath.Rel(absBase, filepath.Dir(path))
		if err != nil {
			return nil
		}

		parts := strings.Split(relDir, string(filepath.Separator))
		var subject, dateOrTitle string
		var globalFilename, perSubjectFilename string

		if len(parts) >= 2 {
			subject = parts[0]
			dateOrTitle = parts[1]
			globalFilename = fmt.Sprintf("%s - %s.md", subject, dateOrTitle)
			perSubjectFilename = fmt.Sprintf("%s.md", dateOrTitle)
		} else if len(parts) == 1 && parts[0] != "." && parts[0] != "" {
			subject = ""
			dateOrTitle = parts[0]
			globalFilename = fmt.Sprintf("%s.md", dateOrTitle)
			perSubjectFilename = ""
		} else {
			return nil
		}

		// Aplicar filtro de materia si se especificó
		if opts.SubjectFilter != "" {
			filterLower := strings.ToLower(strings.TrimSpace(opts.SubjectFilter))
			subjectLower := strings.ToLower(subject)
			if !strings.Contains(subjectLower, filterLower) {
				return nil
			}
		}

		// 1. Copiar al directorio global (output/transcripts/)
		globalPath := filepath.Join(opts.GlobalDestDir, globalFilename)
		if err := copyFile(path, globalPath); err != nil {
			return fmt.Errorf("error al copiar %s a %s: %w", path, globalPath, err)
		}

		res := CollectedResult{
			SourcePath: path,
			GlobalPath: globalPath,
			Subject:    subject,
			Filename:   globalFilename,
		}

		// 2. Copiar al directorio per-materia (output/<materia>/transcripts/) si corresponde
		if opts.PerSubject && subject != "" && perSubjectFilename != "" {
			perSubjectDir := filepath.Join(absBase, subject, "transcripts")
			if err := os.MkdirAll(perSubjectDir, 0755); err == nil {
				perSubjectPath := filepath.Join(perSubjectDir, perSubjectFilename)
				if err := copyFile(path, perSubjectPath); err == nil {
					res.PerSubjectPath = perSubjectPath
				}
			}
		}

		results = append(results, res)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error al recorrer directorios: %w", err)
	}

	return results, nil
}

func copyFile(src, dst string) error {
	srcStat, err := os.Stat(src)
	if err != nil {
		return err
	}

	// Si el archivo destino ya existe con el mismo tamaño y contenido reciente, no hace falta reescribir
	if dstStat, err := os.Stat(dst); err == nil {
		if dstStat.Size() == srcStat.Size() && dstStat.ModTime().After(srcStat.ModTime()) {
			return nil
		}
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}

	return out.Sync()
}
