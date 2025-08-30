package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"

	"GoDash/internal/config"
	calendarwidget "GoDash/widgets/calendar"
	"GoDash/widgets/notes"
	"GoDash/widgets/todo"
)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// renderAppLogo returns a stylized ASCII logo for GoDash
func renderAppLogo() string {
	logo := "🐻‍❄️ GoDash 🐻‍❄️"
	return logoStyle.Render(logo)
}

// openURLInBrowser opens a URL in the default browser, attempting to reuse existing windows/tabs
func openURLInBrowser(url string) error {
	var cmd *exec.Cmd
	
	switch runtime.GOOS {
	case "linux":
		// Try common browsers with flags to reuse existing windows
		if isCommandAvailable("firefox") {
			// Firefox: --new-tab opens in existing window if available
			cmd = exec.Command("firefox", "--new-tab", url)
		} else if isCommandAvailable("google-chrome") {
			// Chrome: no new window flag, opens in existing by default
			cmd = exec.Command("google-chrome", url)
		} else if isCommandAvailable("chromium") {
			cmd = exec.Command("chromium", url)
		} else {
			// Fallback to xdg-open
			cmd = exec.Command("xdg-open", url)
		}
	case "darwin":
		// On macOS, try to use specific browser commands
		if isCommandAvailable("firefox") {
			cmd = exec.Command("firefox", "--new-tab", url)
		} else {
			// macOS open command
			cmd = exec.Command("open", url)
		}
	default:
		// Fallback for other systems
		cmd = exec.Command("xdg-open", url)
	}
	
	return cmd.Start()
}

// isCommandAvailable checks if a command is available in the system PATH
func isCommandAvailable(command string) bool {
	_, err := exec.LookPath(command)
	return err == nil
}

// App States
type appState int

const (
	stateDashboard appState = iota
	stateEditingNote
	stateSetupWeather
	stateSetupCalendar
	stateExitConfirmation
)

// Note Editor Modes
type noteEditorMode int

const (
	notePreviewMode noteEditorMode = iota
	noteSourceMode
)

const (
	minWidth  = 140
	minHeight = 35
)

// Dashboard Focus States
type focusState int

const (
	focusList focusState = iota
	focusNotes
	focusCalendar
)

// --- STYLES ---
var (
	boxStyle          = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#4b5263")).Padding(1, 2)
	focusedBoxStyle   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#61afef")).Padding(1, 2)
	itemStyle         = lipgloss.NewStyle().PaddingLeft(2)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(0).Foreground(lipgloss.Color("#56b6c2"))
	titleStyle        = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#e06c75"))
	logoStyle         = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#61afef"))
	helpTitleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("1"))
	saveMessageStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#98c379")).Bold(true)
	helpBoxStyle      = lipgloss.NewStyle().Padding(1, 2).Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#7287fd"))
	yellowText        = lipgloss.NewStyle().Foreground(lipgloss.Color("226"))
	blueText          = lipgloss.NewStyle().Foreground(lipgloss.Color("81"))
	orangeText        = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	redText           = lipgloss.NewStyle().Foreground(lipgloss.Color("#e06c75"))
)

// --- KEYS ---
type keyMap struct {
	AddTask         key.Binding
	Delete          key.Binding
	Toggle          key.Binding
	EditTask        key.Binding
	SaveTask        key.Binding
	Confirm         key.Binding
	OpenLink        key.Binding
	OpenCalendar    key.Binding
	Cancel          key.Binding
	CreateNote      key.Binding
	DeleteNote      key.Binding
	EditNote        key.Binding
	SaveNote        key.Binding
	ToggleEditMode  key.Binding
	CycleFocus      key.Binding
	ShowHelp        key.Binding
	Quit            key.Binding
}


