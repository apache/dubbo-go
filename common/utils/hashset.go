package utils

import (
	"fmt"
	"strings"
)

var itemExists = struct{}{}

type HashSet struct {
	Items map[interface{}]struct{}
}

func NewSet(values ...interface{}) *HashSet {
	set := &HashSet{Items: make(map[interface{}]struct{})}
	if len(values) > 0 {
		set.Add(values...)
	}
	return set
}

func (set *HashSet) Add(items ...interface{}) {
	for _, item := range items {
		set.Items[item] = itemExists
	}
}

func (set *HashSet) Remove(items ...interface{}) {
	for _, item := range items {
		delete(set.Items, item)
	}
}

func (set *HashSet) Contains(items ...interface{}) bool {
	for _, item := range items {
		if _, contains := set.Items[item]; !contains {
			return false
		}
	}
	return true
}
func (set *HashSet) Empty() bool {
	return set.Size() == 0
}
func (set *HashSet) Size() int {
	return len(set.Items)
}

func (set *HashSet) Clear() {
	set.Items = make(map[interface{}]struct{})
}

func (set *HashSet) Values() []interface{} {
	values := make([]interface{}, set.Size())
	count := 0
	for item := range set.Items {
		values[count] = item
		count++
	}
	return values
}
func (set *HashSet) String() string {
	str := "HashSet\n"
	var items []string
	for k := range set.Items {
		items = append(items, fmt.Sprintf("%v", k))
	}
	str += strings.Join(items, ", ")
	return str
}
