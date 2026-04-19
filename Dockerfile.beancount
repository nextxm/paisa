FROM --platform=$BUILDPLATFORM node:22-alpine3.21 AS web
WORKDIR /usr/src/paisa
COPY package.json package-lock.json* ./
RUN npm install
COPY . .
RUN npx svelte-kit sync && npm run build

# Stage 1: Fetch tools natively for the TARGET architecture
FROM --platform=$TARGETPLATFORM alpine:3.21 AS tool-provider
RUN apk add --no-cache ledger beancount

# Stage 2: Build Paisa using fast native cross-compilation
FROM --platform=$BUILDPLATFORM golang:1.24-alpine3.21 AS go
ARG TARGETOS
ARG TARGETARCH
WORKDIR /usr/src/paisa

# Copy native tools from Stage 1 so Go embeds the correct architecture
COPY --from=tool-provider /usr/bin/ledger ./ledger
COPY --from=tool-provider /usr/bin/bean-report ./bean-report
COPY --from=tool-provider /usr/bin/bean-check ./bean-check
COPY --from=tool-provider /usr/bin/bean-query ./bean-query

COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY . .
COPY --from=web /usr/src/paisa/web/static ./web/static
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o paisa

FROM alpine:3.21
RUN apk --no-cache add ca-certificates ledger beancount tzdata
WORKDIR /root/
COPY --from=go /usr/src/paisa/paisa /usr/bin
EXPOSE 7500
CMD ["paisa", "serve"]
