// Package calendar implements the calendar widget for the GoDash dashboard.
package calendar

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/ethanefung/bubble-datepicker"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"google.golang.org/api/calendar/v3"

	"GoDash/widgets/clock"
	"GoDash/widgets/weather"
)

// Helper functions
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// --- MESSAGES ---
type weatherMsg struct{ w *weather.WeatherResponse }
type weatherErrMsg struct{ err error }

func fetchWeather(city string) tea.Cmd {
	return func() tea.Msg {
		w, err := weather.GetWeather(city)
		if err != nil {
			return weatherErrMsg{err}
		}
		return weatherMsg{w}
	}
}

type calendarState int

const (
	StateIdle calendarState = iota
	StateReady
)

const fetchCoolDown = 5 * time.Second

type Model struct {
	state          calendarState
	DatePicker     datepicker.Model
	events         []*calendar.Event
	selectedDate   time.Time
	cachedEvents   map[string][]*calendar.Event
	fetchingMonths map[string]bool
	lastFetchTime  time.Time
	err            error
	loading        bool
	spinner        spinner.Model
	keys           KeyMap
	clock          clock.Model
	weather        *weather.WeatherResponse
	weatherErr     error
	weatherLoading bool
	location       string
	width, height  int
}

type KeyMap struct {
	Confirm key.Binding
}

func New(keys KeyMap, location string) Model {
	dp := datepicker.New(time.Now())
	dpStyles := datepicker.DefaultStyles()
	dpStyles.SelectedText = lipgloss.NewStyle().Foreground(lipgloss.Color("#61afef"))
	dpStyles.FocusedText = lipgloss.NewStyle().Foreground(lipgloss.Color("#F25D94"))
	dpStyles.HeaderText = lipgloss.NewStyle().Foreground(lipgloss.Color("#e5c07b"))
	dp.Styles = dpStyles
	dp.SelectDate()

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	cachedEvents, err := LoadCalendarCache()
	if err != nil {
		// Log the error but continue with an empty cache
		fmt.Printf("Error loading calendar cache: %v. Starting fresh.\n", err)
		cachedEvents = make(map[string][]*calendar.Event)
	}

	return Model{
		state:          StateIdle,
		DatePicker:     dp,
		selectedDate:   time.Now(),
		cachedEvents:   cachedEvents,
		fetchingMonths: make(map[string]bool),
		spinner:        s,
		keys:           keys,
		clock:          clock.New(),
		weatherLoading: true,
		location:       location,
	}
}


func (m Model) Init() tea.Cmd {
	return tea.Batch(
		FetchEventsForMonth(time.Now()),
		m.clock.Init(),
		fetchWeather(m.location),
	)
}

