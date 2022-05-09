package message

import "wander/formatter"

// NomadJobsMsg is a message for nomad jobs
type NomadJobsMsg struct {
	Table formatter.Table
}

// NomadAllocationMsg is a message for nomad allocations
type NomadAllocationMsg struct {
	Table formatter.Table
}

// ErrMsg is an error message
type ErrMsg struct{ err error }

func (e ErrMsg) Error() string { return e.err.Error() }