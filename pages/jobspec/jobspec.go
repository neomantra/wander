package jobspec

import (
	"fmt"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"strings"
	"wander/components/filter"
	"wander/components/viewport"
	"wander/dev"
	"wander/keymap"
	"wander/pages"
	"wander/style"
)

type Model struct {
	url, token    string
	jobspecData   jobspecData
	width, height int
	viewport      viewport.Model
	filter        filter.Model
	jobID         string
	Loading       bool
}

const filterPrefix = "Job Spec"

func New(url, token string, width, height int) Model {
	jobspecFilter := filter.New(filterPrefix)
	jobspecViewport := viewport.New(width, height-jobspecFilter.ViewHeight())
	jobspecViewport.SetCursorEnabled(false)
	model := Model{
		url:      url,
		token:    token,
		width:    width,
		height:   height,
		viewport: jobspecViewport,
		filter:   jobspecFilter,
		Loading:  true,
	}
	return model
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	dev.Debug(fmt.Sprintf("jobspec %T", msg))

	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	if m.viewport.Saving() {
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	}

	switch msg := msg.(type) {
	case nomadJobspecMessage:
		m.jobspecData.allData = msg
		m.updateJobspecViewport()
		m.Loading = false

	case tea.KeyMsg:
		if m.filter.Focused() {
			switch {
			case key.Matches(msg, keymap.KeyMap.Forward):
				m.filter.Blur()

			case key.Matches(msg, keymap.KeyMap.Back):
				m.clearFilter()
			}
		} else {
			switch {
			case key.Matches(msg, keymap.KeyMap.Filter):
				m.filter.Focus()
				return m, nil

			case key.Matches(msg, keymap.KeyMap.Reload):
				return m, pages.ToJobspecPageCmd

			case key.Matches(msg, keymap.KeyMap.Back):
				if len(m.filter.Filter) == 0 {
					return m, pages.ToJobsPageCmd
				} else {
					m.clearFilter()
				}
			}

			m.viewport, cmd = m.viewport.Update(msg)
			cmds = append(cmds, cmd)
		}

		// filter won't respond to key messages if not focused
		prevFilter := m.filter.Filter
		m.filter, cmd = m.filter.Update(msg)
		if m.filter.Filter != prevFilter {
			m.updateJobspecViewport()
		}
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	content := fmt.Sprintf("Loading job spec for %s...", m.jobID)
	if !m.Loading {
		content = m.viewport.View()
	}
	return lipgloss.JoinVertical(lipgloss.Left, m.filter.View(), content)
}

func (m *Model) SetWindowSize(width, height int) {
	m.width, m.height = width, height
	m.viewport.SetSize(width, height-m.filter.ViewHeight())
}

func (m *Model) clearFilter() {
	m.filter.BlurAndClear()
	m.updateJobspecViewport()
}

func (m *Model) SetJobID(jobID string) {
	m.jobID = jobID
	m.filter.SetPrefix(fmt.Sprintf("%s for %s", filterPrefix, style.Bold.Render(jobID)))
}

func (m *Model) updateFilteredAllocationData() {
	var filteredAllocationData []string
	for _, entry := range m.jobspecData.allData {
		if strings.Contains(entry, m.filter.Filter) {
			filteredAllocationData = append(filteredAllocationData, entry)
		}
	}
	m.jobspecData.filteredData = filteredAllocationData
}

func (m *Model) updateJobspecViewport() {
	m.viewport.Highlight = m.filter.Filter
	m.updateFilteredAllocationData()
	m.viewport.SetHeaderAndContent("", strings.Join(m.jobspecData.filteredData, "\n"))
	m.viewport.SetCursorRow(0)
}
