package allocspec

import (
	"fmt"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"strings"
	"wander/components/filter"
	"wander/components/viewport"
	"wander/dev"
	"wander/formatter"
	"wander/keymap"
	"wander/pages"
	"wander/style"
)

type Model struct {
	url, token    string
	allocspecData allocspecData
	width, height int
	viewport      viewport.Model
	filter        filter.Model
	allocID       string
	taskName      string
	Loading       bool
}

const filterPrefix = "Allocation Spec"

func New(url, token string, width, height int) Model {
	allocspecFilter := filter.New(filterPrefix)
	allocspecViewport := viewport.New(width, height-allocspecFilter.ViewHeight())
	allocspecViewport.SetCursorEnabled(false)
	model := Model{
		url:      url,
		token:    token,
		width:    width,
		height:   height,
		viewport: allocspecViewport,
		filter:   allocspecFilter,
		Loading:  true,
	}
	return model
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	dev.Debug(fmt.Sprintf("allocspec %T", msg))

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
	case nomadAllocspecMessage:
		m.allocspecData.allData = msg
		m.updateAllocspecViewport()
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
				return m, pages.ToAllocspecPageCmd

			case key.Matches(msg, keymap.KeyMap.Back):
				if len(m.filter.Filter) == 0 {
					return m, pages.ToAllocationsPageCmd
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
			m.updateAllocspecViewport()
		}
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	content := fmt.Sprintf("Loading allocation spec for %s...", m.taskName)
	if !m.Loading {
		content = m.viewport.View()
	}
	return lipgloss.JoinVertical(lipgloss.Left, m.filter.View(), content)
}

func (m *Model) SetWindowSize(width, height int) {
	m.width, m.height = width, height
	m.viewport.SetSize(width, height-m.filter.ViewHeight())
}

func (m *Model) ResetXOffset() {
	m.viewport.SetXOffset(0)
}

func (m *Model) SetAllocationData(allocID, taskName string) {
	m.allocID, m.taskName = allocID, taskName
	m.filter.SetPrefix(fmt.Sprintf("%s for %s %s", filterPrefix, style.Bold.Render(taskName), formatter.ShortAllocID(allocID)))
}

func (m *Model) clearFilter() {
	m.filter.BlurAndClear()
	m.updateAllocspecViewport()
}

func (m *Model) updateFilteredAllocationData() {
	var filteredAllocationData []string
	for _, entry := range m.allocspecData.allData {
		if strings.Contains(entry, m.filter.Filter) {
			filteredAllocationData = append(filteredAllocationData, entry)
		}
	}
	m.allocspecData.filteredData = filteredAllocationData
}

func (m *Model) updateAllocspecViewport() {
	m.viewport.Highlight = m.filter.Filter
	m.updateFilteredAllocationData()
	m.viewport.SetHeaderAndContent("", strings.Join(m.allocspecData.filteredData, "\n"))
	m.viewport.SetCursorRow(0)
}
