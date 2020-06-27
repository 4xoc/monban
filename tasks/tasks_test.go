package tasks

import (
	"errors"
	"testing"
)

func NewStringPtr(text string) *string {
	return &text
}

type TestTasks struct {
	task Task
	// the position of this struct expected in the queue
	position int
}

// init "random" tasks
//
// see Push() why some of the positions are in reverse
var (
	TestData []TestTasks = []TestTasks{
		{Task{DN: NewStringPtr("dn=foo")}, 0},
		{Task{DN: NewStringPtr("dn=six,dn=foo")}, 7},
		{Task{DN: NewStringPtr("dn=one,dn=foo")}, 6},
		{Task{DN: NewStringPtr("dn=this_should_be_quite_late_in_the_queue,dn=foo")}, 9},
		{Task{DN: NewStringPtr("dn=s,dn=foo")}, 3},
		{Task{DN: NewStringPtr("dn=two,dn=foo")}, 5},
		{Task{DN: NewStringPtr("dn=three,dn=foo")}, 8},
		{Task{DN: NewStringPtr("dn=m,dn=foo")}, 2},
		{Task{DN: NewStringPtr("dn=a,dn=foo")}, 1},
		{Task{DN: NewStringPtr("dn=aaa,dn=foo")}, 4},
	}
	TestData2 []TestTasks = []TestTasks{
		{Task{DN: NewStringPtr("dn=foo")}, 0},
		{Task{DN: NewStringPtr("dn=six,dn=foo")}, 7},
		{Task{DN: NewStringPtr("dn=one,dn=foo")}, 6},
		{Task{DN: NewStringPtr("dn=this_should_be_quite_late_in_the_queue,dn=foo")}, 9},
		{Task{DN: NewStringPtr("dn=s,dn=foo")}, 3},
		{Task{DN: NewStringPtr("dn=two,dn=foo")}, 5},
		{Task{DN: NewStringPtr("dn=three,dn=foo")}, 8},
		{Task{DN: NewStringPtr("dn=m,dn=foo")}, 2},
		{Task{DN: NewStringPtr("dn=a,dn=foo")}, 1},
		{Task{DN: NewStringPtr("dn=aaa,dn=foo")}, 4},
	}
	queue  Queue = Queue{0, nil, nil}
	queue2 Queue = Queue{0, nil, nil}
)

// adds randomly ordered items into queue and then compares their position
func TestPush(t *testing.T) {
	var (
		i int
	)

	for i = range TestData {
		queue.Push(&TestData[i].task)
		queue2.Push(&TestData2[i].task)
	}

	if queue.Length() != len(TestData) {
		t.Errorf("queue length doesn't match number of test elements (got %d, expected %d)", queue.Length(), len(TestData))
	}

	if queue2.Length() != len(TestData2) {
		t.Errorf("queue2 length doesn't match number of test elements (got %d, expected %d)", queue2.Length(), len(TestData2))
	}

	if TotalLength() != len(TestData)*2 {
		t.Errorf("TotalLength doesn't match number of test elements (got %d, expected %d)", TotalLength(), len(TestData)*2)
	}

}

// runs the For() function on the queue and checks the order of the elements
func TestFor(t *testing.T) {
	var (
		i int
		j int
	)

	queue.For(func(elem *Task) error {
		// elem is nil when the queue has been fully been walked through
		if elem != nil {
			// check location
			for j = range TestData {
				if &TestData[j].task == elem {
					if TestData[j].position != i {
						t.Errorf("position of element in queue is %d but should be %d", i, TestData[j].position)
					}
				}
			}
			i++
		}
		return nil
	})

	i = 0

	// test premature return
	queue.For(func(elem *Task) error {
		i++

		// exit early
		if i == 2 {
			return errors.New("new error")
		}

		return nil
	})

	if i > 2 {
		t.Errorf("for loop should have ended prematurely but didn't")
	}

}

func TestRevFor(t *testing.T) {
	var (
		i int = queue.Length() - 1
		j int
	)

	queue.ForRev(func(elem *Task) error {
		// elem is nil when the queue has been fully been walked through
		if elem != nil {
			// check location
			for j = range TestData {
				if &TestData[j].task == elem {
					if TestData[j].position != i {
						t.Errorf("position of element in queue is %d but should be %d", i, TestData[j].position)
					}
				}
			}
			i--
		}
		return nil
	})
}

func TestDrop(t *testing.T) {
	var ()

	queue.Drop(nil)
	if queue.Length() != len(TestData) {
		t.Errorf("drop for 'nil' should not have deleted anything")
	}

	// drop "random" task
	queue.Drop(&TestData[4].task)
	if queue.Length() != (len(TestData) - 1) {
		t.Errorf("drop of task should have deleted one element but either did not drop anything or too much")
	}

	// drop first task in queue
	queue.Drop(&TestData[0].task)
	if queue.Length() != (len(TestData) - 2) {
		t.Errorf("drop of task should have deleted one element but either did not drop anything or too much")
	}

	// drop last task in queue
	queue.Drop(&TestData[3].task)
	if queue.Length() != (len(TestData) - 3) {
		t.Errorf("drop of task should have deleted one element but either did not drop anything or too much")
	}
}
