// Package models implements structures and custom data types used within Mondan.
package models

import (
	"github.com/go-ldap/ldap/v3"
)

const (
	ObjectTypePosixAccount = iota
	ObjectTypePosixGroup
	ObjectTypeGroupOfNames
	ObjectTypeOrganisationalUnit
	ObjectTypeSudoRole
)

const (
	TaskTypeCreate = iota
	TaskTypeUpdate
	TaskTypeDelete
	TaskTypeAddMember
	TaskTypeDeleteMember
)

// Runtime contains variables containing runtime information to be shared between modules.
type Runtime struct {
	// BasePath is the absolut path to the monban root dir
	BasePath string
	// Config points to the main config struct
	Config *Configuration
	// ConfigFile contains the main config file path
	ConfigFile string
	// GroupDN is the root dn in wich groups exist in
	GroupDN string
	// LatestUID contains global highest UIDNumber seen in peopleDN
	LatestUID int
	// LdapCon is the global LDAP connection struct
	LdapCon *ldap.Conn
	// LogLevel holds a string describing a desired log level
	LogLevel string
	// PeopleDN is the root dn in which people groups exist in
	PeopleDN string
	// Colo defines if diff output is colorful or not
	UseColor bool
	// Userdn contains the user dn to use for binding to LDAP
	UserDN string
	// UserPassword contains the user password to use for binding to LDAP
	UserPassword string
}

// Configuration contains general configuration data
type Configuration struct {
	HostURI             *string `yaml:"host_uri"`
	UserDN              *string `yaml:"user_dn"`
	UserPassword        *string `yaml:"user_password"`
	EnableSSHPublicKeys *bool   `yaml:"enable_ssh_public_keys"`
	GroupDir            *string `yaml:"group_dir"`
	PeopleDir           *string `yaml:"people_dir"`
	RootDN              *string `yaml:"root_dn"`
	PeopleRDN           *string `yaml:"people_rdn"`
	GroupRDN            *string `yaml:"group_rdn"`
	GenerateUID         bool    `yaml:"generate_uid"`
	MinUID              int     `yaml:"min_uid"`
	MaxUID              int     `yaml:"max_uid"`
	EnableSudo          bool    `yaml:"enable_sudo"`
	// contains the default values (or patterns) used when an object doesn't explicitly defines them
	Defaults struct {
		DisplayName  *string   `yaml:"display_name"`
		LoginShell   *string   `yaml:"login_shell"`
		Mail         *string   `yaml:"mail"`
		HomeDir      *string   `yaml:"home_dir"`
		UserPassword *string   `yaml:"user_password"`
		Sudo         *SudoRole `yaml:"sudo"`
	} `yaml:"defaults"`
}

// PosixGroup contains information about a LDAP user group object
type PosixGroup struct {
	DN          string         `yaml:"-"`
	CN          string         `yaml:"cn"`
	GIDNumber   *int           `yaml:"gid_number"`
	Description string         `yaml:"description"`
	Objects     []PosixAccount `yaml:"objects"`
}

// PosixAccount represents a LDAP user object
//
// also used as actionTask.data
// create task: nil ptr means value will not be set
// change task: nil ptr means no change of that attribute
// delete task: only CN is set
type PosixAccount struct {
	DN           string  `yaml:"-"`
	UID          *string `yaml:"username"` // also CN
	UIDNumber    *int    `yaml:"uid_number"`
	GIDNumber    *int    `yaml:"gid_number"`
	GivenName    *string `yaml:"given_name"`
	Surname      *string `yaml:"surname"`
	DisplayName  *string `yaml:"display_name"`
	LoginShell   *string `yaml:"login_shell"`
	Mail         *string `yaml:"mail"`
	SSHPublicKey *string `yaml:"ssh_public_key"`
	HomeDir      *string `yaml:"home_dir"`
	UserPassword *string `yaml:"user_password"`
}

// GroupOfNames contains information about a groups with members
type GroupOfNames struct {
	DN          string   `yaml:"-"` // internal only
	CN          string   `yaml:"cn"`
	Description string   `yaml:"description"`
	Members     []string `yaml:"members"`
}

// ActionTask defines a task to execute against a ldap target
type ActionTask struct {
	// dn is not nil when an object is to be deleted or a member gets added/deleted
	DN *string
	// data can contain different structs depending on the action and object type
	//
	// objectType == objectTypePosixAccount
	//		data is posixAccount struct
	// objectType == objectTypePosixGroup
	//    data is posixGroup struct
	// objectType == objectTypeGroupOfNames
	//    create, update: data is groupOfNames struct
	//    add or delete member: (string) value to add or remove
	// objectType == objectTypeOrganisationalUnit
	//		create: organizationalUnit
	// objectType == objectTypeSudoRole
	//    create, update: data is sudoRole struct
	Data interface{}
}

// see https://github.com/go-yaml/yaml/issues/100
type stringArray []string

func (a *stringArray) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var multi []string
	err := unmarshal(&multi)
	if err != nil {
		var single string
		err := unmarshal(&single)
		if err != nil {
			return err
		}
		*a = []string{single}
	} else {
		*a = multi
	}
	return nil
}

// SudoRole defines a LDAP sudoRule object
type SudoRole struct {
	DN             string      `yaml:"dn"`
	CN             string      `yaml:"name"`
	Description    string      `yaml:"description"`
	SudoUser       stringArray `yaml:"sudo_user"`
	SudoHost       stringArray `yaml:"sudo_host"`
	SudoCommand    stringArray `yaml:"sudo_command"`
	SudoOption     stringArray `yaml:"sudo_option"`
	SudoRunAsUser  stringArray `yaml:"sudo_run_as_user"`
	SudoRunAsGroup stringArray `yaml:"sudo_run_as_group"`
	SudoNotBefore  stringArray `yaml:"sudo_not_before"`
	SudoNotAfter   stringArray `yaml:"sudo_not_after"`
	SudoOrder      *int        `yaml:"sudo_order"`
}

// Sudoers describes a SUDOers file
type Sudoers struct {
	DisableDefaults bool       `yaml:"disable_defaults"`
	Roles           []SudoRole `yaml:"roles"`
}

// OrganizationalUnit defines a LDAP OU object
type OrganizationalUnit struct {
	DN          string
	CN          string
	Description string
}
