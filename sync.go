package main

import (
	"sort"

	"github.com/kpango/glg"
)

// syncChanges performs all changes stages in taskList
// sync order:
// 1. create OUs
// 2. delete group memberships
// 3. delete posixAccounts
// 4. delete posixGroups
// 5. create posixGroups
// 6. update posixGroups
// 7. create posixAccounts
// 8. update posixAccounts
// 9. create groupOfNames
// 10. update groupOfNames
// 11. create group memberships
// 12. create sudoRole
// 13. update sudoRole
// 14. delete sudoRole
// 15. delete groupOfNames
// 16. delete OUs
func syncChanges() error {
	var (
		err error
		t   *actionTask
	)

	glg.Infof("Data comparison complete. %d changes will be synced", taskListCount)

	// 1. create OUs
	glg.Infof("creating new OrganisationalUnits")
	for _, t = range taskList[objectTypeOrganisationalUnit][taskTypeCreate] {
		if err = ldapCreateOrganisationalUnit(t.data.(*organizationalUnit)); err != nil {
			return err
		}
	}

	// 2. delete group memberships
	glg.Infof("deleting obsolete groupOfNames memberships")
	for _, t = range taskList[objectTypeGroupOfNames][taskTypeDeleteMember] {
		if err = ldapDeleteGroupOfNamesMember(*t.dn,
			t.data.(string)); err != nil {
			return err
		}
	}

	// 3. delete posixAccoounts
	glg.Infof("deleting obsolete posixAccount objects")
	for _, t = range taskList[objectTypePosixAccount][taskTypeDelete] {
		if err = ldapDeletePosixAccount(*t.dn); err != nil {
			return err
		}
	}

	// 4. create posixGroups
	glg.Infof("creating new posixGroup objects")
	for _, t = range taskList[objectTypePosixGroup][taskTypeCreate] {
		if err = ldapCreatePosixGroup(t.data.(posixGroup)); err != nil {
			return err
		}
	}

	// 5. create posixGroups
	glg.Infof("deleting obsolete posixGroup objects")
	for _, t = range taskList[objectTypePosixGroup][taskTypeDelete] {
		if err = ldapDeleteObject(*t.dn); err != nil {
			return nil
		}
	}

	// 6. update posixGroups
	glg.Infof("updating posixGroup objects")
	for _, t = range taskList[objectTypePosixGroup][taskTypeUpdate] {
		if err = ldapUpdatePosixGroup(t.data.(posixGroup)); err != nil {
			return nil
		}
	}

	// 7. create posixAccounts
	glg.Infof("creating new posixAccount objects")
	for _, t = range taskList[objectTypePosixAccount][taskTypeCreate] {
		if err = ldapCreatePosixAccount(t.data.(posixAccount)); err != nil {
			return err
		}
	}

	// 8. update posixAccounts
	glg.Infof("updating posixAccount objects")
	for _, t = range taskList[objectTypePosixAccount][taskTypeUpdate] {
		if err = ldapUpdatePosixAccount(t.data.(posixAccount)); err != nil {
			return err
		}
	}

	// 9. create groupOfNames
	glg.Infof("creating new groupOfNames objects")
	for _, t = range taskList[objectTypeGroupOfNames][taskTypeCreate] {
		if err = ldapCreateGroupOfNames(t.data.(groupOfNames)); err != nil {
			return err
		}
	}

	// 10. update groupOfNames
	glg.Infof("updating groupOfNames objects")
	for _, t = range taskList[objectTypeGroupOfNames][taskTypeUpdate] {
		if err = ldapUpdateGroupOfNames(t.data.(groupOfNames)); err != nil {
			return err
		}
	}

	// 11. create group memberships
	glg.Infof("creating new groupOfNames memberships")
	for _, t = range taskList[objectTypeGroupOfNames][taskTypeAddMember] {
		if err = ldapAddGroupOfNamesMember(*t.dn, t.data.(string)); err != nil {
			return err
		}
	}

	// 12. create sudoRole
	glg.Infof("creating sudoRole objects")
	for _, t = range taskList[objectTypeSudoRole][taskTypeCreate] {
		if err = ldapCreateSudoRole(t.data.(sudoRole)); err != nil {
			return err
		}
	}

	// 13. update sudoRole
	glg.Infof("updating sudoRole objects")
	for _, t = range taskList[objectTypeSudoRole][taskTypeUpdate] {
		if err = ldapUpdateSudoRole(t.data.(sudoRole)); err != nil {
			return err
		}
	}

	// 14. delete sudoRole
	glg.Infof("deleting sudoRole objects")
	for _, t = range taskList[objectTypeSudoRole][taskTypeDelete] {
		if err = ldapDeleteObject(*t.dn); err != nil {
			return err
		}
	}

	// 15. delete groupOfNames
	glg.Infof("deleting groupOfNames objects")
	for _, t = range taskList[objectTypeGroupOfNames][taskTypeDelete] {
		if err = ldapDeleteObject(*t.dn); err != nil {
			return err
		}
	}

	// 16. delete OUs
	glg.Infof("deleting intermediate organizationalUnit objects")

	// re-sort tasks to have longer DNs be deleted first
	sort.Slice(taskList[objectTypeOrganisationalUnit][taskTypeDelete], func(i, j int) bool {
		return len(*taskList[objectTypeOrganisationalUnit][taskTypeDelete][i].dn) > len(*taskList[objectTypeOrganisationalUnit][taskTypeDelete][j].dn)
	})

	for _, t = range taskList[objectTypeOrganisationalUnit][taskTypeDelete] {
		if err = ldapDeleteObject(*t.dn); err != nil {
			return err
		}
	}

	glg.Info("Sync completed.")
	return nil
}
