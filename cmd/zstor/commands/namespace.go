package commands

import (
	"fmt"

	log "github.com/Sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/zero-os/0-stor/client/itsyouonline"
)

// namespaceCmd represents the namespace for all namespace subcommands
var namespaceCmd = &cobra.Command{
	Use:   "namespace",
	Short: "Manage namespaces and their permissions.",
}

// namespaceCreateCmd represents the namespace-create command
var namespaceCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a namespace.",
	Args:  cobra.ExactArgs(1),
	RunE: func(_cmd *cobra.Command, args []string) error {
		iyoCl, err := getNamespaceManager()
		if err != nil {
			return err
		}

		name := args[0]

		err = iyoCl.CreateNamespace(name)
		if err != nil {
			return fmt.Errorf("creation of namespace %q failed: %v", name, err)
		}

		log.Infof("namespace %q created", name)
		return nil
	},
}

// namespaceDeleteCmd represents the namespace-delete command
var namespaceDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a namespace.",
	Args:  cobra.ExactArgs(1),
	RunE: func(_cmd *cobra.Command, args []string) error {
		iyoCl, err := getNamespaceManager()
		if err != nil {
			return err
		}

		name := args[0]

		err = iyoCl.DeleteNamespace(name)
		if err != nil {
			return fmt.Errorf("deletion of namespace %q failed: %v", name, err)
		}

		log.Infof("namespace %q deleted", name)
		return nil
	},
}

// namespacePermissionCmd represents the namespace for all namespace-permission subcommands
var namespacePermissionCmd = &cobra.Command{
	Use:   "permission",
	Short: "Manage permissions of namespaces.",
}

// namespaceSetPermissionCmd represents the namespace-permission-set command
var namespaceSetPermissionCmd = &cobra.Command{
	Use:   "set <userID> <namespace>",
	Short: "Set permissions.",
	Long:  "Set permissions for a given user and namespace.",
	Args:  cobra.ExactArgs(2),
	RunE: func(_cmd *cobra.Command, args []string) error {
		iyoCl, err := getNamespaceManager()
		if err != nil {
			return err
		}

		userID, namespace := args[0], args[1]
		currentpermissions, err := iyoCl.GetPermission(namespace, userID)
		if err != nil {
			return fmt.Errorf("fail to retrieve permission(s) for %s:%s: %v",
				userID, namespace, err)
		}

		// remove permission if needed
		toRemove := itsyouonline.Permission{
			Read:   currentpermissions.Read && !namespaceSetPermissionCfg.Read,
			Write:  currentpermissions.Write && !namespaceSetPermissionCfg.Write,
			Delete: currentpermissions.Delete && !namespaceSetPermissionCfg.Delete,
			Admin:  currentpermissions.Admin && !namespaceSetPermissionCfg.Admin,
		}
		if err := iyoCl.RemovePermission(namespace, userID, toRemove); err != nil {
			return fmt.Errorf("fail to remove permission(s) for %s:%s: %v",
				userID, namespace, err)
		}

		// add permission if needed
		toAdd := itsyouonline.Permission{
			Read:   !currentpermissions.Read && namespaceSetPermissionCfg.Read,
			Write:  !currentpermissions.Write && namespaceSetPermissionCfg.Write,
			Delete: !currentpermissions.Delete && namespaceSetPermissionCfg.Delete,
			Admin:  !currentpermissions.Admin && namespaceSetPermissionCfg.Admin,
		}

		// Give requested permission
		if err := iyoCl.GivePermission(namespace, userID, toAdd); err != nil {
			return fmt.Errorf("fail to give permission(s) for %s:%s: %v",
				userID, namespace, err)
		}

		return nil
	},
}

var namespaceSetPermissionCfg struct {
	Read, Write, Delete, Admin bool
}

// namespaceGetPermissionCmd represents the namespace-permission-get command
var namespaceGetPermissionCmd = &cobra.Command{
	Use:   "get <userID> <namespace>",
	Short: "Get permissions.",
	Long:  "Get permissions for a given user and namespace.",
	Args:  cobra.ExactArgs(2),
	RunE: func(_cmd *cobra.Command, args []string) error {
		iyoCl, err := getNamespaceManager()
		if err != nil {
			return err
		}

		userID, namespace := args[0], args[1]
		perm, err := iyoCl.GetPermission(namespace, userID)
		if err != nil {
			return fmt.Errorf("fail to retrieve permission for %s:%s: %v",
				userID, namespace, err)
		}

		fmt.Printf("Read: %v\n", perm.Read)
		fmt.Printf("Write: %v\n", perm.Write)
		fmt.Printf("Delete: %v\n", perm.Delete)
		fmt.Printf("Admin: %v\n", perm.Admin)

		return nil
	},
}

func init() {
	namespaceCmd.AddCommand(
		namespaceCreateCmd,
		namespaceDeleteCmd,
		namespacePermissionCmd,
	)

	namespacePermissionCmd.AddCommand(
		namespaceSetPermissionCmd,
		namespaceGetPermissionCmd,
	)

	namespaceSetPermissionCmd.Flags().BoolVarP(
		&namespaceSetPermissionCfg.Read, "read", "r", false,
		"Set read permissions for the given user and namespace.")
	namespaceSetPermissionCmd.Flags().BoolVarP(
		&namespaceSetPermissionCfg.Write, "write", "w", false,
		"Set write permissions for the given user and namespace.")
	namespaceSetPermissionCmd.Flags().BoolVarP(
		&namespaceSetPermissionCfg.Delete, "delete", "d", false,
		"Set delete permissions for the given user and namespace.")
	namespaceSetPermissionCmd.Flags().BoolVarP(
		&namespaceSetPermissionCfg.Admin, "admin", "a", false,
		"Set admin permissions for the given user and namespace.")
}