package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/go-ldap/ldap/v3"
	"github.com/kpango/glg"
)

// ldapConnect connects and binds to the configured hostURI
func ldapConnect() (*ldap.Conn, error) {
	var (
		con *ldap.Conn
		err error
	)

	con, err = ldap.DialURL(*config.HostURI)
	if err != nil {
		return nil, err
	}

	// bind with given credentials
	err = con.Bind(*config.UserDN, *config.UserPassword)
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
		user      *posixAccount
		group     *posixGroup
		ou        *organizationalUnit
		i         int
		j         int
		tmpPeople posixGroup
		// class will be posixAccount or posixGroup
		class string
	)

	glg.Infof("reading people objects from LDAP")

	// get a list of all existing objects within the peopleDN
	sr, err = ldapCon.Search(&ldap.SearchRequest{
		peopleDN,
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
		if sr.Entries[i].DN == peopleDN {
			// skip peopleDN object
			continue
		}

		user = new(posixAccount)
		group = new(posixGroup)
		ou = new(organizationalUnit)

		// set DN for all objects as it is unknown which object it is
		group.dn = sr.Entries[i].DN
		user.dn = sr.Entries[i].DN
		ou.dn = sr.Entries[i].DN

		// go through all attributes
		for j = range sr.Entries[i].Attributes {
			// assuming that there is only one value for all attributes
			switch sr.Entries[i].Attributes[j].Name {

			// there is no guarantee the class attribute is the first attribute, thus until known both structs are filled
			// with information
			case "objectClass":
				// there is more than one objectClass
				for _, class = range sr.Entries[i].Attributes[j].Values {
					if class == "posixAccount" || class == "posixGroup" || class == "organizationalUnit" {
						break
					}
				}

			case "ou":
				ou.cn = sr.Entries[i].Attributes[j].Values[0]

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
				if *user.UIDNumber > latestUID {
					latestUID = *user.UIDNumber
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

		// check if object was a posixAccount or posixGroup
		switch class {
		case "posixAccount":
			// add to global list of LDAP users
			// NOTE: this is a workaround as append on struct member within a map is not supported
			// see https://suraj.pro/post/golang_workaround/
			tmpPeople = ldapPeople[strings.SplitAfterN(user.dn, ",", 2)[1]]
			tmpPeople.Objects = append(tmpPeople.Objects, *user)
			ldapPeople[strings.SplitAfterN(user.dn, ",", 2)[1]] = tmpPeople
			glg.Debugf("found ldap posixAccount %s", user.dn)

		case "posixGroup":
			ldapPeople[group.dn] = *group
			glg.Debugf("found ldap posixGroup %s", group.dn)

		case "organizationalUnit":
			ldapOUs = append(ldapOUs, ou)
			glg.Debugf("found ldap intermediate OU %s", ou.dn)

		default:
			// using ou.dn but any other struct would work
			glg.Errorf("skipping object because of unknown objectClass in %s", ou.dn)
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
		group *groupOfNames
		ou    *organizationalUnit
		sudo  *sudoRole
		class string
	)

	// get a list of all existing objects within the groupDN
	sr, err = ldapCon.Search(&ldap.SearchRequest{
		groupDN,
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
		if sr.Entries[i].DN == groupDN {
			// skip groupDN object
			continue
		}

		group = new(groupOfNames)
		ou = new(organizationalUnit)
		sudo = new(sudoRole)

		// set DN for all objects as it is unknown which object it is
		group.dn = sr.Entries[i].DN
		ou.dn = sr.Entries[i].DN
		sudo.dn = sr.Entries[i].DN

		// go through all attributes
		for j = range sr.Entries[i].Attributes {
			// assuming that there is only one value for all attributes
			switch sr.Entries[i].Attributes[j].Name {

			// there is no guarantee the class attribute is the first attribute, thus until known both structs are filled
			// with information
			case "objectClass":
				// there is more than one objectClass
				for _, class = range sr.Entries[i].Attributes[j].Values {
					if class == "groupOfNames" || class == "organizationalUnit" || class == "sudoRole" {
						break
					}
				}

			case "ou":
				ou.cn = sr.Entries[i].Attributes[j].Values[0]

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

		// check if object was a posixAccount or posixGroup
		switch class {
		case "groupOfNames":
			// add to global list of LDAP groups
			// NOTE: this is a workaround as append on struct member within a map is not supported
			// see https://suraj.pro/post/golang_workaround/
			ldapGroups[group.dn] = *group
			glg.Debugf("found ldap groupOfNames %s", group.dn)

			for i = range group.Members {
				if group.Members[i] == "uid=MonbanDummyMember" {
					// ignore dummy member
					continue
				}
				glg.Debugf("found member %s", group.Members[i])
			}

		case "organizationalUnit":
			ldapOUs = append(ldapOUs, ou)
			glg.Debugf("found ldap intermediate OU %s", ou.dn)

		case "sudoRole":
			ldapSudoRoles = append(ldapSudoRoles, *sudo)
			glg.Debugf("found ldap sudoRole %s", sudo.dn)

		}
	}

	glg.Infof("successfully loaded group objects from LDAP")
	return nil
}

// ldapDeleteGroupOfNamesMember deletes a given user from a LDAP group
func ldapDeleteGroupOfNamesMember(group string, user string) error {
	var (
		modify *ldap.ModifyRequest
	)

	modify = ldap.NewModifyRequest(group, nil)
	modify.Delete("member", []string{user})

	return ldapCon.Modify(modify)
}

// ldapDeletePosixAccount delets a given posixAccount object from LDAP
func ldapDeletePosixAccount(dn string) error {
	var (
		err         error
		modify      *ldap.ModifyRequest
		dnFragments []string
	)

	// delete user object
	if err = ldapCon.Del(&ldap.DelRequest{
		DN:       dn,
		Controls: nil,
	}); err != nil {
		return err
	}

	// delete memberUid reference in UnixGroup
	dnFragments = strings.Split(dn, ",")

	glg.Debugf("deleting posixGroup member in %s", strings.Join(dnFragments[1:], ","))
	modify = ldap.NewModifyRequest(strings.Join(dnFragments[1:], ","), nil)

	modify.Delete("memberUid", []string{dnFragments[0][4:]})

	return ldapCon.Modify(modify)
}

// ldapCreatePosixAccount creates a new posixAccount object in LDAP
func ldapCreatePosixAccount(user posixAccount) error {
	var (
		err      error
		add      *ldap.AddRequest
		modify   *ldap.ModifyRequest
		dnSplits []string
	)

	glg.Debugf("creating posixAccount %s", user.dn)

	add = ldap.NewAddRequest(user.dn, nil)

	add.Attribute("objectClass", []string{
		"inetOrgPerson",
		"ldapPublicKey",
		"organizationalPerson",
		"person",
		"posixAccount",
		"shadowAccount",
		"top"})

	dnSplits = strings.Split(user.dn, ",")

	// UIDNumber should be set if generateUID is false
	if config.GenerateUID {
		latestUID++

		if latestUID > config.MaxUID {
			glg.Fatalf("reached max_uid limit")
		}

		user.UIDNumber = new(int)
		*user.UIDNumber = latestUID
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

	if *config.EnableSSHPublicKeys {
		if user.SSHPublicKey != nil {
			add.Attribute("sshPublicKey", []string{*user.SSHPublicKey})
		}
	}

	if err = ldapCon.Add(add); err != nil {
		return err
	}

	// create memberUid reference in UnixGroup
	modify = ldap.NewModifyRequest(strings.Join(dnSplits[1:], ","), nil)

	glg.Debugf("adding posixGroup member in %s", strings.Join(dnSplits[1:], ","))

	modify.Add("memberUid", []string{*user.UID})

	return ldapCon.Modify(modify)
}

// ldapUpdatePosixAccount updates an existing posixAccount object in LDAP
func ldapUpdatePosixAccount(user posixAccount) error {
	var (
		modify *ldap.ModifyRequest
	)

	glg.Debugf("updating posixAccount %s", user.dn)

	modify = ldap.NewModifyRequest(user.dn, nil)

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

	if *config.EnableSSHPublicKeys {
		if user.SSHPublicKey != nil {
			modify.Replace("sshPublicKey", []string{*user.SSHPublicKey})
		}
	}

	return ldapCon.Modify(modify)
}

// ldapAddGroupOfNamesMember adds a new given member to a LDAP groupOfNames
func ldapAddGroupOfNamesMember(group string, user string) error {
	var (
		modify *ldap.ModifyRequest
	)

	glg.Debugf("adding groupOfNames member in %s", group)

	modify = ldap.NewModifyRequest(group, nil)

	modify.Add("member", []string{user})

	return ldapCon.Modify(modify)
}

// ldapCreatePosixGroup creates a new posixGroup on LDAP target
func ldapCreatePosixGroup(group posixGroup) error {
	var (
		add *ldap.AddRequest
	)

	glg.Debugf("creating posixGroup %s", group.dn)

	add = ldap.NewAddRequest(group.dn, nil)

	add.Attribute("objectClass", []string{
		"posixGroup",
		"top"})

	// strings
	add.Attribute("cn", []string{group.CN})
	add.Attribute("gidNumber", []string{strconv.Itoa(*group.GIDNumber)})
	add.Attribute("description", []string{group.Description})

	return ldapCon.Add(add)
}

// ldapUpdatePosixGroup updates a given posixGroup on LDAP target
func ldapUpdatePosixGroup(group posixGroup) error {
	var (
		modify *ldap.ModifyRequest
	)

	glg.Debugf("updating posixGroup %s", group.dn)

	modify = ldap.NewModifyRequest(group.dn, nil)

	if group.GIDNumber != nil {
		modify.Replace("gidNumber", []string{strconv.Itoa(*group.GIDNumber)})
	}

	if group.Description != "" {
		modify.Replace("description", []string{group.Description})
	}

	return ldapCon.Modify(modify)
}

// ldapCreateGroupOfNames creates a new user group on LDAP target
func ldapCreateGroupOfNames(group groupOfNames) error {
	var (
		add *ldap.AddRequest
	)

	glg.Debugf("creating groupOfNames %s", group.dn)

	add = ldap.NewAddRequest(group.dn, nil)

	add.Attribute("objectClass", []string{
		"groupOfNames",
		"top"})

	// strings
	add.Attribute("cn", []string{group.CN})
	add.Attribute("member", []string{"uid=MonbanDummyMember"})
	add.Attribute("description", []string{group.Description})

	return ldapCon.Add(add)
}

// ldapUpdateGroupOfNames updates an existing groupOfNames object in LDAP
func ldapUpdateGroupOfNames(group groupOfNames) error {
	var (
		modify *ldap.ModifyRequest
	)

	glg.Debugf("updating groupOfName %s", group.dn)

	modify = ldap.NewModifyRequest(group.dn, nil)

	if group.Description != "" {
		modify.Replace("description", []string{group.Description})
	}

	return ldapCon.Modify(modify)
}

// ldapCreateOrganisationalUnit creates a new OU on LDAP target
func ldapCreateOrganisationalUnit(ou *organizationalUnit) error {
	var (
		add *ldap.AddRequest
	)

	glg.Debugf("creating organizationalUnit %s", ou.dn)

	add = ldap.NewAddRequest(ou.dn, nil)

	add.Attribute("objectClass", []string{
		"organizationalUnit",
		"top"})

	// strings
	add.Attribute("ou", []string{ou.cn})
	add.Attribute("description", []string{ou.description})

	return ldapCon.Add(add)
}

// ldapDeleteObject deletes any object identified by its dn
func ldapDeleteObject(dn string) error {
	return ldapCon.Del(&ldap.DelRequest{
		DN:       dn,
		Controls: nil,
	})
}

// ldapCreateSudoRole creates a new sudoRole on LDAP target
func ldapCreateSudoRole(role sudoRole) error {
	var (
		add *ldap.AddRequest
	)

	glg.Debugf("creating sudoRole %s", role.dn)

	add = ldap.NewAddRequest(role.dn, nil)

	add.Attribute("objectClass", []string{
		"sudoRole",
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

	return ldapCon.Add(add)
}

// ldapUpdateSudoRole updates an existing sudoRole on LDAP target; doesn't update CN
func ldapUpdateSudoRole(role sudoRole) error {
	var (
		modify *ldap.ModifyRequest
	)

	// this is different than other updates as values can be deleted

	glg.Debugf("updating sudoRole %s", role.dn)

	modify = ldap.NewModifyRequest(role.dn, nil)

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

	return ldapCon.Modify(modify)
}
