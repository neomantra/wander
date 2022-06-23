package app

import (
	"fmt"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/robinovitch61/wander/internal/dev"
	"github.com/robinovitch61/wander/internal/tui/components/header"
	"github.com/robinovitch61/wander/internal/tui/components/page"
	"github.com/robinovitch61/wander/internal/tui/constants"
	"github.com/robinovitch61/wander/internal/tui/keymap"
	"github.com/robinovitch61/wander/internal/tui/message"
	"github.com/robinovitch61/wander/internal/tui/nomad"
	"github.com/robinovitch61/wander/internal/tui/style"
)

type Model struct {
	nomadUrl   string
	nomadToken string

	header      header.Model
	currentPage nomad.Page
	pageModels  map[nomad.Page]*page.Model

	jobID        string
	jobNamespace string
	allocID      string
	taskName     string
	logline      string
	logType      nomad.LogType

	width, height int
	initialized   bool
	err           error
}

func InitialModel(url, token string) Model {
	firstPage := nomad.JobsPage
	initialHeader := header.New(constants.LogoString, url, "")

	return Model{
		nomadUrl:    url,
		nomadToken:  token,
		header:      initialHeader,
		currentPage: firstPage,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	dev.Debug(fmt.Sprintf("main %T", msg))
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// always exit if desired, or don't respond if editing filter or saving
		if key.Matches(msg, keymap.KeyMap.Exit) {
			addingQToFilter := m.currentPageFilterFocused()
			saving := m.currentPageViewportSaving()
			typingQWhileFilteringOrSaving := (addingQToFilter || saving) && msg.String() == "q"
			if !typingQWhileFilteringOrSaving {
				return m, tea.Quit
			}
		}

		if !m.currentPageFilterFocused() && !m.currentPageViewportSaving() {
			switch {
			case key.Matches(msg, keymap.KeyMap.Forward):
				if selectedPageRow, err := m.getCurrentPageModel().GetSelectedPageRow(); err == nil {
					switch m.currentPage {
					case nomad.JobsPage:
						m.jobID, m.jobNamespace = nomad.JobIDAndNamespaceFromKey(selectedPageRow.Key)
					case nomad.AllocationsPage:
						m.allocID, m.taskName = nomad.AllocIDAndTaskNameFromKey(selectedPageRow.Key)
					case nomad.LogsPage:
						m.logline = selectedPageRow.Row
					}

					nextPage := m.currentPage.Forward()
					if nextPage != m.currentPage {
						m.getCurrentPageModel().HideToast()
						m.setPage(nextPage)
						return m, m.getCurrentPageCmd()
					}
				}

			case key.Matches(msg, keymap.KeyMap.Back):
				if !m.currentPageFilterApplied() {
					prevPage := m.currentPage.Backward()
					if prevPage != m.currentPage {
						m.getCurrentPageModel().HideToast()
						m.setPage(prevPage)
						return m, m.getCurrentPageCmd()
					}
				}

			case key.Matches(msg, keymap.KeyMap.Reload):
				if m.currentPage.Loads() {
					m.getCurrentPageModel().SetLoading(true)
					return m, m.getCurrentPageCmd()
				}
			}

			if key.Matches(msg, keymap.KeyMap.Spec) {
				if selectedPageRow, err := m.getCurrentPageModel().GetSelectedPageRow(); err == nil {
					switch m.currentPage {
					case nomad.JobsPage:
						m.jobID, m.jobNamespace = nomad.JobIDAndNamespaceFromKey(selectedPageRow.Key)
						m.setPage(nomad.JobSpecPage)
						return m, m.getCurrentPageCmd()
					case nomad.AllocationsPage:
						m.allocID, m.taskName = nomad.AllocIDAndTaskNameFromKey(selectedPageRow.Key)
						m.setPage(nomad.AllocSpecPage)
						return m, m.getCurrentPageCmd()
					}
				}
			}

			if m.currentPage == nomad.LogsPage {
				switch {
				case key.Matches(msg, keymap.KeyMap.StdOut):
					if !m.currentPageLoading() && m.logType != nomad.StdOut {
						m.logType = nomad.StdOut
						m.getCurrentPageModel().SetViewportStyle(style.ViewportHeaderStyle, style.StdOut)
						m.getCurrentPageModel().SetLoading(true)
						return m, m.getCurrentPageCmd()
					}

				case key.Matches(msg, keymap.KeyMap.StdErr):
					if !m.currentPageLoading() && m.logType != nomad.StdErr {
						m.logType = nomad.StdErr
						stdErrHeaderStyle := style.ViewportHeaderStyle.Copy().Inherit(style.StdErr)
						m.getCurrentPageModel().SetViewportStyle(stdErrHeaderStyle, style.StdErr)
						m.getCurrentPageModel().SetLoading(true)
						return m, m.getCurrentPageCmd()
					}
				}
			}
		}

	case message.ErrMsg:
		m.err = msg
		return m, nil

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		if !m.initialized {
			m.initialize()
			cmds = append(cmds, m.getCurrentPageCmd())
		} else {
			m.setPageWindowSize()
		}

	case nomad.PageLoadedMsg:
		m.setPage(msg.Page)
		m.getCurrentPageModel().SetHeader(msg.TableHeader)
		m.getCurrentPageModel().SetAllPageData(msg.AllPageData)
		m.getCurrentPageModel().SetLoading(false)
		m.getCurrentPageModel().SetViewportXOffset(0)
		if m.currentPage == nomad.LogsPage {
			m.pageModels[nomad.LogsPage].SetViewportSelectionToBottom()
		}
	}

	currentPageModel := m.getCurrentPageModel()
	*currentPageModel, cmd = currentPageModel.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v", m.err)
	} else if !m.initialized {
		return ""
	}

	pageView := m.header.View() + "\n" + m.getCurrentPageModel().View()

	return pageView
}

