package internal

import (
	"fmt"
	"time"

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
}

func NewUI(vault Vault) UI {
	passwordInput := textinput.New()
	passwordInput.Focus()
	passwordInput.EchoMode = textinput.EchoPassword
	passwordInput.EchoCharacter = '*'
	passwordInput.Width = 20

	l := list.New(nil, list.NewDefaultDelegate(), 0, 0)
	l.Title = "GoAegis"
	l.SetShowPagination(false)

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
	var passwordInputCmd tea.Cmd
	//nolint:revive
	m.passwordInput, passwordInputCmd = m.passwordInput.Update(teaMsg)

	var cmd tea.Cmd

	switch msg := teaMsg.(type) {
	case tea.KeyMsg:
		//nolint:exhaustive
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyEnter:
			var err error
			//nolint:revive
			m.db, err = m.vault.DecryptDB([]byte(m.passwordInput.Value()))
			if err != nil {
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
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
	case refreshListMsg:
		cmd = tea.Batch(
			m.list.SetItems(newListItems(m.db, msg.t)),
			m.tick(),
		)
	}

	var listCmd tea.Cmd
	//nolint:revive
	m.list, listCmd = m.list.Update(teaMsg)

	return m, tea.Batch(passwordInputCmd, cmd, listCmd)
}

const tickDuration = time.Second

func (m UI) tick() tea.Cmd {
	return tea.Tick(tickDuration, func(t time.Time) tea.Msg {
		return refreshListMsg{t: t}
	})
}

var docStyle = lipgloss.NewStyle().Margin(1, 2)

func (m UI) View() string {
	if m.view == viewList {
		return docStyle.Render(m.list.View())
	}

	return fmt.Sprintf(
		"Enter password:\n\n%s\n\n%s",
		m.passwordInput.View(),
		"(ctrl+c to quit)",
	) + "\n"
}

type listItem struct {
	e DBEntry
	t time.Time
}

func newListItems(db DB, t time.Time) []list.Item {
	items := make([]list.Item, 0, len(db.Entries))
	for _, e := range db.Entries {
		items = append(items, listItem{
			e: e,
			t: t,
		})
	}
	return items
}

func (i listItem) Title() string {
	if i.e.Issuer == "" {
		return i.e.Name
	}
	return i.e.Issuer + " - " + i.e.Name
}

func (i listItem) Description() string {
	otp, remaining, err := generateOTP(i.e, i.t)
	if err != nil {
		return err.Error()
	}
	return fmt.Sprintf("%s - %d", otp, remaining)
}

func (i listItem) FilterValue() string {
	return i.Title()
}
