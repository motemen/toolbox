.PHONY: install
install:
	for d in *; do \
	  if [ -f "$$d/go.mod" ]; then \
	    echo $$d; \
	    ( cd $$d && go install -v ); \
	  fi; \
	done
