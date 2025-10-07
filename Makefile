KUBERNETES_VERSION = 1.32 # renovate: datasource=github-releases depName=kubernetes-sigs/controller-tools extractVersion=^envtest-v(?<version>.+)$
EXTERNAL_DNS_VERSION = v0.19.0 # renovate: datasource=github-releases depName=kubernetes-sigs/external-dns

external-dns:
	crane export registry.k8s.io/external-dns/external-dns:$(EXTERNAL_DNS_VERSION) | tar -xv  --strip-components=1 /ko-app/external-dns

.PHONY: test
test:
	go test -v ./...

.PHONY: e2e
e2e: external-dns
	export KUBEBUILDER_ASSETS=$$(setup-envtest use -p path $(KUBERNETES_VERSION)); \
	go test -v -tags e2e ./...

.PHONY: test-coverage
test-coverage: external-dns
	export KUBEBUILDER_ASSETS=$$(setup-envtest use -p path $(KUBERNETES_VERSION)); \
	go test -v -tags e2e -coverpkg=./... -coverprofile=coverage.txt -covermode count ./...

.PHONY: clean
clean:
	rm -Rf external-dns