var keys = keyMap{
	AddTask:        key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "add task")),
	Delete:         key.NewBinding(key.WithKeys("ctrl+d"), key.WithHelp("ctrl+d", "delete task")),
	Toggle:         key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "toggle task")),
	EditTask:       key.NewBinding(key.WithKeys("i"), key.WithHelp("i", "edit task")),
	SaveTask:       key.NewBinding(key.WithKeys("ctrl+s"), key.WithHelp("ctrl+s", "save task")),
	Confirm:        key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "confirm")),
	OpenLink:       key.NewBinding(key.WithKeys("ctrl+o"), key.WithHelp("ctrl+o", "open/authorize")),
	OpenCalendar:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "open calendar")),
	Cancel:         key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
	CreateNote:     key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "new note")),
	DeleteNote:     key.NewBinding(key.WithKeys("ctrl+d"), key.WithHelp("ctrl+d", "delete note")),
	EditNote:       key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit note")),
	SaveNote:       key.NewBinding(key.WithKeys("ctrl+s"), key.WithHelp("ctrl+s", "save note")),
	ToggleEditMode: key.NewBinding(key.WithKeys("i"), key.WithHelp("i", "toggle edit mode")),
	CycleFocus:     key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "cycle focus")),
	ShowHelp:       key.NewBinding(key.WithKeys("ctrl+k"), key.WithHelp("ctrl+k", "key bindings")),
	Quit:           key.NewBinding(key.WithKeys("ctrl+q"), key.WithHelp("ctrl+q", "quit")),
}

func (m model) ShortHelp() []key.Binding {
	return []key.Binding{m.keys.CycleFocus, m.keys.ShowHelp, m.keys.Quit}
}

func (m model) FullHelp() [][]key.Binding {
	switch m.focus {
	case focusCalendar:
		return [][]key.Binding{
			{m.keys.OpenCalendar},
			{m.keys.CycleFocus, m.keys.ShowHelp, m.keys.Quit},
		}
	case focusNotes:
		// This is a temporary keybinding for display in the help view.
		exitEditorKey := key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel"))
		switch m.notes.State {
		case notes.NoteStateCreate:
			return [][]key.Binding{
				{m.keys.Confirm, m.keys.Cancel},
				{m.keys.CycleFocus, m.keys.ShowHelp, m.keys.Quit},
			}
		default: // NoteStateList
			return [][]key.Binding{
				{m.keys.CreateNote, m.keys.DeleteNote, m.keys.EditNote, m.keys.Confirm},
				{m.keys.SaveNote, m.keys.ToggleEditMode, exitEditorKey},
				{m.keys.CycleFocus, m.keys.ShowHelp, m.keys.Quit},
			}
		}
	default: // focusList
		if m.todo.State == todo.ListStateAdding || m.todo.State == todo.ListStateEditing {
			return [][]key.Binding{
				{m.keys.SaveTask, m.keys.Cancel},
				{m.keys.CycleFocus, m.keys.ShowHelp, m.keys.Quit},
			}
		}
		return [][]key.Binding{
			{m.keys.AddTask, m.keys.Delete, m.keys.Toggle, m.keys.EditTask},
			{m.keys.Confirm, m.keys.Cancel, m.keys.CycleFocus, m.keys.ShowHelp, m.keys.Quit},
		}
	}
}

// --- MODEL ---
type model struct {
	state            appState
	focus            focusState
	todo             todo.Model
	notes            notes.Model
	calendar         calendarwidget.Model
	noteEditor       textarea.Model
	noteViewer       viewport.Model
	noteEditorMode   noteEditorMode
	noteContent      string
	editingNotePath  string
	setupTextInput   textinput.Model
	help             help.Model
	keys             keyMap
	settings         config.Settings
	width, height    int
	spinner          spinner.Model
	showHelp         bool
	calendarAuthURL  string
	err              error
	markdownRenderer *glamour.TermRenderer
	saveMessage      string
	saveMessageTimer int
	hasUnsavedChanges bool
	originalContent  string
	confirmationChoice int // 0 = Yes, 1 = No
}

// tickMsg is sent periodically to update the save message timer
type tickMsg time.Time

