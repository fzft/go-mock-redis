package node

type IReactor interface {
	NewReactor() (r *Reactor, err error)
	Run()
}
