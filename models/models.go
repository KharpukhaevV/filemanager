package models

import (
	"github.com/charmbracelet/lipgloss"
)

// ===================== Константы и глобальные переменные =====================

const (
	ConfigFileName = ".filemanager/sftp_config.json"
	DownloadDir    = ".filemanager/downloads"
	StylesFile     = ".filemanager/filemanager_styles.json"
)

var (
	// Стили интерфейса
	Stls = struct {
		Title        lipgloss.Style
		Header       lipgloss.Style
		Row          lipgloss.Style
		Selected     lipgloss.Style
		TopLine      lipgloss.Style
		TopLineInput lipgloss.Style
		BorderStyle  lipgloss.Style
	}{
		Title:        lipgloss.NewStyle(), // Инициализация стилей
		Header:       lipgloss.NewStyle(),
		Row:          lipgloss.NewStyle(),
		Selected:     lipgloss.NewStyle(),
		TopLine:      lipgloss.NewStyle(),
		TopLineInput: lipgloss.NewStyle(),
		BorderStyle:  lipgloss.NewStyle(),
	}
	// Неподдерживаемые расширения файлов
	UnsupportedExtensions = []string{
		".exe", ".dll", ".so", ".a", ".lib", ".o", ".obj",
		".jpg", ".jpeg", ".png", ".gif", ".bmp", ".tiff", ".svg",
		".mp3", ".wav", ".ogg", ".flac", ".aac",
		".mp4", ".avi", ".mov", ".mkv", ".flv", ".wmv",
		".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx",
		".rar", ".7z", ".tar", ".gz", ".bz2", ".xz",
		".iso", ".dmg", ".img",
		".db", ".sqlite", ".mdb", ".accdb",
		".torrent", ".ttf",
	}
)

// ===================== Структуры данных =====================

// SFTPConfig хранит конфигурацию SFTP подключения
type SFTPConfig struct {
	Host     string `json:"host"`
	User     string `json:"user"`
	Password string `json:"password"`
}

type StylesConfig struct {
	TitleForeground        string `json:"titleForeground"`
	TitleBackground        string `json:"titleBackground"`
	HeaderForeground       string `json:"headerForeground"`
	HeaderBackground       string `json:"headerBackground"`
	SelectedForeground     string `json:"selectedForeground"`
	SelectedBackground     string `json:"selectedBackground"`
	TopLineForeground      string `json:"topLineForeground"`
	TopLineBackground      string `json:"topLineBackground"`
	TopLineInputForeground string `json:"topLineInputForeground"`
	TopLineInputBackground string `json:"topLineInputBackground"`
	BorderForeground       string `json:"borderForeground"`
}
