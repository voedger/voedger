package acl

// NewACLBuilder creates and returns a new instance of aclBuilder.
func NewACLBuilder[Role, Operation, Resource comparable]() *aclBuilder[Role, Operation, Resource] {
	return &aclBuilder[Role, Operation, Resource]{
		permissions: make(map[Role]map[Operation]map[Resource]bool),
	}
}
