package handlers

import (
	"errors"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/filepicker"
	tea "github.com/charmbracelet/bubbletea"
)

type FilePickerModel struct {
	Filepicker   filepicker.Model
	SelectedFile string
	Quitting     bool
	Err          error
}

type clearErrorMsg struct{}

func clearErrorAfter(t time.Duration) tea.Cmd {
	return tea.Tick(t, func(_ time.Time) tea.Msg {
		return clearErrorMsg{}
	})
}

func (m FilePickerModel) Init() tea.Cmd {
	return m.Filepicker.Init()
}

func (m FilePickerModel) Update(msg tea.Msg) (FilePickerModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.Quitting = true
			return m, tea.Quit
		}
	case clearErrorMsg:
		m.Err = nil
	}

	var cmd tea.Cmd
	m.Filepicker, cmd = m.Filepicker.Update(msg)

	// Did the user select a file?
	if didSelect, path := m.Filepicker.DidSelectFile(msg); didSelect {
		// Get the path of the selected file.
		m.SelectedFile = path
	}

	// Did the user select a disabled file?
	// This is only necessary to display an error to the user.
	if didSelect, path := m.Filepicker.DidSelectDisabledFile(msg); didSelect {
		// Let's clear the selectedFile and display an error.
		m.Err = errors.New(path + " is not valid.")
		m.SelectedFile = ""
		return m, tea.Batch(cmd, clearErrorAfter(2*time.Second))
	}

	return m, cmd
}

func (m FilePickerModel) View() string {
	if m.Quitting {
		return ""
	}
	var s strings.Builder
	s.WriteString("\n  ")
	if m.Err != nil {
		s.WriteString(m.Filepicker.Styles.DisabledFile.Render(m.Err.Error()))
	} else if m.SelectedFile == "" {
		s.WriteString("Pick a file:")
	} else {
		s.WriteString("Selected file: " + m.Filepicker.Styles.Selected.Render(m.SelectedFile))
	}
	s.WriteString("\n\n" + m.Filepicker.View() + "\n")
	return s.String()
}

func (m FilePickerModel) InitializeFilePicker() FilePickerModel {
	fp := filepicker.New()
	fp.AllowedTypes = []string{".mod", ".sum", ".go", ".txt", ".md"}
	fp.CurrentDirectory, _ = os.Getwd()
	m.Filepicker = fp
	return m
}