// tickCmd sends a tick every second
func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func initialModel(settings config.Settings) model {
	setupTI := textinput.New()
	setupTI.Placeholder = "Enter your API key here..."
	setupTI.Focus()
	setupTI.CharLimit = 64
	setupTI.Width = 50

	h := help.New()
	h.ShowAll = true
	h.Styles.FullKey = yellowText
	h.Styles.FullDesc = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	h.Styles.FullSeparator = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	noteTa := textarea.New()
	noteTa.Placeholder = "Your notes here..."
	noteTa.ShowLineNumbers = true

	noteVp := viewport.New(80, 24)
	
	// Initialize markdown renderer
	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(80),
	)
	if err != nil {
		renderer = nil
	}

	todoKeys := todo.KeyMap{
		AddTask:    keys.AddTask,
		Delete:     keys.Delete,
		Toggle:     keys.Toggle,
		EditTask:   keys.EditTask,
		SaveTask:   keys.SaveTask,
		Confirm:    keys.Confirm,
		Cancel:     keys.Cancel,
	}

	noteKeys := notes.KeyMap{
		CreateNote: keys.CreateNote,
		DeleteNote: keys.DeleteNote,
		EditNote:   keys.EditNote,
		SaveNote:   keys.SaveNote,
		Confirm:    keys.Confirm,
		Cancel:     keys.Cancel,
	}

	calendarKeys := calendarwidget.KeyMap{
		Confirm: keys.Confirm,
	}

	todoPath, err := config.GetTodoPath()
	if err != nil {
		fmt.Println("could not get todo path:", err)
		os.Exit(1)
	}

	m := model{
		spinner:          s,
		todo:             todo.New(todoKeys, todoPath),
		notes:            notes.New(noteKeys),
		noteEditor:       noteTa,
		noteViewer:       noteVp,
		noteEditorMode:   notePreviewMode,
		calendar:         calendarwidget.New(calendarKeys, settings.Location),
		setupTextInput:   setupTI,
		help:             h,
		keys:             keys,
		settings:         settings,
		focus:            focusList,
		markdownRenderer: renderer,
	}

	if !calendarwidget.IsAuthorized() {
		m.state = stateSetupCalendar
		authURL, err := calendarwidget.GetAuthURL()
		if err != nil {
			m.err = err
		}
		m.calendarAuthURL = authURL
		if calendarwidget.IsUsingManualFlow() {
			m.setupTextInput.Placeholder = "Paste authorization code here..."
		} else {
			m.setupTextInput.Placeholder = "Authorization will complete automatically..."
		}
		m.setupTextInput.Focus()
	} else {
		m.state = stateDashboard
	}

	m.updateKeybindings()
	return m
}

func (m model) Init() tea.Cmd {
	cmds := []tea.Cmd{textinput.Blink, textarea.Blink}
	if m.state == stateDashboard {
		cmds = append(cmds, m.calendar.Init())
	}
	return tea.Batch(cmds...)
}

func (m *model) updateKeybindings() {
	isEditingNote := m.state == stateEditingNote

	// Keybindings for the note editor
	if isEditingNote {
		m.keys.AddTask.SetEnabled(false)
		m.keys.Delete.SetEnabled(false)
		m.keys.Toggle.SetEnabled(false)
		m.keys.EditTask.SetEnabled(false)
		m.keys.Confirm.SetEnabled(false)
		m.keys.OpenLink.SetEnabled(false)
		m.keys.OpenCalendar.SetEnabled(false)
		m.keys.CreateNote.SetEnabled(false)
		m.keys.DeleteNote.SetEnabled(false)
		m.keys.EditNote.SetEnabled(false)
		m.keys.CycleFocus.SetEnabled(false)
		m.keys.ShowHelp.SetEnabled(false)

		m.keys.SaveNote.SetEnabled(m.noteEditorMode == noteSourceMode)
		m.keys.ToggleEditMode.SetEnabled(true)
		m.keys.Cancel.SetEnabled(true) // For exiting the editor
		m.keys.Quit.SetEnabled(true)
		return
	}

	// Keybindings for the dashboard
	isListFocused := m.focus == focusList
	isNotesFocused := m.focus == focusNotes
	isCalendarFocused := m.focus == focusCalendar
	isSetupWeather := m.state == stateSetupWeather
	isSetupCalendar := m.state == stateSetupCalendar
	isSetup := isSetupWeather || isSetupCalendar

	m.keys.AddTask.SetEnabled(!isSetup && isListFocused && m.todo.GetState() == todo.ListStateDefault)
	m.keys.Delete.SetEnabled(!isSetup && isListFocused && m.todo.GetState() == todo.ListStateDefault)
	m.keys.Toggle.SetEnabled(!isSetup && isListFocused && m.todo.GetState() == todo.ListStateDefault)
	m.keys.EditTask.SetEnabled(!isSetup && isListFocused && m.todo.GetState() == todo.ListStateDefault)
	m.keys.Confirm.SetEnabled((!isSetup && isListFocused && (m.todo.GetState() == todo.ListStateAdding || m.todo.GetState() == todo.ListStateEditing)) || isSetup)
	m.keys.OpenLink.SetEnabled(isSetup)
	m.keys.OpenCalendar.SetEnabled(!isSetup && isCalendarFocused)
	m.keys.CreateNote.SetEnabled(!isSetup && isNotesFocused)
	m.keys.DeleteNote.SetEnabled(!isSetup && isNotesFocused)
	m.keys.EditNote.SetEnabled(!isSetup && isNotesFocused)
	m.keys.CycleFocus.SetEnabled(!isSetup)
	m.keys.SaveNote.SetEnabled(false)
	m.keys.Cancel.SetEnabled(!isSetup)
	m.keys.ShowHelp.SetEnabled(true)
	m.keys.Quit.SetEnabled(true)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		if m.saveMessageTimer > 0 {
			m.saveMessageTimer--
			if m.saveMessageTimer == 0 {
				m.saveMessage = ""
			}
			return m, tickCmd()
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.Width = msg.Width
		m.setupTextInput.Width = min(50, m.width-10)

		// Also set size for the note editor
		editorBoxWidth := int(float64(m.width) * 0.8)
		editorBoxHeight := int(float64(m.height) * 0.8)
		hpad := focusedBoxStyle.GetHorizontalPadding()
		vpad := focusedBoxStyle.GetVerticalPadding()
		titleHeight := lipgloss.Height(titleStyle.Render("Edit Note"))

		m.noteEditor.SetWidth(editorBoxWidth - hpad)
		m.noteEditor.SetHeight(editorBoxHeight - vpad - titleHeight)
	}

	if m.err != nil {
		return m, tea.Quit
	}

	switch m.state {
	case stateEditingNote:
		return m.updateNoteEditor(msg)
	case stateExitConfirmation:
		return m.updateExitConfirmation(msg)
	case stateSetupWeather:
		return m.updateSetupWeather(msg)
	case stateSetupCalendar:
		return m.updateSetupCalendar(msg)
	case stateDashboard:
		return m.updateDashboard(msg)
	}
	return m, nil
}