func (m *Model) Update(msg tea.Msg, focused bool) (Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case EventsMsg:
		m.cachedEvents[msg.MonthKey] = msg.Events
		m.fetchingMonths[msg.MonthKey] = false
		m.loading = false
		m.state = StateReady
		m.filterEventsForSelectedDate()
		// Save the updated cache to disk in a non-blocking way
		go SaveCalendarCache(m.cachedEvents)
		return *m, nil
	case EventsErrMsg:
		m.fetchingMonths[msg.MonthKey] = false
		if msg.Err == ErrAuthRequired {
			m.state = StateIdle
			m.loading = false
			// Let the main model handle auth.
			return *m, nil
		}
		m.err = msg.Err
		m.loading = false
		return *m, nil
	case weatherMsg:
		m.weather = msg.w
		m.weatherLoading = false
		return *m, nil
	case weatherErrMsg:
		m.weatherErr = msg.err
		m.weatherLoading = false
		return *m, nil
	}

	m.clock, cmd = m.clock.Update(msg)
	cmds = append(cmds, cmd)

	if m.loading || m.weatherLoading {
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	if focused {
		switch m.state {
		case StateReady:
			var datepickerCmd tea.Cmd
			m.DatePicker.SetFocus(datepicker.FocusCalendar)
			m.DatePicker.SelectDate()
			m.DatePicker, datepickerCmd = m.DatePicker.Update(msg)
			if m.DatePicker.Time.Day() != m.selectedDate.Day() ||
				m.DatePicker.Time.Month() != m.selectedDate.Month() ||
				m.DatePicker.Time.Year() != m.selectedDate.Year() {
				m.selectedDate = m.DatePicker.Time
				monthKey := m.selectedDate.Format("2006-01")

				if _, ok := m.cachedEvents[monthKey]; ok {
					// Month is in cache, just filter
					m.filterEventsForSelectedDate()
					m.loading = false
				} else if !m.fetchingMonths[monthKey] && time.Since(m.lastFetchTime) > fetchCoolDown {
					// Month is not in cache, not being fetched, and cooldown has passed
					m.loading = true
					m.fetchingMonths[monthKey] = true
					m.lastFetchTime = time.Now()
					datepickerCmd = tea.Batch(datepickerCmd, FetchEventsForMonth(m.selectedDate))
				}
				// If it's already being fetched, do nothing, the spinner is already on.
			}
			cmds = append(cmds, datepickerCmd)
		}
		cmds = append(cmds, cmd)
	}

	return *m, tea.Batch(cmds...)
}

func (m *Model) View() string {
	if m.err != nil {
		return "Error: " + m.err.Error()
	}

	// Left side: Calendar and events
	var leftSide string
	if m.loading {
		leftSide = m.spinner.View()
	} else {
		var eventsTodayBuilder strings.Builder
		if len(m.events) > 0 {
			datePickerHeight := 8
			availableLines := m.height - datePickerHeight

			if availableLines > 0 {
				numToShow := min(len(m.events), availableLines)

				maxSummaryLength := max(0, m.width-14)

				for i := range numToShow {
					summary := m.events[i].Summary
					summary = strings.ReplaceAll(summary, "\n", " ")
					if len(summary) > maxSummaryLength {
						summary = summary[:maxSummaryLength]
					}
					redText := lipgloss.NewStyle().Foreground(lipgloss.Color("#be8a59"))
					eventsTodayBuilder.WriteString(redText.Render( summary) + "\n")
				}
			}
		}
		eventsToday := strings.TrimSuffix(eventsTodayBuilder.String(), "\n")
		leftSide = lipgloss.JoinVertical(lipgloss.Left, m.DatePicker.View(), eventsToday)
	}

	// Right side: Clock and Weather
	var weatherContent string
	if m.weatherLoading {
		weatherContent = lipgloss.NewStyle().
			Padding(2).
			Align(lipgloss.Center).
			Render(m.spinner.View())
	} else if m.weatherErr != nil {
		weatherContent = lipgloss.NewStyle().
			Padding(1).
			Align(lipgloss.Center).
			Foreground(lipgloss.Color("9")).
			Render("Weather\nUnavailable")
	} else if m.weather != nil {
		art := weather.GetWeatherArt(m.weather.Icon)
		
		// Format temperature and location for better display
		tempInfo := fmt.Sprintf("%.1f°C", m.weather.Temp)
		locationInfo := m.weather.Name
		descInfo := m.weather.Description
		
		// Create balanced weather layout
		weatherInfo := lipgloss.NewStyle().
			Align(lipgloss.Center).
			Render(lipgloss.JoinVertical(lipgloss.Center,
				locationInfo,
				tempInfo,
				descInfo,
			))
		
		weatherContent = lipgloss.JoinVertical(lipgloss.Center, art, weatherInfo)
	} else {
		weatherContent = lipgloss.NewStyle().
			Padding(2).
			Align(lipgloss.Center).
			Foreground(lipgloss.Color("240")).
			Render("No weather\ndata available")
	}

	// If the screen is too small, the datepicker will break the layout.
	// Let's hide the weather/clock column if the screen is too narrow.
	if m.width < 45 { // 45 is a bit arbitrary, datepicker is ~30, give it some room
		return lipgloss.NewStyle().Width(m.width).Height(m.height).Render(leftSide)
	}

	// Calculate responsive widths
	minLeftWidth := 30  // Minimum width for calendar
	minRightWidth := 20 // Minimum width for clock/weather
	
	// For smaller screens, use fixed proportions
	var leftWidth, rightWidth int
	if m.width < 80 {
		leftWidth = minLeftWidth
		rightWidth = m.width - leftWidth - 2
	} else {
		// For larger screens, use more balanced proportions
		leftWidth = min(m.width*2/3, 40) // Max 50 chars for left side
		rightWidth = m.width - leftWidth - 2
	}
	
	// Ensure minimum widths
	if leftWidth < minLeftWidth {
		leftWidth = minLeftWidth
		rightWidth = m.width - leftWidth - 2
	}
	if rightWidth < minRightWidth {
		rightWidth = minRightWidth
		leftWidth = m.width - rightWidth - 2
	}

	leftSideWithBorder := lipgloss.NewStyle().
		Width(leftWidth).
		Border(lipgloss.NormalBorder(), false, true, false, false).
		PaddingRight(1).
		MarginRight(1).
		Render(leftSide)

	// Create a separator line that scales with the right panel width
	separatorWidth := max(1, rightWidth-4)
	separator := strings.Repeat("─", separatorWidth)

	rightSideContent := lipgloss.JoinVertical(lipgloss.Center,
		lipgloss.NewStyle().MarginTop(1).Render(m.clock.View()),
		lipgloss.NewStyle().
			MarginTop(1).
			MarginBottom(1).
			Foreground(lipgloss.Color("240")).
			Render(separator),
		weatherContent,
	)

	rightSide := lipgloss.NewStyle().
		Width(rightWidth).
		Height(m.height).
		Align(lipgloss.Center).
		AlignVertical(lipgloss.Top).
		Render(rightSideContent)

	return lipgloss.JoinHorizontal(lipgloss.Top, leftSideWithBorder, rightSide)
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m *Model) State() calendarState {
	return m.state
}

