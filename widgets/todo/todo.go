// Package todo provides a terminal-based todo list interface using Bubble Tea.
// It supports adding, editing, toggling, and deleting tasks with persistent storage.
package todo

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ListState int

const (
	ListStateDefault ListState = iota
	ListStateAdding
	ListStateEditing
)

var (
	itemStyle         = lipgloss.NewStyle().PaddingLeft(2)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(0).Foreground(lipgloss.Color("#56b6c2"))
	completedStyle    = lipgloss.NewStyle().Strikethrough(true).Foreground(lipgloss.Color("#5c6370"))
)

type task struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Done        bool   `json:"done"`
}

func (t task) FilterValue() string { return t.Title }

type itemDelegate struct{}

func (d itemDelegate) Height() int                               { return 1 }
func (d itemDelegate) Spacing() int                              { return 0 }
func (d itemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	t, ok := listItem.(task)
	if !ok {
		return
	}
	str := fmt.Sprintf("[%s] %s", " ", t.Title)
	if t.Done {
		str = fmt.Sprintf("[%s] %s", "x", t.Title)
	}
	if index == m.Index() {
		var style lipgloss.Style
		if t.Done {
			style = selectedItemStyle.Strikethrough(true)
		} else {
			style = selectedItemStyle
		}
		fmt.Fprint(w, style.Render("> "+str))
	} else {
		var style lipgloss.Style
		if t.Done {
			style = completedStyle
		} else {
			style = itemStyle
		}
		fmt.Fprint(w, style.Render("  "+str))
	}
}

type Model struct {
	List      list.Model
	TextInput textinput.Model
	State     ListState
	keys      KeyMap
	path      string
}

type KeyMap struct {
	AddTask  key.Binding
	Delete   key.Binding
	Toggle   key.Binding
	EditTask key.Binding
	SaveTask key.Binding
	Confirm  key.Binding
	Cancel   key.Binding
}

func New(keys KeyMap, path string) Model {
	tasks := loadTasks(path)
	items := make([]list.Item, len(tasks))
	for i, t := range tasks {
		items[i] = t
	}

	delegate := itemDelegate{}
	l := list.New(items, delegate, 0, 0)
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetShowTitle(false)

	ti := textinput.New()
	ti.Placeholder = "New task..."
	ti.CharLimit = 156

	return Model{
		List:      l,
		TextInput: ti,
		State:     ListStateDefault,
		keys:      keys,
		path:      path,
	}
}

func (m *Model) Update(msg tea.Msg, focused bool) (Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	if focused {
		switch m.State {
		case ListStateAdding, ListStateEditing:
			switch msg := msg.(type) {
			case tea.KeyMsg:
				switch {
				case key.Matches(msg, m.keys.SaveTask), key.Matches(msg, m.keys.Confirm):
					if m.State == ListStateAdding {
						if m.TextInput.Value() != "" {
							newTask := task{Title: m.TextInput.Value()}
							m.List.InsertItem(len(m.List.Items()), newTask)
						}
					} else { // ListStateEditing
						if i, ok := m.List.SelectedItem().(task); ok {
							i.Title = m.TextInput.Value()
							m.List.SetItem(m.List.Index(), i)
						}
					}
					m.TextInput.Reset()
					m.State = ListStateDefault
					m.saveTasks()
				case key.Matches(msg, m.keys.Cancel):
					m.State = ListStateDefault
					m.TextInput.Reset()
				}
			}
			m.TextInput, cmd = m.TextInput.Update(msg)
			cmds = append(cmds, cmd)
		case ListStateDefault:
			switch msg := msg.(type) {
			case tea.KeyMsg:
				if m.List.FilterState() == list.Filtering {
					break
				}
				switch {
				case key.Matches(msg, m.keys.AddTask):
					m.State = ListStateAdding
					m.TextInput.Focus()
					return *m, textinput.Blink
				case key.Matches(msg, m.keys.EditTask):
					if i, ok := m.List.SelectedItem().(task); ok {
						m.State = ListStateEditing
						m.TextInput.SetValue(i.Title)
						m.TextInput.Focus()
						return *m, textinput.Blink
					}
				case key.Matches(msg, m.keys.Toggle):
					if i, ok := m.List.SelectedItem().(task); ok {
						i.Done = !i.Done
						m.List.SetItem(m.List.Index(), i)
						m.saveTasks()
					}
				case key.Matches(msg, m.keys.Delete):
					if len(m.List.Items()) > 0 {
						m.List.RemoveItem(m.List.Index())
						m.saveTasks()
					}
				}
			}
			m.List, cmd = m.List.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return *m, tea.Batch(cmds...)
}

func (m *Model) View() string {
	if m.State == ListStateAdding || m.State == ListStateEditing {
		return lipgloss.JoinVertical(lipgloss.Left, m.List.View(), m.TextInput.View())
	}
	return m.List.View()
}

func (m *Model) SetSize(width, height int) {
	m.TextInput.Width = width
	m.List.SetSize(width, height)
	if m.State == ListStateAdding || m.State == ListStateEditing {
		m.List.SetSize(width, height-lipgloss.Height(m.TextInput.View()))
	}
}

func (m *Model) GetState() ListState {
	return m.State
}

func (m *Model) saveTasks() {
	saveTasks(m.path, m.List.Items())
}

func saveTasks(path string, items []list.Item) {
	tasks := make([]task, len(items))
	for i, item := range items {
		tasks[i] = item.(task)
	}

	data, err := json.Marshal(tasks)
	if err != nil {
		return
	}

	os.WriteFile(path, data, 0644)
}

func loadTasks(path string) []task {
	data, err := os.ReadFile(path)
	if err != nil {
		// If the file doesn't exist, create it with default tasks
		if os.IsNotExist(err) {
			defaultTasks := []task{
				{Title: "Welcome to GoDash!"},
				{Title: "Press 'o' to add a new task"},
				{Title: "Press 'i' to edit a task"},
				{Title: "Use the arrow keys to navigate"},
				{Title: "Press 'space' to complete a task"},
				{Title: "Press 'enter' to confirm edit"},
				{Title: "Press 'esc' to cancel edit"},
				{Title: "Press 'ctrl+d' to delete a task"},
			}
			// Convert to []list.Item to use saveTasks
			items := make([]list.Item, len(defaultTasks))
			for i, t := range defaultTasks {
				items[i] = t
			}
			saveTasks(path, items)
			return defaultTasks
		}
		// For any other error, return an empty list
		return []task{}
	}

	var tasks []task
	json.Unmarshal(data, &tasks)
	return tasks
}
