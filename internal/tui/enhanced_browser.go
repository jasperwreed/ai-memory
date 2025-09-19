package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	"github.com/jasperwreed/ai-memory/internal/models"
	"github.com/jasperwreed/ai-memory/internal/search"
	"github.com/jasperwreed/ai-memory/internal/storage"
)

type commandMode int

const (
	modeNormal commandMode = iota
	modeCommand
	modeSearch
)

type enhancedModel struct {
	store            *storage.SQLiteStore
	conversations    []models.Conversation
	list             list.Model
	viewport         viewport.Model
	commandInput     textinput.Model
	selectedConv     *models.Conversation
	width            int
	height           int
	ready            bool
	err              error
	mode             commandMode
	statusMessage    string
	searchResults    []models.Conversation
	dbPath           string
}

func NewEnhancedBrowser(store *storage.SQLiteStore, dbPath string) *Browser {
	return &Browser{store: store}
}

func (b *Browser) RunEnhanced() error {
	m := initialEnhancedModel(b.store, "")
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return err
	}
	return nil
}

func initialEnhancedModel(store *storage.SQLiteStore, dbPath string) enhancedModel {
	items := []list.Item{}

	conversations, err := store.ListConversations(100, 0, nil)
	if err == nil {
		for _, conv := range conversations {
			items = append(items, listItem{conversation: conv})
		}
	}

	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = true

	l := list.New(items, delegate, 0, 0)
	l.Title = "Conversations"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.Styles.Title = titleStyle

	vp := viewport.New(0, 0)
	vp.HighPerformanceRendering = false

	cmdInput := textinput.New()
	cmdInput.Prompt = ":"
	cmdInput.CharLimit = 256
	cmdInput.Width = 50

	return enhancedModel{
		store:         store,
		conversations: conversations,
		list:          l,
		viewport:      vp,
		commandInput:  cmdInput,
		err:           err,
		mode:          modeNormal,
		dbPath:        dbPath,
	}
}

func (m enhancedModel) Init() tea.Cmd {
	return nil
}

func (m enhancedModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		if !m.ready {
			m.ready = true
		}

		listWidth := m.width / 3
		m.list.SetSize(listWidth, m.height-3)

		m.viewport.Width = m.width - listWidth - 4
		m.viewport.Height = m.height - 5

		m.commandInput.Width = m.width - 4

	case tea.KeyMsg:
		switch m.mode {
		case modeNormal:
			switch msg.String() {
			case "q":
				return m, tea.Quit

			case ":":
				m.mode = modeCommand
				m.commandInput.Focus()
				m.commandInput.SetValue("")
				return m, textinput.Blink

			case "/":
				m.mode = modeSearch
				m.list.SetFilteringEnabled(true)
				m.statusMessage = "Search mode - Type to filter, ESC to exit"

			case "enter":
				if item, ok := m.list.SelectedItem().(listItem); ok {
					conv, err := m.store.GetConversation(item.conversation.ID)
					if err == nil {
						m.selectedConv = conv
						m.updateViewport()
					}
				}

			case "?":
				m.showHelp()
			}

		case modeCommand:
			switch msg.String() {
			case "enter":
				m = m.executeCommand(m.commandInput.Value())
				m.mode = modeNormal
				m.commandInput.Blur()
				m.commandInput.SetValue("")

			case "esc":
				m.mode = modeNormal
				m.commandInput.Blur()
				m.commandInput.SetValue("")
				m.statusMessage = ""
			}

		case modeSearch:
			switch msg.String() {
			case "esc":
				m.mode = modeNormal
				m.list.SetFilteringEnabled(false)
				m.statusMessage = ""
			}
		}
	}

	switch m.mode {
	case modeNormal, modeSearch:
		m.list, cmd = m.list.Update(msg)
		cmds = append(cmds, cmd)

	case modeCommand:
		m.commandInput, cmd = m.commandInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *enhancedModel) executeCommand(cmdStr string) enhancedModel {
	parts := strings.Fields(cmdStr)
	if len(parts) == 0 {
		return *m
	}

	command := parts[0]
	args := parts[1:]

	switch command {
	case "scan":
		m.statusMessage = "Scanning for conversations..."
		go m.runScan(args)

	case "search":
		if len(args) > 0 {
			query := strings.Join(args, " ")
			m.statusMessage = fmt.Sprintf("Searching for: %s", query)
			go m.runSearch(query)
		} else {
			m.statusMessage = "Usage: :search <query>"
		}

	case "capture":
		m.statusMessage = "Capturing current conversation..."
		go m.runCapture(args)

	case "stats":
		m.showStats()

	case "export":
		if len(args) > 0 {
			m.statusMessage = fmt.Sprintf("Exporting to: %s", args[0])
			go m.runExport(args[0])
		} else {
			m.statusMessage = "Usage: :export <filename>"
		}

	case "import":
		if len(args) > 0 {
			m.statusMessage = fmt.Sprintf("Importing from: %s", args[0])
			go m.runImport(args[0])
		} else {
			m.statusMessage = "Usage: :import <filename>"
		}

	case "delete":
		if m.selectedConv != nil {
			m.statusMessage = "Deleting conversation..."
			go m.runDelete(m.selectedConv.ID)
		} else {
			m.statusMessage = "No conversation selected"
		}

	case "help", "h":
		m.showHelp()

	case "quit", "q":
		m.statusMessage = "Use 'q' key in normal mode to quit"

	case "w", "write":
		m.statusMessage = "Database auto-saves"

	default:
		m.statusMessage = fmt.Sprintf("Unknown command: %s", command)
	}

	return *m
}

