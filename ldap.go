package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/4xoc/monban/models"

	"github.com/go-ldap/ldap/v3"
	"github.com/kpango/glg"
)

// ldapConnect connects and binds to the configured hostURI
func ldapConnect() (*ldap.Conn, error) {
	var (
		con *ldap.Conn
		err error
	)

	con, err = ldap.DialURL(*rt.Config.HostURI)
	if err != nil {
		return nil, err
	}

	// bind with given credentials
	err = con.Bind(*rt.Config.UserDN, *rt.Config.UserPassword)
	if err != nil {
		return nil, err
	}

	glg.Infof("successfully connected & authenticated to LDAP")
	return con, nil
}

// ldapLoadPeople loads all people objects from LDAP
func ldapLoadPeople() error {
	var (
		err       error
		sr        *ldap.SearchResult
		user      *models.PosixAccount
		group     *models.PosixGroup
		ou        *models.OrganizationalUnit
		i         int
		j         int
		tmpPeople *models.PosixGroup
		// class will be models.PosixAccount or models.PosixGroup
		class string
	)

	glg.Infof("reading people objects from LDAP")

	// get a list of all existing objects within the rt.PeopleDN
	sr, err = rt.LdapCon.Search(&ldap.SearchRequest{
		rt.PeopleDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0,
		0,
		false,
		"(objectClass=*)",
		nil,
		nil,
	})
	if err != nil {
		// check if error is only group being missing
		if ldap.IsErrorWithCode(err, ldap.LDAPResultNoSuchObject) {

			return fmt.Errorf("people_rdn doesn't seem to exist: %s", err.Error())
		}
		return err
	}

	// go through all user objects
	// NOTE: this assumes the result is ordered in a way that children don't appear before its parent
	for i = range sr.Entries {
		if sr.Entries[i].DN == rt.PeopleDN {
			// skip rt.PeopleDN object
			continue
		}

		user = new(models.PosixAccount)
		group = new(models.PosixGroup)
		ou = new(models.OrganizationalUnit)

		// set DN for all objects as it is unknown which object it is
		group.DN = sr.Entries[i].DN
		user.DN = sr.Entries[i].DN
		ou.DN = sr.Entries[i].DN

		// go through all attributes
		for j = range sr.Entries[i].Attributes {
			// assuming that there is only one value for all attributes
			switch sr.Entries[i].Attributes[j].Name {

			// there is no guarantee the class attribute is the first attribute, thus until known both structs are filled
			// with information
			case "objectClass":
				// there is more than one objectClass
				for _, class = range sr.Entries[i].Attributes[j].Values {
					if class == "posixAccount" ||
						class == "posixGroup" ||
						class == "organizationalUnit" {
						break
					}
				}

			case "ou":
				ou.CN = sr.Entries[i].Attributes[j].Values[0]

			case "cn":
				group.CN = sr.Entries[i].Attributes[j].Values[0]
				user.UID = &sr.Entries[i].Attributes[j].Values[0]

			case "description":
				group.Description = sr.Entries[i].Attributes[j].Values[0]

			case "gidNumber":
				user.GIDNumber = new(int)
				*user.GIDNumber, _ = strconv.Atoi(sr.Entries[i].Attributes[j].Values[0])

				group.GIDNumber = new(int)
				*group.GIDNumber, _ = strconv.Atoi(sr.Entries[i].Attributes[j].Values[0])

			case "homeDirectory":
				user.HomeDir = &sr.Entries[i].Attributes[j].Values[0]

			case "sn":
				user.Surname = &sr.Entries[i].Attributes[j].Values[0]

			case "uidNumber":
				user.UIDNumber = new(int)
				*user.UIDNumber, _ = strconv.Atoi(sr.Entries[i].Attributes[j].Values[0])

				// determine the highest used UIDNumber
				if *user.UIDNumber > rt.LatestUID {
					rt.LatestUID = *user.UIDNumber
				}

			case "displayName":
				user.DisplayName = &sr.Entries[i].Attributes[j].Values[0]

			case "givenName":
				user.GivenName = &sr.Entries[i].Attributes[j].Values[0]

			case "loginShell":
				user.LoginShell = &sr.Entries[i].Attributes[j].Values[0]

			case "mail":
				user.Mail = &sr.Entries[i].Attributes[j].Values[0]

			case "sshPublicKey":
				user.SSHPublicKey = &sr.Entries[i].Attributes[j].Values[0]

			case "userPassword":
				user.UserPassword = &sr.Entries[i].Attributes[j].Values[0]

			}
		}

		// check objectClass
		switch class {
		case "posixAccount":
			// add to global list of LDAP users
			// NOTE: this is a workaround as append on struct member within a map is not supported
			// see https://suraj.pro/post/golang_workaround/
			tmpPeople = ldapPeople[strings.SplitAfterN(user.DN, ",", 2)[1]]
			tmpPeople.Objects = append(tmpPeople.Objects, user)
			ldapPeople[strings.SplitAfterN(user.DN, ",", 2)[1]] = tmpPeople
			glg.Debugf("found ldap posixAccount %s", user.DN)

		case "posixGroup":
			ldapPeople[group.DN] = group
			glg.Debugf("found ldap posixGroup %s", group.DN)

		case "organizationalUnit":
			ldapOUs = append(ldapOUs, ou)
			glg.Debugf("found ldap intermediate OU %s", ou.DN)

		default:
			// using ou.DN but any other struct would work
			glg.Errorf("skipping object because of unknown objectClass in %s", ou.DN)
		}
	}

	glg.Infof("successfully loaded people objects from LDAP")
	return nil
}

