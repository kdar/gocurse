package gocurse

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
)

const (
	baseURL          = "http://%s.curseforge.com/%s.json"
	gameVersionsPath = "game-versions"
	uploadFilePath   = "addons/%s/upload-file"
)

const (
	FileTypeRelease      = "r"
	FileTypeBeta         = "b"
	FileTypeAlpha        = "a"
	MarkupTypeBBCode     = "bbcode"
	MarkupTypeMarkdown   = "markdown"
	MarkupTypePlain      = "plain"
	MarkupTypeHTML       = "html"
	MarkupTypeWikiCreole = "creole"
)

type GameVersion struct {
	BreaksCompatibility bool   `json:"breaks_compatibility"`
	InternalID          string `json:"internal_id"`
	IsDevelopment       bool   `json:"is_development"`
	Name                string `json:"name"`
	ReleaseDate         string `json:"release_date"`
}

type GameVersions map[string]GameVersion

type FileOptions struct {
	Name              string `json:"name"`
	GameVersions      string `json:"game_versions"`
	FileType          string `json:"file_type"`
	ChangeLog         string `json:"change_log"`
	ChangeMarkupType  string `json:"change_markup_type"`
	KnownCaveats      string `json:"known_caveats"`
	CaveatsMarkupType string `json:"caveats_markup_type"`
}

type curse struct {
	game   string
	apiKey string
}

func New(game, apiKey string) *curse {
	return &curse{
		game:   game,
		apiKey: apiKey,
	}
}

func (c *curse) newRequest(method, path string, body io.Reader) (*http.Request, error) {
	return http.NewRequest(method, fmt.Sprintf(baseURL, c.game, path), body)
}

func (c *curse) GameVersions() (GameVersions, error) {
	req, err := c.newRequest("GET", gameVersionsPath, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "gocurse")
	req.Header.Set("Content-type", "application/json")
	req.Header.Set("X-API-Key", c.apiKey)

	cli := &http.Client{}
	resp, err := cli.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var versions GameVersions
	err = json.NewDecoder(resp.Body).Decode(&versions)
	if err != nil {
		return nil, err
	}

	return versions, nil
}

func (c *curse) LatestGameVersionID() (string, error) {
	versions, err := c.GameVersions()
	if err != nil {
		return "", err
	}

	latest := 0
	for key, _ := range versions {
		id, err := strconv.Atoi(key)
		if err != nil {
			return "", err
		}

		if id > latest {
			latest = id
		}
	}

	return fmt.Sprintf("%d", latest), nil
}

func (c *curse) UploadFile(fileOpt *FileOptions, projectSlug string, filename string, fp io.Reader) error {
	var err error

	if fileOpt.GameVersions == "" {
		fileOpt.GameVersions, err = c.LatestGameVersionID()
		if err != nil {
			return err
		}
	}

	var buf bytes.Buffer
	var fw io.Writer
	w := multipart.NewWriter(&buf)

	optJson, err := json.Marshal(fileOpt)
	if err != nil {
		return err
	}
	var options map[string]string
	err = json.Unmarshal(optJson, &options)
	if err != nil {
		return err
	}

	for key, value := range options {
		fw, err = w.CreateFormField(key)
		if err != nil {
			return err
		}
		fw.Write([]byte(value))
	}

	fw, err = w.CreateFormFile("file", filename)
	if err != nil {
		return err
	}
	io.Copy(fw, fp)
	// curseforge expects the boundary at the end as well. Not sure why.
	fw.Write([]byte(fmt.Sprintf("\n--%s--", w.Boundary())))

	req, err := c.newRequest("POST", fmt.Sprintf(uploadFilePath, projectSlug), &buf)
	if err != nil {
		return err
	}

	req.Header.Set("User-Agent", "gocurse")
	req.Header.Set("Content-type", w.FormDataContentType())
	req.Header.Set("X-API-Key", c.apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	switch resp.StatusCode {
	case http.StatusCreated:
		return nil
	case 422:
		var jsonerrs map[string][]string
		err = json.Unmarshal(body, &jsonerrs)
		if err != nil {
			return err
		}

		var errs []string
		for k, v := range jsonerrs {
			errs = append(errs, fmt.Sprintf("%s: %s", k, strings.Join(v, ", ")))
		}

		return fmt.Errorf(strings.Join(errs, "; "))
	default:
		return fmt.Errorf("error uploading file: [status %d] %s", resp.StatusCode, string(body))
	}

	return nil
}
