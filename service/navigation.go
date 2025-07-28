package service

import (
	"archive/zip"
	"github.com/KharpukhaevV/filemanager/utils"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
)

// ===================== Навигация и управление =====================

func (m *FileManagerState) navigateBack() {
	if m.inArchive {
		if m.isRemote {
			if m.RemoteArchiveFile != nil {
				m.RemoteArchiveFile.Close()
				m.RemoteArchiveFile = nil
			}
		} else {
			if m.ArchiveReader != nil {
				m.ArchiveReader.Close()
				m.ArchiveReader = nil
			}
		}
		m.inArchive = false
		m.archivePath = ""
		m.ArchiveFiles = nil
		m.files = readFiles(m.Cwd, m)

		if m.prevCursorPos >= 0 && m.prevCursorPos < len(m.files) {
			m.cursor = m.prevCursorPos
			m.offset = min(m.cursor-m.visibleItems+1, len(m.files)-m.visibleItems)
		} else {
			baseName := filepath.Base(m.archivePath)
			for i, f := range m.files {
				if f.Name() == baseName {
					m.cursor = i
					break
				}
			}
		}
		return
	}

	parent := filepath.Dir(m.Cwd)
	if parent == m.Cwd {
		return
	}

	baseName := filepath.Base(m.Cwd)
	for i, f := range readFiles(parent, m) {
		if f.Name() == baseName {
			m.cursorPositions[parent] = i
			break
		}
	}

	m.cursorPositions[m.Cwd] = m.cursor
	m.Cwd = parent
	m.files = readFiles(parent, m)

	if pos, exists := m.cursorPositions[parent]; exists {
		if pos < len(m.files) {
			m.cursor = pos
			m.offset = min(m.cursor-m.visibleItems+1, len(m.files)-m.visibleItems)
		} else {
			m.cursor = 0
			m.offset = 0
		}
	} else {
		m.cursor = 0
		m.offset = 0
	}

	m.preview = false
}

func (m *FileManagerState) handleEnter() (tea.Model, tea.Cmd) {
	selected := m.files[m.cursor]

	if selected.IsDir() {
		m.cursorPositions[m.Cwd] = m.cursor
		newPath := filepath.Join(m.Cwd, selected.Name())
		m.Cwd = newPath
		m.files = readFiles(newPath, m)

		if pos, exists := m.cursorPositions[newPath]; exists {
			if pos < len(m.files) {
				m.cursor = pos
				m.offset = min(m.cursor-m.visibleItems+1, len(m.files)-m.visibleItems)
			} else {
				m.cursor = 0
				m.offset = 0
			}
		} else {
			m.cursor = 0
			m.offset = 0
		}

	} else if utils.IsZipArchive(selected.Name()) {
		m.prevCursorPos = m.cursor
		archivePath := filepath.Join(m.Cwd, selected.Name())

		if m.isRemote {
			if m.RemoteArchiveFile != nil {
				m.RemoteArchiveFile.Close()
				m.RemoteArchiveFile = nil
			}

			file, err := m.SftpClient.Open(filepath.ToSlash(archivePath))
			if err != nil {
				return m, tea.Println("Ошибка открытия архива:", err)
			}

			stat, err := file.Stat()
			if err != nil {
				file.Close()
				return m, tea.Println("Ошибка получения информации об архиве:", err)
			}

			reader, err := zip.NewReader(file, stat.Size())
			if err != nil {
				file.Close()
				return m, tea.Println("Ошибка чтения архива:", err)
			}

			m.inArchive = true
			m.archivePath = archivePath
			m.ArchiveFiles = reader.File
			m.RemoteArchiveFile = file
			m.files = readFiles("", m)
		} else {
			if m.ArchiveReader != nil {
				m.ArchiveReader.Close()
				m.ArchiveReader = nil
			}

			reader, err := zip.OpenReader(archivePath)
			if err != nil {
				return m, tea.Println("Ошибка чтения архива:", err)
			}

			m.inArchive = true
			m.archivePath = archivePath
			m.ArchiveFiles = reader.File
			m.ArchiveReader = reader
			m.files = readFiles("", m)
		}

		m.cursor = 0
		m.offset = 0
	} else {
		m.loadPreview()
		m.preview = true
		m.previewFile = selected.Name()
	}
	return m, nil
}