// ldapLoadGroups loads all group objects from LDAP
func ldapLoadGroups() error {
	var (
		err   error
		sr    *ldap.SearchResult
		i     int
		j     int
		k     int
		group *models.GroupOfNames
		ou    *models.OrganizationalUnit
		sudo  *models.SudoRole
		class string
	)

	// get a list of all existing objects within the rt.GroupDN
	sr, err = rt.LdapCon.Search(&ldap.SearchRequest{
		rt.GroupDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0,
		0,
		false,
		"(objectClass=*)",
		nil,
		nil,
	})
	if err != nil {
		// check if error is only group being missing
		if ldap.IsErrorWithCode(err, ldap.LDAPResultNoSuchObject) {
			return fmt.Errorf("group_rdn doesn't seem to exist: %s", err.Error())
		}
		return err
	}

	// go through all user objects
	// NOTE: this assumes the result is ordered in a way that children don't appear before its parent
	for i = range sr.Entries {
		if sr.Entries[i].DN == rt.GroupDN {
			// skip rt.GroupDN object
			continue
		}

		group = new(models.GroupOfNames)
		ou = new(models.OrganizationalUnit)
		sudo = new(models.SudoRole)

		// set DN for all objects as it is unknown which object it is
		group.DN = sr.Entries[i].DN
		ou.DN = sr.Entries[i].DN
		sudo.DN = sr.Entries[i].DN

		// go through all attributes
		for j = range sr.Entries[i].Attributes {
			// assuming that there is only one value for all attributes
			switch sr.Entries[i].Attributes[j].Name {

			// there is no guarantee the class attribute is the first attribute, thus until known both structs are filled
			// with information
			case "objectClass":
				// there is more than one objectClass
				for _, class = range sr.Entries[i].Attributes[j].Values {
					if class == "groupOfNames" ||
						class == "organizationalUnit" ||
						class == "sudoRole" {
						break
					}
				}

			case "ou":
				ou.CN = sr.Entries[i].Attributes[j].Values[0]

			case "cn":
				group.CN = sr.Entries[i].Attributes[j].Values[0]
				sudo.CN = sr.Entries[i].Attributes[j].Values[0]

			case "description":
				group.Description = sr.Entries[i].Attributes[j].Values[0]
				sudo.Description = sr.Entries[i].Attributes[j].Values[0]

			case "member":
				// rewrite members to UIDs only as this the format used in config files

				for k = range sr.Entries[i].Attributes[j].Values {

					if sr.Entries[i].Attributes[j].Values[k] == "uid=MonbanDummyMember" {
						// ignoring dummy member
						continue
					}

					group.Members = append(group.Members, strings.Split(sr.Entries[i].Attributes[j].Values[k], ",")[0][4:])
				}

			case "sudoUser":
				sudo.SudoUser = sr.Entries[i].Attributes[j].Values

			case "sudoHost":
				sudo.SudoHost = sr.Entries[i].Attributes[j].Values

			case "sudoCommand":
				sudo.SudoCommand = sr.Entries[i].Attributes[j].Values

			case "sudoOption":
				sudo.SudoOption = sr.Entries[i].Attributes[j].Values

			case "sudoRunAsUser":
				sudo.SudoRunAsUser = sr.Entries[i].Attributes[j].Values

			case "sudoRunAsGroup":
				sudo.SudoRunAsGroup = sr.Entries[i].Attributes[j].Values

			case "sudoNotBefore":
				sudo.SudoNotBefore = sr.Entries[i].Attributes[j].Values

			case "sudoNotAfter":
				sudo.SudoNotAfter = sr.Entries[i].Attributes[j].Values

			case "sudoOrder":
				sudo.SudoOrder = new(int)
				*sudo.SudoOrder, _ = strconv.Atoi(sr.Entries[i].Attributes[j].Values[0])
			}
		}

		// check if object was a models.PosixAccount or models.PosixGroup
		switch class {
		case "groupOfNames":
			// add to global list of LDAP groups
			// NOTE: this is a workaround as append on struct member within a map is not supported
			// see https://suraj.pro/post/golang_workaround/
			ldapGroups[group.DN] = group
			glg.Debugf("found ldap groupOfNames %s", group.DN)

			for i = range group.Members {
				if group.Members[i] == "uid=MonbanDummyMember" {
					// ignore dummy member
					continue
				}
				glg.Debugf("found member %s", group.Members[i])
			}

		case "organizationalUnit":
			ldapOUs = append(ldapOUs, ou)
			glg.Debugf("found ldap intermediate OU %s", ou.DN)

		case "sudoRole":
			ldapSudoRoles = append(ldapSudoRoles, sudo)
			glg.Debugf("found ldap sudoRole %s", sudo.DN)
		}
	}

	glg.Infof("successfully loaded group objects from LDAP")
	return nil
}

