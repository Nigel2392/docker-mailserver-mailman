package mailmgmt

import (
	"bufio"
	"fmt"
	"regexp"
	"strings"
)

type AliasCommand struct {
	s SetupCommand
}

func (m AliasCommand) CommandAdd(alias, target string) *Command {
	return &Command{
		s: m.s.Arg("add", alias, target),
	}
}

func (m AliasCommand) CommandDelete(alias, target string) *Command {
	return &Command{
		s: m.s.Arg("del", alias, target),
	}
}

func (m AliasCommand) CommandList() *Command {
	return &Command{
		s: m.s.Arg("list"),
	}
}

func (m AliasCommand) Add(alias, target string) error {
	_, _, err := m.CommandAdd(alias, target).Exec()
	return err
}

func (m AliasCommand) Delete(alias, target string) error {
	_, _, err := m.CommandDelete(alias, target).Exec()
	return err
}

var _matchAliasListRegex = regexp.MustCompile(fmt.Sprintf(
	`\* %s %s$`, EMAIL_REGEX, EMAIL_REGEX,
))

// Return a map of target -> []aliases
func (m AliasCommand) Map() (map[string][]string, error) {
	src, _, err := m.CommandList().Exec()
	if err != nil {
		return nil, err
	}

	var scanner = bufio.NewScanner(strings.NewReader(src))
	var result = make(map[string][]string) // map[target]aliasses

	for scanner.Scan() {

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var matches = _matchAliasListRegex.FindStringSubmatch(line)
		if len(matches) < 3 {
			continue // not an alias line
		}

		var (
			alias  = matches[1]
			target = matches[2]
		)

		if l, ok := result[target]; ok {
			l = append(l, alias)
			result[target] = l
		} else {
			aliases := make([]string, 0, 1)
			aliases = append(aliases, alias)
			result[target] = aliases
		}
	}

	return result, nil
}

// Return a list of alias -> target
func (m AliasCommand) List() ([][2]string, error) {
	src, _, err := m.CommandList().Exec()
	if err != nil {
		return nil, err
	}

	var scanner = bufio.NewScanner(strings.NewReader(src))
	var result = make([][2]string, 0) // map[target]aliasses

	for scanner.Scan() {

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var matches = _matchAliasListRegex.FindStringSubmatch(line)
		if len(matches) < 3 {
			continue // not an alias line
		}

		result = append(
			result, [2]string{matches[1], matches[2]},
		)
	}

	return result, nil
}
