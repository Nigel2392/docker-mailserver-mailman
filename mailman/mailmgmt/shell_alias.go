package mailmgmt

import (
	"bufio"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"al.essio.dev/pkg/shellescape"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/src/core/errs"
	"github.com/elliotchance/orderedmap/v2"
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

func (m AliasCommand) CommandCountTotal(query string) *Command {
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

// CommandGet retrieves exact alias matches (alias is in column 2)
func (m AliasCommand) CommandGet(alias string) *Command {
	safeAlias := shellescape.Quote(alias)
	awkScript := fmt.Sprintf(`awk -v a=%s '/^\*/ { if (tolower($2) == tolower(a)) print $0 }'`, safeAlias)

	return &Command{
		s: m.s.Arg(fmt.Sprintf("list | %s", awkScript)),
	}
}

// CommandGetByTarget retrieves any lines where the target string appears in column 3.
func (m AliasCommand) CommandGetByTarget(target string) *Command {
	safeTarget := shellescape.Quote(target)
	// We use ~ to do a broad match in awk to catch it inside comma-separated lists,
	// and accurately filter the exact target inside the Go parser.
	awkScript := fmt.Sprintf(`awk -v t=%s '/^\*/ { if (tolower($3) ~ tolower(t)) print $0 }'`, safeTarget)

	return &Command{
		s: m.s.Arg(fmt.Sprintf("list | %s", awkScript)),
	}
}

type AliasListConfig struct {
	Page            int
	Limit           int
	SearchQuery     string
	TargetToAliases bool
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

func (m AliasCommand) Add(alias, target string) error {
	if !IsValidEmail(alias) || !IsValidEmail(target) {
		return fmt.Errorf("alias and target must be valid email addresses: %w", errs.ErrInvalidSyntax)
	}

	_, _, err := m.CommandAdd(alias, target).Exec()
	return err
}

func (m AliasCommand) Delete(alias, target string) error {
	_, _, err := m.CommandDelete(alias, target).Exec()
	return err
}

// CountTotal executes the count command and returns the integer result.
func (m AliasCommand) CountTotal(query string) (int, error) {
	src, _, err := m.CommandCountTotal(query).Exec()
	if err != nil {
		return 0, err
	}

	// Clean the output
	countStr := strings.TrimSpace(src)

	// Convert to integer
	count, err := strconv.Atoi(countStr)
	if err != nil {
		return 0, fmt.Errorf("failed to parse alias count output %q: %v", countStr, err)
	}

	return count, nil
}

type AliasListResult struct {
	Alias   string
	Targets []string
}

// The regex captures the Alias in matches[1] (from EMAIL_REGEX) and the comma-separated targets in matches[2]
var _matchAliasListRegex = regexp.MustCompile(fmt.Sprintf(`^\* %s (\S+)$`, EMAIL_REGEX))

func fwdListOutputMap(resultmap *orderedmap.OrderedMap[string, []string], alias string, targets []string) {
	// Merge targets if the same alias appears multiple times
	if existing, ok := resultmap.Get(alias); ok {
		resultmap.Set(alias, append(existing, targets...))
	} else {
		resultmap.Set(alias, targets)
	}
}

func revListOutputMap(resultmap *orderedmap.OrderedMap[string, []string], alias string, targets []string) {
	for _, target := range targets {
		if existing, ok := resultmap.Get(target); ok {
			resultmap.Set(target, append(existing, alias))
		} else {
			aliases := make([]string, 0, 4)
			aliases = append(aliases, alias)
			resultmap.Set(target, aliases)
		}
	}
}

// parseAliasListOutput is the universal parser that handles comma separation and multi-line deduplication
// it returns an ordered map of (fwd) alias to targets[] or (!fwd) target to []alias
func parseAliasListOutput(src string, fwd bool) (*orderedmap.OrderedMap[string, []string], error) {
	var scanner = bufio.NewScanner(strings.NewReader(src))
	var resultMap = orderedmap.NewOrderedMap[string, []string]() // Use orderedmap to preserve rendering order

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var matches = _matchAliasListRegex.FindStringSubmatch(line)
		if len(matches) < 3 {
			continue
		}

		alias := matches[1]
		targetsBlob := matches[2]

		targets := strings.FieldsFunc(targetsBlob, func(r rune) bool {
			return r == ',' || r == ' '
		})

		if fwd {
			fwdListOutputMap(resultMap, alias, targets)
		} else {
			revListOutputMap(resultMap, alias, targets)
		}

	}

	return resultMap, nil
}

// List returns a list of [alias, []target]
func (m AliasCommand) List(cnf *AliasListConfig) (*orderedmap.OrderedMap[string, []string], error) {
	src, _, err := m.CommandList(cnf).Exec()
	if err != nil {
		return nil, err
	}

	targetToAliases := cnf != nil && cnf.TargetToAliases
	return parseAliasListOutput(src, !targetToAliases) // false means target -> []alias
}

// Get returns all targets belonging to a specific alias string
func (m AliasCommand) Get(address string, byTarget bool) ([]string, error) {
	var cmd func(string) *Command
	if !byTarget {
		cmd = m.CommandGet
	} else {
		cmd = m.CommandGetByTarget
	}

	src, _, err := cmd(address).Exec()
	if err != nil {
		return nil, err
	}

	parsed, err := parseAliasListOutput(src, !byTarget)
	if err != nil {
		return nil, err
	}

	v, ok := parsed.Get(address)
	if !ok {
		return []string{}, errors.NotExists.Wrapf("alias %q not found", address)
	}

	return v, nil
}
