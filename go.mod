module github.com/secrethub/secrethub-cli

go 1.12

require (
	bitbucket.org/zombiezen/cardcpx v0.0.0-20150417151802-902f68ff43ef
	github.com/alecthomas/kingpin v0.0.0-20190511121203-91c304675159
	github.com/asaskevich/govalidator v0.0.0-20180720115003-f9ffefc3facf
	github.com/atotto/clipboard v0.1.2
	github.com/danieljoos/wincred v1.0.1 // indirect
	github.com/docker/go-units v0.3.3
	github.com/fatih/color v1.7.0
	github.com/godbus/dbus v4.1.0+incompatible // indirect
	github.com/kylelemons/godebug v0.0.0-20170820004349-d65d576e9348
	github.com/masterzen/winrm v0.0.0-20190308153735-1d17eaf15943
	github.com/mattn/go-colorable v0.1.1
	github.com/mattn/go-isatty v0.0.7
	github.com/mitchellh/go-homedir v1.1.0
	github.com/mitchellh/mapstructure v1.1.2
	github.com/op/go-logging v0.0.0-20160315200505-970db520ece7
	github.com/secrethub/secrethub-go v0.20.0
	github.com/stretchr/objx v0.2.0 // indirect
	github.com/zalando/go-keyring v0.0.0-20190208082241-fbe81aec3a07
	golang.org/x/crypto v0.0.0-20190313024323-a1f597ede03a
	golang.org/x/sys v0.0.0-20190222072716-a9d3bda3a223
	golang.org/x/text v0.3.0
	gopkg.in/yaml.v2 v2.2.2
)

replace github.com/alecthomas/kingpin => github.com/simonbarendse/kingpin v0.0.0-20190704132514-1e3080fa9f42
