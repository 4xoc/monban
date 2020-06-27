package main

import (
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/4xoc/monban/models"
	"github.com/4xoc/monban/tasks"

	"github.com/fatih/color"
)

var (
	green  func(string, ...interface{})
	yellow func(string, ...interface{})
	red    func(string, ...interface{})
)

// prettyPrint wraps the whole human readable printing of diffs within the task list
func prettyPrint() {
	// init colors
	if rt.UseColor {
		green = color.New(color.FgGreen).PrintfFunc()
		yellow = color.New(color.FgYellow).PrintfFunc()
		red = color.New(color.FgRed).PrintfFunc()
	} else {
		green = color.New().PrintfFunc()
		yellow = color.New().PrintfFunc()
		red = color.New().PrintfFunc()
	}

	// models.OrganizationalUnit
	prettyPrintOUs()

	// unixGroup
	prettyPrintPosixGroups()

	// unixAccount
	prettyPrintPosixAccounts()

	// models.GroupOfNames
	prettyPrintGroupOfNames()

	// SUDOer roles
	prettyPrintSUDOers()
}

// prettyPrintOUs displays OU changes in a pretty way
func prettyPrintOUs() {
	fmt.Printf("\n==> OrganizationalUnits <==\n")

	queueCreateOU.For(func(task *tasks.Task) error {
		green("++ dn: %s\n",
			*task.DN)
		return nil
	})

	queueDeleteOU.For(func(task *tasks.Task) error {
		green("-- dn: %s\n",
			*task.DN)
		return nil
	})
}

// prettyPrintPosixGroups displays models.PosixGroup changes in a pretty way
func prettyPrintPosixGroups() {
	var (
		data *models.PosixGroup
	)

	fmt.Printf("\n==> PosixGroups <==")

	queueCreatePG.For(func(task *tasks.Task) error {
		data = task.Data.PosixGroup

		green("\n++ dn:          %s\n   name:        %s\n   gid_number:  %d\n   description: %s\n",
			*task.DN,
			data.CN,
			data.GIDNumber,
			data.Description)
		return nil
	})

	queueUpdatePG.For(func(task *tasks.Task) error {
		yellow("\n~~ dn:          %s\n", *task.DN)

		if data.GIDNumber != nil {
			yellow("   gid_number:  %d\n",
				*data.GIDNumber)
		}

		if data.Description != "" {
			yellow("   description:  %s\n",
				data.Description)
		}
		return nil
	})

	fmt.Println()

	queueDeletePG.For(func(task *tasks.Task) error {
		red("-- dn: %s\n", *task.DN)
		return nil
	})
}

// prettyPrintPosixAccounts displays models.PosixAccount changes in a pretty way
func prettyPrintPosixAccounts() {
	var (
		data *models.PosixAccount
	)

	fmt.Printf("\n==> PosixAccounts <==")

	queueCreatePA.For(func(task *tasks.Task) error {
		data = task.Data.PosixAccount

		green("\n++ dn:             %s\n   username:       %s\n   given_name:     %s\n   surname:        %s\n   display_name:   %s\n",
			*task.DN,
			*data.UID,
			*data.GivenName,
			*data.Surname,
			*data.DisplayName)

		if data.UIDNumber != nil {
			green("   uid_number:     *known after sync*\n")
		} else {
			green("   uid_number:     %d\n",
				*data.UIDNumber)
		}

		green("   gid_number:     %d\n   login_shell:    %s\n   mail:           %s\n   home_dir:       %s\n",
			*data.GIDNumber,
			*data.LoginShell,
			*data.Mail,
			*data.HomeDir)

		// check for additional, optional information
		if *rt.Config.EnableSSHPublicKeys {
			if data.SSHPublicKey == nil {
				green("   ssh_public_key: *none*\n")

			} else {
				green("   ssh_public_key: %s\n",
					*data.SSHPublicKey)
			}
		}

		if data.UserPassword != nil {
			green("   user_password:  %s\n",
				*data.UserPassword)
		}
		return nil
	})

	queueUpdatePA.For(func(task *tasks.Task) error {
		data = task.Data.PosixAccount

		yellow("\n~~ dn:             %s\n", *task.DN)
		// changing username will result in a new object therefore not checking for a change here

		if data.GivenName != nil {
			yellow("   given_name:     %s\n",
				*data.GivenName)
		}

		if data.Surname != nil {
			yellow("   surname:        %s\n",
				*data.Surname)
		}

		if data.DisplayName != nil {
			yellow("   display_name:    %s\n",
				*data.DisplayName)
		}

		if data.UIDNumber != nil {
			yellow("   uid_number:     %d\n",
				*data.UIDNumber)
		}

		if data.GIDNumber != nil {
			yellow("   gid_number:     %d\n",
				*data.GIDNumber)
		}

		if data.LoginShell != nil {
			yellow("   login_shell:    %s\n",
				*data.LoginShell)
		}

		if data.Mail != nil {
			yellow("   mail:           %s\n",
				*data.Mail)
		}

		if data.HomeDir != nil {
			yellow("   home_dir:       %s\n",
				*data.HomeDir)
		}

		// check for additional, optional information
		if *rt.Config.EnableSSHPublicKeys &&
			data.SSHPublicKey != nil {

			if *data.SSHPublicKey != "" {
				// key is to be deleted
				red("   ssh_public_key: *to be removed*\n")
			} else {
				yellow("   ssh_public_key: %s\n",
					*data.SSHPublicKey)
			}
		}

		if data.UserPassword != nil {
			yellow("   user_password:  %s\n",
				*data.UserPassword)
		}
		return nil
	})

	fmt.Println()

	queueDeletePA.For(func(task *tasks.Task) error {
		red("-- dn: %s\n", *task.DN)
		return nil
	})
}

