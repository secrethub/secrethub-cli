package secrethub

import (
	"fmt"
	"os"
	"strings"

	"github.com/secrethub/secrethub-go/pkg/secrethub"
	"github.com/secrethub/secrethub-go/pkg/secrethub/iterator"
	"github.com/spf13/cobra"
)

type AutoCompleter struct {
	client *secrethub.Client
}

// SecretSuggestions provides auto-completions for both arguments and flags
// that take as values paths to secrets SecretHub.
func (ac AutoCompleter) SecretSuggestions(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return getFullPaths(ac.client, toComplete, true), cobra.ShellCompDirectiveNoSpace
}

// DirectorySuggestions provides auto-completions for both arguments and flags
// that take as values paths to directories in SecretHub.
func (ac AutoCompleter) DirectorySuggestions(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return getFullPaths(ac.client, toComplete, false), cobra.ShellCompDirectiveNoSpace
}

// RepositorySuggestions provides auto-completions for both arguments and flags
// that take as values paths to repositories in SecretHub.
func (ac AutoCompleter) RepositorySuggestions(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return getNamespacesAndRepos(ac.client), cobra.ShellCompDirectiveNoSpace
}

// NamespaceSuggestions provides auto-completions for both arguments and flags
// that take as values namespaces in SecretHub.
func (ac AutoCompleter) NamespaceSuggestions(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return getNamespaces(ac.client), cobra.ShellCompDirectiveNoSpace
}

func getNamespaces(client *secrethub.Client) []string {
	var suggestions []string
	iter := client.Orgs().Iterator(&secrethub.OrgIteratorParams{})
	for {
		org, err := iter.Next()
		if err == iterator.Done {
			break
		} else if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err.Error())
			return nil
		}
		suggestions = append(suggestions, org.Name)
	}
	me := client.Me()
	user, _ := me.GetUser()
	suggestions = append(suggestions, user.Username)

	return suggestions
}

func getNamespacesAndRepos(client *secrethub.Client) []string {
	var suggestions []string
	iter := client.Me().RepoIterator(&secrethub.RepoIteratorParams{})
	for {
		repo, err := iter.Next()
		if err == iterator.Done {
			break
		} else if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err.Error())
			return nil
		}
		suggestions = append(suggestions, string(repo.Path()+"/"))
	}
	//TAKE PATHS FROM ORGS
	//iter2 := client.Orgs().Iterator(&secrethub.OrgIteratorParams{})
	//for {
	//	org, err := iter2.Next()
	//	if err == iterator.Done {
	//		break
	//	} else if err != nil {
	//		_, _ = fmt.Fprintln(os.Stderr, err.Error())
	//		return nil
	//	}
	//	suggestions = append(suggestions, org.)
	//}
	return suggestions
}

func getFullPaths(client *secrethub.Client, toComplete string, includeSecrets bool) []string {
	if len(toComplete) == 0 {
		return getNamespacesAndRepos(client)
	}
	tree, err := client.Dirs().GetTree(toComplete, 1, false)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err.Error())
		path := toComplete[0:strings.LastIndex(toComplete, "/")]
		_, err = client.Dirs().GetTree(path, 1, false)
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err.Error())
			return getNamespacesAndRepos(client)
		}
		return getFullPaths(client, path, includeSecrets)
	}
	if strings.LastIndex(toComplete, "/") != len(toComplete)-1 {
		toComplete += "/"
	}
	suggestions := make([]string, tree.DirCount()+tree.SecretCount())

	for _, dir := range tree.RootDir.SubDirs {
		suggestions = append(suggestions, toComplete+dir.Name+"/")
	}

	if includeSecrets {
		for _, secret := range tree.RootDir.Secrets {
			suggestions = append(suggestions, toComplete+secret.Name)
		}
	}
	return suggestions
}

// GetClient returns a new SecretHub client.
func GetClient() *secrethub.Client {
	client, err := secrethub.NewClient()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err.Error())
	}
	return client
}
