package main

import (
	"context"
	"embed"

	// Ancora usato per la codifica base64 del QR, ma non per la risposta HTTP diretta
	// "encoding/json"   // Rimosso: non più necessario per la risposta JSON
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"time"

	"cloud.google.com/go/storage"       // Libreria Google Cloud Storage
	qrcode "github.com/skip2/go-qrcode" // Libreria per la generazione del codice QR
)

//go:embed frontend/*
var staticFiles embed.FS // Incorpora tutti i file nella directory 'frontend'

// Variabili globali per il client di storage e il nome del bucket
var (
	storageClient *storage.Client
	bucketName    = "qrcode-generator-bucket" // Sostituisci con il nome del tuo bucket GCS reale
)

// Definizione di una struct per la risposta JSON (rimossa in questo caso)
// type uploadResponse struct {
// 	DocumentURL string `json:"documentUrl"`
// 	QRCodeImage string `json:"qrCodeImage"`
// }

func main() {
	// Initialize the Google Cloud Storage client.
	// Inizializza il client di Google Cloud Storage.
	// It is crucial to set the GOOGLE_APPLICATION_CREDENTIALS environment variable
	// È fondamentale impostare la variabile d'ambiente GOOGLE_APPLICATION_CREDENTIALS
	// or provide a path to the JSON credentials file.
	// o fornire un percorso al file JSON delle credenziali.
	// For Docker, it's recommended to mount the credentials file or use
	// Per Docker, è consigliabile montare il file delle credenziali o utilizzare
	// Workload Identity if on GKE/Cloud Run.
	// Workload Identity se su GKE/Cloud Run.
	ctx := context.Background()
	var err error

	// Option 1: Load credentials from environment variable (recommended for Docker)
	// Opzione 1: Carica le credenziali dalla variabile d'ambiente (consigliato per Docker)
	// If the GOOGLE_APPLICATION_CREDENTIALS variable is set, credentials will be loaded automatically.
	// Se la variabile GOOGLE_APPLICATION_CREDENTIALS è impostata, le credenziali verranno caricate automaticamente.
	storageClient, err = storage.NewClient(ctx)
	if err != nil {
		log.Fatalf("Error initializing Google Cloud Storage client: %v", err) // Errore nell'inizializzazione del client Google Cloud Storage
	}
	defer storageClient.Close() // Ensure the client is closed at the end

	log.Println("Google Cloud Storage client initialized.") // Client Google Cloud Storage inizializzato.
	log.Println("Server started on port :8080")             // Server avviato sulla porta :8080

	// Defines a handler for static files. Serves files from the 'frontend' directory.
	// Definisce un gestore per i file statici. Serve i file dalla directory 'frontend'.
	fs := http.FileServer(http.FS(staticFiles))
	http.Handle("/", fs) // Associates the static file handler with the root ("/")

	// Defines the handler for the upload endpoint.
	// Definisce il gestore per l'endpoint di upload.
	http.HandleFunc("/upload", uploadHandler)

	// Starts the HTTP server on port 8080.
	// Avvia il server HTTP sulla porta 8080.
	err = http.ListenAndServe(":8080", nil) // Starts listening for HTTP connections
	if err != nil {
		log.Fatalf("Error starting server: %v", err) // Prints the error and exits if the server fails to start
	}
}

