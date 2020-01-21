package condition

type BaseRouterRule struct {
	RawRule  string
	Runtime  bool
	Force    bool
	Valid    bool
	Enabled  bool
	Priority int
	Dynamic  bool
	Scope    string
	Key      string
}

type RouterRule struct {
	BaseRouterRule `yaml:",inline""`
	Conditions     []string
}
