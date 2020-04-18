package main

import (
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/4xoc/monban/models"

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
	var (
		i int
	)

	fmt.Printf("\n==> OrganizationalUnits <==\n")

	if len(taskList[models.ObjectTypeOrganisationalUnit][models.TaskTypeCreate]) > 0 {
		for i = range taskList[models.ObjectTypeOrganisationalUnit][models.TaskTypeCreate] {
			green("++ dn: %s\n",
				taskList[models.ObjectTypeOrganisationalUnit][models.TaskTypeCreate][i].Data.(*models.OrganizationalUnit).DN)
		}
	}

	if len(taskList[models.ObjectTypeOrganisationalUnit][models.TaskTypeDelete]) > 0 {
		fmt.Println()
		for i = range taskList[models.ObjectTypeOrganisationalUnit][models.TaskTypeDelete] {
			red("-- dn: %s\n", *taskList[models.ObjectTypeOrganisationalUnit][models.TaskTypeDelete][i].DN)
		}
	}
}

// prettyPrintPosixGroups displays models.PosixGroup changes in a pretty way
func prettyPrintPosixGroups() {
	var (
		i int
	)

	fmt.Printf("\n==> PosixGroups <==")

	if len(taskList[models.ObjectTypePosixGroup][models.TaskTypeCreate]) > 0 {
		for i = range taskList[models.ObjectTypePosixGroup][models.TaskTypeCreate] {
			green("\n++ dn:          %s\n   name:        %s\n   gid_number:  %d\n   description: %s\n",
				taskList[models.ObjectTypePosixGroup][models.TaskTypeCreate][i].Data.(models.PosixGroup).DN,
				taskList[models.ObjectTypePosixGroup][models.TaskTypeCreate][i].Data.(models.PosixGroup).CN,
				*taskList[models.ObjectTypePosixGroup][models.TaskTypeCreate][i].Data.(models.PosixGroup).GIDNumber,
				taskList[models.ObjectTypePosixGroup][models.TaskTypeCreate][i].Data.(models.PosixGroup).Description)
		}
	}

	if len(taskList[models.ObjectTypePosixGroup][models.TaskTypeUpdate]) > 0 {
		for i = range taskList[models.ObjectTypePosixGroup][models.TaskTypeUpdate] {
			yellow("\n~~ dn:          %s\n", taskList[models.ObjectTypePosixGroup][models.TaskTypeUpdate][i].Data.(models.PosixGroup).DN)

			if taskList[models.ObjectTypePosixGroup][models.TaskTypeUpdate][i].Data.(models.PosixGroup).GIDNumber != nil {
				yellow("   gid_number:  %d\n",
					*taskList[models.ObjectTypePosixGroup][models.TaskTypeUpdate][i].Data.(models.PosixGroup).GIDNumber)
			}

			if taskList[models.ObjectTypePosixGroup][models.TaskTypeUpdate][i].Data.(models.PosixGroup).Description != "" {
				yellow("   description:  %s\n",
					taskList[models.ObjectTypePosixGroup][models.TaskTypeUpdate][i].Data.(models.PosixGroup).Description)
			}
		}
	}

	if len(taskList[models.ObjectTypePosixGroup][models.TaskTypeDelete]) > 0 {
		fmt.Println()
		for i = range taskList[models.ObjectTypePosixGroup][models.TaskTypeDelete] {
			red("-- dn: %s\n", *taskList[models.ObjectTypePosixGroup][models.TaskTypeDelete][i].DN)
		}
	}
}

