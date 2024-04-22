package internal

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type view uint8

const (
	viewPassword view = iota + 1
	viewList
)

type UI struct {
	view          view
	vault         Vault
	db            DB
	list          list.Model
	passwordInput textinput.Model
	passwordError error
}

var keyBindingCopy = key.NewBinding(
	key.WithKeys("enter"),
	key.WithHelp("enter", "copy"),
)

func NewUI(appName string, vault Vault) UI {
	passwordInput := textinput.New()
	passwordInput.Focus()
	passwordInput.EchoMode = textinput.EchoPassword
	passwordInput.EchoCharacter = '*'
	passwordInput.Width = 20

	l := list.New(nil, list.NewDefaultDelegate(), 0, 0)
	l.Title = appName
	l.SetShowPagination(false)

	if !clipboard.Unsupported {
		l.AdditionalShortHelpKeys = func() []key.Binding {
			return []key.Binding{
				keyBindingCopy,
			}
		}
		l.AdditionalFullHelpKeys = func() []key.Binding {
			return []key.Binding{
				keyBindingCopy,
			}
		}
	}

	return UI{
		view:          viewPassword,
		vault:         vault,
		list:          l,
		passwordInput: passwordInput,
	}
}

var _ tea.Model = UI{}

func (m UI) Init() tea.Cmd {
	return textinput.Blink
}

type refreshListMsg struct {
	t time.Time
}

func (m UI) Update(teaMsg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := teaMsg.(tea.KeyMsg); ok {
		if slices.Contains([]tea.KeyType{tea.KeyCtrlC, tea.KeyEsc}, msg.Type) {
			return m, tea.Quit
		}
	}

	switch m.view {
	case viewPassword:
		return m.updatePasswordView(teaMsg)
	case viewList:
		return m.updateListView(teaMsg)
	default:
		return m, tea.Quit
	}
}

func (m UI) updatePasswordView(teaMsg tea.Msg) (tea.Model, tea.Cmd) {
	var passwordInputCmd tea.Cmd
	//nolint:revive
	m.passwordInput, passwordInputCmd = m.passwordInput.Update(teaMsg)

	var cmd tea.Cmd

	if msg, ok := teaMsg.(tea.KeyMsg); ok {
		if msg.Type == tea.KeyEnter {
			var err error
			//nolint:revive
			m.db, err = m.vault.DecryptDB([]byte(m.passwordInput.Value()))
			if err != nil {
				//nolint:revive
				m.passwordError = errors.New("invalid password")
				m.passwordInput.Reset()
			} else {
				//nolint:revive
				m.view = viewList
				cmd = tea.Batch(
					m.list.SetItems(newListItems(m.db, time.Now())),
					m.tick(),
				)
			}
		}
	}

	if m.passwordInput.Value() != "" && m.passwordError != nil {
		//nolint:revive
		m.passwordError = nil
	}

	return m, tea.Batch(passwordInputCmd, cmd)
}

func (m UI) updateListView(teaMsg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := teaMsg.(type) {
	case tea.WindowSizeMsg:
		h, v := listStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
	case refreshListMsg:
		cmd = tea.Batch(
			m.list.SetItems(newListItems(m.db, msg.t)),
			m.tick(),
		)
	case tea.KeyMsg:
		if msg.Type != tea.KeyEnter || clipboard.Unsupported {
			break
		}

		item, ok := m.list.SelectedItem().(listItem)
		if !ok {
			break
		}

		otp, _, err := item.entry.GenerateOTP(item.t)
		if err == nil {
			_ = clipboard.WriteAll(otp)
		}
	}

	var listCmd tea.Cmd
	//nolint:revive
	m.list, listCmd = m.list.Update(teaMsg)

	return m, tea.Batch(cmd, listCmd)
}

const tickDuration = time.Second

func (m UI) tick() tea.Cmd {
	return tea.Tick(tickDuration, func(t time.Time) tea.Msg {
		return refreshListMsg{t: t}
	})
}

var (
	listStyle = lipgloss.NewStyle().Margin(1, 2)
	helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Margin(1, 0)
	errStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000"))
)

func (m UI) View() string {
	if m.view == viewList {
		return listStyle.Render(m.list.View())
	}

	var b strings.Builder

	b.WriteString("Enter password:\n\n")
	b.WriteString(m.passwordInput.View())
	b.WriteString("\n")
	if m.passwordError != nil {
		b.WriteString(errStyle.Render(m.passwordError.Error()))
	}
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("(ctrl+c to quit)"))

	return b.String()
}

type listItem struct {
	entry DBEntry
	t     time.Time
}

func newListItems(db DB, t time.Time) []list.Item {
	items := make([]list.Item, 0, len(db.Entries))
	for _, e := range db.Entries {
		items = append(items, listItem{
			entry: e,
			t:     t,
		})
	}
	return items
}

func (i listItem) Title() string {
	if i.entry.Issuer == "" {
		return i.entry.Name
	}
	return i.entry.Issuer + " - " + i.entry.Name
}

func (i listItem) Description() string {
	otp, remaining, err := i.entry.GenerateOTP(i.t)
	if err != nil {
		return err.Error()
	}
	return fmt.Sprintf("%s - %d", otp, remaining)
}

func (i listItem) FilterValue() string {
	return i.Title()
}
