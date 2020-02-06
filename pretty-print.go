package main

import (
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

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
	if useColor {
		green = color.New(color.FgGreen).PrintfFunc()
		yellow = color.New(color.FgYellow).PrintfFunc()
		red = color.New(color.FgRed).PrintfFunc()
	} else {
		green = color.New().PrintfFunc()
		yellow = color.New().PrintfFunc()
		red = color.New().PrintfFunc()
	}

	// organizationalUnit
	prettyPrintOUs()

	// unixGroup
	prettyPrintPosixGroups()

	// unixAccount
	prettyPrintPosixAccounts()

	// groupOfNames
	prettyPrintGroupOfNames()

	// SUDOer roles
	prettyPrintSUDOers()
}

// prettyPrintOUs displays OU changes in a pretty way
func prettyPrintOUs() {
	var (
		i int
	)

	fmt.Printf("\n==> OrganizationalUnits <==\n")

	if len(taskList[objectTypeOrganisationalUnit][taskTypeCreate]) > 0 {
		for i = range taskList[objectTypeOrganisationalUnit][taskTypeCreate] {
			green("++ dn: %s\n",
				taskList[objectTypeOrganisationalUnit][taskTypeCreate][i].data.(*organizationalUnit).dn)
		}
	}

	if len(taskList[objectTypeOrganisationalUnit][taskTypeDelete]) > 0 {
		fmt.Println()
		for i = range taskList[objectTypeOrganisationalUnit][taskTypeDelete] {
			red("-- dn: %s\n", *taskList[objectTypeOrganisationalUnit][taskTypeDelete][i].dn)
		}
	}
}

// prettyPrintPosixGroups displays posixGroup changes in a pretty way
func prettyPrintPosixGroups() {
	var (
		i int
	)

	fmt.Printf("\n==> PosixGroups <==")

	if len(taskList[objectTypePosixGroup][taskTypeCreate]) > 0 {
		for i = range taskList[objectTypePosixGroup][taskTypeCreate] {
			green("\n++ dn:          %s\n   name:        %s\n   gid_number:  %d\n   description: %s\n",
				taskList[objectTypePosixGroup][taskTypeCreate][i].data.(posixGroup).dn,
				taskList[objectTypePosixGroup][taskTypeCreate][i].data.(posixGroup).CN,
				*taskList[objectTypePosixGroup][taskTypeCreate][i].data.(posixGroup).GIDNumber,
				taskList[objectTypePosixGroup][taskTypeCreate][i].data.(posixGroup).Description)
		}
	}

	if len(taskList[objectTypePosixGroup][taskTypeUpdate]) > 0 {
		for i = range taskList[objectTypePosixGroup][taskTypeUpdate] {
			yellow("\n~~ dn:          %s\n", taskList[objectTypePosixGroup][taskTypeUpdate][i].data.(posixGroup).dn)

			if taskList[objectTypePosixGroup][taskTypeUpdate][i].data.(posixGroup).GIDNumber != nil {
				yellow("   gid_number:  %d\n",
					*taskList[objectTypePosixGroup][taskTypeUpdate][i].data.(posixGroup).GIDNumber)
			}

			if taskList[objectTypePosixGroup][taskTypeUpdate][i].data.(posixGroup).Description != "" {
				yellow("   description:  %s\n",
					taskList[objectTypePosixGroup][taskTypeUpdate][i].data.(posixGroup).Description)
			}
		}
	}

	if len(taskList[objectTypePosixGroup][taskTypeDelete]) > 0 {
		fmt.Println()
		for i = range taskList[objectTypePosixGroup][taskTypeDelete] {
			red("-- dn: %s\n", *taskList[objectTypePosixGroup][taskTypeDelete][i].dn)
		}
	}
}

