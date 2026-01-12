.PHONY: help deploy deploy-batch verify clean

help:
	@echo "Sweet Security Deployment Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make deploy CLUSTER=<name> PROJECT=<id> REGION=<region>"
	@echo "  make deploy-batch CLUSTERS_FILE=clusters.txt"
	@echo "  make verify CLUSTER=<name> PROJECT=<id> REGION=<region>"
	@echo "  make clean CLUSTER=<name> PROJECT=<id> REGION=<region>"

deploy:
	@if [ -z "$(CLUSTER)" ] || [ -z "$(PROJECT)" ] || [ -z "$(REGION)" ]; then \
		echo "Error: CLUSTER, PROJECT, and REGION required"; \
		exit 1; \
	fi
	@./deploy.sh $(CLUSTER) $(PROJECT) $(REGION)

deploy-batch:
	@if [ -z "$(CLUSTERS_FILE)" ]; then \
		echo "Error: CLUSTERS_FILE required"; \
		exit 1; \
	fi
	@./deploy-batch.sh $(CLUSTERS_FILE)

verify:
	@if [ -z "$(CLUSTER)" ] || [ -z "$(PROJECT)" ] || [ -z "$(REGION)" ]; then \
		echo "Error: CLUSTER, PROJECT, and REGION required"; \
		exit 1; \
	fi
	@gcloud container clusters get-credentials $(CLUSTER) --region=$(REGION) --project=$(PROJECT)
	@echo "Checking pods..."
	@kubectl get pods -n sweet
	@echo ""
	@echo "Checking DNS..."
	@kubectl run test-dns-$$(date +%s) --image=busybox --rm -i --restart=Never -n sweet -- \
		nslookup registry.sweet.security 2>&1 | grep -A 2 "Name:"

clean:
	@if [ -z "$(CLUSTER)" ] || [ -z "$(PROJECT)" ] || [ -z "$(REGION)" ]; then \
		echo "Error: CLUSTER, PROJECT, and REGION required"; \
		exit 1; \
	fi
	@echo "Cleaning up Sweet Security from $(CLUSTER)..."
	@gcloud container clusters get-credentials $(CLUSTER) --region=$(REGION) --project=$(PROJECT)
	@helm uninstall sweet-operator -n sweet 2>/dev/null || true
	@helm uninstall sweet-scanner -n sweet 2>/dev/null || true
	@kubectl delete -f manifests/frontier-manual.yaml 2>/dev/null || true
	@kubectl delete namespace sweet 2>/dev/null || true
	@echo "Cleanup complete"
