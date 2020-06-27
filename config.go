package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/4xoc/monban/models"

	"github.com/kpango/glg"
	"github.com/urfave/cli/v2"
	"golang.org/x/crypto/ssh"
	"gopkg.in/yaml.v3"
)

// readConfiguration reads a given main rt.Configuration file
func readConfiguration(c *cli.Context) error {
	var (
		yamlFile []byte
		err      error
	)

	glg.Infof("reading main rt.Configuration file")

	// get absolut base path of monban rt.Config
	rt.BasePath, err = filepath.Abs(filepath.Dir(rt.ConfigFile))
	if err != nil {
		return fmt.Errorf("failed to get path to rt.Config file: %s", err.Error())
	}

	rt.ConfigFile = filepath.Join(rt.BasePath, filepath.Base(rt.ConfigFile))

	glg.Debugf("main rt.Config file path: %s", rt.ConfigFile)

	yamlFile, err = ioutil.ReadFile(rt.ConfigFile)
	if err != nil {
		return fmt.Errorf("failed to load rt.Config file: %s", err.Error())
	}

	err = yaml.Unmarshal(yamlFile, &rt.Config)
	if err != nil {
		return fmt.Errorf("failed to parse rt.Config file: %s", err.Error())
	}

	// rt.UserDN and rt.UserPassword are set by cli arguments and shouldn't be overwritten
	// sanity check
	if rt.Config.UserDN == nil && rt.UserDN == "" {
		return fmt.Errorf("user_dn is not set in rt.Config file or supplied as argument")
	} else if rt.UserDN != "" {
		rt.Config.UserDN = &rt.UserDN
	}

	if rt.Config.UserPassword == nil && rt.UserPassword == "" {
		return fmt.Errorf("user_password is not set in rt.Config file or supplied as argument")
	} else if rt.UserPassword != "" {
		rt.Config.UserPassword = &rt.UserPassword
	}

	if rt.Config.EnableSSHPublicKeys == nil {
		rt.Config.EnableSSHPublicKeys = new(bool)
		*rt.Config.EnableSSHPublicKeys = false
	}

	if rt.Config.HostURI == nil {
		return fmt.Errorf("missing required rt.Config `host_uri`")
	}

	if rt.Config.GroupDir == nil {
		return fmt.Errorf("missing required rt.Config `group_dir`")
	} else {
		// if relative, make it absolute
		if !filepath.IsAbs(*rt.Config.GroupDir) {
			*rt.Config.GroupDir = filepath.Join(filepath.Dir(rt.ConfigFile), *rt.Config.GroupDir)
		}
	}

	if rt.Config.PeopleDir == nil {
		return fmt.Errorf("missing required rt.Config `localPeople_dir`")
	} else {
		// if relative, make it absolute
		if !filepath.IsAbs(*rt.Config.PeopleDir) {
			*rt.Config.PeopleDir = filepath.Join(filepath.Dir(rt.ConfigFile), *rt.Config.PeopleDir)
		}
	}

	if rt.Config.RootDN == nil {
		return fmt.Errorf("missing required rt.Config `root_dn`")
	}

	if rt.Config.PeopleRDN != nil {
		rt.PeopleDN = fmt.Sprintf("%s,%s", *rt.Config.PeopleRDN, *rt.Config.RootDN)
	} else {
		rt.PeopleDN = *rt.Config.RootDN
	}

	if rt.Config.GroupRDN != nil {
		rt.GroupDN = fmt.Sprintf("%s,%s", *rt.Config.GroupRDN, *rt.Config.RootDN)
	} else {
		rt.GroupDN = *rt.Config.RootDN
	}

	glg.Debugf("=== rt.Config value from file and arguments ===")
	glg.Debugf("               host_uri: %s", *rt.Config.HostURI)
	glg.Debugf("                user_dn: %s", *rt.Config.UserDN)
	glg.Debugf("          user_password: %s", *rt.Config.UserPassword)
	glg.Debugf(" enable_ssh_public_keys: %t", *rt.Config.EnableSSHPublicKeys)
	glg.Debugf("              group_dir: %s", *rt.Config.GroupDir)
	glg.Debugf("             people_dir: %s", *rt.Config.PeopleDir)
	glg.Debugf("                root_dn: %s", *rt.Config.RootDN)
	glg.Debugf("             people_rdn: %s", *rt.Config.PeopleRDN)
	glg.Debugf("              group_rdn: %s", *rt.Config.GroupRDN)
	glg.Debugf("            enable_sudo: %t", rt.Config.EnableSudo)
	glg.Debugf("           generate_uid: %t", rt.Config.GenerateUID)

	if rt.Config.GenerateUID {
		// only tell about limit if defined
		if rt.Config.MinUID != 0 {
			glg.Debugf("                min_uid: %d", rt.Config.MinUID)
			rt.LatestUID = rt.Config.MinUID - 1 // uid is always incremented
		}

		// only tell about limit if defined
		if rt.Config.MaxUID != 0 {
			glg.Debugf("                max_uid: %d", rt.Config.MaxUID)
		}
	}

	if rt.Config.Defaults.DisplayName != nil {
		glg.Debugf(" (default) display_name: %s", *rt.Config.Defaults.DisplayName)
	}

	if rt.Config.Defaults.LoginShell != nil {
		glg.Debugf("  (default) login_shell: %s", *rt.Config.Defaults.LoginShell)
	}

	if rt.Config.Defaults.Mail != nil {
		glg.Debugf("         (default) mail: %s", *rt.Config.Defaults.Mail)
	}

	if rt.Config.Defaults.HomeDir != nil {
		glg.Debugf("     (default) home_dir: %s", *rt.Config.Defaults.HomeDir)
	}

	if rt.Config.Defaults.UserPassword != nil {
		glg.Debugf("(default) user_password: %s", *rt.Config.Defaults.UserPassword)
	}

	// sudo defaults
	if rt.Config.Defaults.Sudo != nil {
		// always overwrite CN
		rt.Config.Defaults.Sudo.CN = "Defaults"

	}

	glg.Infof("done reading main rt.Configuration file")

	return nil
}

