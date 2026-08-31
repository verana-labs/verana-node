package types

func NewParams() Params {
	return Params{}
}

func DefaultParams() Params {
	return NewParams()
}

func (p Params) Validate() error { return nil }
