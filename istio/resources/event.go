package resources

type XdsEventUpdateType uint32

const (
	XdsEventUpdateLDS XdsEventUpdateType = iota
	XdsEventUpdateRDS
	XdsEventUpdateCDS
	XdsEventUpdateEDS
)

type XdsUpdateEvent struct {
	Type   XdsEventUpdateType
	Object interface{}
}
