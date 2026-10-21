# Vimeo Transcript & Video Fetcher 🚀

CLI ultrarrápida en Go para descargar directamente transcripciones (`transcript.md`) y videos en Full HD (`video.mp4`) de Vimeo y showcases protegidos, sin abrir el navegador.

## Estructura de salida

Las descargas se organizan automáticamente en carpetas por materia y fecha:

```
output/
└── Fundamentos de Ingeniería SF/
    ├── 2026-10-13/
    │   ├── transcript.md
    │   └── video.mp4
    └── 2026-10-14/
        ├── transcript.md
        └── video.mp4
```

*(Si el video no tiene fecha en el título, se guarda como `output/<titulo>/transcript.md` y `output/<titulo>/video.mp4`).*

## Características

- ⚡ **Descarga directa:** Extrae transcripciones en menos de 1 segundo sin emulaciones de navegador.
- 📹 **Descarga de Video MP4:** Usa streams adaptativos HLS e invoca `ffmpeg` para unir audio y video en `.mp4` a máxima calidad (hasta 1080p) sin recodificar.
- 🔒 **Soporte para contraseñas:** Soporta videos y showcases protegidos mediante `-p "tu_password"`.
- 📂 **Soporte para Showcases (Lotes):** Descarga automáticamente todas las clases de una carpeta/showcase o un video particular.
- ⏭️ **Salteo inteligente:** Si una clase ya fue descargada, se saltea automáticamente.
- 📦 **Binario compilado en Go:** Requiere `ffmpeg` instalado para descargar video (`brew install ffmpeg`).

## Instalación y compilación

```bash
cd /Users/noborita/workspace/github.com/Noborita9/Vimeo-Transcript-Fetcher
go build -o vimeo-fetcher ./cmd/vimeo-fetcher
```

## Ejemplos de uso

### 1. Solo transcripción (por defecto, ultrarrápido)
```bash
./vimeo-fetcher -u "https://vimeo.com/showcase/12367680?video=1236100586" -p "M1awjQrt"
```

### 2. Transcripción + Video `.mp4`
```bash
./vimeo-fetcher -u "https://vimeo.com/showcase/12367680?video=1236100586" -p "M1awjQrt" -v
```

### 3. Descargar todo el showcase con videos incluidos
```bash
./vimeo-fetcher -u "https://vimeo.com/showcase/12367680" -p "M1awjQrt" -v
```

### 4. Únicamente video (sin archivo Markdown)
```bash
./vimeo-fetcher -u "https://vimeo.com/showcase/12367680?video=1236100586" -p "M1awjQrt" --only-video
```

### 5. Forzar re-descarga de archivos ya existentes
```bash
./vimeo-fetcher -u "https://vimeo.com/showcase/12367680" -p "M1awjQrt" -f
```

### 6. Recopilar todos los transcripts en una sola carpeta (Selector / LLMs) 📁
Aplana y centraliza todos los `transcript.md` en una única carpeta `output/transcripts/` (con nombres del tipo `<Materia> - <Fecha>.md`) para seleccionarlos fácilmente o subirlos juntos a un LLM:

```bash
# Recopilar todos los transcripts existentes en output/transcripts/:
./vimeo-fetcher collect

# Recopilar solo los de una materia:
./vimeo-fetcher collect -s "Bases de datos 1"
# o pasando el nombre como argumento:
./vimeo-fetcher collect "Fundamentos de Ingeniería SF"

# Descargar y recopilar automáticamente al terminar con la flag -c:
./vimeo-fetcher -u "https://vimeo.com/showcase/12367680" -p "M1awjQrt" -c
```

### Opciones disponibles para descargas

| Flag | Descripción | Por defecto |
|---|---|---|
| `-u, --url` | URL o ID del video / showcase | Argumento posicional |
| `-p, --password` | Contraseña del video o showcase | `""` |
| `-v, --video` | Descargar también el video `.mp4` | `false` |
| `--only-video` | Descargar únicamente el video `.mp4` (sin transcripción) | `false` |
| `-c, --collect` | Recopilar automáticamente en `output/transcripts/` al finalizar | `false` |
| `-f, --force` | Sobrescribir si ya existe en `output/` | `false` |
| `-o, --output` | Carpeta base de destino | `./output` |
| `-l, --lang` | Idioma de subtítulos preferido | `es` |
| `-r, --referer` | Referer HTTP si está restringido por dominio | `""` |
| `--list-tracks` | Solo lista los idiomas y streams disponibles sin descargar | `false` |

### Opciones del comando `collect`

```bash
./vimeo-fetcher collect [opciones] [materia opcional]
```

| Flag | Descripción | Por defecto |
|---|---|---|
| `-o, --output` | Directorio base de output | `./output` |
| `-d, --dest` | Directorio destino para los transcripts planos | `<output>/transcripts` |
| `-s, --subject` | Filtrar por nombre de materia | `""` (todas) |
| `--no-per-subject`| No generar subcarpetas `transcripts/` por materia | `false` |