// --- UPDATE: NOTE EDITOR ---
func (m model) updateNoteEditor(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.ToggleEditMode):
			if m.noteEditorMode == notePreviewMode {
				// Switch to source mode
				m.noteEditorMode = noteSourceMode
				m.noteEditor.Focus()
				m.updateKeybindings()
				return m, nil
			}
			// Note: Don't handle 'i' key when in edit mode to avoid typing conflicts
		case key.Matches(msg, m.keys.SaveNote):
			if m.noteEditorMode == noteSourceMode {
				content := m.noteEditor.Value()
				err := os.WriteFile(m.editingNotePath, []byte(content), 0644)
				if err != nil {
					m.err = fmt.Errorf("could not save note: %w", err)
					return m, nil
				}
				m.noteContent = content
				m.notes = m.notes.Reload()
				
				// Show save confirmation message
				m.saveMessage = "✅ Note saved!"
				m.saveMessageTimer = 3 // Show for 3 seconds
				m.hasUnsavedChanges = false // Reset unsaved changes flag
				m.originalContent = content // Update original content
				
				// Update preview after saving
				if m.markdownRenderer != nil {
					rendered, err := m.markdownRenderer.Render(content)
					if err != nil {
						rendered = content
					}
					m.noteViewer.SetContent(rendered)
				} else {
					m.noteViewer.SetContent(content)
				}
			}
			return m, tickCmd()
		case key.Matches(msg, m.keys.Cancel):
			if m.noteEditorMode == noteSourceMode {
				// If in source mode, check for unsaved changes before going to preview
				currentContent := m.noteEditor.Value()
				hasChanges := currentContent != m.originalContent
				
				if hasChanges {
					// Show confirmation dialog for unsaved changes
					m.hasUnsavedChanges = hasChanges
					m.state = stateExitConfirmation
					m.confirmationChoice = 1 // Default to "No"
					m.updateKeybindings()
					return m, nil
				} else {
					// No changes, go to preview mode normally
					m.noteEditorMode = notePreviewMode
					m.noteContent = currentContent
					m.noteEditor.Blur()
					
					// Update the preview
					if m.markdownRenderer != nil {
						rendered, err := m.markdownRenderer.Render(m.noteContent)
						if err != nil {
							rendered = m.noteContent
						}
						m.noteViewer.SetContent(rendered)
					} else {
						m.noteViewer.SetContent(m.noteContent)
					}
					m.updateKeybindings()
					return m, nil
				}
			} else {
				// If in preview mode, exit directly (no confirmation needed here)
				m.state = stateDashboard
				m.noteEditor.Blur()
				m.updateKeybindings()
				return m, nil
			}
		}
	}

	// Update the appropriate component based on mode
	if m.noteEditorMode == noteSourceMode {
		m.noteEditor, cmd = m.noteEditor.Update(msg)
		// Check for unsaved changes
		currentContent := m.noteEditor.Value()
		m.hasUnsavedChanges = currentContent != m.originalContent
	} else {
		m.noteViewer, cmd = m.noteViewer.Update(msg)
	}
	
	return m, cmd
}

