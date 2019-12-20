package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/kpango/glg"
	"github.com/urfave/cli/v2"
	"golang.org/x/crypto/ssh"
	"gopkg.in/yaml.v3"
)

// readConfiguration reads a given main configuration file
func readConfiguration(c *cli.Context) error {
	var (
		yamlFile []byte
		err      error
	)

	glg.Infof("reading main configuration file")

	// get absolut base path of monban config
	basePath, err = filepath.Abs(filepath.Dir(configFile))
	if err != nil {
		return fmt.Errorf("failed to get path to config file: %s", err.Error())
	}

	configFile = filepath.Join(basePath, filepath.Base(configFile))

	glg.Debugf("main config file path: %s", configFile)

	yamlFile, err = ioutil.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to load config file: %s", err.Error())
	}

	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		return fmt.Errorf("failed to parse config file: %s", err.Error())
	}

	// userDN and userPassword are set by cli arguments and shouldn't be overwritten
	// sanity check
	if config.UserDN == nil && userDN == "" {
		return fmt.Errorf("user_dn is not set in config file or supplied as argument")
	} else if userDN != "" {
		config.UserDN = &userDN
	}

	if config.UserPassword == nil && userPassword == "" {
		return fmt.Errorf("user_password is not set in config file or supplied as argument")
	} else if userPassword != "" {
		config.UserPassword = &userPassword
	}

	if config.EnableSSHPublicKeys == nil {
		config.EnableSSHPublicKeys = new(bool)
		*config.EnableSSHPublicKeys = false
	}

	if config.HostURI == nil {
		return fmt.Errorf("missing required config `host_uri`")
	}

	if config.GroupDir == nil {
		return fmt.Errorf("missing required config `group_dir`")
	} else {
		// if relative, make it absolute
		if !filepath.IsAbs(*config.GroupDir) {
			*config.GroupDir = filepath.Join(filepath.Dir(configFile), *config.GroupDir)
		}
	}

	if config.PeopleDir == nil {
		return fmt.Errorf("missing required config `localPeople_dir`")
	} else {
		// if relative, make it absolute
		if !filepath.IsAbs(*config.PeopleDir) {
			*config.PeopleDir = filepath.Join(filepath.Dir(configFile), *config.PeopleDir)
		}
	}

	if config.RootDN == nil {
		return fmt.Errorf("missing required config `root_dn`")
	}

	if config.PeopleRDN != nil {
		peopleDN = fmt.Sprintf("%s,%s", *config.PeopleRDN, *config.RootDN)
	} else {
		peopleDN = *config.RootDN
	}

	if config.GroupRDN != nil {
		groupDN = fmt.Sprintf("%s,%s", *config.GroupRDN, *config.RootDN)
	} else {
		groupDN = *config.RootDN
	}

	glg.Debugf("=== config value from file and arguments ===")
	glg.Debugf("               host_uri: %s", *config.HostURI)
	glg.Debugf("                user_dn: %s", *config.UserDN)
	glg.Debugf("          user_password: %s", *config.UserPassword)
	glg.Debugf(" enable_ssh_public_keys: %t", *config.EnableSSHPublicKeys)
	glg.Debugf("              group_dir: %s", *config.GroupDir)
	glg.Debugf("             people_dir: %s", *config.PeopleDir)
	glg.Debugf("                root_dn: %s", *config.RootDN)
	glg.Debugf("             people_rdn: %s", *config.PeopleRDN)
	glg.Debugf("              group_rdn: %s", *config.GroupRDN)
	glg.Debugf("           generate_uid: %t", config.GenerateUID)

	if config.GenerateUID {
		// only tell about limit if defined
		if config.MinUID != 0 {
			glg.Debugf("                min_uid: %d", config.MinUID)
			latestUID = config.MinUID - 1 // uid is always incremented
		}

		// only tell about limit if defined
		if config.MaxUID != 0 {
			glg.Debugf("                max_uid: %d", config.MaxUID)
		}
	}

	if config.Defaults.DisplayName != nil {
		glg.Debugf(" (default) display_name: %s", *config.Defaults.DisplayName)
	}

	if config.Defaults.LoginShell != nil {
		glg.Debugf("  (default) login_shell: %s", *config.Defaults.LoginShell)
	}

	if config.Defaults.Mail != nil {
		glg.Debugf("         (default) mail: %s", *config.Defaults.Mail)
	}

	if config.Defaults.HomeDir != nil {
		glg.Debugf("     (default) home_dir: %s", *config.Defaults.HomeDir)
	}

	if config.Defaults.UserPassword != nil {
		glg.Debugf("(default) user_password: %s", *config.Defaults.UserPassword)
	}

	glg.Infof("done reading main configuration file")

	// init maps
	localPeople = make(map[string]posixGroup)
	localGroups = make(map[string]groupOfNames)

	ldapPeople = make(map[string]posixGroup)
	ldapGroups = make(map[string]groupOfNames)

	return nil
}

