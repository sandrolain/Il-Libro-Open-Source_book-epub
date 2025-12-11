package main

import (
	"bytes"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
	"go.abhg.dev/goldmark/frontmatter"
)

// Chapter rappresenta un capitolo del libro con i suoi metadati e contenuto
type Chapter struct {
	Filename string
	Meta     ChapterMeta
	Content  string
	Html     string
	Children []*Chapter
	Images   []string
}

// ChapterMeta contiene i metadati di un capitolo estratti dal frontmatter
type ChapterMeta struct {
	Title string `yaml:"title"`
	Order int    `yaml:"nav_order"`
}

// GetChapters legge e converte i file markdown in capitoli
func GetChapters(cv *goldmark.Markdown, mdPath string) ([]*Chapter, error) {
	slog.Info("Caricamento capitoli", "path", mdPath)

	files, err := fs.ReadDir(os.DirFS(mdPath), ".")
	if err != nil {
		return nil, fmt.Errorf("errore nella lettura della directory %s: %w", mdPath, err)
	}

	var list []*Chapter
	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".md" {
			continue
		}

		chapter, err := processMarkdownFile(cv, mdPath, file.Name())
		if err != nil {
			return nil, err
		}

		list = append(list, chapter)
	}

	slices.SortFunc(list, func(a, b *Chapter) int {
		return a.Meta.Order - b.Meta.Order
	})

	slog.Info("Capitoli caricati", "numero capitoli", len(list))
	return list, nil
}

// processMarkdownFile elabora un singolo file markdown
func processMarkdownFile(cv *goldmark.Markdown, mdPath, fileName string) (*Chapter, error) {
	content, err := os.ReadFile(filepath.Join(mdPath, fileName))
	if err != nil {
		return nil, fmt.Errorf("errore nella lettura del file %s: %w", fileName, err)
	}

	// Pulisce il contenuto dai codici Jekyll
	content = cleanJekyllContent(content)

	// Estrae le immagini
	images := extractImages(content)

	// Converte markdown in HTML
	html, meta, err := convertMarkdownToHTML(cv, content)
	if err != nil {
		return nil, err
	}

	chapter := &Chapter{
		Filename: strings.TrimSuffix(fileName, ".md"),
		Meta:     meta,
		Content:  string(content),
		Html:     html,
		Images:   images,
	}

	// Carica capitoli figli se esistono
	dirPath := filepath.Join(mdPath, fileName[:len(fileName)-3])
	if info, err := os.Stat(dirPath); err == nil && info.IsDir() {
		children, err := GetChapters(cv, dirPath)
		if err != nil {
			return nil, err
		}
		chapter.Children = children
	}

	return chapter, nil
}

// cleanJekyllContent rimuove i codici specifici di Jekyll dal contenuto
func cleanJekyllContent(content []byte) []byte {
	tempContent := string(content)
	tempContent = regexp.MustCompile(`\{:[^}]*\}`).ReplaceAllString(tempContent, "")
	tempContent = strings.ReplaceAll(tempContent, "- TOC", "")
	return []byte(tempContent)
}

// extractImages estrae i percorsi delle immagini dal contenuto markdown
func extractImages(content []byte) []string {
	var images []string
	re := regexp.MustCompile(`!\[.*?\]\((.*?)\)`)
	matches := re.FindAllSubmatch(content, -1)
	for _, match := range matches {
		if len(match) > 1 {
			images = append(images, string(match[1]))
		}
	}
	return images
}

// convertMarkdownToHTML converte il contenuto markdown in HTML ed estrae i metadati
func convertMarkdownToHTML(cv *goldmark.Markdown, content []byte) (string, ChapterMeta, error) {
	ctx := parser.NewContext()
	var buf bytes.Buffer

	if err := (*cv).Convert(content, &buf, parser.WithContext(ctx)); err != nil {
		return "", ChapterMeta{}, fmt.Errorf("errore nella conversione markdown: %w", err)
	}

	d := frontmatter.Get(ctx)
	var meta ChapterMeta
	if err := d.Decode(&meta); err != nil {
		return "", ChapterMeta{}, fmt.Errorf("errore nell'estrazione dei metadati: %w", err)
	}

	html := buf.String()
	html = strings.ReplaceAll(html, "<br>", "<br/>")

	return html, meta, nil
}
