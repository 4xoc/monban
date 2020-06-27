package tasks

var (
	totalElemCount int
)

// TotalLength returns the number of all tasks in all queues
func TotalLength() int {
	return totalElemCount
}

// Length returns the number of elements in this queue.
func (queue *Queue) Length() int {
	return queue.elemCount
}

// Push adds a new element to the queue in correct order.
//
// Correct order means the objects are sorted by the length of their DN as this ensures that parents come before their
// child objects (parent == shorter DN). NOTE: Elements with the same DN length are always added before the first
// element of the same DN already existing in the queue, reversing the order of elements.
//
// !! IMPORTANT !!
// Do *NOT* add the very same task to more than one queue. It will mess up the linked-lists and not make you happy.
func (queue *Queue) Push(task *Task) {
	var (
		elem *Task
	)

	if queue.first == nil {
		queue.first = task
		queue.elemCount++
		totalElemCount++
		return
	}

	elem = queue.first

	for elem != nil {

		if len(*elem.DN) < len(*task.DN) {

			if elem.next == nil {
				// add as next item
				elem.next = task
				task.prev = elem
				queue.elemCount++
				totalElemCount++
				return
			} else {
				if len(*elem.next.DN) >= len(*task.DN) {
					// add as next item
					task.next = elem.next
					elem.next = task
					task.prev = elem
					queue.elemCount++
					totalElemCount++
					return
				}
			}
		}

		elem = elem.next
	}
}

// Drop deletes an element from the queue if it exists in it.
func (queue *Queue) Drop(task *Task) {
	var (
		elem *Task
	)

	if task == nil {
		return
	}

	// loop through queue to ensure the element exists in this queue
	elem = queue.first

	for elem != nil {
		if elem == task {
			// have the previous element point to the next one and vice versa
			if elem.prev != nil {
				elem.prev.next = elem.next

				if elem.next != nil {
					elem.next.prev = elem.prev
				} else {
					queue.last = elem.prev
				}
				queue.elemCount--
				totalElemCount--
			} else {
				// in case elem is the first element in the queue simply shift by one element
				queue.first = elem.next
				queue.elemCount--
				totalElemCount--
			}
			// delete task reference so GC can delete it
			task = nil
			return
		}

		// check next element
		elem = elem.next
	}
}

// For runs a for-style loop over all elements of the queue in sorted order (by DN length ascending) and executes a
// given function with elem pointing to the queue element. The queue is not altered unless an element is run through
// Del().
func (queue *Queue) For(callback func(elem *Task) error) error {
	var (
		elem *Task
		err  error
	)

	elem = queue.first

	for elem != nil {
		err = callback(elem)

		if err != nil {
			// exit prematurely when callback returned an error
			return err
		}

		elem = elem.next
	}

	return nil
}

// ForRev is similar to For() but iterates through the queue in reverse order.
func (queue *Queue) ForRev(callback func(elem *Task) error) error {
	var (
		elem *Task
		err  error
	)

	elem = queue.last

	for elem != nil {
		err = callback(elem)

		if err != nil {
			// exit prematurely when callback returned an error
			return err
		}

		elem = elem.prev
	}

	return nil
}
