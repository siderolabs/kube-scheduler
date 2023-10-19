all: gen build

build:
	docker build -t ghcr.io/siderolabs/kube-scheduler .

gen:
	deepcopy-gen --input-dirs ./apis/config --go-header-file ./hack/boilerplate.txt  -O zz_generated.deepcopy