// readPeopleConfiguration reads a localPeople rt.Config dir and performs some basic sanity checks
func readPeopleConfiguration() error {
	var (
		err           error
		files         []string
		currentFile   string
		yamlFile      []byte
		currentPeople *models.PosixGroup
		userIndex     int
		knownUsers    []string
		i             int
		ok            bool
		pathPieces    []string
		relPath       string
	)

	glg.Infof("reading people rt.Configuration file")

	err = filepath.Walk(filepath.Join(*rt.Config.PeopleDir),
		func(path string, info os.FileInfo, err error) error {
			var (
				ou *models.OrganizationalUnit
			)

			if err != nil {
				return err
			}

			if info.IsDir() {
				// check if dir needs added as a new OU
				// split path into dirs, each one will be its own OU in LDAP

				relPath, _ = filepath.Rel(*rt.Config.PeopleDir, path)

				if relPath != "." {
					// only creating an OU of the path is not the people_dir

					pathPieces = strings.Split(relPath, "/")

					ou = new(models.OrganizationalUnit)
					ou.CN = info.Name()
					ou.Description = "Managed by Monban"

					if len(pathPieces) <= 1 {
						ou.DN = fmt.Sprintf("ou=%s,%s", ou.CN, rt.PeopleDN)
					} else {
						ou.DN = fmt.Sprintf("ou=%s,%s,%s", ou.CN, generateOUDN(pathPieces[:len(pathPieces)-1]), rt.PeopleDN)
					}

					glg.Debugf("found intermediate OU %s", ou.DN)
					localOUs = append(localOUs, ou)
				}

			} else {
				// only collect files for below
				files = append(files, path)
			}

			return nil
		})
	if err != nil {
		return err
	}

	for _, currentFile = range files {

		glg.Infof("reading local people rt.Config file %s", currentFile)

		yamlFile, err = ioutil.ReadFile(currentFile)
		if err != nil {
			return fmt.Errorf("failed to load user rt.Config file: '%s'", err.Error())
		}

		// currentPeople needs to be reset before every Unmarshal
		currentPeople = new(models.PosixGroup)
		err = yaml.Unmarshal(yamlFile, currentPeople)
		if err != nil {
			return fmt.Errorf("failed to parse rt.Config file: '%s'", err.Error())
		}

		// unless cn has been specifically set, set cn based on file name
		if currentPeople.CN == "" {
			currentPeople.CN = filepath.Base(currentFile)
		}

		// set dn
		relPath, _ = filepath.Rel(*rt.Config.PeopleDir, currentFile)
		pathPieces = strings.Split(relPath, "/")
		if len(pathPieces) <= 1 {
			currentPeople.DN = fmt.Sprintf("cn=%s,%s", currentPeople.CN, rt.PeopleDN)
		} else {
			currentPeople.DN = fmt.Sprintf("cn=%s,%s,%s", currentPeople.CN, generateOUDN(pathPieces[:len(pathPieces)-1]), rt.PeopleDN)
		}

		if currentPeople.GIDNumber == nil {
			glg.Fatalf("gid_number missing in '%s'", currentFile)
		}

		if _, ok = localPeople[currentPeople.DN]; ok {
			return fmt.Errorf("dn %s already exists but was declared again in %s", currentPeople.DN, currentFile)
		}

		// set dummy description
		if currentPeople.Description == "" {
			currentPeople.Description = "managed by Monban"
		}

		// sanity check user objects
		for userIndex = range currentPeople.Objects {
			if currentPeople.Objects[userIndex].UID == nil ||
				currentPeople.Objects[userIndex].GivenName == nil ||
				currentPeople.Objects[userIndex].Surname == nil {
				return fmt.Errorf("user object with index '%d' in DN '%s' is missing one or more required fields", userIndex, currentFile)
			}

			// set dn
			currentPeople.Objects[userIndex].DN = fmt.Sprintf("uid=%s,%s", *currentPeople.Objects[userIndex].UID, currentPeople.DN)

			// when UID generation is disabled the UID must be set in file
			if !rt.Config.GenerateUID {
				if currentPeople.Objects[userIndex].UIDNumber == nil {
					glg.Fatalf("uid_number required because generate_uid is disabled but no value was given")
				}
			}

			// when GID is already known set user
			if currentPeople.Objects[userIndex].GIDNumber == nil {
				currentPeople.Objects[userIndex].GIDNumber = currentPeople.GIDNumber
			}

			if *rt.Config.EnableSSHPublicKeys {
				if currentPeople.Objects[userIndex].SSHPublicKey != nil {
					// validate data is indeed a valid ssh key
					_, _, _, _, err = ssh.ParseAuthorizedKey([]byte(*currentPeople.Objects[userIndex].SSHPublicKey))

					if err != nil {
						glg.Errorf("failed to parse ssh_public_key: %s", err.Error())
						// remove ssh key from entry
						currentPeople.Objects[userIndex].SSHPublicKey = nil
					}
				}
			}

			// add defaults if not otherwise rt.Configured
			if currentPeople.Objects[userIndex].DisplayName == nil {
				if rt.Config.Defaults.DisplayName == nil {
					return fmt.Errorf("cannot read object: display_name not set in object and no default is defined")
				}

				// construct from default
				currentPeople.Objects[userIndex].DisplayName = new(string)
				// set given_name
				*currentPeople.Objects[userIndex].DisplayName = strings.ReplaceAll(*rt.Config.Defaults.DisplayName, "%g", *currentPeople.Objects[userIndex].GivenName)
				// set surname
				*currentPeople.Objects[userIndex].DisplayName = strings.ReplaceAll(*currentPeople.Objects[userIndex].DisplayName, "%l", *currentPeople.Objects[userIndex].Surname)
				// set username
				*currentPeople.Objects[userIndex].DisplayName = strings.ReplaceAll(*currentPeople.Objects[userIndex].DisplayName, "%u", *currentPeople.Objects[userIndex].UID)
			}

			if currentPeople.Objects[userIndex].LoginShell == nil {
				if rt.Config.Defaults.LoginShell == nil {
					return fmt.Errorf("cannot read object: login_shell not set in object and no default is defined")
				}
				currentPeople.Objects[userIndex].LoginShell = rt.Config.Defaults.LoginShell
			}

			if currentPeople.Objects[userIndex].Mail == nil {
				if rt.Config.Defaults.Mail == nil {
					return fmt.Errorf("cannot read object: mail not set in object and no default is defined")
				}

				// construct from default
				currentPeople.Objects[userIndex].Mail = new(string)
				// set given_name
				*currentPeople.Objects[userIndex].Mail = strings.ReplaceAll(*rt.Config.Defaults.Mail, "%g", strings.ToLower(*currentPeople.Objects[userIndex].GivenName))
				// set surname
				*currentPeople.Objects[userIndex].Mail = strings.ReplaceAll(*currentPeople.Objects[userIndex].Mail, "%l", strings.ToLower(*currentPeople.Objects[userIndex].Surname))
				// set username
				*currentPeople.Objects[userIndex].Mail = strings.ReplaceAll(*currentPeople.Objects[userIndex].Mail, "%u", strings.ToLower(*currentPeople.Objects[userIndex].UID))
			}

			if currentPeople.Objects[userIndex].HomeDir == nil {
				if rt.Config.Defaults.HomeDir == nil {
					return fmt.Errorf("cannot read object: home_dir not set in object and no default is defined")
				}

				// construct from default
				currentPeople.Objects[userIndex].HomeDir = new(string)
				// set given_name
				*currentPeople.Objects[userIndex].HomeDir = strings.ReplaceAll(*rt.Config.Defaults.HomeDir, "%g", *currentPeople.Objects[userIndex].GivenName)
				// set surname
				*currentPeople.Objects[userIndex].HomeDir = strings.ReplaceAll(*currentPeople.Objects[userIndex].HomeDir, "%l", *currentPeople.Objects[userIndex].Surname)
				// set username
				*currentPeople.Objects[userIndex].HomeDir = strings.ReplaceAll(*currentPeople.Objects[userIndex].HomeDir, "%u", *currentPeople.Objects[userIndex].UID)
			}

			if currentPeople.Objects[userIndex].UserPassword == nil {
				if rt.Config.Defaults.UserPassword == nil {
					return fmt.Errorf("cannot read object: user_password not set in object and no default is defined")
				}
				// construct from default
				currentPeople.Objects[userIndex].UserPassword = new(string)
				// set username
				*currentPeople.Objects[userIndex].UserPassword = strings.ReplaceAll(*rt.Config.Defaults.UserPassword, "%u", *currentPeople.Objects[userIndex].UID)
			}

			// verify the same user isn't rt.Configured multiple times
			for i = range knownUsers {
				if knownUsers[i] == *currentPeople.Objects[userIndex].UID {
					return fmt.Errorf("user with username '%s' is rt.Configured multiple times", *currentPeople.Objects[userIndex].UID)
				}
			}

			// adding id to now known users
			knownUsers = append(knownUsers, *currentPeople.Objects[userIndex].UID)
			glg.Debugf("loaded local user with DN %s", currentPeople.Objects[userIndex].DN)
		}

		// add loaded file to global list of know user objects
		localPeople[currentPeople.DN] = currentPeople
	}

	glg.Infof("done reading people rt.Configuration file")
	return nil
}

