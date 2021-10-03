module github.com/nametaginc/tvm

go 1.16

replace github.com/akrylysov/algnhsa v0.12.1 => github.com/nametaginc/algnhsa v0.12.2

require (
	cloud.google.com/go/firestore v1.5.0
	github.com/akrylysov/algnhsa v0.12.1
	github.com/aws/aws-lambda-go v1.26.0
	github.com/aws/aws-sdk-go v1.40.27
	github.com/pkg/errors v0.9.1
	github.com/tstranex/u2f v1.0.0
	goji.io v2.0.2+incompatible
	golang.org/x/net v0.0.0-20210813160813-60bc85c4be6d // indirect
	golang.org/x/oauth2 v0.0.0-20210819190943-2bc19b11175f
	google.golang.org/grpc v1.35.0
	gotest.tools v2.2.0+incompatible // indirect
)