// prettyPrintPosixAccounts displays posixAccount changes in a pretty way
func prettyPrintPosixAccounts() {
	var (
		i int
	)

	fmt.Printf("\n==> PosixAccounts <==")

	if len(taskList[objectTypePosixGroup][taskTypeCreate]) > 0 {
		for i = range taskList[objectTypePosixGroup][taskTypeCreate] {

			green("\n++ dn:             %s\n   username:       %s\n   given_name:     %s\n   surname:        %s\n   display_name:   %s\n",
				taskList[objectTypePosixAccount][taskTypeCreate][i].data.(posixAccount).dn,
				*taskList[objectTypePosixAccount][taskTypeCreate][i].data.(posixAccount).UID,
				*taskList[objectTypePosixAccount][taskTypeCreate][i].data.(posixAccount).GivenName,
				*taskList[objectTypePosixAccount][taskTypeCreate][i].data.(posixAccount).Surname,
				*taskList[objectTypePosixAccount][taskTypeCreate][i].data.(posixAccount).DisplayName)

			if taskList[objectTypePosixAccount][taskTypeCreate][i].data.(posixAccount).UIDNumber == nil {
				green("   uid_number:     *known after sync*\n")
			} else {
				green("   uid_number:     %d\n",
					*taskList[objectTypePosixAccount][taskTypeCreate][i].data.(posixAccount).UIDNumber)
			}

			green("   gid_number:     %d\n   login_shell:    %s\n   mail:           %s\n   home_dir:       %s\n",
				*taskList[objectTypePosixAccount][taskTypeCreate][i].data.(posixAccount).GIDNumber,
				*taskList[objectTypePosixAccount][taskTypeCreate][i].data.(posixAccount).LoginShell,
				*taskList[objectTypePosixAccount][taskTypeCreate][i].data.(posixAccount).Mail,
				*taskList[objectTypePosixAccount][taskTypeCreate][i].data.(posixAccount).HomeDir)

			// check for additional, optional information
			if *config.EnableSSHPublicKeys {
				if taskList[objectTypePosixAccount][taskTypeCreate][i].data.(posixAccount).SSHPublicKey == nil {
					green("   ssh_public_key: *none*\n")

				} else {
					green("   ssh_public_key: %s\n",
						*taskList[objectTypePosixAccount][taskTypeCreate][i].data.(posixAccount).SSHPublicKey)
				}
			}

			if taskList[objectTypePosixAccount][taskTypeCreate][i].data.(posixAccount).UserPassword != nil {
				green("   user_password:  %s\n",
					*taskList[objectTypePosixAccount][taskTypeCreate][i].data.(posixAccount).UserPassword)
			}
		}
	}

	if len(taskList[objectTypePosixAccount][taskTypeUpdate]) > 0 {
		for i = range taskList[objectTypePosixAccount][taskTypeUpdate] {
			yellow("\n~~ dn:             %s\n", taskList[objectTypePosixAccount][taskTypeUpdate][i].data.(posixAccount).dn)

			// changing username will result in a new object therefore not checking for a change here

			if taskList[objectTypePosixAccount][taskTypeUpdate][i].data.(posixAccount).GivenName != nil {
				yellow("   given_name:     %s\n",
					*taskList[objectTypePosixAccount][taskTypeUpdate][i].data.(posixAccount).GivenName)
			}

			if taskList[objectTypePosixAccount][taskTypeUpdate][i].data.(posixAccount).Surname != nil {
				yellow("   surname:        %s\n",
					*taskList[objectTypePosixAccount][taskTypeUpdate][i].data.(posixAccount).Surname)
			}

			if taskList[objectTypePosixAccount][taskTypeUpdate][i].data.(posixAccount).DisplayName != nil {
				yellow("   display_name:    %s\n",
					*taskList[objectTypePosixAccount][taskTypeUpdate][i].data.(posixAccount).DisplayName)
			}

			if taskList[objectTypePosixAccount][taskTypeUpdate][i].data.(posixAccount).UIDNumber != nil {
				yellow("   uid_number:     %d\n",
					*taskList[objectTypePosixAccount][taskTypeUpdate][i].data.(posixAccount).UIDNumber)
			}

			if taskList[objectTypePosixAccount][taskTypeUpdate][i].data.(posixAccount).GIDNumber != nil {
				yellow("   gid_number:     %d\n",
					*taskList[objectTypePosixAccount][taskTypeUpdate][i].data.(posixAccount).GIDNumber)
			}

			if taskList[objectTypePosixAccount][taskTypeUpdate][i].data.(posixAccount).LoginShell != nil {
				yellow("   login_shell:    %s\n",
					*taskList[objectTypePosixAccount][taskTypeUpdate][i].data.(posixAccount).LoginShell)
			}

			if taskList[objectTypePosixAccount][taskTypeUpdate][i].data.(posixAccount).Mail != nil {
				yellow("   mail:           %s\n",
					*taskList[objectTypePosixAccount][taskTypeUpdate][i].data.(posixAccount).Mail)
			}

			if taskList[objectTypePosixAccount][taskTypeUpdate][i].data.(posixAccount).HomeDir != nil {
				yellow("   home_dir:       %s\n",
					*taskList[objectTypePosixAccount][taskTypeUpdate][i].data.(posixAccount).HomeDir)
			}

			// check for additional, optional information
			if *config.EnableSSHPublicKeys &&
				taskList[objectTypePosixAccount][taskTypeUpdate][i].data.(posixAccount).SSHPublicKey != nil {

				if *taskList[objectTypePosixAccount][taskTypeUpdate][i].data.(posixAccount).SSHPublicKey == "" {
					// key is to be deleted
					red("   ssh_public_key: *to be removed*\n")
				} else {
					yellow("   ssh_public_key: %s\n",
						*taskList[objectTypePosixAccount][taskTypeUpdate][i].data.(posixAccount).SSHPublicKey)
				}
			}

			if taskList[objectTypePosixAccount][taskTypeUpdate][i].data.(posixAccount).UserPassword != nil {
				yellow("   user_password:  %s\n",
					*taskList[objectTypePosixAccount][taskTypeUpdate][i].data.(posixAccount).UserPassword)
			}
		}
	}

	if len(taskList[objectTypePosixAccount][taskTypeDelete]) > 0 {
		fmt.Println()
		for i = range taskList[objectTypePosixAccount][taskTypeDelete] {
			red("-- dn: %s\n", *taskList[objectTypePosixAccount][taskTypeDelete][i].dn)
		}
	}
}

