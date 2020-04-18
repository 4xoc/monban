package main

import (
	"fmt"
	"reflect"
	"sort"

	"github.com/4xoc/monban/models"

	"github.com/kpango/glg"
)

// compareOUs checks for differences between localOUs and ldapOUs and creates tasks to sync LDAP target
func compareOUs() error {
	var (
		i     int
		j     int
		match bool
		task  *models.ActionTask
	)

	// sort both slices before comparing them
	// sorting is done by length of DN to make sure parent DNs (which are shorter, duh) come before children
	sort.Slice(localOUs, func(i, j int) bool {
		return len(localOUs[i].DN) < len(localOUs[j].DN)
	})

	sort.Slice(ldapOUs, func(i, j int) bool {
		return len(ldapOUs[i].DN) < len(ldapOUs[j].DN)
	})

	// first checking for OUs missing on LDAP
	for i = range localOUs {
		match = false
		for j = range ldapOUs {

			if localOUs[i].DN == ldapOUs[j].DN {
				match = true
				break
			}
		}

		if !match {
			glg.Debugf("marked intermediate OU for creation %s", localOUs[i].DN)
			task = new(models.ActionTask)
			task.Data = localOUs[i]
			taskList[models.ObjectTypeOrganisationalUnit][models.TaskTypeCreate] = append(taskList[models.ObjectTypeOrganisationalUnit][models.TaskTypeCreate], task)
			taskListCount++
		}
	}

	// next check for OUs existing extra in LDAP and mark them for deletion
	for i = range ldapOUs {
		match = false
		for j = range localOUs {

			if ldapOUs[i].DN == localOUs[j].DN {
				match = true
				break
			}
		}

		if !match {
			glg.Debugf("marked intermediate OU for deletion %s", ldapOUs[i].DN)
			task = new(models.ActionTask)
			task.DN = &ldapOUs[i].DN
			taskList[models.ObjectTypeOrganisationalUnit][models.TaskTypeDelete] = append(taskList[models.ObjectTypeOrganisationalUnit][models.TaskTypeDelete], task)
			taskListCount++
		}
	}

	return nil
}

