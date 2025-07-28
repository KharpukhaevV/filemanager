package service

import (
	"fmt"
	"github.com/KharpukhaevV/filemanager/models"
	"github.com/KharpukhaevV/filemanager/utils"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ===================== Превью файлов =====================

func (m *FileManagerState) loadPreview() {
	if len(m.files) == 0 {
		return
	}

	var content []byte
	var err error
	var fileName string
	selected := m.files[m.cursor]
	if utils.IsUnsupportedFile(selected.Name(), selected) {
		m.previewView.SetContent("Формат файла не поддерживается для просмотра")
		return
	}

	if m.inArchive {
		file := m.ArchiveFiles[m.cursor]
		fileName = file.Name
		rc, err := file.Open()
		if err != nil {
			m.previewView.SetContent(fmt.Sprintf("Ошибка открытия файла: %v", err))
			return
		}
		defer rc.Close()

		content, err = io.ReadAll(rc)
		if err != nil {
			m.previewView.SetContent(fmt.Sprintf("Ошибка чтения файла: %v", err))
			return
		}
	} else {
		filePath := filepath.Join(m.Cwd, m.files[m.cursor].Name())
		fileName = m.files[m.cursor].Name()

		if m.isRemote {
			file, err := m.SftpClient.Open(filepath.ToSlash(filePath))
			if err != nil {
				m.previewView.SetContent(fmt.Sprintf("Ошибка открытия файла: %v", err))
				return
			}
			defer file.Close()
			content, err = io.ReadAll(file)
			if err != nil {
				m.previewView.SetContent(fmt.Sprintf("Ошибка чтения файла: %v", err))
				return
			}
		} else {
			content, err = os.ReadFile(filePath)
			if err != nil {
				m.previewView.SetContent(fmt.Sprintf("Ошибка чтения файла: %v", err))
				return
			}
		}
	}

	contentStr := string(content)
	ext := filepath.Ext(fileName)
	switch ext {
	case ".json":
		formatted, err := utils.FormatJSON(contentStr)
		if err == nil {
			contentStr = formatted
		}
	case ".xml":
		formatted, err := utils.FormatXML(contentStr)
		if err == nil {
			contentStr = formatted
		}
	case ".md":
		rendered, err := utils.RenderMarkdown(contentStr)
		if err == nil {
			contentStr = rendered
		}
	}

	highlighted := utils.HighlightSyntax(contentStr, fileName)
	m.previewView.SetContent(highlighted)
	m.previewView.GotoTop()
	m.previewContent = contentStr
	m.previewFile = fileName
}

func (m *FileManagerState) findMatches(query string) []int {
	if query == "" {
		return nil
	}

	lines := strings.Split(m.previewContent, "\n")
	var matches []int

	for i, line := range lines {
		if strings.Contains(strings.ToLower(line), strings.ToLower(query)) {
			matches = append(matches, i)
		}
	}

	return matches
}

func (m *FileManagerState) renderPreview(width, height int) string {
	m.previewView.Width = width - 4
	m.previewView.Height = height - 2

	content := m.previewView.View()

	if m.searchQuery != "" && len(m.searchMatches) > 0 {
		lines := strings.Split(content, "\n")
		query := strings.ToLower(m.searchQuery)

		for i, line := range lines {
			if idx := strings.Index(strings.ToLower(line), query); idx != -1 {
				highlighted := line[:idx] +
					lipgloss.NewStyle().
						Background(lipgloss.Color("#FFD700")).
						Foreground(lipgloss.Color("#000000")).
						Render(line[idx:idx+len(m.searchQuery)]) +
					line[idx+len(m.searchQuery):]
				lines[i] = highlighted
			}
		}
		content = strings.Join(lines, "\n")
	}

	lines := strings.Split(content, "\n")
	if len(lines) < height-2 {
		padding := strings.Repeat("\n", height-2-len(lines))
		content += padding
	}

	var title string
	if m.searchMode {
		title = fmt.Sprintf("Поиск: %s", m.searchQuery)
	} else {
		title = fmt.Sprintf("Просмотр: %s", m.previewFile)
	}

	return fmt.Sprintf(
		"%s\n%s",
		models.Stls.Title.Render(title),
		content,
	)
}
