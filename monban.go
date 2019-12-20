package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/go-ldap/ldap/v3"
	"github.com/kpango/glg"
	"github.com/urfave/cli/v2"
)

const (
	objectTypePosixAccount = iota
	objectTypePosixGroup
	objectTypeGroupOfNames
	objectTypeOrganisationalUnit
)

const (
	taskTypeCreate = iota
	taskTypeUpdate
	taskTypeDelete
	taskTypeAddMember
	taskTypeDeleteMember
)

// global vars
var (
	// logLevel holds a string describing a desired log level
	logLevel string
	// config points to the main config struct
	config *configuration
	// configFile contains the main config file path
	configFile string
	// basePath is the absolut path to the monban root dir
	basePath string
	// userdn contains the user dn to use for binding to LDAP
	userDN string
	// userPassword contains the user password to use for binding to LDAP
	userPassword string
	// localPeople holds a map of all users and their organisational parent group
	localPeople map[string]posixGroup
	// ldapPeople holds a map of all users and their organisational parent group existing in LDAP
	ldapPeople map[string]posixGroup
	// localGroups holds a map of all unixGroups and their members
	localGroups map[string]groupOfNames
	// ldapGroups holds a map of all groups and their members existing in LDAP
	ldapGroups map[string]groupOfNames
	// global LDAP connection struct
	ldapCon *ldap.Conn
	// global highest UIDNumber seen below peopleDN
	latestUID int
	// peopledn is the root dn in which people groups exist in
	peopleDN string
	// groupdn is the root dn in wich groups exist in
	groupDN string
	// taskList contains all tasks to be executed in order to sync LDAP with configured values
	taskList []*actionTask

	localOUs []*organizationalUnit
	ldapOUs  []*organizationalUnit

	// filled at build time
	Version string
	Commit  string
	Date    string
)

