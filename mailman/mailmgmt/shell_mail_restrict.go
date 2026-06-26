package mailmgmt

import (
	"bufio"
	"fmt"
	"strings"
)

type BasicRestrictCommand struct {
	s SetupCommand
}

func (m BasicRestrictCommand) CommandSend(email string) *Command {
	return &Command{
		s: m.s.Arg("send", email),
	}
}

func (m BasicRestrictCommand) CommandReceive(email string) *Command {
	return &Command{
		s: m.s.Arg("receive", email),
	}
}

func (m BasicRestrictCommand) Send(email string) error {
	var _, _, err = m.CommandSend(email).Exec()
	return err
}

func (m BasicRestrictCommand) Receive(email string) error {
	var _, _, err = m.CommandReceive(email).Exec()
	return err
}

type RestrictListCommand struct {
	/*
		root@ubuntu:~# docker exec mailserver setup email restrict list send
		mail@example.com                 REJECT
		help1            REJECT
		help2            REJECT
		help3            REJECT
		help4            REJECT
		root@ubuntu:~# docker exec mailserver setup email restrict list receive
		mail@example.com                 REJECT
		help1            REJECT
		help2            REJECT
		help3            REJECT
		help4            REJECT
		root@ubuntu:~#
	*/
	s SetupCommand
}

func (m RestrictListCommand) CommandSend() *Command {
	return &Command{
		s: m.s.Arg("send"),
	}
}

func (m RestrictListCommand) CommandReceive() *Command {
	return &Command{
		s: m.s.Arg("receive"),
	}
}

type RestrictionResult struct {
	Address string
	Status  string
}

func makeRestrictionResultList(src string) ([]RestrictionResult, error) {
	var scanner = bufio.NewScanner(strings.NewReader(src))
	var resList = make([]RestrictionResult, 0)
	var idx = 0
	for scanner.Scan() {
		idx++

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		flds := make([]string, 0, 2)
		for _, fld := range strings.Fields(line) {
			flds = append(flds, fld)
		}
		if len(flds) < 2 {
			return nil, fmt.Errorf("Line %d does not have enough fields: %q", idx, line)
		}
		if len(flds) > 2 {
			return nil, fmt.Errorf("Line %d has too many fields: %q", idx, line)
		}
		resList = append(resList, RestrictionResult{
			Address: flds[0],
			Status:  flds[1],
		})
	}
	return resList, nil
}

func (m RestrictListCommand) Send() ([]RestrictionResult, error) {
	res, _, err := m.CommandSend().Exec()
	if err != nil {
		return nil, err
	}
	return makeRestrictionResultList(res)
}

func (m RestrictListCommand) Receive() ([]RestrictionResult, error) {
	res, _, err := m.CommandReceive().Exec()
	if err != nil {
		return nil, err
	}
	return makeRestrictionResultList(res)
}

type RestrictMailCommand struct {
	s SetupCommand
}

func (m RestrictMailCommand) Add() BasicRestrictCommand {
	return BasicRestrictCommand{
		s: m.s.Arg("add"),
	}
}

func (m RestrictMailCommand) Remove() BasicRestrictCommand {
	return BasicRestrictCommand{
		s: m.s.Arg("del"),
	}
}

func (m RestrictMailCommand) List() RestrictListCommand {
	return RestrictListCommand{
		s: m.s.Arg("list"),
	}
}
