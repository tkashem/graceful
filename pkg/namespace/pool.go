package namespace

type Done func()
type Getter func() (namespace string, done Done, err error)
