package www

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"path/filepath"
	"sync"

	"github.com/angas/solarplant-go/www/maybe"
	"github.com/fsnotify/fsnotify"
)

//go:embed templates
var templatesDirEmbed embed.FS

type TemplateManager struct {
	templates *template.Template
	mutex     sync.RWMutex
	logger    *slog.Logger
}

var funcMap = template.FuncMap{
	"Subtract": func(a, b int) int { return a - b },
	"MaybeString": func(m maybe.Maybe[string]) string {
		if m.IsValid() {
			return m.Value()
		}
		return "-"
	},
	"MaybeUint8": func(m maybe.Maybe[uint8]) string {
		if m.IsValid() {
			return fmt.Sprintf("%d", m.Value())
		}
		return "-"
	},
	"MaybeFloat64": func(m maybe.Maybe[float64], decimals int) string {
		if m.IsValid() {
			return fmt.Sprintf("%."+fmt.Sprintf("%d", decimals)+"f", m.Value())
		}
		return "-"
	},
}

func NewTemplateManager(logger *slog.Logger, extDir *string) (*TemplateManager, error) {
	tm := &TemplateManager{
		logger: logger,
	}

	if extDir != nil {
		if err := tm.loadExternalTemplates(*extDir); err != nil {
			return nil, err
		}
	} else if err := tm.loadInternalTemplates(); err != nil {
		return nil, err
	}

	return tm, nil
}

func (tm *TemplateManager) loadInternalTemplates() error {
	tm.logger.Debug("loading embedded templates...")
	tmpl, err := template.New("").Funcs(funcMap).ParseFS(templatesDirEmbed, "templates/*.html")
	if err != nil {
		return fmt.Errorf("failed to parse templates: %w", err)
	}
	tm.templates = tmpl
	return nil
}

func (tm *TemplateManager) loadExternalTemplates(extDir string) error {
	templatesDir := filepath.Join(extDir, "templates")
	reload := func() error {
		tm.logger.Debug("loading external templates...")
		pattern := filepath.Join(templatesDir, "*.html")
		tmpl, err := template.New("").Funcs(funcMap).ParseGlob(pattern)
		if err != nil {
			return fmt.Errorf("failed to parse templates: %w", err)
		}

		tm.mutex.Lock()
		tm.templates = tmpl
		tm.mutex.Unlock()
		return nil
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create template watcher: %w", err)
	}

	go func() {
		for {
			select {
			case event := <-watcher.Events:
				if event.Op&fsnotify.Write == fsnotify.Write {
					if err := reload(); err != nil {
						tm.logger.Error("error reloading templates", slog.Any("error", err))
					} else {
						tm.logger.Debug("templates reloaded")
					}
				}
			case err := <-watcher.Errors:
				tm.logger.Debug("error watching templates", slog.Any("error", err))
			}
		}
	}()

	if err := watcher.Add(templatesDir); err != nil {
		return fmt.Errorf("failed to watch templates: %w", err)
	}

	if err := reload(); err != nil {
		return fmt.Errorf("failed to load templates: %w", err)
	}

	return nil
}

func (tm *TemplateManager) Execute(name string, data interface{}) (bytes.Buffer, error) {
	var buf bytes.Buffer

	tm.mutex.RLock()
	err := tm.templates.ExecuteTemplate(&buf, name, data)
	tm.mutex.RUnlock()

	if err != nil {
		return bytes.Buffer{}, fmt.Errorf("failed to execute template %s: %w", name, err)
	}

	return buf, nil
}

func (tm *TemplateManager) ExecuteToWriter(name string, data any, wr *http.ResponseWriter) error {
	tm.mutex.RLock()
	err := tm.templates.ExecuteTemplate(*wr, name, data)
	tm.mutex.RUnlock()

	if err != nil {
		return fmt.Errorf("failed to execute template %s: %w", name, err)
	}

	return nil
}
