# Fase di compilazione (Builder Stage)
FROM golang:1.22-alpine AS builder

# Imposta la directory di lavoro all'interno del container
WORKDIR /app

# Copia i file go.mod e go.sum per gestire le dipendenze
COPY go.mod .
COPY go.sum .

# Scarica le dipendenze del Go module
# `go mod download` scarica le dipendenze e le cache.
RUN go mod download

# Copia il codice sorgente dell'applicazione Go
COPY . .

# Compila l'applicazione Go.
# `-o app` specifica il nome del binario di output come 'app'.
# `go build -tags netgo -ldflags '-s -w'` crea un binario statico e più piccolo.
# `-tags netgo` evita l'uso di librerie di sistema per la rete, rendendo il binario più portabile.
# `-ldflags '-s -w'` rimuove le tabelle dei simboli e le informazioni di debug, riducendo le dimensioni del binario.
RUN CGO_ENABLED=0 GOOS=linux go build -o app -a -installsuffix cgo -ldflags '-s -w' ./main.go

# ---

# Fase finale (Runner Stage)
FROM alpine:latest

# Crea una directory per i file caricati (sebbene simulata in Go, è buona pratica averla)
RUN mkdir /uploads

# Imposta la directory di lavoro
WORKDIR /app

# Copia il binario compilato dalla fase di compilazione
COPY --from=builder /app/app .

# Espone la porta su cui l'applicazione Go ascolterà
EXPOSE 8080

# Comando per avviare l'applicazione quando il container viene eseguito
# `exec` format è preferito per Docker per una corretta gestione dei segnali
CMD ["./app"]

