package zookeeper

type DataListener interface {
	DataChange(eventType ZkEvent) bool //bool is return for interface implement is interesting
}
