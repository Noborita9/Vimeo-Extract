package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Noborita9/Vimeo-Transcript-Fetcher/pkg/storage"
	"github.com/Noborita9/Vimeo-Transcript-Fetcher/pkg/transcript"
	"github.com/Noborita9/Vimeo-Transcript-Fetcher/pkg/vimeo"
)

func main() {
	if len(os.Args) > 1 {
		subcmd := os.Args[1]
		if subcmd == "collect" || subcmd == "export-transcripts" || subcmd == "collect-transcripts" {
			runCollect(os.Args[2:])
			return
		}
	}

	var (
		rawURL        string
		password      string
		outputDir     string
		langPref      string
		referer       string
		downloadVideo bool
		onlyVideo     bool
		force         bool
		collectAfter  bool
		listTracks    bool
	)

	flag.StringVar(&rawURL, "u", "", "URL o ID del video / showcase de Vimeo")
	flag.StringVar(&rawURL, "url", "", "URL o ID del video / showcase de Vimeo")
	flag.StringVar(&password, "p", "", "Contraseña del video o showcase (si está protegido)")
	flag.StringVar(&password, "password", "", "Contraseña del video o showcase (si está protegido)")
	flag.StringVar(&outputDir, "o", "./output", "Directorio de salida para guardar las transcripciones y videos")
	flag.StringVar(&outputDir, "output", "./output", "Directorio de salida para guardar las transcripciones y videos")
	flag.StringVar(&langPref, "l", "es", "Código de idioma preferido para subtítulos (default: es)")
	flag.StringVar(&langPref, "lang", "es", "Código de idioma preferido para subtítulos (default: es)")
	flag.StringVar(&referer, "r", "", "Referer HTTP (ej: https://aulas.ort.edu.uy/)")
	flag.StringVar(&referer, "referer", "", "Referer HTTP (ej: https://aulas.ort.edu.uy/)")
	flag.BoolVar(&downloadVideo, "v", false, "Descargar también el video .mp4 (además del transcript)")
	flag.BoolVar(&downloadVideo, "video", false, "Descargar también el video .mp4 (además del transcript)")
	flag.BoolVar(&onlyVideo, "only-video", false, "Descargar únicamente el video .mp4 (sin transcripción)")
	flag.BoolVar(&force, "f", false, "Sobrescribir si ya existe")
	flag.BoolVar(&force, "force", false, "Sobrescribir si ya existe")
	flag.BoolVar(&collectAfter, "c", false, "Recopilar todos los transcripts en output/transcripts/ al finalizar")
	flag.BoolVar(&collectAfter, "collect", false, "Recopilar todos los transcripts en output/transcripts/ al finalizar")
	flag.BoolVar(&listTracks, "list-tracks", false, "Solo listar los tracks de subtítulos disponibles")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Uso: vimeo-fetcher [comando | opciones] <URL o ID>\n\n")
		fmt.Fprintf(os.Stderr, "Comandos disponibles:\n")
		fmt.Fprintf(os.Stderr, "  collect [opciones]       Recopila todos los transcripts en una sola carpeta (output/transcripts/)\n\n")
		fmt.Fprintf(os.Stderr, "Ejemplos de descarga:\n")
		fmt.Fprintf(os.Stderr, "  # Solo transcripción (por defecto):\n")
		fmt.Fprintf(os.Stderr, "  vimeo-fetcher -u \"https://vimeo.com/showcase/12367680?video=1236100586\" -p \"M1awjQrt\"\n\n")
		fmt.Fprintf(os.Stderr, "  # Transcripción + Video .mp4:\n")
		fmt.Fprintf(os.Stderr, "  vimeo-fetcher -u \"https://vimeo.com/showcase/12367680?video=1236100586\" -p \"M1awjQrt\" -v\n\n")
		fmt.Fprintf(os.Stderr, "  # Descargar todo un showcase con video y recopilar transcripts:\n")
		fmt.Fprintf(os.Stderr, "  vimeo-fetcher -u \"https://vimeo.com/showcase/12367680\" -p \"M1awjQrt\" -v -c\n\n")
		fmt.Fprintf(os.Stderr, "  # Recopilar todos los transcripts existentes en output/transcripts/:\n")
		fmt.Fprintf(os.Stderr, "  vimeo-fetcher collect\n\n")
		fmt.Fprintf(os.Stderr, "Opciones:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if rawURL == "" && flag.NArg() > 0 {
		rawURL = flag.Arg(0)
	}

	if rawURL == "" {
		flag.Usage()
		os.Exit(1)
	}

	// Migrar formato antiguo automáticamente si existe
	storage.MigrateOldFormat(outputDir)

	target, err := vimeo.ParseURL(rawURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
		os.Exit(1)
	}

	client, err := vimeo.NewClient(referer)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error al iniciar cliente: %v\n", err)
		os.Exit(1)
	}

	wantTranscript := !onlyVideo
	wantVideo := downloadVideo || onlyVideo

	// Caso 1: Showcase completo
	if target.IsShowcase {
		fmt.Printf("📂 Detectado Showcase ID: %s\n", target.ShowcaseID)
		var hashedPass string
		if password != "" {
			fmt.Println("🔑 Autenticando con contraseña en el showcase...")
			h, err := client.AuthenticateShowcase(target.ShowcaseID, password)
			if err != nil {
				fmt.Fprintf(os.Stderr, "❌ Error al autenticar showcase: %v\n", err)
				os.Exit(1)
			}
			hashedPass = h
			fmt.Println("✅ Showcase autenticado.")
		}

		fmt.Println("🔍 Obteniendo lista de videos del showcase...")
		videos, err := client.FetchShowcaseVideos(target.ShowcaseID, hashedPass)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Error al obtener videos del showcase: %v\n", err)
			os.Exit(1)
		}

		if len(videos) == 0 {
			fmt.Println("⚠️ No se encontraron videos en el showcase.")
			return
		}

		fmt.Printf("✅ Se encontraron %d videos. Procesando...\n\n", len(videos))
		successCount := 0
		skippedCount := 0
		showcaseReferer := fmt.Sprintf("https://vimeo.com/showcase/%s", target.ShowcaseID)

		for i, vid := range videos {
			fmt.Printf("[%d/%d] 🎬 %s (ID: %s)\n", i+1, len(videos), vid.Title, vid.ID)

			skipTranscript := !wantTranscript || storage.TranscriptExists(outputDir, vid.Title)
			skipVideo := !wantVideo || storage.VideoExists(outputDir, vid.Title)

			if !force && skipTranscript && skipVideo {
				fmt.Printf("   ⏭️ Ya existe en output/, salteando. (Usá -f para forzar descarga)\n\n")
				skippedCount++
				continue
			}

			err := processShowcaseVideo(client, target.ShowcaseID, vid.ID, hashedPass, showcaseReferer, outputDir, langPref, wantTranscript, wantVideo, force, listTracks)
			if err != nil {
				fmt.Printf("   ⚠️ Error: %v\n\n", err)
			} else {
				successCount++
				fmt.Println()
			}
		}
		fmt.Printf("🎉 Proceso finalizado: %d procesados, %d salteados (total: %d videos).\n", successCount, skippedCount, len(videos))

		if collectAfter {
			autoCollect(outputDir)
		}
		return
	}

	// Caso 2: Video específico dentro de un showcase (?video=ID)
	if target.ShowcaseID != "" && target.VideoID != "" {
		fmt.Printf("🎬 Procesando video ID: %s (dentro del showcase %s)\n", target.VideoID, target.ShowcaseID)
		var hashedPass string
		if password != "" {
			fmt.Println("🔑 Autenticando con contraseña en el showcase...")
			h, err := client.AuthenticateShowcase(target.ShowcaseID, password)
			if err != nil {
				fmt.Fprintf(os.Stderr, "❌ Error al autenticar showcase: %v\n", err)
				os.Exit(1)
			}
			hashedPass = h
			fmt.Println("✅ Showcase autenticado.")
		}

		showcaseReferer := fmt.Sprintf("https://vimeo.com/showcase/%s", target.ShowcaseID)
		err := processShowcaseVideo(client, target.ShowcaseID, target.VideoID, hashedPass, showcaseReferer, outputDir, langPref, wantTranscript, wantVideo, force, listTracks)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✅ ¡Completado con éxito!")

		if collectAfter {
			autoCollect(outputDir)
		}
		return
	}

	// Caso 3: Video individual estándar
	fmt.Printf("🎬 Procesando video ID: %s\n", target.VideoID)
	config, err := client.GetPlayerConfig(target.VideoID, target.UnlistedHash, password)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error al obtener configuración del video: %v\n", err)
		os.Exit(1)
	}

	refererURL := fmt.Sprintf("https://vimeo.com/%s", target.VideoID)
	if referer != "" {
		refererURL = referer
	}

	err = processVideoMedia(client, config, target.VideoID, refererURL, outputDir, langPref, wantTranscript, wantVideo, force, listTracks)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✅ ¡Completado con éxito!")

	if collectAfter {
		autoCollect(outputDir)
	}
}

