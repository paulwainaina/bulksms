module example.com/bulksms

go 1.19

replace (
	example.com/members => ./module/members
	example.com/session => ./module/session
	example.com/users => ./module/users
)

require (
	example.com/members v0.0.0-00010101000000-000000000000
	example.com/session v0.0.0-00010101000000-000000000000
	example.com/users v0.0.0-00010101000000-000000000000
)

require (
	github.com/google/uuid v1.3.0 // indirect
	golang.org/x/crypto v0.5.0 // indirect
)
