package main

import (
	"fmt"
	"log/slog"
	"strings"

	epub "github.com/go-shiori/go-epub"
)

// BookBuilder gestisce la creazione e configurazione di un libro EPUB
type BookBuilder struct {
	book   *epub.Epub
	config *Config
}

// NewBookBuilder crea un nuovo builder per il libro EPUB
func NewBookBuilder(cfg *Config) (*BookBuilder, error) {
	book, err := epub.NewEpub("Il Libro Open Source")
	if err != nil {
		return nil, fmt.Errorf("errore durante la creazione del libro EPUB: %w", err)
	}

	return &BookBuilder{
		book:   book,
		config: cfg,
	}, nil
}

// SetupMetadata configura i metadati del libro
func (bb *BookBuilder) SetupMetadata() error {
	bb.book.SetTitle("Il manuale del buon dev")
	bb.book.SetAuthor("Community")
	bb.book.SetIdentifier("urn:uuid:" + bb.config.Uuid)
	bb.book.SetLang("it")

	slog.Info("Metadati del libro configurati")
	return nil
}

// AddCover aggiunge la copertina al libro
func (bb *BookBuilder) AddCover() error {
	coverPath, err := bb.book.AddImage(bb.config.Cover, "cover.jpg")
	if err != nil {
		return fmt.Errorf("errore nell'aggiunta della copertina: %w", err)
	}

	bb.book.SetCover(coverPath, "")
	slog.Info("Copertina aggiunta con successo")
	return nil
}

// AddStylesheet aggiunge il foglio di stile al libro
func (bb *BookBuilder) AddStylesheet() (string, error) {
	cssPath, err := bb.book.AddCSS(bb.config.Style, "style.css")
	if err != nil {
		return "", fmt.Errorf("errore nell'aggiunta del foglio di stile: %w", err)
	}

	slog.Info("Foglio di stile aggiunto con successo")
	return cssPath, nil
}

// AddImages aggiunge tutte le immagini dei capitoli al libro
func (bb *BookBuilder) AddImages(chapters []*Chapter, actPath string) error {
	slog.Info("Inizio aggiunta immagini")
	passedImages := make(map[string]string)

	if err := bb.addImagesRecursive(chapters, actPath, passedImages); err != nil {
		return err
	}

	slog.Info("Immagini aggiunte con successo")
	return nil
}

// addImagesRecursive aggiunge le immagini ricorsivamente per tutti i capitoli
func (bb *BookBuilder) addImagesRecursive(chapters []*Chapter, actPath string, passedImages map[string]string) error {
	for _, chapter := range chapters {
		for _, image := range chapter.Images {
			intPath, ok := passedImages[image]
			if !ok {
				fsPath := strings.Replace(image, "/book", actPath, 1)
				fileName := strings.ReplaceAll(strings.TrimLeft(image, "/"), "/", "_")

				var err error
				intPath, err = bb.book.AddImage(fsPath, fileName)
				if err != nil {
					return fmt.Errorf("errore nell'aggiunta dell'immagine %s: %w", fsPath, err)
				}
				passedImages[image] = intPath
			}

			chapter.Html = strings.ReplaceAll(chapter.Html, image, intPath)
		}

		if len(chapter.Children) > 0 {
			if err := bb.addImagesRecursive(chapter.Children, actPath, passedImages); err != nil {
				return err
			}
		}
	}
	return nil
}

// CreateChapters crea i capitoli nel libro EPUB
func (bb *BookBuilder) CreateChapters(chapters []*Chapter, cssPath string) error {
	slog.Info("Inizio creazione capitoli")

	if err := bb.createChaptersRecursive(chapters, cssPath, ""); err != nil {
		return err
	}

	slog.Info("Capitoli creati con successo")
	return nil
}

// createChaptersRecursive crea i capitoli ricorsivamente
func (bb *BookBuilder) createChaptersRecursive(chapters []*Chapter, cssPath, parent string) error {
	for _, chapter := range chapters {
		baseName := ""
		if parent != "" {
			baseName = parent + "__"
		}
		baseName += chapter.Filename
		fileName := baseName + ".xhtml"

		var err error
		if parent != "" {
			_, err = bb.book.AddSubSection(parent, chapter.Html, chapter.Meta.Title, fileName, cssPath)
		} else {
			_, err = bb.book.AddSection(chapter.Html, chapter.Meta.Title, fileName, cssPath)
		}

		if err != nil {
			return fmt.Errorf("errore nella creazione del capitolo %s: %w", chapter.Meta.Title, err)
		}

		if len(chapter.Children) > 0 {
			if err := bb.createChaptersRecursive(chapter.Children, cssPath, fileName); err != nil {
				return err
			}
		}
	}
	return nil
}

// Save salva il libro EPUB su file
func (bb *BookBuilder) Save() error {
	if err := bb.book.Write(bb.config.Output); err != nil {
		return fmt.Errorf("errore durante il salvataggio del libro EPUB: %w", err)
	}

	slog.Info("Libro EPUB salvato con successo", "output", bb.config.Output)
	return nil
}
