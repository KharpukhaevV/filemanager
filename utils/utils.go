package utils

import (
	"encoding/json"
	"fmt"
	"github.com/KharpukhaevV/filemanger/models"
	"github.com/charmbracelet/lipgloss"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
)

// ===================== Утилиты =====================

func FormatSize(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	}
	if size < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(size)/1024)
	}
	return fmt.Sprintf("%.1f MB", float64(size)/(1024*1024))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func TruncateFileName(name string, maxLen int) string {
	if len(name) <= maxLen {
		return name
	}
	return name[:maxLen-3] + "..."
}

func MarkdownToANSI(md string) string {
	// Заголовки уровня 1

	md = strings.ReplaceAll(md, "### ", "\033[1m") // Заголовок уровня 3 (жирный)
	// Заголовки уровня 2
	md = strings.ReplaceAll(md, "## ", "\033[1m\033[4m") // Заголовок уровня 2 (жирный и подчёркнутый)

	// Заголовки уровня 3

	md = strings.ReplaceAll(md, "# ", "\033[1m") // Заголовок уровня 1 (жирный)
	// Жирный текст
	md = strings.ReplaceAll(md, "**", "\033[1m") // Жирный
	md = strings.ReplaceAll(md, "__", "\033[1m") // Жирный

	// Курсив
	md = strings.ReplaceAll(md, "*", "\033[3m") // Курсив
	md = strings.ReplaceAll(md, "_", "\033[3m") // Курсив

	// Зачёркнутый текст
	md = strings.ReplaceAll(md, "~~", "\033[9m") // Зачёркнутый

	// Списки
	md = strings.ReplaceAll(md, "- ", "• ")   // Маркированный список
	md = strings.ReplaceAll(md, "* ", "• ")   // Маркированный список
	md = strings.ReplaceAll(md, "1. ", "1. ") // Нумерованный список (без изменений)

	// Код (inline)
	md = strings.ReplaceAll(md, "`", "\033[7m") // Инвертированный цвет для кода

	// Блоки кода
	md = regexp.MustCompile("(?s)```.*?```").ReplaceAllStringFunc(md, func(s string) string {
		return "\033[7m" + strings.Trim(s, "`") + "\033[0m" // Инвертированный цвет для блока кода
	})

	// Ссылки
	md = regexp.MustCompile(`\[(.*?)\]\((.*?)\)`).ReplaceAllString(md, "\033[4m$1\033[0m") // Подчёркивание для ссылок

	// Изображения (заменяем на текст)
	md = regexp.MustCompile(`!\[(.*?)\]\((.*?)\)`).ReplaceAllString(md, "[Изображение: $1]")

	// Горизонтальные линии
	md = strings.ReplaceAll(md, "---", "──────────") // Горизонтальная линия

	// Сброс стилей в конце каждой строки
	md = strings.ReplaceAll(md, "\n", "\033[0m\n")

	return md
}

// isUnsupportedFile проверяет, поддерживается ли файл для просмотра
func IsUnsupportedFile(filename string, fileInfo os.FileInfo) bool {
	if filepath.Ext(filename) == "" {
		if !fileInfo.IsDir() && !isSymlink(fileInfo) {
			if fileInfo.Mode()&0111 != 0 {
				return true
			}
			if isLikelyBinary(filename) {
				return true
			}
		}
		return false
	}

	ext := strings.ToLower(filepath.Ext(filename))
	if slices.Contains(models.UnsupportedExtensions, ext) {
		return true
	}
	return false
}

func isSymlink(fileInfo os.FileInfo) bool {
	return fileInfo.Mode()&os.ModeSymlink != 0
}

func isLikelyBinary(filename string) bool {
	file, err := os.Open(filename)
	if err != nil {
		return false
	}
	defer file.Close()

	buf := make([]byte, 512)

	for _, e := range buf {
		if e == 0 {
			return true
		}
		if e < 32 && e != 9 && e != 10 && e != 13 {
			return true
		}
	}
	return false
}

func IsZipArchive(filename string) bool {
	return strings.HasSuffix(strings.ToLower(filename), ".zip")
}

// ===================== Инициализация интерфейса =====================

func LoadStylesConfig() (*models.StylesConfig, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	configPath := filepath.Join(homeDir, models.StylesFile)
	var config models.StylesConfig

	// Если файл существует, загружаем стили из него
	if _, err := os.Stat(configPath); err == nil {
		data, err := os.ReadFile(configPath)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(data, &config); err != nil {
			return nil, err
		}
	} else if os.IsNotExist(err) {
		// Если файл не существует, создаем его с дефолтными значениями
		config = models.StylesConfig{
			TitleForeground:        "#FFFFFF",
			TitleBackground:        "#4A90E2",
			HeaderForeground:       "#FFFFFF",
			HeaderBackground:       "#4A4A4A",
			SelectedForeground:     "#FFFFFF",
			SelectedBackground:     "#6C6C6C",
			TopLineForeground:      "#FFFFFF",
			TopLineBackground:      "#4A4A4A",
			TopLineInputForeground: "#FFFFFF",
			TopLineInputBackground: "#4A90E2",
			BorderForeground:       "#6C6C6C",
		}

		data, err := json.MarshalIndent(config, "", "  ")
		if err != nil {
			return nil, err
		}

		if err := os.WriteFile(configPath, data, 0644); err != nil {
			return nil, err
		}
	} else {
		return nil, err
	}

	return &config, nil
}

// Инициализация стилей на основе конфигурации
func InitStyles(config *models.StylesConfig) {
	models.Stls.Title = lipgloss.NewStyle().
		Foreground(lipgloss.Color(config.TitleForeground)).
		Background(lipgloss.Color(config.TitleBackground)).
		Bold(true).
		Padding(0, 1)
	models.Stls.Header = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(config.HeaderForeground)).
		Background(lipgloss.Color(config.HeaderBackground)).
		Padding(0, 1)
	models.Stls.Row = lipgloss.NewStyle().
		Padding(0, 1)
	models.Stls.Selected = lipgloss.NewStyle().
		Foreground(lipgloss.Color(config.SelectedForeground)).
		Background(lipgloss.Color(config.SelectedBackground)).
		Padding(0, 1)
	models.Stls.TopLine = lipgloss.NewStyle().
		Foreground(lipgloss.Color(config.TopLineForeground)).
		Background(lipgloss.Color(config.TopLineBackground)).
		Bold(true).
		Padding(0, 1)
	models.Stls.TopLineInput = lipgloss.NewStyle().
		Foreground(lipgloss.Color(config.TopLineInputForeground)).
		Background(lipgloss.Color(config.TopLineInputBackground)).
		Bold(true).
		Padding(0, 1)
	models.Stls.BorderStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(config.BorderForeground)).
		Padding(0, 1)
}

func SortFiles(files []os.FileInfo) {
	sort.Slice(files, func(i, j int) bool {
		if files[i].IsDir() && !files[j].IsDir() {
			return true
		}
		if !files[i].IsDir() && files[j].IsDir() {
			return false
		}
		return strings.ToLower(files[i].Name()) < strings.ToLower(files[j].Name())
	})
}
