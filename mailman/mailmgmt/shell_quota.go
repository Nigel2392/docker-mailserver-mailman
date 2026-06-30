package mailmgmt

import "al.essio.dev/pkg/shellescape"

type QuotaCommand struct {
	s SetupCommand
}

func (m QuotaCommand) CommandAdd(target, quota string) *Command {
	return &Command{
		s: m.s.Arg("set", shellescape.Quote(target), shellescape.Quote(quota)),
	}
}

func (m QuotaCommand) CommandDelete(target string) *Command {
	return &Command{
		s: m.s.Arg("del", shellescape.Quote(target)),
	}
}
