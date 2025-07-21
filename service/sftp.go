package service

import (
	"encoding/json"
	"fmt"
	"github.com/KharpukhaevV/filemanger/models"
	"github.com/KharpukhaevV/filemanger/utils"
	"io"
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// ===================== Работа с SFTP =====================

func loadSFTPConfig() (*models.SFTPConfig, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	configPath := filepath.Join(homeDir, models.ConfigFileName)
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config models.SFTPConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

func saveSFTPConfig(config *models.SFTPConfig) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	configPath := filepath.Join(homeDir, models.ConfigFileName)
	return os.WriteFile(configPath, data, 0600)
}

func (m *FileManagerState) readRemoteFiles(dir string) []os.FileInfo {
	files, err := m.SftpClient.ReadDir(dir)
	if err != nil {
		fmt.Printf("Ошибка чтения директории %s: %v\n", dir, err)
		return nil
	}

	utils.SortFiles(files)
	return files
}

func (m *FileManagerState) initSFTP() error {
	config := &ssh.ClientConfig{
		User: m.remoteUser,
		Auth: []ssh.AuthMethod{
			ssh.Password(m.remotePassword),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	client, err := ssh.Dial("tcp", m.remoteHost, config)
	if err != nil {
		return fmt.Errorf("не удалось подключиться к серверу: %v", err)
	}

	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("не удалось создать сессию: %v", err)
	}

	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		return fmt.Errorf("не удалось создать SFTP клиент: %v", err)
	}

	m.SftpClient = sftpClient
	m.SftpSession = session
	m.isRemote = true
	m.switchToRemote()

	return nil
}

func (m *FileManagerState) switchToRemote() {
	m.Cwd = "/"
	m.files = m.readRemoteFiles(m.Cwd)
	m.cursorPositions = make(map[string]int)
}

func (m *FileManagerState) disconnectSFTP() {
	if m.SftpClient != nil {
		m.SftpClient.Close()
		m.SftpClient = nil
	}
	if m.SftpSession != nil {
		m.SftpSession.Close()
		m.SftpSession = nil
	}
	m.isRemote = false
	m.remoteHost = ""
	m.remoteUser = ""
	m.remotePassword = ""
	m.Cwd, _ = os.Getwd()
	m.files = readFiles(m.Cwd, m)
}

func (m *FileManagerState) downloadFile() tea.Cmd {
	// Проверяем, что курсор указывает на файл, а не папку
	if m.cursor < 0 || m.cursor >= len(m.files) {
		return tea.Println("Нет файла для скачивания")
	}

	selected := m.files[m.cursor]
	if selected.IsDir() {
		return tea.Println("Нельзя скачивать папки. Выберите файл.")
	}

	remotePath := filepath.ToSlash(filepath.Join(m.Cwd, selected.Name()))

	// Создаем структуру для папки загрузок
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return tea.Println("Ошибка получения домашней директории:", err)
	}

	now := time.Now()
	downloadDir := filepath.Join(
		homeDir,
		models.DownloadDir,
		fmt.Sprintf("%d", now.Year()),
		fmt.Sprintf("%02d", now.Month()),
		fmt.Sprintf("%02d", now.Day()),
	)

	// Создаем директорию, если она не существует
	if err := os.MkdirAll(downloadDir, 0755); err != nil {
		return tea.Println("Ошибка создания директории для скачивания:", err)
	}

	localPath := filepath.Join(downloadDir, selected.Name())

	// Открываем файлы
	remoteFile, err := m.SftpClient.Open(remotePath)
	if err != nil {
		return tea.Println("Ошибка открытия файла на сервере:", err)
	}
	defer remoteFile.Close()

	localFile, err := os.Create(localPath)
	if err != nil {
		return tea.Println("Ошибка создания локального файла:", err)
	}
	defer localFile.Close()

	// Копируем с прогрессом
	fileSize := selected.Size()
	buf := make([]byte, 32*1024) // 32KB буфер
	var totalRead int64

	for {
		n, err := remoteFile.Read(buf)
		if n > 0 {
			if _, err := localFile.Write(buf[:n]); err != nil {
				return tea.Println("Ошибка записи в файл:", err)
			}
			totalRead += int64(n)
			progress := int(float64(totalRead) / float64(fileSize) * 100)
			m.status = fmt.Sprintf("Скачивание %s: %d%%", selected.Name(), progress)
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			return tea.Println("Ошибка чтения файла:", err)
		}
	}

	m.status = fmt.Sprintf("Файл %s скачан в %s", selected.Name(), downloadDir)
	return nil
}
