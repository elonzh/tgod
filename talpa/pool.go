package talpa

import "sync"

type worker struct {
	jobs <-chan func()
	stop <-chan struct{}
	wg   *sync.WaitGroup
}

func (w *worker) start() {
	go func() {
		for {
			select {
			case job, ok := <-w.jobs:
				if !ok {
					return
				}
				w.wg.Add(1)
				job()
				w.wg.Done()
			case <-w.stop:
			}
		}
	}()
}

type Pool struct {
	Jobs chan<- func()
	wg   sync.WaitGroup
	stop chan struct{}
}

func (p *Pool) Close() {
	close(p.stop)
	close(p.Jobs)
	p.wg.Wait()
}
