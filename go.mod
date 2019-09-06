module github.com/luoxianginc/leaf

go 1.12

require (
	github.com/golang/protobuf v1.3.2
	github.com/gorilla/websocket v1.4.1
	github.com/kr/pretty v0.1.0 // indirect
	github.com/name5566/leaf v0.0.0-00010101000000-000000000000
	gopkg.in/check.v1 v1.0.0-20190902080502-41f04d3bba15 // indirect
	gopkg.in/mgo.v2 v2.0.0-20190816093944-a6b53ec6cb22
	gopkg.in/yaml.v2 v2.2.2 // indirect
)

replace github.com/name5566/leaf => ./
