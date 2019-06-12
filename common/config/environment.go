package config

import "sync"

type Environment struct {
}

var instance *Environment

var once sync.Once

func GetEnvInstance() *Environment {
	once.Do(func() {
		instance = &Environment{}
	})
	return instance
}
