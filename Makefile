.PHONY: test install-hooks

test:
	go test ./...

install-hooks:
	cp scripts/pre-commit .git/hooks/pre-commit
	chmod +x .git/hooks/pre-commit