// comparePosixGroups compares local and ldap posixGroups
func comparePosixGroups() error {
	var (
		dn             string
		userIndex      int
		ldapUserIndex  int
		foundUser      bool
		task           *models.ActionTask
		group          *models.PosixGroup
		ok             bool
		err            error
		groupIsMissing bool
		mismatch       bool
	)

	glg.Info("comparing posixGroups")

	// check with users groups should exist
	for dn = range localPeople {
		// reset
		groupIsMissing = false
		mismatch = false

		// check if group already exists in LDAP
		if _, ok = ldapPeople[dn]; !ok {
			groupIsMissing = true

			glg.Debugf("marked posixGroup for creation %s", dn)

			// add task to create group
			task = new(models.ActionTask)
			task.Data = localPeople[dn]
			taskList[models.ObjectTypePosixGroup][models.TaskTypeCreate] = append(taskList[models.ObjectTypePosixGroup][models.TaskTypeCreate], task)
			taskListCount++
		}

		// check for localPeopleGroup diff
		// only do so when group exists
		if !groupIsMissing {
			// reset tmpPeople in case it was previously used above
			group = new(models.PosixGroup)
			group.DN = dn

			// gid_number can change
			if *localPeople[dn].GIDNumber != *ldapPeople[dn].GIDNumber {
				mismatch = true
				group.GIDNumber = localPeople[dn].GIDNumber
			}

			// description can change
			if localPeople[dn].Description != ldapPeople[dn].Description {
				mismatch = true
				group.Description = localPeople[dn].Description
			}

			if mismatch {
				glg.Debugf("marked posixGroup for update %s", dn)

				// add task to update group
				task = new(models.ActionTask)
				task.Data = *group
				taskList[models.ObjectTypePosixGroup][models.TaskTypeUpdate] = append(taskList[models.ObjectTypePosixGroup][models.TaskTypeUpdate], task)
				taskListCount++
			}
		}

		// check users within group
		for userIndex = range localPeople[dn].Objects {
			foundUser = false

			// only check users when group exists; if it is missing, foundUser must be false so users get added to the group
			// that will be created in the same sync cycle
			if !groupIsMissing {
				for ldapUserIndex = range ldapPeople[dn].Objects {
					if *localPeople[dn].Objects[userIndex].UID == *ldapPeople[dn].Objects[ldapUserIndex].UID {
						// user exists in LDAP but might need update
						foundUser = true
						err = comparePosixAccount(&localPeople[dn].Objects[userIndex], &ldapPeople[dn].Objects[ldapUserIndex])
						if err != nil {
							return err
						}
					}
				}
			}

			if !foundUser {
				glg.Debugf("marked posixAccount for creation %s", localPeople[dn].Objects[userIndex].DN)

				// create new task
				task = new(models.ActionTask)
				task.Data = localPeople[dn].Objects[userIndex]
				taskList[models.ObjectTypePosixAccount][models.TaskTypeCreate] = append(taskList[models.ObjectTypePosixAccount][models.TaskTypeCreate], task)
				taskListCount++
			}
		}
	}

	// go through all user objects in LDAP and find objects that only exist in LDAP and therefore need to be deleted
	for dn = range ldapPeople {

		// check if group exists only in LDAP and needs to be deleted
		if _, ok = localPeople[dn]; !ok {
			glg.Debugf("marked posixGroup for deletion %s", dn)

			task = new(models.ActionTask)
			task.DN = new(string)
			*task.DN = dn
			taskList[models.ObjectTypePosixGroup][models.TaskTypeDelete] = append(taskList[models.ObjectTypePosixGroup][models.TaskTypeDelete], task)
			taskListCount++
		}

		for ldapUserIndex = range ldapPeople[dn].Objects {
			foundUser = false
			for userIndex = range localPeople[dn].Objects {
				if *ldapPeople[dn].Objects[ldapUserIndex].UID == *localPeople[dn].Objects[userIndex].UID {
					foundUser = true
				}
			}

			if !foundUser {
				glg.Debugf("marked posixAccount for deletion %s", ldapPeople[dn].Objects[ldapUserIndex].DN)

				task = new(models.ActionTask)
				task.DN = &ldapPeople[dn].Objects[ldapUserIndex].DN
				taskList[models.ObjectTypePosixAccount][models.TaskTypeDelete] = append(taskList[models.ObjectTypePosixAccount][models.TaskTypeDelete], task)
				taskListCount++
			}
		}
	}

	glg.Info("finished comparing posixGroups")

	return nil
}

