module github.com/mashenjun/mole

go 1.16

require (
	github.com/cznic/mathutil v0.0.0-20181122101859-297441e03548
	github.com/go-sql-driver/mysql v1.6.0
	github.com/gorilla/schema v1.2.0
	github.com/pingcap/errors v0.11.5-0.20200917111840-a15ef68f753d
	github.com/pingcap/parser v0.0.0-20200623164729-3a18f1e5dceb
	github.com/pingcap/sysutil v0.0.0-20210315073920-cc0985d983a3 // indirect
	github.com/pingcap/tidb v1.1.0-beta.0.20200630082100-328b6d0a955c
	github.com/pingcap/tidb-dashboard v0.0.0-20220110095800-367ff3b010e6
	github.com/pingcap/tiup v1.5.3
	github.com/prometheus/common v0.32.1
	github.com/prometheus/prometheus v1.8.2-0.20220105174609-ba03f7fc2368
	github.com/spf13/cobra v1.2.1
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	google.golang.org/grpc v1.43.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b
	gorm.io/driver/mysql v1.1.1
	gorm.io/gorm v1.21.12
)

replace google.golang.org/grpc v1.43.0 => google.golang.org/grpc v1.26.0