// prettyPrintPosixAccounts displays models.PosixAccount changes in a pretty way
func prettyPrintPosixAccounts() {
	var (
		i int
	)

	fmt.Printf("\n==> PosixAccounts <==")

	if len(taskList[models.ObjectTypePosixGroup][models.TaskTypeCreate]) > 0 {
		for i = range taskList[models.ObjectTypePosixGroup][models.TaskTypeCreate] {

			green("\n++ dn:             %s\n   username:       %s\n   given_name:     %s\n   surname:        %s\n   display_name:   %s\n",
				taskList[models.ObjectTypePosixAccount][models.TaskTypeCreate][i].Data.(models.PosixAccount).DN,
				*taskList[models.ObjectTypePosixAccount][models.TaskTypeCreate][i].Data.(models.PosixAccount).UID,
				*taskList[models.ObjectTypePosixAccount][models.TaskTypeCreate][i].Data.(models.PosixAccount).GivenName,
				*taskList[models.ObjectTypePosixAccount][models.TaskTypeCreate][i].Data.(models.PosixAccount).Surname,
				*taskList[models.ObjectTypePosixAccount][models.TaskTypeCreate][i].Data.(models.PosixAccount).DisplayName)

			if taskList[models.ObjectTypePosixAccount][models.TaskTypeCreate][i].Data.(models.PosixAccount).UIDNumber == nil {
				green("   uid_number:     *known after sync*\n")
			} else {
				green("   uid_number:     %d\n",
					*taskList[models.ObjectTypePosixAccount][models.TaskTypeCreate][i].Data.(models.PosixAccount).UIDNumber)
			}

			green("   gid_number:     %d\n   login_shell:    %s\n   mail:           %s\n   home_dir:       %s\n",
				*taskList[models.ObjectTypePosixAccount][models.TaskTypeCreate][i].Data.(models.PosixAccount).GIDNumber,
				*taskList[models.ObjectTypePosixAccount][models.TaskTypeCreate][i].Data.(models.PosixAccount).LoginShell,
				*taskList[models.ObjectTypePosixAccount][models.TaskTypeCreate][i].Data.(models.PosixAccount).Mail,
				*taskList[models.ObjectTypePosixAccount][models.TaskTypeCreate][i].Data.(models.PosixAccount).HomeDir)

			// check for additional, optional information
			if *rt.Config.EnableSSHPublicKeys {
				if taskList[models.ObjectTypePosixAccount][models.TaskTypeCreate][i].Data.(models.PosixAccount).SSHPublicKey == nil {
					green("   ssh_public_key: *none*\n")

				} else {
					green("   ssh_public_key: %s\n",
						*taskList[models.ObjectTypePosixAccount][models.TaskTypeCreate][i].Data.(models.PosixAccount).SSHPublicKey)
				}
			}

			if taskList[models.ObjectTypePosixAccount][models.TaskTypeCreate][i].Data.(models.PosixAccount).UserPassword != nil {
				green("   user_password:  %s\n",
					*taskList[models.ObjectTypePosixAccount][models.TaskTypeCreate][i].Data.(models.PosixAccount).UserPassword)
			}
		}
	}

	if len(taskList[models.ObjectTypePosixAccount][models.TaskTypeUpdate]) > 0 {
		for i = range taskList[models.ObjectTypePosixAccount][models.TaskTypeUpdate] {
			yellow("\n~~ dn:             %s\n", taskList[models.ObjectTypePosixAccount][models.TaskTypeUpdate][i].Data.(models.PosixAccount).DN)

			// changing username will result in a new object therefore not checking for a change here

			if taskList[models.ObjectTypePosixAccount][models.TaskTypeUpdate][i].Data.(models.PosixAccount).GivenName != nil {
				yellow("   given_name:     %s\n",
					*taskList[models.ObjectTypePosixAccount][models.TaskTypeUpdate][i].Data.(models.PosixAccount).GivenName)
			}

			if taskList[models.ObjectTypePosixAccount][models.TaskTypeUpdate][i].Data.(models.PosixAccount).Surname != nil {
				yellow("   surname:        %s\n",
					*taskList[models.ObjectTypePosixAccount][models.TaskTypeUpdate][i].Data.(models.PosixAccount).Surname)
			}

			if taskList[models.ObjectTypePosixAccount][models.TaskTypeUpdate][i].Data.(models.PosixAccount).DisplayName != nil {
				yellow("   display_name:    %s\n",
					*taskList[models.ObjectTypePosixAccount][models.TaskTypeUpdate][i].Data.(models.PosixAccount).DisplayName)
			}

			if taskList[models.ObjectTypePosixAccount][models.TaskTypeUpdate][i].Data.(models.PosixAccount).UIDNumber != nil {
				yellow("   uid_number:     %d\n",
					*taskList[models.ObjectTypePosixAccount][models.TaskTypeUpdate][i].Data.(models.PosixAccount).UIDNumber)
			}

			if taskList[models.ObjectTypePosixAccount][models.TaskTypeUpdate][i].Data.(models.PosixAccount).GIDNumber != nil {
				yellow("   gid_number:     %d\n",
					*taskList[models.ObjectTypePosixAccount][models.TaskTypeUpdate][i].Data.(models.PosixAccount).GIDNumber)
			}

			if taskList[models.ObjectTypePosixAccount][models.TaskTypeUpdate][i].Data.(models.PosixAccount).LoginShell != nil {
				yellow("   login_shell:    %s\n",
					*taskList[models.ObjectTypePosixAccount][models.TaskTypeUpdate][i].Data.(models.PosixAccount).LoginShell)
			}

			if taskList[models.ObjectTypePosixAccount][models.TaskTypeUpdate][i].Data.(models.PosixAccount).Mail != nil {
				yellow("   mail:           %s\n",
					*taskList[models.ObjectTypePosixAccount][models.TaskTypeUpdate][i].Data.(models.PosixAccount).Mail)
			}

			if taskList[models.ObjectTypePosixAccount][models.TaskTypeUpdate][i].Data.(models.PosixAccount).HomeDir != nil {
				yellow("   home_dir:       %s\n",
					*taskList[models.ObjectTypePosixAccount][models.TaskTypeUpdate][i].Data.(models.PosixAccount).HomeDir)
			}

			// check for additional, optional information
			if *rt.Config.EnableSSHPublicKeys &&
				taskList[models.ObjectTypePosixAccount][models.TaskTypeUpdate][i].Data.(models.PosixAccount).SSHPublicKey != nil {

				if *taskList[models.ObjectTypePosixAccount][models.TaskTypeUpdate][i].Data.(models.PosixAccount).SSHPublicKey == "" {
					// key is to be deleted
					red("   ssh_public_key: *to be removed*\n")
				} else {
					yellow("   ssh_public_key: %s\n",
						*taskList[models.ObjectTypePosixAccount][models.TaskTypeUpdate][i].Data.(models.PosixAccount).SSHPublicKey)
				}
			}

			if taskList[models.ObjectTypePosixAccount][models.TaskTypeUpdate][i].Data.(models.PosixAccount).UserPassword != nil {
				yellow("   user_password:  %s\n",
					*taskList[models.ObjectTypePosixAccount][models.TaskTypeUpdate][i].Data.(models.PosixAccount).UserPassword)
			}
		}
	}

	if len(taskList[models.ObjectTypePosixAccount][models.TaskTypeDelete]) > 0 {
		fmt.Println()
		for i = range taskList[models.ObjectTypePosixAccount][models.TaskTypeDelete] {
			red("-- dn: %s\n", *taskList[models.ObjectTypePosixAccount][models.TaskTypeDelete][i].DN)
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
	if len(taskList[models.ObjectTypeGroupOfNames][models.TaskTypeAddMember]) > 0 {
		for i = range taskList[models.ObjectTypeGroupOfNames][models.TaskTypeAddMember] {
			// init map for this DN
			if changedMembers[*taskList[models.ObjectTypeGroupOfNames][models.TaskTypeAddMember][i].DN] == nil {
				changedMembers[*taskList[models.ObjectTypeGroupOfNames][models.TaskTypeAddMember][i].DN] = make(map[string][]string)
			}

			changedMembers[*taskList[models.ObjectTypeGroupOfNames][models.TaskTypeAddMember][i].DN]["add"] = append(changedMembers[*taskList[models.ObjectTypeGroupOfNames][models.TaskTypeAddMember][i].DN]["add"], taskList[models.ObjectTypeGroupOfNames][models.TaskTypeAddMember][i].Data.(string))
		}
	}

	if len(taskList[models.ObjectTypeGroupOfNames][models.TaskTypeDeleteMember]) > 0 {
		for i = range taskList[models.ObjectTypeGroupOfNames][models.TaskTypeDeleteMember] {
			if changedMembers[*taskList[models.ObjectTypeGroupOfNames][models.TaskTypeDeleteMember][i].DN] == nil {
				changedMembers[*taskList[models.ObjectTypeGroupOfNames][models.TaskTypeDeleteMember][i].DN] = make(map[string][]string)
			}

			changedMembers[*taskList[models.ObjectTypeGroupOfNames][models.TaskTypeDeleteMember][i].DN]["del"] = append(changedMembers[*taskList[models.ObjectTypeGroupOfNames][models.TaskTypeDeleteMember][i].DN]["del"], taskList[models.ObjectTypeGroupOfNames][models.TaskTypeDeleteMember][i].Data.(string))
		}
	}

	fmt.Printf("\n==> GroupOfNames <==")

	if len(taskList[models.ObjectTypeGroupOfNames][models.TaskTypeCreate]) > 0 {
		for i = range taskList[models.ObjectTypeGroupOfNames][models.TaskTypeCreate] {
			green("\n++ dn:          %s\n   cn:          %s\n   description: %s\n",
				taskList[models.ObjectTypeGroupOfNames][models.TaskTypeCreate][i].Data.(models.GroupOfNames).DN,
				taskList[models.ObjectTypeGroupOfNames][models.TaskTypeCreate][i].Data.(models.GroupOfNames).CN,
				taskList[models.ObjectTypeGroupOfNames][models.TaskTypeCreate][i].Data.(models.GroupOfNames).Description)

			// check for new members of this group
			if _, ok = changedMembers[taskList[models.ObjectTypeGroupOfNames][models.TaskTypeCreate][i].Data.(models.GroupOfNames).DN]; ok {
				green("   members:\n")

				// check for members to add
				for j = range changedMembers[taskList[models.ObjectTypeGroupOfNames][models.TaskTypeCreate][i].Data.(models.GroupOfNames).DN]["add"] {

					green("     ++ dn:     %s\n", changedMembers[taskList[models.ObjectTypeGroupOfNames][models.TaskTypeCreate][i].Data.(models.GroupOfNames).DN]["add"][j])
				}

				// delete from map
				delete(changedMembers, taskList[models.ObjectTypeGroupOfNames][models.TaskTypeCreate][i].Data.(models.GroupOfNames).DN)
			}
		}
	}

	if len(taskList[models.ObjectTypeGroupOfNames][models.TaskTypeUpdate]) > 0 {
		for i = range taskList[models.ObjectTypeGroupOfNames][models.TaskTypeUpdate] {
			yellow("\n~~ dn:          %s\n", taskList[models.ObjectTypeGroupOfNames][models.TaskTypeUpdate][i].Data.(models.GroupOfNames).DN)

			if taskList[models.ObjectTypeGroupOfNames][models.TaskTypeUpdate][i].Data.(models.GroupOfNames).Description != "" {
				yellow("   description: %s\n",
					taskList[models.ObjectTypeGroupOfNames][models.TaskTypeUpdate][i].Data.(models.GroupOfNames).Description)
			}

			// check for member changes in this models.GroupOfNames
			if _, ok = changedMembers[taskList[models.ObjectTypeGroupOfNames][models.TaskTypeUpdate][i].Data.(models.GroupOfNames).DN]; ok {
				yellow("   members:\n")

				// check for members to add
				for j = range changedMembers[taskList[models.ObjectTypeGroupOfNames][models.TaskTypeUpdate][i].Data.(models.GroupOfNames).DN]["add"] {

					green("     ++ dn:     %s\n", changedMembers[taskList[models.ObjectTypeGroupOfNames][models.TaskTypeUpdate][i].Data.(models.GroupOfNames).DN]["add"][j])
				}

				// check for members to delete
				for j = range changedMembers[taskList[models.ObjectTypeGroupOfNames][models.TaskTypeUpdate][i].Data.(models.GroupOfNames).DN]["del"] {

					red("     -- dn:     %s\n", changedMembers[taskList[models.ObjectTypeGroupOfNames][models.TaskTypeUpdate][i].Data.(models.GroupOfNames).DN]["del"][j])
				}

				// delete from map
				delete(changedMembers, taskList[models.ObjectTypeGroupOfNames][models.TaskTypeUpdate][i].Data.(models.GroupOfNames).DN)
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

	if len(taskList[models.ObjectTypeGroupOfNames][models.TaskTypeDelete]) > 0 {
		fmt.Println()
		for i = range taskList[models.ObjectTypeGroupOfNames][models.TaskTypeDelete] {
			red("-- dn: %s\n", *taskList[models.ObjectTypeGroupOfNames][models.TaskTypeDelete][i].DN)
		}
	}
}

// prettyPrintSUDOers displayes SUDOer diffs in a nice way
func prettyPrintSUDOers() {
	var (
		i    int
		role models.SudoRole
	)

	fmt.Printf("\n==> SUDOers <==")

	if len(taskList[models.ObjectTypeSudoRole][models.TaskTypeCreate]) > 0 {
		for i = range taskList[models.ObjectTypeSudoRole][models.TaskTypeCreate] {
			role = taskList[models.ObjectTypeSudoRole][models.TaskTypeCreate][i].Data.(models.SudoRole)

			green("\n++ dn:  %s\n   name:   %s\n",
				role.DN,
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

	if len(taskList[models.ObjectTypeSudoRole][models.TaskTypeUpdate]) > 0 {
		for i = range taskList[models.ObjectTypeSudoRole][models.TaskTypeUpdate] {
			role = taskList[models.ObjectTypeSudoRole][models.TaskTypeUpdate][i].Data.(models.SudoRole)

			yellow("\n~~ dn:\t%s\n", role.DN)

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

	if len(taskList[models.ObjectTypeSudoRole][models.TaskTypeDelete]) > 0 {
		fmt.Println()
		for i = range taskList[models.ObjectTypeSudoRole][models.TaskTypeDelete] {
			red("-- dn: %s\n", *taskList[models.ObjectTypeSudoRole][models.TaskTypeDelete][i].DN)
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
