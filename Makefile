.PHONY: backend frontend test install

install:
	cd frontend && npm install

backend:
	cd backend && go run .

frontend:
	cd frontend && npm run dev

test:
	cd backend && go test ./...
	cd frontend && npm test
