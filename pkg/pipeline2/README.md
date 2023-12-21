# pipeline2

Attemtp to use generics in pipelines.

## Problems

In the SyncPipeline a workpiece can change the type, e.g.:

```go
func (p SyncPipeline[T]) SendSync(work T) (err error) {
	if p.ctx.Err() != nil {
		return p.ctx.Err()
	}
	p.stdin <- work
	outWork := <-p.stdout
	if err, ok := outWork.(error); ok { // <-- invalid operation: cannot use type assertion on type parameter value outWork (variable of type T constrained by any)
		return err
	}
	return nil
}
```

```go
func puller_sync[T any](wo *WiredOperator[T]) {
	for work := range wo.Stdin {
		if work == nil {
			pipelinePanic("nil in puller_sync stdin", wo.name, wo.wctx)
		}
		if err, ok := work.(IErrorPipeline); ok { // invalid operation: cannot use type assertion on type parameter value work (variable of type T constrained by any)
			if catch, ok := wo.Operator.(ICatch); ok {
				if newerr := catch.OnErr(err, err.GetWork(), wo.wctx); newerr != nil {
					wo.Stdout <- wo.NewError(fmt.Errorf("nested error '%w' while handling '%w'", newerr, err), err.GetWork(), placeCatchOnErr)
					continue
				}
			} else {
```