package store

type fsm struct {
	dataFile string
	lock     *flock.Flock
}

type fsmSnapshot struct {
	data []byte
}
