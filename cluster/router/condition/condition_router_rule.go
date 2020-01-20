package condition

type RouterRule struct {
	RawRule    string
	Runtime    bool
	Force      bool
	Valid      bool
	Enabled    bool
	Priority   int
	Dynamic    bool
	Scope      string
	Key        string
	Conditions []string
}