// comparePosixAccount compares two models.PosixAccount structs and create a new task to update the LDAP object to match local
// local and remote must have the same UID as otherwise the comparison makes no sense
// local must always be the rt.Config file user while remote is the read data from LDAP
func comparePosixAccount(local *models.PosixAccount, remote *models.PosixAccount) error {
	var (
		task *models.ActionTask
		// userDiff contains only those values that need to be changed and their new values
		userDiff *models.PosixAccount
		mismatch bool
	)

	if *local.UID != *remote.UID {
		return fmt.Errorf("can't compare user objects when UIDs don't match")
	}

	// init userDiff struct
	userDiff = new(models.PosixAccount)
	userDiff.DN = local.DN

	if *local.GivenName != *remote.GivenName {
		mismatch = true
		userDiff.GivenName = local.GivenName
	}

	if *local.Surname != *remote.Surname {
		mismatch = true
		userDiff.Surname = local.Surname
	}

	if *local.DisplayName != *remote.DisplayName {
		mismatch = true
		userDiff.DisplayName = local.DisplayName
	}

	if *local.LoginShell != *remote.LoginShell {
		mismatch = true
		userDiff.LoginShell = local.LoginShell
	}

	if *local.Mail != *remote.Mail {
		mismatch = true
		userDiff.Mail = local.Mail
	}

	if *rt.Config.EnableSSHPublicKeys {
		// SSHPublicKey can be nil
		switch {

		case local.SSHPublicKey == nil && remote.SSHPublicKey != nil:
			// delete ssh key in LDAP
			// to tell that it is to be deleted, set empty string
			mismatch = true
			userDiff.SSHPublicKey = new(string)

		case local.SSHPublicKey != nil && remote.SSHPublicKey != nil:
			if *local.SSHPublicKey != *remote.SSHPublicKey {
				// update ssh key in LDAP
				mismatch = true
				userDiff.SSHPublicKey = local.SSHPublicKey
			}

		case local.SSHPublicKey != nil && remote.SSHPublicKey == nil:
			// create new ssh key in LDAP
			mismatch = true
			userDiff.SSHPublicKey = local.SSHPublicKey
		}
	}

	if *local.HomeDir != *remote.HomeDir {
		mismatch = true
		userDiff.HomeDir = local.HomeDir
	}

	if *local.UserPassword != *remote.UserPassword {
		mismatch = true
		userDiff.UserPassword = local.UserPassword
	}

	// UIDNumber can be nil in rt.Config
	// in such case we can assume that the ldap UIDNumber should be used further (because generate_uid = true)
	if local.UIDNumber != nil {
		if *local.UIDNumber != *remote.UIDNumber {
			mismatch = true
			userDiff.UIDNumber = local.UIDNumber
		}
	}

	if *local.GIDNumber != *remote.GIDNumber {
		mismatch = true
		userDiff.GIDNumber = local.GIDNumber
	}

	if mismatch {
		glg.Debugf("marked posixAccount for update %s", local.DN)

		task = new(models.ActionTask)
		task.Data = *userDiff
		taskList[models.ObjectTypePosixAccount][models.TaskTypeUpdate] = append(taskList[models.ObjectTypePosixAccount][models.TaskTypeUpdate], task)
		taskListCount++
	}

	return nil
}

