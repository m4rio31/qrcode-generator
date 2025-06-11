document.addEventListener('DOMContentLoaded', () => {
    const documentInput = document.getElementById('documentInput');
    const fileNameSpan = document.getElementById('fileName');
    const uploadButton = document.getElementById('uploadButton');
    const qrCodeImage = document.getElementById('qrCodeImage');
    const documentUrlParagraph = document.getElementById('documentUrl'); // Questo elemento non sarà più popolato dal backend
    const resultContainer = document.getElementById('resultContainer');
    const messageBox = document.getElementById('messageBox');
    const messageText = document.getElementById('messageText');
    const closeMessageButton = document.getElementById('closeMessage');

    let selectedFile = null;

    // Handles file selection and updates the displayed file name.
    // Gestisce la selezione del file e aggiorna il nome del file visualizzato.
    documentInput.addEventListener('change', (event) => {
        selectedFile = event.target.files[0];
        if (selectedFile) {
            fileNameSpan.textContent = selectedFile.name;
            // Hides the previous result when a new file is selected.
            // Nasconde il risultato precedente quando un nuovo file viene selezionato.
            resultContainer.classList.add('hidden');
        } else {
            fileNameSpan.textContent = 'Nessun file selezionato';
        }
    });

    // Handles the click on the upload button.
    // Gestisce il click sul pulsante di upload.
    uploadButton.addEventListener('click', async () => {
        if (!selectedFile) {
            showMessage("Per favore, seleziona un documento da caricare.");
            return;
        }

        // Create a FormData object to send the file.
        // Crea un oggetto FormData per inviare il file.
        const formData = new FormData();
        formData.append('document', selectedFile); // "document" must match the field name expected by the Go backend.

        // Show a loading message.
        // Mostra un messaggio di caricamento.
        showMessage("Caricamento in corso...", true); // Show message without OK button

        try {
            // Send the file to the Go backend.
            // Invia il file al backend Go.
            // The URL is '/upload' because the Go server serves the frontend on the root
            // L'URL è '/upload' perché il server Go serve il frontend sulla root
            // and handles uploads on the /upload endpoint.
            // e gestisce gli upload sull'endpoint /upload.
            const response = await fetch('/upload', {
                method: 'POST',
                body: formData, // Send the file as FormData
            });

            // Close the loading message.
            // Chiude il messaggio di caricamento.
            messageBox.classList.add('hidden');

            if (!response.ok) {
                // If the response is not OK, it might not be JSON, but an error text.
                // Se la risposta non è OK, potrebbe non essere un JSON, ma un testo di errore.
                const errorText = await response.text();
                throw new Error(`Errore HTTP: ${response.status} - ${errorText}`);
            }

            // At this point, the backend sends the PNG image directly.
            // A questo punto, il backend invia direttamente l'immagine PNG.
            // We need to read the response as a Blob (Binary Large Object)
            // Dobbiamo leggere la risposta come Blob (Binary Large Object)
            const imageBlob = await response.blob();

            // Create an object URL for the image blob
            // Creiamo un URL oggetto per l'immagine blob
            const imageUrl = URL.createObjectURL(imageBlob);

            // Update the QR code image.
            // Aggiorna l'immagine del codice QR.
            qrCodeImage.src = imageUrl;
            // Remove or make the document URL unpopulated, as it's no longer sent.
            // Rimuovi o rendi non popolato l'URL del documento, dato che non viene più inviato.
            // documentUrlParagraph.textContent = `URL del Documento: Non disponibile con questa modalità.`;
            documentUrlParagraph.textContent = ''; // Clear text
            documentUrlParagraph.href = '#'; // Remove actual URL if there was one


            // Show the results container.
            // Mostra il container dei risultati.
            resultContainer.classList.remove('hidden');

            // Reset the file input field
            // Reset del campo input del file
            documentInput.value = '';
            selectedFile = null;
            fileNameSpan.textContent = 'Nessun file selezionato';

        } catch (error) {
            console.error('Error during upload:', error); // Errore durante il caricamento
            showMessage(`Errore durante il caricamento del documento: ${error.message}`); // Errore durante il caricamento del documento
        }
    });

    // Function to show custom messages.
    // Funzione per mostrare messaggi personalizzati.
    function showMessage(message, isLoading = false) {
        messageText.textContent = message;
        if (isLoading) {
            closeMessageButton.classList.add('hidden'); // Hide OK button if it's a loading message
        } else {
            closeMessageButton.classList.remove('hidden'); // Show OK button for normal messages
        }
        messageBox.classList.remove('hidden');
    }

    // Handles closing the message box.
    // Gestisce la chiusura della message box.
    closeMessageButton.addEventListener('click', () => {
        messageBox.classList.add('hidden');
    });
});

