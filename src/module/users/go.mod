module example.com/users

go 1.19

require (
	example.com/session v0.0.0-00010101000000-000000000000
	golang.org/x/crypto v0.5.0
)

require github.com/google/uuid v1.3.0 // indirect

replace example.com/session => ../session
