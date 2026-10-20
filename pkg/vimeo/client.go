package vimeo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const (
	defaultUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	viewerInfoURL    = "https://vimeo.com/_next/viewer"
)

var (
	showcaseURLRegex = regexp.MustCompile(`vimeo\.com/showcase/(\d+)`)
	playerURLRegex   = regexp.MustCompile(`player\.vimeo\.com/video/(\d+)`)
	vimeoURLRegex    = regexp.MustCompile(`vimeo\.com/(?:video/)?(\d+)(?:/([a-zA-Z0-9]+))?`)
	numericRegex     = regexp.MustCompile(`^\d+$`)
)

type Client struct {
	httpClient *http.Client
	viewerInfo *ViewerInfo
	referer    string
	userAgent  string
}

func NewClient(referer string) (*Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("error al inicializar cookie jar: %w", err)
	}

	return &Client{
		httpClient: &http.Client{
			Jar:     jar,
			Timeout: 30 * time.Second,
		},
		referer:   referer,
		userAgent: defaultUserAgent,
	}, nil
}

func (c *Client) FetchViewerInfo() (*ViewerInfo, error) {
	if c.viewerInfo != nil {
		return c.viewerInfo, nil
	}

	req, err := http.NewRequest("GET", viewerInfoURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("falló la solicitud a viewer info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("viewer info devolvió status %d: %s", resp.StatusCode, string(body))
	}

	var info ViewerInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("error al decodificar viewer info: %w", err)
	}

	c.viewerInfo = &info

	// Asegurar que la cookie vuid esté registrada en el jar para .vimeo.com
	if info.VUID != "" && c.httpClient.Jar != nil {
		vimeoURL, _ := url.Parse("https://vimeo.com")
		c.httpClient.Jar.SetCookies(vimeoURL, []*http.Cookie{
			{
				Name:   "vuid",
				Value:  info.VUID,
				Domain: ".vimeo.com",
				Path:   "/",
				Secure: true,
			},
		})
	}

	return c.viewerInfo, nil
}