// prettyPrintGroupOfNames displays groupOfName changes in a pretty way
func prettyPrintGroupOfNames() {
	var (
		i  int
		j  int
		ok bool
		// map[DN][add|del][]userDN
		changedMembers map[string]map[string][]string
		dn             string
	)

	changedMembers = make(map[string]map[string][]string)

	// reorganize addMember & deleteMember tasks
	if len(taskList[objectTypeGroupOfNames][taskTypeAddMember]) > 0 {
		for i = range taskList[objectTypeGroupOfNames][taskTypeAddMember] {
			// init map for this DN
			if changedMembers[*taskList[objectTypeGroupOfNames][taskTypeAddMember][i].dn] == nil {
				changedMembers[*taskList[objectTypeGroupOfNames][taskTypeAddMember][i].dn] = make(map[string][]string)
			}

			changedMembers[*taskList[objectTypeGroupOfNames][taskTypeAddMember][i].dn]["add"] = append(changedMembers[*taskList[objectTypeGroupOfNames][taskTypeAddMember][i].dn]["add"], taskList[objectTypeGroupOfNames][taskTypeAddMember][i].data.(string))
		}
	}

	if len(taskList[objectTypeGroupOfNames][taskTypeDeleteMember]) > 0 {
		for i = range taskList[objectTypeGroupOfNames][taskTypeDeleteMember] {
			if changedMembers[*taskList[objectTypeGroupOfNames][taskTypeDeleteMember][i].dn] == nil {
				changedMembers[*taskList[objectTypeGroupOfNames][taskTypeDeleteMember][i].dn] = make(map[string][]string)
			}

			changedMembers[*taskList[objectTypeGroupOfNames][taskTypeDeleteMember][i].dn]["del"] = append(changedMembers[*taskList[objectTypeGroupOfNames][taskTypeDeleteMember][i].dn]["del"], taskList[objectTypeGroupOfNames][taskTypeDeleteMember][i].data.(string))
		}
	}

	fmt.Printf("\n==> GroupOfNames <==")

	if len(taskList[objectTypeGroupOfNames][taskTypeCreate]) > 0 {
		for i = range taskList[objectTypeGroupOfNames][taskTypeCreate] {
			green("\n++ dn:          %s\n   cn:          %s\n   description: %s\n",
				taskList[objectTypeGroupOfNames][taskTypeCreate][i].data.(groupOfNames).dn,
				taskList[objectTypeGroupOfNames][taskTypeCreate][i].data.(groupOfNames).CN,
				taskList[objectTypeGroupOfNames][taskTypeCreate][i].data.(groupOfNames).Description)

			// check for new members of this group
			if _, ok = changedMembers[taskList[objectTypeGroupOfNames][taskTypeCreate][i].data.(groupOfNames).dn]; ok {
				green("   members:\n")

				// check for members to add
				for j = range changedMembers[taskList[objectTypeGroupOfNames][taskTypeCreate][i].data.(groupOfNames).dn]["add"] {

					green("     ++ dn:     %s\n", changedMembers[taskList[objectTypeGroupOfNames][taskTypeCreate][i].data.(groupOfNames).dn]["add"][j])
				}

				// delete from map
				delete(changedMembers, taskList[objectTypeGroupOfNames][taskTypeCreate][i].data.(groupOfNames).dn)
			}
		}
	}

	if len(taskList[objectTypeGroupOfNames][taskTypeUpdate]) > 0 {
		for i = range taskList[objectTypeGroupOfNames][taskTypeUpdate] {
			yellow("\n~~ dn:          %s\n", taskList[objectTypeGroupOfNames][taskTypeUpdate][i].data.(groupOfNames).dn)

			if taskList[objectTypeGroupOfNames][taskTypeUpdate][i].data.(groupOfNames).Description != "" {
				yellow("   description: %s\n",
					taskList[objectTypeGroupOfNames][taskTypeUpdate][i].data.(groupOfNames).Description)
			}

			// check for member changes in this groupOfNames
			if _, ok = changedMembers[taskList[objectTypeGroupOfNames][taskTypeUpdate][i].data.(groupOfNames).dn]; ok {
				yellow("   members:\n")

				// check for members to add
				for j = range changedMembers[taskList[objectTypeGroupOfNames][taskTypeUpdate][i].data.(groupOfNames).dn]["add"] {

					green("     ++ dn:     %s\n", changedMembers[taskList[objectTypeGroupOfNames][taskTypeUpdate][i].data.(groupOfNames).dn]["add"][j])
				}

				// check for members to delete
				for j = range changedMembers[taskList[objectTypeGroupOfNames][taskTypeUpdate][i].data.(groupOfNames).dn]["del"] {

					red("     -- dn:     %s\n", changedMembers[taskList[objectTypeGroupOfNames][taskTypeUpdate][i].data.(groupOfNames).dn]["del"][j])
				}

				// delete from map
				delete(changedMembers, taskList[objectTypeGroupOfNames][taskTypeUpdate][i].data.(groupOfNames).dn)
			}

		}
	}

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

	if len(taskList[objectTypeGroupOfNames][taskTypeDelete]) > 0 {
		fmt.Println()
		for i = range taskList[objectTypeGroupOfNames][taskTypeDelete] {
			red("-- dn: %s\n", *taskList[objectTypeGroupOfNames][taskTypeDelete][i].dn)
		}
	}
}

