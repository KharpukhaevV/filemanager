package service

import (
	"archive/zip"
	"fmt"
	"github.com/KharpukhaevV/filemanager/icons"
	"github.com/KharpukhaevV/filemanager/models"
	"github.com/KharpukhaevV/filemanager/utils"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// FileManagerState хранит состояние файлового менеджера
type FileManagerState struct {
	Cwd               string
	files             []os.FileInfo
	cursor            int
	offset            int
	preview           bool
	previewView       viewport.Model
	width             int
	height            int
	visibleItems      int
	previewFile       string
	cursorPositions   map[string]int
	mode              string
	input             string
	confirmDelete     bool
	ExitWithDir       bool
	searchMode        bool
	searchQuery       string
	searchPosition    int
	searchMatches     []int
	currentMatch      int
	previewContent    string
	SftpClient        *sftp.Client
	SftpSession       *ssh.Session
	isRemote          bool
	remoteHost        string
	remoteUser        string
	remotePassword    string
	inArchive         bool
	archivePath       string
	ArchiveFiles      []*zip.File
	ArchiveReader     *zip.ReadCloser
	RemoteArchiveFile io.ReadCloser
	prevCursorPos     int
	status            string
}

// ===================== Работа с файлами и директориями =====================

func readFiles(dir string, m *FileManagerState) []os.FileInfo {
	if m != nil && m.inArchive {
		var files []os.FileInfo
		for _, f := range m.ArchiveFiles {
			files = append(files, f.FileInfo())
		}
		return files
	}

	if m != nil && m.isRemote {
		dir = filepath.ToSlash(dir)
		return m.readRemoteFiles(dir)
	}

	files, _ := os.ReadDir(dir)
	infos := make([]os.FileInfo, len(files))
	for i, f := range files {
		infos[i], _ = f.Info()
	}

	utils.SortFiles(infos)
	return infos
}

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
		} else {
			m.cursor = 0
		}
	} else {
		m.cursor = 0
	}

	m.offset = 0
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
			} else {
				m.cursor = 0
			}
		} else {
			m.cursor = 0
		}

		m.offset = 0
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

// ===================== Обработка ввода =====================

func (m *FileManagerState) handleInput() (tea.Model, tea.Cmd) {
	switch m.mode {
	case "create":
		if strings.HasSuffix(m.input, "/") {
			if m.isRemote {
				m.SftpClient.Mkdir(filepath.ToSlash(filepath.Join(m.Cwd, m.input)))
			} else {
				os.Mkdir(filepath.Join(m.Cwd, m.input), 0755)
			}
		} else {
			if m.isRemote {
				m.SftpClient.Create(filepath.ToSlash(filepath.Join(m.Cwd, m.input)))
			} else {
				os.Create(filepath.Join(m.Cwd, m.input))
			}
		}
	case "rename":
		if m.isRemote {
			m.SftpClient.Rename(
				filepath.ToSlash(filepath.Join(m.Cwd, m.files[m.cursor].Name())),
				filepath.ToSlash(filepath.Join(m.Cwd, m.input)),
			)
		} else {
			os.Rename(
				filepath.Join(m.Cwd, m.files[m.cursor].Name()),
				filepath.Join(m.Cwd, m.input),
			)
		}
	case "move":
		newPath := filepath.Join(m.Cwd, m.input)
		if m.isRemote {
			if err := m.SftpClient.Rename(
				filepath.ToSlash(filepath.Join(m.Cwd, m.files[m.cursor].Name())),
				filepath.ToSlash(newPath),
			); err != nil {
				fmt.Println("Ошибка перемещения:", err)
			}
		} else {
			if err := os.Rename(
				filepath.Join(m.Cwd, m.files[m.cursor].Name()),
				newPath,
			); err != nil {
				fmt.Println("Ошибка перемещения:", err)
			}
		}
	case "delete":
		if m.input == "y" {
			if m.isRemote {
				if err := m.SftpClient.Remove(filepath.ToSlash(filepath.Join(m.Cwd, m.files[m.cursor].Name()))); err != nil {
					fmt.Println("Ошибка удаления:", err)
				}
			} else {
				if err := os.RemoveAll(filepath.Join(m.Cwd, m.files[m.cursor].Name())); err != nil {
					fmt.Println("Ошибка удаления:", err)
				}
			}
			newPos := max(m.cursor-1, 0)
			m.cursor = newPos
		}

	case "sftp_confirm":
		if m.input == "y" {
			// Если пользователь подтвердил использование сохраненных данных
			config, _ := loadSFTPConfig()
			m.remoteHost = config.Host
			m.remoteUser = config.User
			m.remotePassword = config.Password
			if err := m.initSFTP(); err != nil {
				return m, tea.Println("Ошибка подключения:", err)
			}
			m.mode = "normal"
			return m, nil
		} else if m.input == "n" {
			// Если пользователь выбрал ввод новых данных
			m.mode = "sftp_host"
			m.input = ""
			return m, nil
		}
		return m, nil

	case "sftp_host":
		m.remoteHost = m.input
		m.mode = "sftp_user"
		m.input = ""
		return m, nil

	case "sftp_user":
		m.remoteUser = m.input
		m.mode = "sftp_password"
		m.input = ""
		return m, nil

	case "sftp_password":
		m.remotePassword = m.input

		config := &models.SFTPConfig{
			Host:     m.remoteHost,
			User:     m.remoteUser,
			Password: m.remotePassword,
		}
		if err := saveSFTPConfig(config); err != nil {
			return m, tea.Println("Не удалось сохранить конфигурацию:", err)
		}

		if err := m.initSFTP(); err != nil {
			m.mode = "normal"
			m.input = ""
			return m, tea.Println("Ошибка подключения:", err)
		}
		m.mode = "normal"
		m.input = ""
		return m, nil
	}

	m.mode = "normal"
	m.files = readFiles(m.Cwd, m)
	return m, nil
}