func main() {
	var (
		app      *cli.App
		err      error
		compiled time.Time
	)

	compiled, _ = time.Parse(time.RFC1123, Date)

	app = &cli.App{
		Name:     "monban",
		Usage:    "manage LDAP users and their group memberships in YAML files",
		Version:  fmt.Sprintf("%s (commit %s, compiled %s)", Version, Commit, compiled.Format(time.RFC1123)),
		Compiled: compiled,
		Authors: []*cli.Author{
			&cli.Author{
				Name:  "xoc",
				Email: "xoc@4xoc.com",
			},
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "log-level",
				Aliases:     []string{"l"},
				Value:       "warning",
				Usage:       "set log level [debug|info|warning|error] (default: warning)",
				EnvVars:     []string{"MONBAN_LOG_LEVEL"},
				Destination: &logLevel,
			},
			&cli.StringFlag{
				Name:        "config",
				Aliases:     []string{"c"},
				Value:       "config.yaml",
				Usage:       "read main configuration from `FILE`",
				EnvVars:     []string{"MONBAN_CONFIG_FILE"},
				Destination: &configFile,
			},
			&cli.StringFlag{
				Name:        "user_dn",
				Aliases:     []string{"u"},
				Usage:       "user dn to bind to",
				EnvVars:     []string{"MONBAN_USER_DN"},
				Destination: &userDN,
			},
			&cli.StringFlag{
				Name:        "user_pass",
				Aliases:     []string{"p"},
				Usage:       "user passwort to bind with",
				EnvVars:     []string{"MONBAN_USER_PASSWORD"},
				Destination: &userPassword,
			},
		},
		Before: func(c *cli.Context) error {
			// set log level
			switch logLevel {
			case "debug":
				glg.Get().
					SetMode(glg.STD).
					SetLevelMode(glg.DEBG, glg.STD).
					SetLevelMode(glg.INFO, glg.STD).
					SetLevelMode(glg.WARN, glg.STD).
					SetLevelMode(glg.ERR, glg.STD).
					SetLevelMode(glg.FATAL, glg.STD)

				glg.Warnf("debug logging enabled; be aware that secrets like passwords will be printed in clear text!")

			case "info":
				glg.Get().
					SetMode(glg.STD).
					SetLevelMode(glg.DEBG, glg.NONE).
					SetLevelMode(glg.INFO, glg.STD).
					SetLevelMode(glg.WARN, glg.STD).
					SetLevelMode(glg.ERR, glg.STD).
					SetLevelMode(glg.FATAL, glg.STD)

			case "warning":
				glg.Get().
					SetMode(glg.STD).
					SetLevelMode(glg.DEBG, glg.NONE).
					SetLevelMode(glg.INFO, glg.NONE).
					SetLevelMode(glg.WARN, glg.STD).
					SetLevelMode(glg.ERR, glg.STD).
					SetLevelMode(glg.FATAL, glg.STD)
			case "error":
				glg.Get().
					SetMode(glg.STD).
					SetLevelMode(glg.DEBG, glg.NONE).
					SetLevelMode(glg.INFO, glg.NONE).
					SetLevelMode(glg.WARN, glg.NONE).
					SetLevelMode(glg.ERR, glg.STD).
					SetLevelMode(glg.FATAL, glg.STD)

			default:
				glg.Get().
					SetMode(glg.STD).
					SetLevelMode(glg.DEBG, glg.NONE).
					SetLevelMode(glg.INFO, glg.NONE).
					SetLevelMode(glg.WARN, glg.STD).
					SetLevelMode(glg.ERR, glg.STD).
					SetLevelMode(glg.FATAL, glg.STD)

				glg.Warnf("unknown log-level %s, using warning instead", logLevel)
			}

			return nil
		},
		Commands: []*cli.Command{
			&cli.Command{
				Name:    "sync",
				Aliases: []string{"s"},
				Usage:   "synchronize changes to LDAP host",
				Action: func(c *cli.Context) error {
					// sync order
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
					// 12. delete groupOfNames
					// 13. delete OUs

					var (
						err    error
						i      int
						ouList []string
					)

					if err = initConfig(c); err != nil {
						return err
					}

					if err = initLDAP(); err != nil {
						return err
					}

					if err = compareOUs(); err != nil {
						return fmt.Errorf("failed to compare organizationalUnit objects: %s", err.Error())
					}

					if err = comparePosixGroups(); err != nil {
						return fmt.Errorf("failed to compare posixGroup objects: %s", err.Error())
					}

					if err = compareGroupOfNames(); err != nil {
						return fmt.Errorf("failed to compare groupOfNames objects: %s", err.Error())
					}

					if len(taskList) == 0 {
						glg.Infof("Data comparison complete. No changes to be synced.")
						return nil
					} else {
						glg.Infof("Data comparison complete. %d changes will be synced", len(taskList))
					}

					// 1. create OUs
					glg.Infof("creating intermediate organizationalUnit objects")
					for i = range taskList {
						if taskList[i].objectType == objectTypeOrganisationalUnit &&
							taskList[i].taskType == taskTypeCreate {
							// order is ensured by originally sorting all OUs by shortest first (see compareOUs())
							if err = ldapCreateOrganisationalUnit(taskList[i].data.(*organizationalUnit)); err != nil {
								return err
							}
						}
					}

					// 2. delete group memberships
					glg.Infof("deleting obsolete groupOfNames memberships")
					for i = range taskList {
						if taskList[i].objectType == objectTypeGroupOfNames &&
							taskList[i].taskType == taskTypeDeleteMember {
							if err = ldapDeleteGroupOfNamesMember(taskList[i].dn, taskList[i].data.(string)); err != nil {
								return err
							}
						}
					}

					// 3. delete posixAccoounts
					glg.Infof("deleting obsolete posixAccount objects")
					for i = range taskList {
						if taskList[i].objectType == objectTypePosixAccount &&
							taskList[i].taskType == taskTypeDelete {
							if err = ldapDeletePosixAccount(taskList[i].dn); err != nil {
								return err
							}
						}
					}

					// 4. create posixGroups
					glg.Infof("creating new posixGroup objects")
					for i = range taskList {
						if taskList[i].objectType == objectTypePosixGroup &&
							taskList[i].taskType == taskTypeCreate {
							if err = ldapCreatePosixGroup(taskList[i].data.(posixGroup)); err != nil {
								return err
							}
						}
					}

					// 5. create posixGroups
					glg.Infof("deleting posixGroup objects")
					for i = range taskList {
						if taskList[i].objectType == objectTypePosixGroup &&
							taskList[i].taskType == taskTypeDelete {
							if err = ldapDeletePosixGroup(taskList[i].dn); err != nil {
								return err
							}
						}
					}

					// 6. update posixGroups
					glg.Infof("updating posixGroup objects")
					for i = range taskList {
						if taskList[i].objectType == objectTypePosixGroup &&
							taskList[i].taskType == taskTypeUpdate {
							if err = ldapUpdatePosixGroup(taskList[i].data.(*posixGroup)); err != nil {
								return err
							}
						}
					}

					// 7. create posixAccounts
					glg.Infof("creating new posixAccount objects")
					for i = range taskList {
						if taskList[i].objectType == objectTypePosixAccount &&
							taskList[i].taskType == taskTypeCreate {
							if err = ldapCreatePosixAccount(taskList[i].data.(*posixAccount)); err != nil {
								return err
							}
						}
					}

					// 8. update posixAccounts
					glg.Infof("updating posixAccount objects")
					for i = range taskList {
						if taskList[i].objectType == objectTypePosixAccount &&
							taskList[i].taskType == taskTypeUpdate {
							if err = ldapUpdatePosixAccount(taskList[i].data.(*posixAccount)); err != nil {
								return err
							}
						}
					}

					// 9. create groupOfNames
					glg.Infof("creating new groupOfNames objects")
					for i = range taskList {
						if taskList[i].objectType == objectTypeGroupOfNames &&
							taskList[i].taskType == taskTypeCreate {
							if err = ldapCreateGroupOfNames(taskList[i].data.(groupOfNames)); err != nil {
								return err
							}
						}
					}

					// 10. update groupOfNames
					glg.Infof("updating groupOfNames objects")
					for i = range taskList {
						if taskList[i].objectType == objectTypeGroupOfNames &&
							taskList[i].taskType == taskTypeUpdate {
							if err = ldapUpdateGroupOfNames(taskList[i].data.(*groupOfNames)); err != nil {
								return err
							}
						}
					}

					// 11. create group memberships
					glg.Infof("creating new groupOfNames memberships")
					for i = range taskList {
						if taskList[i].objectType == objectTypeGroupOfNames &&
							taskList[i].taskType == taskTypeAddMember {
							if err = ldapAddGroupOfNamesMember(taskList[i].dn, taskList[i].data.(string)); err != nil {
								return err
							}
						}
					}

					// 12. delete groupOfNames
					glg.Infof("deleteing groupOfNames objects")
					for i = range taskList {
						if taskList[i].objectType == objectTypeGroupOfNames &&
							taskList[i].taskType == taskTypeDelete {
							if err = ldapDeleteGroupOfNames(taskList[i].dn); err != nil {
								return err
							}
						}
					}

					// 13. delete OUs
					glg.Infof("deleting intermediate organizationalUnit objects")
					for i = range taskList {
						if taskList[i].objectType == objectTypeOrganisationalUnit &&
							taskList[i].taskType == taskTypeDelete {
							// order of the tasks MUST be ensured
							ouList = append(ouList, taskList[i].dn)
						}
					}

					// sort ouList; longest dn first to start further down the three
					sort.Slice(ouList, func(i, j int) bool {
						return len(ouList[i]) > len(ouList[j])
					})

					// now actually delete OU
					for i = range ouList {
						if err = ldapDeleteOrianisationalUnit(ouList[i]); err != nil {
							return err
						}
					}

					glg.Info("Sync completed.")

					return nil
				},
			},
			&cli.Command{
				Name:    "diff",
				Aliases: []string{"d"},
				Usage:   "show diff between configured and existsing users/groups",
				Action: func(c *cli.Context) error {
					var (
						i   int
						err error
					)

					if err = initConfig(c); err != nil {
						return err
					}

					if err = initLDAP(); err != nil {
						return err
					}

					if err = compareOUs(); err != nil {
						return fmt.Errorf("failed to compare organizationalUnit objects: %s", err.Error())
					}

					if err = comparePosixGroups(); err != nil {
						return fmt.Errorf("failed to compare posixGroup objects: %s", err.Error())
					}

					if err = compareGroupOfNames(); err != nil {
						return fmt.Errorf("failed to compare groupOfNames objects: %s", err.Error())
					}

					glg.Infof("Data comparison complete. %d changes detected", len(taskList))

					if len(taskList) == 0 {
						return nil
					}

					// pretty print changes
					fmt.Printf("\n ==>> OrganisationalUnit Objects <<==\n")
					fmt.Printf("\n     == New OrganisationalUnit Objects ==\n")
					for i = range taskList {
						if taskList[i].objectType == objectTypeOrganisationalUnit &&
							taskList[i].taskType == taskTypeCreate {

							fmt.Printf("\n       -------\n       DN: %s\n       -------\n",
								taskList[i].data.(*organizationalUnit).dn)
						}
					}
					fmt.Printf("\n     == Deleted OrganisationalUnit Objects ==\n")
					for i = range taskList {
						if taskList[i].objectType == objectTypeOrganisationalUnit && taskList[i].taskType == taskTypeDelete {

							fmt.Printf("\n       -------\n       DN: %s\n       -------\n",
								taskList[i].dn)
						}
					}

					fmt.Printf("\n ==>> PosixGroup Objects <<==\n")
					fmt.Printf("\n     == New PosixGroup Objects ==\n")
					for i = range taskList {
						if taskList[i].objectType == objectTypePosixGroup && taskList[i].taskType == taskTypeCreate {

							fmt.Printf("\n       -------\n       DN:           %s\n       GID Number:   %d\n       Description:  %s\n       -------\n",
								taskList[i].data.(posixGroup).dn,
								*taskList[i].data.(posixGroup).GIDNumber,
								taskList[i].data.(posixGroup).Description)
						}
					}

					fmt.Printf("\n     == Updated Group Objects ==\n")
					for i = range taskList {
						if taskList[i].objectType == objectTypePosixGroup &&
							taskList[i].taskType == taskTypeUpdate {

							fmt.Printf("\n       -------\n       DN:           %s\n       NEW VALUES:\n",
								taskList[i].data.(*posixGroup).CN)

							if taskList[i].data.(*posixGroup).GIDNumber != nil {
								fmt.Printf("         GID Number:     %d\n", *taskList[i].data.(*posixGroup).GIDNumber)
							}

							if taskList[i].data.(*posixGroup).Description != "" {
								fmt.Printf("         Description:    %s\n", taskList[i].data.(*posixGroup).Description)
							}
							fmt.Printf("       -------\n")
						}
					}

					fmt.Printf("\n     == Deleted Group Objects ==\n")
					for i = range taskList {
						if taskList[i].objectType == objectTypePosixGroup &&
							taskList[i].taskType == taskTypeDelete {
							fmt.Printf("\n       -------\n       DN: %s\n       -------\n",
								taskList[i].dn)
						}
					}

					fmt.Printf("\n ==>> PosixAccount Objects <<==\n")
					fmt.Printf("\n     == New PosixAccount Objects ==\n")
					for i = range taskList {
						if taskList[i].objectType == objectTypePosixAccount &&
							taskList[i].taskType == taskTypeCreate {

							fmt.Printf("\n       -------\n       Username:    %s\n       Given Name:  %s\n       Last Name:   %s\n       Group:       %s\n       -------\n",
								*taskList[i].data.(*posixAccount).UID,
								*taskList[i].data.(*posixAccount).GivenName,
								*taskList[i].data.(*posixAccount).Surname,
								strings.Join(strings.Split(taskList[i].dn, ",")[1:], ","))
						}
					}

					fmt.Printf("\n     == Updated Objects ==\n")
					for i = range taskList {
						if taskList[i].objectType == objectTypePosixAccount &&
							taskList[i].taskType == taskTypeUpdate {

							fmt.Printf("\n       -------\n       Username: %s\n       NEW VALUES:\n", strings.Split(taskList[i].dn, ",")[0][4:])

							if taskList[i].data.(*posixAccount).GivenName != nil {
								fmt.Printf("         Given Name:     %s\n", *taskList[i].data.(*posixAccount).GivenName)
							}

							if taskList[i].data.(*posixAccount).Surname != nil {
								fmt.Printf("         Last Name:      %s\n", *taskList[i].data.(*posixAccount).Surname)
							}

							if taskList[i].data.(*posixAccount).DisplayName != nil {
								fmt.Printf("         Display Name:   %s\n", *taskList[i].data.(*posixAccount).DisplayName)
							}

							if taskList[i].data.(*posixAccount).LoginShell != nil {
								fmt.Printf("         Login Shell:    %s\n", *taskList[i].data.(*posixAccount).LoginShell)
							}

							if taskList[i].data.(*posixAccount).HomeDir != nil {
								fmt.Printf("         Home Dir:    %s\n", *taskList[i].data.(*posixAccount).HomeDir)
							}

							if taskList[i].data.(*posixAccount).Mail != nil {
								fmt.Printf("         Mail:           %s\n", *taskList[i].data.(*posixAccount).Mail)
							}

							if taskList[i].data.(*posixAccount).SSHPublicKey != nil {
								if *taskList[i].data.(*posixAccount).SSHPublicKey == "" {

									fmt.Printf("         SSH Public Key: *to be deleted*\n")
								} else {

									fmt.Printf("         SSH Public Key: %s\n", *taskList[i].data.(*posixAccount).SSHPublicKey)
								}
							}

							if taskList[i].data.(*posixAccount).UserPassword != nil {
								fmt.Printf("         User Password:  ********\n")
							}

							fmt.Printf("       -------\n")
						}
					}

					fmt.Printf("\n     == Deleted Objects ==\n")
					for i = range taskList {
						if taskList[i].objectType == objectTypePosixAccount &&
							taskList[i].taskType == taskTypeDelete {

							fmt.Printf("\n       -------\n       Username: %s\n       Given Name:  %s\n       Last Name:   %s\n       Group:       %s\n       -------\n",
								*taskList[i].data.(*posixAccount).UID,
								*taskList[i].data.(*posixAccount).GivenName,
								*taskList[i].data.(*posixAccount).Surname,
								strings.Join(strings.Split(taskList[i].dn, ",")[1:], ","))
						}
					}

					fmt.Printf("\n ==>> GroupOfNames Objects <<==\n")
					fmt.Printf("\n     == New GroupOfNames Object ==\n")
					for i = range taskList {
						if taskList[i].objectType == objectTypeGroupOfNames &&
							taskList[i].taskType == taskTypeCreate {

							fmt.Printf("\n       -------\n       DN: %s\n       Description:  %s\n       -------\n",
								taskList[i].data.(groupOfNames).dn,
								taskList[i].data.(groupOfNames).Description)
						}
					}

					fmt.Printf("\n     == Updated GroupOfNames Object ==\n")
					for i = range taskList {
						if taskList[i].objectType == objectTypeGroupOfNames &&
							taskList[i].taskType == taskTypeUpdate {

							fmt.Printf("\n       -------\n       DN: %s\n       NEW VALUES:\n", taskList[i].dn)

							if taskList[i].data.(*groupOfNames).Description != "" {
								fmt.Printf("         Description:     %s\n", taskList[i].data.(*groupOfNames).Description)
							}

							fmt.Printf("       -------\n")
						}
					}

					fmt.Printf("\n     == Deleted GroupOfNames Object ==\n")
					for i = range taskList {
						if taskList[i].objectType == objectTypeGroupOfNames &&
							taskList[i].taskType == taskTypeDelete {

							fmt.Printf("\n       -------\n       DN: %s\n       -------\n",
								taskList[i].dn)
						}
					}

					fmt.Printf("\n ==>> GroupOfNames Memberships <<==\n")
					fmt.Printf("\n     == New GroupOfNames Members ==\n")
					for i = range taskList {
						if taskList[i].objectType == objectTypeGroupOfNames &&
							taskList[i].taskType == taskTypeAddMember {

							fmt.Printf("\n       -------\n       Username:   %s\n       Group:      %s\n       -------\n",
								strings.Split(taskList[i].data.(string), ",")[0][4:],
								taskList[i].dn)
						}
					}

					fmt.Printf("\n     == Deleted Members ==\n")
					for i = range taskList {
						if taskList[i].objectType == objectTypeGroupOfNames &&
							taskList[i].taskType == taskTypeDeleteMember {

							fmt.Printf("\n       -------\n       User Object: %s\n       Group:       %s\n       -------\n",
								strings.Split(taskList[i].data.(string), ",")[0][4:],
								taskList[i].dn)
						}
					}

					fmt.Printf("\n")

					return nil
				},
			},
			&cli.Command{
				Name:    "validate",
				Aliases: []string{"v"},
				Usage:   "validate config files and check for proper syntax",
				Action: func(c *cli.Context) error {
					var err error

					if err = initConfig(c); err != nil {
						return err
					}

					glg.Infof("validation complete - things seem okay *terms and conditions apply*")

					return nil
				},
			},
			&cli.Command{
				Name:    "audit",
				Aliases: []string{"a"},
				Usage:   "displays all (in file) configured user objects and group membership for easy audit",
				Action: func(c *cli.Context) error {
					var (
						err         error
						dn          string
						index       int
						dnFragments []string
						dn2         string
						index2      int
					)

					if err = initConfig(c); err != nil {
						return err
					}

					glg.Infof("!! drifts between files and LDAP are not displayed")

					fmt.Printf("\n\n====== START AUDIT ======")

					for dn = range localPeople {
						dnFragments = strings.Split(dn, ",")

						fmt.Printf("\n  === %s ===\n\n", dnFragments[0][3:])

						for index = range localPeople[dn].Objects {
							fmt.Printf("    -------\n")

							fmt.Printf("    Username: %s\n    Given Name: %s\n    Last Name: %s\n    Memberships:\n",
								*localPeople[dn].Objects[index].UID,
								*localPeople[dn].Objects[index].GivenName,
								*localPeople[dn].Objects[index].Surname)

							for dn2 = range localGroups {
								for index2 = range localGroups[dn2].Members {
									if *localPeople[dn].Objects[index].UID == localGroups[dn2].Members[index2] {
										fmt.Printf("      %s\n", dn2)
									}
								}
							}

							fmt.Printf("    -------\n")
						}
					}

					fmt.Printf("====== END AUDIT ======\n")

					return nil
				},
			},
		},
	}

	err = app.Run(os.Args)
	if err != nil {
		glg.Fatal(err)
	}
}

// initConfig reads config files
func initConfig(c *cli.Context) error {
	var err error

	err = readConfiguration(c)
	if err != nil {
		return fmt.Errorf("failed to read main config file: %s", err.Error())
	}

	err = readPeopleConfiguration()
	if err != nil {
		return fmt.Errorf("failed to read people configuration file: %s", err.Error())
	}

	err = readGroupConfiguration()
	if err != nil {
		return fmt.Errorf("failed to read groups configuration file: %s", err.Error())
	}

	return nil
}

// initLDAP connects to LDAP, authenticates and reads object details
func initLDAP() error {
	var err error

	ldapCon, err = ldapConnect()
	if err != nil {
		return fmt.Errorf("failed to connect to LDAP host: %s", err.Error())
	}

	err = ldapLoadPeople()
	if err != nil {
		return fmt.Errorf("failed to load people objects from LDAP: %s", err.Error())
	}

	err = ldapLoadGroups()
	if err != nil {
		return fmt.Errorf("failed to load group objects from LDAP: %s", err.Error())
	}

	return nil
}
