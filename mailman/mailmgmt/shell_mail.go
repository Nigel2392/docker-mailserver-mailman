package mailmgmt

import (
	"bufio"
	"errors"
	"fmt"
	"net/mail"
	"regexp"
	"strconv"
	"strings"

	"al.essio.dev/pkg/shellescape"
	"github.com/Nigel2392/go-django/src/core/logger"
)

type MailCommand struct {
	s SetupCommand
}

func (m MailCommand) CommandCountTotal(query string) *Command {
	if query == "" {
		return &Command{
			s: m.s.Arg(`list | grep -c '^\*'`),
		}
	}

	safeSearch := shellescape.Quote(query)
	cmdString := fmt.Sprintf(`list | grep '^\*' | grep -ic %s`, safeSearch)
	return &Command{
		s: m.s.Arg(cmdString),
	}
}

func (m MailCommand) CommandAdd(email string, passwd string) *Command {
	return &Command{
		s: m.s.Arg("add", shellescape.Quote(email), shellescape.Quote(passwd)),
	}
}

func (m MailCommand) CommandUpdate(email string, newpasswd string) *Command {
	return &Command{
		s: m.s.Arg("update", shellescape.Quote(email), shellescape.Quote(newpasswd)),
	}
}

func (m MailCommand) CommandDelete(emails ...string) *Command {
	if len(emails) == 0 {
		return &Command{err: errors.New("no email adresses provided to delete command")}
	}
	var nl = make([]string, len(emails)+1)
	nl[0] = "del"
	for i, e := range emails {
		nl[i+1] = shellescape.Quote(e)
	}
	return &Command{
		s: m.s.Arg(nl...),
	}
}

type EmailListConfig struct {
	Page           int
	Limit          int
	SearchQuery    string
	ExcludeAliases bool // If true, only returns the "* email@domain.com..." lines
}

func (m MailCommand) CommandList(config *EmailListConfig) *Command {
	var cfg EmailListConfig
	if config != nil {
		cfg = *config
	}
	if cfg.Page <= 0 {
		cfg.Page = 1
	}
	if cfg.Limit <= 0 { // Changed to 0 so you can actually set limit to 1 if you ever wanted to
		cfg.Limit = 10
	}

	startRecord := ((cfg.Page - 1) * cfg.Limit) + 1
	endRecord := cfg.Page * cfg.Limit

	// Python's shlex.quote equivalent for Bash in Go.
	// We safely escape single quotes so we can wrap the awk variable in single quotes.
	var safeSearch string
	if cfg.SearchQuery != "" {
		safeSearch = shellescape.Quote(cfg.SearchQuery)
	}

	var awkScript string
	switch {
	// Case 1: Searching, NO Aliases
	case cfg.SearchQuery != "" && cfg.ExcludeAliases:
		awkScript = fmt.Sprintf(
			`awk -v start=%d -v end=%d -v q=%s '/^\*/ && tolower($0) ~ tolower(q) {c++; if(c>=start && c<=end) print; if(c>end) exit}'`,
			startRecord, endRecord, safeSearch,
		)

	// Case 2: Searching, WITH Aliases included
	case cfg.SearchQuery != "" && !cfg.ExcludeAliases:
		// Logic: m evaluates to true/false if search matches. If true, increment c. Print while m is true.
		awkScript = fmt.Sprintf(
			`awk -v start=%d -v end=%d -v q=%s '/^\*/ {m=(tolower($0) ~ tolower(q)); if(m) c++} m && c>=start && c<=end {print} c>end {exit}'`,
			startRecord, endRecord, safeSearch,
		)

	// Case 3: No Search, NO Aliases
	case cfg.SearchQuery == "" && cfg.ExcludeAliases:
		awkScript = fmt.Sprintf(
			`awk -v start=%d -v end=%d '/^\*/ {c++; if(c>=start && c<=end) print; if(c>end) exit}'`,
			startRecord, endRecord,
		)

	// Case 4: No Search, WITH Aliases included (Your original default)
	case cfg.SearchQuery == "" && !cfg.ExcludeAliases:
		awkScript = fmt.Sprintf(
			`awk -v start=%d -v end=%d '/^\*/{c++} c>=start && c<=end {print} c>end {exit}'`,
			startRecord, endRecord,
		)
	}

	return &Command{
		s: m.s.Arg(fmt.Sprintf("list | %s", awkScript)),
	}
}

func (m MailCommand) CountTotal(query string) (int, error) {
	src, _, err := m.CommandCountTotal(query).Exec()
	if err != nil {
		return 0, err
	}

	// Clean the output (e.g., "42\n" -> "42")
	countStr := strings.TrimSpace(src)

	// Convert to integer
	count, err := strconv.Atoi(countStr)
	if err != nil {
		return 0, fmt.Errorf("failed to parse total count output %q: %v", countStr, err)
	}

	return count, nil
}

func (m MailCommand) Add(email string, passwd string) error {
	if !_matchEmail.MatchString(email) {
		return errors.New("email does not match predefined pattern")
	}
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

var _matchEmailListRegex = regexp.MustCompile(fmt.Sprintf(`\* %s \( ([\w\.\~]+) \/ ([\w\.\~]+) \) \[(\d+)%%\]`, EMAIL_REGEX))
var _matchEmailListAliasRegex = regexp.MustCompile(`\[\s*aliases\s*->\s+([^\]]*)\]`)

func (m MailCommand) List(cnf *EmailListConfig) ([]ListedAddress, error) {
	src, _, err := m.CommandList(cnf).Exec()
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

		fullP, err := strconv.Atoi(str_fullP)
		if err != nil {
			return nil, fmt.Errorf("invalid percentage full value %q", str_fullP)
		}

		resList = append(resList, ListedAddress{
			Raw:            matches[0],
			Email:          matches[1],
			CurrentQuota:   str_curQ,
			MaxQuota:       str_maxQ,
			PercentageFull: fullP,
		})

		idx++
	}

	return resList, nil
}