// prettyPrintGroupOfNames displays groupOfName changes in a pretty way
func prettyPrintGroupOfNames() {
	var (
		i  int
		ok bool
		// map[DN][add|del][]userDN
		changedMembers map[string]map[string][]string
		dn             string

		data *models.GroupOfNames
	)

	changedMembers = make(map[string]map[string][]string)

	// reorganize addMember & deleteMember tasks
	queueAddMemberGoN.For(func(task *tasks.Task) error {

		// init map for this DN
		if changedMembers[*task.DN] == nil {
			changedMembers[*task.DN] = make(map[string][]string)
		}

		changedMembers[*task.DN]["add"] = append(changedMembers[*task.DN]["add"], *task.Data.DN)
		return nil
	})

	queueDelMemberGoN.For(func(task *tasks.Task) error {

		// init map for this DN
		if changedMembers[*task.DN] == nil {
			changedMembers[*task.DN] = make(map[string][]string)
		}

		changedMembers[*task.DN]["del"] = append(changedMembers[*task.DN]["del"], *task.Data.DN)
		return nil
	})

	fmt.Printf("\n==> GroupOfNames <==")

	queueCreateGoN.For(func(task *tasks.Task) error {
		data = task.Data.GroupOfNames

		green("\n++ dn:          %s\n   cn:          %s\n   description: %s\n",
			data.DN,
			data.CN,
			data.Description)

		// check for new members of this group
		if _, ok = changedMembers[*task.DN]; ok {
			green("   members:\n")

			// check for members to add
			for _, dn = range changedMembers[*task.DN]["add"] {
				green("     ++ dn:     %s\n", dn)
			}

			// delete from map
			delete(changedMembers, *task.DN)
		}
		return nil
	})

	queueUpdateGoN.For(func(task *tasks.Task) error {
		data = task.Data.GroupOfNames

		yellow("\n~~ dn:          %s\n", *task.DN)

		if data.Description != "" {
			yellow("   description: %s\n",
				data.Description)
		}

		// check for member changes in this models.GroupOfNames
		if _, ok = changedMembers[*task.DN]; ok {
			yellow("   members:\n")

			// check for members to add
			for _, dn = range changedMembers[*task.DN]["add"] {
				green("     ++ dn:     %s\n", dn)
			}

			// check for members to delete
			for _, dn = range changedMembers[*task.DN]["del"] {
				red("     -- dn:     %s\n", dn)
			}

			// delete from map
			delete(changedMembers, *task.DN)
		}
		return nil
	})

	// checking for member changes for any other dn
	for dn = range changedMembers {
		yellow("\n~~ dn:          %s\n", dn)
		yellow("   members:\n")

		for i = range changedMembers[dn]["add"] {
			green("     ++ dn:     %s\n", changedMembers[dn]["add"][i])
		}

		for i = range changedMembers[dn]["del"] {
			red("     -- dn:     %s\n", changedMembers[dn]["del"][i])
		}
	}

	fmt.Println()

	queueDeleteGoN.For(func(task *tasks.Task) error {
		red("-- dn: %s\n", *task.DN)
		return nil
	})
}

