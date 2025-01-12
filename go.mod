module github.com/xaionaro-go/speech

go 1.23.2

toolchain go1.23.4

replace github.com/asticode/go-astiav v0.30.0 => github.com/xaionaro-go/astiav v0.0.0-20250106205037-a1605f324663

replace github.com/mutablelogic/go-whisper v0.0.22-0.20241221210700-ba095bdd5196 => github.com/xaionaro-go/whisper v0.0.0-20250112001617-f62796353e33

require (
	fyne.io/fyne/v2 v2.5.3
	github.com/facebookincubator/go-belt v0.0.0-20240804203001-846c4409d41c
	github.com/hashicorp/go-multierror v1.1.1
	github.com/hashicorp/go-version v1.7.0
	github.com/hashicorp/golang-lru/v2 v2.0.7
	github.com/mutablelogic/go-whisper v0.0.22-0.20241221210700-ba095bdd5196
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.10.0
	github.com/xaionaro-go/audio v0.0.0-20250112164406-5f5703997d03
	github.com/xaionaro-go/observability v0.0.0-20250111142240-5d72f17a6d12
	github.com/xaionaro-go/player v0.0.0-20250112171237-124c9f68a262
	github.com/xaionaro-go/xcontext v0.0.0-20250111150717-e70e1f5b299c
	github.com/xaionaro-go/xsync v0.0.0-20250112014853-6c166ba9b463
	google.golang.org/grpc v1.69.2
	google.golang.org/protobuf v1.36.2
)

require (
	fyne.io/systray v1.11.0 // indirect
	github.com/BurntSushi/toml v1.4.0 // indirect
	github.com/DataDog/gostackparse v0.7.0 // indirect
	github.com/asticode/go-astiav v0.30.0 // indirect
	github.com/asticode/go-astikit v0.51.1 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/ebitengine/oto/v3 v3.3.2 // indirect
	github.com/ebitengine/purego v0.8.0 // indirect
	github.com/fredbi/uri v1.1.0 // indirect
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/fyne-io/gl-js v0.0.0-20220119005834-d2da28d9ccfe // indirect
	github.com/fyne-io/glfw-js v0.0.0-20241126112943-313d8a0fe1d0 // indirect
	github.com/fyne-io/image v0.0.0-20220602074514-4956b0afb3d2 // indirect
	github.com/go-gl/gl v0.0.0-20211210172815-726fda9656d6 // indirect
	github.com/go-gl/glfw/v3.3/glfw v0.0.0-20240506104042-037f3cc74f2a // indirect
	github.com/go-ng/slices v0.0.0-20230703171042-6195d35636a2 // indirect
	github.com/go-ng/sort v0.0.0-20220617173827-2cc7cd04f7c7 // indirect
	github.com/go-ng/xatomic v0.0.0-20230519181013-85c0ec87e55f // indirect
	github.com/go-ng/xsort v0.0.0-20220617174223-1d146907bccc // indirect
	github.com/go-text/render v0.2.0 // indirect
	github.com/go-text/typesetting v0.2.0 // indirect
	github.com/goccy/go-yaml v1.15.13 // indirect
	github.com/godbus/dbus/v5 v5.1.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/gopherjs/gopherjs v1.17.2 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/huandu/go-tls v0.0.0-20200109070953-6f75fb441850 // indirect
	github.com/jeandeaual/go-locale v0.0.0-20240223122105-ce5225dcaa49 // indirect
	github.com/jfreymuth/oggvorbis v1.0.5 // indirect
	github.com/jfreymuth/pulse v0.1.1 // indirect
	github.com/jfreymuth/vorbis v1.0.2 // indirect
	github.com/jsummers/gobmp v0.0.0-20151104160322-e2ba15ffa76e // indirect
	github.com/klauspost/cpuid/v2 v2.2.7 // indirect
	github.com/lazybeaver/entropy v0.0.0-20190817091901-99e00c014ccd
	github.com/nicksnyder/go-i18n/v2 v2.4.0 // indirect
	github.com/phuslu/goid v1.0.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/rymdport/portal v0.3.0 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/srwiley/oksvg v0.0.0-20221011165216-be6e8873101c // indirect
	github.com/srwiley/rasterx v0.0.0-20220730225603-2ab79fcdd4ef // indirect
	github.com/xaionaro-go/gorex v0.0.0-20241010205749-bcd59d639c4d // indirect
	github.com/xaionaro-go/libsrt v0.0.0-20250105232601-e760c79b2bc3 // indirect
	github.com/xaionaro-go/logrustash v0.0.0-20240804141650-d48034780a5f // indirect
	github.com/xaionaro-go/object v0.0.0-20241026212449-753ce10ec94c
	github.com/xaionaro-go/proxy v0.0.0-20250111150848-1f0e7b262638 // indirect
	github.com/xaionaro-go/recoder v0.0.0-20250111153658-7e55cef13b0f // indirect
	github.com/xaionaro-go/spinlock v0.0.0-20200518175509-30e6d1ce68a1 // indirect
	github.com/xaionaro-go/unsafetools v0.0.0-20241024014258-a46e1ce3763e // indirect
	github.com/yuin/goldmark v1.7.1 // indirect
	golang.org/x/exp v0.0.0-20240823005443-9b4947da3948 // indirect
	golang.org/x/image v0.18.0 // indirect
	golang.org/x/mobile v0.0.0-20231127183840-76ac6878050a // indirect
	golang.org/x/net v0.34.0 // indirect
	golang.org/x/sys v0.29.0 // indirect
	golang.org/x/text v0.21.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250106144421-5f5ef82da422 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	lukechampine.com/blake3 v1.3.0 // indirect
)
