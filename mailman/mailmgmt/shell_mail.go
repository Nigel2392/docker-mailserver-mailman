package mailmgmt

import (
	"bufio"
	"encoding/csv"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
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

var emailListRegex = regexp.MustCompile(`\* ([a-zA-Z0-9_.+-]+@[a-zA-Z0-9-]+\.[a-zA-Z0-9-.]+) \( ([\w\.\~]+) \/ ([\w\.\~]+) \) \[(\d+)%\]`)
var emailListAliasRegex = regexp.MustCompile(`\[\s*aliases\s*->\s+([^\]]*)\]`)

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

		var matches = emailListRegex.FindStringSubmatch(line)
		if len(matches) < 5 {
			matches = emailListAliasRegex.FindStringSubmatch(line)
			if len(matches) < 2 {
				return nil, fmt.Errorf("no alias block found")
			}

			reader := csv.NewReader(
				strings.NewReader(
					strings.TrimSpace(matches[1]),
				),
			)
			reader.TrimLeadingSpace = true // ignore spaces after commas
			reader.LazyQuotes = true       // be lenient with quotes

			records, err := reader.Read()
			if err != nil {
				return nil, fmt.Errorf("csv parse error: %w", err)
			}

			// 3. Trim any remaining whitespace and filter out empty fields
			var aliases = make([]string, 0, len(records))
			for _, field := range records {
				field = strings.TrimSpace(field)
				if field != "" {
					aliases = append(aliases, field)
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