// ldapDeleteGroupOfNamesMember deletes a given user from a LDAP group
func ldapDeleteGroupOfNamesMember(group *string, user *string) error {
	var (
		modify *ldap.ModifyRequest
	)

	modify = ldap.NewModifyRequest(*group, nil)
	modify.Delete("member", []string{*user})

	return rt.LdapCon.Modify(modify)
}

// ldapDeletePosixAccount delets a given models.PosixAccount object from LDAP
func ldapDeletePosixAccount(dn *string) error {
	var (
		err         error
		modify      *ldap.ModifyRequest
		dnFragments []string
	)

	// delete user object
	if err = rt.LdapCon.Del(&ldap.DelRequest{
		DN:       *dn,
		Controls: nil,
	}); err != nil {
		return err
	}

	// delete memberUid reference in UnixGroup
	dnFragments = strings.Split(*dn, ",")

	glg.Debugf("deleting posixGroup member in %s", strings.Join(dnFragments[1:], ","))
	modify = ldap.NewModifyRequest(strings.Join(dnFragments[1:], ","), nil)

	modify.Delete("memberUid", []string{dnFragments[0][4:]})

	return rt.LdapCon.Modify(modify)
}

// ldapCreatePosixAccount creates a new models.PosixAccount object in LDAP
func ldapCreatePosixAccount(user *models.PosixAccount) error {
	var (
		err      error
		add      *ldap.AddRequest
		modify   *ldap.ModifyRequest
		dnSplits []string
	)

	glg.Debugf("creating posixAccount %s", user.DN)

	add = ldap.NewAddRequest(user.DN, nil)

	add.Attribute("objectClass", []string{
		"inetOrgPerson",
		"ldapPublicKey",
		"organizationalPerson",
		"person",
		"models.PosixAccount",
		"shadowAccount",
		"top"})

	dnSplits = strings.Split(user.DN, ",")

	// UIDNumber should be set if generateUID is false
	if rt.Config.GenerateUID {
		rt.LatestUID++

		if rt.LatestUID > rt.Config.MaxUID {
			glg.Fatalf("reached max_uid limit")
		}

		user.UIDNumber = new(int)
		*user.UIDNumber = rt.LatestUID
	}

	// strings
	add.Attribute("cn", []string{*user.UID})
	add.Attribute("gidNumber", []string{strconv.Itoa(*user.GIDNumber)})
	add.Attribute("uidNumber", []string{strconv.Itoa(*user.UIDNumber)})
	add.Attribute("homeDirectory", []string{*user.HomeDir})
	add.Attribute("sn", []string{*user.Surname})
	add.Attribute("uid", []string{*user.UID})
	add.Attribute("displayName", []string{*user.DisplayName})
	add.Attribute("givenName", []string{*user.GivenName})
	add.Attribute("loginShell", []string{*user.LoginShell})
	add.Attribute("mail", []string{*user.Mail})
	add.Attribute("userPassword", []string{*user.UserPassword})

	if *rt.Config.EnableSSHPublicKeys {
		if user.SSHPublicKey != nil {
			add.Attribute("sshPublicKey", []string{*user.SSHPublicKey})
		}
	}

	if err = rt.LdapCon.Add(add); err != nil {
		return err
	}

	// create memberUid reference in UnixGroup
	modify = ldap.NewModifyRequest(strings.Join(dnSplits[1:], ","), nil)

	glg.Debugf("adding posixGroup member in %s", strings.Join(dnSplits[1:], ","))

	modify.Add("memberUid", []string{*user.UID})

	return rt.LdapCon.Modify(modify)
}

