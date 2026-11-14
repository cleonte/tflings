.PHONY: build test vet ci
build:
	go build -o bin/tflings ./cmd/tflings
test:
	go test ./...
vet:
	go vet ./...

# ci is what .github/workflows/ci.yml does, on this machine: vet, then the whole
# suite once per Terraform in TF_BINS, with every exercise proven. Exercises that
# use the AWS provider run when Floci is up (`tflings aws up`) and TFLINGS_TEST_HOME
# is a home with the provider cached (`TFLINGS_HOME=... tflings doctor --prefetch`);
# otherwise they are skipped and the output says so.
#
#   make ci TF_BINS="/usr/bin/terraform" TFLINGS_TEST_HOME=/path/to/home
TF_BINS ?= spikes/.cache/dl/x1.12.2/terraform spikes/.cache/dl/x1.17.2/terraform
TFLINGS_TEST_HOME ?= $(CURDIR)/spikes/.cache
ci: vet
	@for tf in $(TF_BINS); do \
		echo "== $$tf"; \
		abs=$$tf; if [ "$${tf#/}" = "$$tf" ]; then abs=$(CURDIR)/$$tf; fi; \
		TFLINGS_TEST_HOME=$(TFLINGS_TEST_HOME) TFLINGS_TEST_TERRAFORM=$$abs \
			go test -count=1 ./... || exit 1; \
	done
