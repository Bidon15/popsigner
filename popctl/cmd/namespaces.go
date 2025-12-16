package cmd

import (
	"context"
	"fmt"

	"github.com/Bidon15/popsigner/popctl/internal/api"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var namespacesCmd = &cobra.Command{
	Use:     "namespaces",
	Aliases: []string{"ns"},
	Short:   "Manage namespaces",
	Long: `Namespace management commands for the POPSigner control plane.

Namespaces are logical groupings of keys within an organization.
Use them to separate keys by environment (prod/staging/dev) or by project.

Examples:
  popctl namespaces list
  popctl namespaces create production --description "Production keys"
  popctl namespaces delete 01HXYZ... --force`,
}

var namespacesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all namespaces",
	RunE:  runNamespacesList,
}

var namespacesCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new namespace",
	Long: `Create a new namespace in the organization.

Examples:
  popctl namespaces create production
  popctl namespaces create staging --description "Staging environment keys"`,
	Args: cobra.ExactArgs(1),
	RunE: runNamespacesCreate,
}

var namespacesGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get namespace details",
	Args:  cobra.ExactArgs(1),
	RunE:  runNamespacesGet,
}

var namespacesDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a namespace",
	Long: `Delete a namespace and all its keys.

WARNING: This will permanently delete all keys in the namespace.
This action cannot be undone.`,
	Args: cobra.ExactArgs(1),
	RunE: runNamespacesDelete,
}

func init() {
	// Create flags
	namespacesCreateCmd.Flags().String("description", "", "namespace description")

	// Delete flags
	namespacesDeleteCmd.Flags().BoolP("force", "f", false, "skip confirmation prompt")
	namespacesDeleteCmd.Flags().BoolP("yes", "y", false, "skip confirmation prompt (alias for --force)")

	// Add subcommands
	namespacesCmd.AddCommand(namespacesListCmd)
	namespacesCmd.AddCommand(namespacesCreateCmd)
	namespacesCmd.AddCommand(namespacesGetCmd)
	namespacesCmd.AddCommand(namespacesDeleteCmd)

	rootCmd.AddCommand(namespacesCmd)
}

// getOrgID gets the first organization ID (most users have one).
func getOrgID(client *api.Client, ctx context.Context) (uuid.UUID, error) {
	orgs, err := client.ListOrganizations(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	if len(orgs) == 0 {
		return uuid.Nil, fmt.Errorf("no organizations found")
	}
	return orgs[0].ID, nil
}

func runNamespacesList(cmd *cobra.Command, args []string) error {
	client, err := getClient()
	if err != nil {
		return err
	}

	ctx := context.Background()

	orgID, err := getOrgID(client, ctx)
	if err != nil {
		printError(err)
		return err
	}

	namespaces, err := client.ListNamespaces(ctx, orgID)
	if err != nil {
		printError(err)
		return err
	}

	if jsonOut {
		return printJSON(map[string]interface{}{
			"namespaces": namespaces,
			"count":      len(namespaces),
		})
	}

	if len(namespaces) == 0 {
		fmt.Println("No namespaces found")
		return nil
	}

	w := newTable()
	printTableHeader(w, "ID", "NAME", "DESCRIPTION", "CREATED")
	for _, ns := range namespaces {
		desc := ns.Description
		if desc == "" {
			desc = "-"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			truncate(ns.ID.String(), 12),
			ns.Name,
			truncate(desc, 30),
			ns.CreatedAt.Format("2006-01-02 15:04"),
		)
	}
	return w.Flush()
}

func runNamespacesCreate(cmd *cobra.Command, args []string) error {
	name := args[0]
	description, _ := cmd.Flags().GetString("description")

	client, err := getClient()
	if err != nil {
		return err
	}

	ctx := context.Background()

	orgID, err := getOrgID(client, ctx)
	if err != nil {
		printError(err)
		return err
	}

	ns, err := client.CreateNamespace(ctx, orgID, api.CreateNamespaceRequest{
		Name:        name,
		Description: description,
	})
	if err != nil {
		printError(err)
		return err
	}

	if jsonOut {
		return printJSON(ns)
	}

	fmt.Printf("%s Created namespace: %s\n", colorGreen("✓"), name)
	fmt.Printf("  ID:          %s\n", ns.ID)
	if description != "" {
		fmt.Printf("  Description: %s\n", description)
	}
	fmt.Printf("\nTo use this namespace by default, add to ~/.popsigner.yaml:\n")
	fmt.Printf("  namespace_id: %s\n", ns.ID)

	return nil
}

func runNamespacesGet(cmd *cobra.Command, args []string) error {
	nsID, err := uuid.Parse(args[0])
	if err != nil {
		return fmt.Errorf("invalid namespace ID: %w", err)
	}

	client, err := getClient()
	if err != nil {
		return err
	}

	ctx := context.Background()

	orgID, err := getOrgID(client, ctx)
	if err != nil {
		printError(err)
		return err
	}

	ns, err := client.GetNamespace(ctx, orgID, nsID)
	if err != nil {
		printError(err)
		return err
	}

	if jsonOut {
		return printJSON(ns)
	}

	fmt.Printf("ID:          %s\n", ns.ID)
	fmt.Printf("Name:        %s\n", ns.Name)
	fmt.Printf("Description: %s\n", ns.Description)
	fmt.Printf("Org ID:      %s\n", ns.OrgID)
	fmt.Printf("Created:     %s\n", ns.CreatedAt.Format("2006-01-02 15:04:05"))

	return nil
}

func runNamespacesDelete(cmd *cobra.Command, args []string) error {
	nsID, err := uuid.Parse(args[0])
	if err != nil {
		return fmt.Errorf("invalid namespace ID: %w", err)
	}

	force, _ := cmd.Flags().GetBool("force")
	yes, _ := cmd.Flags().GetBool("yes")
	if yes {
		force = true
	}

	if !force {
		fmt.Printf("%s Are you sure you want to delete namespace %s?\n", colorYellow("⚠"), nsID)
		fmt.Printf("  %s This will delete ALL KEYS in this namespace!\n", colorRed("WARNING:"))
		fmt.Print("Type 'yes' to confirm: ")
		var response string
		fmt.Scanln(&response)
		if response != "yes" {
			fmt.Println("Aborted")
			return nil
		}
	}

	client, err := getClient()
	if err != nil {
		return err
	}

	ctx := context.Background()

	orgID, err := getOrgID(client, ctx)
	if err != nil {
		printError(err)
		return err
	}

	if err := client.DeleteNamespace(ctx, orgID, nsID); err != nil {
		printError(err)
		return err
	}

	if jsonOut {
		return printJSON(map[string]string{
			"status":  "deleted",
			"message": fmt.Sprintf("Namespace %s deleted", nsID),
		})
	}

	fmt.Printf("%s Deleted namespace: %s\n", colorGreen("✓"), nsID)
	return nil
}