// ===================== Отображение интерфейса =====================

func (m *FileManagerState) View() string {
	leftWidth := int(float64(m.width) * 0.4)
	rightWidth := m.width - leftWidth - 6
	panelHeight := m.height - 6

	var topLine string
	if m.mode != "normal" {
		prompt := m.getPrompt()
		input := fmt.Sprintf("%s%s", prompt, m.input)
		topLine = models.Stls.TopLineInput.Render(input)
	} else {
		topLine = models.Stls.TopLine.Render(m.Cwd)
	}

	leftContent := m.renderNavigation(leftWidth, panelHeight)

	var rightContent string
	if m.preview {
		rightContent = m.renderPreview(rightWidth, panelHeight)
	} else {
		rightContent = lipgloss.NewStyle().
			Height(panelHeight).
			Render("Выберите файл для просмотра (Пробел)")
	}

	borderStyle := models.Stls.BorderStyle
	leftBox := borderStyle.Copy().
		Width(leftWidth).
		Height(m.height - 4).
		Render(leftContent)

	rightBox := borderStyle.Copy().
		Width(rightWidth).
		Height(m.height - 4).
		Render(rightContent)

	mainContent := lipgloss.JoinHorizontal(lipgloss.Top, leftBox, rightBox)

	status := models.Stls.Header.Copy().
		Width(m.width).
		Render(fmt.Sprintf("↑/↓: навигация | Enter: открыть | Пробел: превью | b: назад | q: выход | SFTP: %s (%s)", m.remoteHost, func() string {
			if m.isRemote {
				if m.status != "" {
					return m.status
				}
				return "подключен"
			}
			return "отключен"
		}()))

	fullUI := lipgloss.JoinVertical(lipgloss.Left,
		topLine,
		mainContent,
		status,
	)

	return fullUI
}

