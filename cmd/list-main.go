package cmd

import (
	"fmt"
	"os"
	"path"
	"sort"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/zbiljic/sicc/store"
)

// listCmd represents the 'list' command
var listCmd = &cobra.Command{
	Use:   "list <prefix>",
	Short: "List the configurations for a prefix",
	Args:  cobra.ExactArgs(1), //nolint:gomnd
	RunE:  runList,
}

var listParameters struct {
	WithValues    bool
	SortByTime    bool
	SortByUser    bool
	SortByVersion bool
}

func init() {
	listCmd.Flags().BoolVarP(&listParameters.WithValues, "expand", "e", false, "Expand parameter list with values")
	listCmd.Flags().BoolVarP(&listParameters.SortByTime, "time", "t", false, "Sort by modified time")
	listCmd.Flags().BoolVarP(&listParameters.SortByUser, "user", "u", false, "Sort by user")
	listCmd.Flags().BoolVarP(&listParameters.SortByVersion, "version", "v", false, "Sort by version")
	// add 'list' command to root command
	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) error {
	prefixPath := path.Join("/", args[0])

	if err := validateConfigPathName(prefixPath); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	configStore, err := getConfigurationStore()
	if err != nil {
		return fmt.Errorf("failed to get configuration store: %w", err)
	}

	configs, err := configStore.List(prefixPath, listParameters.WithValues)
	if err != nil {
		return fmt.Errorf("failed to list store contents (%s): %w", prefixPath, err)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, '\t', 0)

	fmt.Fprint(w, "Key\tVersion\tLastModified\tUser")

	if listParameters.WithValues {
		fmt.Fprint(w, "\tValue")
	}

	fmt.Fprintln(w, "")

	sort.Sort(ByName(configs))

	if listParameters.SortByTime {
		sort.Sort(ByTime(configs))
	}

	if listParameters.SortByUser {
		sort.Sort(ByUser(configs))
	}

	if listParameters.SortByVersion {
		sort.Sort(ByVersion(configs))
	}

	for _, config := range configs {
		fmt.Fprintf(w, "%s\t%d\t%s\t%s",
			stripPrefix(config.Meta.Key, prefixPath),
			config.Meta.Version,
			config.Meta.LastModifiedDate.Local().Format(shortTimeFormat),
			config.Meta.LastModifiedUser,
		)

		if listParameters.WithValues {
			fmt.Fprintf(w, "\t%s", *config.Value)
		}

		fmt.Fprintln(w, "")
	}

	w.Flush()

	return nil
}

type ByName []store.Value

func (a ByName) Len() int           { return len(a) }
func (a ByName) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByName) Less(i, j int) bool { return a[i].Meta.Key < a[j].Meta.Key }

type ByTime []store.Value

func (a ByTime) Len() int      { return len(a) }
func (a ByTime) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByTime) Less(i, j int) bool {
	return a[i].Meta.LastModifiedDate.Before(a[j].Meta.LastModifiedDate)
}

type ByUser []store.Value

func (a ByUser) Len() int           { return len(a) }
func (a ByUser) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByUser) Less(i, j int) bool { return a[i].Meta.LastModifiedUser < a[j].Meta.LastModifiedUser }

type ByVersion []store.Value

func (a ByVersion) Len() int           { return len(a) }
func (a ByVersion) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByVersion) Less(i, j int) bool { return a[i].Meta.Version < a[j].Meta.Version }
