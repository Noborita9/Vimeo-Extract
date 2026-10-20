package vimeo

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func (c *Client) DownloadVideo(hlsURL, referer, targetPath string) error {
	ffmpegPath, err := exec.LookPath("ffmpeg")
	if err != nil {
		return fmt.Errorf("ffmpeg no está instalado en el sistema. Podés instalarlo en Mac con: brew install ffmpeg")
	}

	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return fmt.Errorf("error al crear directorio para el video: %w", err)
	}

	headers := fmt.Sprintf("Referer: %s\r\nUser-Agent: %s\r\n", referer, c.userAgent)

	// Invocamos ffmpeg con -c copy para descargar y empaquetar sin recodificar
	cmd := exec.Command(ffmpegPath,
		"-y",
		"-headers", headers,
		"-i", hlsURL,
		"-c", "copy",
		"-movflags", "+faststart",
		targetPath,
	)

	// Conectamos Stderr para que el usuario pueda ver el progreso de descarga de ffmpeg
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg falló al descargar el video: %w", err)
	}

	return nil
}
