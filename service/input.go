package service

import (
	"fmt"
	"github.com/KharpukhaevV/filemanager/models"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

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
