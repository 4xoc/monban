# Monban - Automated LDAP Account Management

Monban provides a way to synchronize any ldap-compliant system with configurations defined in YAML files to create 
user objects as well as manage their group memberships. While it is very flexible to accommodate (hopefully) any
usecase it also allows for minimal configuration using default values that support templating to generate object
attributes.

This tool is supposed to be a very easy and yet powerfull way to create user objects (you need a working ldap, duh)
and add them to the groups as per configuration file. Monban also detects drifts between config and ldap and will sync
ldap to be identical to what is stated in the files. This includes adding new objects (users or groups) as well as
deleting objects where required.

Monban maintains two kinds of groups: PosixGroup and generic OrganisationalUnit. While the first ist used for SSH based
logins, the latter is more common for services that map service-specific roles or groups to members of the OU.

Since Monban considers the configuration files as the single source of truth, it will take actions to create, modify and delete ojects within the sub-trees defined (see configuration below). To prevent unexpected behaviour, it is highly recommended to not mix Mondan managed sub-trees and other, manually or otherwise manages objects. In short, give Monban it's own sub-tree to work in. This will also prevent Monban from deleteing objects used by something else.

## Requirements

* working LDAP system
* nis schema
* (optional, but recommended) [ssh schema](resources/ssh.schema) for SSH public keys

## Roadmap

* add support for SUDOers objects

## Usage

### Running Monban
Monban has some commands that can be executed:

* validate - basic syntax & sanity checks; it does not connect to any LDAP system
* diff - checks for differences between the configured and existing settings and displays them nicely
* sync - synchronizes the changes to LDAP and ensures that LDAP contains the same settings as defined in config files
* audit - Prints the current configs in a nicer way for easy access audits. This doesn't check for drifts beforehand so be sure that `diff` or `sync` has been run before as otherwise the audit output might be incorrect.

For more details on the commands and flags run `monban help`.

### Configuring Monban

There are different config files that Monban needs to run: general config, people config and group config files. All
files are written in YAML. Check out the [examples](examples/) to get started.

#### General Config

The general configuration file defined overall details that Monban need to run. It is also the entrypoint for Monban
to then find all other configuration files thay might exist.