// --- UPDATE: EXIT CONFIRMATION ---
func (m model) updateExitConfirmation(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("left", "h"))):
			m.confirmationChoice = 0 // Yes
		case key.Matches(msg, key.NewBinding(key.WithKeys("right", "l"))):
			m.confirmationChoice = 1 // No
		case key.Matches(msg, key.NewBinding(key.WithKeys("y", "Y"))):
			m.confirmationChoice = 0 // Yes
		case key.Matches(msg, key.NewBinding(key.WithKeys("n", "N"))):
			m.confirmationChoice = 1 // No
		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			if m.confirmationChoice == 0 { // Yes - Continue without saving
				// Go to preview mode without saving changes
				m.noteEditorMode = notePreviewMode
				m.noteContent = m.originalContent // Restore original content
				m.noteEditor.SetValue(m.originalContent) // Reset editor
				m.hasUnsavedChanges = false
				m.noteEditor.Blur()
				
				// Update the preview with original content
				if m.markdownRenderer != nil {
					rendered, err := m.markdownRenderer.Render(m.originalContent)
					if err != nil {
						rendered = m.originalContent
					}
					m.noteViewer.SetContent(rendered)
				} else {
					m.noteViewer.SetContent(m.originalContent)
				}
				
				m.state = stateEditingNote
				m.updateKeybindings()
				return m, nil
			} else { // No - Go back to editor
				m.state = stateEditingNote
				m.updateKeybindings()
				return m, nil
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
			// ESC goes back to editor
			m.state = stateEditingNote
			m.updateKeybindings()
			return m, nil
		}
	}
	return m, nil
}

// --- UPDATE: SETUP ---
func (m model) updateSetupWeather(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.keys.Confirm):
			city := m.setupTextInput.Value()
			if city != "" {
				m.settings.Location = city
				if err := config.SaveSettings(m.settings); err == nil {
					if !calendarwidget.IsAuthorized() {
						m.state = stateSetupCalendar
						authURL, err := calendarwidget.GetAuthURL()
						if err != nil {
							m.err = err
						}
						m.calendarAuthURL = authURL
						m.setupTextInput.Reset()
						if calendarwidget.IsUsingManualFlow() {
			m.setupTextInput.Placeholder = "Paste authorization code here..."
		} else {
			m.setupTextInput.Placeholder = "Authorization will complete automatically..."
		}
						m.setupTextInput.Focus()
						return m, textinput.Blink
					} else {
						m.state = stateDashboard
						m.updateKeybindings()
						return m, m.calendar.Init()
					}
				}
			}
		}
	}
	m.setupTextInput, cmd = m.setupTextInput.Update(msg)
	return m, cmd
}

func (m model) updateSetupCalendar(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	
	// Check if auth is complete first
	if calendarwidget.IsAuthorized() {
		m.state = stateDashboard
		m.updateKeybindings()
		return m, m.calendar.Init()
	}
	
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.keys.OpenLink):
			_ = openURLInBrowser(m.calendarAuthURL)
			if !calendarwidget.IsUsingManualFlow() {
				// Start waiting for auth completion in background for automatic flow
				go func() {
					calendarwidget.WaitForAuth()
				}()
			}
		case key.Matches(msg, m.keys.Confirm):
			if calendarwidget.IsUsingManualFlow() {
				// Manual flow - get code from text input
				authCode := m.setupTextInput.Value()
				if authCode != "" {
					err := calendarwidget.CompleteAuth(authCode)
					if err == nil {
						m.state = stateDashboard
						m.updateKeybindings()
						return m, m.calendar.Init()
					} else {
						m.err = err
					}
				}
			} else {
				// Automatic flow - just check if auth is complete
				if calendarwidget.IsAuthorized() {
					m.state = stateDashboard
					m.updateKeybindings()
					return m, m.calendar.Init()
				}
			}
		}
	}
	
	m.setupTextInput, cmd = m.setupTextInput.Update(msg)
	return m, cmd
}

