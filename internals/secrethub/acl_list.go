package secrethub

import (
	"fmt"
	"sort"
	"text/tabwriter"

	"github.com/secrethub/secrethub-go/internals/api/uuid"

	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/internals/errio"
)

// ACLListCommand prints access rules for the given directory.
type ACLListCommand struct {
	path          api.DirPath
	depth         int
	ancestors     bool
	useTimestamps bool
	timeFormatter TimeFormatter
	io            ui.IO
	newClient     newClientFunc
}

// NewACLListCommand creates a new ACLListCommand.
func NewACLListCommand(io ui.IO, newClient newClientFunc) *ACLListCommand {
	return &ACLListCommand{
		io:        io,
		newClient: newClient,
	}
}

// Register registers the command, arguments and flags on the provided Registerer.
func (cmd *ACLListCommand) Register(r Registerer) {
	clause := r.Command("ls", "List access rules of a directory and its children.")
	clause.Arg("dir-path", "The path of the directory to list the access rules for (<namespace>/<repo>[/<dir>])").Required().SetValue(&cmd.path)
	clause.Flag("depth", "The maximum depth to which the rules of child directories should be displayed. Defaults to -1 (no limit).").Short('d').Default("-1").IntVar(&cmd.depth)
	clause.Flag("all", "List all rules that apply on the directory, including rules on parent directories.").Short('a').BoolVar(&cmd.ancestors)
	registerTimestampFlag(clause).BoolVar(&cmd.useTimestamps)

	BindAction(clause, cmd.Run)
}

// Run prints access rules for the given directory.
func (cmd *ACLListCommand) Run() error {
	cmd.beforeRun()
	return cmd.run()
}

// beforeRun configures the command using the flag values.
func (cmd *ACLListCommand) beforeRun() {
	cmd.timeFormatter = NewTimeFormatter(cmd.useTimestamps)
}

func (cmd *ACLListCommand) run() error {
	client, err := cmd.newClient()
	if err != nil {
		return errio.Error(err)
	}

	rules, err := client.AccessRules().List(cmd.path.Value(), cmd.depth, cmd.ancestors)
	if err != nil {
		return errio.Error(err)
	}

	tree, err := client.Dirs().GetTree(cmd.path.Value(), cmd.depth, cmd.ancestors)
	if err != nil {
		return errio.Error(err)
	}

	// Separate all rules into lists of rules per directory.
	ruleIDMap := make(map[uuid.UUID][]int)
	for i, rule := range rules {
		list := ruleIDMap[*rule.DirID]
		ruleIDMap[*rule.DirID] = append(list, i)
	}

	// Map the directories to rule lists.
	ruleMap := make(map[api.DirPath][]*api.AccessRule)
	for dirID, list := range ruleIDMap {
		dirPath, err := tree.AbsDirPath(&dirID)
		if err != nil {
			return errio.Error(err)
		}

		dirRules := make([]*api.AccessRule, len(list))
		for i, ruleIndex := range list {
			dirRules[i] = rules[ruleIndex]
		}

		ruleMap[dirPath] = dirRules
	}

	paths := make([]api.DirPath, len(ruleMap))
	i := 0
	for p := range ruleMap {
		paths[i] = p
		i++
	}

	sort.Sort(api.SortDirPaths(paths))

	tabWriter := tabwriter.NewWriter(cmd.io.Stdout(), 0, 4, 4, ' ', 0)
	fmt.Fprintf(tabWriter, "%s\t%s\t%s\t%s\n", "PATH", "PERMISSIONS", "LAST EDITED", "ACCOUNT")

	for _, p := range paths {
		rulesForPath := ruleMap[p]
		sort.Sort(api.SortAccessRules(rules))

		for _, rule := range rulesForPath {
			fmt.Fprintf(tabWriter, "%s\t%s\t%s\t%s\n",
				p,
				rule.Permission,
				cmd.timeFormatter.Format(rule.LastChangedAt.Local()),
				rule.Account.Name,
			)
		}
	}

	err = tabWriter.Flush()
	if err != nil {
		return errio.Error(err)
	}

	return nil
}
