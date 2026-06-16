FROM golang:1.26-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
	-ldflags '-s -w' \
	-o /out/lets \
	./cmd/lets

FROM scratch

COPY --from=build /out/lets /lets

ENV PORT=8080
ENV DATABASE_PATH=/data/lets.db

EXPOSE 8080

ENTRYPOINT ["/lets"]