func (c *Client) SubmitPassword(videoID, password string) error {
	info, err := c.FetchViewerInfo()
	if err != nil {
		return fmt.Errorf("no se pudo obtener token para verificar contraseña: %w", err)
	}

	payload, err := json.Marshal(map[string]string{
		"password": password,
		"token":    info.XSRFT,
	})
	if err != nil {
		return err
	}

	pwdURL := fmt.Sprintf("https://vimeo.com/%s/password", videoID)
	req, err := http.NewRequest("POST", pwdURL, bytes.NewBuffer(payload))
	if err != nil {
		return err
	}

	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Referer", fmt.Sprintf("https://vimeo.com/%s", videoID))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("error al enviar contraseña: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 418 {
		return fmt.Errorf("la contraseña ingresada es incorrecta (HTTP 418)")
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusFound {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("error al validar contraseña (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

func (c *Client) AuthenticateShowcase(showcaseID, password string) (string, error) {
	info, err := c.FetchViewerInfo()
	if err != nil {
		return "", fmt.Errorf("error al obtener viewer info para showcase: %w", err)
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("password", password); err != nil {
		return "", err
	}
	if err := writer.WriteField("token", info.XSRFT); err != nil {
		return "", err
	}
	if err := writer.WriteField("referer_url", fmt.Sprintf("/showcase/%s", showcaseID)); err != nil {
		return "", err
	}
	if err := writer.Close(); err != nil {
		return "", err
	}

	authURL := fmt.Sprintf("https://vimeo.com/showcase/%s/auth", showcaseID)
	req, err := http.NewRequest("POST", authURL, &body)
	if err != nil {
		return "", err
	}

	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Referer", fmt.Sprintf("https://vimeo.com/showcase/%s", showcaseID))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("error al autenticar showcase: %w", err)
	}
	defer resp.Body.Close()

	respData, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusUnauthorized {
		return "", fmt.Errorf("contraseña incorrecta para el showcase (HTTP 401)")
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("error al autenticar showcase (HTTP %d): %s", resp.StatusCode, string(respData))
	}

	var authRes struct {
		HashedPass string `json:"hashed_pass"`
		ErrorCode  *int   `json:"error_code"`
	}
	if err := json.Unmarshal(respData, &authRes); err != nil {
		return "", fmt.Errorf("error al decodificar respuesta de auth del showcase: %w", err)
	}

	if authRes.ErrorCode != nil && *authRes.ErrorCode != 0 {
		return "", fmt.Errorf("showcase auth devolvió código de error %d", *authRes.ErrorCode)
	}

	return authRes.HashedPass, nil
}

func (c *Client) GetPlayerConfig(videoID, unlistedHash, password string) (*PlayerConfig, error) {
	configURL := fmt.Sprintf("https://player.vimeo.com/video/%s/config", videoID)
	if unlistedHash != "" {
		configURL += "?h=" + url.QueryEscape(unlistedHash)
	}

	fetchConfig := func() (*PlayerConfig, int, error) {
		req, err := http.NewRequest("GET", configURL, nil)
		if err != nil {
			return nil, 0, err
		}
		req.Header.Set("User-Agent", c.userAgent)
		req.Header.Set("Accept", "application/json")
		if c.referer != "" {
			req.Header.Set("Referer", c.referer)
		} else {
			req.Header.Set("Referer", fmt.Sprintf("https://vimeo.com/%s", videoID))
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, 0, err
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, resp.StatusCode, err
		}

		if resp.StatusCode != http.StatusOK {
			return nil, resp.StatusCode, fmt.Errorf("config devolvió HTTP %d: %s", resp.StatusCode, string(body))
		}

		var cfg PlayerConfig
		if err := json.Unmarshal(body, &cfg); err != nil {
			return nil, resp.StatusCode, fmt.Errorf("error al parsear JSON de config: %w", err)
		}
		return &cfg, resp.StatusCode, nil
	}

	cfg, statusCode, err := fetchConfig()
	if err != nil && (statusCode == http.StatusForbidden || statusCode == http.StatusUnauthorized || strings.Contains(err.Error(), "403")) {
		if password != "" {
			if submitErr := c.SubmitPassword(videoID, password); submitErr != nil {
				return nil, fmt.Errorf("falló la autenticación con contraseña: %w", submitErr)
			}
			cfg, _, err = fetchConfig()
			if err != nil {
				return nil, fmt.Errorf("error al obtener config tras ingresar contraseña: %w", err)
			}
			return cfg, nil
		}
		return nil, fmt.Errorf("el video requiere contraseña o referer válido (HTTP %d)", statusCode)
	}

	if err != nil {
		if password != "" {
			if submitErr := c.SubmitPassword(videoID, password); submitErr == nil {
				cfg, _, err = fetchConfig()
				if err == nil {
					return cfg, nil
				}
			}
		}
		return nil, err
	}

	return cfg, nil
}

func (c *Client) GetPlayerConfigFromEmbedURL(embedURL, referer string) (*PlayerConfig, error) {
	req, err := http.NewRequest("GET", embedURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", c.userAgent)
	if referer != "" {
		req.Header.Set("Referer", referer)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error al solicitar embed player: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("el embed player devolvió HTTP %d", resp.StatusCode)
	}

	htmlContent := string(body)
	prefix := "window.playerConfig = "
	idx := strings.Index(htmlContent, prefix)
	if idx == -1 {
		return nil, fmt.Errorf("no se encontró window.playerConfig en la página del player")
	}

	endIdx := strings.Index(htmlContent[idx:], "</script>")
	if endIdx == -1 {
		return nil, fmt.Errorf("no se pudo delimitar el script de playerConfig")
	}

	jsonStr := strings.TrimSpace(htmlContent[idx+len(prefix) : idx+endIdx])
	jsonStr = strings.TrimSuffix(jsonStr, ";")

	var cfg PlayerConfig
	if err := json.Unmarshal([]byte(jsonStr), &cfg); err != nil {
		return nil, fmt.Errorf("error al parsear JSON de playerConfig: %w", err)
	}

	return &cfg, nil
}

func (c *Client) DownloadTrack(trackURL string) (string, error) {
	req, err := http.NewRequest("GET", trackURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("error al descargar subtítulos: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("la descarga de subtítulos devolvió status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error al leer contenido de subtítulos: %w", err)
	}

	return string(data), nil
}

type ParsedTarget struct {
	VideoID      string
	UnlistedHash string
	IsShowcase   bool
	ShowcaseID   string
}

func ParseURL(raw string) (*ParsedTarget, error) {
	raw = strings.TrimSpace(raw)
	if numericRegex.MatchString(raw) {
		return &ParsedTarget{VideoID: raw}, nil
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("URL inválida: %w", err)
	}

	// Caso: showcase con video query param
	if m := showcaseURLRegex.FindStringSubmatch(raw); len(m) > 1 {
		showcaseID := m[1]
		videoParam := parsed.Query().Get("video")
		if videoParam != "" {
			return &ParsedTarget{
				VideoID:    videoParam,
				IsShowcase: false,
				ShowcaseID: showcaseID,
			}, nil
		}
		return &ParsedTarget{
			IsShowcase: true,
			ShowcaseID: showcaseID,
		}, nil
	}

	// Caso: player.vimeo.com/video/123456?h=abcdef
	if m := playerURLRegex.FindStringSubmatch(raw); len(m) > 1 {
		h := parsed.Query().Get("h")
		return &ParsedTarget{
			VideoID:      m[1],
			UnlistedHash: h,
		}, nil
	}

	// Caso: vimeo.com/123456 o vimeo.com/123456/abcdef
	if m := vimeoURLRegex.FindStringSubmatch(raw); len(m) > 1 {
		res := &ParsedTarget{VideoID: m[1]}
		if len(m) > 2 && m[2] != "" {
			res.UnlistedHash = m[2]
		}
		if h := parsed.Query().Get("h"); h != "" {
			res.UnlistedHash = h
		}
		return res, nil
	}

	return nil, fmt.Errorf("no se pudo reconocer un ID de video o showcase de Vimeo en: %s", raw)
}