func runCollect(args []string) {
	fs := flag.NewFlagSet("collect", flag.ExitOnError)
	var (
		outputDir     string
		destDir       string
		subjectFilter string
		noPerSubject  bool
	)

	fs.StringVar(&outputDir, "o", "./output", "Directorio base de output")
	fs.StringVar(&outputDir, "output", "./output", "Directorio base de output")
	fs.StringVar(&destDir, "d", "", "Directorio de destino (por defecto: <output>/transcripts)")
	fs.StringVar(&destDir, "dest", "", "Directorio de destino (por defecto: <output>/transcripts)")
	fs.StringVar(&subjectFilter, "s", "", "Filtrar por materia específica")
	fs.StringVar(&subjectFilter, "subject", "", "Filtrar por materia específica")
	fs.BoolVar(&noPerSubject, "no-per-subject", false, "No crear subcarpetas transcripts/ dentro de cada materia")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Uso: vimeo-fetcher collect [opciones] [materia opcional]\n\n")
		fmt.Fprintf(os.Stderr, "Recopila todos los archivos transcript.md en una carpeta única y plana dentro de output.\n")
		fmt.Fprintf(os.Stderr, "Ideal para seleccionar o pasar todos los transcripts juntos a herramientas, LLMs o selectores.\n\n")
		fmt.Fprintf(os.Stderr, "Ejemplos:\n")
		fmt.Fprintf(os.Stderr, "  # Recopilar todos los transcripts en output/transcripts/:\n")
		fmt.Fprintf(os.Stderr, "  vimeo-fetcher collect\n\n")
		fmt.Fprintf(os.Stderr, "  # Recopilar solo una materia en particular:\n")
		fmt.Fprintf(os.Stderr, "  vimeo-fetcher collect -s \"Bases de datos 1\"\n\n")
		fmt.Fprintf(os.Stderr, "  # Guardar en una carpeta personalizada:\n")
		fmt.Fprintf(os.Stderr, "  vimeo-fetcher collect -d \"./mis_transcripts\"\n\n")
		fmt.Fprintf(os.Stderr, "Opciones:\n")
		fs.PrintDefaults()
	}

	_ = fs.Parse(args)

	// Si se pasa argumento posicional, usarlo como filtro de materia
	if subjectFilter == "" && fs.NArg() > 0 {
		subjectFilter = fs.Arg(0)
	}

	if destDir == "" {
		destDir = filepath.Join(outputDir, "transcripts")
	}

	// Migrar formato antiguo si todavía existe
	storage.MigrateOldFormat(outputDir)

	fmt.Printf("🔍 Recopilando transcripts desde: %s\n", outputDir)
	results, err := storage.CollectTranscripts(storage.CollectOptions{
		BaseDir:       outputDir,
		GlobalDestDir: destDir,
		SubjectFilter: subjectFilter,
		PerSubject:    !noPerSubject,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error al recopilar transcripts: %v\n", err)
		os.Exit(1)
	}

	if len(results) == 0 {
		fmt.Println("⚠️ No se encontraron archivos transcript.md para recopilar.")
		return
	}

	fmt.Printf("✅ Se recopilaron %d transcripts en:\n👉 %s\n", len(results), destDir)
	if !noPerSubject {
		fmt.Println("📁 También se actualizaron las subcarpetas per-materia (ej: output/<materia>/transcripts/)")
	}
}

