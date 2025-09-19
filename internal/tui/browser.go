package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	"github.com/jasper/ai-memory/internal/models"
	"github.com/jasper/ai-memory/internal/storage"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7D56F4"))

	itemStyle = lipgloss.NewStyle().
			PaddingLeft(2)

	selectedItemStyle = lipgloss.NewStyle().
				PaddingLeft(2).
				Foreground(lipgloss.Color("#FAFAFA")).
				Background(lipgloss.Color("#7D56F4"))

	paneStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7D56F4"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262"))
)

type Browser struct {
	store  *storage.SQLiteStore
	dbPath string
}

func NewBrowser(store *storage.SQLiteStore) *Browser {
	return &Browser{store: store, dbPath: ""}
}

func NewBrowserWithPath(store *storage.SQLiteStore, dbPath string) *Browser {
	return &Browser{store: store, dbPath: dbPath}
}

func (b *Browser) Run() error {
	m := initialEnhancedModel(b.store, b.dbPath)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return err
	}
	return nil
}

type listItem struct {
	conversation models.Conversation
}

func (i listItem) FilterValue() string {
	return i.conversation.Title
}

func (i listItem) Title() string {
	return i.conversation.Title
}

func (i listItem) Description() string {
	desc := fmt.Sprintf("%s | %s", i.conversation.Tool, i.conversation.CreatedAt.Format("2006-01-02 15:04"))
	if i.conversation.Project != "" {
		desc = fmt.Sprintf("%s | %s", i.conversation.Project, desc)
	}
	return desc
}

type model struct {
	store            *storage.SQLiteStore
	conversations    []models.Conversation
	list             list.Model
	viewport         viewport.Model
	selectedConv     *models.Conversation
	width            int
	height           int
	ready            bool
	err              error
	searchMode       bool
	searchQuery      string
}

func initialModel(store *storage.SQLiteStore) model {
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

	return model{
		store:         store,
		conversations: conversations,
		list:          l,
		viewport:      vp,
		err:           err,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		m.list.SetSize(listWidth, m.height-2)

		m.viewport.Width = m.width - listWidth - 4
		m.viewport.Height = m.height - 4

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "enter":
			if item, ok := m.list.SelectedItem().(listItem); ok {
				conv, err := m.store.GetConversation(item.conversation.ID)
				if err == nil {
					m.selectedConv = conv
					m.updateViewport()
				}
			}

		case "/":
			m.searchMode = true
			m.list.SetFilteringEnabled(true)

		case "esc":
			m.searchMode = false
			m.list.SetFilteringEnabled(false)
		}
	}

	if !m.searchMode {
		m.list, cmd = m.list.Update(msg)
		cmds = append(cmds, cmd)
	}

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *model) updateViewport() {
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

func (m model) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}

	if m.err != nil {
		return fmt.Sprintf("\n  Error: %v\n", m.err)
	}

	listView := paneStyle.
		Width(m.width/3 - 2).
		Height(m.height - 2).
		Render(m.list.View())

	contentView := paneStyle.
		Width(m.width - m.width/3 - 2).
		Height(m.height - 2).
		Render(m.viewport.View())

	help := helpStyle.Render("  j/k: navigate • enter: select • /: search • q: quit")

	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		listView,
		contentView,
	) + "\n" + help
}