// ldapUpdatePosixAccount updates an existing models.PosixAccount object in LDAP
func ldapUpdatePosixAccount(user *models.PosixAccount) error {
	var (
		modify *ldap.ModifyRequest
	)

	glg.Debugf("updating posixAccount %s", user.DN)

	modify = ldap.NewModifyRequest(user.DN, nil)

	if user.UID != nil {
		modify.Replace("cn", []string{*user.UID})
	}

	if user.GIDNumber != nil {
		modify.Replace("gidNumber", []string{strconv.Itoa(*user.GIDNumber)})
	}

	if user.UIDNumber != nil {
		modify.Replace("uidNumber", []string{strconv.Itoa(*user.UIDNumber)})
	}

	if user.HomeDir != nil {
		modify.Replace("homeDirectory", []string{*user.HomeDir})
	}

	if user.Surname != nil {
		modify.Replace("sn", []string{*user.Surname})
	}

	if user.DisplayName != nil {
		modify.Replace("displayName", []string{*user.DisplayName})
	}

	if user.GivenName != nil {
		modify.Replace("givenName", []string{*user.GivenName})
	}

	if user.LoginShell != nil {
		modify.Replace("loginShell", []string{*user.LoginShell})
	}

	if user.Mail != nil {
		modify.Replace("mail", []string{*user.Mail})
	}

	if user.UserPassword != nil {
		modify.Replace("userPassword", []string{*user.UserPassword})
	}

	if *rt.Config.EnableSSHPublicKeys {
		if user.SSHPublicKey != nil {
			modify.Replace("sshPublicKey", []string{*user.SSHPublicKey})
		}
	}

	return rt.LdapCon.Modify(modify)
}