// --- UPDATE: DASHBOARD ---
func (m model) updateDashboard(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	if m.todo.GetState() == todo.ListStateAdding || m.todo.GetState() == todo.ListStateEditing {
		m.todo, cmd = m.todo.Update(msg, m.focus == focusList)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	}

	switch msg := msg.(type) {
	case notes.EditNoteMsg:
		m.state = stateEditingNote
		m.editingNotePath = msg.Path
		m.noteContent = string(msg.Content)
		m.originalContent = m.noteContent // Save original for comparison
		m.hasUnsavedChanges = false
		m.noteEditor.SetValue(m.noteContent)
		m.noteEditorMode = notePreviewMode
		
		// Initialize preview
		if m.markdownRenderer != nil {
			rendered, err := m.markdownRenderer.Render(m.noteContent)
			if err != nil {
				rendered = m.noteContent
			}
			m.noteViewer.SetContent(rendered)
		} else {
			m.noteViewer.SetContent(m.noteContent)
		}
		
		m.updateKeybindings()
		return m, nil
	case tea.MouseMsg:
		if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
			leftColumnWidth := m.width * 2 / 5
			if msg.X < leftColumnWidth {
				if msg.Y < m.height/2 {
					m.focus = focusList
				} else {
					m.focus = focusCalendar
				}
			} else {
				m.focus = focusNotes
			}
			m.updateKeybindings()
		}
	}

	m.todo, cmd = m.todo.Update(msg, m.focus == focusList)
	cmds = append(cmds, cmd)
	m.notes, cmd = m.notes.Update(msg, m.focus == focusNotes)
	cmds = append(cmds, cmd)
	m.calendar, cmd = m.calendar.Update(msg, m.focus == focusCalendar)
	cmds = append(cmds, cmd)

	if msg, ok := msg.(tea.KeyMsg); ok {
		if m.focus == focusCalendar && key.Matches(msg, m.keys.OpenCalendar) {
			_ = openURLInBrowser("https://calendar.google.com/calendar/u/0/r")
		}

		if msg.String() == "q" {
			return m, nil
		}
		switch {
		case key.Matches(msg, m.keys.ShowHelp):
			m.showHelp = !m.showHelp
			return m, nil
		case key.Matches(msg, m.keys.Cancel):
			if m.showHelp {
				m.showHelp = false
			}
			return m, nil
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.keys.CycleFocus):
			if m.todo.GetState() != todo.ListStateAdding {
				m.focus = (m.focus + 1) % 3
				m.updateKeybindings()
			}
			return m, nil
		}
	}

	return m, tea.Batch(cmds...)
}


// --- VIEW ---
func (m model) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v", m.err)
	}

	if m.width < minWidth || m.height < minHeight {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
			"GoDash requires a larger terminal size to run correctly.\nPlease increase the size of your terminal window.",
		)
	}

	if m.width == 0 {
		return "Initializing..."
	}

	if m.showHelp {
		var focusedPanelTitle string
		switch m.focus {
		case focusList:
			focusedPanelTitle = "To-Do List"
		case focusNotes:
			focusedPanelTitle = "Notes"
		case focusCalendar:
			focusedPanelTitle = "Calendar"
		}

		helpView := m.help.View(m)
		title := helpTitleStyle.Render(focusedPanelTitle)
		helpContent := lipgloss.JoinVertical(lipgloss.Center, title, helpView)
		helpBox := helpBoxStyle.Render(helpContent)
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, helpBox)
	}

	switch m.state {
	case stateEditingNote:
		return m.viewNoteEditor()
	case stateExitConfirmation:
		return m.viewExitConfirmation()
	case stateSetupWeather, stateSetupCalendar:
		return m.viewSetup()
	case stateDashboard:
		return m.viewDashboard()
	}

	return ""
}

