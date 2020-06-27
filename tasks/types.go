package tasks

import (
	"github.com/4xoc/monban/models"
)

const (
	TaskCreate = iota
	TaskUpdate
	TaskDelete
	// group membership actions
	TaskAddMember
	TaskDeleteMember
)

const (
	ObjectClassPosixAccount = iota
	ObjectClassPosixGroup
	ObjectClassGroupOfNames
	ObjectClassOrganisationalUnit
	ObjectClassSudoRole
)

// Task defines a task to execute against a ldap target
type Task struct {
	// DN is not nil when an object is to be deleted or a member gets added/deleted
	DN *string
	// ObjectClass defines what objectClass this task is for
	ObjectClass int
	// TaskType defines what kind of task this is (create, update...)
	TaskType int
	// Data contains data depending on the ObjectClass and TaskType.
	Data struct {
		// when adding or removing members
		DN *string
		// when creating or updating PosixAccount objects
		PosixAccount *models.PosixAccount
		// when creating or updating PosixGroup objects
		PosixGroup *models.PosixGroup
		// when creating or updating GroupOfNames objects
		GroupOfNames *models.GroupOfNames
		// when creating or updating OrganizationalUnit objects
		OrganizationalUnit *models.OrganizationalUnit
		// when creating or updating SudoRole objects
		SudoRole *models.SudoRole
	}
	// next points to the next Task of the same type used in a queue
	next *Task
	// prev points to the previous Task of the same type used in a queue
	prev *Task
}

// Queue implements a queue for a given objectClass and taskType. Elements are in a double-linked list style pattern
// and sorted on insert by length of DN
type Queue struct {
	// ElemCount is the overall number of elements in the queue
	elemCount int
	// first points to the first Task in a queue
	first *Task
	// last points to the last Task in a queue
	last *Task
}