// ldapAddGroupOfNamesMember adds a new given member to a LDAP models.GroupOfNames
func ldapAddGroupOfNamesMember(group *string, user *string) error {
	var (
		modify *ldap.ModifyRequest
	)

	glg.Debugf("adding groupOfNames member in %s", group)

	modify = ldap.NewModifyRequest(*group, nil)

	modify.Add("member", []string{*user})

	return rt.LdapCon.Modify(modify)
}

// ldapCreatePosixGroup creates a new models.PosixGroup on LDAP target
func ldapCreatePosixGroup(group *models.PosixGroup) error {
	var (
		add *ldap.AddRequest
	)

	glg.Debugf("creating posixGroup %s", group.DN)

	add = ldap.NewAddRequest(group.DN, nil)

	add.Attribute("objectClass", []string{
		"models.PosixGroup",
		"top"})

	// strings
	add.Attribute("cn", []string{group.CN})
	add.Attribute("gidNumber", []string{strconv.Itoa(*group.GIDNumber)})
	add.Attribute("description", []string{group.Description})

	return rt.LdapCon.Add(add)
}

// ldapUpdatePosixGroup updates a given models.PosixGroup on LDAP target
func ldapUpdatePosixGroup(group *models.PosixGroup) error {
	var (
		modify *ldap.ModifyRequest
	)

	glg.Debugf("updating posixGroup %s", group.DN)

	modify = ldap.NewModifyRequest(group.DN, nil)

	if group.GIDNumber != nil {
		modify.Replace("gidNumber", []string{strconv.Itoa(*group.GIDNumber)})
	}

	if group.Description != "" {
		modify.Replace("description", []string{group.Description})
	}

	return rt.LdapCon.Modify(modify)
}

// ldapCreateGroupOfNames creates a new user group on LDAP target
func ldapCreateGroupOfNames(group *models.GroupOfNames) error {
	var (
		add *ldap.AddRequest
	)

	glg.Debugf("creating groupOfNames %s", group.DN)

	add = ldap.NewAddRequest(group.DN, nil)

	add.Attribute("objectClass", []string{
		"models.GroupOfNames",
		"top"})

	// strings
	add.Attribute("cn", []string{group.CN})
	add.Attribute("member", []string{"uid=MonbanDummyMember"})
	add.Attribute("description", []string{group.Description})

	return rt.LdapCon.Add(add)
}

// ldapUpdateGroupOfNames updates an existing models.GroupOfNames object in LDAP
func ldapUpdateGroupOfNames(group *models.GroupOfNames) error {
	var (
		modify *ldap.ModifyRequest
	)

	glg.Debugf("updating groupOfName %s", group.DN)

	modify = ldap.NewModifyRequest(group.DN, nil)

	if group.Description != "" {
		modify.Replace("description", []string{group.Description})
	}

	return rt.LdapCon.Modify(modify)
}

// ldapCreateOrganisationalUnit creates a new OU on LDAP target
func ldapCreateOrganisationalUnit(ou *models.OrganizationalUnit) error {
	var (
		add *ldap.AddRequest
	)

	glg.Debugf("creating organizationalUnit %s", ou.DN)

	add = ldap.NewAddRequest(ou.DN, nil)

	add.Attribute("objectClass", []string{
		"models.OrganizationalUnit",
		"top"})

	// strings
	add.Attribute("ou", []string{ou.CN})
	add.Attribute("description", []string{ou.Description})

	return rt.LdapCon.Add(add)
}

// ldapDeleteObject deletes any object identified by its dn
func ldapDeleteObject(dn *string) error {
	return rt.LdapCon.Del(&ldap.DelRequest{
		DN:       *dn,
		Controls: nil,
	})
}

