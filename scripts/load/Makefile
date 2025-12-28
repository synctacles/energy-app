SHELL := /bin/bash
ROOT := $(shell pwd)
RESULTS := $(ROOT)/results
TMP := /tmp/synctacles-load
DATE := $(shell date -u +"%Y%m%dT%H%M%SZ")

K6 ?= k6
COUNTRY ?= NL

# Default test sizes (override on CLI)
VUS_SOLO ?= 100
DURATION_SOLO ?= 60s

VUS_COMBINED ?= 150
DURATION_COMBINED ?= 120s

VUS_SOAK ?= 80
DURATION ?= 45m

.PHONY: dirs sanity solo combined soak clean

dirs:
	@mkdir -p "$(RESULTS)" "$(TMP)"

sanity: dirs
	@echo "Sanity check (single user, short run)..."
	$(K6) run --vus 1 --duration 10s k6/combined.js \
	  --tag test=sanity \
	  --env COUNTRY=$(COUNTRY) \
	  --summary-export "$(RESULTS)/sanity_$(DATE).json"

solo: dirs
	@echo "Running SOLO baselines..."
	$(K6) run --vus $(VUS_SOLO) --duration $(DURATION_SOLO) k6/solo_load.js \
	  --tag test=solo_load \
	  --env COUNTRY=$(COUNTRY) \
	  --summary-export "$(RESULTS)/solo_load_$(DATE).json"
	$(K6) run --vus $(VUS_SOLO) --duration $(DURATION_SOLO) k6/solo_generation_mix.js \
	  --tag test=solo_generation_mix \
	  --env COUNTRY=$(COUNTRY) \
	  --summary-export "$(RESULTS)/solo_generation_mix_$(DATE).json"
	$(K6) run --vus $(VUS_SOLO) --duration $(DURATION_SOLO) k6/solo_balance.js \
	  --tag test=solo_balance \
	  --env COUNTRY=$(COUNTRY) \
	  --summary-export "$(RESULTS)/solo_balance_$(DATE).json"

combined: dirs
	@echo "Running COMBINED mixed load..."
	$(K6) run --vus $(VUS_COMBINED) --duration $(DURATION_COMBINED) k6/combined.js \
	  --tag test=combined \
	  --env COUNTRY=$(COUNTRY) \
	  --summary-export "$(RESULTS)/combined_$(DATE).json"

soak: dirs
	@echo "Running SOAK test for $(DURATION) with $(VUS_SOAK) VUs..."
	$(K6) run --vus $(VUS_SOAK) --duration $(DURATION) k6/soak.js \
	  --tag test=soak \
	  --env COUNTRY=$(COUNTRY) \
	  --summary-export "$(RESULTS)/soak_$(DATE).json"

clean:
	@rm -rf "$(RESULTS)"/*
