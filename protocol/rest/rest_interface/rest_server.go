package rest_interface

type RestServer interface {
	Start()
	Deploy()
	Undeploy()
	Destory()
}