// ldapCreateSudoRole creates a new models.SudoRole on LDAP target
func ldapCreateSudoRole(role *models.SudoRole) error {
	var (
		add *ldap.AddRequest
	)

	glg.Debugf("creating sudoRole %s", role.DN)

	add = ldap.NewAddRequest(role.DN, nil)

	add.Attribute("objectClass", []string{
		"models.SudoRole",
		"top"})

	// cn & description are always set
	add.Attribute("cn", []string{role.CN})
	add.Attribute("description", []string{role.Description})

	if role.SudoOrder != nil {
		add.Attribute("sudoOrder", []string{strconv.Itoa(*role.SudoOrder)})
	}

	if len(role.SudoUser) > 0 {
		add.Attribute("sudoUser", role.SudoUser)
	}

	if len(role.SudoHost) > 0 {
		add.Attribute("sudoHost", role.SudoHost)
	}

	if len(role.SudoCommand) > 0 {
		add.Attribute("sudoCommand", role.SudoCommand)
	}

	if len(role.SudoOption) > 0 {
		add.Attribute("sudoOption", role.SudoOption)
	}

	if len(role.SudoRunAsUser) > 0 {
		add.Attribute("sudoRunAsUser", role.SudoRunAsUser)
	}

	if len(role.SudoRunAsGroup) > 0 {
		add.Attribute("sudoRunAsGroup", role.SudoRunAsGroup)
	}

	if len(role.SudoNotBefore) > 0 {
		add.Attribute("sudoNotBefore", role.SudoNotBefore)
	}

	if len(role.SudoNotAfter) > 0 {
		add.Attribute("sudoNotAfter", role.SudoNotAfter)
	}

	return rt.LdapCon.Add(add)
}

// ldapUpdateSudoRole updates an existing models.SudoRole on LDAP target; doesn't update CN
func ldapUpdateSudoRole(role *models.SudoRole) error {
	var (
		modify *ldap.ModifyRequest
	)

	// this is different than other updates as values can be deleted

	glg.Debugf("updating sudoRole %s", role.DN)

	modify = ldap.NewModifyRequest(role.DN, nil)

	// can be deleted
	if role.SudoOrder != nil {
		modify.Replace("sudoOrder", []string{strconv.Itoa(*role.SudoOrder)})
	} else {
		modify.Replace("sudoOrder", []string{})
	}

	// is always "Managed by Monban" or something configured in file - never empty
	if role.Description != "" {
		modify.Replace("description", []string{role.Description})
	}

	if len(role.SudoUser) > 0 {
		modify.Replace("sudoUser", role.SudoUser)
	} else {
		modify.Replace("sudoUser", []string{})
	}

	if len(role.SudoHost) > 0 {
		modify.Replace("sudoHost", role.SudoHost)
	} else {
		modify.Replace("sudoHost", []string{})
	}

	if len(role.SudoCommand) > 0 {
		modify.Replace("sudoCommand", role.SudoCommand)
	} else {
		modify.Replace("sudoCommand", []string{})
	}

	if len(role.SudoOption) > 0 {
		modify.Replace("sudoOption", role.SudoOption)
	} else {
		modify.Replace("sudoOption", []string{})
	}

	if len(role.SudoRunAsUser) > 0 {
		modify.Replace("sudoRunAsUser", role.SudoRunAsUser)
	} else {
		modify.Replace("sudoRunAsUser", []string{})
	}

	if len(role.SudoRunAsGroup) > 0 {
		modify.Replace("sudoRunAsGroup", role.SudoRunAsGroup)
	} else {
		modify.Replace("sudoRunAsGroup", []string{})
	}

	if len(role.SudoNotBefore) > 0 {
		modify.Replace("sudoNotBefore", role.SudoNotBefore)
	} else {
		modify.Replace("sudoNotBefore", []string{})
	}

	if len(role.SudoNotAfter) > 0 {
		modify.Replace("sudoNotAfter", role.SudoNotAfter)
	} else {
		modify.Replace("sudoNotAfter", []string{})
	}

	return rt.LdapCon.Modify(modify)
}
