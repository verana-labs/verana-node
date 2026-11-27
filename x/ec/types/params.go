package types

func NewParams() Params {
	return Params{}
}

func DefaultParams() Params {
	return NewParams()
}

// Validate returns nil; the module owns no parameter group.
func (p Params) Validate() error { return nil }