// prettyPrintSUDOers displayes SUDOer diffs in a nice way
func prettyPrintSUDOers() {
	var (
		i    int
		data *models.SudoRole
	)

	fmt.Printf("\n==> SUDOers <==")

	queueCreateSR.For(func(task *tasks.Task) error {
		data = task.Data.SudoRole

		green("\n++ dn:  %s\n   name:   %s\n",
			data.DN,
			data.CN)

		// now all optional details
		if data.Description != "" {
			green("   description:\t%s\n", data.Description)
		}

		if len(data.SudoUser) > 0 {
			green("   sudo_user:\t\n")

			for i = range data.SudoUser {
				green("     value:\t%s\n", data.SudoUser[i])
			}
		}

		if len(data.SudoHost) > 0 {
			green("   sudo_host:\t\n")

			for i = range data.SudoHost {
				green("     value:\t%s\n", data.SudoHost[i])
			}
		}

		if len(data.SudoCommand) > 0 {
			green("   sudo_command:\t\n")

			for i = range data.SudoCommand {
				green("     value:\t%s\n", data.SudoCommand[i])
			}
		}

		if len(data.SudoOption) > 0 {
			green("   sudo_option:\t\n")

			for i = range data.SudoOption {
				green("     value:\t%s\n", data.SudoOption[i])
			}
		}

		if len(data.SudoRunAsUser) > 0 {
			green("   sudo_run_as_user:\t\n")

			for i = range data.SudoRunAsUser {
				green("     value:\t%s\n", data.SudoRunAsUser[i])
			}
		}

		if len(data.SudoRunAsGroup) > 0 {
			green("   sudo_run_as_group:\t\n")

			for i = range data.SudoRunAsGroup {
				green("     value:\t%s\n", data.SudoRunAsGroup[i])
			}
		}

		if len(data.SudoNotBefore) > 0 {
			green("   sudo_not_before:\t\n")

			for i = range data.SudoNotBefore {
				green("     value:\t%s\n", data.SudoNotBefore[i])
			}
		}

		if len(data.SudoNotAfter) > 0 {
			green("   sudo_not_after:\t\n")

			for i = range data.SudoNotAfter {
				green("     value:\t%s\n", data.SudoNotAfter[i])
			}
		}

		if data.SudoOrder != nil {
			green("   sudo_order:\t%d\n", *data.SudoOrder)
		}
		return nil
	})

	queueUpdateSR.For(func(task *tasks.Task) error {
		data = task.Data.SudoRole

		yellow("\n~~ dn:\t%s\n", data.DN)

		// now all optional details
		if data.Description != "" {
			yellow("   description:\t%s\n", data.Description)
		}

		if len(data.SudoUser) > 0 {
			yellow("   sudo_user:\t\n")

			for i = range data.SudoUser {
				yellow("     value:\t%s\n", data.SudoUser[i])
			}
		}

		if len(data.SudoHost) > 0 {
			yellow("   sudo_host:\t\n")

			for i = range data.SudoHost {
				yellow("     value:\t%s\n", data.SudoHost[i])
			}
		}

		if len(data.SudoCommand) > 0 {
			yellow("   sudo_command:\t\n")

			for i = range data.SudoCommand {
				yellow("     value:\t%s\n", data.SudoCommand[i])
			}
		}

		if len(data.SudoOption) > 0 {
			yellow("   sudo_option:\t\n")

			for i = range data.SudoOption {
				yellow("     value:\t%s\n", data.SudoOption[i])
			}
		}

		if len(data.SudoRunAsUser) > 0 {
			yellow("   sudo_run_as_user:\t\n")

			for i = range data.SudoRunAsUser {
				yellow("     value:\t%s\n", data.SudoRunAsUser[i])
			}
		}

		if len(data.SudoRunAsGroup) > 0 {
			yellow("   sudo_run_as_group:\t\n")

			for i = range data.SudoRunAsGroup {
				yellow("     value:\t%s\n", data.SudoRunAsGroup[i])
			}
		}

		if len(data.SudoNotBefore) > 0 {
			yellow("   sudo_not_before:\t\n")

			for i = range data.SudoNotBefore {
				yellow("     value:\t%s\n", data.SudoNotBefore[i])
			}
		}

		if len(data.SudoNotAfter) > 0 {
			yellow("   sudo_not_after:\t\n")

			for i = range data.SudoNotAfter {
				yellow("     value:\t%s\n", data.SudoNotAfter[i])
			}
		}

		if data.SudoOrder != nil {
			yellow("   sudo_order:\t%d\n", *data.SudoOrder)
		}
		return nil
	})

	queueDeleteSR.For(func(task *tasks.Task) error {
		red("-- dn: %s\n", *task.DN)
		return nil
	})

	fmt.Println()
}

