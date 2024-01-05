module github.com/singchia/frontier

go 1.20

replace github.com/singchia/geminio => ../../moresec/singchia/geminio

require (
	github.com/jumboframes/armorigo v0.2.5
	github.com/singchia/geminio v1.1.0
	github.com/singchia/go-timer/v2 v2.2.1
	github.com/spf13/pflag v1.0.5
	gopkg.in/yaml.v2 v2.4.0
	gorm.io/driver/sqlite v1.5.4
	gorm.io/gorm v1.25.5
	k8s.io/klog/v2 v2.110.1
)

require (
	github.com/go-logr/logr v1.3.0 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/mattn/go-sqlite3 v1.14.17 // indirect
	github.com/singchia/yafsm v1.0.1 // indirect
)