func autoCollect(outputDir string) {
	destDir := filepath.Join(outputDir, "transcripts")
	results, err := storage.CollectTranscripts(storage.CollectOptions{
		BaseDir:       outputDir,
		GlobalDestDir: destDir,
		PerSubject:    true,
	})
	if err == nil && len(results) > 0 {
		fmt.Printf("📦 Transcripts sincronizados automáticamente en: %s\n", destDir)
	}
}

func processShowcaseVideo(client *vimeo.Client, showcaseID, videoID, hashedPass, referer, outputDir, langPref string, wantTranscript, wantVideo, force, listOnly bool) error {
	embedURL, _, err := client.GetShowcaseVideoEmbedURL(showcaseID, videoID, hashedPass)
	if err != nil {
		return fmt.Errorf("no se pudo obtener la URL del reproductor: %w", err)
	}

	config, err := client.GetPlayerConfigFromEmbedURL(embedURL, referer)
	if err != nil {
		return fmt.Errorf("no se pudo obtener la configuración del reproductor: %w", err)
	}

	return processVideoMedia(client, config, videoID, referer, outputDir, langPref, wantTranscript, wantVideo, force, listOnly)
}

func processVideoMedia(client *vimeo.Client, config *vimeo.PlayerConfig, videoID, referer, outputDir, langPref string, wantTranscript, wantVideo, force, listOnly bool) error {
	title := config.Video.Title
	if title == "" {
		title = fmt.Sprintf("video-%s", videoID)
	}

	itemDir := storage.DetermineItemDir(outputDir, title)

	if listOnly {
		fmt.Printf("   Configuración para: %s\n", title)
		fmt.Println("   Pistas de subtítulos:")
		for _, t := range config.Request.TextTracks {
			fmt.Printf("   - [%s] %s (default: %v, provenance: %s)\n", t.Lang, t.Label, t.Default, t.Provenance)
		}
		if hlsURL := config.GetHLSURL(); hlsURL != "" {
			fmt.Println("   Stream de video: HLS disponible ✅")
		} else {
			fmt.Println("   Stream de video: No disponible ❌")
		}
		return nil
	}

	// 1. Descarga de transcripción
	if wantTranscript {
		if !force && storage.TranscriptExists(outputDir, title) {
			fmt.Printf("   ⏭️ transcript.md ya existe en %s\n", itemDir)
		} else {
			tracks := config.Request.TextTracks
			if len(tracks) == 0 {
				fmt.Println("   ⚠️ El video no tiene pistas de subtítulos disponibles.")
			} else {
				selectedTrack := selectTrack(tracks, langPref)
				if selectedTrack == nil {
					fmt.Println("   ⚠️ No se encontró una pista de subtítulos adecuada.")
				} else {
					fmt.Printf("   📝 Descargando transcripción (%s - %s)...\n", selectedTrack.Lang, selectedTrack.Label)
					vttRaw, err := client.DownloadTrack(selectedTrack.URL)
					if err != nil {
						fmt.Printf("   ⚠️ Error al descargar subtítulos: %v\n", err)
					} else {
						cues := transcript.ParseVTT(vttRaw)
						if len(cues) == 0 {
							fmt.Println("   ⚠️ Los subtítulos están vacíos.")
						} else {
							paragraphs := transcript.GroupIntoParagraphs(cues)
							md := transcript.BuildMarkdown(title, videoID, paragraphs)
							savedPath, err := storage.SaveTranscript(outputDir, title, md, force)
							if err != nil {
								fmt.Printf("   ⚠️ Error al guardar transcript.md: %v\n", err)
							} else {
								fmt.Printf("   💾 Guardado: %s\n", savedPath)
							}
						}
					}
				}
			}
		}
	}

	// 2. Descarga de video
	if wantVideo {
		if !force && storage.VideoExists(outputDir, title) {
			fmt.Printf("   ⏭️ video.mp4 ya existe en %s\n", itemDir)
		} else {
			hlsURL := config.GetHLSURL()
			if hlsURL == "" {
				return fmt.Errorf("no se encontró enlace de video HLS en la configuración")
			}
			videoTarget := storage.GetVideoPath(outputDir, title)
			fmt.Printf("   📹 Descargando video.mp4 con ffmpeg...\n")
			err := client.DownloadVideo(hlsURL, referer, videoTarget)
			if err != nil {
				return fmt.Errorf("error al descargar video: %w", err)
			}
			fmt.Printf("   💾 Video guardado en: %s\n", videoTarget)
		}
	}

	return nil
}

func selectTrack(tracks []vimeo.TextTrack, langPref string) *vimeo.TextTrack {
	langPref = strings.ToLower(strings.TrimSpace(langPref))

	// 1. Coincidencia exacta de idioma
	for _, t := range tracks {
		if strings.ToLower(t.Lang) == langPref {
			return &t
		}
	}

	// 2. Coincidencia de prefijo (ej: "es-es" o "es-x-autogen" con "es")
	for _, t := range tracks {
		if strings.HasPrefix(strings.ToLower(t.Lang), langPref) {
			return &t
		}
	}

	// 3. Track marcado como default
	for _, t := range tracks {
		if t.Default {
			return &t
		}
	}

	// 4. Primer track disponible
	if len(tracks) > 0 {
		return &tracks[0]
	}

	return nil
}