// readGroupConfiguration reads a group rt.Config dir and performs basic sanity checks
func readGroupConfiguration() error {
	var (
		err          error
		files        []string
		currentFile  string
		currentGroup *models.GroupOfNames
		yamlFile     []byte
		i            int
		j            int
		match        int
		dn           string
		pathPieces   []string
		relPath      string
	)

	glg.Infof("reading group rt.Configuration file")

	err = filepath.Walk(filepath.Join(*rt.Config.GroupDir),
		func(path string, info os.FileInfo, err error) error {
			var (
				ou *models.OrganizationalUnit
			)

			if err != nil {
				return err
			}

			if info.IsDir() {
				// check if dir needs added as a new OU
				// split path into dirs, each one will be its own OU in LDAP

				relPath, _ = filepath.Rel(*rt.Config.GroupDir, path)

				if relPath != "." {
					// only creating an OU of the path is not the people_dir

					pathPieces = strings.Split(relPath, "/")

					ou = new(models.OrganizationalUnit)
					ou.CN = info.Name()
					ou.Description = "Managed by Monban"

					if len(pathPieces) <= 1 {
						ou.DN = fmt.Sprintf("ou=%s,%s", ou.CN, rt.GroupDN)
					} else {
						ou.DN = fmt.Sprintf("ou=%s,%s,%s", ou.CN, generateOUDN(pathPieces[:len(pathPieces)-1]), rt.GroupDN)
					}

					localOUs = append(localOUs, ou)
				}

			} else {
				// only collect files for below
				files = append(files, path)
			}

			return nil
		})
	if err != nil {
		return err
	}

	for _, currentFile = range files {

		if filepath.Base(currentFile) == "SUDOers" {

			// only check SUDOers files when feature is enabled
			if rt.Config.EnableSudo {
				err = readSUDOersFile(currentFile)
				if err != nil {
					glg.Errorf("failed to load SUDOers file: %s", err.Error())
				}
			}

			continue
		}

		glg.Infof("reading group config file %s", currentFile)

		yamlFile, err = ioutil.ReadFile(currentFile)
		if err != nil {
			return fmt.Errorf("failed to load group config file: %s", err.Error())
		}

		// currentGroup needs to be reset before every Unmarshal
		currentGroup = new(models.GroupOfNames)
		err = yaml.Unmarshal(yamlFile, currentGroup)
		if err != nil {
			return fmt.Errorf("failed to parse group config file: %s", err.Error())
		}

		// CN wasn't explicitly set, using filename instead
		if currentGroup.CN == "" {
			currentGroup.CN = filepath.Base(currentFile)
		}

		// generate DN
		relPath, _ = filepath.Rel(*rt.Config.GroupDir, currentFile)
		pathPieces = strings.Split(relPath, "/")
		if len(pathPieces) <= 1 {
			currentGroup.DN = fmt.Sprintf("cn=%s,%s", currentGroup.CN, rt.GroupDN)
		} else {
			currentGroup.DN = fmt.Sprintf("cn=%s,%s,%s", currentGroup.CN, generateOUDN(pathPieces[:len(pathPieces)-1]), rt.GroupDN)
		}
		glg.Debugf("loaded local group with DN %s", currentGroup.DN)

		// check if description is set
		if currentGroup.Description == "" {
			currentGroup.Description = "Managed by Monban"
		}

		// verify members are only added once
		for i = range currentGroup.Members {
			match = 0
			for j = range currentGroup.Members {
				if currentGroup.Members[i] == currentGroup.Members[j] {
					match++
				}

				if match > 1 {
					return fmt.Errorf("duplicated member entry with uid %s in group '%s'", currentGroup.Members[i], currentGroup.DN)
				}
			}

			// add member with DN not UID

			glg.Debugf("loaded group member with uid %s", currentGroup.Members[i])
		}

		// verify members also exist within rt.Config
		for i = range currentGroup.Members {
			match = 0
			for dn = range localPeople {
				for j = range localPeople[dn].Objects {

					if currentGroup.Members[i] == *localPeople[dn].Objects[j].UID {
						match++
						break
					}
				}
			}

			if match == 0 {
				return fmt.Errorf("member uid %s in group %s doesn't exist as user object", currentGroup.Members[i], currentGroup.DN)
			}
		}

		// add group to global list of groups
		localGroups[currentGroup.DN] = currentGroup
		glg.Debugf("loaded local group with DN %s", currentGroup.DN)
	}

	glg.Infof("done reading group rt.Configuration file")
	return nil
}