// prettyPrintAudit displays audit information in a nice way
func prettyPrintAudit() {
	var (
		dn      string
		index   int
		dn2     string
		index2  int
		w       *tabwriter.Writer
		tmpList []string
	)

	// init tabwriter
	w = new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 8, 0, '\t', 0)

	fmt.Printf("====== START AUDIT ======")

	// first pretty print people and their memberships
	for dn = range ldapPeople {

		fmt.Printf("\n====== START POSIXGROUP ======")

		fmt.Fprintf(w, "\ndn:\t%s\nname:\t%s\ndescription:\t%s\ngid_number:\t%d\nmembers:",
			ldapPeople[dn].DN,
			ldapPeople[dn].CN,
			ldapPeople[dn].Description,
			*ldapPeople[dn].GIDNumber)

		w.Flush()

		for index = range localPeople[dn].Objects {

			//fmt.Fprintf(w, "\ndn:\t%s\nusername:\t%s\ngiven_name:\t%s\nsurname:\t%s\ndisplay_name:\t%s\nuid_number:\t%d\ngid_number:\t%d\nlogin_shell:\t%s\nmail:\t%s\nhome_dir:\t%s\n",
			fmt.Fprintf(w, "\n  dn:\t%s\n  username:\t%s\n  given_name:\t%s\n  surname:\t%s\n  display_name:\t%s\n  uid_number:\t%d\n  gid_number:\t%d\n  login_shell:\t%s\n  mail:\t%s\n  home_dir:\t%s\n",
				ldapPeople[dn].Objects[index].DN,
				*ldapPeople[dn].Objects[index].UID,
				*ldapPeople[dn].Objects[index].GivenName,
				*ldapPeople[dn].Objects[index].Surname,
				*ldapPeople[dn].Objects[index].DisplayName,
				*ldapPeople[dn].Objects[index].UIDNumber,
				*ldapPeople[dn].Objects[index].GIDNumber,
				*ldapPeople[dn].Objects[index].LoginShell,
				*ldapPeople[dn].Objects[index].Mail,
				*ldapPeople[dn].Objects[index].HomeDir)

			// check for additional, optional information
			if ldapPeople[dn].Objects[index].SSHPublicKey == nil {
				fmt.Fprintf(w, "  ssh_public_key:\t*none*\n")

			} else {
				fmt.Fprintf(w, "  ssh_public_key:\t%s\n",
					*ldapPeople[dn].Objects[index].SSHPublicKey)
			}

			fmt.Fprintf(w, "  group_memberships:\t\n")

			// gather and sort memberof DNs
			for dn2 = range ldapGroups {
				for index2 = range ldapGroups[dn2].Members {
					if *ldapPeople[dn].Objects[index].UID == ldapGroups[dn2].Members[index2] {
						tmpList = append(tmpList, dn2)
					}
				}
			}

			sort.Slice(tmpList, func(i, j int) bool {
				return len(tmpList[i]) < len(tmpList[j])
			})

			for index2 = range tmpList {
				fmt.Fprintf(w, "    %s\n", tmpList[index2])
			}

			w.Flush()
		}

		fmt.Printf("\n====== END POSIXGROUP ======\n")
	}

	fmt.Printf("\n====== START GROUPOFNAMES ======\n")

	// TODO: next print SUDOer roles

	fmt.Printf("\n====== END GROUPOFNAMES ======\n")

	fmt.Printf("====== END AUDIT ======\n")

}
