// Package notes provides functionality for managing and displaying notes in a TUI application.
package notes

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"GoDash/internal/config"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// EditNoteMsg is a message sent when a note is to be edited.
type EditNoteMsg struct {
	Path    string
	Content []byte
}

type NoteState int

const (
	NoteStateList NoteState = iota
	NoteStateCreate
)

// note represents a single note in the list.
type note struct {
	title string
	path  string
}

// These methods implement the list.Item interface.
func (n note) Title() string       { return n.title }
func (n note) Description() string { return "" }
func (n note) FilterValue() string { return n.title }

type itemDelegate struct{}

func (d itemDelegate) Height() int                               { return 1 }
func (d itemDelegate) Spacing() int                              { return 0 }
func (d itemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	n, ok := listItem.(note)
	if !ok {
		return
	}

	str := n.title
	// Render selected state
	if index == m.Index() {
		fmt.Fprint(w, lipgloss.NewStyle().PaddingLeft(0).Foreground(lipgloss.Color("#56b6c2")).Render("> "+str))
	} else {
		// Render unselected state
		fmt.Fprint(w, lipgloss.NewStyle().PaddingLeft(2).Render("  "+str))
	}
}

var (
	noteBoxStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 2)
)

type Model struct {
	List         list.Model
	TextInput    textinput.Model
	State        NoteState
	keys         KeyMap
	width, height int
}

type KeyMap struct {
	CreateNote key.Binding
	DeleteNote key.Binding
	EditNote   key.Binding
	SaveNote   key.Binding
	Confirm    key.Binding
	Cancel     key.Binding
}

func New(keys KeyMap) Model {
	notes, err := loadNotes()
	if err != nil {
		// Handle error, maybe return a model with the error set
		fmt.Println("Error loading notes:", err)
	}

	items := make([]list.Item, len(notes))
	for i, n := range notes {
		items[i] = n
	}

	delegate := itemDelegate{}
	l := list.New(items, delegate, 0, 0)
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetShowTitle(false)

	ti := textinput.New()
	ti.Placeholder = "New note title..."
	ti.CharLimit = 100

	return Model{
		List:           l,
		TextInput:      ti,
		State:          NoteStateList,
		keys:           keys,
	}
}

func sanitizeFilename(name string) string {
	// Replace spaces with hyphens
	name = strings.ReplaceAll(name, " ", "-")
	// Remove any other invalid characters
	reg, err := regexp.Compile("[^a-zA-Z0-9-]+")
	if err != nil {
		// This should not happen with a static regex
		return "sanitization-error"
	}
	sanitized := reg.ReplaceAllString(name, "")
	if sanitized == "" {
		return "untitled-note"
	}
	return sanitized
}

func loadNotes() ([]note, error) {
	notesDir, err := config.GetNotesDir()
	if err != nil {
		return nil, err
	}

	files, err := os.ReadDir(notesDir)
	if err != nil {
		return nil, err
	}

	noteCount := 0
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".md") {
			noteCount++
		}
	}

	// Only create default notes on first run, not every time notes directory is empty
	if noteCount == 0 {
		settings, err := config.LoadSettings()
		if err == nil && !settings.DefaultNotesCreated {
			createDefaultNotes(notesDir)
			
			// Mark default notes as created
			settings.DefaultNotesCreated = true
			config.SaveSettings(settings)
			
			// Re-read files after creating the default ones
			files, err = os.ReadDir(notesDir)
			if err != nil {
				return nil, err
			}
		}
	}

	var notes []note
	re := regexp.MustCompile(`^\d+\s`)
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".md") {
			title := strings.TrimSuffix(file.Name(), ".md")
			title = strings.ReplaceAll(title, "-", " ") // Replace hyphens with spaces for display
			title = re.ReplaceAllString(title, "")      // Strip numerical prefix
			notes = append(notes, note{
				title: title,
				path:  filepath.Join(notesDir, file.Name()),
			})
		}
	}
	return notes, nil
}