| Attribute | Mandatory | Description |
|-----------|-----------|-------------|
| host_uri | yes | Full host URI used to connect to target LDAP system. |
| user_dn | no | DN of user to authenticate against LDAP target. Can be empty when `MONBAN_USER_DN` env var is set. |
| user_password | no | Password of user to authenticate against LDAP target. Can be empty when `MONBAN_USER_PASSWORD` env var is set. |
| enable_ssh_public_keys | no | Enables SSH public key support within Monban. LDAP target must support schema. Default: false |
| group_dir | yes | Path (relative to general config or absolute) to directory containing group config files. |
| people_dir | yes | Path (relative to general config or absolute) to directory containing people config files. |
| root_dn | yes | Schema root DN or root in which Monban is to place objects. |
| people_rdn | no | RDN of where to add people groups under. Must alreadt exist. Default: same as root_dn |
| group_rdn | no | RDN of where to add groups under. Must alreadt exist. Default: same as root_dn |
| generate_uid | no | When true Monban will automatically pick the next available UID for a user object. Default: false |
| min_uid | no | Min UID when generating UIDs. |
| max_uid | no | Max UID when generating UIDs, |
| defaults | no | Defines various default templates (see next table and [Templating](#templating)). |

**Default attributes:**

| Attribute | Description |
|-----------|-------------|
| display_name | Display name template. |
| login_shell | Login Shell template. |
| mail | Mail template. |
| home_dir | Home dir template. |
| user_password | User Password template. |

#### People Configuration

People/user objects are managed in files that describe each individual posixGroup. Those files must be stored in the
path that is set in `people_dir` in the main config file. Flat hierarchy is the typical layout but directories are also supported which also create new sub-trees in LDAP.

| Attribute | Mandatory | Description |
|-----------|-----------|-------------|
| cn | no | Common name of the posixGroup (only the name, no DN!). Filename is used if attribute is not set explicitly. |
| gid_number | yes |GID number of the unix group. |
| description | no | Description of the object. |
| objects | yes | List of user objects part of this posixGroup (see below). |

Objects itself are described with the following attributes. Note that attributes are only not mandatory when a default (see [Templating](#Templating) for that attribute is defined. A default can always be overwritten when explicitly defining the attribute in the objects.

| Attribute | Mandatory | Description |
|-----------|-----------|------------|
| username | yes | Username of the object. Is also UID and CN of the object. |
| given_name | yes | Given (first) name of the person. |
| surname | yes | Last name (surname) of the person. |
| display_name | no  | Pretty formatted name displayed in supported applications. |
| uid_number | no | UID number (interger). (see **NOTE** below) |
| gid_number | no | GID number (integer). (see **NOTE** below) |
| login_shell | no | Login shell to be dropped in on successful login. |
| mail | no | Email address of the person. |
| ssh_public_key | no  | (only if `enable_ssh_public_keys` is true) SSH public key string (any type) |
| home_dir | no | Home directory of the user. |
| user_password | no | LDAP supported password string (see https://www.openldap.org/doc/admin24/security.html: 14.4 Password Storage) |

**NOTE:** `uid_number` becomes mandatory when `generate_uid` is disabled in main config file.
**NOTE:** `gid_number` in the user object is always defaulted to the `gid_number` set in the people config file at the top (see first table). It is however possible to set a different `gid_number` for the objects. Only use different IDs if you know what you're doing!

**Example:**
```
cn: devops
gid_number: 1001
description: All DevOps team members.

objects:
  - username: johndoe
    given_name: John
    surname: Doe
    ssh_public_key: ecdsa-sha2-nistp256 AAAAbksdas...

  - username: peterpan
    given_name: Peter
    surname: Pan
    uid_number: 14356
    userPassword: "{SMD5}4QWGWZpj9GCmfuqEvm8HtZhZS6E=""
```

##### User Passwords

Obviously having cleartext passwords in the config file would be insane. LDAP by design supports various hashing algorithms that allow safely storing passwords in LDAP and also in file. This is surely a subject to discussion as to which way saving the data into LDAP is the safest. Monban doesn't try to force any way but doesn't in any way takes care of password security. It's recommended to use SASL passthrough or some other way of setting the user password into LDAP if the hashed options feels insecure to operators. When using SASL passthrough for passwords a default template can be used (`{SASL}%u`) and saslauthd needs to be configured on the LDAP system.

#### Group Configurations

Once people object exists those users can be added as members to groups.

| Attribute | Mandatory | Description |
|-----------|-----------|-------------|
| cn | no | Common name of the posixGroup (only the name, no DN!). Filename is used if attribute is not set explicitly. |
| description | no | Description of the object. |
| members | no | List of usernames configured as people. |

**NOTE:** Every groups automacally gets a dummy member added ("uid=MonbanDummyMember") to allow for empty groups. Ensure this dummy member does not exists and has no means to logging in!

**Example:**
```
cn: ldap-admin
description: This group gives total and unrestricted access to anything within openLDAP. With great power comes great responsibility!

members:
  - johndoe
  - peterpan
```
## Templating

Templating allows for dynamic attribute generation of people objects. Attributes that follow a common pattern like mail
or Display Name don't have to be specified in every individual object definition but as defaults with a custom pattern.
Values based of templates are always the last one to be generated. Whenever an object has a defined value for an
attribute it will take precendence of any default that might exist. This allows for special cases where a template
might not work to be supported too. All template variabled are object-specific meaning that it is not possible to get
a value from any other object than what the default is for. I.e. you cannot use the attribute of a different object.

The table below shows which variables are available in which scope (for which default attribute).

| Variable | Description | Scope |
|----------|-------------|-------|
| %u | username | display_name, home_dir, mail, password |
| %g | given_name | display_name, home_dir, mail |
| %l | surname | display_name, home_dir, mail |

**Example:**
```
[...]
defaults:
  # generate email as '<given_name>.<surname>@my-domain.com'
  mail: "%g.%l@my-domain.com"
  # have all users use SASL for password authentication
  user_password: "{SASL}%u"
  # Display name is '<given_name> <surname>'
  display_name: "%g %l"
  # home dir is always /tmp/
  home_dir: /tmp/
  # login shell is always /bin/sh
  login_shell: /bin/sh

```

## Compiling

To compile the source into a binary just run `make` in the directory. Golang must be installed. The Makefile creates
binaries for multiple platforms (Linux, FreeBSD, Darwin (MacOS)). All binaries are linked staticly and can be
copied and used without dependencies. To build for more platforms check out Golang's means of cross-compiling.

## Dependencies

Monban uses the following dependencies and other open source libraries. Thanks for providing them!!
* https://github.com/davecgh/go-spew
* https://github.com/go-ldap/ldap/v3
* https://github.com/kpango/glg
* https://github.com/urfave/cli
* https://golang.org/x/crypto
* https://gopkg.in/yaml.v3

## License

Licensed under the [MIT License](https://choosealicense.com/licenses/mit/)