// uploadHandler handles file uploads and QR code generation.
// uploadHandler gestisce il caricamento dei file e la generazione del codice QR.
func uploadHandler(w http.ResponseWriter, r *http.Request) {
	// Verify that the HTTP method is POST.
	// Verifica che il metodo HTTP sia POST.
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed) // Metodo non consentito
		return
	}

	// Set the file size limit to 10MB to prevent abuse.
	// Imposta il limite di dimensione del file a 10MB per prevenire abusi.
	r.ParseMultipartForm(10 << 20) // 10 MB

	// Retrieve the file from the multipart form.
	// Recupera il file dal form multipart.
	file, handler, err := r.FormFile("document") // "document" is the name of the file input field in the frontend
	if err != nil {
		log.Printf("Error retrieving file: %v", err)                  // Errore nel recupero del file
		http.Error(w, "Error retrieving file", http.StatusBadRequest) // Errore nel recupero del file
		return
	}
	defer file.Close() // Ensures the file is closed at the end of the function

	log.Printf("File received: %s (Size: %d bytes)", handler.Filename, handler.Size) // File ricevuto: %s (Dimensione: %d byte)

	// Create a unique name for the object in the GCS bucket
	// Crea un nome univoco per l'oggetto nel bucket GCS
	objectName := fmt.Sprintf("documenti/%s-%d%s",
		filepath.Base(handler.Filename[:len(handler.Filename)-len(filepath.Ext(handler.Filename))]), // Name without extension
		time.Now().UnixNano(),
		filepath.Ext(handler.Filename)) // Original file extension

	// Start uploading to Google Cloud Storage
	// Inizia il caricamento su Google Cloud Storage
	ctx := context.Background()
	wc := storageClient.Bucket(bucketName).Object(objectName).NewWriter(ctx)

	// Set the appropriate Content-Type for the object (important for the browser)
	// Imposta il Content-Type appropriato per l'oggetto (importante per il browser)
	wc.ContentType = handler.Header.Get("Content-Type")
	if wc.ContentType == "" {
		// If Content-Type is not provided by the client, try to deduce it or use a default
		// Se Content-Type non è fornito dal client, prova a dedurlo o usa un default
		wc.ContentType = "application/octet-stream"
	}

	// Copy the file content to the GCS object
	// Copia il contenuto del file nell'oggetto GCS
	if _, err = io.Copy(wc, file); err != nil {
		log.Printf("Error uploading file to GCS: %v", err)                                     // Errore durante il caricamento del file su GCS
		wc.Close()                                                                             // Close the writer in case of error
		http.Error(w, "Error uploading file to cloud storage", http.StatusInternalServerError) // Errore durante il caricamento del file su cloud storage
		return
	}

	// Close the writer to finalize the upload
	// Chiudi il writer per finalizzare il caricamento
	if err := wc.Close(); err != nil {
		log.Printf("Error closing GCS writer: %v", err)                                           // Errore nella chiusura del writer GCS
		http.Error(w, "Error finalizing upload to cloud storage", http.StatusInternalServerError) // Errore durante la finalizzazione del caricamento su cloud storage
		return
	}

	// Generate the public (or signed) URL of the document.
	// Genera l'URL pubblico (o firmato) del documento.
	// For unauthenticated public access, ensure the object has public read permissions.
	// Per un accesso pubblico non autenticato, assicurati che l'oggetto abbia permessi di lettura pubblici.
	// In production, you might want to generate a signed URL for temporary and secure access.
	// In produzione, potresti voler generare un URL firmato per un accesso temporaneo e sicuro.
	documentURL := fmt.Sprintf("https://storage.googleapis.com/%s/%s", bucketName, objectName)
	log.Printf("File uploaded to GCS: %s", documentURL) // File caricato su GCS: %s

	// Generate the QR code using github.com/skip2/go-qrcode.
	// Genera il codice QR utilizzando github.com/skip2/go-qrcode.
	// The `Encode` function of this library returns `[]byte` (the PNG image data).
	// La funzione `Encode` di questa libreria restituisce direttamente `[]byte` (l'immagine PNG).
	pngBytes, err := qrcode.Encode(documentURL, qrcode.Medium, 256)
	if err != nil {
		log.Printf("Error generating QR code: %v", err)                           // Errore nella generazione del codice QR
		http.Error(w, "Error generating QR code", http.StatusInternalServerError) // Errore nella generazione del codice QR
		return
	}

	// Set the Content-Type header to image/png.
	// Imposta l'header Content-Type a image/png.
	w.Header().Set("Content-Type", "image/png")
	// Set the CORS header to allow requests from the frontend.
	// Imposta l'header CORS per consentire le richieste dal frontend.
	w.Header().Set("Access-Control-Allow-Origin", "*") // For development environment. In production, specify specific domains.

	// Write the PNG bytes directly to the response writer.
	// Scrivi i byte PNG direttamente al writer della risposta.
	if _, err := w.Write(pngBytes); err != nil {
		log.Printf("Error writing QR code PNG to response: %v", err) // Errore durante la scrittura del PNG del codice QR nella risposta
	}
	log.Println("QR code PNG image sent directly.") // Immagine PNG del codice QR inviata direttamente.
}