func (m *enhancedModel) runScan(args []string) {
	// Scanner implementation would need to be adapted
	// For now, we'll just show a message
	m.statusMessage = "Scan functionality needs implementation"
}

func (m *enhancedModel) runSearch(query string) {
	searcher := search.NewSearcher(m.store)
	results, err := searcher.Search(query, 100)
	if err != nil {
		m.statusMessage = fmt.Sprintf("Search failed: %v", err)
		return
	}

	conversations := []models.Conversation{}
	for _, result := range results {
		conversations = append(conversations, result.Conversation)
	}
	m.searchResults = conversations
	items := []list.Item{}
	for _, conv := range conversations {
		items = append(items, listItem{conversation: conv})
	}
	m.list.SetItems(items)
	m.statusMessage = fmt.Sprintf("Found %d results", len(results))
}

func (m *enhancedModel) runCapture(args []string) {
	// Capture functionality would need adapting based on the actual implementation
	m.statusMessage = "Capture functionality needs implementation"
}

func (m *enhancedModel) runExport(filename string) {
	file, err := os.Create(filename)
	if err != nil {
		m.statusMessage = fmt.Sprintf("Export failed: %v", err)
		return
	}
	defer file.Close()

	conversations, err := m.store.ListConversations(0, 0, nil)
	if err != nil {
		m.statusMessage = fmt.Sprintf("Export failed: %v", err)
		return
	}

	for _, conv := range conversations {
		fmt.Fprintf(file, "# %s\n\n", conv.Title)
		fmt.Fprintf(file, "Tool: %s\n", conv.Tool)
		fmt.Fprintf(file, "Date: %s\n\n", conv.CreatedAt.Format("2006-01-02 15:04:05"))

		for _, msg := range conv.Messages {
			fmt.Fprintf(file, "## %s\n\n%s\n\n", msg.Role, msg.Content)
		}
		fmt.Fprintf(file, "\n---\n\n")
	}

	m.statusMessage = fmt.Sprintf("Exported %d conversations to %s", len(conversations), filename)
}

func (m *enhancedModel) runImport(filename string) {
	m.statusMessage = "Import functionality not yet implemented"
}

func (m *enhancedModel) runDelete(id int64) {
	if err := m.store.DeleteConversation(int64(id)); err != nil {
		m.statusMessage = fmt.Sprintf("Delete failed: %v", err)
	} else {
		m.statusMessage = "Conversation deleted"
		m.selectedConv = nil
		m.refreshList()
	}
}

func (m *enhancedModel) showStats() {
	conversations, _ := m.store.ListConversations(0, 0, nil)

	toolCounts := make(map[string]int)
	projectCounts := make(map[string]int)

	for _, conv := range conversations {
		toolCounts[conv.Tool]++
		if conv.Project != "" {
			projectCounts[conv.Project]++
		}
	}

	var content strings.Builder
	content.WriteString(titleStyle.Render("Statistics"))
	content.WriteString("\n\n")
	content.WriteString(fmt.Sprintf("Total Conversations: %d\n\n", len(conversations)))

	content.WriteString("By Tool:\n")
	for tool, count := range toolCounts {
		content.WriteString(fmt.Sprintf("  %s: %d\n", tool, count))
	}

	content.WriteString("\nBy Project:\n")
	for project, count := range projectCounts {
		content.WriteString(fmt.Sprintf("  %s: %d\n", project, count))
	}

	m.viewport.SetContent(content.String())
	m.viewport.GotoTop()
}