// compareGroupOfNames checks for differences between local and ldap groupOfNames
func compareGroupOfNames() error {
	var (
		dn              string
		ok              bool
		index           int
		ldapIndex       int
		match           bool
		task            *models.ActionTask
		dn2             string
		tmpGroupOfNames *models.GroupOfNames
	)

	glg.Info("comparing groupOfNames")

	for dn = range localGroups {

		// check if group already exists in LDAP
		if _, ok = ldapGroups[dn]; !ok {
			glg.Debugf("marked groupOfNames for creation %s", dn)

			// add task to create group
			task = new(models.ActionTask)
			task.Data = localGroups[dn]
			taskList[models.ObjectTypeGroupOfNames][models.TaskTypeCreate] = append(taskList[models.ObjectTypeGroupOfNames][models.TaskTypeCreate], task)
			taskListCount++

			// check for members to be created in this new group
			for index = range localGroups[dn].Members {
				task = new(models.ActionTask)
				task.DN = new(string)
				*task.DN = dn

				// get full user DN
				task.Data = getDNFromUsername(localGroups[dn].Members[index])
				if task.Data.(string) == "" {
					glg.Errorf("unknown username for member %s", localGroups[dn].Members[index])
					continue
				}

				taskList[models.ObjectTypeGroupOfNames][models.TaskTypeAddMember] = append(taskList[models.ObjectTypeGroupOfNames][models.TaskTypeAddMember], task)
				taskListCount++
			}
			continue
		}

		if ldapGroups[dn].Description != localGroups[dn].Description {
			glg.Debugf("marked groupOfNames for update %s", dn)

			// add task to update group
			tmpGroupOfNames = new(models.GroupOfNames)
			tmpGroupOfNames.Description = localGroups[dn].Description
			tmpGroupOfNames.DN = dn

			task = new(models.ActionTask)
			task.Data = *tmpGroupOfNames
			taskList[models.ObjectTypeGroupOfNames][models.TaskTypeUpdate] = append(taskList[models.ObjectTypeGroupOfNames][models.TaskTypeUpdate], task)
			taskListCount++
		}

		for index = range localGroups[dn].Members {
			match = false
			for ldapIndex = range ldapGroups[dn].Members {
				if localGroups[dn].Members[index] == ldapGroups[dn].Members[ldapIndex] {
					match = true
				}
			}

			if !match {
				task = new(models.ActionTask)
				// dn will change but task.DN should not
				task.DN = new(string)
				*task.DN = dn
				task.Data = getDNFromUsername(localGroups[dn].Members[index])

				if task.Data.(string) == "" {
					glg.Errorf("unknown username for member %s", localGroups[dn].Members[index])
					continue
				}

				glg.Debugf("marked member for creation %s", task.Data.(string))
				taskList[models.ObjectTypeGroupOfNames][models.TaskTypeAddMember] = append(taskList[models.ObjectTypeGroupOfNames][models.TaskTypeAddMember], task)
				taskListCount++
			}
		}
	}

	// go through all group objects in LDAP and find objects that only exist in LDAP and therefore need to be deleted
	for dn = range ldapGroups {

		if _, ok = localGroups[dn]; !ok {
			glg.Debugf("marked GroupOfNames for deletion %s", dn)

			task = new(models.ActionTask)
			// dn will change but task.DN should not
			task.DN = new(string)
			*task.DN = dn
			taskList[models.ObjectTypeGroupOfNames][models.TaskTypeDelete] = append(taskList[models.ObjectTypeGroupOfNames][models.TaskTypeDelete], task)
			taskListCount++
			continue
		}

		for ldapIndex = range ldapGroups[dn].Members {

			// ignore dummy member
			if ldapGroups[dn].Members[ldapIndex] == "uid=MonbanDummyMember" {
				continue
			}

			match = false
			for index = range localGroups[dn].Members {
				if ldapGroups[dn].Members[ldapIndex] == localGroups[dn].Members[index] {
					match = true
				}
			}

			if !match {
				// user needs to be deleted from group

			out2:
				// get user DN from ldap
				for dn2 = range ldapPeople {
					for index = range ldapPeople[dn2].Objects {
						if ldapGroups[dn].Members[ldapIndex] == *ldapPeople[dn2].Objects[index].UID {
							task = new(models.ActionTask)
							task.DN = new(string)
							*task.DN = dn
							task.Data = ldapPeople[dn2].Objects[index].DN
							// fast leaving of loops
							break out2
						}
					}
				}

				glg.Debugf("marked member for deletion %s", ldapPeople[dn2].Objects[index].DN)
				taskList[models.ObjectTypeGroupOfNames][models.TaskTypeDeleteMember] = append(taskList[models.ObjectTypeGroupOfNames][models.TaskTypeDeleteMember], task)
				taskListCount++
			}
		}
	}

	glg.Info("finished comparing groupOfNames")

	return nil
}

// compareSudoRoles checks for differences between local and ldap models.SudoRoles
func compareSudoRoles() error {
	var (
		i     int
		j     int
		match bool
		task  *models.ActionTask
	)

	glg.Info("comparing sudoGroups")

	// sort both slices before comparing them
	// sorting is done by length of DN to make sure parent DNs (which are shorter, duh) come before children
	sort.Slice(localSudoRoles, func(i, j int) bool {
		return len(localSudoRoles[i].DN) < len(localSudoRoles[j].DN)
	})

	sort.Slice(ldapSudoRoles, func(i, j int) bool {
		return len(ldapSudoRoles[i].DN) < len(ldapSudoRoles[j].DN)
	})

	// first checking for roles missing on LDAP
	for i = range localSudoRoles {
		match = false
		for j = range ldapSudoRoles {

			if localSudoRoles[i].DN == ldapSudoRoles[j].DN {
				// check for differences between both objects
				compareSudoRole(localSudoRoles[i], ldapSudoRoles[j])

				match = true
				break
			}
		}

		if !match {
			glg.Debugf("marked sudoRole for creation %s", localSudoRoles[i].DN)
			task = new(models.ActionTask)
			task.Data = localSudoRoles[i]
			taskList[models.ObjectTypeSudoRole][models.TaskTypeCreate] = append(taskList[models.ObjectTypeSudoRole][models.TaskTypeCreate], task)
			taskListCount++
		}
	}

	// next check for roles existing extra in LDAP and mark them for deletion
	for i = range ldapSudoRoles {
		match = false
		for j = range localSudoRoles {

			if ldapSudoRoles[i].DN == localSudoRoles[j].DN {
				match = true
				break
			}
		}

		if !match {
			glg.Debugf("marked sudoRole for deletion %s", ldapSudoRoles[i].DN)
			task = new(models.ActionTask)
			task.DN = &ldapSudoRoles[i].DN
			taskList[models.ObjectTypeSudoRole][models.TaskTypeDelete] = append(taskList[models.ObjectTypeSudoRole][models.TaskTypeDelete], task)
			taskListCount++
		}
	}

	glg.Info("finished comparing sudoGroups")
	return nil
}

