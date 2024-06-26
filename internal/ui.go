package internal

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type state uint8

const (
	stateEnteringPassword state = iota + 1
	stateBrowsingList
)

var (
	keyBindingCopy = key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "copy to clipboard"),
	)
	keyBindingUnlock = key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "unlock vault"),
	)
	keyBindingQuit = key.NewBinding(
		key.WithKeys("esc", "ctrl+c"),
		key.WithHelp("esc/ctrl+c", "quit"),
	)
	keyClearFilter = key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "clear filter"),
	)
)

type UI struct {
	state            state
	vault            Vault
	db               DB
	list             list.Model
	passwordViewHelp help.Model
	passwordInput    textinput.Model
	passwordError    error
}

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

	l.KeyMap.Quit = keyBindingQuit
	l.KeyMap.ClearFilter = keyClearFilter

	return UI{
		state:            stateEnteringPassword,
		vault:            vault,
		list:             l,
		passwordInput:    passwordInput,
		passwordViewHelp: help.New(),
	}
}

var _ tea.Model = UI{}

func (m UI) Init() tea.Cmd {
	return textinput.Blink
}

type refreshListMsg struct {
	t time.Time
}

//nolint:gocyclo
func (m UI) Update(teaMsg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	if msg, ok := teaMsg.(tea.WindowSizeMsg); ok {
		h, v := listStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
	}

	return m, tea.Batch(cmd, m.handleEnteringPassword(teaMsg), m.handleBrowsingList(teaMsg))
}

func (m *UI) handleEnteringPassword(teaMsg tea.Msg) tea.Cmd {
	if m.state != stateEnteringPassword {
		return nil
	}

	var passwordInputCmd tea.Cmd
	m.passwordInput, passwordInputCmd = m.passwordInput.Update(teaMsg)

	if m.passwordInput.Value() != "" && m.passwordError != nil {
		m.passwordError = nil
	}

	var cmd tea.Cmd

	if msg, ok := teaMsg.(tea.KeyMsg); ok {
		switch {
		case key.Matches(msg, keyBindingQuit):
			return tea.Quit
		case key.Matches(msg, keyBindingUnlock):
			var err error
			//nolint:revive
			m.db, err = m.vault.DecryptDB([]byte(m.passwordInput.Value()))
			if err != nil {
				//nolint:revive
				m.passwordError = errors.New("invalid password")
				m.passwordInput.Reset()
			} else {
				//nolint:revive
				m.state = stateBrowsingList
				cmd = tea.Batch(
					m.list.SetItems(newListItems(m.db, time.Now())),
					m.tick(),
				)
			}
		}
	}

	return tea.Batch(passwordInputCmd, cmd)
}

func (m *UI) handleBrowsingList(teaMsg tea.Msg) tea.Cmd {
	if m.state != stateBrowsingList {
		return nil
	}

	var cmd tea.Cmd

	switch msg := teaMsg.(type) {
	case refreshListMsg:
		for _, i := range m.list.Items() {
			converted, ok := i.(*listItem)
			if !ok {
				continue
			}

			converted.t = msg.t
		}

		cmd = m.tick()
	case tea.KeyMsg:
		if key.Matches(msg, keyBindingCopy) && !clipboard.Unsupported {
			item, ok := m.list.SelectedItem().(*listItem)
			if !ok {
				break
			}

			otp, _, err := item.entry.GenerateOTP(item.t)
			if err == nil {
				_ = clipboard.WriteAll(otp)
			}
		}
	}

	var listCmd tea.Cmd
	//nolint:revive
	m.list, listCmd = m.list.Update(teaMsg)

	return tea.Batch(listCmd, cmd)
}

const tickDuration = time.Second

func (m UI) tick() tea.Cmd {
	return tea.Tick(tickDuration, func(t time.Time) tea.Msg {
		return refreshListMsg{t: t}
	})
}

var (
	listStyle = lipgloss.NewStyle().Margin(1, 2)
	errStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000"))
)

func (m UI) View() string {
	if m.state == stateBrowsingList {
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
	b.WriteString(m.passwordViewHelp.ShortHelpView([]key.Binding{
		keyBindingUnlock,
		keyBindingQuit,
	}))

	return b.String()
}

type listItem struct {
	entry DBEntry
	t     time.Time
}

func newListItems(db DB, t time.Time) []list.Item {
	items := make([]list.Item, 0, len(db.Entries))
	for _, e := range db.Entries {
		items = append(items, &listItem{
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