func (m model) viewNoteEditor() string {
	editorBoxWidth := int(float64(m.width) * 0.8)
	editorBoxHeight := int(float64(m.height) * 0.8)

	// Update viewport dimensions to match editor box
	hpad := focusedBoxStyle.GetHorizontalPadding()
	vpad := focusedBoxStyle.GetVerticalPadding()
	titleHeight := 1 // Title takes 1 line
	
	m.noteViewer.Width = editorBoxWidth - hpad
	m.noteViewer.Height = editorBoxHeight - vpad - titleHeight - 2 // Extra space for mode indicator
	
	// Update textarea dimensions as well
	m.noteEditor.SetWidth(editorBoxWidth - hpad)
	m.noteEditor.SetHeight(editorBoxHeight - vpad - titleHeight - 2)

	var title string
	var content string
	
	if m.noteEditorMode == notePreviewMode {
		title = titleStyle.Render("Note Preview (press 'i' to edit)")
		content = m.noteViewer.View()
	} else {
		title = titleStyle.Render("Edit Note (press 'i' to preview)")
		content = m.noteEditor.View()
	}
	
	// Add save message if present
	var editorContent string
	if m.saveMessage != "" {
		saveMessageRender := saveMessageStyle.Render(m.saveMessage)
		titleWithMessage := lipgloss.JoinHorizontal(lipgloss.Left, title, "  ", saveMessageRender)
		editorContent = lipgloss.JoinVertical(lipgloss.Left, titleWithMessage, content)
	} else {
		editorContent = lipgloss.JoinVertical(lipgloss.Left, title, content)
	}
	editorBox := focusedBoxStyle.Width(editorBoxWidth).Height(editorBoxHeight).Render(editorContent)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, editorBox)
}

func (m model) viewExitConfirmation() string {
	title := "⚠️ Unsaved Changes ⚠️"
	message := "You have unsaved changes. Discard changes and continue?"
	
	yesStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#abb2bf")).Padding(0, 1)
	noStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#abb2bf")).Padding(0, 1)
	
	if m.confirmationChoice == 0 { // Yes selected
		yesStyle = yesStyle.Background(lipgloss.Color("#e06c75")).Foreground(lipgloss.Color("#ffffff")).Bold(true)
	} else { // No selected
		noStyle = noStyle.Background(lipgloss.Color("#98c379")).Foreground(lipgloss.Color("#ffffff")).Bold(true)
	}
	
	yesButton := yesStyle.Render("Yes")
	noButton := noStyle.Render("No")
	
	buttons := lipgloss.JoinHorizontal(lipgloss.Left, yesButton, "  ", noButton)
	
	instructions := "Use ←/→ or Y/N to choose, Enter to confirm, Esc to cancel"
	
	content := lipgloss.JoinVertical(lipgloss.Center,
		redText.Render(title),
		"",
		message,
		"",
		buttons,
		"",
		lipgloss.NewStyle().Foreground(lipgloss.Color("#7c7c7c")).Render(instructions),
	)
	
	dialogBox := helpBoxStyle.Width(60).Render(content)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, dialogBox)
}

