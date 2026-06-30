package mailmgmt

import (
	"bufio"
	"fmt"
	"regexp"
	"strings"

	"al.essio.dev/pkg/shellescape"
)

type AliasCommand struct {
	s SetupCommand
}

func (m AliasCommand) CommandAdd(alias, target string) *Command {
	return &Command{
		s: m.s.Arg("add", shellescape.Quote(alias), shellescape.Quote(target)),
	}
}

func (m AliasCommand) CommandDelete(alias, target string) *Command {
	return &Command{
		s: m.s.Arg("del", shellescape.Quote(alias), shellescape.Quote(target)),
	}
}

type AliasListConfig struct {
	Page        int
	Limit       int
	SearchQuery string
}

func (m AliasCommand) CommandList(config *AliasListConfig) *Command {
	var cfg AliasListConfig
	if config != nil {
		cfg = *config
	}
	if cfg.Page <= 0 {
		cfg.Page = 1
	}
	if cfg.Limit <= 0 {
		cfg.Limit = 10
	}

	startRecord := ((cfg.Page - 1) * cfg.Limit) + 1
	endRecord := cfg.Page * cfg.Limit

	var safeSearch string
	if cfg.SearchQuery != "" {
		safeSearch = shellescape.Quote(cfg.SearchQuery)
	}

	var awkScript string
	if cfg.SearchQuery != "" {
		// Notice -v q=%s without manual single quotes, as shellescape handles them.
		awkScript = fmt.Sprintf(
			`awk -v start=%d -v end=%d -v q=%s '/^\*/ && tolower($0) ~ tolower(q) {c++; if(c>=start && c<=end) print; if(c>end) exit}'`,
			startRecord, endRecord, safeSearch,
		)
	} else {
		awkScript = fmt.Sprintf(
			`awk -v start=%d -v end=%d '/^\*/ {c++; if(c>=start && c<=end) print; if(c>end) exit}'`,
			startRecord, endRecord,
		)
	}

	return &Command{
		s: m.s.Arg("list", "|", awkScript),
	}
}

// CommandGet retrieves all targets for a specific, exact alias match
func (m AliasCommand) CommandGet(alias string) *Command {
	safeAlias := shellescape.Quote(alias)

	// By comparing $2 directly, we ensure exact matches on the alias column only,
	// rather than partial substring matches that could hit the target column.
	// We print $0 (the whole line) so the existing regex parser handles it natively.
	awkScript := fmt.Sprintf(`awk -v a=%s '/^\*/ { if (tolower($2) == tolower(a)) print $0 }'`, safeAlias)

	return &Command{
		s: m.s.Arg(fmt.Sprintf("list | %s", awkScript)),
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

// Map returns a map of target -> []aliases
func (m AliasCommand) Map(cnf *AliasListConfig) (map[string][]string, error) {
	src, _, err := m.CommandList(cnf).Exec()
	if err != nil {
		return nil, err
	}

	var scanner = bufio.NewScanner(strings.NewReader(src))
	var result = make(map[string][]string) // map[target]aliases

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

		// Go's append automatically handles empty map keys elegantly
		result[target] = append(result[target], alias)
	}

	return result, nil
}

// List returns a list of [alias, target]
func (m AliasCommand) List(cnf *AliasListConfig) ([][2]string, error) {
	src, _, err := m.CommandList(cnf).Exec()
	if err != nil {
		return nil, err
	}

	var scanner = bufio.NewScanner(strings.NewReader(src))
	var result = make([][2]string, 0)

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

// Get returns all targets belonging to a specific alias string
func (m AliasCommand) Get(alias string) ([]string, error) {
	src, _, err := m.CommandGet(alias).Exec()
	if err != nil {
		return nil, err
	}

	var scanner = bufio.NewScanner(strings.NewReader(src))
	var targets = make([]string, 0)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var matches = _matchAliasListRegex.FindStringSubmatch(line)
		if len(matches) < 3 {
			continue
		}

		// matches[2] will always contain the target
		targets = append(targets, matches[2])
	}

	return targets, nil
}
