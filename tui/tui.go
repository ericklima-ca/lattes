package tui

type state int

const (
	commit state = iota
)

type Model struct {
	commitModel *commitModel
	state       state
}

func NewModel() Model {
	return Model{
		commitModel: newCommitModel(),
	}
}