// readPeopleConfiguration reads a localPeople config dir and performs some basic sanity checks
func readPeopleConfiguration() error {
	var (
		err           error
		files         []string
		currentFile   string
		dn            string
		yamlFile      []byte
		currentPeople *posixGroup
		userIndex     int
		knownUsers    []string
		i             int
		ok            bool
		pathPieces    []string
		relPath       string
	)

	glg.Infof("reading people configuration file")

	err = filepath.Walk(filepath.Join(*config.PeopleDir),
		func(path string, info os.FileInfo, err error) error {
			var (
				ou *organizationalUnit
			)

			if err != nil {
				return err
			}

			if info.IsDir() {
				// check if dir needs added as a new OU
				// split path into dirs, each one will be its own OU in LDAP

				relPath, _ = filepath.Rel(*config.PeopleDir, path)

				if relPath != "." {
					// only creating an OU of the path is not the people_dir

					pathPieces = strings.Split(relPath, "/")

					ou = new(organizationalUnit)
					ou.cn = info.Name()
					ou.description = "Managed by Monban"

					if len(pathPieces) <= 1 {
						ou.dn = fmt.Sprintf("ou=%s,%s", ou.cn, peopleDN)
					} else {
						ou.dn = fmt.Sprintf("ou=%s,%s,%s", ou.cn, generateOUDN(pathPieces[:len(pathPieces)-1]), peopleDN)
					}

					glg.Debugf("found intermediate OU %s", ou.dn)
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

		glg.Infof("reading local people config file %s", currentFile)

		yamlFile, err = ioutil.ReadFile(currentFile)
		if err != nil {
			return fmt.Errorf("failed to load user config file: '%s'", err.Error())
		}

		// currentPeople needs to be reset before every Unmarshal
		currentPeople = new(posixGroup)
		err = yaml.Unmarshal(yamlFile, currentPeople)
		if err != nil {
			return fmt.Errorf("failed to parse config file: '%s'", err.Error())
		}

		// unless cn has been specifically set, set cn based on file name
		if currentPeople.CN == "" {
			currentPeople.CN = filepath.Base(currentFile)
		}

		// set dn
		relPath, _ = filepath.Rel(*config.PeopleDir, currentFile)
		pathPieces = strings.Split(relPath, "/")
		if len(pathPieces) <= 1 {
			currentPeople.dn = fmt.Sprintf("cn=%s,%s", currentPeople.CN, peopleDN)
		} else {
			currentPeople.dn = fmt.Sprintf("cn=%s,%s,%s", currentPeople.CN, generateOUDN(pathPieces[:len(pathPieces)-1]), peopleDN)
		}

		if currentPeople.GIDNumber == nil {
			glg.Fatalf("gid_number missing in '%s'", currentFile)
		}

		if _, ok = localPeople[dn]; ok {
			return fmt.Errorf("dn %s already exists but was declared again in %s", dn, currentFile)
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
			currentPeople.Objects[userIndex].dn = fmt.Sprintf("uid=%s,%s", *currentPeople.Objects[userIndex].UID, currentPeople.dn)

			// when UID generation is disabled the UID must be set in file
			if !config.GenerateUID {
				if currentPeople.Objects[userIndex].UIDNumber == nil {
					glg.Fatalf("uid_number required because generate_uid is disabled but no value was given")
				}
			}

			// when GID is already known set user
			if currentPeople.Objects[userIndex].GIDNumber == nil {
				currentPeople.Objects[userIndex].GIDNumber = currentPeople.GIDNumber
			}

			if *config.EnableSSHPublicKeys {
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

			// add defaults if not otherwise configured
			if currentPeople.Objects[userIndex].DisplayName == nil {
				if config.Defaults.DisplayName == nil {
					return fmt.Errorf("cannot read object: display_name not set in object and no default is defined")
				}

				// construct from default
				currentPeople.Objects[userIndex].DisplayName = new(string)
				// set given_name
				*currentPeople.Objects[userIndex].DisplayName = strings.ReplaceAll(*config.Defaults.DisplayName, "%g", *currentPeople.Objects[userIndex].GivenName)
				// set surname
				*currentPeople.Objects[userIndex].DisplayName = strings.ReplaceAll(*currentPeople.Objects[userIndex].DisplayName, "%l", *currentPeople.Objects[userIndex].Surname)
				// set username
				*currentPeople.Objects[userIndex].DisplayName = strings.ReplaceAll(*currentPeople.Objects[userIndex].DisplayName, "%u", *currentPeople.Objects[userIndex].UID)
			}

			if currentPeople.Objects[userIndex].LoginShell == nil {
				if config.Defaults.LoginShell == nil {
					return fmt.Errorf("cannot read object: login_shell not set in object and no default is defined")
				}
				currentPeople.Objects[userIndex].LoginShell = config.Defaults.LoginShell
			}

			if currentPeople.Objects[userIndex].Mail == nil {
				if config.Defaults.Mail == nil {
					return fmt.Errorf("cannot read object: mail not set in object and no default is defined")
				}

				// construct from default
				currentPeople.Objects[userIndex].Mail = new(string)
				// set given_name
				*currentPeople.Objects[userIndex].Mail = strings.ReplaceAll(*config.Defaults.Mail, "%g", strings.ToLower(*currentPeople.Objects[userIndex].GivenName))
				// set surname
				*currentPeople.Objects[userIndex].Mail = strings.ReplaceAll(*currentPeople.Objects[userIndex].Mail, "%l", strings.ToLower(*currentPeople.Objects[userIndex].Surname))
				// set username
				*currentPeople.Objects[userIndex].Mail = strings.ReplaceAll(*currentPeople.Objects[userIndex].Mail, "%u", strings.ToLower(*currentPeople.Objects[userIndex].UID))
			}

			if currentPeople.Objects[userIndex].HomeDir == nil {
				if config.Defaults.HomeDir == nil {
					return fmt.Errorf("cannot read object: home_dir not set in object and no default is defined")
				}

				// construct from default
				currentPeople.Objects[userIndex].HomeDir = new(string)
				// set given_name
				*currentPeople.Objects[userIndex].HomeDir = strings.ReplaceAll(*config.Defaults.HomeDir, "%g", *currentPeople.Objects[userIndex].GivenName)
				// set surname
				*currentPeople.Objects[userIndex].HomeDir = strings.ReplaceAll(*currentPeople.Objects[userIndex].HomeDir, "%l", *currentPeople.Objects[userIndex].Surname)
				// set username
				*currentPeople.Objects[userIndex].HomeDir = strings.ReplaceAll(*currentPeople.Objects[userIndex].HomeDir, "%u", *currentPeople.Objects[userIndex].UID)
			}

			if currentPeople.Objects[userIndex].UserPassword == nil {
				if config.Defaults.UserPassword == nil {
					return fmt.Errorf("cannot read object: user_password not set in object and no default is defined")
				}
				// construct from default
				currentPeople.Objects[userIndex].UserPassword = new(string)
				// set username
				*currentPeople.Objects[userIndex].UserPassword = strings.ReplaceAll(*config.Defaults.UserPassword, "%u", *currentPeople.Objects[userIndex].UID)
			}

			// verify the same user isn't configured multiple times
			for i = range knownUsers {
				if knownUsers[i] == *currentPeople.Objects[userIndex].UID {
					return fmt.Errorf("user with username '%s' is configured multiple times", *currentPeople.Objects[userIndex].UID)
				}
			}

			// adding id to now known users
			knownUsers = append(knownUsers, *currentPeople.Objects[userIndex].UID)
			glg.Debugf("loaded local user with DN %s", currentPeople.Objects[userIndex].dn)
		}

		// add loaded file to global list of know user objects
		localPeople[currentPeople.dn] = *currentPeople
	}

	glg.Infof("done reading people configuration file")
	return nil
}

// readGroupConfiguration reads a group config dir and performs basic sanity checks
func readGroupConfiguration() error {
	var (
		err          error
		files        []string
		currentFile  string
		currentGroup *groupOfNames
		yamlFile     []byte
		i            int
		j            int
		match        int
		dn           string
		pathPieces   []string
		relPath      string
	)

	glg.Infof("reading group configuration file")

	err = filepath.Walk(filepath.Join(*config.GroupDir),
		func(path string, info os.FileInfo, err error) error {
			var (
				ou *organizationalUnit
			)

			if err != nil {
				return err
			}

			if info.IsDir() {
				// check if dir needs added as a new OU
				// split path into dirs, each one will be its own OU in LDAP

				relPath, _ = filepath.Rel(*config.GroupDir, path)

				if relPath != "." {
					// only creating an OU of the path is not the people_dir

					pathPieces = strings.Split(relPath, "/")

					ou = new(organizationalUnit)
					ou.cn = info.Name()
					ou.description = "Managed by Monban"

					if len(pathPieces) <= 1 {
						ou.dn = fmt.Sprintf("ou=%s,%s", ou.cn, groupDN)
					} else {
						ou.dn = fmt.Sprintf("ou=%s,%s,%s", ou.cn, generateOUDN(pathPieces[:len(pathPieces)-1]), groupDN)
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
		glg.Infof("reading group config file %s", currentFile)

		yamlFile, err = ioutil.ReadFile(currentFile)
		if err != nil {
			return fmt.Errorf("failed to load group config file: %s", err.Error())
		}

		// currentGroup needs to be reset before every Unmarshal
		currentGroup = new(groupOfNames)
		err = yaml.Unmarshal(yamlFile, currentGroup)
		if err != nil {
			return fmt.Errorf("failed to parse group config file: %s", err.Error())
		}

		// CN wasn't explicitly set, using filename instead
		if currentGroup.CN == "" {
			currentGroup.CN = filepath.Base(currentFile)
		}

		// generate DN
		relPath, _ = filepath.Rel(*config.GroupDir, currentFile)
		pathPieces = strings.Split(relPath, "/")
		if len(pathPieces) <= 1 {
			currentGroup.dn = fmt.Sprintf("cn=%s,%s", currentGroup.CN, groupDN)
		} else {
			currentGroup.dn = fmt.Sprintf("cn=%s,%s,%s", currentGroup.CN, generateOUDN(pathPieces[:len(pathPieces)-1]), groupDN)
		}
		glg.Debugf("loaded local group with DN %s", currentGroup.dn)

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
					return fmt.Errorf("duplicated member entry with uid %s in group '%s'", currentGroup.Members[i], currentGroup.dn)
				}
			}

			// add member with DN not UID

			glg.Debugf("loaded group member with uid %s", currentGroup.Members[i])
		}

		// verify members also exist within config
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
				return fmt.Errorf("member uid %s in group %s doesn't exist as user object", currentGroup.Members[i], currentGroup.dn)
			}
		}

		// add group to global list of groups
		localGroups[currentGroup.dn] = *currentGroup
		glg.Debugf("loaded local group with DN %s", currentGroup.dn)
	}

	glg.Infof("done reading group configuration file")
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