func (m model) viewSetup() string {
	var title, mainPrompt, inputSection, instructions, keybinds string

	switch m.state {
	case stateSetupWeather:
		title = "🌟 Welcome to GoDash!"
		mainPrompt = "Enter your city name for weather data (e.g., Athens, London, New York)."
		inputSection = m.setupTextInput.View()
		instructions = "💡 You can change this later in the config file"
		keybinds = yellowText.Render("Enter") + " to Continue    " + yellowText.Render("Esc") + " to Skip"
	case stateSetupCalendar:
		title = "📅 Calendar Authorization"
		mainPrompt = "Connect your Google Calendar"
		
		if calendarwidget.IsUsingManualFlow() {
			inputSection = m.setupTextInput.View()
			instructions = "📋 After authorization, paste the code here"
			keybinds = yellowText.Render("Ctrl+O") + " Authorize    " + yellowText.Render("Enter") + " Submit Code"
		} else {
			inputSection = ""
			instructions = ""
			keybinds = yellowText.Render("Ctrl+O") + " Authorize"
		}
	}

	// Create sections with proper spacing
	titleSection := helpTitleStyle.Render(title)
	promptSection := lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Render(mainPrompt)
	keybindsStyled := lipgloss.NewStyle().Padding(1, 0).Render(keybinds)

	// Build content dynamically based on what we have
	var contentParts []string
	contentParts = append(contentParts, titleSection, "", promptSection)
	
	if inputSection != "" {
		inputStyled := lipgloss.NewStyle().Padding(1, 0).Render(inputSection)
		contentParts = append(contentParts, "", inputStyled)
	}
	
	if instructions != "" {
		instructionsStyled := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Italic(true).Render(instructions)
		contentParts = append(contentParts, instructionsStyled)
	}
	
	contentParts = append(contentParts, "", keybindsStyled)
	content := lipgloss.JoinVertical(lipgloss.Center, contentParts...)

	// Dynamic box width based on content
	var boxWidth int
	if m.state == stateSetupWeather {
		boxWidth = 80 // Wider for weather setup with longer text
	} else {
		boxWidth = 60 // Narrower for calendar setup
	}

	// Center the content within the box width
	centeredContent := lipgloss.NewStyle().
		Width(boxWidth).
		Align(lipgloss.Center).
		Render(content)

	box := helpBoxStyle.Render(centeredContent)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

func (m model) viewDashboard() string {
	// Add logo at the top
	logo := renderAppLogo()
	logoHeight := lipgloss.Height(logo)
	
	gridHeight := m.height - 3 // Reserve space for logo at top

	leftColumnWidth := m.width / 2
	rightColumnWidth := m.width - leftColumnWidth

	// Left column
	todoBoxHeight := gridHeight * 3 / 7
	calendarBoxHeight := gridHeight - todoBoxHeight

	listTitle := "To-Do List"
	m.todo.SetSize(leftColumnWidth-8, todoBoxHeight-3-lipgloss.Height(listTitle))
	todoBoxContent := lipgloss.JoinVertical(lipgloss.Left, titleStyle.Render(listTitle), m.todo.View())

	todoBoxStyle := boxStyle
	if m.focus == focusList {
		todoBoxStyle = focusedBoxStyle
	}
	todoBox := todoBoxStyle.Width(leftColumnWidth - 2).Height(todoBoxHeight - 4).Render(todoBoxContent)

	calendarTitle := "Calendar"
	m.calendar.SetSize(leftColumnWidth-8, calendarBoxHeight-4-lipgloss.Height(calendarTitle))
	calendarBoxContent := lipgloss.JoinVertical(lipgloss.Left, titleStyle.Render(calendarTitle), m.calendar.View())

	calendarBoxStyle := boxStyle
	if m.focus == focusCalendar {
		calendarBoxStyle = focusedBoxStyle
	}
	calendarBox := calendarBoxStyle.Width(leftColumnWidth - 2).Height(calendarBoxHeight - 4).Render(calendarBoxContent)

	leftColumn := lipgloss.JoinVertical(lipgloss.Top, todoBox, calendarBox)

	// Right column
	notesTitle := "Notes"
	m.notes.SetSize(rightColumnWidth-8, gridHeight-3-lipgloss.Height(notesTitle))
	notesBoxContent := lipgloss.JoinVertical(lipgloss.Left, titleStyle.Render(notesTitle), m.notes.View())

	notesBoxStyle := boxStyle
	if m.focus == focusNotes {
		notesBoxStyle = focusedBoxStyle
	}
	notesBox := notesBoxStyle.Width(rightColumnWidth - 2).Height(gridHeight - 4).Render(notesBoxContent)

	grid := lipgloss.JoinHorizontal(lipgloss.Left, leftColumn, notesBox)

	// --- STATUS BAR ---
	var focusedPanelTitle string
	switch m.focus {
	case focusList:
		focusedPanelTitle = "To-Do List"
	case focusNotes:
		focusedPanelTitle = "Notes"
	case focusCalendar:
		focusedPanelTitle = "Calendar"
	}
	leftStatus := "Press " + yellowText.Render("Ctrl+k") + " to see key bindings from " + redText.Render(focusedPanelTitle)
	rightStatus := "Made with ❤️ by " + blueText.Render("Hellas Dev")

	statusWidth := m.width - lipgloss.Width(leftStatus) - lipgloss.Width(rightStatus)
	statusBar := lipgloss.JoinHorizontal(lipgloss.Top,
		leftStatus,
		lipgloss.NewStyle().Width(statusWidth).Render(""),
		rightStatus,
	)

	// Center the logo and combine everything
	centeredLogo := lipgloss.Place(m.width, logoHeight, lipgloss.Center, lipgloss.Center, logo)
	
	return lipgloss.JoinVertical(lipgloss.Top, centeredLogo, grid, statusBar)
}

func main() {
	if err := config.EnsureDirs(); err != nil {
		fmt.Println("could not create directories:", err)
		os.Exit(1)
	}

	settings, err := config.LoadSettings()
	if err != nil {
		fmt.Println("could not load settings:", err)
		os.Exit(1)
	}

	p := tea.NewProgram(initialModel(settings), tea.WithAltScreen(), tea.WithMouseAllMotion())
	if _, err := p.Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
