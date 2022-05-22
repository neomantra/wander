package allocspec

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"wander/formatter"
	"wander/message"
	"wander/nomad"
)

type allocspecData struct {
	allData, filteredData []string
}

type nomadAllocspecMessage []string

func FetchAllocspec(url, token, allocID string) tea.Cmd {
	return func() tea.Msg {
		fullPath := fmt.Sprintf("%s%s%s", url, "/v1/allocation/", allocID)
		body, err := nomad.Get(fullPath, token, nil)
		if err != nil {
			return message.ErrMsg{Err: err}
		}

		return nomadAllocspecMessage(formatter.PrettyJsonStringAsLines(string(body)))
	}
}