// compareSudoRole checks two models.SudoRole objects for differences
func compareSudoRole(local models.SudoRole, remote models.SudoRole) {
	var (
		task     models.ActionTask
		mismatch bool
	)

	// sort all slices
	sort.Strings(local.SudoUser)
	sort.Strings(local.SudoHost)
	sort.Strings(local.SudoCommand)
	sort.Strings(local.SudoOption)
	sort.Strings(local.SudoRunAsUser)
	sort.Strings(local.SudoRunAsGroup)
	sort.Strings(local.SudoNotBefore)
	sort.Strings(local.SudoNotAfter)
	sort.Strings(remote.SudoUser)
	sort.Strings(remote.SudoHost)
	sort.Strings(remote.SudoCommand)
	sort.Strings(remote.SudoOption)
	sort.Strings(remote.SudoRunAsUser)
	sort.Strings(remote.SudoRunAsGroup)
	sort.Strings(remote.SudoNotBefore)
	sort.Strings(remote.SudoNotAfter)

	// Things here are different than with other objects as here values can be deleted in rt.Config that also need to be
	// deleted on target. Thererfore, when a diff is detected, the local version gets added as task data which in turn
	// allows ldapUpdateSudoRole to replace or delete attributes

	if local.Description != remote.Description ||
		!reflect.DeepEqual(local.SudoUser, remote.SudoUser) ||
		!reflect.DeepEqual(local.SudoHost, remote.SudoHost) ||
		!reflect.DeepEqual(local.SudoCommand, remote.SudoCommand) ||
		!reflect.DeepEqual(local.SudoOption, remote.SudoOption) ||
		!reflect.DeepEqual(local.SudoRunAsUser, remote.SudoRunAsUser) ||
		!reflect.DeepEqual(local.SudoRunAsGroup, remote.SudoRunAsGroup) ||
		!reflect.DeepEqual(local.SudoNotBefore, remote.SudoNotBefore) ||
		!reflect.DeepEqual(local.SudoNotAfter, remote.SudoNotAfter) {
		mismatch = true
	}

	// sudoOrder is a ptr so we need to check first if it is not nill
	if local.SudoOrder != nil && remote.SudoOrder != nil {
		if *local.SudoOrder != *remote.SudoOrder {
			mismatch = true
		}
	} else {
		// if one of those is nil, there is a change
		if !(local.SudoOrder == nil && remote.SudoOrder == nil) {
			mismatch = true
		}
	}

	if mismatch {
		// create update task
		task.Data = local
		taskList[models.ObjectTypeSudoRole][models.TaskTypeUpdate] = append(taskList[models.ObjectTypeSudoRole][models.TaskTypeUpdate], &task)
		taskListCount++

		glg.Debugf("marked sudoRole for update %s", local.DN)
	}
}

// getDNFromUsername returns the full DN of a user object identified by its username
func getDNFromUsername(uid string) string {
	var (
		pgDN string
		i    int
		dn   string
	)

	for pgDN = range localPeople {
		for i = range localPeople[pgDN].Objects {
			if *localPeople[pgDN].Objects[i].UID == uid {
				return localPeople[pgDN].Objects[i].DN
			}
		}
	}

	return dn
}
