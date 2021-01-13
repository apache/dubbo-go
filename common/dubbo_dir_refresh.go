package common

import "sync"

// DirMap is used by dubbo-getty, it can notify delete event to dubbo directory directly by getty session
// look up at:
// remoting/getty/listener.go
// registry/directory/directory.go
var DirMap sync.Map
