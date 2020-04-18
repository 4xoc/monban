package main

import (
	"sort"

	"github.com/4xoc/monban/models"

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
		t   *models.ActionTask
	)

	glg.Infof("Data comparison complete. %d changes will be synced", taskListCount)

	// 1. create OUs
	glg.Infof("creating new OrganisationalUnits")
	for _, t = range taskList[models.ObjectTypeOrganisationalUnit][models.TaskTypeCreate] {
		if err = ldapCreateOrganisationalUnit(t.Data.(*models.OrganizationalUnit)); err != nil {
			return err
		}
	}

	// 2. delete group memberships
	glg.Infof("deleting obsolete groupOfNames memberships")
	for _, t = range taskList[models.ObjectTypeGroupOfNames][models.TaskTypeDeleteMember] {
		if err = ldapDeleteGroupOfNamesMember(*t.DN,
			t.Data.(string)); err != nil {
			return err
		}
	}

	// 3. delete posixAccoounts
	glg.Infof("deleting obsolete posixAccount objects")
	for _, t = range taskList[models.ObjectTypePosixAccount][models.TaskTypeDelete] {
		if err = ldapDeletePosixAccount(*t.DN); err != nil {
			return err
		}
	}

	// 4. create posixGroups
	glg.Infof("creating new posixGroup objects")
	for _, t = range taskList[models.ObjectTypePosixGroup][models.TaskTypeCreate] {
		if err = ldapCreatePosixGroup(t.Data.(models.PosixGroup)); err != nil {
			return err
		}
	}

	// 5. create posixGroups
	glg.Infof("deleting obsolete posixGroup objects")
	for _, t = range taskList[models.ObjectTypePosixGroup][models.TaskTypeDelete] {
		if err = ldapDeleteObject(*t.DN); err != nil {
			return nil
		}
	}

	// 6. update posixGroups
	glg.Infof("updating posixGroup objects")
	for _, t = range taskList[models.ObjectTypePosixGroup][models.TaskTypeUpdate] {
		if err = ldapUpdatePosixGroup(t.Data.(models.PosixGroup)); err != nil {
			return nil
		}
	}

	// 7. create posixAccounts
	glg.Infof("creating new posixAccount objects")
	for _, t = range taskList[models.ObjectTypePosixAccount][models.TaskTypeCreate] {
		if err = ldapCreatePosixAccount(t.Data.(models.PosixAccount)); err != nil {
			return err
		}
	}

	// 8. update posixAccounts
	glg.Infof("updating posixAccount objects")
	for _, t = range taskList[models.ObjectTypePosixAccount][models.TaskTypeUpdate] {
		if err = ldapUpdatePosixAccount(t.Data.(models.PosixAccount)); err != nil {
			return err
		}
	}

	// 9. create groupOfNames
	glg.Infof("creating new groupOfNames objects")
	for _, t = range taskList[models.ObjectTypeGroupOfNames][models.TaskTypeCreate] {
		if err = ldapCreateGroupOfNames(t.Data.(models.GroupOfNames)); err != nil {
			return err
		}
	}

	// 10. update groupOfNames
	glg.Infof("updating groupOfNames objects")
	for _, t = range taskList[models.ObjectTypeGroupOfNames][models.TaskTypeUpdate] {
		if err = ldapUpdateGroupOfNames(t.Data.(models.GroupOfNames)); err != nil {
			return err
		}
	}

	// 11. create group memberships
	glg.Infof("creating new groupOfNames memberships")
	for _, t = range taskList[models.ObjectTypeGroupOfNames][models.TaskTypeAddMember] {
		if err = ldapAddGroupOfNamesMember(*t.DN, t.Data.(string)); err != nil {
			return err
		}
	}

	// 12. create sudoRole
	glg.Infof("creating sudoRole objects")
	for _, t = range taskList[models.ObjectTypeSudoRole][models.TaskTypeCreate] {
		if err = ldapCreateSudoRole(t.Data.(models.SudoRole)); err != nil {
			return err
		}
	}

	// 13. update sudoRole
	glg.Infof("updating sudoRole objects")
	for _, t = range taskList[models.ObjectTypeSudoRole][models.TaskTypeUpdate] {
		if err = ldapUpdateSudoRole(t.Data.(models.SudoRole)); err != nil {
			return err
		}
	}

	// 14. delete sudoRole
	glg.Infof("deleting sudoRole objects")
	for _, t = range taskList[models.ObjectTypeSudoRole][models.TaskTypeDelete] {
		if err = ldapDeleteObject(*t.DN); err != nil {
			return err
		}
	}

	// 15. delete groupOfNames
	glg.Infof("deleting groupOfNames objects")
	for _, t = range taskList[models.ObjectTypeGroupOfNames][models.TaskTypeDelete] {
		if err = ldapDeleteObject(*t.DN); err != nil {
			return err
		}
	}

	// 16. delete OUs
	glg.Infof("deleting intermediate organizationalUnit objects")

	// re-sort tasks to have longer DNs be deleted first
	sort.Slice(taskList[models.ObjectTypeOrganisationalUnit][models.TaskTypeDelete], func(i, j int) bool {
		return len(*taskList[models.ObjectTypeOrganisationalUnit][models.TaskTypeDelete][i].DN) > len(*taskList[models.ObjectTypeOrganisationalUnit][models.TaskTypeDelete][j].DN)
	})

	for _, t = range taskList[models.ObjectTypeOrganisationalUnit][models.TaskTypeDelete] {
		if err = ldapDeleteObject(*t.DN); err != nil {
			return err
		}
	}

	glg.Info("Sync completed.")
	return nil
}
