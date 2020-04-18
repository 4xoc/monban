package main

import (
	"fmt"
	"os"
	"time"

	"github.com/4xoc/monban/models"

	"github.com/kpango/glg"
	"github.com/urfave/cli/v2"
)

// global vars
var (
	rt models.Runtime
	// localPeople holds a map of all users and their organisational parent group
	localPeople map[string]models.PosixGroup = make(map[string]models.PosixGroup)
	// ldapPeople holds a map of all users and their organisational parent group existing in LDAP
	ldapPeople map[string]models.PosixGroup = make(map[string]models.PosixGroup)
	// localGroups holds a map of all unixGroups and their members
	localGroups map[string]models.GroupOfNames = make(map[string]models.GroupOfNames)
	// ldapGroups holds a map of all groups and their members existing in LDAP
	ldapGroups map[string]models.GroupOfNames = make(map[string]models.GroupOfNames)
	// taskList contains all tasks to be executed in order to sync LDAP with configured values
	// it is mapped by object type and action
	taskList map[int]map[int][]*models.ActionTask = make(map[int]map[int][]*models.ActionTask)
	// taskListCount contains the number of planned changes to be synced
	taskListCount int
	// localOUs holds all locally found OUs
	localOUs []*models.OrganizationalUnit
	// ldapOUs holds all existing OUs on target
	ldapOUs []*models.OrganizationalUnit
	// localSUDOers contains a map of ou=SUDOers DN and it's set of roles configured in file
	localSudoRoles []models.SudoRole
	// ldapSUDOers contains a map of ou=SUDOers DN and it's set of roles on LDAP target
	ldapSudoRoles []models.SudoRole

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
				Destination: &rt.LogLevel,
			},
			&cli.StringFlag{
				Name:        "config",
				Aliases:     []string{"c"},
				Value:       "config.yml",
				Usage:       "read main configuration from `FILE`",
				EnvVars:     []string{"MONBAN_CONFIG_FILE"},
				Destination: &rt.ConfigFile,
			},
			&cli.StringFlag{
				Name:        "user_dn",
				Aliases:     []string{"u"},
				Usage:       "user dn to bind to",
				EnvVars:     []string{"MONBAN_USER_DN"},
				Destination: &rt.UserDN,
			},
			&cli.StringFlag{
				Name:        "user_pass",
				Aliases:     []string{"p"},
				Usage:       "user passwort to bind with",
				EnvVars:     []string{"MONBAN_USER_PASSWORD"},
				Destination: &rt.UserPassword,
			},
		},
		Before: func(c *cli.Context) error {
			// set log level
			switch rt.LogLevel {
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

				glg.Warnf("unknown log-level %s, using warning instead", rt.LogLevel)
			}

			return nil
		},
		Commands: []*cli.Command{
			&cli.Command{
				Name:    "sync",
				Aliases: []string{"s"},
				Usage:   "synchronize changes to LDAP host",
				Action: func(c *cli.Context) error {
					var (
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

					if err = compareSudoRoles(); err != nil {
						return fmt.Errorf("failed to compare groupOfNames objects: %s", err.Error())
					}

					if taskListCount == 0 {
						glg.Infof("Data comparison complete. No changes to be synced.")
						return nil
					}

					return syncChanges()
				},
			},
			&cli.Command{
				Name:    "diff",
				Aliases: []string{"d"},
				Usage:   "show diff between configured and existsing users/groups",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:        "color",
						Aliases:     []string{"c"},
						Usage:       "when true output is displayed with colors",
						EnvVars:     []string{"MONBAN_DIFF_COLORS"},
						Value:       false,
						Destination: &rt.UseColor,
					},
				},
				Action: func(c *cli.Context) error {
					var (
						//i   int
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

					if err = compareSudoRoles(); err != nil {
						return fmt.Errorf("failed to compare sudoRole objects: %s", err.Error())
					}

					glg.Infof("Data comparison complete. %d changes detected", taskListCount)

					if taskListCount == 0 {
						return nil
					}

					prettyPrint()
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
				Usage:   "displays all in LDAP existing and Monban managed objects for easy audit",
				Action: func(c *cli.Context) error {
					var (
						err error
					)

					if err = initConfig(c); err != nil {
						return err
					}

					if err = initLDAP(); err != nil {
						return err
					}

					glg.Warnf("!! drifts between files and LDAP are not displayed !!")
					prettyPrintAudit()
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

	rt.LdapCon, err = ldapConnect()
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