func createDefaultNotes(dir string) {
	welcomeTitle := "01 Welcome to GoDash"
	welcomeContent := `# 🐻‍❄️ Welcome to GoDash

> **Your Terminal Personal Productivity Dashboard**

Welcome to GoDash, a modern terminal-based productivity suite that brings all your essential tools into one beautiful interface. Built with Go and the Charm ecosystem, GoDash seamlessly combines task management, note-taking, calendar integration, and weather updates in a sleek TUI experience.

---

## 🌟 **What You Can Do**

### 📋 **Task Management**
- Create, edit, delete, and organize your tasks
- Toggle completion status with a single keystroke
- Persistent storage with automatic saving

### 📝 **Smart Notes**
- Full **Markdown support** with live preview
- Dual-mode editor (preview ↔ source editing)
- File-based storage with auto-save confirmation
- Unsaved changes protection

### 📅 **Calendar Integration**
- **Google Calendar OAuth 2.0** with automatic flow
- View today's events and navigate dates
- Quick browser access to full calendar

### 🌤️ **Weather & Time**
- Real-time weather using **wttr.in** (no API key needed)
- Current conditions with beautiful ASCII art
- Always-current digital clock

---

## 🚀 **Getting Started**

### **Navigation**
- **Tab** - Cycle between panels
- **Mouse clicks** - Quick panel switching
- **Ctrl+K** - Show contextual help
- **Ctrl+Q** - Quit application

### **First Time Setup**
1. Enter your city name for weather
2. Authenticate with Google Calendar
3. Start being productive!

---

## 💡 **Pro Tips**

- Use **Ctrl+S** in note editor for instant save confirmation
- **ESC** from edit mode shows unsaved changes protection
- Calendar authentication works with automatic port detection
- Interface adapts to your terminal size automatically

---

**🎨 Crafted with the One Dark theme and polar bear charm**  
**💻 Built for developers, by developers**

Made with ❤️ by **Hellas Dev**
`
	keybindingsTitle := "02 Keybindings for the entire GoDash app"
	keybindingsContent := `# ⌨️ **GoDash Keyboard Reference**

> **Master GoDash with these essential keyboard shortcuts**

GoDash is designed for keyboard efficiency. Each panel has its own set of keybindings that activate when focused. Use **Ctrl+K** anytime to see contextual help for the active panel.

---

## 🌍 **Global Controls**

| Key | Action | Description |
|-----|--------|-------------|
| **Tab** | Cycle Focus | Switch between Todo, Notes, and Calendar panels |
| **Ctrl+K** | Show Help | Display keybindings for the current panel |
| **Ctrl+Q** | Quit App | Exit GoDash completely |

---

## 📋 **Todo List Panel**

### **Task Management**
| Key | Action | Description |
|-----|--------|-------------|
| **o** | Add Task | Create a new task |
| **i** | Edit Task | Edit the selected task |
| **Space** | Toggle Complete | Mark task as done/undone |
| **Ctrl+D** | Delete Task | Remove the selected task |
| **↑ / ↓** | Navigate | Move through your task list |

### **While Editing Tasks**
| Key | Action | Description |
|-----|--------|-------------|
| **Enter** | Confirm | Save your changes |
| **Esc** | Cancel | Discard changes and return to list |
| **Ctrl+S** | Save Task | Alternative save method |

---

## 📝 **Notes Panel**

### **Note Navigation**
| Key | Action | Description |
|-----|--------|-------------|
| **o** | New Note | Create a new markdown note |
| **e** | Edit Note | Open selected note in editor |
| **Ctrl+D** | Delete Note | Remove the selected note |
| **↑ / ↓** | Navigate | Browse through your notes |
| **Enter** | Open Note | View/edit the selected note |

### **Note Editor Controls**
| Key | Action | Description |
|-----|--------|-------------|
| **i** | Toggle Mode | Switch between Preview ↔ Edit modes |
| **Ctrl+S** | Save Note | Save with visual confirmation |
| **Esc** | Exit Editor | Return to notes list (with unsaved changes protection) |

> **💡 Pro Tip:** The 'i' key only works for mode switching in Preview mode. In Edit mode, you can type freely including the letter 'i'.

---

## 📅 **Calendar Panel**

| Key | Action | Description |
|-----|--------|-------------|
| **↑ ↓ ← →** | Navigate Calendar | Move through dates and months |
| **Enter** | Open Calendar | Launch Google Calendar in your browser |
| **Ctrl+O** | Re-authorize | Refresh your Google Calendar connection |

---

## ⚡ **Quick Tips**

- **Mouse Support**: Click any panel to focus it instantly
- **Visual Feedback**: Focused panels have blue borders
- **Save Confirmation**: Look for "✅ Note saved!" message
- **Unsaved Changes**: Get prompted before losing your work
- **Dynamic Help**: Press Ctrl+K in any panel for contextual shortcuts

---

**🎯 Designed for maximum productivity and minimal friction**  
**⚡ Every keystroke optimized for your workflow**
`

	welcomeFilename := sanitizeFilename(welcomeTitle) + ".md"
	keybindingsFilename := sanitizeFilename(keybindingsTitle) + ".md"

	os.WriteFile(filepath.Join(dir, welcomeFilename), []byte(welcomeContent), 0644)
	os.WriteFile(filepath.Join(dir, keybindingsFilename), []byte(keybindingsContent), 0644)
}

