package main

import (
	"time"
)

// configuration contains general configuration data
type configuration struct {
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
	// contains the default values (or patterns) used when an object doesn't explicitly defines them
	Defaults struct {
		DisplayName  *string `yaml:"display_name"`
		LoginShell   *string `yaml:"login_shell"`
		Mail         *string `yaml:"mail"`
		HomeDir      *string `yaml:"home_dir"`
		UserPassword *string `yaml:"user_password"`
	} `yaml:"defaults"`
}

// posixGroup contains information about a LDAP user group object
type posixGroup struct {
	dn          string         `yaml:"-"`
	CN          string         `yaml:"cn"`
	GIDNumber   *int           `yaml:"gid_number"`
	Description string         `yaml:"description"`
	Objects     []posixAccount `yaml:"objects"`
}

// posixAccount represents a LDAP user object
//
// also used as actionTask.data
// create task: nil ptr means value will not be set
// change task: nil ptr means no change of that attribute
// delete task: only CN is set
type posixAccount struct {
	dn           string  `yaml:"-"`
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

// groupOfNames contains information about a groups with members
type groupOfNames struct {
	dn          string   `yaml:"-"` // internal only
	CN          string   `yaml:"cn"`
	Description string   `yaml:"description"`
	Members     []string `yaml:"members"`
}

// actionTask defines a task to execute against a ldap target
type actionTask struct {
	dn         string
	objectType int
	taskType   int
	// data can contain different structs depending on the action and object type
	//
	// objectType == objectTypePosixAccount
	//		data is posixAccount struct
	// objectType == objectTypePosixGroup
	//    data is posixGroup struct
	// objectType == objectTypeGroupOfNames
	//    create, delete, update: data is groupOfNames struct
	//    add or delete member: (string) value to add or remove
	// objectType == objectTypeOrganisationalUnit
	//		create: organizationalUnit
	// 		delete: nil (but dn set above)
	data interface{}
}

// sudoersRule defines a LDAP SUDOers object
type sudoersRule struct {
	SudoUser      *string   `yaml:"sudo_user"`
	SudoHost      *string   `yaml:"sudo_host"`
	SudoCommand   *string   `yaml:"sudo_command"`
	SudoOption    *string   `yaml:"sudo_option"`
	SudoRunAsUser *string   `yaml:"sudo_run_as_user"`
	SudoNotBefore time.Time `yaml:"sudo_not_before"`
	SudoNotAfter  time.Time `yaml:"sudo_not_after"`
	SudoOrder     *int      `yaml:"sudo_order"`
}

// organizationalUnit defines a LDAP OU object
type organizationalUnit struct {
	dn          string
	cn          string
	description string
}
