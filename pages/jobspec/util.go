package jobspec

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"wander/formatter"
	"wander/message"
	"wander/nomad"
)

type jobspecData struct {
	allData, filteredData []string
}

type nomadJobspecMessage []string

func FetchJobspec(url, token, jobID string) tea.Cmd {
	return func() tea.Msg {
		fullPath := fmt.Sprintf("%s%s%s", url, "/v1/job/", jobID)
		body, err := nomad.Get(fullPath, token, nil)
		if err != nil {
			return message.ErrMsg{Err: err}
		}

		return nomadJobspecMessage(formatter.PrettyJsonStringAsLines(string(body)))
	}
}