// prettyPrintSUDOers displayes SUDOer diffs in a nice way
func prettyPrintSUDOers() {
	var (
		i    int
		role sudoRole
	)

	fmt.Printf("\n==> SUDOers <==")

	if len(taskList[objectTypeSudoRole][taskTypeCreate]) > 0 {
		for i = range taskList[objectTypeSudoRole][taskTypeCreate] {
			role = taskList[objectTypeSudoRole][taskTypeCreate][i].data.(sudoRole)

			green("\n++ dn:  %s\n   name:   %s\n",
				role.dn,
				role.CN)

			// now all optional details
			if role.Description != "" {
				green("   description:\t%s\n", role.Description)
			}

			if len(role.SudoUser) > 0 {
				green("   sudo_user:\t\n")

				for i = range role.SudoUser {
					green("     value:\t%s\n", role.SudoUser[i])
				}
			}

			if len(role.SudoHost) > 0 {
				green("   sudo_host:\t\n")

				for i = range role.SudoHost {
					green("     value:\t%s\n", role.SudoHost[i])
				}
			}

			if len(role.SudoCommand) > 0 {
				green("   sudo_command:\t\n")

				for i = range role.SudoCommand {
					green("     value:\t%s\n", role.SudoCommand[i])
				}
			}

			if len(role.SudoOption) > 0 {
				green("   sudo_option:\t\n")

				for i = range role.SudoOption {
					green("     value:\t%s\n", role.SudoOption[i])
				}
			}

			if len(role.SudoRunAsUser) > 0 {
				green("   sudo_run_as_user:\t\n")

				for i = range role.SudoRunAsUser {
					green("     value:\t%s\n", role.SudoRunAsUser[i])
				}
			}

			if len(role.SudoRunAsGroup) > 0 {
				green("   sudo_run_as_group:\t\n")

				for i = range role.SudoRunAsGroup {
					green("     value:\t%s\n", role.SudoRunAsGroup[i])
				}
			}

			if len(role.SudoNotBefore) > 0 {
				green("   sudo_not_before:\t\n")

				for i = range role.SudoNotBefore {
					green("     value:\t%s\n", role.SudoNotBefore[i])
				}
			}

			if len(role.SudoNotAfter) > 0 {
				green("   sudo_not_after:\t\n")

				for i = range role.SudoNotAfter {
					green("     value:\t%s\n", role.SudoNotAfter[i])
				}
			}

			if role.SudoOrder != nil {
				green("   sudo_order:\t%d\n", *role.SudoOrder)
			}
		}
	}

	if len(taskList[objectTypeSudoRole][taskTypeUpdate]) > 0 {
		for i = range taskList[objectTypeSudoRole][taskTypeUpdate] {
			role = taskList[objectTypeSudoRole][taskTypeUpdate][i].data.(sudoRole)

			yellow("\n~~ dn:\t%s\n", role.dn)

			// now all optional details
			if role.Description != "" {
				yellow("   description:\t%s\n", role.Description)
			}

			if len(role.SudoUser) > 0 {
				yellow("   sudo_user:\t\n")

				for i = range role.SudoUser {
					yellow("     value:\t%s\n", role.SudoUser[i])
				}
			}

			if len(role.SudoHost) > 0 {
				yellow("   sudo_host:\t\n")

				for i = range role.SudoHost {
					yellow("     value:\t%s\n", role.SudoHost[i])
				}
			}

			if len(role.SudoCommand) > 0 {
				yellow("   sudo_command:\t\n")

				for i = range role.SudoCommand {
					yellow("     value:\t%s\n", role.SudoCommand[i])
				}
			}

			if len(role.SudoOption) > 0 {
				yellow("   sudo_option:\t\n")

				for i = range role.SudoOption {
					yellow("     value:\t%s\n", role.SudoOption[i])
				}
			}

			if len(role.SudoRunAsUser) > 0 {
				yellow("   sudo_run_as_user:\t\n")

				for i = range role.SudoRunAsUser {
					yellow("     value:\t%s\n", role.SudoRunAsUser[i])
				}
			}

			if len(role.SudoRunAsGroup) > 0 {
				yellow("   sudo_run_as_group:\t\n")

				for i = range role.SudoRunAsGroup {
					yellow("     value:\t%s\n", role.SudoRunAsGroup[i])
				}
			}

			if len(role.SudoNotBefore) > 0 {
				yellow("   sudo_not_before:\t\n")

				for i = range role.SudoNotBefore {
					yellow("     value:\t%s\n", role.SudoNotBefore[i])
				}
			}

			if len(role.SudoNotAfter) > 0 {
				yellow("   sudo_not_after:\t\n")

				for i = range role.SudoNotAfter {
					yellow("     value:\t%s\n", role.SudoNotAfter[i])
				}
			}

			if role.SudoOrder != nil {
				yellow("   sudo_order:\t%d\n", *role.SudoOrder)
			}

		}
	}

	if len(taskList[objectTypeSudoRole][taskTypeDelete]) > 0 {
		fmt.Println()
		for i = range taskList[objectTypeSudoRole][taskTypeDelete] {
			red("-- dn: %s\n", *taskList[objectTypeSudoRole][taskTypeDelete][i].dn)
		}
	}

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
			ldapPeople[dn].dn,
			ldapPeople[dn].CN,
			ldapPeople[dn].Description,
			*ldapPeople[dn].GIDNumber)

		w.Flush()

		for index = range localPeople[dn].Objects {

			//fmt.Fprintf(w, "\ndn:\t%s\nusername:\t%s\ngiven_name:\t%s\nsurname:\t%s\ndisplay_name:\t%s\nuid_number:\t%d\ngid_number:\t%d\nlogin_shell:\t%s\nmail:\t%s\nhome_dir:\t%s\n",
			fmt.Fprintf(w, "\n  dn:\t%s\n  username:\t%s\n  given_name:\t%s\n  surname:\t%s\n  display_name:\t%s\n  uid_number:\t%d\n  gid_number:\t%d\n  login_shell:\t%s\n  mail:\t%s\n  home_dir:\t%s\n",
				ldapPeople[dn].Objects[index].dn,
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
