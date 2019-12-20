package main

import (
	"fmt"
	"sort"

	"github.com/kpango/glg"
)

// compareOUs checks for differences between localOUs and ldapOUs and creates tasks to sync LDAP target
func compareOUs() error {
	var (
		i     int
		j     int
		match bool
		task  *actionTask
	)

	// sort both maps before comparing them
	// sorting is done by length of DN to make sure parent DNs (which are shorter, duh) come before children
	sort.Slice(localOUs, func(i, j int) bool {
		return len(localOUs[i].dn) < len(localOUs[j].dn)
	})

	sort.Slice(ldapOUs, func(i, j int) bool {
		return len(ldapOUs[i].dn) < len(ldapOUs[j].dn)
	})

	// first checking for OUs missing on LDAP
	for i = range localOUs {
		match = false
		for j = range ldapOUs {

			if localOUs[i].dn == ldapOUs[j].dn {
				match = true
				break
			}
		}

		if !match {
			glg.Debugf("marked intermediate OU for creation %s", localOUs[i].dn)
			task = new(actionTask)
			task.objectType = objectTypeOrganisationalUnit
			task.taskType = taskTypeCreate
			task.data = localOUs[i]

			taskList = append(taskList, task)
		}
	}

	// next check for OUs existing extra in LDAP and mark them for deletion
	for i = range ldapOUs {
		match = false
		for j = range localOUs {

			if ldapOUs[i].dn == localOUs[j].dn {
				match = true
				break
			}
		}

		if !match {
			glg.Debugf("marked intermediate OU for deletion %s", ldapOUs[i].dn)
			task = new(actionTask)
			task.objectType = objectTypeOrganisationalUnit
			task.taskType = taskTypeDelete
			task.dn = ldapOUs[i].dn

			taskList = append(taskList, task)
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
		task           *actionTask
		group          *posixGroup
		ok             bool
		err            error
		groupIsMissing bool
		missmatch      bool
	)

	glg.Info("comparing posixGroups")

	// check with users groups should exist
	for dn = range localPeople {
		// reset
		groupIsMissing = false
		missmatch = false

		// check if group already exists in LDAP
		if _, ok = ldapPeople[dn]; !ok {
			groupIsMissing = true

			glg.Debugf("marked posixGroup for creation %s", dn)

			// add task to create group
			task = new(actionTask)
			task.dn = dn
			task.objectType = objectTypePosixGroup
			task.taskType = taskTypeCreate
			task.data = localPeople[dn]
			taskList = append(taskList, task)
		}

		// check for localPeopleGroup diff
		// only do so when group exists
		if !groupIsMissing {
			// reset tmpPeople in case it was previously used above
			group = new(posixGroup)
			group.dn = dn

			// gid_number can change
			if *localPeople[dn].GIDNumber != *ldapPeople[dn].GIDNumber {
				missmatch = true
				group.GIDNumber = localPeople[dn].GIDNumber
			}

			// description can change
			if localPeople[dn].Description != ldapPeople[dn].Description {
				missmatch = true
				group.Description = localPeople[dn].Description
			}

			if missmatch {
				glg.Debugf("marked posixGroup for update %s", dn)

				// add task to update group
				task = new(actionTask)
				task.dn = dn
				task.objectType = objectTypePosixGroup
				task.taskType = taskTypeUpdate
				task.data = group
				taskList = append(taskList, task)
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
				glg.Debugf("marked posixAccount for creation %s", localPeople[dn].Objects[userIndex].dn)

				// create new task
				task = new(actionTask)
				task.dn = localPeople[dn].Objects[userIndex].dn
				task.objectType = objectTypePosixAccount
				task.taskType = taskTypeCreate
				task.data = &localPeople[dn].Objects[userIndex]
				taskList = append(taskList, task)
			}
		}
	}

	// go through all user objects in LDAP and find objects that only exist in LDAP and therefore need to be deleted
	for dn = range ldapPeople {

		// check if group exists only in LDAP and needs to be deleted
		if _, ok = localPeople[dn]; !ok {
			glg.Debugf("marked posixGroup for deletion %s", dn)

			task = new(actionTask)
			task.dn = dn
			task.objectType = objectTypePosixGroup
			task.taskType = taskTypeDelete
			taskList = append(taskList, task)
		}

		for ldapUserIndex = range ldapPeople[dn].Objects {
			foundUser = false
			for userIndex = range localPeople[dn].Objects {
				if *ldapPeople[dn].Objects[ldapUserIndex].UID == *localPeople[dn].Objects[userIndex].UID {
					foundUser = true
				}
			}

			if !foundUser {
				glg.Debugf("marked posixAccount for deletion %s", ldapPeople[dn].Objects[ldapUserIndex].dn)

				task = new(actionTask)
				task.dn = ldapPeople[dn].Objects[ldapUserIndex].dn
				task.data = &ldapPeople[dn].Objects[ldapUserIndex]
				task.objectType = objectTypePosixAccount
				task.taskType = taskTypeDelete
				taskList = append(taskList, task)
			}
		}
	}

	glg.Info("finished comparing posixGroups")

	return nil
}

// comparePosixAccount compares two posixAccount structs and create a new task to update the LDAP object to match local
// local and remote must have the same UID as otherwise the comparison makes no sense
// local must always be the config file user while remote is the read data from LDAP
// GIDNumber & UIDNumber is not checked because it is always derived from LDAP
func comparePosixAccount(local *posixAccount, remote *posixAccount) error {
	var (
		task *actionTask
		// userDiff contains only those values that need to be changed and their new values
		userDiff *posixAccount
		mismatch bool
	)

	if *local.UID != *remote.UID {
		return fmt.Errorf("can't compare user objects when UIDs don't match")
	}

	// init userDiff struct
	userDiff = new(posixAccount)
	userDiff.dn = local.dn

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

	if *config.EnableSSHPublicKeys {
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

	if mismatch {
		glg.Debugf("marked posixAccount for update %s", local.dn)

		task = new(actionTask)
		task.dn = local.dn
		task.objectType = objectTypePosixAccount
		task.taskType = taskTypeUpdate
		task.data = userDiff
		taskList = append(taskList, task)
	}

	return nil
}

// compareGroupOfNames checks for differences between local and ldap groupOfNames
func compareGroupOfNames() error {
	var (
		dn        string
		ok        bool
		index     int
		ldapIndex int
		match     bool
		task      *actionTask
		dn2       string
		index2    int
	)

	glg.Info("comparing groupOfNames")

	for dn = range localGroups {

		// check if group already exists in LDAP
		if _, ok = ldapGroups[dn]; !ok {
			glg.Debugf("marked groupOfNames for creation %s", dn)

			// add task to create group
			task = new(actionTask)
			task.dn = dn
			task.objectType = objectTypeGroupOfNames
			task.taskType = taskTypeCreate
			task.data = localGroups[dn]
			taskList = append(taskList, task)
		}

		// TODO: check for missing value
		if ldapGroups[dn].Description != localGroups[dn].Description {
			glg.Debugf("marked groupOfNames for update %s", dn)

			// add task to update group
			task = new(actionTask)
			task.dn = dn
			task.objectType = objectTypeGroupOfNames
			task.taskType = taskTypeUpdate
			task.data = new(groupOfNames)
			task.data.(*groupOfNames).Description = localGroups[dn].Description
			task.data.(*groupOfNames).dn = dn
			taskList = append(taskList, task)
		}

		for index = range localGroups[dn].Members {
			match = false
			for ldapIndex = range ldapGroups[dn].Members {
				if localGroups[dn].Members[index] == ldapGroups[dn].Members[ldapIndex] {
					match = true
				}
			}

			if !match {
			out1:
				// get user DN
				for dn2 = range localPeople {
					for index2 = range localPeople[dn2].Objects {
						if *localPeople[dn2].Objects[index2].UID == localGroups[dn].Members[index] {
							task = new(actionTask)
							task.data = localPeople[dn2].Objects[index2].dn
							// fast leaving of loops
							break out1
						}
					}
				}

				glg.Debugf("marked member for creation %s", localPeople[dn2].Objects[index2].dn)

				task.dn = dn
				task.objectType = objectTypeGroupOfNames
				task.taskType = taskTypeAddMember
				taskList = append(taskList, task)
			}
		}
	}

	// go through all group objects in LDAP and find objects that only exist in LDAP and therefore need to be deleted
	for dn = range ldapGroups {

		if _, ok = localGroups[dn]; !ok {
			glg.Debugf("marked GroupOfNames for deletion %s", dn)

			task = new(actionTask)
			task.dn = dn
			task.objectType = objectTypeGroupOfNames
			task.taskType = taskTypeDelete
			taskList = append(taskList, task)
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
							task = new(actionTask)
							task.data = ldapPeople[dn2].Objects[index].dn
							// fast leaving of loops
							break out2
						}
					}
				}

				glg.Debugf("marked member for deletion %s", ldapPeople[dn2].Objects[index].dn)

				task.dn = dn
				task.objectType = objectTypeGroupOfNames
				task.taskType = taskTypeDeleteMember
				taskList = append(taskList, task)
			}
		}
	}

	glg.Info("finished comparing groupOfNames")

	return nil
}
