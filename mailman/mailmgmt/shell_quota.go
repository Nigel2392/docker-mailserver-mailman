package mailmgmt

type QuotaCommand struct {
	s SetupCommand
}

func (m QuotaCommand) CommandAdd(target, quota string) *Command {
	return &Command{
		s: m.s.Arg("add", target, quota),
	}
}

func (m QuotaCommand) CommandDelete(alias, target string) *Command {
	return &Command{
		s: m.s.Arg("del", target),
	}
}
