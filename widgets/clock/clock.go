// Package clock provides a terminal UI component for displaying a live clock.
package clock

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Model struct {
	Time time.Time
}

func New() Model {
	return Model{Time: time.Now()}
}

func (m Model) Init() tea.Cmd {
	return Tick()
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg.(type) {
	case TickMsg:
		m.Time = time.Now()
		return m, Tick()
	}
	return m, nil
}

func (m Model) View() string {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#61afef")).
		Render(m.Time.Format("15:04:05"))
}

type TickMsg struct{}

func Tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return TickMsg{}
	})
}
