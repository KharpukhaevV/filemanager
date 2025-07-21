package main

import (
	"fmt"
	"github.com/KharpukhaevV/filemanager/service"
	"github.com/KharpukhaevV/filemanager/utils"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
)

// ===================== Основная функция =====================

func main() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("Ошибка получения домашней директории: %v\n", err)
	}
	configPath := filepath.Join(homeDir, ".filemanager")
	_, err = os.Stat(configPath)
	if os.IsNotExist(err) {
		os.Mkdir(configPath, 0755)
	}

	stylesConfig, err := utils.LoadStylesConfig()
	if err != nil {
		fmt.Printf("Ошибка загрузки стилей: %v\n", err)
		return
	}
	utils.InitStyles(stylesConfig)

	p := tea.NewProgram(service.InitialModel(), tea.WithAltScreen())
	m, err := p.Run()
	if err != nil {
		fmt.Printf("Ошибка: %v", err)
		os.Exit(1)
	}

	if mdl, ok := m.(*service.FileManagerState); ok {
		if mdl.ArchiveReader != nil {
			mdl.ArchiveReader.Close()
		}
		if mdl.RemoteArchiveFile != nil {
			mdl.RemoteArchiveFile.Close()
		}
		if mdl.SftpClient != nil {
			mdl.SftpClient.Close()
		}
		if mdl.SftpSession != nil {
			mdl.SftpSession.Close()
		}
		if mdl.ExitWithDir {
			fmt.Printf("cd %s\n", mdl.Cwd)
		}
	}
}
