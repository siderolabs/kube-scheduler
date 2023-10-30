PUSH ?= false

all: gen build

build:
	docker buildx build --push=$(PUSH) --platform=linux/amd64,linux/arm64 -t ghcr.io/siderolabs/kube-scheduler .

gen:
	deepcopy-gen --input-dirs ./apis/config --go-header-file ./hack/boilerplate.txt  -O zz_generated.deepcopy
	deepcopy-gen --input-dirs ./apis/config/v1alpha1 --go-header-file ./hack/boilerplate.txt  -O zz_generated.deepcopy
	defaulter-gen --input-dirs ./apis/config/v1alpha1 --go-header-file ./hack/boilerplate.txt  -O zz_generated.defaults
	conversion-gen --input-dirs ./apis/config/v1alpha1 --go-header-file ./hack/boilerplate.txt  -O zz_generated.conversion

tools:
	go install k8s.io/code-generator/cmd/deepcopy-gen@v0.28.3
	go install k8s.io/code-generator/cmd/defaulter-gen@v0.28.3
	go install k8s.io/code-generator/cmd/conversion-gen@v0.28.3
