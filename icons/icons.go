package icons

import (
	"path/filepath"
)

func GetIcon(filename string, isDir bool) string {
	if isDir {
		return icons["dir"]
	}

	ext := filepath.Ext(filename)
	if icon, ok := icons[ext]; ok {
		return icon
	}
	return icons["default"]
}

var icons = map[string]string{
	// Языки программирования
	".py":    "", // Python
	".java":  "", // Java
	".js":    "", // JavaScript
	".ts":    "", // TypeScript
	".cpp":   "", // C++
	".c":     "", // C
	".cs":    "", // C#
	".php":   "", // PHP
	".rb":    "", // Ruby
	".swift": "", // Swift
	".kt":    "", // Kotlin
	".rs":    "", // Rust
	".go":    "", // Go

	// Веб-технологии
	".html": "", // HTML
	".css":  "", // CSS
	".scss": "", // SCSS
	".less": "", // LESS
	".xml":  "󰗀", // XML
	".json": "", // JSON
	".yaml": "", // YAML

	// Документы
	".doc":  "", // Word
	".docx": "", // Word
	".xls":  "", // Excel
	".xlsx": "", // Excel
	".ppt":  "", // PowerPoint
	".pptx": "", // PowerPoint
	".pdf":  "", // PDF

	// Архивы
	".zip": "", // ZIP
	".tar": "", // TAR
	".gz":  "", // GZIP
	".rar": "", // RAR
	".7z":  "", // 7-Zip

	// Медиа
	".mp3": "", // Audio
	".wav": "", // Audio
	".mp4": "", // Video
	".avi": "", // Video
	".mkv": "", // Video
	".png": "", // Image
	".jpg": "", // Image
	".gif": "", // Image
	".svg": "", // Image

	// Системные файлы
	".exe": "", // Executable
	".dll": "", // DLL
	".so":  "", // Shared Object
	".deb": "", // Debian Package
	".rpm": "", // RPM Package

	// Конфигурационные файлы
	".ini":  "", // INI
	".conf": "", // Config
	".toml": "", // TOML
	".env":  "", // Environment

	// Директории
	"dir":     "", // Directory
	"default": "", // Default
}
