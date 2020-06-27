package main

import (
	"github.com/4xoc/monban/tasks"

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

	glg.Infof("Data comparison complete. %d changes will be synced", tasks.TotalLength())

	// 1. create OUs
	glg.Infof("creating new OrganisationalUnits")
	queueCreateOU.For(func(task *tasks.Task) error {
		return ldapCreateOrganisationalUnit(task.Data.OrganizationalUnit)
	})

	// 2. delete group memberships
	glg.Infof("deleting obsolete groupOfNames memberships")
	queueDelMemberGoN.For(func(task *tasks.Task) error {
		return ldapDeleteGroupOfNamesMember(task.DN, task.Data.DN)
	})

	// 3. delete posixAccoounts
	glg.Infof("deleting obsolete posixAccount objects")
	queueDeletePA.For(func(task *tasks.Task) error {
		return ldapDeletePosixAccount(task.DN)
	})

	// 4. create posixGroups
	glg.Infof("creating new posixGroup objects")
	queueCreatePG.For(func(task *tasks.Task) error {
		return ldapCreatePosixGroup(task.Data.PosixGroup)
	})

	// 5. delete posixGroups
	glg.Infof("deleting obsolete posixGroup objects")
	queueDeletePG.For(func(task *tasks.Task) error {
		return ldapDeleteObject(task.DN)
	})

	// 6. update posixGroups
	glg.Infof("updating posixGroup objects")
	queueUpdatePG.For(func(task *tasks.Task) error {
		return ldapUpdatePosixGroup(task.Data.PosixGroup)
	})

	// 7. create posixAccounts
	glg.Infof("creating new posixAccount objects")
	queueCreatePA.For(func(task *tasks.Task) error {
		return ldapCreatePosixAccount(task.Data.PosixAccount)
	})

	// 8. update posixAccounts
	glg.Infof("updating posixAccount objects")
	queueUpdatePA.For(func(task *tasks.Task) error {
		return ldapUpdatePosixAccount(task.Data.PosixAccount)
	})

	// 9. create groupOfNames
	glg.Infof("creating new groupOfNames objects")
	queueCreateGoN.For(func(task *tasks.Task) error {
		return ldapCreateGroupOfNames(task.Data.GroupOfNames)
	})

	// 10. update groupOfNames
	glg.Infof("updating groupOfNames objects")
	queueUpdateGoN.For(func(task *tasks.Task) error {
		return ldapUpdateGroupOfNames(task.Data.GroupOfNames)
	})

	// 11. create group memberships
	glg.Infof("creating new groupOfNames memberships")
	queueAddMemberGoN.For(func(task *tasks.Task) error {
		return ldapAddGroupOfNamesMember(task.DN, task.Data.DN)
	})

	// 12. create sudoRole
	glg.Infof("creating sudoRole objects")
	queueCreateSR.For(func(task *tasks.Task) error {
		return ldapCreateSudoRole(task.Data.SudoRole)
	})

	// 13. update sudoRole
	glg.Infof("updating sudoRole objects")
	queueUpdateSR.For(func(task *tasks.Task) error {
		return ldapUpdateSudoRole(task.Data.SudoRole)
	})

	// 14. delete sudoRole
	glg.Infof("deleting sudoRole objects")
	queueDeleteSR.For(func(task *tasks.Task) error {
		return ldapDeleteObject(task.DN)
	})

	// 15. delete groupOfNames
	glg.Infof("deleting groupOfNames objects")
	queueDeleteGoN.For(func(task *tasks.Task) error {
		return ldapDeleteObject(task.DN)
	})

	// 16. delete OUs
	glg.Infof("deleting intermediate organizationalUnit objects")

	// for reversed to delete longest dn first
	queueDeleteOU.ForRev(func(task *tasks.Task) error {
		return ldapDeleteObject(task.DN)
	})

	glg.Info("Sync completed.")
	return nil
}