func (m *FileManagerState) renderNavigation(width, height int) string {
	var sb strings.Builder

	sb.WriteString(models.Stls.Header.Render(
		fmt.Sprintf("%-20s %-10s   %s", "Последнее изменение", "Размер", "Имя"),
	) + "\n")

	maxNameLength := width - 35

	if len(m.files) == 0 {
		sb.WriteString(models.Stls.Row.Render("Папка пуста") + "\n")
		return sb.String()
	}

	visibleCount := min(m.visibleItems, len(m.files))
	start := max(0, min(m.offset, len(m.files)-visibleCount))
	end := start + visibleCount

	for i := start; i < end; i++ {
		f := m.files[i]
		icon := icons.GetIcon(f.Name(), f.IsDir())
		name := utils.TruncateFileName(icon+" "+f.Name(), maxNameLength)
		lastWrite := f.ModTime().Format("2006-01-02 15:04:05")
		size := utils.FormatSize(f.Size())

		style := models.Stls.Row
		if i == m.cursor {
			style = models.Stls.Selected
		}

		sb.WriteString(style.Render(
			fmt.Sprintf("%-20s %-10s   %s", lastWrite, size, name),
		) + "\n")
	}

	return sb.String()
}

func (m *FileManagerState) getPrompt() string {
	switch m.mode {
	case "create":
		return "Создать (файл или директорию с /):"
	case "rename":
		return "Переименовать:"
	case "move":
		return "Переместить (относительный путь):"
	case "delete":
		return "Удалить? (y/n):"
	case "sftp_confirm":
		return "Использовать последнее сохраненное подключение? (y/n):"
	case "sftp_host":
		return "Введите хост:"
	case "sftp_user":
		return "Введите логин:"
	case "sftp_password":
		return "Введите пароль:"
	default:
		return ""
	}
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

func InitialModel() tea.Model {
	cwd, _ := os.Getwd()
	files := readFiles(cwd, nil)

	m := &FileManagerState{
		Cwd:             cwd,
		files:           files,
		cursor:          0,
		preview:         false,
		mode:            "normal",
		cursorPositions: make(map[string]int),
		previewView:     viewport.New(0, 0),
		prevCursorPos:   -1,
	}

	if parent := filepath.Dir(cwd); parent != cwd {
		m.cursorPositions[parent] = getPositionInParent(cwd, parent, m)
	}

	return m
}

func getPositionInParent(current, parent string, m *FileManagerState) int {
	var files []os.FileInfo
	var base string
	if m.isRemote {
		files = m.readRemoteFiles(filepath.ToSlash(parent))
		base = filepath.ToSlash(filepath.Base(current))
	} else {
		files = readFiles(parent, nil)
		base = filepath.Base(current)
	}

	for i, f := range files {
		if f.Name() == base {
			return i
		}
	}
	return 0
}

func (m *FileManagerState) Init() tea.Cmd {
	return nil
}

func (m *FileManagerState) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.visibleItems = m.height - 8
		m.previewView.Width = msg.Width
		m.previewView.Height = m.height - 4
		return m, nil

	case tea.KeyMsg:
		if m.preview {
			if m.searchMode {
				switch msg.String() {
				case "esc":
					m.searchMode = false
					m.searchQuery = ""
				case "enter":
					m.searchMatches = m.findMatches(m.searchQuery)
					if len(m.searchMatches) > 0 {
						m.currentMatch = 0
						m.searchPosition = m.searchMatches[0]
						m.previewView.SetYOffset(m.searchPosition)
					}
				case "backspace":
					if len(m.searchQuery) > 0 {
						m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
					}
				case "n":
					if len(m.searchMatches) > 0 {
						m.currentMatch = (m.currentMatch + 1) % len(m.searchMatches)
						m.searchPosition = m.searchMatches[m.currentMatch]
						m.previewView.SetYOffset(m.searchPosition)
					}
				case "p":
					if len(m.searchMatches) > 0 {
						m.currentMatch = (m.currentMatch - 1 + len(m.searchMatches)) % len(m.searchMatches)
						m.searchPosition = m.searchMatches[m.currentMatch]
						m.previewView.SetYOffset(m.searchPosition)
					}
				default:
					if len(msg.String()) == 1 {
						m.searchQuery += msg.String()
					}
				}
				return m, nil
			}
			switch msg.String() {
			case "f":
				m.searchMode = true
				m.searchQuery = ""
				return m, nil
			case "esc", "h":
				m.preview = false
				return m, nil
			case "ctrl+c", "q":
				return m, tea.Quit
			case "ctrl+k":
				for i := 0; i < 10; i++ {
					m.previewView.LineUp(1)
				}
				return m, nil
			case "ctrl+j":
				for i := 0; i < 10; i++ {
					m.previewView.LineDown(1)
				}
				return m, nil
			case "ctrl+u":
				m.previewView.GotoTop()
				return m, nil
			case "ctrl+d":
				m.previewView.GotoBottom()
				return m, nil
			default:
				var cmd tea.Cmd
				m.previewView, cmd = m.previewView.Update(msg)
				return m, cmd
			}
		} else {
			if m.mode != "normal" {
				switch msg.String() {
				case "esc":
					m.mode = "normal"
					return m, nil
				case "enter":
					return m.handleInput()
				case "backspace":
					if len(m.input) > 0 {
						m.input = m.input[:len(m.input)-1]
					}
				default:
					if len(msg.String()) == 1 {
						m.input += msg.String()
					}
				}
				return m, nil
			}
			if m.mode == "delete" && !m.confirmDelete {
				switch msg.String() {
				case "y", "n":
					m.input = msg.String()
					m.confirmDelete = true
					return m, nil
				}
			}

			switch msg.String() {
			case "a":
				m.mode = "create"
				m.input = ""
				return m, nil
			case "r":
				m.mode = "rename"
				m.input = m.files[m.cursor].Name()
				return m, nil
			case "m":
				m.mode = "move"
				m.input = ""
				return m, nil
			case "d":
				m.mode = "delete"
				m.input = ""
				m.confirmDelete = false
				return m, nil
			case "ctrl+c", "q":
				return m, tea.Quit
			case "ctrl+k":
				if m.cursor > 0 {
					newPos := m.cursor - 10
					m.cursor = max(newPos, 0)
					if m.cursor < m.offset {
						m.offset = max(m.cursor, 0)
					}
				}
			case "up", "k":
				if m.cursor > 0 {
					m.cursor--
					if m.cursor < m.offset {
						m.offset = max(m.cursor, 0)
					}
				}
			case "ctrl+j":
				if m.cursor < len(m.files)-1 {
					newPos := m.cursor + 10
					m.cursor = min(newPos, len(m.files)-1)
					if m.cursor >= m.offset+m.visibleItems {
						m.offset = min(m.cursor-m.visibleItems+11, len(m.files)-m.visibleItems)
					}
				}
			case "down", "j":
				if m.cursor < len(m.files)-1 {
					m.cursor++
					if m.cursor >= m.offset+m.visibleItems {
						m.offset = min(m.cursor-m.visibleItems+1, len(m.files)-m.visibleItems)
					}
				}
			case "ctrl+u":
				m.cursor = 0
				if m.cursor < m.offset {
					m.offset = max(m.cursor, 0)
				}
			case "ctrl+d":
				m.cursor = len(m.files) - 1
				if m.cursor >= m.offset+m.visibleItems {
					m.offset = min(m.cursor-m.visibleItems+1, len(m.files)-m.visibleItems)
				}
			case "enter", "l":
				return m.handleEnter()
			case " ":
				m.loadPreview()
				m.preview = true
			case "b", "h":
				m.navigateBack()
			case "ctrl+o":
				m.ExitWithDir = true
				return m, tea.Quit
			case "ctrl+s":
				if m.isRemote {
					m.disconnectSFTP()
					return m, tea.Println("Отключено от SFTP")
				} else {
					config, err := loadSFTPConfig()
					if err == nil && config.Host != "" && config.User != "" && config.Password != "" {
						m.mode = "sftp_confirm"
						m.input = ""
						return m, nil
					}

					m.mode = "sftp_host"
					m.input = ""
					return m, nil
				}
			case "ctrl+x":
				if !m.isRemote {
					return m, tea.Println("SFTP не подключен")
				}
				return m, m.downloadFile()
			}
		}
	}
	return m, nil
}