// generateOUDN generates a full DN of a list of pieces in hierarical order and an optional suffix
func generateOUDN(pieces []string) string {
	switch len(pieces) {
	case 1:
		return fmt.Sprintf("ou=%s", pieces[0])

	default:
		return fmt.Sprintf("%s,ou=%s", generateOUDN(pieces[1:]), pieces[0])
	}
}

// readSUDOersFile reads a SUDOers file and validates its contents
func readSUDOersFile(file string) error {
	var (
		yamlFile   []byte
		err        error
		tmpSudoers models.Sudoers
		relPath    string
		pathPieces []string
		dn         string
		i          int
		ou         models.OrganizationalUnit
		tmpRole    models.SudoRole
	)

	glg.Infof("reading SUDOers file %s", file)

	yamlFile, err = ioutil.ReadFile(file)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(yamlFile, &tmpSudoers)
	if err != nil {
		return fmt.Errorf("failed to parse rt.Config file: %s", err.Error())
	}

	// if there are no roles set, skip this file
	if len(tmpSudoers.Roles) == 0 {
		glg.Debugf("skipped SUDOers file without roles")
		return nil
	}

	// generate DN
	relPath, _ = filepath.Rel(*rt.Config.GroupDir, file)
	pathPieces = strings.Split(relPath, "/")
	if len(pathPieces) <= 1 {
		dn = fmt.Sprintf("ou=SUDOers,%s", rt.GroupDN)
	} else {
		dn = fmt.Sprintf("ou=SUDOers,%s,%s", generateOUDN(pathPieces[:len(pathPieces)-1]), rt.GroupDN)
	}

	// generate OU entry
	ou.DN = dn
	ou.CN = "SUDOers"
	ou.Description = "Managed by Monban"
	localOUs = append(localOUs, &ou)
	glg.Debugf("found intermediate OU %s", ou.DN)

	if tmpSudoers.DisableDefaults {
		glg.Debugf("defaults disabled for SUDOers file %s", file)
	} else {
		// adding default rule
		if rt.Config.Defaults.Sudo != nil {
			tmpRole = *rt.Config.Defaults.Sudo
			tmpRole.DN = fmt.Sprintf("cn=Defaults,%s", dn)
			localSudoRoles = append(localSudoRoles, &tmpRole)

			glg.Debugf("loaded default SUDOers role %s", tmpRole.DN)
		}
	}

	for i = range tmpSudoers.Roles {

		if tmpSudoers.Roles[i].CN == "" {
			return fmt.Errorf("mandatory name attribute is missing")
		}

		if tmpSudoers.Roles[i].Description == "" {
			tmpSudoers.Roles[i].Description = "Managed by Monban"
		}

		tmpSudoers.Roles[i].DN = fmt.Sprintf("cn=%s,%s", tmpSudoers.Roles[i].CN, dn)
		localSudoRoles = append(localSudoRoles, &tmpSudoers.Roles[i])

		glg.Debugf("loaded sudoRole with DN %s", tmpSudoers.Roles[i].DN)
	}

	return nil
}