func (m *Model) initialize() {
	pageHeight := m.getPageHeight()

	m.pageModels = make(map[nomad.Page]*page.Model)

	jobsPage := page.New(m.width, pageHeight, m.getFilterPrefix(nomad.JobsPage), nomad.JobsPage.LoadingString(), true, false)
	m.pageModels[nomad.JobsPage] = &jobsPage

	jobSpecPage := page.New(m.width, pageHeight, m.getFilterPrefix(nomad.JobSpecPage), nomad.JobSpecPage.LoadingString(), false, true)
	m.pageModels[nomad.JobSpecPage] = &jobSpecPage

	allocationsPage := page.New(m.width, pageHeight, m.getFilterPrefix(nomad.AllocationsPage), nomad.AllocationsPage.LoadingString(), true, false)
	m.pageModels[nomad.AllocationsPage] = &allocationsPage

	allocSpecPage := page.New(m.width, pageHeight, m.getFilterPrefix(nomad.AllocSpecPage), nomad.AllocSpecPage.LoadingString(), false, true)
	m.pageModels[nomad.AllocSpecPage] = &allocSpecPage

	logsPage := page.New(m.width, pageHeight, m.getFilterPrefix(nomad.LogsPage), nomad.LogsPage.LoadingString(), true, false)
	m.pageModels[nomad.LogsPage] = &logsPage

	loglinePage := page.New(m.width, pageHeight, m.getFilterPrefix(nomad.LoglinePage), nomad.LoglinePage.LoadingString(), false, true)
	m.pageModels[nomad.LoglinePage] = &loglinePage

	m.initialized = true
}

func (m *Model) setPageWindowSize() {
	for _, pm := range m.pageModels {
		pm.SetWindowSize(m.width, m.getPageHeight())
	}
}

func (m *Model) setPage(page nomad.Page) {
	m.currentPage = page
	m.header.KeyHelp = nomad.GetPageKeyHelp(page)
	m.getCurrentPageModel().SetFilterPrefix(m.getFilterPrefix(page))
	if page.Loads() {
		m.getCurrentPageModel().SetLoading(true)
	} else {
		m.getCurrentPageModel().SetLoading(false)
	}
}

func (m *Model) getCurrentPageModel() *page.Model {
	return m.pageModels[m.currentPage]
}

func (m *Model) getCurrentPageCmd() tea.Cmd {
	switch m.currentPage {
	case nomad.JobsPage:
		return nomad.FetchJobs(m.nomadUrl, m.nomadToken)
	case nomad.JobSpecPage:
		return nomad.FetchJobSpec(m.nomadUrl, m.nomadToken, m.jobID, m.jobNamespace)
	case nomad.AllocationsPage:
		return nomad.FetchAllocations(m.nomadUrl, m.nomadToken, m.jobID, m.jobNamespace)
	case nomad.AllocSpecPage:
		return nomad.FetchAllocSpec(m.nomadUrl, m.nomadToken, m.allocID)
	case nomad.LogsPage:
		return nomad.FetchLogs(m.nomadUrl, m.nomadToken, m.allocID, m.taskName, m.logType)
	case nomad.LoglinePage:
		return nomad.FetchLogLine(m.logline)
	default:
		panic("page load command not found")
	}
}

func (m Model) getPageHeight() int {
	return m.height - m.header.ViewHeight()
}

func (m Model) currentPageLoading() bool {
	return m.getCurrentPageModel().Loading()
}

func (m Model) currentPageFilterFocused() bool {
	return m.getCurrentPageModel().FilterFocused()
}

func (m Model) currentPageFilterApplied() bool {
	return m.getCurrentPageModel().FilterApplied()
}

func (m Model) currentPageViewportSaving() bool {
	return m.getCurrentPageModel().ViewportSaving()
}

func (m Model) getFilterPrefix(page nomad.Page) string {
	return page.GetFilterPrefix(m.jobID, m.taskName, m.allocID)
}