func (m *Model) Update(msg tea.Msg, focused bool) (Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	if focused {
		switch m.State {
		case NoteStateCreate:
			switch msg := msg.(type) {
			case tea.KeyMsg:
				switch {
				case key.Matches(msg, m.keys.Cancel):
					m.State = NoteStateList
					m.TextInput.Reset()
				case key.Matches(msg, m.keys.Confirm):
					title := m.TextInput.Value()
					if title != "" {
						notesDir, _ := config.GetNotesDir()
						filename := sanitizeFilename(title) + ".md"
						filePath := filepath.Join(notesDir, filename)

						os.WriteFile(filePath, []byte("# "+title+"\n\n"), 0644)

						newNote := note{title: title, path: filePath}
						m.List.InsertItem(len(m.List.Items()), newNote)

						m.TextInput.Reset()
						m.State = NoteStateList
					}
				}
			}
			m.TextInput, cmd = m.TextInput.Update(msg)
			cmds = append(cmds, cmd)

		case NoteStateList:
			switch msg := msg.(type) {
			case tea.KeyMsg:
				if m.List.FilterState() == list.Filtering {
					break
				}
				switch {
				case key.Matches(msg, m.keys.CreateNote):
					m.State = NoteStateCreate
					m.TextInput.Focus()
					return *m, textinput.Blink
				case key.Matches(msg, m.keys.DeleteNote):
					if len(m.List.Items()) > 0 {
						if selected, ok := m.List.SelectedItem().(note); ok {
							os.Remove(selected.path)
							m.List.RemoveItem(m.List.Index())
						}
					}
				case key.Matches(msg, m.keys.Confirm): // Enter key
					if selected, ok := m.List.SelectedItem().(note); ok {
						content, err := os.ReadFile(selected.path)
						if err != nil {
							// Handle error appropriately, maybe return a message to display
							content = []byte("Could not read file: " + err.Error())
						}
						editCmd := func() tea.Msg {
							return EditNoteMsg{Path: selected.path, Content: content}
						}
						return *m, editCmd
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
	switch m.State {
	case NoteStateCreate:
		return lipgloss.JoinVertical(lipgloss.Left, m.List.View(), m.TextInput.View())
	default: // NoteStateList
		return m.List.View()
	}
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height

	m.List.SetSize(width, height)
	m.TextInput.Width = width

	if m.State == NoteStateCreate {
		m.List.SetSize(width, height-lipgloss.Height(m.TextInput.View()))
	}
}

func (m Model) Reload() Model {
	notes, err := loadNotes()
	if err != nil {
		fmt.Println("Error reloading notes:", err)
		return m
	}
	items := make([]list.Item, len(notes))
	for i, n := range notes {
		items[i] = n
	}
	m.List.SetItems(items)
	return m
}
