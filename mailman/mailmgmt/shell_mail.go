package mailmgmt

import (
	"bufio"
	"errors"
	"fmt"
	"net/mail"
	"regexp"
	"strconv"
	"strings"

	"github.com/Nigel2392/go-django/src/core/logger"
)

type MailCommand struct {
	s SetupCommand
}

func (m MailCommand) CommandAdd(email string, passwd string) *Command {
	return &Command{
		s: m.s.Arg("add", email, passwd),
	}
}

func (m MailCommand) CommandUpdate(email string, newpasswd string) *Command {
	return &Command{
		s: m.s.Arg("update", email, newpasswd),
	}
}

func (m MailCommand) CommandDelete(emails ...string) *Command {
	if len(emails) == 0 {
		return &Command{err: errors.New("no email adresses provided to delete command")}
	}
	return &Command{
		s: m.s.Arg(append([]string{"del"}, emails...)...),
	}
}

func (m MailCommand) CommandList() *Command {
	return &Command{
		s: m.s.Arg("list"),
	}
}

func (m MailCommand) Add(email string, passwd string) error {
	_, _, err := m.CommandAdd(email, passwd).Exec()
	return err
}

func (m MailCommand) Update(email string, newpasswd string) error {
	_, _, err := m.CommandUpdate(email, newpasswd).Exec()
	return err

}

func (m MailCommand) Delete(emails ...string) error {
	_, _, err := m.CommandDelete(emails...).Exec()
	return err

}

type ListedAddress struct {
	Raw            string
	Email          string
	CurrentQuota   int
	MaxQuota       int
	PercentageFull int
	Aliases        []string
}

var _matchEmailListRegex = regexp.MustCompile(fmt.Sprintf(`\* %s \( ([\w\.\~]+) \/ ([\w\.\~]+) \) \[(\d+)%%\]`, EMAIL_REGEX))
var _matchEmailListAliasRegex = regexp.MustCompile(`\[\s*aliases\s*->\s+([^\]]*)\]`)

func (m MailCommand) List() ([]ListedAddress, error) {
	src, _, err := m.CommandList().Exec()
	if err != nil {
		return nil, err
	}

	var scanner = bufio.NewScanner(strings.NewReader(src))
	var resList = make([]ListedAddress, 0)
	var idx = 0
	for scanner.Scan() {

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var matches = _matchEmailListRegex.FindStringSubmatch(line)
		if len(matches) < 5 {
			matches = _matchEmailListAliasRegex.FindStringSubmatch(line)
			if len(matches) < 2 {
				continue
				// return nil, fmt.Errorf("no alias block found")
			}

			addrs, err := mail.ParseAddressList(matches[1])
			if err != nil {
				logger.Warnf("failed to parse adress from text: %s", matches[1])
				continue
			}

			// 3. Trim any remaining whitespace and filter out empty fields
			var aliases = make([]string, 0, len(addrs))
			for _, addr := range addrs {
				if addr.Address != "" {
					aliases = append(aliases, addr.Address)
				}
			}

			resList[idx-1].Aliases = aliases
			continue
		}

		var (
			str_curQ  = matches[2]
			str_maxQ  = matches[3]
			str_fullP = matches[4]
		)

		curQ, err := strconv.Atoi(str_curQ)
		if err != nil {
			return nil, fmt.Errorf("invalid current quota value %q", str_curQ)
		}

		var maxQ int
		if str_maxQ == "~" {
			maxQ = -1
		} else {
			maxQ, err = strconv.Atoi(str_maxQ)
		}
		if err != nil {
			return nil, fmt.Errorf("invalid max quota value %q", str_maxQ)
		}

		fullP, err := strconv.Atoi(str_fullP)
		if err != nil {
			return nil, fmt.Errorf("invalid percentage full value %q", str_fullP)
		}

		resList = append(resList, ListedAddress{
			Raw:            matches[0],
			Email:          matches[1],
			CurrentQuota:   curQ,
			MaxQuota:       maxQ,
			PercentageFull: fullP,
		})

		idx++
	}

	return resList, nil
}
