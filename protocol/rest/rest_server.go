package rest

type RestServer interface {
	Start()
	Deploy()
	Undeploy()
	Destory()
}