func (m *Model) filterEventsForSelectedDate() {
	monthKey := m.selectedDate.Format("2006-01")
	if monthlyEvents, ok := m.cachedEvents[monthKey]; ok {
		var dailyEvents []*calendar.Event
		for _, event := range monthlyEvents {
			var eventDate time.Time
			var err error
			if event.Start.DateTime != "" {
				eventDate, err = time.Parse(time.RFC3339, event.Start.DateTime)
			} else {
				eventDate, err = time.Parse("2006-01-02", event.Start.Date)
			}

			if err != nil {
				continue // Or handle error
			}

			if eventDate.Day() == m.selectedDate.Day() &&
				eventDate.Month() == m.selectedDate.Month() &&
				eventDate.Year() == m.selectedDate.Year() {
				dailyEvents = append(dailyEvents, event)
			}
		}
		m.events = dailyEvents
	} else {
		m.events = nil // No events for this month in cache
	}
}

// --- Messages ---

// EventsMsg represents a message containing calendar events for a specific month.
type EventsMsg struct {
	MonthKey string
	Events   []*calendar.Event
}
type EventsErrMsg struct {
	MonthKey string
	Err      error
}


// --- Commands ---

// FetchEventsForMonth creates a command to fetch calendar events for the specified month.
func FetchEventsForMonth(month time.Time) tea.Cmd {
	monthKey := month.Format("2006-01")
	return func() tea.Msg {
		srv, err := GetCalendarService()
		if err != nil {
			return EventsErrMsg{MonthKey: monthKey, Err: err}
		}
		events, err := GetCalendarEventsForMonth(srv, month)
		if err != nil {
			return EventsErrMsg{MonthKey: monthKey, Err: err}
		}
		return EventsMsg{
			MonthKey: monthKey,
			Events:   events,
		}
	}
}
