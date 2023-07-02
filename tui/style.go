package tui

import "github.com/charmbracelet/lipgloss"

var (
	errorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("red")).Bold(true)
	titleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("blue")).Bold(true)
	textStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("gray"))
)