func (m *enhancedModel) showHelp() {
	help := `
Vim-style Commands (press : to enter command mode):

  :scan [dir]     - Scan directory for AI conversations
  :search <query> - Search conversations
  :capture [tool] - Capture current conversation
  :stats          - Show statistics
  :export <file>  - Export conversations
  :import <file>  - Import conversations
  :delete         - Delete selected conversation
  :help           - Show this help

Normal Mode Keys:
  j/k or ↑/↓     - Navigate list
  enter          - View conversation
  /              - Filter list
  :              - Command mode
  ?              - Show help
  q              - Quit

Search Mode:
  Type to filter conversations
  ESC to exit search mode
`

	m.viewport.SetContent(help)
	m.viewport.GotoTop()
}

func (m *enhancedModel) refreshList() {
	conversations, _ := m.store.ListConversations(100, 0, nil)
	items := []list.Item{}
	for _, conv := range conversations {
		items = append(items, listItem{conversation: conv})
	}
	m.list.SetItems(items)
	m.conversations = conversations
}

func (m *enhancedModel) updateViewport() {
	if m.selectedConv == nil {
		m.viewport.SetContent("Select a conversation to view")
		return
	}

	var content strings.Builder
	content.WriteString(titleStyle.Render(m.selectedConv.Title))
	content.WriteString("\n\n")
	content.WriteString(fmt.Sprintf("Tool: %s\n", m.selectedConv.Tool))
	if m.selectedConv.Project != "" {
		content.WriteString(fmt.Sprintf("Project: %s\n", m.selectedConv.Project))
	}
	if len(m.selectedConv.Tags) > 0 {
		content.WriteString(fmt.Sprintf("Tags: %s\n", strings.Join(m.selectedConv.Tags, ", ")))
	}
	content.WriteString(fmt.Sprintf("Created: %s\n", m.selectedConv.CreatedAt.Format("2006-01-02 15:04:05")))

	// FilePath field can be added to the model if needed

	content.WriteString("\n" + strings.Repeat("─", 40) + "\n\n")

	for _, msg := range m.selectedConv.Messages {
		roleStyle := lipgloss.NewStyle().Bold(true)
		if msg.Role == "user" {
			roleStyle = roleStyle.Foreground(lipgloss.Color("#00FF00"))
			content.WriteString(roleStyle.Render("User:"))
		} else {
			roleStyle = roleStyle.Foreground(lipgloss.Color("#00BFFF"))
			content.WriteString(roleStyle.Render("Assistant:"))
		}
		content.WriteString("\n")
		content.WriteString(msg.Content)
		content.WriteString("\n\n")
	}

	m.viewport.SetContent(content.String())
	m.viewport.GotoTop()
}

func (m enhancedModel) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}

	if m.err != nil {
		return fmt.Sprintf("\n  Error: %v\n", m.err)
	}

	listView := paneStyle.
		Width(m.width/3 - 2).
		Height(m.height - 3).
		Render(m.list.View())

	contentView := paneStyle.
		Width(m.width - m.width/3 - 2).
		Height(m.height - 3).
		Render(m.viewport.View())

	var bottomBar string
	switch m.mode {
	case modeCommand:
		bottomBar = m.commandInput.View()
	case modeNormal:
		if m.statusMessage != "" {
			bottomBar = helpStyle.Render(m.statusMessage)
		} else {
			bottomBar = helpStyle.Render("  j/k: navigate • enter: select • /: filter • :: command • ?: help • q: quit")
		}
	case modeSearch:
		bottomBar = helpStyle.Render("  Search mode - Type to filter • ESC: exit search")
	}

	dbInfo := fmt.Sprintf("DB: %s", filepath.Base(m.dbPath))
	if m.dbPath == "" {
		dbInfo = "DB: default"
	}

	topBar := lipgloss.JoinHorizontal(
		lipgloss.Left,
		titleStyle.Render("AI Memory"),
		lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Render("  "+dbInfo),
	)

	return topBar + "\n" +
		lipgloss.JoinHorizontal(
			lipgloss.Top,
			listView,
			contentView,
		) + "\n" + bottomBar
}